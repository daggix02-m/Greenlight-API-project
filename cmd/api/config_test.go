package main

import "testing"

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		secret  string
		wantErr bool
	}{
		{"development with a custom secret", "development", "a-sufficiently-long-development-secret", false},
		{"development with the default secret", "development", devJWTSecret, false},
		{"staging with a strong secret", "staging", "0123456789abcdef0123456789abcdef", false},
		{"missing secret", "development", "", true},
		{"production with a short secret", "production", "short", true},
		{"production with exactly 31 bytes", "production", "1234567890123456789012345678901", true},
		{"production with exactly 32 bytes", "production", "12345678901234567890123456789012", false},
		{"production with the default secret", "production", devJWTSecret, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config{env: tt.env}
			cfg.jwt.secret = tt.secret

			err := validateConfig(cfg)

			if tt.wantErr {
				if err == nil {
					t.Fatal("validateConfig() error = nil, want an error")
				}
				return
			}

			if err != nil {
				t.Fatalf("validateConfig() error = %v, want nil", err)
			}
		})
	}
}
