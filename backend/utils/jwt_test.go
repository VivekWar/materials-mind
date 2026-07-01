package utils

import "testing"

func TestGenerateAndVerifyToken_Roundtrip(t *testing.T) {
	jwtSecret = []byte("test-secret-for-unit-tests")

	token, err := GenerateToken("42")
	if err != nil {
		t.Fatalf("expected no error generating token, got: %v", err)
	}

	userID, err := VerifyToken(token)
	if err != nil {
		t.Fatalf("expected token to verify, got error: %v", err)
	}
	if userID != "42" {
		t.Fatalf("expected user_id 42, got: %s", userID)
	}
}

func TestVerifyToken_RejectsGarbage(t *testing.T) {
	jwtSecret = []byte("test-secret-for-unit-tests")

	if _, err := VerifyToken("not-a-real-token"); err == nil {
		t.Fatal("expected error for malformed token, got nil")
	}
}

func TestVerifyToken_RejectsTokenSignedWithDifferentSecret(t *testing.T) {
	jwtSecret = []byte("secret-a")
	token, err := GenerateToken("7")
	if err != nil {
		t.Fatalf("expected no error generating token, got: %v", err)
	}

	jwtSecret = []byte("secret-b")
	if _, err := VerifyToken(token); err == nil {
		t.Fatal("expected verification to fail after secret rotation, got nil error")
	}
}
