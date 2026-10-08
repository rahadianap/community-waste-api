package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/rahadianap/community-waste-api/db/sqlc"
	"github.com/rahadianap/community-waste-api/internal/domain"
	"github.com/rahadianap/community-waste-api/internal/repository"
	"github.com/rahadianap/community-waste-api/pkg/response"
)

type PickupService interface {
	Create(ctx context.Context, req domain.CreatePickupRequest) (*domain.PickupResponse, error)
	GetByID(ctx context.Context, id string) (*domain.PickupResponse, error)
	List(ctx context.Context, statusStr, householdIDStr string, page, limit int) ([]domain.PickupResponse, *response.Meta, error)
	Schedule(ctx context.Context, id string, req domain.SchedulePickupRequest) (*domain.PickupResponse, error)
	Complete(ctx context.Context, id string) (*domain.PickupResponse, error)
	Cancel(ctx context.Context, id string) (*domain.PickupResponse, error)
}

type pickupService struct {
	store repository.Store
}

func NewPickupService(store repository.Store) PickupService {
	return &pickupService{store: store}
}

// Create enforces BR-01: A household cannot create a new pickup request if they have any payment with pending status.
func (s *pickupService) Create(ctx context.Context, req domain.CreatePickupRequest) (*domain.PickupResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	householdUUID, err := domain.StringToUUID(req.HouseholdID)
	if err != nil {
		return nil, fmt.Errorf("invalid household_id uuid: %w", err)
	}

	// 1. Verify household exists
	_, err = s.store.GetHouseholdByID(ctx, householdUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrHouseholdNotFound
		}
		return nil, fmt.Errorf("failed to get household: %w", err)
	}

	// 2. Business Rule BR-01: Check for pending payments
	pendingCount, err := s.store.CountPendingPaymentsByHouseholdID(ctx, householdUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to check pending payments: %w", err)
	}
	if pendingCount > 0 {
		return nil, domain.ErrPendingPaymentExists
	}

	// 3. Create pickup
	pickup, err := s.store.CreatePickup(ctx, db.CreatePickupParams{
		HouseholdID: householdUUID,
		Type:        db.WasteType(req.Type),
		SafetyCheck: req.SafetyCheck,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pickup: %w", err)
	}

	res := domain.ToPickupResponse(pickup)
	return &res, nil
}

func (s *pickupService) GetByID(ctx context.Context, id string) (*domain.PickupResponse, error) {
	pickupUUID, err := domain.StringToUUID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid: %w", err)
	}

	pickup, err := s.store.GetPickupByID(ctx, pickupUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPickupNotFound
		}
		return nil, fmt.Errorf("failed to get pickup: %w", err)
	}

	res := domain.ToPickupResponse(pickup)
	return &res, nil
}

