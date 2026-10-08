package domain

import "errors"

var (
	// Household Errors
	ErrHouseholdNotFound = errors.New("household not found")

	// Waste Pickup Errors & Business Rules
	ErrPickupNotFound           = errors.New("waste pickup not found")
	ErrPendingPaymentExists     = errors.New("household has pending payments and cannot create a new pickup request") // BR-01
	ErrPickupNotPending         = errors.New("pickup can only be scheduled when current status is pending")          // BR-02
	ErrSafetyCheckRequired      = errors.New("safety check must be true before scheduling electronic waste pickup")  // BR-03
	ErrPickupAlreadyCompleted   = errors.New("waste pickup is already completed")
	ErrPickupAlreadyCanceled    = errors.New("waste pickup is already canceled")
	ErrInvalidPickupStatus      = errors.New("invalid pickup status transition")

	// Payment Errors & Business Rules
	ErrPaymentNotFound    = errors.New("payment not found")
	ErrPaymentAlreadyPaid = errors.New("payment has already been completed")
	ErrProofFileRequired  = errors.New("proof of payment file is required") // BR-05
	ErrInvalidFileType    = errors.New("invalid file type: only jpg, jpeg, png, and pdf are supported")
	ErrFileSizeExceeded   = errors.New("file size exceeds maximum allowed limit")
)
