package domain

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/rahadianap/community-waste-api/db/sqlc"
)

// --- Household DTOs ---

type CreateHouseholdRequest struct {
	OwnerName string `json:"owner_name"`
	Address   string `json:"address"`
}

func (r *CreateHouseholdRequest) Validate() error {
	if strings.TrimSpace(r.OwnerName) == "" {
		return fmt.Errorf("owner_name is required")
	}
	if strings.TrimSpace(r.Address) == "" {
		return fmt.Errorf("address is required")
	}
	return nil
}

type HouseholdResponse struct {
	ID        string    `json:"id"`
	OwnerName string    `json:"owner_name"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToHouseholdResponse(h db.Household) HouseholdResponse {
	return HouseholdResponse{
		ID:        UUIDToString(h.ID),
		OwnerName: h.OwnerName,
		Address:   h.Address,
		CreatedAt: h.CreatedAt.Time,
		UpdatedAt: h.UpdatedAt.Time,
	}
}

// --- Waste Pickup DTOs ---

type CreatePickupRequest struct {
	HouseholdID string `json:"household_id"`
	Type        string `json:"type"`
	SafetyCheck bool   `json:"safety_check"`
}

func (r *CreatePickupRequest) Validate() error {
	if strings.TrimSpace(r.HouseholdID) == "" {
		return fmt.Errorf("household_id is required")
	}
	if _, err := uuid.Parse(r.HouseholdID); err != nil {
		return fmt.Errorf("invalid household_id uuid format")
	}

	validTypes := map[string]bool{
		"organic":    true,
		"plastic":    true,
		"paper":      true,
		"electronic": true,
	}
	if !validTypes[strings.ToLower(r.Type)] {
		return fmt.Errorf("invalid type: must be organic, plastic, paper, or electronic")
	}
	return nil
}

type SchedulePickupRequest struct {
	PickupDate string `json:"pickup_date"`
}

func (r *SchedulePickupRequest) Validate() (time.Time, error) {
	if strings.TrimSpace(r.PickupDate) == "" {
		return time.Time{}, fmt.Errorf("pickup_date is required")
	}
	t, err := time.Parse(time.RFC3339, r.PickupDate)
	if err != nil {
		// Also try layout "2006-01-02 15:04:05"
		t, err = time.Parse("2006-01-02 15:04:05", r.PickupDate)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid pickup_date format, expected RFC3339 (e.g. 2026-10-15T09:00:00Z)")
		}
	}
	return t, nil
}

type PickupResponse struct {
	ID          string     `json:"id"`
	HouseholdID string     `json:"household_id"`
	Type        string     `json:"type"`
	Status      string     `json:"status"`
	PickupDate  *time.Time `json:"pickup_date"`
	SafetyCheck bool       `json:"safety_check"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func ToPickupResponse(p db.WastePickup) PickupResponse {
	var pickupDate *time.Time
	if p.PickupDate.Valid {
		pickupDate = &p.PickupDate.Time
	}

	return PickupResponse{
		ID:          UUIDToString(p.ID),
		HouseholdID: UUIDToString(p.HouseholdID),
		Type:        string(p.Type),
		Status:      string(p.Status),
		PickupDate:  pickupDate,
		SafetyCheck: p.SafetyCheck,
		CreatedAt:   p.CreatedAt.Time,
		UpdatedAt:   p.UpdatedAt.Time,
	}
}

// --- Payment DTOs ---

type CreatePaymentRequest struct {
	HouseholdID string  `json:"household_id"`
	WasteID     string  `json:"waste_id"`
	Amount      float64 `json:"amount"`
}

func (r *CreatePaymentRequest) Validate() error {
	if _, err := uuid.Parse(r.HouseholdID); err != nil {
		return fmt.Errorf("invalid household_id uuid format")
	}
	if _, err := uuid.Parse(r.WasteID); err != nil {
		return fmt.Errorf("invalid waste_id uuid format")
	}
	if r.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	return nil
}

type PaymentResponse struct {
	ID           string     `json:"id"`
	HouseholdID  string     `json:"household_id"`
	WasteID      string     `json:"waste_id"`
	Amount       float64    `json:"amount"`
	PaymentDate  *time.Time `json:"payment_date"`
	Status       string     `json:"status"`
	ProofFileURL *string    `json:"proof_file_url"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func ToPaymentResponse(p db.Payment) PaymentResponse {
	var paymentDate *time.Time
	if p.PaymentDate.Valid {
		paymentDate = &p.PaymentDate.Time
	}

	var proofURL *string
	if p.ProofFileUrl.Valid && p.ProofFileUrl.String != "" {
		proofURL = &p.ProofFileUrl.String
	}

	return PaymentResponse{
		ID:           UUIDToString(p.ID),
		HouseholdID:  UUIDToString(p.HouseholdID),
		WasteID:      UUIDToString(p.WasteID),
		Amount:       NumericToFloat64(p.Amount),
		PaymentDate:  paymentDate,
		Status:       string(p.Status),
		ProofFileURL: proofURL,
		CreatedAt:    p.CreatedAt.Time,
		UpdatedAt:    p.UpdatedAt.Time,
	}
}

// --- Reports DTOs ---

type WasteSummaryItem struct {
	Type   string `json:"type"`
	Status string `json:"status"`
	Total  int64  `json:"total"`
}

type PaymentStatusSummary struct {
	Status      string  `json:"status"`
	Count       int64   `json:"count"`
	TotalAmount float64 `json:"total_amount"`
}

type PaymentSummaryReport struct {
	ByStatus     []PaymentStatusSummary `json:"by_status"`
	TotalRevenue float64                `json:"total_revenue"`
}

// --- Helper Conversion Functions ---

func UUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	parsed, err := uuid.FromBytes(u.Bytes[:])
	if err != nil {
		return ""
	}
	return parsed.String()
}

func StringToUUID(s string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{
		Bytes: parsed,
		Valid: true,
	}, nil
}

func Float64ToNumeric(f float64) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	strVal := fmt.Sprintf("%.2f", f)
	if err := n.Scan(strVal); err != nil {
		return pgtype.Numeric{}, err
	}
	return n, nil
}

func NumericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid || n.Int == nil {
		return 0
	}
	f, _ := new(big.Rat).SetFrac(n.Int, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-n.Exp)), nil)).Float64()
	return f
}
