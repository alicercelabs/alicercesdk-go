package alicercelabs

// Round two of coverage: the branches client_test.go/coverage_test.go/
// resources_test.go/errorpaths_test.go don't reach — every doJSON/doRaw
// failure mode (bad request, transport failure, malformed envelope,
// empty data, type-mismatched data), every optional-field branch that
// only fires when the field is actually set, and every jsonBody call
// site that can genuinely fail (map[string]any/[]any parameters, where a
// caller can hand in something unmarshalable — plain string/int
// parameters can't, so those call sites aren't chased here).

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// ---- jsonBody itself ----

func TestJSONBodyMarshalError(t *testing.T) {
	if _, err := jsonBody(map[string]any{"bad": make(chan int)}); err == nil {
		t.Fatal("expected an error marshaling an unsupported type")
	}
}

// ---- doJSON failure modes ----

func TestDoJSON_RequestBuildError(t *testing.T) {
	// A control character in the base URL makes http.NewRequestWithContext
	// itself fail (url.Parse rejects it), before any network call happens.
	c := New("tok", WithAPIBase("http://example.com\n"))
	_, err := c.IP.Self(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "build request") {
		t.Errorf("got %v, want a build-request error", err)
	}
}

func TestDoJSON_TransportError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	srv, _, _ := newTestServer(t)
	c := New("tok", WithAPIBase(srv.URL))
	_, err := c.IP.Self(ctx, nil)
	if err == nil || !strings.Contains(err.Error(), "request failed") {
		t.Errorf("got %v, want a request-failed error", err)
	}
}

func TestDoJSON_MalformedEnvelope(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/ip/self"] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{not valid json`))
	}

	c := New("tok", WithAPIBase(srv.URL))
	_, err := c.IP.Self(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "decode response envelope") {
		t.Errorf("got %v, want a decode-envelope error", err)
	}
}

func TestDoJSON_NoDataField(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/ip/self"] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"meta":{"request_id":"req_test"}}`))
	}

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.IP.Self(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IP != "" {
		t.Errorf("got %+v, want the zero value (no data field to decode)", result)
	}
}

func TestDoJSON_DataTypeMismatch(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/ip/self"] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"data":"not-an-object"}`))
	}

	c := New("tok", WithAPIBase(srv.URL))
	_, err := c.IP.Self(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "decode response data") {
		t.Errorf("got %v, want a decode-data error", err)
	}
}

func TestDoJSON_ReadBodyError(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/ip/self"] = truncatedBody(t)

	c := New("tok", WithAPIBase(srv.URL))
	_, err := c.IP.Self(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "read response body") {
		t.Errorf("got %v, want a read-response-body error", err)
	}
}

// ---- doRaw failure modes (same rawDo machinery as doJSON, different
// caller) ----

func TestDoRaw_RequestBuildError(t *testing.T) {
	c := New("tok", WithAPIBase("http://example.com\n"))
	_, err := c.QRCode.Generate(context.Background(), "hello", 0)
	if err == nil || !strings.Contains(err.Error(), "build request") {
		t.Errorf("got %v, want a build-request error", err)
	}
}

func TestDoRaw_TransportError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	srv, _, _ := newTestServer(t)
	c := New("tok", WithAPIBase(srv.URL))
	_, err := c.QRCode.Generate(ctx, "hello", 0)
	if err == nil || !strings.Contains(err.Error(), "request failed") {
		t.Errorf("got %v, want a request-failed error", err)
	}
}

func TestDoRaw_ReadBodyError(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/qrcode"] = truncatedBody(t)

	c := New("tok", WithAPIBase(srv.URL))
	_, err := c.QRCode.Generate(context.Background(), "hello", 0)
	if err == nil || !strings.Contains(err.Error(), "read response body") {
		t.Errorf("got %v, want a read-response-body error", err)
	}
}

// truncatedBody answers with a Content-Length larger than what's actually
// written — the standard library's io.ReadAll(resp.Body) then reliably
// fails with io.ErrUnexpectedEOF, the one way to exercise doJSON/doRaw's
// "read response body" branch without a custom broken io.Reader.
func truncatedBody(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("short"))
	}
}

// ---- apiErrorFromBody's request-id passthrough ----

