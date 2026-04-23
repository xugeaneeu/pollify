package config

import "os"

type Config struct {
	HTTPAddress    string
	DatabaseURL    string
	JWTSecret      string
	OpenAPIPath    string
	MigrationsPath string
}

func Load() Config {
	return Config{
		HTTPAddress:    envOrDefault("HTTP_ADDRESS", ":8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		OpenAPIPath:    envOrDefault("OPENAPI_PATH", "../api/openapi.yaml"),
		MigrationsPath: envOrDefault("MIGRATIONS_PATH", "migrations"),
	}
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
