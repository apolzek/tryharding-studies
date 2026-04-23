mod repository;
mod service;
mod telemetry;
mod http_api;
mod grpc_api;

use std::sync::Arc;

use anyhow::Result;
use axum::{routing::get, Router};
use deadpool_postgres::{Config, ManagerConfig, Pool, RecyclingMethod, Runtime};
use tokio_postgres::NoTls;
use tonic::transport::Server;
use tracing::info;

pub mod proto {
    tonic::include_proto!("product.v1");
}

use grpc_api::ProductGrpc;
use proto::product_service_server::ProductServiceServer;
use repository::PgProductRepository;
use service::ProductService;

fn build_pool() -> Pool {
    let mut cfg = Config::new();
    cfg.host = Some(std::env::var("POSTGRES_HOST").unwrap_or_else(|_| "postgres".into()));
    cfg.port = Some(
        std::env::var("POSTGRES_PORT")
            .unwrap_or_else(|_| "5432".into())
            .parse()
            .unwrap_or(5432),
    );
    cfg.user = Some(std::env::var("POSTGRES_USER").unwrap_or_else(|_| "postgres".into()));
    cfg.password = Some(std::env::var("POSTGRES_PASSWORD").unwrap_or_else(|_| "postgres".into()));
    cfg.dbname = Some(std::env::var("POSTGRES_DB").unwrap_or_else(|_| "product".into()));
    cfg.manager = Some(ManagerConfig {
        recycling_method: RecyclingMethod::Fast,
    });
    cfg.create_pool(Some(Runtime::Tokio1), NoTls).unwrap()
}

async fn migrate(pool: &Pool) -> Result<()> {
    let client = pool.get().await?;
    client
        .batch_execute(
            r#"CREATE TABLE IF NOT EXISTS products (
                id UUID PRIMARY KEY,
                sku TEXT UNIQUE NOT NULL,
                name TEXT NOT NULL,
                description TEXT,
                price DOUBLE PRECISION NOT NULL,
                stock INTEGER NOT NULL,
                catalog_id TEXT
            );"#,
        )
        .await?;
    Ok(())
}

#[tokio::main]
async fn main() -> Result<()> {
    let _guard = telemetry::init("product");

    let pool = build_pool();
    migrate(&pool).await?;

    let repo = Arc::new(PgProductRepository::new(pool.clone()));
    let svc = Arc::new(ProductService::new(repo));

    let http_svc = svc.clone();
    let grpc_svc = svc.clone();

    let http_port: u16 = std::env::var("HTTP_PORT")
        .unwrap_or_else(|_| "8004".into())
        .parse()?;
    let grpc_port: u16 = std::env::var("GRPC_PORT")
        .unwrap_or_else(|_| "50051".into())
        .parse()?;

    let http_app = Router::new()
        .route("/health", get(|| async { "ok" }))
        .merge(http_api::routes(http_svc));

    let http_addr = format!("0.0.0.0:{}", http_port);
    let grpc_addr = format!("0.0.0.0:{}", grpc_port).parse()?;

    info!(%http_addr, %grpc_addr, "product service starting");

    let http_listener = tokio::net::TcpListener::bind(&http_addr).await?;
    let http_task = tokio::spawn(async move { axum::serve(http_listener, http_app).await });

    let grpc_task = tokio::spawn(async move {
        Server::builder()
            .add_service(ProductServiceServer::new(ProductGrpc::new(grpc_svc)))
            .serve(grpc_addr)
            .await
    });

    tokio::select! {
        r = http_task => { r??; }
        r = grpc_task => { r??; }
    }
    Ok(())
}
