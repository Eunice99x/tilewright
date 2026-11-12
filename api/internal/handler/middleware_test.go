package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"saas-starter/api/internal/service"
)

func TestClientIP(t *testing.T) {
	t.Setenv("INTERNAL_KEY", "shared")
	req := func(headers map[string]string) *http.Request {
		r := httptest.NewRequest("POST", "/", nil)
		r.RemoteAddr = "10.0.0.1:5000"
		for k, v := range headers {
			r.Header.Set(k, v)
		}
		return r
	}
	cases := []struct {
		name    string
		headers map[string]string
		client  string
	}{
		{"nothing", nil, "10.0.0.1"},
		{"right key", map[string]string{"X-Internal-Key": "shared", "X-Client-IP": "203.0.113.5"}, "203.0.113.5"},
		{"no key", map[string]string{"X-Client-IP": "203.0.113.5"}, "10.0.0.1"},
		{"wrong key", map[string]string{"X-Internal-Key": "nope", "X-Client-IP": "203.0.113.5"}, "10.0.0.1"},
		{"junk from the web server", map[string]string{"X-Internal-Key": "shared", "X-Client-IP": "junk"}, "10.0.0.1"},
		{"forwarded is not trusted", map[string]string{"X-Forwarded-For": "203.0.113.7, 10.0.0.9"}, "10.0.0.1"},
		{"ipv6 is grouped by /64", map[string]string{"X-Internal-Key": "shared", "X-Client-IP": "2001:db8:1:2:3:4:5:6"}, "2001:db8:1:2::"},
	}
	for _, c := range cases {
		if got := clientIP(req(c.headers)); got != c.client {
			t.Errorf("%s: clientIP = %q, want %q", c.name, got, c.client)
		}
	}

	t.Setenv("INTERNAL_KEY", "")
	if got := clientIP(req(map[string]string{"X-Internal-Key": "", "X-Client-IP": "203.0.113.5"})); got != "10.0.0.1" {
		t.Errorf("unset INTERNAL_KEY: clientIP = %q", got)
	}
}

func TestWriteErrorTooManyAttempts(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, service.ErrTooManyAttempts)
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if w.Code != http.StatusTooManyRequests || body["error"] != "too_many_attempts" {
		t.Errorf("got %d %v", w.Code, body)
	}
}

func TestOAuthRoute(t *testing.T) {
	h := New(service.New(nil, service.Config{})).Routes()
	for _, c := range []struct {
		body   string
		status int
		code   string
	}{
		{`{"provider":"github","code":"c","redirect_uri":"https://app.example/auth/github/callback"}`, http.StatusServiceUnavailable, "oauth_not_configured"},
		{`{"provider":"google","code":"c","redirect_uri":"https://app.example/auth/google/callback"}`, http.StatusServiceUnavailable, "oauth_not_configured"},
		{`{"provider":"facebook","code":"c","redirect_uri":"x"}`, http.StatusBadRequest, "invalid_input"},
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("POST", "/api/auth/oauth", strings.NewReader(c.body)))
		var got struct {
			Error string `json:"error"`
		}
		json.Unmarshal(w.Body.Bytes(), &got)
		if w.Code != c.status || got.Error != c.code {
			t.Errorf("%s: got %d %q, want %d %q", c.body, w.Code, got.Error, c.status, c.code)
		}
	}
}
