package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/yourusername/saga-service/internal/config"
	"github.com/yourusername/saga-service/internal/handlers"
	"github.com/yourusername/saga-service/internal/repositories/postgres"
	"github.com/yourusername/saga-service/internal/services"
	"github.com/yourusername/saga-service/pkg/logger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"log"
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
			semconv.ServiceNameKey.String(cfg.OtelServiceName),
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
	defer logger.Sync()

	shutdown := initTracer(cfg)
	defer shutdown()

	repo, err := postgres.NewCreateOrderSagaPostRepo(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Close()

	orderSagaService := services.NewOrderSagaService(repo, cfg)
	orderSagaHandlers := handlers.NewOrderSagaHandler(orderSagaService)

	router := gin.Default()
	router.Use(otelgin.Middleware(cfg.OtelServiceName))

	api := router.Group("/api/v1")
	{
		api.POST("/start_saga", orderSagaHandlers.StartSaga)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Printf("Inventory service is running on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
