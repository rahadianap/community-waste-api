package unit

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/rahadianap/community-waste-api/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func encodeRaw(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

func TestCursorRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 10, 8, 10, 30, 45, 123456000, time.UTC)
	id := "550e8400-e29b-41d4-a716-446655440000"

	cursor := domain.EncodeCursor(createdAt, id)
	assert.NotContains(t, cursor, "=")
	assert.NotContains(t, cursor, "+")
	assert.NotContains(t, cursor, "/")

	decodedTS, decodedID, err := domain.DecodeCursor(cursor)
	require.NoError(t, err)
	assert.True(t, decodedTS.Equal(createdAt), "expected %v got %v", createdAt, decodedTS)
	assert.Equal(t, id, decodedID)
}

func TestDecodeCursorInvalid(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"not base64", "!!bukan-base64!!"},
		{"missing separator", encodeRaw("tanpa-pemisah")},
		{"invalid timestamp", encodeRaw("bukan-waktu|550e8400-e29b-41d4-a716-446655440000")},
		{"invalid uuid", encodeRaw("2026-10-08T10:30:45Z|bukan-uuid")},
		{"wrong separator", encodeRaw("2026-10-08T10:30:45Z;550e8400-e29b-41d4-a716-446655440000")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := domain.DecodeCursor(tt.cursor)
			assert.Error(t, err)
		})
	}
}
