package receipts

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestLocalStorageUsesRandomPrivateKeyAndRejectsTraversal(t *testing.T) {
	storage, err := NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	result, err := storage.Save(context.Background(), StorageSaveInput{
		Reader:           bytes.NewReader([]byte("%PDF-1.7\n")),
		OriginalFilename: "../../sensitive-name.pdf",
		ContentType:      "application/pdf",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.StorageKey, "sensitive-name") || strings.Contains(result.StorageKey, "..") {
		t.Fatalf("storage key leaked or traversed: %q", result.StorageKey)
	}
	if _, err := storage.Open(context.Background(), "../../etc/passwd"); err == nil {
		t.Fatal("expected traversal storage key to be rejected")
	}
}
