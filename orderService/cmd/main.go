package main

import (
	"context"
	"github.com/yourusername/order-service/internal/repository"
	"github.com/yourusername/order-service/pkg/logger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/propagation"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/order-service/internal/config"
	"github.com/yourusername/order-service/internal/handler"
	"github.com/yourusername/order-service/internal/service"
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
	// Инициализируем логгер
	logger.Init()
	defer logger.Sync() // Сброс буферов при завершении

	cfg := config.LoadConfig()

	// Jaeger
	shutdown := initTracer()
	defer shutdown()

	// Подключение к БД
	repo, err := repository.NewOrderRepository(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer repo.Close()

	// Сервисы
	orderService := service.NewOrderService(repo)

	// Хендлеры
	orderhandler := handler.NewOrderHandler(orderService)

	// Роутер
	router := gin.Default()

	router.Use(otelgin.Middleware("user-service"))

	// Роуты
	order := router.Group("/api")
	{
		order.POST("/create_order", orderhandler.CreateOrder)
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Printf("Order service is running on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
