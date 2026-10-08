package service

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/rahadianap/community-waste-api/db/sqlc"
	"github.com/rahadianap/community-waste-api/internal/domain"
	"github.com/rahadianap/community-waste-api/internal/repository"
	"github.com/rahadianap/community-waste-api/internal/response"
)

type PickupService interface {
	Create(ctx context.Context, req domain.CreatePickupRequest) (*domain.PickupResponse, error)
	List(ctx context.Context, statusStr, householdIDStr, cursorStr string, limit int) ([]domain.PickupResponse, *response.Meta, error)
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

// Create enforces BR-01: household with pending payments cannot create a pickup.
func (s *pickupService) Create(ctx context.Context, req domain.CreatePickupRequest) (*domain.PickupResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, invalidInput(err)
	}

	householdUUID, err := domain.StringToUUID(req.HouseholdID)
	if err != nil {
		return nil, invalidInputf("invalid household_id: %v", err)
	}

	_, err = s.store.GetHouseholdByID(ctx, householdUUID)
	if err != nil {
		return nil, lookup(err, domain.ErrHouseholdNotFound, "gagal mengambil data household")
	}

	// BR-01
	pendingCount, err := s.store.CountPendingPaymentsByHouseholdID(ctx, householdUUID)
	if err != nil {
		return nil, wrap(err, "gagal menghitung data payment pending")
	}
	if pendingCount > 0 {
		return nil, domain.ErrPendingPaymentExists
	}

	pickup, err := s.store.CreatePickup(ctx, db.CreatePickupParams{
		HouseholdID: householdUUID,
		Type:        db.WasteType(req.Type),
		SafetyCheck: req.SafetyCheck,
	})
	if err != nil {
		return nil, wrap(err, "gagal membuat data pickup")
	}

	res := domain.ToPickupResponse(pickup)
	return &res, nil
}

