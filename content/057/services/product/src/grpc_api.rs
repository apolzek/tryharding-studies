use std::sync::Arc;

use tonic::{Request, Response, Status};
use uuid::Uuid;

use crate::proto::product_service_server::ProductService as GrpcTrait;
use crate::proto::{
    CreateProductRequest, DecrementStockRequest, DecrementStockResponse, GetProductRequest,
    ListProductsRequest, ListProductsResponse, Product as ProtoProduct,
};
use crate::repository::Product;
use crate::service::{CreateProductInput, ProductService};

pub struct ProductGrpc {
    svc: Arc<ProductService>,
}

impl ProductGrpc {
    pub fn new(svc: Arc<ProductService>) -> Self {
        Self { svc }
    }
}

#[tonic::async_trait]
impl GrpcTrait for ProductGrpc {
    async fn get_product(
        &self,
        req: Request<GetProductRequest>,
    ) -> Result<Response<ProtoProduct>, Status> {
        let id = Uuid::parse_str(&req.into_inner().id)
            .map_err(|_| Status::invalid_argument("bad uuid"))?;
        let found = self
            .svc
            .get(id)
            .await
            .map_err(|e| Status::internal(e.to_string()))?
            .ok_or_else(|| Status::not_found("product not found"))?;
        Ok(Response::new(to_proto(&found)))
    }

    async fn list_products(
        &self,
        req: Request<ListProductsRequest>,
    ) -> Result<Response<ListProductsResponse>, Status> {
        let r = req.into_inner();
        let items = self
            .svc
            .list(r.limit, r.offset)
            .await
            .map_err(|e| Status::internal(e.to_string()))?;
        let total = items.len() as i32;
        Ok(Response::new(ListProductsResponse {
            items: items.iter().map(to_proto).collect(),
            total,
        }))
    }

    async fn create_product(
        &self,
        req: Request<CreateProductRequest>,
    ) -> Result<Response<ProtoProduct>, Status> {
        let r = req.into_inner();
        let created = self
            .svc
            .create(CreateProductInput {
                sku: r.sku,
                name: r.name,
                description: r.description,
                price: r.price,
                stock: r.stock,
                catalog_id: r.catalog_id,
            })
            .await
            .map_err(|e| Status::internal(e.to_string()))?;
        Ok(Response::new(to_proto(&created)))
    }

    async fn decrement_stock(
        &self,
        req: Request<DecrementStockRequest>,
    ) -> Result<Response<DecrementStockResponse>, Status> {
        let r = req.into_inner();
        let id =
            Uuid::parse_str(&r.id).map_err(|_| Status::invalid_argument("bad uuid"))?;
        let remaining = self
            .svc
            .decrement_stock(id, r.quantity)
            .await
            .map_err(|e| Status::internal(e.to_string()))?;
        match remaining {
            Some(v) => Ok(Response::new(DecrementStockResponse {
                ok: true,
                remaining: v,
            })),
            None => Ok(Response::new(DecrementStockResponse {
                ok: false,
                remaining: 0,
            })),
        }
    }
}

fn to_proto(p: &Product) -> ProtoProduct {
    ProtoProduct {
        id: p.id.to_string(),
        sku: p.sku.clone(),
        name: p.name.clone(),
        description: p.description.clone(),
        price: p.price,
        stock: p.stock,
        catalog_id: p.catalog_id.clone(),
    }
}
