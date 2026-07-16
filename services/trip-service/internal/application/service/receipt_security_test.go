package service

import (
	"bytes"
	"strings"
	"testing"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/receipts"
)

func TestValidateReceiptFileSecurityBoundaries(t *testing.T) {
	jpeg := append([]byte{0xff, 0xd8, 0xff, 0xe0}, bytes.Repeat([]byte{0}, 32)...)
	tests := []struct {
		name        string
		filename    string
		contentType string
		body        []byte
		size        int64
		wantErr     bool
	}{
		{name: "valid jpeg", filename: "receipt.jpg", contentType: "image/jpeg", body: jpeg, size: int64(len(jpeg))},
		{name: "invalid extension", filename: "receipt.exe", contentType: "image/jpeg", body: jpeg, size: int64(len(jpeg)), wantErr: true},
		{name: "mime mismatch", filename: "receipt.jpg", contentType: "application/pdf", body: jpeg, size: int64(len(jpeg)), wantErr: true},
		{name: "spoofed plain text", filename: "receipt.jpg", contentType: "image/jpeg", body: []byte("not an image"), size: 12, wantErr: true},
		{name: "empty", filename: "receipt.pdf", contentType: "application/pdf", body: nil, size: 0, wantErr: true},
		{name: "oversized", filename: "receipt.jpg", contentType: "image/jpeg", body: jpeg, size: 11 * 1024 * 1024, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &Service{receiptConfig: receipts.DefaultConfig()}
			_, _, err := svc.validateReceiptFile(appdto.UploadReceiptInput{
				OriginalFilename: tt.filename,
				ContentType:      tt.contentType,
				SizeBytes:        tt.size,
				File:             bytes.NewReader(tt.body),
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("error=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestCleanReceiptFilenameRemovesTraversal(t *testing.T) {
	for _, input := range []string{"../../private/receipt.pdf", `..\..\private\receipt.pdf`} {
		clean := cleanReceiptFilename(input)
		if clean != "receipt.pdf" || strings.Contains(clean, "..") || strings.Contains(clean, "/") || strings.Contains(clean, "\\") {
			t.Fatalf("unsafe filename %q", clean)
		}
	}
}
