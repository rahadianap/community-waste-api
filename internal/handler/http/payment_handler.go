package http

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rahadianap/community-waste-api/internal/domain"
	"github.com/rahadianap/community-waste-api/internal/service"
	"github.com/rahadianap/community-waste-api/pkg/response"
)

type PaymentHandler struct {
	service service.PaymentService
}

func NewPaymentHandler(service service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

func (h *PaymentHandler) Create(c *fiber.Ctx) error {
	var req domain.CreatePaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid JSON request body", err.Error())
	}

	result, err := h.service.Create(c.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrHouseholdNotFound) || errors.Is(err, domain.ErrPickupNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "Payment record created successfully", result)
}

func (h *PaymentHandler) List(c *fiber.Ctx) error {
	status := c.Query("status")
	householdID := c.Query("household_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	items, meta, err := h.service.List(c.Context(), status, householdID, startDate, endDate, page, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to retrieve payments", err.Error())
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Payments retrieved successfully", items, meta)
}

func (h *PaymentHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			return response.Error(c, fiber.StatusNotFound, "Payment record not found", nil)
		}
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Payment retrieved successfully", result)
}

func (h *PaymentHandler) Confirm(c *fiber.Ctx) error {
	id := c.Params("id")

	// Parse multipart file with key "proof_file"
	file, err := c.FormFile("proof_file")
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Proof of payment file is required (field 'proof_file')", err.Error())
	}

	result, err := h.service.Confirm(c.Context(), c, id, file)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		if errors.Is(err, domain.ErrPaymentAlreadyPaid) {
			return response.Error(c, fiber.StatusConflict, err.Error(), nil)
		}
		if errors.Is(err, domain.ErrInvalidFileType) || errors.Is(err, domain.ErrFileSizeExceeded) {
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, "Failed to confirm payment", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Payment confirmed successfully", result)
}
