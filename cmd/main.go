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
	fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/koriebruh/payment-service/config"
	"github.com/koriebruh/payment-service/internal/adapter/gateway"
	"github.com/koriebruh/payment-service/internal/adapter/handler"
	"github.com/koriebruh/payment-service/internal/adapter/publisher"
	"github.com/koriebruh/payment-service/internal/adapter/repository"
	"github.com/koriebruh/payment-service/internal/adapter/worker"
	"github.com/koriebruh/payment-service/internal/middleware"
	"github.com/koriebruh/payment-service/internal/usecase"
	"github.com/koriebruh/payment-service/pkg/idempotency"
	"github.com/koriebruh/payment-service/pkg/logger"
	"github.com/koriebruh/payment-service/pkg/response"
	"github.com/koriebruh/payment-service/pkg/tracing"
)

func main() {
	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Setup logger (stdout + file if LOG_FILE_PATH is set)
	logg := logger.New(&cfg.App)
	logg.Info("starting service",
		"name", cfg.App.Name,
		"port", cfg.App.Port,
		"log_level", cfg.App.LogLevel,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Setup OTEL
	shutdownOTEL, err := tracing.SetupOTEL(ctx, &cfg.OTEL)
	if err != nil {
		logg.Error("failed to setup OTEL", "error", err.Error())
	} else {
		defer shutdownOTEL()
	}

	// 4. Connect Postgres
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		cfg.DB.Host, cfg.DB.User, cfg.DB.Password, cfg.DB.Name, cfg.DB.Port)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logg.Error("failed to connect database", "error", err.Error())
		os.Exit(1)
	}
	logg.Info("database connected", "host", cfg.DB.Host, "db", cfg.DB.Name)

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
		logg.Warn("redis connection failed, falling back to memory store", "error", err.Error())
	} else {
		logg.Info("redis connected", "addr", fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port))
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
	webhookUsecase := usecase.NewHandleMidtransWebhookUsecase(txManager, trxRepo, webhookRepo, outboxRepo, cfg.Midtrans.ServerKey)
	getStatusUsecase := usecase.NewGetTransactionStatusUsecase(trxRepo, refundRepo)
	refundUsecase := usecase.NewRefundTransactionUsecase(txManager, trxRepo, refundRepo, midtransClient, idempotencyStore)
	cancelUsecase := usecase.NewCancelTransactionUsecase(txManager, trxRepo, midtransClient)
	getListUsecase := usecase.NewGetTransactionsUsecase(trxRepo)
	getPaymentMethodsUsecase := usecase.NewGetActivePaymentMethodsUsecase(pmRepo)

	// 9. Start Outbox Worker
	outboxWorker := worker.NewOutboxWorker(outboxRepo, kafkaPublisher, txManager, cfg, 2*time.Second)
	go outboxWorker.Start(ctx)

	// 10. Init handlers
	paymentHandler := handler.NewPaymentHandler(chargeUsecase, getStatusUsecase, refundUsecase, cancelUsecase, getListUsecase, validate, respFactory)
	webhookHandler := handler.NewWebhookHandler(webhookUsecase, respFactory)
	paymentMethodHandler := handler.NewPaymentMethodHandler(getPaymentMethodsUsecase, respFactory)
	healthHandler := handler.NewHealthHandler(db, redisClient)

	// 11. Setup Fiber
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			logg.Error("unhandled error", "error", err.Error(), "path", c.Path())
			return c.Status(500).JSON(respFactory.Error("", nil))
		},
	})

	// Middleware order per GEMINI.md rule 16:
	// 1. Recovery  2. RequestID+Logger  3. Prometheus  4. CORS
	app.Use(fiberrecover.New())
	app.Use(middleware.RequestLogger(logg))

	prometheus := fiberprometheus.New(cfg.App.Name)
	prometheus.RegisterAt(app, "/metrics")
	app.Use(prometheus.Middleware)

	app.Use(cors.New())

	// Routes
	app.Get("/health", healthHandler.Check)

	api := app.Group("/api/v1")

	pm := api.Group("/payment-methods")
	pm.Get("/", paymentMethodHandler.GetActiveMethods)

	payments := api.Group("/payments")
	payments.Get("/", paymentHandler.GetTransactions)
	payments.Post("/charge", paymentHandler.Charge)
	payments.Get("/:order_id", paymentHandler.GetStatus)
	payments.Post("/:order_id/refund", paymentHandler.Refund)
	payments.Post("/:order_id/cancel", paymentHandler.Cancel)
	payments.Post("/webhook/midtrans", webhookHandler.MidtransCallback)

	// 12. Start Server gracefully
	go func() {
		port := cfg.App.Port
		if port == 0 {
			port = 8080
		}
		logg.Info("HTTP server listening", "port", port)
		if err := app.Listen(fmt.Sprintf(":%d", port)); err != nil {
			logg.Error("server error", "error", err.Error())
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	logg.Info("graceful shutdown initiated")
	cancel() // Stops outbox worker
	_ = app.Shutdown()
	logg.Info("service stopped")
}
