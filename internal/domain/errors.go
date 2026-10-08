package domain

import "errors"

var (
	// General Errors
	ErrInvalidInput = errors.New("invalid input")

	// Household Errors
	ErrHouseholdNotFound = errors.New("household not found")

	// Waste Pickup Errors & Business Rules
	ErrPickupNotFound         = errors.New("waste pickup not found")
	ErrPendingPaymentExists   = errors.New("household has pending payments and cannot create a new pickup request")
	ErrPickupNotPending       = errors.New("pickup can only be scheduled when current status is pending")
	ErrSafetyCheckRequired    = errors.New("safety check must be true before scheduling electronic waste pickup")
	ErrPickupAlreadyCompleted = errors.New("waste pickup is already completed")
	ErrPickupAlreadyCanceled  = errors.New("waste pickup is already canceled")
	ErrCannotCancelCompleted  = errors.New("tidak bisa membatalkan pickup yang sudah selesai")

	// Payment Errors & Business Rules
	ErrPaymentNotFound    = errors.New("payment not found")
	ErrPaymentAlreadyPaid = errors.New("payment has already been completed")
	ErrProofFileRequired  = errors.New("proof of payment file is required")
	ErrInvalidFileType    = errors.New("invalid file type: only jpg, jpeg, png, and pdf are supported")
	ErrFileSizeExceeded   = errors.New("file size exceeds maximum allowed limit")
)
