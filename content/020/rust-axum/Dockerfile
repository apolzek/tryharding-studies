FROM rust:1.87-slim AS builder
WORKDIR /app

RUN apt-get update && apt-get install -y pkg-config libssl-dev protobuf-compiler && rm -rf /var/lib/apt/lists/*

# Cache dependency compilation by building a stub first
COPY Cargo.toml ./
RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo build --release 2>/dev/null || true
RUN rm -f target/release/deps/rust_axum* target/release/rust-axum

# Build the real source
COPY src ./src
RUN cargo build --release

FROM debian:bookworm-slim AS runtime
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/target/release/rust-axum /usr/local/bin/rust-axum
EXPOSE 8083
CMD ["rust-axum"]
