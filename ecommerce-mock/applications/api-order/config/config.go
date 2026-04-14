package config

import "os"

type Config struct {
	Port              string
	DatabaseURL       string
	CartServiceURL    string
	ProductServiceURL string
}

func Load() *Config {
	return &Config{
		Port:              getEnv("PORT", "8084"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://ecommerce:ecommerce_pass@postgres:5432/order_db?sslmode=disable"),
		CartServiceURL:    getEnv("CART_SERVICE_URL", "http://api-cart:8083"),
		ProductServiceURL: getEnv("PRODUCT_SERVICE_URL", "http://api-product:8081"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
