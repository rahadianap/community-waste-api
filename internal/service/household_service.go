package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	db "github.com/rahadianap/community-waste-api/db/sqlc"
	"github.com/rahadianap/community-waste-api/internal/domain"
	"github.com/rahadianap/community-waste-api/internal/repository"
	"github.com/rahadianap/community-waste-api/pkg/response"
)

type HouseholdService interface {
	Create(ctx context.Context, req domain.CreateHouseholdRequest) (*domain.HouseholdResponse, error)
	GetByID(ctx context.Context, id string) (*domain.HouseholdResponse, error)
	List(ctx context.Context, page, limit int) ([]domain.HouseholdResponse, *response.Meta, error)
	Delete(ctx context.Context, id string) error
}

type householdService struct {
	store repository.Store
}

func NewHouseholdService(store repository.Store) HouseholdService {
	return &householdService{store: store}
}

func (s *householdService) Create(ctx context.Context, req domain.CreateHouseholdRequest) (*domain.HouseholdResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	household, err := s.store.CreateHousehold(ctx, db.CreateHouseholdParams{
		OwnerName: req.OwnerName,
		Address:   req.Address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create household: %w", err)
	}

	res := domain.ToHouseholdResponse(household)
	return &res, nil
}

func (s *householdService) GetByID(ctx context.Context, id string) (*domain.HouseholdResponse, error) {
	pgUUID, err := domain.StringToUUID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid: %w", err)
	}

	household, err := s.store.GetHouseholdByID(ctx, pgUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrHouseholdNotFound
		}
		return nil, fmt.Errorf("failed to get household: %w", err)
	}

	res := domain.ToHouseholdResponse(household)
	return &res, nil
}

func (s *householdService) List(ctx context.Context, page, limit int) ([]domain.HouseholdResponse, *response.Meta, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	households, err := s.store.ListHouseholds(ctx, db.ListHouseholdsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list households: %w", err)
	}

	totalItems, err := s.store.CountHouseholds(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to count households: %w", err)
	}

	items := make([]domain.HouseholdResponse, len(households))
	for i, h := range households {
		items[i] = domain.ToHouseholdResponse(h)
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

func (s *householdService) Delete(ctx context.Context, id string) error {
	pgUUID, err := domain.StringToUUID(id)
	if err != nil {
		return fmt.Errorf("invalid uuid: %w", err)
	}

	// Verify exists first
	_, err = s.store.GetHouseholdByID(ctx, pgUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrHouseholdNotFound
		}
		return err
	}

	if err := s.store.DeleteHousehold(ctx, pgUUID); err != nil {
		return fmt.Errorf("failed to delete household: %w", err)
	}
	return nil
}
