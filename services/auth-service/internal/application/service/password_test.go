package service

import "testing"

func TestBcryptPasswordHasher(t *testing.T) {
	hasher := NewPasswordHasher()
	password := "StrongPassword123!"

	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("hash returned error: %v", err)
	}
	if hash == password {
		t.Fatal("hash matches plaintext password")
	}
	if !hasher.Verify(password, hash) {
		t.Fatal("expected password verification to succeed")
	}
	if hasher.Verify("WrongPassword123!", hash) {
		t.Fatal("expected wrong password verification to fail")
	}
}

func TestValidatePasswordStrength(t *testing.T) {
	if err := ValidatePasswordStrength("StrongPassword123!"); err != nil {
		t.Fatalf("expected strong password to pass: %v", err)
	}
	if err := ValidatePasswordStrength("weakpass"); err == nil {
		t.Fatal("expected weak password to fail")
	}
}
