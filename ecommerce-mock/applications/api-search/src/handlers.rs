use axum::{
    extract::{Query, State},
    http::StatusCode,
    response::IntoResponse,
    Json,
};
use serde::Deserialize;
use serde_json::{json, Value};
use tracing::{error, info, warn};

use crate::elastic::EsClient;

#[derive(Clone)]
pub struct AppState {
    pub es: EsClient,
}

// ── GET /health ───────────────────────────────────────────────────────────────
pub async fn health(State(state): State<AppState>) -> impl IntoResponse {
    let alive = state.es.ping().await;
    if alive {
        info!(service = "api-search", category = "SYSTEM", "health check ok");
        Json(json!({ "status": "ok", "service": "api-search", "elasticsearch": "up" }))
            .into_response()
    } else {
        warn!(service = "api-search", category = "SEARCH_UNAVAILABLE", "health check — elasticsearch unreachable");
        (
            StatusCode::SERVICE_UNAVAILABLE,
            Json(json!({ "status": "degraded", "elasticsearch": "down" })),
        )
            .into_response()
    }
}

// ── GET /search?q=&category=&page= ───────────────────────────────────────────
#[derive(Deserialize)]
pub struct SearchParams {
    pub q:        Option<String>,
    pub category: Option<String>,
    pub page:     Option<u32>,
}

pub async fn search(
    State(state): State<AppState>,
    Query(params): Query<SearchParams>,
) -> impl IntoResponse {
    let q = match params.q.as_deref().filter(|s| !s.is_empty()) {
        Some(q) => q.to_string(),
        None => {
            warn!(service = "api-search", category = "ERROR", "search called without query param");
            return (
                StatusCode::BAD_REQUEST,
                Json(json!({ "error": "query param 'q' is required" })),
            )
                .into_response();
        }
    };

    let page     = params.page.unwrap_or(1).max(1);
    let category = params.category.as_deref();

    info!(
        service  = "api-search",
        category = "SEARCH",
        query    = %q,
        page     = page,
        filter_category = category.unwrap_or(""),
        "executing search query"
    );

    match state.es.search(&q, category, page, 10).await {
        Ok(body) => {
            let hits      = body["hits"]["hits"].as_array().cloned().unwrap_or_default();
            let total     = body["hits"]["total"]["value"].as_u64().unwrap_or(0);
            let took_ms   = body["took"].as_u64().unwrap_or(0);

            info!(
                service      = "api-search",
                category     = "SEARCH",
                query        = %q,
                results      = hits.len(),
                total        = total,
                took_ms      = took_ms,
                page         = page,
                "search completed"
            );

            let results: Vec<Value> = hits
                .iter()
                .map(|h| h["_source"].clone())
                .collect();

            Json(json!({
                "data":  results,
                "total": total,
                "page":  page,
                "took_ms": took_ms,
            }))
            .into_response()
        }
        Err(e) => {
            error!(
                service  = "api-search",
                category = "SEARCH_UNAVAILABLE",
                query    = %q,
                error    = %e,
                "elasticsearch search failed"
            );
            (
                StatusCode::SERVICE_UNAVAILABLE,
                Json(json!({ "error": "search unavailable", "detail": e })),
            )
                .into_response()
        }
    }
}

// ── GET /search/suggest?q= ────────────────────────────────────────────────────
#[derive(Deserialize)]
pub struct SuggestParams {
    pub q: Option<String>,
}

pub async fn suggest(
    State(state): State<AppState>,
    Query(params): Query<SuggestParams>,
) -> impl IntoResponse {
    let q = match params.q.as_deref().filter(|s| s.len() >= 2) {
        Some(q) => q.to_string(),
        None => {
            return (
                StatusCode::BAD_REQUEST,
                Json(json!({ "error": "query param 'q' must be at least 2 characters" })),
            )
                .into_response();
        }
    };

    info!(
        service  = "api-search",
        category = "SEARCH",
        prefix   = %q,
        "executing suggest query"
    );

    match state.es.suggest(&q).await {
        Ok(body) => {
            let hits: Vec<Value> = body["hits"]["hits"]
                .as_array()
                .cloned()
                .unwrap_or_default()
                .iter()
                .map(|h| h["_source"].clone())
                .collect();

            info!(
                service    = "api-search",
                category   = "SEARCH",
                prefix     = %q,
                suggestions = hits.len(),
                "suggest completed"
            );

            Json(json!({ "suggestions": hits })).into_response()
        }
        Err(e) => {
            error!(
                service  = "api-search",
                category = "SEARCH_UNAVAILABLE",
                prefix   = %q,
                error    = %e,
                "elasticsearch suggest failed"
            );
            (
                StatusCode::SERVICE_UNAVAILABLE,
                Json(json!({ "error": "suggest unavailable", "detail": e })),
            )
                .into_response()
        }
    }
}

// ── POST /search/index ────────────────────────────────────────────────────────
pub async fn index_document(
    State(state): State<AppState>,
    Json(doc): Json<Value>,
) -> impl IntoResponse {
    let doc_id = doc["id"].as_str().unwrap_or("unknown").to_string();

    info!(
        service    = "api-search",
        category   = "SEARCH",
        doc_id     = %doc_id,
        "manual index document requested"
    );

    match state.es.index_document(&doc).await {
        Ok(result) => {
            let action = result["result"].as_str().unwrap_or("indexed");
            info!(
                service  = "api-search",
                category = "SEARCH",
                doc_id   = %doc_id,
                action   = %action,
                "document indexed successfully"
            );
            Json(json!({ "indexed": true, "id": doc_id, "result": action })).into_response()
        }
        Err(e) => {
            error!(
                service  = "api-search",
                category = "SEARCH_UNAVAILABLE",
                doc_id   = %doc_id,
                error    = %e,
                "failed to index document"
            );
            (
                StatusCode::SERVICE_UNAVAILABLE,
                Json(json!({ "error": "index failed", "detail": e })),
            )
                .into_response()
        }
    }
}
