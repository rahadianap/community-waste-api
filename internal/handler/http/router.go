package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rahadianap/community-waste-api/pkg/response"
)

type Handlers struct {
	Household *HouseholdHandler
	Pickup    *PickupHandler
	Payment   *PaymentHandler
	Report    *ReportHandler
}

func RegisterRoutes(app *fiber.App, h Handlers, uploadDir string) {
	// Static file serving for uploaded files
	app.Static("/uploads", "./uploads")

	// Health Check
	app.Get("/health", func(c *fiber.Ctx) error {
		return response.Success(c, fiber.StatusOK, "Community Waste API is healthy", fiber.Map{
			"status": "UP",
		})
	})

	api := app.Group("/api")

	// 1. Household Routes
	households := api.Group("/households")
	households.Post("/", h.Household.Create)
	households.Get("/", h.Household.List)
	households.Get("/:id", h.Household.GetByID)
	households.Delete("/:id", h.Household.Delete)

	// 2. Waste Pickup Routes
	pickups := api.Group("/pickups")
	pickups.Post("/", h.Pickup.Create)
	pickups.Get("/", h.Pickup.List)
	pickups.Get("/:id", h.Pickup.GetByID)
	pickups.Put("/:id/schedule", h.Pickup.Schedule)
	pickups.Put("/:id/complete", h.Pickup.Complete)
	pickups.Put("/:id/cancel", h.Pickup.Cancel)

	// 3. Payment Routes
	payments := api.Group("/payments")
	payments.Post("/", h.Payment.Create)
	payments.Get("/", h.Payment.List)
	payments.Get("/:id", h.Payment.GetByID)
	payments.Put("/:id/confirm", h.Payment.Confirm)

	// 4. Reporting Routes
	reports := api.Group("/reports")
	reports.Get("/waste-summary", h.Report.GetWasteSummary)
	reports.Get("/payment-summary", h.Report.GetPaymentSummary)
}
