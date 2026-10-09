package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/daggix02-m/greenlight-let-s-go-further-/internal/validator"
	"github.com/julienschmidt/httprouter"
)

func TestReadIDParam(t *testing.T) {
	app := &application{}

	tests := []struct {
		name    string
		value   string
		wantID  int64
		wantErr bool
	}{
		{"valid id", "12", 12, false},
		{"large id", "9007199254740993", 9007199254740993, false},
		{"not a number", "abc", 0, true},
		{"zero", "0", 0, true},
		{"negative", "-1", 0, true},
		{"empty", "", 0, true},
		{"float", "1.5", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/v1/movies/1", nil)
			params := httprouter.Params{{Key: "id", Value: tt.value}}
			r = r.WithContext(context.WithValue(r.Context(), httprouter.ParamsKey, params))

			id, err := app.readIDParam(r)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("readIDParam(%q) = %d, want an error", tt.value, id)
				}
				return
			}

			if err != nil {
				t.Fatalf("readIDParam(%q) error = %v", tt.value, err)
			}
			if id != tt.wantID {
				t.Errorf("readIDParam(%q) = %d, want %d", tt.value, id, tt.wantID)
			}
		})
	}
}

func TestReadJSON(t *testing.T) {
	app := &application{}

	type moviePayload struct {
		Title string `json:"title"`
	}

	t.Run("valid body", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/v1/movies", strings.NewReader(`{"title":"Heat"}`))
		w := httptest.NewRecorder()

		var dst moviePayload
		if err := app.readJSON(w, r, &dst); err != nil {
			t.Fatalf("readJSON() error = %v", err)
		}
		if dst.Title != "Heat" {
			t.Errorf("dst.Title = %q, want Heat", dst.Title)
		}
	})

	tests := []struct {
		name       string
		body       string
		wantPrefix string
	}{
		{
			"malformed JSON",
			`{"title":`,
			"body contains badly-formed JSON",
		},
		{
			"unknown field",
			`{"title":"Heat","rating":8}`,
			"body contains unknown key",
		},
		{
			"empty body",
			``,
			"body must not be empty",
		},
		{
			"trailing value",
			`{"title":"Heat"}{"title":"Aliens"}`,
			"body must only contain a single JSON value",
		},
		{
			"wrong type",
			`{"title":123}`,
			`body contains incorrect JSON type for field "title"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/v1/movies", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			var dst moviePayload
			err := app.readJSON(w, r, &dst)
			if err == nil {
				t.Fatal("readJSON() error = nil, want an error")
			}
			if !strings.HasPrefix(err.Error(), tt.wantPrefix) {
				t.Errorf("readJSON() error = %q, want prefix %q", err, tt.wantPrefix)
			}
		})
	}

	t.Run("body too large", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/v1/movies", strings.NewReader(`{"title":"`+strings.Repeat("a", 1_048_577)+`"}`))
		w := httptest.NewRecorder()

		var dst moviePayload
		err := app.readJSON(w, r, &dst)
		if err == nil {
			t.Fatal("readJSON() error = nil, want an error")
		}
		want := "body must not be larger than 1048576 bytes"
		if err.Error() != want {
			t.Errorf("readJSON() error = %q, want %q", err, want)
		}
	})
}

func TestReadString(t *testing.T) {
	app := &application{}

	qs := url.Values{}
	if got := app.readString(qs, "title", "default"); got != "default" {
		t.Errorf("readString() on empty query = %q, want default", got)
	}

	qs.Set("title", "Heat")
	if got := app.readString(qs, "title", "default"); got != "Heat" {
		t.Errorf("readString() = %q, want Heat", got)
	}
}

func TestReadCSV(t *testing.T) {
	app := &application{}

	qs := url.Values{}
	if got := app.readCSV(qs, "genres", []string{"drama"}); len(got) != 1 || got[0] != "drama" {
		t.Errorf("readCSV() on empty query = %v, want the default slice", got)
	}

	defaults := []string{"drama"}
	qs.Set("genres", "crime,drama,thriller")
	got := app.readCSV(qs, "genres", defaults)

	want := []string{"crime", "drama", "thriller"}
	if len(got) != len(want) {
		t.Fatalf("readCSV() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("readCSV()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestReadInt(t *testing.T) {
	app := &application{}

	qs := url.Values{}
	v := validator.New()
	if got := app.readInt(qs, "year", 2000, v); got != 2000 {
		t.Errorf("readInt() on empty query = %d, want 2000", got)
	}
	if !v.Valid() {
		t.Errorf("readInt() on empty query recorded errors: %v", v.Errors)
	}

	qs.Set("year", "1995")
	v = validator.New()
	if got := app.readInt(qs, "year", 2000, v); got != 1995 {
		t.Errorf("readInt() = %d, want 1995", got)
	}
	if !v.Valid() {
		t.Errorf("readInt() on a valid value recorded errors: %v", v.Errors)
	}

	qs.Set("year", "not-a-number")
	v = validator.New()
	if got := app.readInt(qs, "year", 2000, v); got != 2000 {
		t.Errorf("readInt() on invalid value = %d, want fallback 2000", got)
	}
	if got, exists := v.Errors["year"]; !exists || got != "must be an integer value" {
		t.Errorf("Errors[year] = %q (exists: %v), want %q", got, exists, "must be an integer value")
	}
}