func (s *pickupService) List(ctx context.Context, statusStr, householdIDStr, cursorStr string, limit int) ([]domain.PickupResponse, *response.Meta, error) {
	limit = normalizeLimit(limit)
	cursorCreatedAt, cursorID, err := decodeCursor(cursorStr)
	if err != nil {
		return nil, nil, err
	}

	validStatuses := map[string]bool{
		"pending":   true,
		"scheduled": true,
		"completed": true,
		"canceled":  true,
	}
	if statusStr != "" && !validStatuses[statusStr] {
		return nil, nil, invalidInputf("invalid status: must be pending, scheduled, completed, or canceled")
	}

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
		if err != nil {
			return nil, nil, invalidInputf("invalid household_id uuid: %v", err)
		}
		householdUUID = parsed
	}

	pickups, err := s.store.ListPickups(ctx, db.ListPickupsParams{
		Status:          nullStatus,
		HouseholdID:     householdUUID,
		CursorCreatedAt: cursorCreatedAt,
		CursorID:        cursorID,
		Limit:           int32(limit + 1),
	})
	if err != nil {
		return nil, nil, wrap(err, "gagal memuat data pickup")
	}

	hasMore := len(pickups) > limit
	if hasMore {
		pickups = pickups[:limit]
	}

	items := make([]domain.PickupResponse, len(pickups))
	for i, p := range pickups {
		items[i] = domain.ToPickupResponse(p)
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

// Schedule enforces BR-02 (only pending pickups) and BR-03 (electronic needs safety_check).
func (s *pickupService) Schedule(ctx context.Context, id string, req domain.SchedulePickupRequest) (*domain.PickupResponse, error) {
	pickupUUID, err := domain.StringToUUID(id)
	if err != nil {
		return nil, invalidInputf("invalid uuid: %v", err)
	}

	pickupDate, err := req.Validate()
	if err != nil {
		return nil, invalidInput(err)
	}

	pickup, err := s.store.GetPickupByID(ctx, pickupUUID)
	if err != nil {
		return nil, lookup(err, domain.ErrPickupNotFound, "gagal mengambil data pickup")
	}

	// BR-02
	if pickup.Status != db.PickupStatusPending {
		return nil, domain.ErrPickupNotPending
	}

	// BR-03
	if pickup.Type == db.WasteTypeElectronic && !pickup.SafetyCheck {
		return nil, domain.ErrSafetyCheckRequired
	}

	updatedPickup, err := s.store.UpdatePickupSchedule(ctx, db.UpdatePickupScheduleParams{
		ID: pickupUUID,
		PickupDate: pgtype.Timestamptz{
			Time:  pickupDate,
			Valid: true,
		},
	})
	if err != nil {
		return nil, wrap(err, "gagal menjadwalkan pickup")
	}

	res := domain.ToPickupResponse(updatedPickup)
	return &res, nil
}

// Complete enforces BR-04: completing a pickup auto-creates the payment (50k, electronic 100k) inside ExecTx.
func (s *pickupService) Complete(ctx context.Context, id string) (*domain.PickupResponse, error) {
	pickupUUID, err := domain.StringToUUID(id)
	if err != nil {
		return nil, invalidInputf("invalid uuid: %v", err)
	}

	pickup, err := s.store.GetPickupByID(ctx, pickupUUID)
	if err != nil {
		return nil, lookup(err, domain.ErrPickupNotFound, "gagal mengambil data pickup")
	}

	if pickup.Status == db.PickupStatusCompleted {
		return nil, domain.ErrPickupAlreadyCompleted
	}
	if pickup.Status == db.PickupStatusCanceled {
		return nil, domain.ErrPickupAlreadyCanceled
	}

	// BR-04 amount
	var amount float64
	if pickup.Type == db.WasteTypeElectronic {
		amount = 100000.00
	} else {
		amount = 50000.00
	}

	numericAmount, err := domain.Float64ToNumeric(amount)
	if err != nil {
		return nil, wrap(err, "gagal mengonversi data amount")
	}

	var updatedPickup db.WastePickup

	err = s.store.ExecTx(ctx, func(q db.Querier) error {
		var err error
		updatedPickup, err = q.UpdatePickupStatus(ctx, db.UpdatePickupStatusParams{
			ID:     pickupUUID,
			Status: db.PickupStatusCompleted,
		})
		if err != nil {
			return wrap(err, "gagal mengupdate status pickup")
		}

		_, err = q.CreatePayment(ctx, db.CreatePaymentParams{
			HouseholdID: pickup.HouseholdID,
			WasteID:     pickup.ID,
			Amount:      numericAmount,
			Status:      db.PaymentStatusPending,
		})
		if err != nil {
			return wrap(err, "gagal membuat data payment otomatis")
		}

		return nil
	})

	if err != nil {
		return nil, wrap(err, "gagal menyelesaikan transaksi pickup")
	}

	res := domain.ToPickupResponse(updatedPickup)
	return &res, nil
}

// Cancel updates pickup status to canceled
func (s *pickupService) Cancel(ctx context.Context, id string) (*domain.PickupResponse, error) {
	pickupUUID, err := domain.StringToUUID(id)
	if err != nil {
		return nil, invalidInputf("invalid uuid: %v", err)
	}

	pickup, err := s.store.GetPickupByID(ctx, pickupUUID)
	if err != nil {
		return nil, lookup(err, domain.ErrPickupNotFound, "gagal mengambil data pickup")
	}

	if pickup.Status == db.PickupStatusCompleted {
		return nil, domain.ErrCannotCancelCompleted
	}

	updatedPickup, err := s.store.UpdatePickupStatus(ctx, db.UpdatePickupStatusParams{
		ID:     pickupUUID,
		Status: db.PickupStatusCanceled,
	})
	if err != nil {
		return nil, wrap(err, "gagal membatalkan data pickup")
	}

	res := domain.ToPickupResponse(updatedPickup)
	return &res, nil
}