func TestAPIErrorFromBody_RequestID(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/ip/self"] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		body, _ := json.Marshal(map[string]any{"success": false, "error": "deu ruim", "meta": map[string]any{"request_id": "req_abc"}})
		w.Write(body)
	}

	c := New("tok", WithAPIBase(srv.URL))
	_, err := c.IP.Self(context.Background(), nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %v", err)
	}
	if apiErr.RequestID != "req_abc" {
		t.Errorf("got RequestID=%q", apiErr.RequestID)
	}
}

// ---- IPLookupOptions.query() ----

func TestIPLookupOptionsQuery(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	var gotQuery url.Values
	routes["GET /api/v1/ip/8.8.8.8"] = func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		envelopeJSON(t, map[string]any{"ip": "8.8.8.8"})(w, r)
	}

	c := New("tok", WithAPIBase(srv.URL))
	opts := &IPLookupOptions{Fields: []string{"location.country", "network.asn"}, IncludeSourceDetails: true, Lang: "pt-BR"}
	if _, err := c.IP.Lookup(context.Background(), "8.8.8.8", opts); err != nil {
		t.Fatal(err)
	}
	if got := gotQuery.Get("fields"); got != "location.country,network.asn" {
		t.Errorf("fields=%q", got)
	}
	if got := gotQuery.Get("include"); got != "source_details" {
		t.Errorf("include=%q", got)
	}
	if got := gotQuery.Get("lang"); got != "pt-BR" {
		t.Errorf("lang=%q", got)
	}
}

func TestIPBatch_ErrorPropagates(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/ip/batch"] = errorJSON(http.StatusInternalServerError, "erro interno")

	c := New("tok", WithAPIBase(srv.URL))
	_, err := c.IP.Batch(context.Background(), []string{"8.8.8.8"}, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %v", err)
	}
}

// ---- Auth.Login's own error paths (it doesn't go through doJSON) ----

func TestAuthLogin_RequestBuildError(t *testing.T) {
	c := New("", WithAPIBase("http://example.com\n"))
	_, err := c.Auth.Login(context.Background(), "u", "p")
	if err == nil || !strings.Contains(err.Error(), "build request") {
		t.Errorf("got %v, want a build-request error", err)
	}
}

func TestAuthLogin_TransportError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	srv, _, _ := newTestServer(t)
	c := New("", WithAPIBase(srv.URL))
	_, err := c.Auth.Login(ctx, "u", "p")
	if err == nil || !strings.Contains(err.Error(), "request failed") {
		t.Errorf("got %v, want a request-failed error", err)
	}
}

func TestAuthLogin_ReadBodyError(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/user/login"] = truncatedBody(t)

	c := New("", WithAPIBase(srv.URL))
	_, err := c.Auth.Login(context.Background(), "u", "p")
	if err == nil || !strings.Contains(err.Error(), "read response body") {
		t.Errorf("got %v, want a read-response-body error", err)
	}
}

func TestAuthLogin_MalformedResponse(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/user/login"] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{not valid json`))
	}

	c := New("", WithAPIBase(srv.URL))
	_, err := c.Auth.Login(context.Background(), "u", "p")
	if err == nil || !strings.Contains(err.Error(), "decode response") {
		t.Errorf("got %v, want a decode error", err)
	}
}

// ---- Cron/UpTime/EdgeDB: jsonBody call sites that take a
// map[string]any/[]any parameter a caller can genuinely break ----

func TestCronCreate_MarshalError(t *testing.T) {
	c := New("tok", WithAPIBase("http://unused.invalid"))
	if _, err := c.Cron.Create(context.Background(), CronJob{"bad": make(chan int)}); err == nil {
		t.Fatal("expected a marshal error")
	}
}

func TestCronUpdate_MarshalError(t *testing.T) {
	c := New("tok", WithAPIBase("http://unused.invalid"))
	if _, err := c.Cron.Update(context.Background(), "1", CronJob{"bad": make(chan int)}); err == nil {
		t.Fatal("expected a marshal error")
	}
}

func TestUpTimeCreate_MergesFieldsAndDetectsMarshalError(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	var gotBody map[string]any
	routes["POST /api/v1/uptime/monitors"] = func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		envelopeJSON(t, map[string]any{})(w, r)
	}

	c := New("tok", WithAPIBase(srv.URL))
	if _, err := c.UpTime.Create(context.Background(), "https://a.com", UpTimeMonitor{"interval_sec": float64(60)}); err != nil {
		t.Fatal(err)
	}
	if gotBody["url"] != "https://a.com" || gotBody["interval_sec"] != float64(60) {
		t.Errorf("got body=%+v", gotBody)
	}

	badClient := New("tok", WithAPIBase("http://unused.invalid"))
	if _, err := badClient.UpTime.Create(context.Background(), "https://a.com", UpTimeMonitor{"bad": make(chan int)}); err == nil {
		t.Fatal("expected a marshal error")
	}
}

