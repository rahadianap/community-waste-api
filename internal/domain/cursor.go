package domain

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const cursorSeparator = "|"

func EncodeCursor(createdAt time.Time, id string) string {
	raw := createdAt.UTC().Format(time.RFC3339Nano) + cursorSeparator + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(s string) (time.Time, string, error) {
	if strings.TrimSpace(s) == "" {
		return time.Time{}, "", fmt.Errorf("cursor kosong")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, "", err
	}
	parts := strings.SplitN(string(decoded), cursorSeparator, 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return time.Time{}, "", fmt.Errorf("format cursor tidak valid")
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", err
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		return time.Time{}, "", fmt.Errorf("id cursor bukan UUID: %w", err)
	}
	return ts.UTC(), parts[1], nil
}
