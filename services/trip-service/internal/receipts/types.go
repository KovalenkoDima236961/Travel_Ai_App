package receipts

import (
	"context"
	"io"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type Config struct {
	StorageProvider   string
	LocalDir          string
	MaxFileSizeMB     int
	MaxFileSizeBytes  int64
	AllowedMIMEs      []string
	AllowedExtensions []string
	ScanningEnabled   bool
	ScanningFailOpen  bool
	OCREnabled        bool
	OCRProvider       entity.ReceiptOCRProvider
	OCRTimeout        time.Duration
	OCRFailOpen       bool
	StoreRawText      bool
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

// LocalPathProvider is implemented only by private local storage. It exposes a
// validated server-controlled path to the optional scanner, never to HTTP
// callers or API responses.
type LocalPathProvider interface {
	PathForScanning(storageKey string) (string, error)
}

type ScanResult struct {
	Available bool
	Clean     bool
	Threat    string
}

type FileScanner interface {
	Scan(ctx context.Context, filePath string) (ScanResult, error)
}

// NoopFileScanner keeps local development dependency-free. When scanning is
// enabled it reports itself unavailable so production can fail closed.
type NoopFileScanner struct{}

func (NoopFileScanner) Scan(context.Context, string) (ScanResult, error) {
	return ScanResult{Available: false, Clean: false}, nil
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
		StorageProvider:  "local",
		LocalDir:         "./data/receipts",
		MaxFileSizeMB:    10,
		MaxFileSizeBytes: 10 * 1024 * 1024,
		AllowedMIMEs: []string{
			"image/jpeg",
			"image/png",
			"image/webp",
			"application/pdf",
		},
		AllowedExtensions: []string{".jpg", ".jpeg", ".png", ".webp", ".pdf"},
		ScanningEnabled:   false,
		ScanningFailOpen:  false,
		OCREnabled:        true,
		OCRProvider:       entity.ReceiptOCRProviderMock,
		OCRTimeout:        30 * time.Second,
		OCRFailOpen:       true,
		StoreRawText:      true,
	}
}
