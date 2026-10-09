package data

import (
	"crypto/sha256"
	"testing"
	"time"

	"github.com/daggix02-m/greenlight-let-s-go-further-/internal/validator"
)

func TestValidateTokenPlaintext(t *testing.T) {
	valid := "VGF54QR5FDJOPJVYLOYZZHP5GQ"

	tests := []struct {
		name  string
		token string
		valid bool
	}{
		{"valid token", valid, true},
		{"empty value", "", false},
		{"too short", valid[:25], false},
		{"too long", valid + "X", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.New()
			ValidateTokenPlaintext(v, tt.token)

			if got := v.Valid(); got != tt.valid {
				t.Errorf("ValidateTokenPlaintext(%q) valid = %v, want %v (errors: %v)", tt.token, got, tt.valid, v.Errors)
			}
		})
	}
}

func TestGenerateToken(t *testing.T) {
	before := time.Now().Add(-time.Second)
	token, err := generateToken(7, time.Hour, ScopeActivation)
	after := time.Now().Add(time.Second)

	if err != nil {
		t.Fatalf("generateToken() error = %v", err)
	}

	if len(token.Plaintext) != 26 {
		t.Errorf("Plaintext length = %d, want 26", len(token.Plaintext))
	}

	wantHash := sha256.Sum256([]byte(token.Plaintext))
	if string(token.Hash) != string(wantHash[:]) {
		t.Error("Hash is not the SHA-256 of Plaintext")
	}

	if token.UserID != 7 {
		t.Errorf("UserID = %d, want 7", token.UserID)
	}
	if token.Scope != ScopeActivation {
		t.Errorf("Scope = %q, want %q", token.Scope, ScopeActivation)
	}
	if token.Expiry.Before(before) || token.Expiry.After(after.Add(time.Hour)) {
		t.Errorf("Expiry = %v, want within the next hour", token.Expiry)
	}
}

func TestGenerateTokenProducesUniquePlaintexts(t *testing.T) {
	first, err := generateToken(1, time.Hour, ScopeActivation)
	if err != nil {
		t.Fatalf("generateToken() error = %v", err)
	}
	second, err := generateToken(1, time.Hour, ScopeActivation)
	if err != nil {
		t.Fatalf("generateToken() error = %v", err)
	}

	if first.Plaintext == second.Plaintext {
		t.Error("two generated tokens share the same plaintext")
	}
}
