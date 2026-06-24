package config

import "os"

const (
	defaultDatabaseURL = "postgres://takehome:takehome@localhost:5438/inventory?sslmode=disable"
	defaultAPIAddr     = ":8080"
	defaultEventsDir   = "../data/events"
	defaultProductsCSV = "../data/products.csv"
	defaultMigrations  = "migrations"
)

type Config struct {
	DatabaseURL string
	APIAddr     string
	EventsDir   string
	ProductsCSV string
	Migrations  string
}

func Load() Config {
	return Config{
		DatabaseURL: envOrDefault("DATABASE_URL", defaultDatabaseURL),
		APIAddr:     envOrDefault("API_ADDR", defaultAPIAddr),
		EventsDir:   envOrDefault("EVENTS_DIR", defaultEventsDir),
		ProductsCSV: envOrDefault("PRODUCTS_CSV", defaultProductsCSV),
		Migrations:  envOrDefault("MIGRATIONS_DIR", defaultMigrations),
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
