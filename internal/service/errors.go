package service

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/rahadianap/community-waste-api/internal/domain"
)

// wrapper internal error
func wrap(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}

func invalidInput(err error) error {
	return fmt.Errorf("%w: %v", domain.ErrInvalidInput, err)
}

func invalidInputf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", domain.ErrInvalidInput, fmt.Sprintf(format, args...))
}

func lookup(err error, notFound error, msg string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}
	return fmt.Errorf("%s: %w", msg, err)
}
