package unit

import (
	"context"
	"mime/multipart"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/rahadianap/community-waste-api/db/sqlc"
	"github.com/rahadianap/community-waste-api/internal/domain"
	"github.com/rahadianap/community-waste-api/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUploader struct {
	mock.Mock
}

func (m *MockUploader) UploadProof(c *fiber.Ctx, file *multipart.FileHeader) (string, error) {
	args := m.Called(c, file)
	return args.String(0), args.Error(1)
}

func TestPaymentService_BR05_PaymentConfirmation(t *testing.T) {
	ctx := context.Background()

	t.Run("BR-05 Fail: Reject confirm payment if already paid", func(t *testing.T) {
		mockStore := new(MockStore)
		mockUploader := new(MockUploader)
		paymentSvc := service.NewPaymentService(mockStore, mockUploader)

		payIDStr, payID := newTestUUID()
		_, hID := newTestUUID()
		_, wID := newTestUUID()

		mockStore.On("GetPaymentByID", ctx, payID).Return(db.Payment{
			ID:          payID,
			HouseholdID: hID,
			WasteID:     wID,
			Status:      db.PaymentStatusPaid, // Already paid!
		}, nil)

		fileHeader := &multipart.FileHeader{Filename: "proof.jpg"}
		res, err := paymentSvc.Confirm(ctx, nil, payIDStr, fileHeader)

		assert.ErrorIs(t, err, domain.ErrPaymentAlreadyPaid)
		assert.Nil(t, res)
		mockStore.AssertExpectations(t)
	})

	t.Run("BR-05 Success: Confirm payment with valid proof file", func(t *testing.T) {
		mockStore := new(MockStore)
		mockUploader := new(MockUploader)
		paymentSvc := service.NewPaymentService(mockStore, mockUploader)

		payIDStr, payID := newTestUUID()
		_, hID := newTestUUID()
		_, wID := newTestUUID()
		amount, _ := domain.Float64ToNumeric(50000.00)

		mockStore.On("GetPaymentByID", ctx, payID).Return(db.Payment{
			ID:          payID,
			HouseholdID: hID,
			WasteID:     wID,
			Amount:      amount,
			Status:      db.PaymentStatusPending,
		}, nil)

		fileHeader := &multipart.FileHeader{Filename: "proof.jpg", Size: 1024}
		uploadedURL := "/uploads/payments/proof_123.jpg"

		mockUploader.On("UploadProof", (*fiber.Ctx)(nil), fileHeader).Return(uploadedURL, nil)

		mockStore.On("ConfirmPayment", ctx, db.ConfirmPaymentParams{
			ID:           payID,
			ProofFileUrl: pgtype.Text{String: uploadedURL, Valid: true},
		}).Return(db.Payment{
			ID:           payID,
			HouseholdID:  hID,
			WasteID:      wID,
			Amount:       amount,
			Status:       db.PaymentStatusPaid,
			ProofFileUrl: pgtype.Text{String: uploadedURL, Valid: true},
		}, nil)

		res, err := paymentSvc.Confirm(ctx, nil, payIDStr, fileHeader)

		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "paid", res.Status)
		assert.NotNil(t, res.ProofFileURL)
		assert.Equal(t, uploadedURL, *res.ProofFileURL)
		mockStore.AssertExpectations(t)
		mockUploader.AssertExpectations(t)
	})
}

// Client input validation maps to ErrInvalidInput
func TestPaymentService_InvalidInput(t *testing.T) {
	ctx := context.Background()

	t.Run("create: amount nol", func(t *testing.T) {
		mockStore := new(MockStore)
		mockUploader := new(MockUploader)
		paymentSvc := service.NewPaymentService(mockStore, mockUploader)

		hIDStr, _ := newTestUUID()
		wIDStr, _ := newTestUUID()

		res, err := paymentSvc.Create(ctx, domain.CreatePaymentRequest{
			HouseholdID: hIDStr,
			WasteID:     wIDStr,
			Amount:      0,
		})
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Nil(t, res)
		mockStore.AssertExpectations(t)
	})

	t.Run("create: uuid tidak valid", func(t *testing.T) {
		mockStore := new(MockStore)
		mockUploader := new(MockUploader)
		paymentSvc := service.NewPaymentService(mockStore, mockUploader)

		res, err := paymentSvc.Create(ctx, domain.CreatePaymentRequest{
			HouseholdID: "bukan-uuid",
			WasteID:     "bukan-uuid",
			Amount:      1000,
		})
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Nil(t, res)
		mockStore.AssertExpectations(t)
	})

	t.Run("list: status tidak valid", func(t *testing.T) {
		mockStore := new(MockStore)
		mockUploader := new(MockUploader)
		paymentSvc := service.NewPaymentService(mockStore, mockUploader)

		res, meta, err := paymentSvc.List(ctx, "bukan-status", "", "", "", "", 10)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Nil(t, res)
		assert.Nil(t, meta)
		mockStore.AssertExpectations(t)
	})

	t.Run("list: tanggal tidak valid", func(t *testing.T) {
		mockStore := new(MockStore)
		mockUploader := new(MockUploader)
		paymentSvc := service.NewPaymentService(mockStore, mockUploader)

		res, meta, err := paymentSvc.List(ctx, "", "", "32-13-2026", "", "", 10)
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Nil(t, res)
		assert.Nil(t, meta)
		mockStore.AssertExpectations(t)
	})
}
