package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
)

type Config struct {
	ServerPort        string
	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	JWTSecret         string
	JWTRefreshSecret  string
	JWTExpiryHours    int
	RefreshExpiryDays int
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
		ServerPort:        getEnv("SERVER_PORT", "8080"),
		DBHost:            getEnv("DB_HOST", "postgres-user-service"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USER", "postgres"),
		DBPassword:        getEnv("DB_PASSWORD", "password"),
		DBName:            getEnv("DB_NAME", "users_db"),
		JWTSecret:         getEnv("JWT_SECRET", "super_secret_key"),
		JWTRefreshSecret:  getEnv("JWT_REFRESH_SECRET", "super_secret_key"),
		JWTExpiryHours:    getEnvAsInt("JWT_EXPIRY_HOURS", 24),
		RefreshExpiryDays: getEnvAsInt("REFRESH_EXPIRY_DAYS", 7),
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
