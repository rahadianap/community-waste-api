package http

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rahadianap/community-waste-api/internal/domain"
	"github.com/rahadianap/community-waste-api/internal/service"
	"github.com/rahadianap/community-waste-api/pkg/response"
)

type PickupHandler struct {
	service service.PickupService
}

func NewPickupHandler(service service.PickupService) *PickupHandler {
	return &PickupHandler{service: service}
}

func (h *PickupHandler) Create(c *fiber.Ctx) error {
	var req domain.CreatePickupRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid JSON request body", err.Error())
	}

	result, err := h.service.Create(c.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrPendingPaymentExists) {
			return response.Error(c, fiber.StatusUnprocessableEntity, err.Error(), nil)
		}
		if errors.Is(err, domain.ErrHouseholdNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "Pickup request created successfully", result)
}

func (h *PickupHandler) List(c *fiber.Ctx) error {
	status := c.Query("status")
	householdID := c.Query("household_id")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	items, meta, err := h.service.List(c.Context(), status, householdID, page, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Failed to retrieve pickups", err.Error())
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Pickups retrieved successfully", items, meta)
}

func (h *PickupHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrPickupNotFound) {
			return response.Error(c, fiber.StatusNotFound, "Waste pickup not found", nil)
		}
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Pickup retrieved successfully", result)
}

func (h *PickupHandler) Schedule(c *fiber.Ctx) error {
	id := c.Params("id")
	var req domain.SchedulePickupRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid JSON request body", err.Error())
	}

	result, err := h.service.Schedule(c.Context(), id, req)
	if err != nil {
		if errors.Is(err, domain.ErrPickupNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		if errors.Is(err, domain.ErrPickupNotPending) || errors.Is(err, domain.ErrSafetyCheckRequired) {
			return response.Error(c, fiber.StatusUnprocessableEntity, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Pickup scheduled successfully", result)
}

func (h *PickupHandler) Complete(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := h.service.Complete(c.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrPickupNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		if errors.Is(err, domain.ErrPickupAlreadyCompleted) || errors.Is(err, domain.ErrPickupAlreadyCanceled) {
			return response.Error(c, fiber.StatusConflict, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Pickup marked as completed and payment created", result)
}

func (h *PickupHandler) Cancel(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := h.service.Cancel(c.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrPickupNotFound) {
			return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Pickup canceled successfully", result)
}
