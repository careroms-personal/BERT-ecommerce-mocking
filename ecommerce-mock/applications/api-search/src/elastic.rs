use reqwest::{Client, StatusCode};
use serde_json::{json, Value};

pub const INDEX: &str = "products";

#[derive(Clone)]
pub struct EsClient {
    pub base_url: String,
    pub client: Client,
}

impl EsClient {
    pub fn new(base_url: String) -> Self {
        Self {
            base_url,
            client: Client::builder()
                .timeout(std::time::Duration::from_secs(5))
                .build()
                .expect("failed to build HTTP client"),
        }
    }

    pub async fn ping(&self) -> bool {
        self.client
            .get(format!("{}/_cluster/health", self.base_url))
            .send()
            .await
            .map(|r| r.status().is_success())
            .unwrap_or(false)
    }

    pub async fn search(
        &self,
        query: &str,
        category: Option<&str>,
        page: u32,
        size: u32,
    ) -> Result<Value, String> {
        let from = (page.saturating_sub(1)) * size;

        let mut must: Vec<Value> = vec![json!({
            "multi_match": {
                "query": query,
                "fields": ["name^3", "description", "category"],
                "type": "best_fields",
                "fuzziness": "AUTO"
            }
        })];

        let mut filter: Vec<Value> = vec![];
        if let Some(cat) = category {
            if !cat.is_empty() {
                filter.push(json!({ "term": { "category.keyword": cat } }));
            }
        }

        let es_query = json!({
            "query": { "bool": { "must": must, "filter": filter } },
            "from": from,
            "size": size,
            "_source": ["id", "name", "description", "price", "category", "stock"]
        });

        let resp = self
            .client
            .post(format!("{}/{}/_search", self.base_url, INDEX))
            .json(&es_query)
            .send()
            .await
            .map_err(|e| e.to_string())?;

        if !resp.status().is_success() {
            return Err(format!("ES returned {}", resp.status()));
        }

        resp.json::<Value>().await.map_err(|e| e.to_string())
    }

    pub async fn suggest(&self, prefix: &str) -> Result<Value, String> {
        let es_query = json!({
            "query": {
                "match_phrase_prefix": {
                    "name": { "query": prefix, "max_expansions": 10 }
                }
            },
            "_source": ["id", "name", "category"],
            "size": 8
        });

        let resp = self
            .client
            .post(format!("{}/{}/_search", self.base_url, INDEX))
            .json(&es_query)
            .send()
            .await
            .map_err(|e| e.to_string())?;

        if !resp.status().is_success() {
            return Err(format!("ES returned {}", resp.status()));
        }

        resp.json::<Value>().await.map_err(|e| e.to_string())
    }

    pub async fn index_document(&self, doc: &Value) -> Result<Value, String> {
        let id = doc["id"].as_str().unwrap_or("unknown");

        let resp = self
            .client
            .put(format!("{}/{}/{}/_doc/{}", self.base_url, INDEX, INDEX, id))
            .json(doc)
            .send()
            .await
            .map_err(|e| e.to_string())?;

        let status = resp.status();
        let body = resp.json::<Value>().await.map_err(|e| e.to_string())?;

        if !status.is_success() {
            return Err(format!("ES index failed: {}", body));
        }

        Ok(body)
    }

    pub async fn ensure_index(&self) -> Result<(), String> {
        let url = format!("{}/{}", self.base_url, INDEX);

        // Check if index exists
        let exists = self
            .client
            .head(&url)
            .send()
            .await
            .map(|r| r.status() == StatusCode::OK)
            .unwrap_or(false);

        if exists {
            return Ok(());
        }

        // Create index with mapping
        let mapping = json!({
            "mappings": {
                "properties": {
                    "id":          { "type": "keyword" },
                    "name":        { "type": "text", "fields": { "keyword": { "type": "keyword" } } },
                    "description": { "type": "text" },
                    "category":    { "type": "text", "fields": { "keyword": { "type": "keyword" } } },
                    "price":       { "type": "double" },
                    "stock":       { "type": "integer" }
                }
            }
        });

        let resp = self
            .client
            .put(&url)
            .json(&mapping)
            .send()
            .await
            .map_err(|e| e.to_string())?;

        if !resp.status().is_success() {
            return Err(format!("failed to create index: {}", resp.status()));
        }

        Ok(())
    }
}
