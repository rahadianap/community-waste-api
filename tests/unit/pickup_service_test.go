package unit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/rahadianap/community-waste-api/db/sqlc"
	"github.com/rahadianap/community-waste-api/internal/domain"
	"github.com/rahadianap/community-waste-api/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestUUID() (string, pgtype.UUID) {
	u := uuid.New()
	pgU := pgtype.UUID{Bytes: u, Valid: true}
	return u.String(), pgU
}

// BR-01: Household cannot create new pickup if they have pending payment
func TestPickupService_BR01_PendingPaymentCheck(t *testing.T) {
	ctx := context.Background()

	t.Run("BR-01 Fail: Should reject pickup creation when household has pending payments", func(t *testing.T) {
		mockStore := new(MockStore)
		pickupSvc := service.NewPickupService(mockStore)

		hIDStr, hID := newTestUUID()

		mockStore.On("GetHouseholdByID", ctx, hID).Return(db.Household{
			ID:        hID,
			OwnerName: "Budi",
			Address:   "Jl. Mawar",
		}, nil)

		// Household has 1 pending payment!
		mockStore.On("CountPendingPaymentsByHouseholdID", ctx, hID).Return(int64(1), nil)

		req := domain.CreatePickupRequest{
			HouseholdID: hIDStr,
			Type:        "organic",
			SafetyCheck: false,
		}

		res, err := pickupSvc.Create(ctx, req)
		assert.ErrorIs(t, err, domain.ErrPendingPaymentExists)
		assert.Nil(t, res)
		mockStore.AssertExpectations(t)
	})

	t.Run("BR-01 Success: Should allow pickup creation when household has 0 pending payments", func(t *testing.T) {
		mockStore := new(MockStore)
		pickupSvc := service.NewPickupService(mockStore)

		hIDStr, hID := newTestUUID()
		pIDStr, pID := newTestUUID()
		_ = pIDStr

		mockStore.On("GetHouseholdByID", ctx, hID).Return(db.Household{
			ID:        hID,
			OwnerName: "Budi",
			Address:   "Jl. Mawar",
		}, nil)

		// 0 pending payments
		mockStore.On("CountPendingPaymentsByHouseholdID", ctx, hID).Return(int64(0), nil)

		mockStore.On("CreatePickup", ctx, db.CreatePickupParams{
			HouseholdID: hID,
			Type:        db.WasteTypeOrganic,
			SafetyCheck: false,
		}).Return(db.WastePickup{
			ID:          pID,
			HouseholdID: hID,
			Type:        db.WasteTypeOrganic,
			Status:      db.PickupStatusPending,
		}, nil)

		req := domain.CreatePickupRequest{
			HouseholdID: hIDStr,
			Type:        "organic",
			SafetyCheck: false,
		}

		res, err := pickupSvc.Create(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "organic", res.Type)
		assert.Equal(t, "pending", res.Status)
		mockStore.AssertExpectations(t)
	})
}

// BR-02: A pickup can only be scheduled if current status is pending
func TestPickupService_BR02_OnlyPendingCanBeScheduled(t *testing.T) {
	ctx := context.Background()

	t.Run("BR-02 Fail: Reject scheduling when status is already scheduled", func(t *testing.T) {
		mockStore := new(MockStore)
		pickupSvc := service.NewPickupService(mockStore)

		_, hID := newTestUUID()
		pIDStr, pID := newTestUUID()

		mockStore.On("GetPickupByID", ctx, pID).Return(db.WastePickup{
			ID:          pID,
			HouseholdID: hID,
			Type:        db.WasteTypeOrganic,
			Status:      db.PickupStatusScheduled, // Already scheduled!
		}, nil)

		req := domain.SchedulePickupRequest{
			PickupDate: "2026-10-15T10:00:00Z",
		}

		res, err := pickupSvc.Schedule(ctx, pIDStr, req)
		assert.ErrorIs(t, err, domain.ErrPickupNotPending)
		assert.Nil(t, res)
		mockStore.AssertExpectations(t)
	})
}

