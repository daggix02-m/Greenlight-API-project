package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/daggix02-m/greenlight-let-s-go-further-/internal/jsonlog"
)

// The metrics middleware registers expvar variables when routes() is built,
// and expvar panics on duplicate names — so the test application is shared
// across every test in this package and built exactly once.
var (
	sharedOnce    sync.Once
	sharedApp     *application
	sharedHandler http.Handler
)

func testApplication() (*application, http.Handler) {
	sharedOnce.Do(func() {
		cfg := config{port: 4000, env: "test"}
		cfg.limiter.rps = 1
		cfg.limiter.burst = 1
		cfg.jwt.secret = "test-secret"

		sharedApp = &application{
			config: cfg,
			logger: jsonlog.New(io.Discard, jsonlog.LevelFatal),
		}
		sharedHandler = sharedApp.routes()
	})

	return sharedApp, sharedHandler
}

func TestHealthcheckHandler(t *testing.T) {
	_, h := testApplication()

	req := httptest.NewRequest(http.MethodGet, "/v1/healthcheck", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rr.Code, http.StatusOK, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body struct {
		Status string `json:"status"`
		System struct {
			Environment string `json:"environment"`
			Version     string `json:"version"`
		} `json:"system_info"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v (%s)", err, rr.Body.String())
	}

	if body.Status != "available" {
		t.Errorf("status = %q, want available", body.Status)
	}
	if body.System.Environment != "test" {
		t.Errorf("environment = %q, want test", body.System.Environment)
	}
	if body.System.Version != version {
		t.Errorf("version = %q, want %q", body.System.Version, version)
	}
}

func TestUnknownRouteReturnsJSONNotFound(t *testing.T) {
	_, h := testApplication()

	req := httptest.NewRequest(http.MethodGet, "/v1/nonsense", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}

	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v (%s)", err, rr.Body.String())
	}
	if body.Error != "the requested resource could not be found" {
		t.Errorf("error = %q, want %q", body.Error, "the requested resource could not be found")
	}
}

func TestMethodNotAllowedReturnsJSONError(t *testing.T) {
	_, h := testApplication()

	req := httptest.NewRequest(http.MethodPost, "/v1/healthcheck", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}

	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v (%s)", err, rr.Body.String())
	}
	if !strings.Contains(body.Error, "POST") {
		t.Errorf("error = %q, want it to mention POST", body.Error)
	}
}
