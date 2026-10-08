package unit

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/rahadianap/community-waste-api/db/sqlc"
	"github.com/stretchr/testify/mock"
)

type MockStore struct {
	mock.Mock
}

func (m *MockStore) ExecTx(ctx context.Context, fn func(q db.Querier) error) error {
	args := m.Called(ctx, fn)
	if args.Get(0) != nil {
		return args.Error(0)
	}
	return fn(m)
}

func (m *MockStore) CreateHousehold(ctx context.Context, arg db.CreateHouseholdParams) (db.Household, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.Household), args.Error(1)
}

func (m *MockStore) DeleteHousehold(ctx context.Context, id pgtype.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockStore) GetHouseholdByID(ctx context.Context, id pgtype.UUID) (db.Household, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.Household), args.Error(1)
}

func (m *MockStore) ListHouseholds(ctx context.Context, arg db.ListHouseholdsParams) ([]db.Household, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]db.Household), args.Error(1)
}

func (m *MockStore) ConfirmPayment(ctx context.Context, arg db.ConfirmPaymentParams) (db.Payment, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.Payment), args.Error(1)
}

func (m *MockStore) CountPendingPaymentsByHouseholdID(ctx context.Context, householdID pgtype.UUID) (int64, error) {
	args := m.Called(ctx, householdID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockStore) CreatePayment(ctx context.Context, arg db.CreatePaymentParams) (db.Payment, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.Payment), args.Error(1)
}

func (m *MockStore) GetPaymentByID(ctx context.Context, id pgtype.UUID) (db.Payment, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.Payment), args.Error(1)
}

func (m *MockStore) ListPayments(ctx context.Context, arg db.ListPaymentsParams) ([]db.Payment, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]db.Payment), args.Error(1)
}

func (m *MockStore) CreatePickup(ctx context.Context, arg db.CreatePickupParams) (db.WastePickup, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.WastePickup), args.Error(1)
}

func (m *MockStore) GetPickupByID(ctx context.Context, id pgtype.UUID) (db.WastePickup, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.WastePickup), args.Error(1)
}

func (m *MockStore) ListPickups(ctx context.Context, arg db.ListPickupsParams) ([]db.WastePickup, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).([]db.WastePickup), args.Error(1)
}

func (m *MockStore) UpdatePickupSchedule(ctx context.Context, arg db.UpdatePickupScheduleParams) (db.WastePickup, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.WastePickup), args.Error(1)
}

func (m *MockStore) UpdatePickupStatus(ctx context.Context, arg db.UpdatePickupStatusParams) (db.WastePickup, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(db.WastePickup), args.Error(1)
}

func (m *MockStore) GetPaymentSummaryReport(ctx context.Context) ([]db.GetPaymentSummaryReportRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]db.GetPaymentSummaryReportRow), args.Error(1)
}

func (m *MockStore) GetTotalRevenue(ctx context.Context) (pgtype.Numeric, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgtype.Numeric), args.Error(1)
}

func (m *MockStore) GetWasteSummaryReport(ctx context.Context) ([]db.GetWasteSummaryReportRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]db.GetWasteSummaryReportRow), args.Error(1)
}
