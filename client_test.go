package alicercelabs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestServer builds a real HTTP server (not a mocked RoundTripper) that
// answers routes registered via handle — same discipline as the rest of
// AlicerceLabs: test against something real.
func newTestServer(t *testing.T) (*httptest.Server, map[string]http.HandlerFunc, *http.Request) {
	t.Helper()
	routes := map[string]http.HandlerFunc{}
	var lastReq *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastReq = r
		key := r.Method + " " + r.URL.Path
		if h, ok := routes[key]; ok {
			h(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"success":false,"error":"not found"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, routes, lastReq
}

func envelopeJSON(t *testing.T, data any) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		body, _ := json.Marshal(map[string]any{"success": true, "data": data, "meta": map[string]any{"request_id": "req_test"}})
		w.Write(body)
	}
}

func errorJSON(status int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if status == http.StatusTooManyRequests {
			w.Header().Set("Retry-After", "60")
		}
		w.WriteHeader(status)
		body, _ := json.Marshal(map[string]any{"success": false, "error": message})
		w.Write(body)
	}
}

func TestSuccessfulCallUnwrapsData(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/ip/8.8.8.8"] = envelopeJSON(t, map[string]any{"ip": "8.8.8.8", "version": 4, "scope": "public", "routable": true})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.IP.Lookup(context.Background(), "8.8.8.8", nil)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if result.IP != "8.8.8.8" || result.Scope != "public" {
		t.Errorf("got %+v", result)
	}
}

func TestSendsBearerHeader(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	var gotAuth string
	routes["GET /api/v1/ip/8.8.8.8"] = func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		envelopeJSON(t, map[string]any{})(w, r)
	}

	c := New("alk_abc123", WithAPIBase(srv.URL))
	if _, err := c.IP.Lookup(context.Background(), "8.8.8.8", nil); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer alk_abc123" {
		t.Errorf("got Authorization=%q", gotAuth)
	}
}

func TestErrorStatusMapsToAPIError(t *testing.T) {
	cases := []struct {
		status int
		check  func(error) bool
	}{
		{400, IsValidationError},
		{401, IsAuthenticationError},
		{404, IsNotFound},
		{503, IsServiceUnavailable},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("status_%d", tc.status), func(t *testing.T) {
			srv, routes, _ := newTestServer(t)
			routes["GET /api/v1/ip/8.8.8.8"] = errorJSON(tc.status, "something went wrong")

			c := New("tok", WithAPIBase(srv.URL))
			_, err := c.IP.Lookup(context.Background(), "8.8.8.8", nil)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !tc.check(err) {
				t.Errorf("status %d: typed check failed for err %v", tc.status, err)
			}
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatal("expected *APIError")
			}
			if apiErr.Message != "something went wrong" {
				t.Errorf("got message %q", apiErr.Message)
			}
		})
	}
}

func TestRateLimitErrorCarriesRetryAfter(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/ip/8.8.8.8"] = errorJSON(http.StatusTooManyRequests, "rate limit exceeded")

	c := New("tok", WithAPIBase(srv.URL))
	_, err := c.IP.Lookup(context.Background(), "8.8.8.8", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected *APIError")
	}
	if apiErr.RetryAfter != 60 {
		t.Errorf("got RetryAfter=%d, want 60", apiErr.RetryAfter)
	}
}
