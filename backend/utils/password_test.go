package utils

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("s3cure-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "s3cure-password" {
		t.Fatal("hash must not equal plaintext")
	}
	if err := CheckPassword("s3cure-password", hash); err != nil {
		t.Fatalf("CheckPassword valid: %v", err)
	}
	if err := CheckPassword("wrong-password", hash); err == nil {
		t.Fatal("expected error for wrong password")
	}
	if err := CheckPassword("s3cure-password", "not-a-bcrypt-hash"); err == nil {
		t.Fatal("expected error for malformed hash")
	}
}

func TestHashPasswordUniqueSalts(t *testing.T) {
	a, _ := HashPassword("same-password")
	b, _ := HashPassword("same-password")
	if a == b {
		t.Fatal("bcrypt salts should make identical passwords hash differently")
	}
}
