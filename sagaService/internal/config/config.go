package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
)

type Config struct {
	ServerPort string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	JWTSecret         string
	JWTRefreshSecret  string
	JWTExpiryHours    int
	RefreshExpiryDays int

	OtelExporterOtlpEndpoint string
	OtelServiceName          string

	OrderServiceUrl     string
	InventoryServiceUrl string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "saga_postgres"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "password"),
		DBName:     getEnv("DB_NAME", "saga_db"),

		JWTSecret:         getEnv("JWT_SECRET", "super_secret_key"),
		JWTRefreshSecret:  getEnv("JWT_REFRESH_SECRET", "super_secret_key"),
		JWTExpiryHours:    getEnvAsInt("JWT_EXPIRY_HOURS", 24),
		RefreshExpiryDays: getEnvAsInt("REFRESH_EXPIRY_DAYS", 7),

		OtelExporterOtlpEndpoint: getEnv("OTEL_EXPORTER_OTLP_ENDPOIN", "jaeger:4317"),
		OtelServiceName:          getEnv("OTEL_SERVICE_NAME", "saga-service"),

		OrderServiceUrl:     getEnv("ORDER_SERVICE_URL", "http://order-service:8080"),
		InventoryServiceUrl: getEnv("INVENTORY_URL", "http://inventory-service:8080"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}
