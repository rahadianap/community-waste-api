package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rahadianap/community-waste-api/internal/response"
)

type Handlers struct {
	Household *HouseholdHandler
	Pickup    *PickupHandler
	Payment   *PaymentHandler
	Report    *ReportHandler
}

func RegisterRoutes(app *fiber.App, h Handlers) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return response.Success(c, fiber.StatusOK, "Community Waste API is healthy", fiber.Map{
			"status": "UP",
		})
	})

	api := app.Group("/api")

	// Household routes
	households := api.Group("/households")
	households.Post("/", h.Household.Create)
	households.Get("/", h.Household.List)
	households.Get("/:id", h.Household.GetByID)
	households.Delete("/:id", h.Household.Delete)

	// Waste pickup routes
	pickups := api.Group("/pickups")
	pickups.Post("/", h.Pickup.Create)
	pickups.Get("/", h.Pickup.List)
	pickups.Put("/:id/schedule", h.Pickup.Schedule)
	pickups.Put("/:id/complete", h.Pickup.Complete)
	pickups.Put("/:id/cancel", h.Pickup.Cancel)

	// Payment routes
	payments := api.Group("/payments")
	payments.Post("/", h.Payment.Create)
	payments.Get("/", h.Payment.List)
	payments.Put("/:id/confirm", h.Payment.Confirm)

	// Report routes
	reports := api.Group("/reports")
	reports.Get("/waste-summary", h.Report.GetWasteSummary)
	reports.Get("/payment-summary", h.Report.GetPaymentSummary)
}
