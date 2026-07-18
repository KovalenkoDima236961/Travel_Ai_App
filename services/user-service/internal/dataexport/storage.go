// Package dataexport provides private local storage for account portability
// packages. The file key is opaque and never exposed in HTTP responses.
package dataexport

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalStorage struct{ baseDir string }

func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "./data/exports"
	}
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve data export dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("create data export dir: %w", err)
	}
	return &LocalStorage{baseDir: abs}, nil
}

func (s *LocalStorage) Save(key string, content []byte) (int64, string, error) {
	path, err := s.pathForKey(key)
	if err != nil {
		return 0, "", err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return 0, "", fmt.Errorf("create export package: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(content); err != nil {
		_ = os.Remove(path)
		return 0, "", fmt.Errorf("write export package: %w", err)
	}
	sum := sha256.Sum256(content)
	return int64(len(content)), hex.EncodeToString(sum[:]), nil
}

func (s *LocalStorage) Open(key string) (io.ReadCloser, error) {
	path, err := s.pathForKey(key)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open export package: %w", err)
	}
	return file, nil
}
func (s *LocalStorage) Delete(key string) error {
	path, err := s.pathForKey(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete export package: %w", err)
	}
	return nil
}
func (s *LocalStorage) pathForKey(key string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(key)))
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid export storage key")
	}
	path := filepath.Join(s.baseDir, clean)
	rel, err := filepath.Rel(s.baseDir, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid export storage key")
	}
	return path, nil
}
