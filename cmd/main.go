package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/koriebruh/payment-service/config"
	"github.com/koriebruh/payment-service/internal/adapter/gateway"
	"github.com/koriebruh/payment-service/internal/adapter/handler"
	"github.com/koriebruh/payment-service/internal/adapter/publisher"
	"github.com/koriebruh/payment-service/internal/adapter/repository"
	"github.com/koriebruh/payment-service/internal/adapter/worker"
	"github.com/koriebruh/payment-service/internal/usecase"
	"github.com/koriebruh/payment-service/pkg/idempotency"
	"github.com/koriebruh/payment-service/pkg/logger"
	"github.com/koriebruh/payment-service/pkg/tracing"
	"github.com/koriebruh/payment-service/pkg/response"
)

func main() {
	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Setup logger
	logg := logger.New(&cfg.App)
	logg.Info("Starting service", "name", cfg.App.Name)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Setup OTEL
	shutdownOTEL, err := tracing.SetupOTEL(ctx, &cfg.OTEL)
	if err != nil {
		logg.Error("Failed to setup OTEL", "error", err.Error())
	} else {
		defer shutdownOTEL()
	}

	// 4. Connect Postgres
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable", 
		cfg.DB.Host, cfg.DB.User, cfg.DB.Password, cfg.DB.Name, cfg.DB.Port)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logg.Error("Failed to connect database", "error", err.Error())
		os.Exit(1)
	}

	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(cfg.DB.MaxOpenConns)
		sqlDB.SetMaxIdleConns(cfg.DB.MaxIdleConns)
		sqlDB.SetConnMaxLifetime(cfg.DB.ConnMaxLifetime)
		sqlDB.SetConnMaxIdleTime(cfg.DB.ConnMaxIdleTime)
	}

	// 5. Connect Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logg.Error("Failed to connect redis, falling back to memory store", "error", err.Error())
	}

	// 6. Init pkg dependencies
	validate := validator.New()
	respFactory := response.NewApiResponseFactory()
	
	var idempotencyStore idempotency.IdempotencyStore
	if redisClient.Ping(ctx).Err() == nil {
		idempotencyStore = idempotency.NewRedisStore(redisClient)
	} else {
		idempotencyStore = idempotency.NewMemoryStore() // Fallback
	}

	// 7. Init adapters
	txManager := repository.NewGormTxManager(db)
	trxRepo := repository.NewTransactionRepository(db)
	webhookRepo := repository.NewWebhookLogRepository(db)
	outboxRepo := repository.NewOutboxRepository(db)
	pmRepo := repository.NewPaymentMethodRepository(db)
	refundRepo := repository.NewRefundRepository(db)
	midtransClient := gateway.NewMidtransClient(cfg)
	kafkaPublisher := publisher.NewKafkaPublisher(cfg)

	// 8. Init usecases
	chargeUsecase := usecase.NewChargeTransactionUsecase(txManager, trxRepo, midtransClient, idempotencyStore)
	webhookUsecase := usecase.NewHandleMidtransWebhookUsecase(txManager, trxRepo, webhookRepo, outboxRepo, cfg)
	getStatusUsecase := usecase.NewGetTransactionStatusUsecase(trxRepo)
	refundUsecase := usecase.NewRefundTransactionUsecase(txManager, trxRepo, refundRepo, midtransClient, idempotencyStore)
	getPaymentMethodsUsecase := usecase.NewGetActivePaymentMethodsUsecase(pmRepo)

	// 9. Start Outbox Worker
	outboxWorker := worker.NewOutboxWorker(outboxRepo, kafkaPublisher, txManager, cfg, 2*time.Second)
	go outboxWorker.Start(ctx)

	// 10. Init handlers
	paymentHandler := handler.NewPaymentHandler(chargeUsecase, getStatusUsecase, refundUsecase, validate, respFactory)
	webhookHandler := handler.NewWebhookHandler(webhookUsecase, respFactory)
	paymentMethodHandler := handler.NewPaymentMethodHandler(getPaymentMethodsUsecase, respFactory)

	// 11. Setup Fiber
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			logg.Error("unhandled error", "error", err.Error())
			return c.Status(500).JSON(respFactory.Error("", nil))
		},
	})

	// Prometheus metrics
	prometheus := fiberprometheus.New(cfg.App.Name)
	prometheus.RegisterAt(app, "/metrics")
	app.Use(prometheus.Middleware)

	app.Use(recover.New())
	app.Use(cors.New())

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("request_id", "req-"+c.Get("X-Request-Id")) // Should use a proper ID generator
		c.Locals("trace_id", "trace-12345") // OpenTelemetry trace ID should be extracted here
		return c.Next()
	})

	// Routes
	api := app.Group("/api/v1")
	
	pm := api.Group("/payment-methods")
	pm.Get("/", paymentMethodHandler.GetActiveMethods)

	payments := api.Group("/payments")
	payments.Post("/charge", paymentHandler.Charge)
	payments.Get("/:order_id", paymentHandler.GetStatus)
	payments.Post("/:order_id/refund", paymentHandler.Refund)
	payments.Post("/webhook/midtrans", webhookHandler.MidtransCallback)

	// 12. Start Server gracefully
	go func() {
		port := cfg.App.Port
		if port == 0 {
			port = 8080
		}
		if err := app.Listen(fmt.Sprintf(":%d", port)); err != nil {
			logg.Error("server error", "error", err.Error())
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	logg.Info("Gracefully shutting down...")
	cancel() // Stops outbox worker
	_ = app.Shutdown()
}