func TestUpTimeUpdate_MarshalError(t *testing.T) {
	c := New("tok", WithAPIBase("http://unused.invalid"))
	if _, err := c.UpTime.Update(context.Background(), "1", UpTimeMonitor{"bad": make(chan int)}); err == nil {
		t.Fatal("expected a marshal error")
	}
}

func TestEdgeDBQuery_MarshalError(t *testing.T) {
	c := New("tok", WithAPIBase("http://unused.invalid"))
	if _, err := c.EdgeDB.Query(context.Background(), "db", "SELECT ?", []any{make(chan int)}); err == nil {
		t.Fatal("expected a marshal error")
	}
}

// ---- QRCode: optional-parameter branches ----

func TestQRCodeGenerate_WithSize(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	var gotQuery string
	routes["GET /api/v1/qrcode"] = func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("png-bytes"))
	}

	c := New("tok", WithAPIBase(srv.URL))
	if _, err := c.QRCode.Generate(context.Background(), "hello", 256); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, "size=256") {
		t.Errorf("got query=%q", gotQuery)
	}
}

func TestQRCodePix_AllOptionalFields(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	var gotQuery string
	routes["GET /api/v1/qrcode/pix"] = func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("X-Pix-Copia-Cola", "00020126...")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("png-bytes"))
	}

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.QRCode.Pix(context.Background(), PixParams{
		Chave: "11999999999", Nome: "Fulano", Cidade: "Sao Paulo",
		Valor: 10.5, TxID: "TX123", Descricao: "pedido 1", Size: 512,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"valor=10.50", "txid=TX123", "descricao=pedido+1", "size=512"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
	if result.CopiaCola != "00020126..." {
		t.Errorf("got CopiaCola=%q", result.CopiaCola)
	}
}

// ---- Imagem: the exactly-one-source guard, and the image-bytes (not
// url) path on both Transform and Analyze ----

func TestImagemTransform_ExactlyOneSourceGuard(t *testing.T) {
	c := New("tok", WithAPIBase("http://unused.invalid"))
	if _, err := c.Imagem.Transform(context.Background(), nil, "", nil); !errors.Is(err, ErrExactlyOneSource) {
		t.Errorf("neither source: got %v", err)
	}
	if _, err := c.Imagem.Transform(context.Background(), []byte("x"), "https://a.com/x.jpg", nil); !errors.Is(err, ErrExactlyOneSource) {
		t.Errorf("both sources: got %v", err)
	}
}

func TestImagemTransform_UploadedBytes(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	var gotBody []byte
	routes["POST /api/v1/imagem/transform"] = func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("png-bytes"))
	}

	c := New("tok", WithAPIBase(srv.URL))
	if _, err := c.Imagem.Transform(context.Background(), []byte("raw-image-bytes"), "", url.Values{"resize": {"100x100"}}); err != nil {
		t.Fatal(err)
	}
	if string(gotBody) != "raw-image-bytes" {
		t.Errorf("got body=%q", gotBody)
	}
}

func TestImagemAnalyze_ExactlyOneSourceGuard(t *testing.T) {
	c := New("tok", WithAPIBase("http://unused.invalid"))
	if _, err := c.Imagem.Analyze(context.Background(), nil, ""); !errors.Is(err, ErrExactlyOneSource) {
		t.Errorf("neither source: got %v", err)
	}
	if _, err := c.Imagem.Analyze(context.Background(), []byte("x"), "https://a.com/x.jpg"); !errors.Is(err, ErrExactlyOneSource) {
		t.Errorf("both sources: got %v", err)
	}
}

func TestImagemAnalyze_UploadedBytes(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	var gotBody []byte
	routes["POST /api/v1/imagem/analyze"] = func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		envelopeJSON(t, map[string]any{"width": 100, "height": 100})(w, r)
	}

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Imagem.Analyze(context.Background(), []byte("raw-image-bytes"), "")
	if err != nil {
		t.Fatal(err)
	}
	if string(gotBody) != "raw-image-bytes" || result.Width != 100 {
		t.Errorf("got body=%q result=%+v", gotBody, result)
	}
}
