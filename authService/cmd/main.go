package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/yourusername/user-service/internal/config"
	"github.com/yourusername/user-service/internal/handler"
	"github.com/yourusername/user-service/internal/repository/postgres"
	"github.com/yourusername/user-service/internal/service"
	"github.com/yourusername/user-service/pkg/jwt"
)

func main() {
	cfg := config.LoadConfig()

	// Подключение к БД
	repo, err := postgres.NewUserRepository(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer repo.Close()

	// JWT менеджер
	jwtManager := jwt.NewJWTManager(
		cfg.JWTSecret,
		cfg.JWTRefreshSecret,
		cfg.JWTExpiryHours,
		cfg.RefreshExpiryDays,
	)

	// Сервисы
	authService := service.NewAuthService(repo, jwtManager)

	// Хендлеры
	authHandler := handler.NewAuthHandler(authService)

	// Роутер
	router := gin.Default()

	// Роуты
	auth := router.Group("/api")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/verify", authHandler.Verify)
		auth.POST("/register", authHandler.Register)
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Printf("User service is running on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
