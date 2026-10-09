package data

import (
	"testing"
	"time"

	"github.com/daggix02-m/greenlight-let-s-go-further-/internal/validator"
)

func validMovie() *Movie {
	return &Movie{
		Title:   "Heat",
		Year:    1995,
		Runtime: Runtime(101),
		Genres:  []string{"Crime", "Drama"},
	}
}

func TestValidateMovieAcceptsValidMovie(t *testing.T) {
	v := validator.New()
	ValidateMovie(v, validMovie())

	if !v.Valid() {
		t.Errorf("ValidateMovie() errors = %v, want none", v.Errors)
	}
}

func TestValidateMovieRejectsInvalidMovie(t *testing.T) {
	currentYear := int32(time.Now().Year())

	tests := []struct {
		name    string
		mutate  func(*Movie)
		wantKey string
		wantMsg string
	}{
		{
			"empty title",
			func(m *Movie) { m.Title = "" },
			"title",
			"must be provided",
		},
		{
			"title too long",
			func(m *Movie) { m.Title = string(make([]byte, 501)) },
			"title",
			"must not be more than 500 bytes long",
		},
		{
			"year missing",
			func(m *Movie) { m.Year = 0 },
			"year",
			"must be provided",
		},
		{
			"year before cinema",
			func(m *Movie) { m.Year = 1887 },
			"year",
			"must be greater than 1888",
		},
		{
			"year in the future",
			func(m *Movie) { m.Year = currentYear + 1 },
			"year",
			"must not be in the future",
		},
		{
			"runtime missing",
			func(m *Movie) { m.Runtime = 0 },
			"runtime",
			"must be provided",
		},
		{
			"negative runtime",
			func(m *Movie) { m.Runtime = -1 },
			"runtime",
			"must be a positive integer",
		},
		{
			"nil genres",
			func(m *Movie) { m.Genres = nil },
			"genres",
			"must be provided",
		},
		{
			"empty genres",
			func(m *Movie) { m.Genres = []string{} },
			"genres",
			"must contain at least 1 genre",
		},
		{
			"too many genres",
			func(m *Movie) { m.Genres = []string{"a", "b", "c", "d", "e", "f"} },
			"genres",
			"must not contain more than 5 genres",
		},
		{
			"duplicate genres",
			func(m *Movie) { m.Genres = []string{"Crime", "Crime"} },
			"genres",
			"must not contain duplicate values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			movie := validMovie()
			tt.mutate(movie)

			v := validator.New()
			ValidateMovie(v, movie)

			if v.Valid() {
				t.Fatalf("ValidateMovie() errors = none, want an error for %q", tt.wantKey)
			}
			got, exists := v.Errors[tt.wantKey]
			if !exists {
				t.Fatalf("ValidateMovie() errors = %v, want key %q", v.Errors, tt.wantKey)
			}
			if got != tt.wantMsg {
				t.Errorf("Errors[%q] = %q, want %q", tt.wantKey, got, tt.wantMsg)
			}
		})
	}
}
