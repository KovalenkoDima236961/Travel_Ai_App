package crypto

import "testing"

func TestStringCipherRoundTrip(t *testing.T) {
	cipher, err := NewStringCipher("12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	encrypted, err := cipher.EncryptString("token")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	decrypted, err := cipher.DecryptString(encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if decrypted != "token" {
		t.Fatalf("got %q", decrypted)
	}
}

func TestStringCipherUsesRandomNonce(t *testing.T) {
	cipher, err := NewStringCipher("12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	first, err := cipher.EncryptString("token")
	if err != nil {
		t.Fatalf("encrypt first: %v", err)
	}
	second, err := cipher.EncryptString("token")
	if err != nil {
		t.Fatalf("encrypt second: %v", err)
	}
	if first == second {
		t.Fatal("expected different ciphertext for same plaintext")
	}
}

func TestStringCipherRejectsInvalidCiphertext(t *testing.T) {
	cipher, err := NewStringCipher("12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	if _, err := cipher.DecryptString("not-base64"); err == nil {
		t.Fatal("expected invalid ciphertext to fail")
	}
}

func TestStringCipherWrongKeyFails(t *testing.T) {
	cipher, err := NewStringCipher("12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	encrypted, err := cipher.EncryptString("token")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	other, err := NewStringCipher("abcdefghijklmnopabcdefghijklmnop")
	if err != nil {
		t.Fatalf("new other cipher: %v", err)
	}
	if _, err := other.DecryptString(encrypted); err == nil {
		t.Fatal("expected wrong key to fail")
	}
}
