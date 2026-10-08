package service

import (
	"context"

	db "github.com/rahadianap/community-waste-api/db/sqlc"
	"github.com/rahadianap/community-waste-api/internal/domain"
	"github.com/rahadianap/community-waste-api/internal/repository"
	"github.com/rahadianap/community-waste-api/internal/response"
)

type HouseholdService interface {
	Create(ctx context.Context, req domain.CreateHouseholdRequest) (*domain.HouseholdResponse, error)
	GetByID(ctx context.Context, id string) (*domain.HouseholdResponse, error)
	List(ctx context.Context, cursorStr string, limit int) ([]domain.HouseholdResponse, *response.Meta, error)
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
		return nil, invalidInput(err)
	}

	household, err := s.store.CreateHousehold(ctx, db.CreateHouseholdParams{
		OwnerName: req.OwnerName,
		Address:   req.Address,
	})
	if err != nil {
		return nil, wrap(err, "gagal membuat data household")
	}

	res := domain.ToHouseholdResponse(household)
	return &res, nil
}

func (s *householdService) GetByID(ctx context.Context, id string) (*domain.HouseholdResponse, error) {
	pgUUID, err := domain.StringToUUID(id)
	if err != nil {
		return nil, invalidInputf("invalid uuid: %v", err)
	}

	household, err := s.store.GetHouseholdByID(ctx, pgUUID)
	if err != nil {
		return nil, lookup(err, domain.ErrHouseholdNotFound, "gagal mengambil data household")
	}

	res := domain.ToHouseholdResponse(household)
	return &res, nil
}

func (s *householdService) List(ctx context.Context, cursorStr string, limit int) ([]domain.HouseholdResponse, *response.Meta, error) {
	limit = normalizeLimit(limit)
	cursorCreatedAt, cursorID, err := decodeCursor(cursorStr)
	if err != nil {
		return nil, nil, err
	}

	households, err := s.store.ListHouseholds(ctx, db.ListHouseholdsParams{
		Limit:           int32(limit + 1),
		CursorCreatedAt: cursorCreatedAt,
		CursorID:        cursorID,
	})
	if err != nil {
		return nil, nil, wrap(err, "gagal memuat data household")
	}

	hasMore := len(households) > limit
	if hasMore {
		households = households[:limit]
	}

	items := make([]domain.HouseholdResponse, len(households))
	for i, h := range households {
		items[i] = domain.ToHouseholdResponse(h)
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

func (s *householdService) Delete(ctx context.Context, id string) error {
	pgUUID, err := domain.StringToUUID(id)
	if err != nil {
		return invalidInputf("invalid uuid: %v", err)
	}

	_, err = s.store.GetHouseholdByID(ctx, pgUUID)
	if err != nil {
		return lookup(err, domain.ErrHouseholdNotFound, "gagal mengambil data household")
	}

	if err := s.store.DeleteHousehold(ctx, pgUUID); err != nil {
		return wrap(err, "gagal menghapus data household")
	}
	return nil
}
