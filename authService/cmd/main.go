package main

import (
	"context"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/propagation"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/user-service/internal/config"
	"github.com/yourusername/user-service/internal/handler"
	"github.com/yourusername/user-service/internal/repository/postgres"
	"github.com/yourusername/user-service/internal/service"
	"github.com/yourusername/user-service/pkg/jwt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func initTracer() func() {
	exporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("Failed to create exporter: %v", err)
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(os.Getenv("OTEL_SERVICE_NAME")),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create resource: %v", err)
	}

	processor := sdktrace.NewSimpleSpanProcessor(exporter)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(processor),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.TraceContext{})

	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}
}

func main() {
	cfg := config.LoadConfig()

	// Jaeger
	shutdown := initTracer()
	defer shutdown()

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

	router.Use(otelgin.Middleware("user-service"))

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
