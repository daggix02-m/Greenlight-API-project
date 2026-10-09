package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/daggix02-m/greenlight-let-s-go-further-/internal/data"
	"github.com/pascaldekloe/jwt"
)

const (
	testIssuer   = "github.com/daggix02-m/greenlight-let-s-go-further-"
	testAudience = "github.com/daggix02-m/greenlight-let-s-go-further-"
	testSecret   = "test-secret-0123456789abcdef0123456789abcdef"
)

func authenticateTestApp() *application {
	app := &application{}
	app.config.jwt.secret = testSecret
	return app
}

func testHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func signTestToken(t *testing.T, mutate func(*jwt.Claims)) string {
	t.Helper()

	var claims jwt.Claims
	claims.Subject = "1"
	claims.Issued = jwt.NewNumericTime(time.Now())
	claims.NotBefore = jwt.NewNumericTime(time.Now())
	claims.Expires = jwt.NewNumericTime(time.Now().Add(time.Hour))
	claims.Issuer = testIssuer
	claims.Audiences = []string{testAudience}

	if mutate != nil {
		mutate(&claims)
	}

	token, err := claims.HMACSign(jwt.HS256, []byte(testSecret))
	if err != nil {
		t.Fatalf("HMACSign() error = %v", err)
	}

	return string(token)
}

func authenticateStatus(t *testing.T, app *application, authorization string) int {
	t.Helper()

	r := httptest.NewRequest(http.MethodGet, "/v1/healthcheck", nil)
	if authorization != "" {
		r.Header.Set("Authorization", authorization)
	}
	w := httptest.NewRecorder()

	app.authenticate(testHandler()).ServeHTTP(w, r)

	return w.Code
}

func TestAuthenticateRejectsBadCredentialsAndClaims(t *testing.T) {
	app := authenticateTestApp()

	tests := []struct {
		name          string
		authorization string
	}{
		{"non-bearer scheme", "Basic dXNlcjpwYXNz"},
		{"too many header parts", "Bearer one two"},
		{"garbage token", "Bearer not-a-jwt"},
		{"token signed with another secret", "Bearer " + signWithSecret(t, "another-secret-another-secret-01234", nil)},
		{"missing expiry", "Bearer " + signTestToken(t, func(c *jwt.Claims) { c.Expires = nil })},
		{"missing not-before", "Bearer " + signTestToken(t, func(c *jwt.Claims) { c.NotBefore = nil })},
		{"missing audience", "Bearer " + signTestToken(t, func(c *jwt.Claims) { c.Audiences = nil })},
		{"wrong issuer", "Bearer " + signTestToken(t, func(c *jwt.Claims) { c.Issuer = "https://evil.example" })},
		{"wrong audience", "Bearer " + signTestToken(t, func(c *jwt.Claims) { c.Audiences = []string{"https://evil.example"} })},
		{"expired", "Bearer " + signTestToken(t, func(c *jwt.Claims) {
			c.Issued = jwt.NewNumericTime(time.Now().Add(-2 * time.Hour))
			c.NotBefore = jwt.NewNumericTime(time.Now().Add(-2 * time.Hour))
			c.Expires = jwt.NewNumericTime(time.Now().Add(-time.Hour))
		})},
		{"not yet valid", "Bearer " + signTestToken(t, func(c *jwt.Claims) {
			c.NotBefore = jwt.NewNumericTime(time.Now().Add(time.Hour))
			c.Expires = jwt.NewNumericTime(time.Now().Add(2 * time.Hour))
		})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if code := authenticateStatus(t, app, tt.authorization); code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", code, http.StatusUnauthorized)
			}
		})
	}
}

func signWithSecret(t *testing.T, secret string, mutate func(*jwt.Claims)) string {
	t.Helper()

	var claims jwt.Claims
	claims.Subject = "1"
	claims.Issued = jwt.NewNumericTime(time.Now())
	claims.NotBefore = jwt.NewNumericTime(time.Now())
	claims.Expires = jwt.NewNumericTime(time.Now().Add(time.Hour))
	claims.Issuer = testIssuer
	claims.Audiences = []string{testAudience}

	if mutate != nil {
		mutate(&claims)
	}

	token, err := claims.HMACSign(jwt.HS256, []byte(secret))
	if err != nil {
		t.Fatalf("HMACSign() error = %v", err)
	}

	return string(token)
}

func TestAuthenticatePassesAnonymousUserThrough(t *testing.T) {
	app := authenticateTestApp()

	var seenAnonymous bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAnonymous = app.contextGetUser(r).IsAnonymous()
		w.WriteHeader(http.StatusOK)
	})

	r := httptest.NewRequest(http.MethodGet, "/v1/healthcheck", nil)
	w := httptest.NewRecorder()
	app.authenticate(handler).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !seenAnonymous {
		t.Error("handler should see the anonymous user when no Authorization header is sent")
	}
}

func TestRequireActivatedUser(t *testing.T) {
	app := authenticateTestApp()

	handler := app.requireActivatedUser(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name string
		user *data.User
		want int
	}{
		{"anonymous", data.AnonymousUser, http.StatusUnauthorized},
		{"inactive account", &data.User{ID: 1}, http.StatusForbidden},
		{"activated account", &data.User{ID: 1, Activated: true}, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/v1/movies", nil)
			r = app.contextSetUser(r, tt.user)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, r)

			if w.Code != tt.want {
				t.Errorf("status = %d, want %d", w.Code, tt.want)
			}
		})
	}
}
