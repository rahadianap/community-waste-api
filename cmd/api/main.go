package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rahadianap/community-waste-api/internal/config"
	httpHandler "github.com/rahadianap/community-waste-api/internal/handler/http"
	"github.com/rahadianap/community-waste-api/internal/handler/middleware"
	"github.com/rahadianap/community-waste-api/internal/repository"
	"github.com/rahadianap/community-waste-api/internal/service"
	"github.com/rahadianap/community-waste-api/pkg/database"
	"github.com/rahadianap/community-waste-api/pkg/uploader"
)

func main() {
	log.Println("Starting Community Waste Collection API...")

	// 1. Load Configurations
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 2. Initialize Database Connection Pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// 3. Run Database Migrations
	migrationsPath := "db/migrations"
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		migrationsPath = "/app/db/migrations"
	}
	if err := database.RunMigrations(cfg.DatabaseURL(), migrationsPath); err != nil {
		log.Printf("Warning during database migrations: %v", err)
	}

	// 4. Initialize Store & Layer Components
	store := repository.NewStore(pool)
	fileUploader := uploader.NewLocalUploader(cfg.UploadDir, cfg.MaxUploadSizeMB)

	// Services
	householdSvc := service.NewHouseholdService(store)
	pickupSvc := service.NewPickupService(store)
	paymentSvc := service.NewPaymentService(store, fileUploader)
	reportSvc := service.NewReportService(store)

	// Handlers
	handlers := httpHandler.Handlers{
		Household: httpHandler.NewHouseholdHandler(householdSvc),
		Pickup:    httpHandler.NewPickupHandler(pickupSvc),
		Payment:   httpHandler.NewPaymentHandler(paymentSvc),
		Report:    httpHandler.NewReportHandler(reportSvc),
	}

	// 5. Initialize Fiber App
	app := fiber.New(fiber.Config{
		AppName:      "Community Waste Collection API v1.0",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		BodyLimit:    cfg.MaxUploadSizeMB * 1024 * 1024,
	})

	// Middlewares
	middleware.SetupMiddlewares(app)

	// Routes
	httpHandler.RegisterRoutes(app, handlers, cfg.UploadDir)

	// 6. Start Server with Graceful Shutdown
	serverShutdown := make(chan struct{})
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Received shutdown signal, gracefully shutting down server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := app.ShutdownWithContext(shutdownCtx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
		}
		close(serverShutdown)
	}()

	addr := fmt.Sprintf(":%s", cfg.AppPort)
	log.Printf("Server listening on %s (env: %s)", addr, cfg.AppEnv)
	if err := app.Listen(addr); err != nil {
		log.Printf("Server stopped: %v", err)
	}

	<-serverShutdown
	log.Println("Server gracefully stopped.")
}
