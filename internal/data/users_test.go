package data

import (
	"strings"
	"testing"

	"github.com/daggix02-m/greenlight-let-s-go-further-/internal/validator"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr string
	}{
		{"valid address", "alice@example.com", ""},
		{"empty value", "", "email"},
		{"malformed address", "not-an-email", "email"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.New()
			ValidateEmail(v, tt.email)

			if tt.wantErr == "" {
				if !v.Valid() {
					t.Fatalf("ValidateEmail(%q) errors = %v, want none", tt.email, v.Errors)
				}
				return
			}

			if v.Valid() {
				t.Fatalf("ValidateEmail(%q) errors = none, want an error", tt.email)
			}
			if _, exists := v.Errors[tt.wantErr]; !exists {
				t.Errorf("ValidateEmail(%q) errors = %v, want key %q", tt.email, v.Errors, tt.wantErr)
			}
		})
	}
}

func TestValidatePasswordPlaintext(t *testing.T) {
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{"valid password", "pa55word123", true},
		{"empty value", "", false},
		{"too short", "short7.", false},
		{"too long", strings.Repeat("a", 73), false},
		{"exactly 8 bytes", "12345678", true},
		{"exactly 72 bytes", strings.Repeat("a", 72), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validator.New()
			ValidatePasswordPlaintext(v, tt.password)

			if got := v.Valid(); got != tt.valid {
				t.Errorf("ValidatePasswordPlaintext(%q) valid = %v, want %v (errors: %v)", tt.password, got, tt.valid, v.Errors)
			}
		})
	}
}

func TestValidateUser(t *testing.T) {
	t.Run("valid user", func(t *testing.T) {
		user := &User{Name: "Alice Smith", Email: "alice@example.com"}
		if err := user.Password.Set("pa55word123"); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		v := validator.New()
		ValidateUser(v, user)

		if !v.Valid() {
			t.Errorf("ValidateUser() errors = %v, want none", v.Errors)
		}
	})

	t.Run("missing name", func(t *testing.T) {
		user := &User{Name: "", Email: "alice@example.com"}
		if err := user.Password.Set("pa55word123"); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		v := validator.New()
		ValidateUser(v, user)

		if got, exists := v.Errors["name"]; !exists || got != "must be provided" {
			t.Errorf("Errors[name] = %q (exists: %v), want \"must be provided\"", got, exists)
		}
	})

	t.Run("invalid email", func(t *testing.T) {
		user := &User{Name: "Alice", Email: "not-an-email"}
		if err := user.Password.Set("pa55word123"); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		v := validator.New()
		ValidateUser(v, user)

		if _, exists := v.Errors["email"]; !exists {
			t.Errorf("ValidateUser() errors = %v, want an error for email", v.Errors)
		}
	})

	t.Run("short password", func(t *testing.T) {
		user := &User{Name: "Alice", Email: "alice@example.com"}
		if err := user.Password.Set("short7."); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		v := validator.New()
		ValidateUser(v, user)

		if got, exists := v.Errors["password"]; !exists || got != "must be at least 8 bytes long" {
			t.Errorf("Errors[password] = %q (exists: %v), want %q", got, exists, "must be at least 8 bytes long")
		}
	})

	t.Run("activation flow without plaintext password", func(t *testing.T) {
		user := &User{Name: "Alice", Email: "alice@example.com"}
		if err := user.Password.Set("pa55word123"); err != nil {
			t.Fatalf("Set() error = %v", err)
		}
		user.Password.plaintext = nil

		v := validator.New()
		ValidateUser(v, user)

		if !v.Valid() {
			t.Errorf("ValidateUser() errors = %v, want none when only the hash is present", v.Errors)
		}
	})

	t.Run("panics without password hash", func(t *testing.T) {
		user := &User{Name: "Alice", Email: "alice@example.com"}

		defer func() {
			if recover() == nil {
				t.Error("ValidateUser() should panic when the password hash is missing")
			}
		}()

		v := validator.New()
		ValidateUser(v, user)
	})
}

func TestPasswordSetAndMatches(t *testing.T) {
	var p password

	if err := p.Set("pa55word123"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	if p.hash == nil {
		t.Fatal("Set() left hash nil")
	}
	if p.plaintext == nil || *p.plaintext != "pa55word123" {
		t.Errorf("Set() plaintext = %v, want pa55word123", p.plaintext)
	}

	matched, err := p.Matches("pa55word123")
	if err != nil {
		t.Fatalf("Matches() error = %v", err)
	}
	if !matched {
		t.Error("Matches() = false for the correct password")
	}

	matched, err = p.Matches("wrong-password")
	if err != nil {
		t.Fatalf("Matches() error = %v", err)
	}
	if matched {
		t.Error("Matches() = true for an incorrect password")
	}
}

func TestUserIsAnonymous(t *testing.T) {
	if !AnonymousUser.IsAnonymous() {
		t.Error("AnonymousUser.IsAnonymous() = false, want true")
	}

	user := &User{ID: 1, Name: "Alice"}
	if user.IsAnonymous() {
		t.Error("regular user IsAnonymous() = true, want false")
	}
}
