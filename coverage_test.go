package alicercelabs

// Branches the breadth tests in resources_test.go don't exercise on their
// own: error formatting, IsRateLimit, malformed error bodies, client
// options, and the optional-query-parameter paths (opts=nil is the common
// case covered elsewhere; this file covers opts set).

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestAPIErrorString(t *testing.T) {
	err := &APIError{StatusCode: 404, Message: "não encontrado"}
	if got := err.Error(); got != "alicercelabs: 404 não encontrado" {
		t.Errorf("got %q", got)
	}
}

func TestIsRateLimit(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/ip/8.8.8.8"] = errorJSON(http.StatusTooManyRequests, "rate limit exceeded")

	c := New("tok", WithAPIBase(srv.URL))
	_, err := c.IP.Lookup(context.Background(), "8.8.8.8", nil)
	if !IsRateLimit(err) {
		t.Errorf("expected IsRateLimit, got %v", err)
	}
}

func TestStatusIsFalseForNonAPIError(t *testing.T) {
	if IsNotFound(errors.New("boring error")) {
		t.Error("expected false for a non-*APIError")
	}
	if IsNotFound(nil) {
		t.Error("expected false for nil")
	}
}

func TestErrorFromMalformedBody(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/ip/8.8.8.8"] = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("not json"))
	}

	c := New("tok", WithAPIBase(srv.URL))
	_, err := c.IP.Lookup(context.Background(), "8.8.8.8", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("expected *APIError")
	}
	if apiErr.StatusCode != 500 {
		t.Errorf("got status %d", apiErr.StatusCode)
	}
}

func TestWithHTTPClientAndTimeout(t *testing.T) {
	c := New("tok", WithTimeout(5*time.Second))
	if c.HTTPClient.Timeout != 5*time.Second {
		t.Errorf("got timeout %v", c.HTTPClient.Timeout)
	}

	custom := &http.Client{Timeout: 2 * time.Second}
	c2 := New("tok", WithHTTPClient(custom))
	if c2.HTTPClient != custom {
		t.Error("WithHTTPClient did not replace the client")
	}
}

func TestCEPGetWithOptions(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	var gotQuery string
	routes["GET /api/v1/cep/01310100"] = func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		envelopeJSON(t, map[string]any{"cep": "01310100"})(w, r)
	}

	c := New("tok", WithAPIBase(srv.URL))
	if _, err := c.CEP.Get(context.Background(), "01310100", &CEPGetOptions{DDD: true}); err != nil {
		t.Fatal(err)
	}
	if gotQuery != "ddd=true" {
		t.Errorf("got query=%q", gotQuery)
	}
}

func TestCEPDistanceWithRota(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	var gotQuery string
	routes["GET /api/v1/cep/distance/01310100/13083010"] = func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		envelopeJSON(t, map[string]any{"distance_km": 96.4, "duration_min": 80.0})(w, r)
	}

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.CEP.Distance(context.Background(), "01310100", "13083010", true)
	if err != nil {
		t.Fatal(err)
	}
	if gotQuery != "rota=true" || result.DurationMin != 80 {
		t.Errorf("got query=%q result=%+v", gotQuery, result)
	}
}

func TestKVListWithCursorAndCount(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	var gotQuery string
	routes["GET /api/v1/kv"] = func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		envelopeJSON(t, map[string]any{"keys": []string{}, "next_cursor": 20, "total": 0})(w, r)
	}

	c := New("tok", WithAPIBase(srv.URL))
	if _, err := c.KV.List(context.Background(), 10, 20); err != nil {
		t.Fatal(err)
	}
	if gotQuery != "count=20&cursor=10" {
		t.Errorf("got query=%q", gotQuery)
	}
}

func TestQueuePullWithWait(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	var gotQuery string
	routes["POST /api/v1/queue/fila/pull"] = func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		envelopeJSON(t, map[string]any{"message": "oi"})(w, r)
	}

	c := New("tok", WithAPIBase(srv.URL))
	message, ok, err := c.Queue.Pull(context.Background(), "fila", 3)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || message != "oi" || gotQuery != "wait=3" {
		t.Errorf("got message=%q ok=%v query=%q", message, ok, gotQuery)
	}
}

func TestQueuePullPropagatesNonNotFoundError(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/queue/fila/pull"] = errorJSON(http.StatusServiceUnavailable, "fila indisponível")

	c := New("tok", WithAPIBase(srv.URL))
	_, _, err := c.Queue.Pull(context.Background(), "fila", 0)
	if !IsServiceUnavailable(err) {
		t.Errorf("expected 503, got %v", err)
	}
}

func TestAuthLoginErrorPath(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/user/login"] = errorJSON(http.StatusUnauthorized, "credenciais inválidas")

	c := New("", WithAPIBase(srv.URL))
	_, err := c.Auth.Login(context.Background(), "voce", "senha-errada")
	if !IsAuthenticationError(err) {
		t.Errorf("expected 401, got %v", err)
	}
}

func TestBinaryResponseSave(t *testing.T) {
	b := &BinaryResponse{Content: []byte("conteudo"), ContentType: "text/plain"}
	path := t.TempDir() + "/arquivo.txt"
	if err := b.Save(path); err != nil {
		t.Fatal(err)
	}
}
