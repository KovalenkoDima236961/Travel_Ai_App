package receipts

import (
	"context"
	"io"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type Config struct {
	StorageProvider string
	LocalDir        string
	MaxFileSizeMB   int
	AllowedMIMEs    []string
	OCREnabled      bool
	OCRProvider     entity.ReceiptOCRProvider
	OCRTimeout      time.Duration
	OCRFailOpen     bool
	StoreRawText    bool
}

type StorageSaveInput struct {
	Reader           io.Reader
	OriginalFilename string
	ContentType      string
}

type StorageSaveResult struct {
	StorageKey string
	SHA256     string
	SizeBytes  int64
}

type StoredFile struct {
	Reader io.ReadCloser
}

type Storage interface {
	Save(ctx context.Context, input StorageSaveInput) (StorageSaveResult, error)
	Open(ctx context.Context, storageKey string) (*StoredFile, error)
	Delete(ctx context.Context, storageKey string) error
}

type OCRMetadata struct {
	OriginalFilename string
	ContentType      string
	SizeBytes        int64
}

type OCRTripContext struct {
	DefaultCurrency string
}

type OCRProvider interface {
	Name() entity.ReceiptOCRProvider
	Extract(ctx context.Context, file io.Reader, metadata OCRMetadata, trip OCRTripContext) (*entity.ReceiptOCRResult, error)
}

func DefaultConfig() Config {
	return Config{
		StorageProvider: "local",
		LocalDir:        "./data/receipts",
		MaxFileSizeMB:   10,
		AllowedMIMEs: []string{
			"image/jpeg",
			"image/png",
			"image/webp",
			"application/pdf",
		},
		OCREnabled:   true,
		OCRProvider:  entity.ReceiptOCRProviderMock,
		OCRTimeout:   30 * time.Second,
		OCRFailOpen:  true,
		StoreRawText: true,
	}
}
