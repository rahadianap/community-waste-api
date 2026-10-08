package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rahadianap/community-waste-api/internal/service"
	"github.com/rahadianap/community-waste-api/pkg/response"
)

type ReportHandler struct {
	service service.ReportService
}

func NewReportHandler(service service.ReportService) *ReportHandler {
	return &ReportHandler{service: service}
}

func (h *ReportHandler) GetWasteSummary(c *fiber.Ctx) error {
	summary, err := h.service.GetWasteSummary(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to retrieve waste summary report", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Waste summary report retrieved successfully", summary)
}

func (h *ReportHandler) GetPaymentSummary(c *fiber.Ctx) error {
	summary, err := h.service.GetPaymentSummary(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to retrieve payment summary report", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Payment summary report retrieved successfully", summary)
}
