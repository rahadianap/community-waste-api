package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"mime/multipart"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/rahadianap/community-waste-api/db/sqlc"
	"github.com/rahadianap/community-waste-api/internal/domain"
	"github.com/rahadianap/community-waste-api/internal/repository"
	"github.com/rahadianap/community-waste-api/pkg/response"
	"github.com/rahadianap/community-waste-api/pkg/uploader"
)

type PaymentService interface {
	Create(ctx context.Context, req domain.CreatePaymentRequest) (*domain.PaymentResponse, error)
	GetByID(ctx context.Context, id string) (*domain.PaymentResponse, error)
	List(ctx context.Context, statusStr, householdIDStr, startDateStr, endDateStr string, page, limit int) ([]domain.PaymentResponse, *response.Meta, error)
	Confirm(ctx context.Context, c *fiber.Ctx, id string, fileHeader *multipart.FileHeader) (*domain.PaymentResponse, error)
}

type paymentService struct {
	store    repository.Store
	uploader uploader.FileUploader
}

func NewPaymentService(store repository.Store, uploader uploader.FileUploader) PaymentService {
	return &paymentService{
		store:    store,
		uploader: uploader,
	}
}

func (s *paymentService) Create(ctx context.Context, req domain.CreatePaymentRequest) (*domain.PaymentResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	householdUUID, err := domain.StringToUUID(req.HouseholdID)
	if err != nil {
		return nil, fmt.Errorf("invalid household_id: %w", err)
	}

	wasteUUID, err := domain.StringToUUID(req.WasteID)
	if err != nil {
		return nil, fmt.Errorf("invalid waste_id: %w", err)
	}

	amountNumeric, err := domain.Float64ToNumeric(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	// Verify household and waste pickup exist
	if _, err := s.store.GetHouseholdByID(ctx, householdUUID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrHouseholdNotFound
		}
		return nil, err
	}
	if _, err := s.store.GetPickupByID(ctx, wasteUUID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPickupNotFound
		}
		return nil, err
	}

	payment, err := s.store.CreatePayment(ctx, db.CreatePaymentParams{
		HouseholdID: householdUUID,
		WasteID:     wasteUUID,
		Amount:      amountNumeric,
		Status:      db.PaymentStatusPending,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	res := domain.ToPaymentResponse(payment)
	return &res, nil
}

func (s *paymentService) GetByID(ctx context.Context, id string) (*domain.PaymentResponse, error) {
	paymentUUID, err := domain.StringToUUID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid: %w", err)
	}

	payment, err := s.store.GetPaymentByID(ctx, paymentUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	res := domain.ToPaymentResponse(payment)
	return &res, nil
}

// List implements GET /api/payments with filter status, household_id, and bonus date range filter
func (s *paymentService) List(ctx context.Context, statusStr, householdIDStr, startDateStr, endDateStr string, page, limit int) ([]domain.PaymentResponse, *response.Meta, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	var nullStatus db.NullPaymentStatus
	if statusStr != "" {
		nullStatus = db.NullPaymentStatus{
			PaymentStatus: db.PaymentStatus(statusStr),
			Valid:         true,
		}
	}

	var householdUUID pgtype.UUID
	if householdIDStr != "" {
		parsed, err := domain.StringToUUID(householdIDStr)
		if err == nil {
			householdUUID = parsed
		}
	}

	var startTimestamp, endTimestamp pgtype.Timestamptz
	if startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startTimestamp = pgtype.Timestamptz{Time: t, Valid: true}
		} else if t, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			startTimestamp = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	if endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			// End of the day
			t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			endTimestamp = pgtype.Timestamptz{Time: t, Valid: true}
		} else if t, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			endTimestamp = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	payments, err := s.store.ListPayments(ctx, db.ListPaymentsParams{
		Status:      nullStatus,
		HouseholdID: householdUUID,
		StartDate:   startTimestamp,
		EndDate:     endTimestamp,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list payments: %w", err)
	}

	totalItems, err := s.store.CountPayments(ctx, db.CountPaymentsParams{
		Status:      nullStatus,
		HouseholdID: householdUUID,
		StartDate:   startTimestamp,
		EndDate:     endTimestamp,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count payments: %w", err)
	}

	items := make([]domain.PaymentResponse, len(payments))
	for i, p := range payments {
		items[i] = domain.ToPaymentResponse(p)
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))
	meta := &response.Meta{
		Page:       page,
		Limit:      limit,
		TotalItems: totalItems,
		TotalPages: totalPages,
	}

	return items, meta, nil
}

// Confirm enforces BR-05: Payment confirmation requires uploading a proof of payment file.
// The file is stored locally and its URL/path is saved to the payment record.
func (s *paymentService) Confirm(ctx context.Context, c *fiber.Ctx, id string, fileHeader *multipart.FileHeader) (*domain.PaymentResponse, error) {
	paymentUUID, err := domain.StringToUUID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid: %w", err)
	}

	// 1. Fetch current payment
	payment, err := s.store.GetPaymentByID(ctx, paymentUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	if payment.Status == db.PaymentStatusPaid {
		return nil, domain.ErrPaymentAlreadyPaid
	}

	// 2. Upload proof file to local storage
	proofURL, err := s.uploader.UploadProof(c, fileHeader)
	if err != nil {
		return nil, err
	}

	// 3. Update payment record to paid
	updatedPayment, err := s.store.ConfirmPayment(ctx, db.ConfirmPaymentParams{
		ID: paymentUUID,
		ProofFileUrl: pgtype.Text{
			String: proofURL,
			Valid:  true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to confirm payment in database: %w", err)
	}

	res := domain.ToPaymentResponse(updatedPayment)
	return &res, nil
}