// BR-03: Electronic type pickups cannot be scheduled unless safety_check is true
func TestPickupService_BR03_ElectronicSafetyCheck(t *testing.T) {
	ctx := context.Background()

	t.Run("BR-03 Fail: Reject scheduling electronic pickup when safety_check is false", func(t *testing.T) {
		mockStore := new(MockStore)
		pickupSvc := service.NewPickupService(mockStore)

		_, hID := newTestUUID()
		pIDStr, pID := newTestUUID()

		mockStore.On("GetPickupByID", ctx, pID).Return(db.WastePickup{
			ID:          pID,
			HouseholdID: hID,
			Type:        db.WasteTypeElectronic,
			Status:      db.PickupStatusPending,
			SafetyCheck: false, // safety_check is false!
		}, nil)

		req := domain.SchedulePickupRequest{
			PickupDate: "2026-10-15T10:00:00Z",
		}

		res, err := pickupSvc.Schedule(ctx, pIDStr, req)
		assert.ErrorIs(t, err, domain.ErrSafetyCheckRequired)
		assert.Nil(t, res)
		mockStore.AssertExpectations(t)
	})

	t.Run("BR-03 Success: Allow scheduling electronic pickup when safety_check is true", func(t *testing.T) {
		mockStore := new(MockStore)
		pickupSvc := service.NewPickupService(mockStore)

		_, hID := newTestUUID()
		pIDStr, pID := newTestUUID()
		schedTime := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)

		mockStore.On("GetPickupByID", ctx, pID).Return(db.WastePickup{
			ID:          pID,
			HouseholdID: hID,
			Type:        db.WasteTypeElectronic,
			Status:      db.PickupStatusPending,
			SafetyCheck: true, // safety_check is true!
		}, nil)

		mockStore.On("UpdatePickupSchedule", ctx, db.UpdatePickupScheduleParams{
			ID: pID,
			PickupDate: pgtype.Timestamptz{
				Time:  schedTime,
				Valid: true,
			},
		}).Return(db.WastePickup{
			ID:          pID,
			HouseholdID: hID,
			Type:        db.WasteTypeElectronic,
			Status:      db.PickupStatusScheduled,
			SafetyCheck: true,
			PickupDate:  pgtype.Timestamptz{Time: schedTime, Valid: true},
		}, nil)

		req := domain.SchedulePickupRequest{
			PickupDate: "2026-10-15T10:00:00Z",
		}

		res, err := pickupSvc.Schedule(ctx, pIDStr, req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "scheduled", res.Status)
		mockStore.AssertExpectations(t)
	})
}

// BR-04: Once pickup is completed, automatically create payment (organic=50k, electronic=100k)
func TestPickupService_BR04_AutoPaymentGenerationOnComplete(t *testing.T) {
	ctx := context.Background()

	t.Run("BR-04: Organic pickup generates payment with 50,000 amount", func(t *testing.T) {
		mockStore := new(MockStore)
		pickupSvc := service.NewPickupService(mockStore)

		_, hID := newTestUUID()
		pIDStr, pID := newTestUUID()
		payIDStr, payID := newTestUUID()
		_ = payIDStr

		amount50k, _ := domain.Float64ToNumeric(50000.00)

		mockStore.On("GetPickupByID", ctx, pID).Return(db.WastePickup{
			ID:          pID,
			HouseholdID: hID,
			Type:        db.WasteTypeOrganic,
			Status:      db.PickupStatusScheduled,
		}, nil)

		mockStore.On("ExecTx", ctx, mock.AnythingOfType("func(db.Querier) error")).Return(nil)
		mockStore.On("UpdatePickupStatus", ctx, db.UpdatePickupStatusParams{
			ID:     pID,
			Status: db.PickupStatusCompleted,
		}).Return(db.WastePickup{
			ID:          pID,
			HouseholdID: hID,
			Type:        db.WasteTypeOrganic,
			Status:      db.PickupStatusCompleted,
		}, nil)

		mockStore.On("CreatePayment", ctx, db.CreatePaymentParams{
			HouseholdID: hID,
			WasteID:     pID,
			Amount:      amount50k,
			Status:      db.PaymentStatusPending,
		}).Return(db.Payment{
			ID:          payID,
			HouseholdID: hID,
			WasteID:     pID,
			Amount:      amount50k,
			Status:      db.PaymentStatusPending,
		}, nil)

		res, err := pickupSvc.Complete(ctx, pIDStr)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "completed", res.Status)
		mockStore.AssertExpectations(t)
	})

	t.Run("BR-04: Electronic pickup generates payment with 100,000 amount", func(t *testing.T) {
		mockStore := new(MockStore)
		pickupSvc := service.NewPickupService(mockStore)

		_, hID := newTestUUID()
		pIDStr, pID := newTestUUID()
		_, payID := newTestUUID()

		amount100k, _ := domain.Float64ToNumeric(100000.00)

		mockStore.On("GetPickupByID", ctx, pID).Return(db.WastePickup{
			ID:          pID,
			HouseholdID: hID,
			Type:        db.WasteTypeElectronic,
			Status:      db.PickupStatusScheduled,
		}, nil)

		mockStore.On("ExecTx", ctx, mock.AnythingOfType("func(db.Querier) error")).Return(nil)
		mockStore.On("UpdatePickupStatus", ctx, db.UpdatePickupStatusParams{
			ID:     pID,
			Status: db.PickupStatusCompleted,
		}).Return(db.WastePickup{
			ID:          pID,
			HouseholdID: hID,
			Type:        db.WasteTypeElectronic,
			Status:      db.PickupStatusCompleted,
		}, nil)

		mockStore.On("CreatePayment", ctx, db.CreatePaymentParams{
			HouseholdID: hID,
			WasteID:     pID,
			Amount:      amount100k,
			Status:      db.PaymentStatusPending,
		}).Return(db.Payment{
			ID:          payID,
			HouseholdID: hID,
			WasteID:     pID,
			Amount:      amount100k,
			Status:      db.PaymentStatusPending,
		}, nil)

		res, err := pickupSvc.Complete(ctx, pIDStr)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "completed", res.Status)
		mockStore.AssertExpectations(t)
	})
}

