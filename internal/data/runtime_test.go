package data

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestRuntimeMarshalJSON(t *testing.T) {
	b, err := Runtime(101).MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	if string(b) != `"101 mins"` {
		t.Errorf("MarshalJSON() = %s, want %q", b, "101 mins")
	}
}

func TestRuntimeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Runtime
		wantErr bool
	}{
		{"valid value", `"101 mins"`, 101, false},
		{"single digit value", `"7 mins"`, 7, false},
		{"missing units", `"101"`, 0, true},
		{"wrong unit", `"101 min"`, 0, true},
		{"trailing content", `"101 mins extra"`, 0, true},
		{"non numeric", `"abc mins"`, 0, true},
		{"empty string", `""`, 0, true},
		{"unquoted value", `101 mins`, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r Runtime
			err := r.UnmarshalJSON([]byte(tt.input))

			if tt.wantErr {
				if !errors.Is(err, ErrInvalidRuntimeFormat) {
					t.Fatalf("UnmarshalJSON(%s) error = %v, want %v", tt.input, err, ErrInvalidRuntimeFormat)
				}
				return
			}

			if err != nil {
				t.Fatalf("UnmarshalJSON(%s) error = %v", tt.input, err)
			}
			if r != tt.want {
				t.Errorf("UnmarshalJSON(%s) = %d, want %d", tt.input, r, tt.want)
			}
		})
	}
}

func TestRuntimeJSONRoundTrip(t *testing.T) {
	original := Runtime(94)

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded Runtime
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", encoded, err)
	}

	if decoded != original {
		t.Errorf("round trip = %d, want %d", decoded, original)
	}
}