func (s *pickupService) List(ctx context.Context, statusStr, householdIDStr string, page, limit int) ([]domain.PickupResponse, *response.Meta, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	var nullStatus db.NullPickupStatus
	if statusStr != "" {
		nullStatus = db.NullPickupStatus{
			PickupStatus: db.PickupStatus(statusStr),
			Valid:        true,
		}
	}

	var householdUUID pgtype.UUID
	if householdIDStr != "" {
		parsed, err := domain.StringToUUID(householdIDStr)
		if err == nil {
			householdUUID = parsed
		}
	}

	pickups, err := s.store.ListPickups(ctx, db.ListPickupsParams{
		Status:      nullStatus,
		HouseholdID: householdUUID,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list pickups: %w", err)
	}

	totalItems, err := s.store.CountPickups(ctx, db.CountPickupsParams{
		Status:      nullStatus,
		HouseholdID: householdUUID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count pickups: %w", err)
	}

	items := make([]domain.PickupResponse, len(pickups))
	for i, p := range pickups {
		items[i] = domain.ToPickupResponse(p)
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

// Schedule enforces:
// BR-02: A pickup can only be scheduled if its current status is pending.
// BR-03: Electronic type pickups cannot be scheduled unless safety_check is true.
func (s *pickupService) Schedule(ctx context.Context, id string, req domain.SchedulePickupRequest) (*domain.PickupResponse, error) {
	pickupUUID, err := domain.StringToUUID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid: %w", err)
	}

	pickupDate, err := req.Validate()
	if err != nil {
		return nil, err
	}

	// 1. Fetch current pickup
	pickup, err := s.store.GetPickupByID(ctx, pickupUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPickupNotFound
		}
		return nil, fmt.Errorf("failed to get pickup: %w", err)
	}

	// 2. Business Rule BR-02: Must be pending
	if pickup.Status != db.PickupStatusPending {
		return nil, domain.ErrPickupNotPending
	}

	// 3. Business Rule BR-03: Electronic waste requires safety check
	if pickup.Type == db.WasteTypeElectronic && !pickup.SafetyCheck {
		return nil, domain.ErrSafetyCheckRequired
	}

	// 4. Update schedule in DB
	updatedPickup, err := s.store.UpdatePickupSchedule(ctx, db.UpdatePickupScheduleParams{
		ID: pickupUUID,
		PickupDate: pgtype.Timestamptz{
			Time:  pickupDate,
			Valid: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to schedule pickup: %w", err)
	}

	res := domain.ToPickupResponse(updatedPickup)
	return &res, nil
}

// Complete enforces:
// BR-04: Once a pickup is completed, automatically generate a payment record:
// - organic, plastic, paper -> amount = 50,000
// - electronic -> amount = 100,000
// Executed atomically inside a database transaction (tx).
func (s *pickupService) Complete(ctx context.Context, id string) (*domain.PickupResponse, error) {
	pickupUUID, err := domain.StringToUUID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid: %w", err)
	}

	// 1. Fetch current pickup
	pickup, err := s.store.GetPickupByID(ctx, pickupUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPickupNotFound
		}
		return nil, fmt.Errorf("failed to get pickup: %w", err)
	}

	if pickup.Status == db.PickupStatusCompleted {
		return nil, domain.ErrPickupAlreadyCompleted
	}
	if pickup.Status == db.PickupStatusCanceled {
		return nil, domain.ErrPickupAlreadyCanceled
	}

	// 2. Calculate payment amount according to BR-04
	var amount float64
	if pickup.Type == db.WasteTypeElectronic {
		amount = 100000.00
	} else {
		amount = 50000.00
	}

	numericAmount, err := domain.Float64ToNumeric(amount)
	if err != nil {
		return nil, fmt.Errorf("failed to convert amount: %w", err)
	}

	var updatedPickup db.WastePickup

	// 3. Atomic Database Transaction: Update pickup -> completed AND create payment record
	err = s.store.ExecTx(ctx, func(q db.Querier) error {
		var err error
		updatedPickup, err = q.UpdatePickupStatus(ctx, db.UpdatePickupStatusParams{
			ID:     pickupUUID,
			Status: db.PickupStatusCompleted,
		})
		if err != nil {
			return fmt.Errorf("failed to update pickup status to completed: %w", err)
		}

		_, err = q.CreatePayment(ctx, db.CreatePaymentParams{
			HouseholdID: pickup.HouseholdID,
			WasteID:     pickup.ID,
			Amount:      numericAmount,
			Status:      db.PaymentStatusPending,
		})
		if err != nil {
			return fmt.Errorf("failed to automatically generate payment: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("complete pickup transaction failed: %w", err)
	}

	res := domain.ToPickupResponse(updatedPickup)
	return &res, nil
}

// Cancel updates pickup status to canceled
func (s *pickupService) Cancel(ctx context.Context, id string) (*domain.PickupResponse, error) {
	pickupUUID, err := domain.StringToUUID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid: %w", err)
	}

	pickup, err := s.store.GetPickupByID(ctx, pickupUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPickupNotFound
		}
		return nil, fmt.Errorf("failed to get pickup: %w", err)
	}

	if pickup.Status == db.PickupStatusCompleted {
		return nil, errors.New("cannot cancel a completed pickup")
	}

	updatedPickup, err := s.store.UpdatePickupStatus(ctx, db.UpdatePickupStatusParams{
		ID:     pickupUUID,
		Status: db.PickupStatusCanceled,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to cancel pickup: %w", err)
	}

	res := domain.ToPickupResponse(updatedPickup)
	return &res, nil
}
