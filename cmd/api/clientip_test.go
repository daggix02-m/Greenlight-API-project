package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func clientIPRequest(remoteAddr string, xffValues ...string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/v1/healthcheck", nil)
	r.RemoteAddr = remoteAddr
	for _, v := range xffValues {
		r.Header.Add("X-Forwarded-For", v)
	}
	return r
}

func TestParseTrustedProxies(t *testing.T) {
	proxies, err := parseTrustedProxies("10.0.0.0/8 192.168.1.5")
	if err != nil {
		t.Fatalf("parseTrustedProxies() error = %v", err)
	}
	if len(proxies) != 2 {
		t.Fatalf("len(proxies) = %d, want 2", len(proxies))
	}
	if !proxies[1].Contains(net.ParseIP("192.168.1.5")) {
		t.Error("second proxy should cover 192.168.1.5")
	}

	if _, err := parseTrustedProxies("10.0.0.0/8 not-an-ip"); err == nil {
		t.Error("parseTrustedProxies() should reject an invalid entry")
	}

	proxies, err = parseTrustedProxies("")
	if err != nil || len(proxies) != 0 {
		t.Errorf("parseTrustedProxies(\"\") = %v, %v; want empty, nil", proxies, err)
	}
}

func TestClientIPWithoutTrustedProxiesIgnoresForwardedHeaders(t *testing.T) {
	app := &application{}

	r := clientIPRequest("192.0.2.10:5000", "8.8.8.8", "203.0.113.1")

	if got := app.clientIP(r); got != "192.0.2.10" {
		t.Errorf("clientIP() = %q, want 192.0.2.10", got)
	}
}

func TestClientIPWithUntrustedPeerIgnoresForwardedHeaders(t *testing.T) {
	proxies, err := parseTrustedProxies("10.0.0.0/8")
	if err != nil {
		t.Fatalf("parseTrustedProxies() error = %v", err)
	}
	app := &application{}
	app.config.trustedProxies = proxies

	r := clientIPRequest("203.0.113.50:6000", "8.8.8.8")

	if got := app.clientIP(r); got != "203.0.113.50" {
		t.Errorf("clientIP() = %q, want 203.0.113.50", got)
	}
}

func TestClientIPWithTrustedProxyUsesRightmostUntrustedHop(t *testing.T) {
	proxies, err := parseTrustedProxies("10.0.0.0/8")
	if err != nil {
		t.Fatalf("parseTrustedProxies() error = %v", err)
	}
	app := &application{}
	app.config.trustedProxies = proxies

	tests := []struct {
		name       string
		remoteAddr string
		xff        []string
		want       string
	}{
		{
			"single client hop",
			"10.0.0.1:8080",
			[]string{"203.0.113.9"},
			"203.0.113.9",
		},
		{
			"client spoofed leftmost entry, proxy appended real address",
			"10.0.0.1:8080",
			[]string{"8.8.8.8, 192.0.2.7"},
			"192.0.2.7",
		},
		{
			"chained trusted proxies",
			"10.0.0.1:8080",
			[]string{"8.8.8.8, 10.0.0.9, 192.0.2.7"},
			"192.0.2.7",
		},
		{
			"multiple header values",
			"10.0.0.1:8080",
			[]string{"8.8.8.8, 192.0.2.7", "10.0.0.9"},
			"192.0.2.7",
		},
		{
			"no forwarded header",
			"10.0.0.1:8080",
			nil,
			"10.0.0.1",
		},
		{
			"garbage rightmost entry falls back to socket address",
			"10.0.0.1:8080",
			[]string{"not-an-ip"},
			"10.0.0.1",
		},
		{
			"all hops trusted falls back to socket address",
			"10.0.0.1:8080",
			[]string{"10.0.0.9"},
			"10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := clientIPRequest(tt.remoteAddr, tt.xff...)
			if got := app.clientIP(r); got != tt.want {
				t.Errorf("clientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRateLimitKeyIsNotClientControlled(t *testing.T) {
	app := &application{}
	app.config.limiter.enabled = true
	app.config.limiter.rps = 0.01
	app.config.limiter.burst = 1

	handler := app.rateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	send := func(xff string) int {
		r := httptest.NewRequest(http.MethodPost, "/v1/tokens/authentication", nil)
		r.RemoteAddr = "192.0.2.10:7000"
		if xff != "" {
			r.Header.Set("X-Forwarded-For", xff)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w.Code
	}

	if code := send("198.51.100.1"); code != http.StatusOK {
		t.Fatalf("first request status = %d, want %d", code, http.StatusOK)
	}

	if code := send("198.51.100.2"); code != http.StatusTooManyRequests {
		t.Errorf("second request with a different X-Forwarded-For status = %d, want %d — forwarded header must not reset the bucket",
			code, http.StatusTooManyRequests)
	}
}
