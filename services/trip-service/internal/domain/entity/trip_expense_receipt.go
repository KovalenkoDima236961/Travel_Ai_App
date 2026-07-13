package entity

import (
	"time"

	"github.com/google/uuid"
)

type ReceiptStatus string

const (
	ReceiptStatusUploaded         ReceiptStatus = "uploaded"
	ReceiptStatusProcessing       ReceiptStatus = "processing"
	ReceiptStatusExtracted        ReceiptStatus = "extracted"
	ReceiptStatusExtractionFailed ReceiptStatus = "extraction_failed"
	ReceiptStatusAttached         ReceiptStatus = "attached"
	ReceiptStatusDeleted          ReceiptStatus = "deleted"
)

type ReceiptOCRProvider string

const (
	ReceiptOCRProviderMock   ReceiptOCRProvider = "mock"
	ReceiptOCRProviderLocal  ReceiptOCRProvider = "local"
	ReceiptOCRProviderManual ReceiptOCRProvider = "manual"
)

type ReceiptOCRConfidence string

const (
	ReceiptOCRConfidenceLow    ReceiptOCRConfidence = "low"
	ReceiptOCRConfidenceMedium ReceiptOCRConfidence = "medium"
	ReceiptOCRConfidenceHigh   ReceiptOCRConfidence = "high"
)

type TripExpenseReceipt struct {
	ID               uuid.UUID
	TripID           uuid.UUID
	ExpenseID        *uuid.UUID
	Status           ReceiptStatus
	OriginalFilename string
	ContentType      string
	SizeBytes        int64
	StorageKey       string
	FileSHA256       *string
	CreatedByUserID  uuid.UUID
	UpdatedByUserID  *uuid.UUID
	DeletedAt        *time.Time
	DeletedByUserID  *uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ReceiptOCRResult struct {
	ID              uuid.UUID
	ReceiptID       uuid.UUID
	TripID          uuid.UUID
	Provider        ReceiptOCRProvider
	Status          ReceiptStatus
	Merchant        *string
	ExpenseDate     *time.Time
	Amount          *float64
	Currency        *string
	TaxAmount       *float64
	Category        *ExpenseCategory
	SuggestedTitle  *string
	Confidence      ReceiptOCRConfidence
	FieldConfidence map[string]ReceiptOCRConfidence
	Warnings        []string
	RawText         *string
	Normalized      map[string]any
	ErrorMessage    *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
