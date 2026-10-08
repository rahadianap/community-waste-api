package unit

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rahadianap/community-waste-api/internal/domain"
	httpHandler "github.com/rahadianap/community-waste-api/internal/handler/http"
	"github.com/rahadianap/community-waste-api/internal/response"
	"github.com/rahadianap/community-waste-api/internal/service"
	"github.com/stretchr/testify/assert"
)

// --- Stub services (embed interface so only used methods need implementing) ---

type stubPickupService struct {
	service.PickupService
	createFn   func(context.Context, domain.CreatePickupRequest) (*domain.PickupResponse, error)
	scheduleFn func(context.Context, string, domain.SchedulePickupRequest) (*domain.PickupResponse, error)
	cancelFn   func(context.Context, string) (*domain.PickupResponse, error)
}

func (s *stubPickupService) Create(ctx context.Context, req domain.CreatePickupRequest) (*domain.PickupResponse, error) {
	return s.createFn(ctx, req)
}

func (s *stubPickupService) Schedule(ctx context.Context, id string, req domain.SchedulePickupRequest) (*domain.PickupResponse, error) {
	return s.scheduleFn(ctx, id, req)
}

func (s *stubPickupService) Cancel(ctx context.Context, id string) (*domain.PickupResponse, error) {
	return s.cancelFn(ctx, id)
}

type stubHouseholdService struct {
	service.HouseholdService
	createFn  func(context.Context, domain.CreateHouseholdRequest) (*domain.HouseholdResponse, error)
	getByIDFn func(context.Context, string) (*domain.HouseholdResponse, error)
	deleteFn  func(context.Context, string) error
}

func (s *stubHouseholdService) Create(ctx context.Context, req domain.CreateHouseholdRequest) (*domain.HouseholdResponse, error) {
	return s.createFn(ctx, req)
}

func (s *stubHouseholdService) GetByID(ctx context.Context, id string) (*domain.HouseholdResponse, error) {
	return s.getByIDFn(ctx, id)
}

func (s *stubHouseholdService) Delete(ctx context.Context, id string) error {
	return s.deleteFn(ctx, id)
}

type stubPaymentService struct {
	service.PaymentService
	createFn  func(context.Context, domain.CreatePaymentRequest) (*domain.PaymentResponse, error)
	listFn    func(context.Context, string, string, string, string, string, int) ([]domain.PaymentResponse, *response.Meta, error)
	confirmFn func(context.Context, *fiber.Ctx, string, *multipart.FileHeader) (*domain.PaymentResponse, error)
}

func (s *stubPaymentService) Create(ctx context.Context, req domain.CreatePaymentRequest) (*domain.PaymentResponse, error) {
	return s.createFn(ctx, req)
}

func (s *stubPaymentService) List(ctx context.Context, status, householdID, startDate, endDate, cursor string, limit int) ([]domain.PaymentResponse, *response.Meta, error) {
	return s.listFn(ctx, status, householdID, startDate, endDate, cursor, limit)
}

func (s *stubPaymentService) Confirm(ctx context.Context, c *fiber.Ctx, id string, fileHeader *multipart.FileHeader) (*domain.PaymentResponse, error) {
	return s.confirmFn(ctx, c, id, fileHeader)
}

// --- Test helpers ---

func newJSONRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func newConfirmRequest(target string) *http.Request {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("proof_file", "proof.jpg")
	_, _ = part.Write([]byte("dummy-content"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPut, target, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func newPickupApp(stub *stubPickupService) *fiber.App {
	app := fiber.New()
	h := httpHandler.NewPickupHandler(stub)
	app.Post("/api/pickups/", h.Create)
	app.Put("/api/pickups/:id/schedule", h.Schedule)
	app.Put("/api/pickups/:id/cancel", h.Cancel)
	return app
}

// --- Handler status-code mapping tests ---

func TestPickupHandler_ErrorStatusMapping(t *testing.T) {
	const createBody = `{"household_id":"3f1d3f2e-0000-4000-8000-000000000001","type":"organic","safety_check":false}`
	const anyID = "3f1d3f2e-0000-4000-8000-000000000002"

	t.Run("create 400", func(t *testing.T) {
		app := newPickupApp(&stubPickupService{
			createFn: func(context.Context, domain.CreatePickupRequest) (*domain.PickupResponse, error) {
				return nil, domain.ErrInvalidInput
			},
		})
		resp, err := app.Test(newJSONRequest(http.MethodPost, "/api/pickups/", createBody))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("create 422 ada tagihan pending", func(t *testing.T) {
		app := newPickupApp(&stubPickupService{
			createFn: func(context.Context, domain.CreatePickupRequest) (*domain.PickupResponse, error) {
				return nil, domain.ErrPendingPaymentExists
			},
		})
		resp, err := app.Test(newJSONRequest(http.MethodPost, "/api/pickups/", createBody))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
	})

	t.Run("create 404 household ga ada", func(t *testing.T) {
		app := newPickupApp(&stubPickupService{
			createFn: func(context.Context, domain.CreatePickupRequest) (*domain.PickupResponse, error) {
				return nil, domain.ErrHouseholdNotFound
			},
		})
		resp, err := app.Test(newJSONRequest(http.MethodPost, "/api/pickups/", createBody))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("create 500", func(t *testing.T) {
		app := newPickupApp(&stubPickupService{
			createFn: func(context.Context, domain.CreatePickupRequest) (*domain.PickupResponse, error) {
				return nil, assert.AnError
			},
		})
		resp, err := app.Test(newJSONRequest(http.MethodPost, "/api/pickups/", createBody))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("schedule 422 safety check", func(t *testing.T) {
		app := newPickupApp(&stubPickupService{
			scheduleFn: func(context.Context, string, domain.SchedulePickupRequest) (*domain.PickupResponse, error) {
				return nil, domain.ErrSafetyCheckRequired
			},
		})
		resp, err := app.Test(newJSONRequest(http.MethodPut, "/api/pickups/"+anyID+"/schedule", `{"pickup_date":"2026-10-15T10:00:00Z"}`))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
	})

	t.Run("schedule 400", func(t *testing.T) {
		app := newPickupApp(&stubPickupService{
			scheduleFn: func(context.Context, string, domain.SchedulePickupRequest) (*domain.PickupResponse, error) {
				return nil, domain.ErrInvalidInput
			},
		})
		resp, err := app.Test(newJSONRequest(http.MethodPut, "/api/pickups/"+anyID+"/schedule", `{"pickup_date":"besok"}`))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("cancel 409 sudah completed", func(t *testing.T) {
		app := newPickupApp(&stubPickupService{
			cancelFn: func(context.Context, string) (*domain.PickupResponse, error) {
				return nil, domain.ErrCannotCancelCompleted
			},
		})
		resp, err := app.Test(newJSONRequest(http.MethodPut, "/api/pickups/"+anyID+"/cancel", ""))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
	})

	t.Run("cancel 500", func(t *testing.T) {
		app := newPickupApp(&stubPickupService{
			cancelFn: func(context.Context, string) (*domain.PickupResponse, error) {
				return nil, assert.AnError
			},
		})
		resp, err := app.Test(newJSONRequest(http.MethodPut, "/api/pickups/"+anyID+"/cancel", ""))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

func TestHouseholdHandler_ErrorStatusMapping(t *testing.T) {
	const anyID = "3f1d3f2e-0000-4000-8000-000000000003"

	newApp := func(stub *stubHouseholdService) *fiber.App {
		app := fiber.New()
		h := httpHandler.NewHouseholdHandler(stub)
		app.Post("/api/households/", h.Create)
		app.Get("/api/households/:id", h.GetByID)
		app.Delete("/api/households/:id", h.Delete)
		return app
	}

	t.Run("create 400", func(t *testing.T) {
		app := newApp(&stubHouseholdService{
			createFn: func(context.Context, domain.CreateHouseholdRequest) (*domain.HouseholdResponse, error) {
				return nil, domain.ErrInvalidInput
			},
		})
		resp, err := app.Test(newJSONRequest(http.MethodPost, "/api/households/", `{"owner_name":"","address":""}`))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("create 500", func(t *testing.T) {
		app := newApp(&stubHouseholdService{
			createFn: func(context.Context, domain.CreateHouseholdRequest) (*domain.HouseholdResponse, error) {
				return nil, assert.AnError
			},
		})
		resp, err := app.Test(newJSONRequest(http.MethodPost, "/api/households/", `{"owner_name":"Budi","address":"Jl. Mawar"}`))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("get by id 404 household ga ada", func(t *testing.T) {
		app := newApp(&stubHouseholdService{
			getByIDFn: func(context.Context, string) (*domain.HouseholdResponse, error) {
				return nil, domain.ErrHouseholdNotFound
			},
		})
		resp, err := app.Test(newJSONRequest(http.MethodGet, "/api/households/"+anyID, ""))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("delete 400", func(t *testing.T) {
		app := newApp(&stubHouseholdService{
			deleteFn: func(context.Context, string) error {
				return domain.ErrInvalidInput
			},
		})
		resp, err := app.Test(newJSONRequest(http.MethodDelete, "/api/households/"+anyID, ""))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("delete 500", func(t *testing.T) {
		app := newApp(&stubHouseholdService{
			deleteFn: func(context.Context, string) error {
				return assert.AnError
			},
		})
		resp, err := app.Test(newJSONRequest(http.MethodDelete, "/api/households/"+anyID, ""))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}

func TestPaymentHandler_ErrorStatusMapping(t *testing.T) {
	const anyID = "3f1d3f2e-0000-4000-8000-000000000004"
	const createBody = `{"household_id":"3f1d3f2e-0000-4000-8000-000000000005","waste_id":"3f1d3f2e-0000-4000-8000-000000000006","amount":50000}`

	newApp := func(stub *stubPaymentService) *fiber.App {
		app := fiber.New()
		h := httpHandler.NewPaymentHandler(stub)
		app.Post("/api/payments/", h.Create)
		app.Get("/api/payments/", h.List)
		app.Put("/api/payments/:id/confirm", h.Confirm)
		return app
	}

	t.Run("create 404 household ga ada", func(t *testing.T) {
		app := newApp(&stubPaymentService{
			createFn: func(context.Context, domain.CreatePaymentRequest) (*domain.PaymentResponse, error) {
				return nil, domain.ErrHouseholdNotFound
			},
		})
		resp, err := app.Test(newJSONRequest(http.MethodPost, "/api/payments/", createBody))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("create 400", func(t *testing.T) {
		app := newApp(&stubPaymentService{
			createFn: func(context.Context, domain.CreatePaymentRequest) (*domain.PaymentResponse, error) {
				return nil, domain.ErrInvalidInput
			},
		})
		resp, err := app.Test(newJSONRequest(http.MethodPost, "/api/payments/", createBody))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("create 500", func(t *testing.T) {
		app := newApp(&stubPaymentService{
			createFn: func(context.Context, domain.CreatePaymentRequest) (*domain.PaymentResponse, error) {
				return nil, assert.AnError
			},
		})
		resp, err := app.Test(newJSONRequest(http.MethodPost, "/api/payments/", createBody))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("list 400", func(t *testing.T) {
		app := newApp(&stubPaymentService{
			listFn: func(context.Context, string, string, string, string, string, int) ([]domain.PaymentResponse, *response.Meta, error) {
				return nil, nil, domain.ErrInvalidInput
			},
		})
		resp, err := app.Test(newJSONRequest(http.MethodGet, "/api/payments/?status=bukan-status", ""))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("confirm 404", func(t *testing.T) {
		app := newApp(&stubPaymentService{
			confirmFn: func(context.Context, *fiber.Ctx, string, *multipart.FileHeader) (*domain.PaymentResponse, error) {
				return nil, domain.ErrPaymentNotFound
			},
		})
		resp, err := app.Test(newConfirmRequest("/api/payments/" + anyID + "/confirm"))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})

	t.Run("confirm 409 sudah paid", func(t *testing.T) {
		app := newApp(&stubPaymentService{
			confirmFn: func(context.Context, *fiber.Ctx, string, *multipart.FileHeader) (*domain.PaymentResponse, error) {
				return nil, domain.ErrPaymentAlreadyPaid
			},
		})
		resp, err := app.Test(newConfirmRequest("/api/payments/" + anyID + "/confirm"))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
	})

	t.Run("confirm 400 tanpa file", func(t *testing.T) {
		app := newApp(&stubPaymentService{
			confirmFn: func(context.Context, *fiber.Ctx, string, *multipart.FileHeader) (*domain.PaymentResponse, error) {
				return nil, domain.ErrProofFileRequired
			},
		})
		resp, err := app.Test(newConfirmRequest("/api/payments/" + anyID + "/confirm"))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("confirm 500", func(t *testing.T) {
		app := newApp(&stubPaymentService{
			confirmFn: func(context.Context, *fiber.Ctx, string, *multipart.FileHeader) (*domain.PaymentResponse, error) {
				return nil, assert.AnError
			},
		})
		resp, err := app.Test(newConfirmRequest("/api/payments/" + anyID + "/confirm"))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
	})
}
