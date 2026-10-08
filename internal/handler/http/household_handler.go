package http

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/rahadianap/community-waste-api/internal/domain"
	"github.com/rahadianap/community-waste-api/internal/response"
	"github.com/rahadianap/community-waste-api/internal/service"
)

type HouseholdHandler struct {
	service service.HouseholdService
}

func NewHouseholdHandler(service service.HouseholdService) *HouseholdHandler {
	return &HouseholdHandler{service: service}
}

func (h *HouseholdHandler) Create(c *fiber.Ctx) error {
	var req domain.CreateHouseholdRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid JSON request body", err.Error())
	}

	result, err := h.service.Create(c.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "Data berhasil ditambahkan", result)
}

func (h *HouseholdHandler) List(c *fiber.Ctx) error {
	cursor := c.Query("cursor")
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	items, meta, err := h.service.List(c.Context(), cursor, limit)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, "Data gagal diambil", err.Error())
	}

	return response.SuccessWithMeta(c, fiber.StatusOK, "Data berhasil diambil", items, meta)
}

func (h *HouseholdHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	result, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrHouseholdNotFound) {
			return response.Error(c, fiber.StatusNotFound, "Data tidak ditemukan", nil)
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Data berhasil diambil", result)
}

func (h *HouseholdHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	err := h.service.Delete(c.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrHouseholdNotFound) {
			return response.Error(c, fiber.StatusNotFound, "Data tidak ditemukan", nil)
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Data berhasil dihapus", nil)
}
