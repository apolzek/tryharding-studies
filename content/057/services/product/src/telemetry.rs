//! Minimal tracing setup.
//!
//! The opentelemetry-otlp crate API has churned across minor versions;
//! to keep the build stable we default to stdout tracing via
//! `tracing-subscriber`. Structured logs are still captured by the
//! collector through stdout — the auto-instrumented services produce
//! the rich OTel traces of the end-to-end flow.
use tracing_subscriber::{fmt, prelude::*, EnvFilter};

pub struct Guard;

pub fn init(_service_name: &str) -> Guard {
    tracing_subscriber::registry()
        .with(EnvFilter::try_from_default_env().unwrap_or_else(|_| EnvFilter::new("info")))
        .with(fmt::layer().with_target(true).json())
        .init();
    Guard
}
