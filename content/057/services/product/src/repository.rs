use async_trait::async_trait;
use deadpool_postgres::Pool;
use uuid::Uuid;

#[derive(Clone, Debug)]
pub struct Product {
    pub id: Uuid,
    pub sku: String,
    pub name: String,
    pub description: String,
    pub price: f64,
    pub stock: i32,
    pub catalog_id: String,
}

#[async_trait]
pub trait ProductRepository: Send + Sync {
    async fn create(&self, p: Product) -> anyhow::Result<Product>;
    async fn get(&self, id: Uuid) -> anyhow::Result<Option<Product>>;
    async fn list(&self, limit: i32, offset: i32) -> anyhow::Result<Vec<Product>>;
    async fn decrement_stock(&self, id: Uuid, qty: i32) -> anyhow::Result<Option<i32>>;
}

pub struct PgProductRepository {
    pool: Pool,
}

impl PgProductRepository {
    pub fn new(pool: Pool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl ProductRepository for PgProductRepository {
    async fn create(&self, p: Product) -> anyhow::Result<Product> {
        let client = self.pool.get().await?;
        client
            .execute(
                "INSERT INTO products (id, sku, name, description, price, stock, catalog_id) \
                 VALUES ($1,$2,$3,$4,$5,$6,$7)",
                &[
                    &p.id,
                    &p.sku,
                    &p.name,
                    &p.description,
                    &p.price,
                    &p.stock,
                    &p.catalog_id,
                ],
            )
            .await?;
        Ok(p)
    }

    async fn get(&self, id: Uuid) -> anyhow::Result<Option<Product>> {
        let client = self.pool.get().await?;
        let row = client
            .query_opt(
                "SELECT id, sku, name, description, price, stock, catalog_id FROM products WHERE id=$1",
                &[&id],
            )
            .await?;
        Ok(row.map(row_to_product))
    }

    async fn list(&self, limit: i32, offset: i32) -> anyhow::Result<Vec<Product>> {
        let client = self.pool.get().await?;
        let limit = if limit <= 0 { 50 } else { limit };
        let rows = client
            .query(
                "SELECT id, sku, name, description, price, stock, catalog_id FROM products \
                 ORDER BY sku LIMIT $1 OFFSET $2",
                &[&(limit as i64), &(offset as i64)],
            )
            .await?;
        Ok(rows.into_iter().map(row_to_product).collect())
    }

    async fn decrement_stock(&self, id: Uuid, qty: i32) -> anyhow::Result<Option<i32>> {
        let client = self.pool.get().await?;
        let row = client
            .query_opt(
                "UPDATE products SET stock = stock - $1 WHERE id=$2 AND stock >= $1 RETURNING stock",
                &[&qty, &id],
            )
            .await?;
        Ok(row.map(|r| r.get::<_, i32>("stock")))
    }
}

fn row_to_product(r: tokio_postgres::Row) -> Product {
    Product {
        id: r.get("id"),
        sku: r.get("sku"),
        name: r.get("name"),
        description: r.get::<_, Option<String>>("description").unwrap_or_default(),
        price: r.get("price"),
        stock: r.get("stock"),
        catalog_id: r.get::<_, Option<String>>("catalog_id").unwrap_or_default(),
    }
}
