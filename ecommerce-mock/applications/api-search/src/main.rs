mod config;
mod elastic;
mod handlers;

use axum::{
    routing::{get, post},
    Router,
};
use tower_http::trace::TraceLayer;
use tracing::{error, info};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt, EnvFilter};

use crate::config::Config;
use crate::elastic::EsClient;
use crate::handlers::AppState;

#[tokio::main]
async fn main() {
    // ── Tracing — JSON to stdout ──────────────────────────────────────────────
    tracing_subscriber::registry()
        .with(EnvFilter::from_default_env().add_directive("info".parse().unwrap()))
        .with(tracing_subscriber::fmt::layer().json().with_current_span(false))
        .init();

    let cfg = Config::from_env();

    info!(
        service  = "api-search",
        category = "SYSTEM",
        version  = "1.0.0",
        port     = cfg.port,
        "service starting"
    );

    // ── Elasticsearch client ──────────────────────────────────────────────────
    let es = EsClient::new(cfg.elasticsearch_url.clone());

    // Try to reach ES — log warning if unavailable, continue anyway
    let mut attempts = 0u32;
    loop {
        if es.ping().await {
            info!(service = "api-search", category = "SYSTEM", "elasticsearch connected");
            match es.ensure_index().await {
                Ok(_)  => info!(service = "api-search", category = "SYSTEM", index = elastic::INDEX, "products index ready"),
                Err(e) => error!(service = "api-search", category = "SEARCH_UNAVAILABLE", error = %e, "failed to ensure index"),
            }
            break;
        }
        attempts += 1;
        if attempts >= 10 {
            error!(service = "api-search", category = "SEARCH_UNAVAILABLE", "elasticsearch not reachable, starting without it — search requests will return errors");
            break;
        }
        info!(service = "api-search", category = "SYSTEM", attempt = attempts, "waiting for elasticsearch...");
        tokio::time::sleep(std::time::Duration::from_secs(3)).await;
    }

    let state = AppState { es };

    // ── Router ────────────────────────────────────────────────────────────────
    let app = Router::new()
        .route("/health",         get(handlers::health))
        .route("/search",         get(handlers::search))
        .route("/search/suggest", get(handlers::suggest))
        .route("/search/index",   post(handlers::index_document))
        .with_state(state)
        .layer(TraceLayer::new_for_http());

    let addr = format!("0.0.0.0:{}", cfg.port);
    let listener = tokio::net::TcpListener::bind(&addr).await.unwrap();

    info!(
        service  = "api-search",
        category = "SYSTEM",
        addr     = %addr,
        "server listening"
    );

    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await
        .unwrap();

    info!(service = "api-search", category = "SYSTEM", "service stopped");
}

async fn shutdown_signal() {
    use tokio::signal;

    let ctrl_c = async { signal::ctrl_c().await.expect("failed to install Ctrl+C handler") };

    #[cfg(unix)]
    let terminate = async {
        signal::unix::signal(signal::unix::SignalKind::terminate())
            .expect("failed to install SIGTERM handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c   => info!(service = "api-search", category = "SYSTEM", signal = "SIGINT",  "shutdown signal received"),
        _ = terminate => info!(service = "api-search", category = "SYSTEM", signal = "SIGTERM", "shutdown signal received"),
    }
}
