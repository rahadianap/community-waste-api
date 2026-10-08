package service

import (
	"context"
	"mime/multipart"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/rahadianap/community-waste-api/db/sqlc"
	"github.com/rahadianap/community-waste-api/internal/domain"
	"github.com/rahadianap/community-waste-api/internal/repository"
	"github.com/rahadianap/community-waste-api/internal/response"
	"github.com/rahadianap/community-waste-api/internal/uploader"
)

type PaymentService interface {
	Create(ctx context.Context, req domain.CreatePaymentRequest) (*domain.PaymentResponse, error)
	List(ctx context.Context, statusStr, householdIDStr, startDateStr, endDateStr, cursorStr string, limit int) ([]domain.PaymentResponse, *response.Meta, error)
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
		return nil, invalidInput(err)
	}

	householdUUID, err := domain.StringToUUID(req.HouseholdID)
	if err != nil {
		return nil, invalidInputf("invalid household_id: %v", err)
	}

	wasteUUID, err := domain.StringToUUID(req.WasteID)
	if err != nil {
		return nil, invalidInputf("invalid waste_id: %v", err)
	}

	amountNumeric, err := domain.Float64ToNumeric(req.Amount)
	if err != nil {
		return nil, invalidInputf("invalid amount: %v", err)
	}

	if _, err := s.store.GetHouseholdByID(ctx, householdUUID); err != nil {
		return nil, lookup(err, domain.ErrHouseholdNotFound, "gagal mengambil data household")
	}
	if _, err := s.store.GetPickupByID(ctx, wasteUUID); err != nil {
		return nil, lookup(err, domain.ErrPickupNotFound, "gagal mengambil data pickup")
	}

	payment, err := s.store.CreatePayment(ctx, db.CreatePaymentParams{
		HouseholdID: householdUUID,
		WasteID:     wasteUUID,
		Amount:      amountNumeric,
		Status:      db.PaymentStatusPending,
	})
	if err != nil {
		return nil, wrap(err, "gagal membuat data payment")
	}

	res := domain.ToPaymentResponse(payment)
	return &res, nil
}

// List implements GET /api/payments with filter status, household_id, and bonus date range filter
func (s *paymentService) List(ctx context.Context, statusStr, householdIDStr, startDateStr, endDateStr, cursorStr string, limit int) ([]domain.PaymentResponse, *response.Meta, error) {
	limit = normalizeLimit(limit)
	cursorCreatedAt, cursorID, err := decodeCursor(cursorStr)
	if err != nil {
		return nil, nil, err
	}

	validStatuses := map[string]bool{
		"pending": true,
		"paid":    true,
		"failed":  true,
	}
	if statusStr != "" && !validStatuses[statusStr] {
		return nil, nil, invalidInputf("invalid status: must be pending, paid, or failed")
	}

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
		if err != nil {
			return nil, nil, invalidInputf("invalid household_id uuid: %v", err)
		}
		householdUUID = parsed
	}

	var startTimestamp, endTimestamp pgtype.Timestamptz
	if startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startTimestamp = pgtype.Timestamptz{Time: t, Valid: true}
		} else if t, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			startTimestamp = pgtype.Timestamptz{Time: t, Valid: true}
		} else {
			return nil, nil, invalidInputf("invalid start_date format, expected YYYY-MM-DD or RFC3339")
		}
	}

	if endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			// End of the day
			t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			endTimestamp = pgtype.Timestamptz{Time: t, Valid: true}
		} else if t, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			endTimestamp = pgtype.Timestamptz{Time: t, Valid: true}
		} else {
			return nil, nil, invalidInputf("invalid end_date format, expected YYYY-MM-DD or RFC3339")
		}
	}

	payments, err := s.store.ListPayments(ctx, db.ListPaymentsParams{
		Status:          nullStatus,
		HouseholdID:     householdUUID,
		StartDate:       startTimestamp,
		EndDate:         endTimestamp,
		CursorCreatedAt: cursorCreatedAt,
		CursorID:        cursorID,
		Limit:           int32(limit + 1),
	})
	if err != nil {
		return nil, nil, wrap(err, "gagal memuat data payments")
	}

	hasMore := len(payments) > limit
	if hasMore {
		payments = payments[:limit]
	}

	items := make([]domain.PaymentResponse, len(payments))
	for i, p := range payments {
		items[i] = domain.ToPaymentResponse(p)
	}

	meta := &response.Meta{
		Limit:   limit,
		HasMore: hasMore,
	}
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		next := domain.EncodeCursor(last.CreatedAt, last.ID)
		meta.NextCursor = &next
	}

	return items, meta, nil
}

// Confirm enforces BR-05: payment confirmation requires a proof-of-payment upload.
func (s *paymentService) Confirm(ctx context.Context, c *fiber.Ctx, id string, fileHeader *multipart.FileHeader) (*domain.PaymentResponse, error) {
	paymentUUID, err := domain.StringToUUID(id)
	if err != nil {
		return nil, invalidInputf("invalid uuid: %v", err)
	}

	payment, err := s.store.GetPaymentByID(ctx, paymentUUID)
	if err != nil {
		return nil, lookup(err, domain.ErrPaymentNotFound, "gagal mengambil data payment")
	}

	if payment.Status == db.PaymentStatusPaid {
		return nil, domain.ErrPaymentAlreadyPaid
	}

	proofURL, err := s.uploader.UploadProof(c, fileHeader)
	if err != nil {
		return nil, err
	}

	updatedPayment, err := s.store.ConfirmPayment(ctx, db.ConfirmPaymentParams{
		ID: paymentUUID,
		ProofFileUrl: pgtype.Text{
			String: proofURL,
			Valid:  true,
		},
	})
	if err != nil {
		return nil, wrap(err, "gagal mengupdate data payment")
	}

	res := domain.ToPaymentResponse(updatedPayment)
	return &res, nil
}
