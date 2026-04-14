pub struct Config {
    pub port: u16,
    pub elasticsearch_url: String,
}

impl Config {
    pub fn from_env() -> Self {
        Self {
            port: std::env::var("PORT")
                .unwrap_or_else(|_| "8086".to_string())
                .parse()
                .unwrap_or(8086),
            elasticsearch_url: std::env::var("ELASTICSEARCH_URL")
                .unwrap_or_else(|_| "http://elasticsearch:9200".to_string()),
        }
    }
}
