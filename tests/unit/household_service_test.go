package unit

import (
	"context"
	"testing"

	"github.com/rahadianap/community-waste-api/internal/domain"
	"github.com/rahadianap/community-waste-api/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestHouseholdService_InvalidInput(t *testing.T) {
	ctx := context.Background()

	t.Run("create: field wajib kosong", func(t *testing.T) {
		mockStore := new(MockStore)
		householdSvc := service.NewHouseholdService(mockStore)

		res, err := householdSvc.Create(ctx, domain.CreateHouseholdRequest{})
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Nil(t, res)
		mockStore.AssertExpectations(t)
	})

	t.Run("get by id: uuid tidak valid", func(t *testing.T) {
		mockStore := new(MockStore)
		householdSvc := service.NewHouseholdService(mockStore)

		res, err := householdSvc.GetByID(ctx, "bukan-uuid")
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		assert.Nil(t, res)
		mockStore.AssertExpectations(t)
	})

	t.Run("delete: uuid tidak valid", func(t *testing.T) {
		mockStore := new(MockStore)
		householdSvc := service.NewHouseholdService(mockStore)

		err := householdSvc.Delete(ctx, "bukan-uuid")
		assert.ErrorIs(t, err, domain.ErrInvalidInput)
		mockStore.AssertExpectations(t)
	})
}
