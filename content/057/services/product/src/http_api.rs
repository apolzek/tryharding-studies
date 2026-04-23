use std::sync::Arc;

use axum::{
    extract::{Path, Query, State},
    http::StatusCode,
    routing::{get, post},
    Json, Router,
};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

use crate::service::{CreateProductInput, ProductService};

#[derive(Deserialize)]
pub struct CreateProductPayload {
    pub sku: String,
    pub name: String,
    #[serde(default)]
    pub description: String,
    pub price: f64,
    pub stock: i32,
    #[serde(default)]
    pub catalog_id: String,
}

#[derive(Serialize)]
pub struct ProductOut {
    pub id: String,
    pub sku: String,
    pub name: String,
    pub description: String,
    pub price: f64,
    pub stock: i32,
    pub catalog_id: String,
}

#[derive(Deserialize)]
pub struct ListQuery {
    #[serde(default)]
    pub limit: i32,
    #[serde(default)]
    pub offset: i32,
}

pub fn routes(svc: Arc<ProductService>) -> Router {
    Router::new()
        .route("/products", post(create).get(list))
        .route("/products/:id", get(get_one))
        .with_state(svc)
}

async fn create(
    State(svc): State<Arc<ProductService>>,
    Json(p): Json<CreateProductPayload>,
) -> Result<(StatusCode, Json<ProductOut>), StatusCode> {
    let result = svc
        .create(CreateProductInput {
            sku: p.sku,
            name: p.name,
            description: p.description,
            price: p.price,
            stock: p.stock,
            catalog_id: p.catalog_id,
        })
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    Ok((StatusCode::CREATED, Json(to_out(&result))))
}

async fn list(
    State(svc): State<Arc<ProductService>>,
    Query(q): Query<ListQuery>,
) -> Result<Json<Vec<ProductOut>>, StatusCode> {
    let items = svc
        .list(q.limit, q.offset)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?;
    Ok(Json(items.iter().map(to_out).collect()))
}

async fn get_one(
    State(svc): State<Arc<ProductService>>,
    Path(id): Path<String>,
) -> Result<Json<ProductOut>, StatusCode> {
    let uuid = Uuid::parse_str(&id).map_err(|_| StatusCode::BAD_REQUEST)?;
    let item = svc
        .get(uuid)
        .await
        .map_err(|_| StatusCode::INTERNAL_SERVER_ERROR)?
        .ok_or(StatusCode::NOT_FOUND)?;
    Ok(Json(to_out(&item)))
}

fn to_out(p: &crate::repository::Product) -> ProductOut {
    ProductOut {
        id: p.id.to_string(),
        sku: p.sku.clone(),
        name: p.name.clone(),
        description: p.description.clone(),
        price: p.price,
        stock: p.stock,
        catalog_id: p.catalog_id.clone(),
    }
}
