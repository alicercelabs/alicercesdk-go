package alicercelabs

import (
	"context"
	"net/http"
	"testing"
)

// One representative call per resource file — the request/error-handling
// machinery itself is covered exhaustively in client_test.go.

func TestCEPGet(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/cep/01310100"] = envelopeJSON(t, map[string]any{"cep": "01310100", "logradouro": "Avenida Paulista"})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.CEP.Get(context.Background(), "01310100", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Logradouro != "Avenida Paulista" {
		t.Errorf("got %+v", result)
	}
}

func TestKVPutThenGet(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["PUT /api/v1/kv/tema"] = envelopeJSON(t, map[string]any{"message": "gravado"})
	routes["GET /api/v1/kv/tema"] = envelopeJSON(t, map[string]any{"key": "tema", "value": "escuro"})

	c := New("tok", WithAPIBase(srv.URL))
	if err := c.KV.Put(context.Background(), "tema", "escuro", 0); err != nil {
		t.Fatal(err)
	}
	value, err := c.KV.Get(context.Background(), "tema")
	if err != nil {
		t.Fatal(err)
	}
	if value != "escuro" {
		t.Errorf("got value=%q", value)
	}
}

func TestQueuePullReturnsNotOKOnEmpty(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/queue/fila/pull"] = errorJSON(http.StatusNotFound, "fila vazia")

	c := New("tok", WithAPIBase(srv.URL))
	message, ok, err := c.Queue.Pull(context.Background(), "fila", 0)
	if err != nil {
		t.Fatal(err)
	}
	if ok || message != "" {
		t.Errorf("got message=%q ok=%v, want empty/false", message, ok)
	}
}

func TestCronList(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/cron/jobs"] = envelopeJSON(t, []map[string]any{{"id": "1", "name": "job"}})

	c := New("tok", WithAPIBase(srv.URL))
	jobs, err := c.Cron.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0]["name"] != "job" {
		t.Errorf("got %+v", jobs)
	}
}

func TestQRCodeGenerateReturnsBinaryResponse(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/qrcode"] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("\x89PNGfakepngbytes"))
	}

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.QRCode.Generate(context.Background(), "hello", 0)
	if err != nil {
		t.Fatal(err)
	}
	if string(result.Content) != "\x89PNGfakepngbytes" || result.ContentType != "image/png" {
		t.Errorf("got %+v", result)
	}
}

func TestImagemTransformRequiresExactlyOneSource(t *testing.T) {
	c := New("tok")
	if _, err := c.Imagem.Transform(context.Background(), nil, "", nil); err != ErrExactlyOneSource {
		t.Errorf("neither image nor url: got %v", err)
	}
	if _, err := c.Imagem.Transform(context.Background(), []byte("x"), "https://exemplo.com/a.png", nil); err != ErrExactlyOneSource {
		t.Errorf("both image and url: got %v", err)
	}
}

func TestFunctionsInvokeReturnsBinaryResponse(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/functions/echo/invoke"] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ola de volta"))
	}

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Functions.Invoke(context.Background(), "echo", []byte("ola"))
	if err != nil {
		t.Fatal(err)
	}
	if string(result.Content) != "ola de volta" {
		t.Errorf("got content=%q", result.Content)
	}
}

func TestAuthRegisterStoresTokenOnClient(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/user/register"] = envelopeJSON(t, map[string]any{
		"user":  map[string]any{"id": 1, "username": "voce"},
		"token": "new-token",
	})

	c := New("", WithAPIBase(srv.URL))
	if _, err := c.Auth.Register(context.Background(), "voce", "voce@exemplo.com", "senha1234"); err != nil {
		t.Fatal(err)
	}
	if c.APIKey != "new-token" {
		t.Errorf("got APIKey=%q, want new-token", c.APIKey)
	}
}

func TestAccountUsage(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/account/usage"] = envelopeJSON(t, []map[string]any{
		{"api": "cep", "operation": "lookup", "request_count": 42},
	})

	c := New("tok", WithAccountBase(srv.URL))
	rows, err := c.Account.Usage(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].RequestCount != 42 {
		t.Errorf("got %+v", rows)
	}
}

func TestAccountAPIKeysCreate(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/account/apikeys"] = envelopeJSON(t, map[string]any{
		"key":     "alk_raw",
		"api_key": map[string]any{"id": 1, "name": "ci"},
	})

	c := New("tok", WithAccountBase(srv.URL))
	result, err := c.Account.APIKeys.Create(context.Background(), "ci")
	if err != nil {
		t.Fatal(err)
	}
	if result.Key != "alk_raw" {
		t.Errorf("got %+v", result)
	}
}

func TestEdgeDBQuery(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/edgedb/meubanco/query"] = envelopeJSON(t, map[string]any{
		"columns": []string{"id", "nome"},
		"rows":    [][]any{{1, "Fulano"}},
	})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.EdgeDB.Query(context.Background(), "meubanco", "SELECT * FROM t", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Columns) != 2 || len(result.Rows) != 1 {
		t.Errorf("got %+v", result)
	}
}