// Client input validation and cancel rules map to the correct sentinel errors
func TestPickupService_InvalidInputAndCancelRules(t *testing.T) {
	ctx := context.Background()

	t.Run("create: household_id bukan uuid", func(t *testing.T) {
		mockStore := new(MockStore)
		pickupSvc := service.NewPickupService(mockStore)

		res, err := pickupSvc.Create(ctx, domain.CreatePickupRequest{
			HouseholdID: "bukan-uuid",
			Type:        "organic",
		})
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Nil(t, res)
		mockStore.AssertExpectations(t)
	})

	t.Run("create: tipe sampah tidak valid", func(t *testing.T) {
		mockStore := new(MockStore)
		pickupSvc := service.NewPickupService(mockStore)

		hIDStr, _ := newTestUUID()
		res, err := pickupSvc.Create(ctx, domain.CreatePickupRequest{
			HouseholdID: hIDStr,
			Type:        "chemical",
		})
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Nil(t, res)
		mockStore.AssertExpectations(t)
	})

	t.Run("schedule: uuid tidak valid", func(t *testing.T) {
		mockStore := new(MockStore)
		pickupSvc := service.NewPickupService(mockStore)

		res, err := pickupSvc.Schedule(ctx, "bukan-uuid", domain.SchedulePickupRequest{
			PickupDate: "2026-10-15T10:00:00Z",
		})
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Nil(t, res)
		mockStore.AssertExpectations(t)
	})

	t.Run("schedule: pickup_date tidak valid", func(t *testing.T) {
		mockStore := new(MockStore)
		pickupSvc := service.NewPickupService(mockStore)

		pIDStr, _ := newTestUUID()
		res, err := pickupSvc.Schedule(ctx, pIDStr, domain.SchedulePickupRequest{
			PickupDate: "besok",
		})
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Nil(t, res)
		mockStore.AssertExpectations(t)
	})

	t.Run("list: status tidak valid", func(t *testing.T) {
		mockStore := new(MockStore)
		pickupSvc := service.NewPickupService(mockStore)

		res, meta, err := pickupSvc.List(ctx, "bukan-status", "", "", 10)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Nil(t, res)
		assert.Nil(t, meta)
		mockStore.AssertExpectations(t)
	})

	t.Run("list: household_id bukan uuid", func(t *testing.T) {
		mockStore := new(MockStore)
		pickupSvc := service.NewPickupService(mockStore)

		res, meta, err := pickupSvc.List(ctx, "", "bukan-uuid", "", 10)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Nil(t, res)
		assert.Nil(t, meta)
		mockStore.AssertExpectations(t)
	})

	t.Run("list: cursor rusak", func(t *testing.T) {
		mockStore := new(MockStore)
		pickupSvc := service.NewPickupService(mockStore)

		res, meta, err := pickupSvc.List(ctx, "", "", "cursor-rusak!", 10)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Nil(t, res)
		assert.Nil(t, meta)
		mockStore.AssertExpectations(t)
	})

	t.Run("cancel: pickup sudah completed", func(t *testing.T) {
		mockStore := new(MockStore)
		pickupSvc := service.NewPickupService(mockStore)

		_, hID := newTestUUID()
		pIDStr, pID := newTestUUID()

		mockStore.On("GetPickupByID", ctx, pID).Return(db.WastePickup{
			ID:          pID,
			HouseholdID: hID,
			Type:        db.WasteTypeOrganic,
			Status:      db.PickupStatusCompleted,
		}, nil)

		res, err := pickupSvc.Cancel(ctx, pIDStr)
		assert.ErrorIs(t, err, domain.ErrCannotCancelCompleted)
		assert.Nil(t, res)
		mockStore.AssertExpectations(t)
	})
}
