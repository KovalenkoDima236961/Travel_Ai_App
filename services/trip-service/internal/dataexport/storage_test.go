package dataexport

import (
	"io"
	"testing"
)

func TestLocalStorageSaveCreatesNestedDirectories(t *testing.T) {
	storage, err := NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}

	key, size, checksum, err := storage.Save("trip-exports/test.zip", []byte("hello"))
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if key != "trip-exports/test.zip" || size != 5 || checksum == "" {
		t.Fatalf("unexpected save result: key=%q size=%d checksum=%q", key, size, checksum)
	}

	file, err := storage.Open(key)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer file.Close()
	contents, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(contents) != "hello" {
		t.Fatalf("unexpected contents %q", string(contents))
	}
}

func TestLocalStorageSaveRejectsTraversalKey(t *testing.T) {
	storage, err := NewLocalStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalStorage() error = %v", err)
	}

	if _, _, _, err := storage.Save("../test.zip", []byte("hello")); err == nil {
		t.Fatal("expected traversal key to be rejected")
	}
}
