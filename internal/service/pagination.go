package service

import (
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rahadianap/community-waste-api/internal/domain"
)

const (
	defaultPageLimit = 10
	maxPageLimit     = 100
)

// normalizeLimit membatasi limit ke rentang [1, maxPageLimit] dengan default defaultPageLimit.
func normalizeLimit(limit int) int {
	if limit < 1 {
		return defaultPageLimit
	}
	if limit > maxPageLimit {
		return maxPageLimit
	}
	return limit
}

// decodeCursor memparse cursor jadi param keyset: kosong = halaman pertama, tidak valid = ErrInvalidInput.
func decodeCursor(cursorStr string) (pgtype.Timestamptz, pgtype.UUID, error) {
	if strings.TrimSpace(cursorStr) == "" {
		return pgtype.Timestamptz{}, pgtype.UUID{}, nil
	}
	ts, id, err := domain.DecodeCursor(cursorStr)
	if err != nil {
		return pgtype.Timestamptz{}, pgtype.UUID{}, invalidInputf("invalid cursor: %v", err)
	}
	uuid, err := domain.StringToUUID(id)
	if err != nil {
		return pgtype.Timestamptz{}, pgtype.UUID{}, invalidInputf("invalid cursor: %v", err)
	}
	return pgtype.Timestamptz{Time: ts, Valid: true}, uuid, nil
}
