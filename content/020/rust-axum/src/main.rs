use axum::{
    extract::{Json, Request},
    http::{HeaderMap, StatusCode},
    middleware::{from_fn, Next},
    response::{IntoResponse, Response},
    routing::{get, post},
    Router,
};
use opentelemetry::{global, KeyValue};
use opentelemetry::propagation::Extractor;
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::{propagation::TraceContextPropagator, runtime};
use serde::{Deserialize, Serialize};
use std::env;
use tracing::{info, Instrument};
use tracing_opentelemetry::OpenTelemetrySpanExt;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

// ── Models ────────────────────────────────────────────────────────────────────

#[derive(Debug, Deserialize)]
struct ScoreRequest {
    application_id: String,
    customer_id: String,
    amount_usd: f64,
    credit_score: i32,
    account_age_days: i32,
    monthly_income: f64,
    currency: String,
}

#[derive(Debug, Serialize)]
struct ScoreResponse {
    application_id: String,
    fraud_score: f64,
    risk_level: String,
    decision: String,
    reason: String,
    max_approved_amount_usd: f64,
}

// ── OTel bootstrap ────────────────────────────────────────────────────────────

fn init_otel(endpoint: &str) -> opentelemetry_sdk::trace::Tracer {
    global::set_text_map_propagator(TraceContextPropagator::new());

    let service_name = env::var("OTEL_SERVICE_NAME")
        .unwrap_or_else(|_| "rust-axum".to_string());

    let resource = opentelemetry_sdk::Resource::new(vec![
        KeyValue::new(
            opentelemetry_semantic_conventions::resource::SERVICE_NAME,
            service_name.clone(),
        ),
        KeyValue::new(
            opentelemetry_semantic_conventions::resource::SERVICE_VERSION,
            "1.0.0",
        ),
    ]);

    let _meter_provider = opentelemetry_otlp::new_pipeline()
        .metrics(runtime::Tokio)
        .with_exporter(
            opentelemetry_otlp::new_exporter()
                .tonic()
                .with_endpoint(endpoint),
        )
        .with_resource(resource.clone())
        .with_period(std::time::Duration::from_secs(10))
        .build()
        .expect("metrics pipeline");

    opentelemetry_otlp::new_pipeline()
        .tracing()
        .with_exporter(
            opentelemetry_otlp::new_exporter()
                .tonic()
                .with_endpoint(endpoint),
        )
        .with_trace_config(
            opentelemetry_sdk::trace::config().with_resource(resource),
        )
        .install_batch(runtime::Tokio)
        .expect("trace pipeline")
}

// ── W3C context propagation middleware ───────────────────────────────────────
// Extracts the incoming traceparent header and sets it as the parent of a new
// server span. Business handlers have zero OTel code.

struct HeaderExtractor<'a>(&'a HeaderMap);

impl<'a> Extractor for HeaderExtractor<'a> {
    fn get(&self, key: &str) -> Option<&str> {
        self.0.get(key).and_then(|v| v.to_str().ok())
    }
    fn keys(&self) -> Vec<&str> {
        self.0.keys().map(|k| k.as_str()).collect()
    }
}

async fn otel_middleware(headers: HeaderMap, req: Request, next: Next) -> Response {
    let parent_cx = global::get_text_map_propagator(|prop| {
        prop.extract(&HeaderExtractor(&headers))
    });

    let span = tracing::info_span!("http.server.request");
    span.set_parent(parent_cx);

    next.run(req).instrument(span).await
}

// ── Business logic ────────────────────────────────────────────────────────────

// #[tracing::instrument] creates a child span automatically — no manual code.
#[tracing::instrument(skip(req), fields(
    application_id = %req.application_id,
    credit_score   = req.credit_score,
    amount_usd     = req.amount_usd,
))]
fn calculate_fraud_score(req: &ScoreRequest) -> f64 {
    let mut score = 0.0_f64;
    let credit_factor = ((850 - req.credit_score) as f64) / 550.0;
    score += credit_factor * 35.0;
    let ratio = req.amount_usd / (req.monthly_income * 12.0).max(1.0);
    score += (ratio * 30.0).min(30.0);
    let age_factor = 1.0 - (req.account_age_days as f64 / 3650.0).min(1.0);
    score += age_factor * 20.0;
    if req.amount_usd > 50_000.0 {
        score += 15.0;
    }
    score.min(100.0)
}

fn risk_level(score: f64) -> &'static str {
    match score as u32 {
        0..=29  => "LOW",
        30..=59 => "MEDIUM",
        60..=79 => "HIGH",
        _       => "VERY_HIGH",
    }
}

fn max_approved_amount(credit_score: i32, monthly_income: f64) -> f64 {
    let base = monthly_income * 12.0;
    let multiplier = match credit_score {
        750..=850 => 5.0,
        700..=749 => 3.5,
        650..=699 => 2.0,
        600..=649 => 1.0,
        _         => 0.5,
    };
    (base * multiplier).min(500_000.0)
}

// ── Handler ───────────────────────────────────────────────────────────────────
// No OTel code here — spans are created by otel_middleware and
// #[tracing::instrument] on calculate_fraud_score.

async fn score_handler(Json(payload): Json<ScoreRequest>) -> impl IntoResponse {
    info!(application_id = %payload.application_id, "Starting fraud score calculation");

    let fraud_score = calculate_fraud_score(&payload);
    let level       = risk_level(fraud_score);
    let max_amount  = max_approved_amount(payload.credit_score, payload.monthly_income);

    let (decision, reason) = if payload.amount_usd > max_amount {
        ("REJECTED", format!("Amount exceeds approved limit ${:.2}", max_amount))
    } else if fraud_score >= 80.0 {
        ("REJECTED", "Fraud score too high".to_string())
    } else if fraud_score >= 60.0 {
        ("MANUAL_REVIEW", "Elevated risk — manual review required".to_string())
    } else {
        ("APPROVED", format!("Risk within range (score={:.1})", fraud_score))
    };

    info!(fraud_score, risk_level = level, decision, "Scoring complete");

    (
        StatusCode::OK,
        Json(ScoreResponse {
            application_id:        payload.application_id,
            fraud_score,
            risk_level:            level.to_string(),
            decision:              decision.to_string(),
            reason,
            max_approved_amount_usd: max_amount,
        }),
    )
}

async fn health() -> impl IntoResponse {
    Json(serde_json::json!({ "status": "ok", "service": "rust-axum" }))
}

// ── Main ─────────────────────────────────────────────────────────────────────

#[tokio::main]
async fn main() {
    let endpoint = env::var("OTEL_EXPORTER_OTLP_ENDPOINT")
        .unwrap_or_else(|_| "http://localhost:4317".to_string());

    let tracer = init_otel(&endpoint);

    let otel_layer = tracing_opentelemetry::layer().with_tracer(tracer);

    tracing_subscriber::registry()
        .with(EnvFilter::try_from_default_env().unwrap_or_else(|_| "info".into()))
        .with(tracing_subscriber::fmt::layer())
        .with(otel_layer)
        .init();

    let app = Router::new()
        .route("/api/loan/score", post(score_handler))
        .route("/health", get(health))
        // otel_middleware handles W3C context propagation for every request
        .layer(from_fn(otel_middleware));

    let addr = "0.0.0.0:8083";
    info!("rust-axum listening on {}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app)
        .with_graceful_shutdown(async {
            tokio::signal::ctrl_c().await.ok();
            global::shutdown_tracer_provider();
        })
        .await
        .unwrap();
}
