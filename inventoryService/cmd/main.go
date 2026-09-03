package main

import (
	"context"
	"github.com/yourusername/inventory-service/pkg/logger"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/inventory-service/internal/config"
	"github.com/yourusername/inventory-service/internal/handler"
	"github.com/yourusername/inventory-service/internal/repository/postgres"
	"github.com/yourusername/inventory-service/internal/service"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func initTracer(cfg *config.Config) func() {
	exporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(cfg.OtelExporterOtlpEndpoint),
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

	logger.Init()
	shutdown := initTracer(cfg)
	defer shutdown()

	repo, err := postgres.NewInventoryRepository(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer repo.Close()

	inventoryService := service.NewInventoryService(repo)
	inventoryHandler := handler.NewInventoryHandler(inventoryService)

	router := gin.Default()
	router.Use(otelgin.Middleware("inventory-service"))

	api := router.Group("/api/v1/inventory")
	{
		api.POST("/check", inventoryHandler.CheckAvailability)
		api.POST("/reserve", inventoryHandler.ReserveItems)
		api.POST("/reserve/cancel", inventoryHandler.CancelReservation)
		api.POST("/reserve/confirm", inventoryHandler.ConfirmReservation)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Printf("Inventory service is running on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
