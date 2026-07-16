package receipts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type LocalStorage struct {
	baseDir string
}

func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		baseDir = "./data/receipts"
	}
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve receipt storage dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("create receipt storage dir: %w", err)
	}
	return &LocalStorage{baseDir: abs}, nil
}

func (s *LocalStorage) Save(ctx context.Context, input StorageSaveInput) (StorageSaveResult, error) {
	if input.Reader == nil {
		return StorageSaveResult{}, fmt.Errorf("receipt file reader is required")
	}
	ext := extensionForContentType(input.ContentType)
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(input.OriginalFilename))
	}
	if ext == "" {
		return StorageSaveResult{}, fmt.Errorf("receipt file extension is required")
	}
	now := time.Now().UTC()
	key := filepath.ToSlash(filepath.Join(
		"receipts",
		fmt.Sprintf("%04d", now.Year()),
		fmt.Sprintf("%02d", int(now.Month())),
		uuid.NewString()+ext,
	))
	path, err := s.pathForKey(key)
	if err != nil {
		return StorageSaveResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return StorageSaveResult{}, fmt.Errorf("create receipt storage path: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return StorageSaveResult{}, fmt.Errorf("create receipt file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	written, err := io.Copy(file, io.TeeReader(input.Reader, hasher))
	if err != nil {
		_ = os.Remove(path)
		return StorageSaveResult{}, fmt.Errorf("write receipt file: %w", err)
	}
	select {
	case <-ctx.Done():
		_ = os.Remove(path)
		return StorageSaveResult{}, ctx.Err()
	default:
	}
	return StorageSaveResult{
		StorageKey: key,
		SHA256:     hex.EncodeToString(hasher.Sum(nil)),
		SizeBytes:  written,
	}, nil
}

func (s *LocalStorage) Open(ctx context.Context, storageKey string) (*StoredFile, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	path, err := s.pathForKey(storageKey)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open receipt file: %w", err)
	}
	return &StoredFile{Reader: file}, nil
}

func (s *LocalStorage) Delete(ctx context.Context, storageKey string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	path, err := s.pathForKey(storageKey)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete receipt file: %w", err)
	}
	return nil
}

func (s *LocalStorage) PathForScanning(storageKey string) (string, error) {
	return s.pathForKey(storageKey)
}

func (s *LocalStorage) pathForKey(storageKey string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(storageKey)))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("invalid receipt storage key")
	}
	path := filepath.Join(s.baseDir, clean)
	rel, err := filepath.Rel(s.baseDir, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid receipt storage key")
	}
	return path, nil
}

func extensionForContentType(contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	default:
		exts, _ := mime.ExtensionsByType(contentType)
		if len(exts) > 0 {
			return exts[0]
		}
		return ""
	}
}
