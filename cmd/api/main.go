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
	"github.com/rahadianap/community-waste-api/internal/database"
	httpHandler "github.com/rahadianap/community-waste-api/internal/handler/http"
	"github.com/rahadianap/community-waste-api/internal/handler/middleware"
	"github.com/rahadianap/community-waste-api/internal/repository"
	"github.com/rahadianap/community-waste-api/internal/response"
	"github.com/rahadianap/community-waste-api/internal/service"
	"github.com/rahadianap/community-waste-api/internal/uploader"
)

func main() {
	log.Println("Starting API...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("gagal memuat config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL())
	if err != nil {
		log.Fatalf("gagal terhubung ke database: %v", err)
	}
	defer pool.Close()

	migrationsPath := "db/migrations"
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		migrationsPath = "/app/db/migrations"
	}
	if err := database.RunMigrations(cfg.DatabaseURL(), migrationsPath); err != nil {
		log.Printf("Warning saat migrasi database: %v", err)
	}

	store := repository.NewStore(pool)
	fileUploader := uploader.NewLocalUploader(cfg.UploadDir, cfg.MaxUploadSizeMB)

	householdSvc := service.NewHouseholdService(store)
	pickupSvc := service.NewPickupService(store)
	paymentSvc := service.NewPaymentService(store, fileUploader)
	reportSvc := service.NewReportService(store)

	handlers := httpHandler.Handlers{
		Household: httpHandler.NewHouseholdHandler(householdSvc),
		Pickup:    httpHandler.NewPickupHandler(pickupSvc),
		Payment:   httpHandler.NewPaymentHandler(paymentSvc),
		Report:    httpHandler.NewReportHandler(reportSvc),
	}

	app := fiber.New(fiber.Config{
		AppName:      "Community Waste Collection API",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		BodyLimit:    cfg.MaxUploadSizeMB * 1024 * 1024,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			message := "Internal Server Error"
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
				message = e.Message
			}
			return c.Status(code).JSON(response.APIResponse{
				StatusCode: code,
				Success:    false,
				Message:    message,
			})
		},
	})

	middleware.SetupMiddlewares(app)
	httpHandler.RegisterRoutes(app, handlers)

	addr := fmt.Sprintf(":%s", cfg.AppPort)
	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("Server berjalan pada %s (env: %s)", addr, cfg.AppEnv)
		if err := app.Listen(addr); err != nil {
			serverErrors <- err
		}
	}()

	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Printf("Server gagal berjalan: %v", err)
		return
	case sig := <-channel:
		log.Printf("Menerima sinyal (%v), mematikan server...", sig)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("Error saat mematikan server: %v", err)
	}

	log.Println("Server berjalan dengan normal")
}
