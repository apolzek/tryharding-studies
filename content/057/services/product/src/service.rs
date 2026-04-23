use std::sync::Arc;

use uuid::Uuid;

use crate::repository::{Product, ProductRepository};

pub struct ProductService {
    repo: Arc<dyn ProductRepository>,
}

pub struct CreateProductInput {
    pub sku: String,
    pub name: String,
    pub description: String,
    pub price: f64,
    pub stock: i32,
    pub catalog_id: String,
}

impl ProductService {
    pub fn new(repo: Arc<dyn ProductRepository>) -> Self {
        Self { repo }
    }

    pub async fn create(&self, i: CreateProductInput) -> anyhow::Result<Product> {
        let p = Product {
            id: Uuid::new_v4(),
            sku: i.sku,
            name: i.name,
            description: i.description,
            price: i.price,
            stock: i.stock,
            catalog_id: i.catalog_id,
        };
        self.repo.create(p).await
    }

    pub async fn get(&self, id: Uuid) -> anyhow::Result<Option<Product>> {
        self.repo.get(id).await
    }

    pub async fn list(&self, limit: i32, offset: i32) -> anyhow::Result<Vec<Product>> {
        self.repo.list(limit, offset).await
    }

    pub async fn decrement_stock(&self, id: Uuid, qty: i32) -> anyhow::Result<Option<i32>> {
        self.repo.decrement_stock(id, qty).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use async_trait::async_trait;
    use std::sync::Mutex;

    struct MemRepo {
        items: Mutex<Vec<Product>>,
    }

    #[async_trait]
    impl ProductRepository for MemRepo {
        async fn create(&self, p: Product) -> anyhow::Result<Product> {
            self.items.lock().unwrap().push(p.clone());
            Ok(p)
        }
        async fn get(&self, id: Uuid) -> anyhow::Result<Option<Product>> {
            Ok(self
                .items
                .lock()
                .unwrap()
                .iter()
                .find(|x| x.id == id)
                .cloned())
        }
        async fn list(&self, _l: i32, _o: i32) -> anyhow::Result<Vec<Product>> {
            Ok(self.items.lock().unwrap().clone())
        }
        async fn decrement_stock(&self, id: Uuid, qty: i32) -> anyhow::Result<Option<i32>> {
            let mut items = self.items.lock().unwrap();
            if let Some(p) = items.iter_mut().find(|x| x.id == id) {
                if p.stock >= qty {
                    p.stock -= qty;
                    return Ok(Some(p.stock));
                }
            }
            Ok(None)
        }
    }

    #[tokio::test]
    async fn create_and_decrement() {
        let repo = Arc::new(MemRepo {
            items: Mutex::new(vec![]),
        });
        let svc = ProductService::new(repo);
        let p = svc
            .create(CreateProductInput {
                sku: "ABC".into(),
                name: "Widget".into(),
                description: "".into(),
                price: 9.99,
                stock: 10,
                catalog_id: "c1".into(),
            })
            .await
            .unwrap();
        let remaining = svc.decrement_stock(p.id, 3).await.unwrap();
        assert_eq!(remaining, Some(7));
    }
}
