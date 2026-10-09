package validator

import "testing"

func TestValidatorStartsValidAndRecordsFirstErrorPerKey(t *testing.T) {
	v := New()
	if !v.Valid() {
		t.Fatal("fresh validator should be valid")
	}

	v.Check(false, "title", "must be provided")
	if v.Valid() {
		t.Fatal("validator should be invalid after a failed check")
	}
	if got := v.Errors["title"]; got != "must be provided" {
		t.Errorf("Errors[title] = %q, want %q", got, "must be provided")
	}

	v.Check(false, "title", "second message")
	if got := v.Errors["title"]; got != "must be provided" {
		t.Errorf("first error for a key must win, got %q", got)
	}

	v.Check(true, "year", "must be provided")
	if _, exists := v.Errors["year"]; exists {
		t.Error("a passing check must not record an error")
	}
}

func TestIn(t *testing.T) {
	tests := []struct {
		name  string
		value string
		list  []string
		want  bool
	}{
		{"value present", "id", []string{"id", "title"}, true},
		{"value absent", "rating", []string{"id", "title"}, false},
		{"single element list", "x", []string{"x"}, true},
		{"empty list", "x", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := In(tt.value, tt.list...); got != tt.want {
				t.Errorf("In(%q, %v) = %v, want %v", tt.value, tt.list, got, tt.want)
			}
		})
	}
}

func TestMatches(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"simple address", "alice@example.com", true},
		{"address with plus tag", "first.last+tag@sub.domain.org", true},
		{"numeric local part", "user_1@test.co", true},
		{"empty value", "", false},
		{"no at sign", "plainaddress", false},
		{"double at sign", "a@@b.com", false},
		{"space in local part", "two words@example.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Matches(tt.value, EmailRX); got != tt.want {
				t.Errorf("Matches(%q, EmailRX) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestUnique(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   bool
	}{
		{"all distinct", []string{"crime", "drama"}, true},
		{"duplicate present", []string{"crime", "crime"}, false},
		{"duplicate later", []string{"crime", "drama", "crime"}, false},
		{"empty slice", []string{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Unique(tt.values); got != tt.want {
				t.Errorf("Unique(%v) = %v, want %v", tt.values, got, tt.want)
			}
		})
	}
}
