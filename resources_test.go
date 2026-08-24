package alicercelabs

// One test per API method — every endpoint the SDK exposes gets a route
// and a real HTTP request through httptest.Server. client_test.go covers
// the request/error-handling machinery itself (auth header, error
// mapping, retry-after) — this file is just breadth: 62 methods, 62
// checks that the right verb hits the right path and unwraps the right
// field.

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

// ---- ip ----

func TestIPLookup(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/ip/8.8.8.8"] = envelopeJSON(t, map[string]any{"ip": "8.8.8.8", "country": "US"})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.IP.Lookup(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatal(err)
	}
	if result.Country != "US" {
		t.Errorf("got %+v", result)
	}
}

func TestIPSelf(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/ip/self"] = envelopeJSON(t, map[string]any{"ip": "203.0.113.9", "country": "BR"})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.IP.Self(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.IP != "203.0.113.9" {
		t.Errorf("got %+v", result)
	}
}

// ---- cep ----

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

func TestCEPSearch(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/cep/busca"] = envelopeJSON(t, []map[string]any{{"cep": "01310100"}})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.CEP.Search(context.Background(), "SP", "São Paulo", "Avenida Paulista")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].CEP != "01310100" {
		t.Errorf("got %+v", result)
	}
}

func TestCEPCities(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/cep/cidades"] = envelopeJSON(t, []string{"São Paulo", "Campinas"})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.CEP.Cities(context.Background(), "SP")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Errorf("got %+v", result)
	}
}

func TestCEPNeighborhoods(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/cep/bairros"] = envelopeJSON(t, []string{"Bela Vista", "Jardins"})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.CEP.Neighborhoods(context.Background(), "SP", "São Paulo")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Errorf("got %+v", result)
	}
}

func TestCEPDistance(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/cep/distance/01310100/13083010"] = envelopeJSON(t, map[string]any{"distance_km": 96.4})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.CEP.Distance(context.Background(), "01310100", "13083010", false)
	if err != nil {
		t.Fatal(err)
	}
	if result.DistanceKM != 96.4 {
		t.Errorf("got %+v", result)
	}
}

func TestCEPBulk(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/cep/lote"] = envelopeJSON(t, []map[string]any{
		{"cep": "01310100", "endereco": map[string]any{"logradouro": "Avenida Paulista"}},
	})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.CEP.Bulk(context.Background(), []string{"01310100"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Endereco == nil || result[0].Endereco.Logradouro != "Avenida Paulista" {
		t.Errorf("got %+v", result)
	}
}

// ---- dns ----

func TestDNSLookup(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/dns/exemplo.com"] = envelopeJSON(t, map[string]any{"domain": "exemplo.com"})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.DNS.Lookup(context.Background(), "exemplo.com")
	if err != nil {
		t.Fatal(err)
	}
	if result.Domain != "exemplo.com" {
		t.Errorf("got %+v", result)
	}
}

// ---- email ----

func TestEmailVerify(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/email/verify"] = envelopeJSON(t, map[string]any{"email": "gente@exemplo.com", "valid": true})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Email.Verify(context.Background(), "gente@exemplo.com")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Valid {
		t.Errorf("got %+v", result)
	}
}

// ---- ssl ----

func TestSSLCheck(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/ssl/exemplo.com"] = envelopeJSON(t, map[string]any{"domain": "exemplo.com", "is_valid": true})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.SSL.Check(context.Background(), "exemplo.com")
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsValid {
		t.Errorf("got %+v", result)
	}
}

// ---- maps ----

func TestMapsGeocode(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/maps/geocode"] = envelopeJSON(t, map[string]any{"address": "Avenida Paulista, 1000", "lat": -23.56, "lon": -46.65})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Maps.Geocode(context.Background(), "Avenida Paulista, 1000")
	if err != nil {
		t.Fatal(err)
	}
	if result.Lat != -23.56 {
		t.Errorf("got %+v", result)
	}
}

func TestMapsReverse(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/maps/reverse"] = envelopeJSON(t, map[string]any{"address": "Avenida Paulista, 1000", "lat": -23.56, "lon": -46.65})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Maps.Reverse(context.Background(), -23.56, -46.65)
	if err != nil {
		t.Fatal(err)
	}
	if result.Address != "Avenida Paulista, 1000" {
		t.Errorf("got %+v", result)
	}
}

func TestMapsRoute(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/maps/route"] = envelopeJSON(t, map[string]any{"distance_km": 12.3, "duration_min": 25.0})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Maps.Route(context.Background(), "-23.56,-46.65", "-23.5,-46.6")
	if err != nil {
		t.Fatal(err)
	}
	if result.DurationMin != 25 {
		t.Errorf("got %+v", result)
	}
}

// ---- trust ----

func TestTrustCheckPassesCNPJQueryParam(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	var gotQuery string
	routes["GET /api/v1/trust/exemplo.com"] = func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		envelopeJSON(t, map[string]any{"score": 90})(w, r)
	}

	c := New("tok", WithAPIBase(srv.URL))
	if _, err := c.Trust.Check(context.Background(), "exemplo.com", "00000000000000"); err != nil {
		t.Fatal(err)
	}
	if gotQuery != "cnpj=00000000000000" {
		t.Errorf("got query=%q", gotQuery)
	}
}

// ---- kv ----

func TestKVList(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/kv"] = envelopeJSON(t, map[string]any{"keys": []string{"tema"}, "next_cursor": 0, "total": 1})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.KV.List(context.Background(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Keys) != 1 {
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

func TestKVDelete(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["DELETE /api/v1/kv/tema"] = envelopeJSON(t, map[string]any{})

	c := New("tok", WithAPIBase(srv.URL))
	if err := c.KV.Delete(context.Background(), "tema"); err != nil {
		t.Fatal(err)
	}
}

// ---- queue ----

func TestQueuePush(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/queue/fila/push"] = envelopeJSON(t, map[string]any{})

	c := New("tok", WithAPIBase(srv.URL))
	if err := c.Queue.Push(context.Background(), "fila", "pedido-123"); err != nil {
		t.Fatal(err)
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

func TestQueueStats(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/queue/fila/stats"] = envelopeJSON(t, map[string]any{"name": "fila", "depth": 3})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Queue.Stats(context.Background(), "fila")
	if err != nil {
		t.Fatal(err)
	}
	if result.Depth != 3 {
		t.Errorf("got %+v", result)
	}
}

func TestQueueDelete(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["DELETE /api/v1/queue/fila"] = envelopeJSON(t, map[string]any{})

	c := New("tok", WithAPIBase(srv.URL))
	if err := c.Queue.Delete(context.Background(), "fila"); err != nil {
		t.Fatal(err)
	}
}

// ---- edgedb ----

func TestEdgeDBList(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/edgedb"] = envelopeJSON(t, map[string]any{"databases": []map[string]any{{"name": "meubanco", "size_bytes": 4096}}})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.EdgeDB.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Name != "meubanco" {
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

func TestEdgeDBDelete(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["DELETE /api/v1/edgedb/meubanco"] = envelopeJSON(t, map[string]any{})

	c := New("tok", WithAPIBase(srv.URL))
	if err := c.EdgeDB.Delete(context.Background(), "meubanco"); err != nil {
		t.Fatal(err)
	}
}

// ---- cron ----

func TestCronCreate(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/cron/jobs"] = envelopeJSON(t, map[string]any{"id": "1", "name": "job"})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Cron.Create(context.Background(), CronJob{"name": "job", "schedule": "@daily"})
	if err != nil {
		t.Fatal(err)
	}
	if result["id"] != "1" {
		t.Errorf("got %+v", result)
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

func TestCronGet(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/cron/jobs/1"] = envelopeJSON(t, map[string]any{"id": "1", "name": "job"})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Cron.Get(context.Background(), "1")
	if err != nil {
		t.Fatal(err)
	}
	if result["id"] != "1" {
		t.Errorf("got %+v", result)
	}
}

func TestCronUpdate(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["PUT /api/v1/cron/jobs/1"] = envelopeJSON(t, map[string]any{"id": "1", "name": "job-renomeado"})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Cron.Update(context.Background(), "1", CronJob{"name": "job-renomeado"})
	if err != nil {
		t.Fatal(err)
	}
	if result["name"] != "job-renomeado" {
		t.Errorf("got %+v", result)
	}
}

func TestCronDelete(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["DELETE /api/v1/cron/jobs/1"] = envelopeJSON(t, map[string]any{})

	c := New("tok", WithAPIBase(srv.URL))
	if err := c.Cron.Delete(context.Background(), "1"); err != nil {
		t.Fatal(err)
	}
}

func TestCronTrigger(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/cron/jobs/1/trigger"] = envelopeJSON(t, map[string]any{})

	c := New("tok", WithAPIBase(srv.URL))
	if err := c.Cron.Trigger(context.Background(), "1"); err != nil {
		t.Fatal(err)
	}
}

func TestCronWorkerStatus(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/cron/worker/status"] = envelopeJSON(t, map[string]any{"running": true})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Cron.WorkerStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result["running"] != true {
		t.Errorf("got %+v", result)
	}
}

func TestCronWorkerStart(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/cron/worker/start"] = envelopeJSON(t, map[string]any{})

	c := New("tok", WithAPIBase(srv.URL))
	if err := c.Cron.WorkerStart(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestCronWorkerStop(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/cron/worker/stop"] = envelopeJSON(t, map[string]any{})

	c := New("tok", WithAPIBase(srv.URL))
	if err := c.Cron.WorkerStop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// ---- uptime ----

func TestUpTimeCreate(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/uptime/monitors"] = envelopeJSON(t, map[string]any{"id": "1", "url": "https://exemplo.com"})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.UpTime.Create(context.Background(), "https://exemplo.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result["id"] != "1" {
		t.Errorf("got %+v", result)
	}
}

func TestUpTimeList(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/uptime/monitors"] = envelopeJSON(t, []map[string]any{{"id": "1", "url": "https://exemplo.com"}})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.UpTime.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Errorf("got %+v", result)
	}
}

func TestUpTimeGet(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/uptime/monitors/1"] = envelopeJSON(t, map[string]any{"id": "1", "url": "https://exemplo.com"})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.UpTime.Get(context.Background(), "1")
	if err != nil {
		t.Fatal(err)
	}
	if result["url"] != "https://exemplo.com" {
		t.Errorf("got %+v", result)
	}
}

func TestUpTimeUpdate(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["PUT /api/v1/uptime/monitors/1"] = envelopeJSON(t, map[string]any{"id": "1", "interval_sec": 60})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.UpTime.Update(context.Background(), "1", UpTimeMonitor{"interval_sec": 60})
	if err != nil {
		t.Fatal(err)
	}
	if result["interval_sec"] != float64(60) {
		t.Errorf("got %+v", result)
	}
}

func TestUpTimeDelete(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["DELETE /api/v1/uptime/monitors/1"] = envelopeJSON(t, map[string]any{})

	c := New("tok", WithAPIBase(srv.URL))
	if err := c.UpTime.Delete(context.Background(), "1"); err != nil {
		t.Fatal(err)
	}
}

func TestUpTimeChecks(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/uptime/monitors/1/checks"] = envelopeJSON(t, []map[string]any{{"status": 200}})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.UpTime.Checks(context.Background(), "1")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Errorf("got %+v", result)
	}
}

func TestUpTimeWorkerStatus(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/uptime/worker/status"] = envelopeJSON(t, map[string]any{"running": true})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.UpTime.WorkerStatus(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result["running"] != true {
		t.Errorf("got %+v", result)
	}
}

func TestUpTimeWorkerStart(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/uptime/worker/start"] = envelopeJSON(t, map[string]any{})

	c := New("tok", WithAPIBase(srv.URL))
	if err := c.UpTime.WorkerStart(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestUpTimeWorkerStop(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/uptime/worker/stop"] = envelopeJSON(t, map[string]any{})

	c := New("tok", WithAPIBase(srv.URL))
	if err := c.UpTime.WorkerStop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// ---- media ----

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

func TestImagemTransform(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/imagem/transform"] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fakejpegbytes"))
	}

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Imagem.Transform(context.Background(), nil, "https://exemplo.com/foto.jpg", url.Values{"resize": {"800x600"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.ContentType != "image/jpeg" {
		t.Errorf("got %+v", result)
	}
}

func TestImagemAnalyze(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/imagem/analyze"] = envelopeJSON(t, map[string]any{
		"width": 800, "height": 600, "format": "jpeg", "dominant_color": "#336699", "palette": []string{}, "blurhash": "abc",
	})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Imagem.Analyze(context.Background(), nil, "https://exemplo.com/foto.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if result.Width != 800 {
		t.Errorf("got %+v", result)
	}
}

func TestTemplatingInvoice(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/templating/invoice"] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("%PDF-fake"))
	}

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Templating.Invoice(context.Background(), InvoiceRequest{
		Issuer:    InvoiceParty{Name: "Minha Empresa"},
		Recipient: InvoiceParty{Name: "Cliente Exemplo"},
		Items:     []InvoiceItem{{Description: "Consultoria", Quantity: 2, UnitPrice: 500}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ContentType != "application/pdf" {
		t.Errorf("got %+v", result)
	}
}

// ---- functions (compute) ----

func TestFunctionsList(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/functions"] = envelopeJSON(t, map[string]any{"functions": []map[string]any{{"name": "minha", "size_bytes": 2048}}})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Functions.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Name != "minha" {
		t.Errorf("got %+v", result)
	}
}

func TestFunctionsGet(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/functions/minha"] = envelopeJSON(t, map[string]any{"name": "minha", "size_bytes": 2048})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Functions.Get(context.Background(), "minha")
	if err != nil {
		t.Fatal(err)
	}
	if result.SizeBytes != 2048 {
		t.Errorf("got %+v", result)
	}
}

func TestFunctionsDeploy(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["PUT /api/v1/functions/minha"] = envelopeJSON(t, map[string]any{"name": "minha", "size_bytes": 4096})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Functions.Deploy(context.Background(), "minha", []byte{0, 'a', 's', 'm'})
	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "minha" {
		t.Errorf("got %+v", result)
	}
}

func TestFunctionsDelete(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["DELETE /api/v1/functions/minha"] = envelopeJSON(t, map[string]any{})

	c := New("tok", WithAPIBase(srv.URL))
	if err := c.Functions.Delete(context.Background(), "minha"); err != nil {
		t.Fatal(err)
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

// ---- auth ----

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

func TestAuthLoginStoresTokenOnClient(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/user/login"] = envelopeJSON(t, map[string]any{"token": "login-token"})

	c := New("", WithAPIBase(srv.URL))
	token, err := c.Auth.Login(context.Background(), "voce", "senha1234")
	if err != nil {
		t.Fatal(err)
	}
	if token != "login-token" || c.APIKey != "login-token" {
		t.Errorf("got token=%q APIKey=%q", token, c.APIKey)
	}
}

func TestAuthMe(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/user/me"] = envelopeJSON(t, map[string]any{"id": 1, "username": "voce", "email": "voce@exemplo.com", "role": "user"})

	c := New("tok", WithAPIBase(srv.URL))
	result, err := c.Auth.Me(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Username != "voce" {
		t.Errorf("got %+v", result)
	}
}

func TestAuthDeleteAccount(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["DELETE /api/v1/user/me"] = envelopeJSON(t, map[string]any{})

	c := New("tok", WithAPIBase(srv.URL))
	if err := c.Auth.DeleteAccount(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// ---- account ----

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

func TestAccountChangePassword(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["POST /api/v1/account/password"] = envelopeJSON(t, map[string]any{})

	c := New("tok", WithAccountBase(srv.URL))
	if err := c.Account.ChangePassword(context.Background(), "senha-antiga", "senha-nova"); err != nil {
		t.Fatal(err)
	}
}

func TestAccountAPIKeysList(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["GET /api/v1/account/apikeys"] = envelopeJSON(t, []map[string]any{{"id": 1, "name": "ci", "active": true}})

	c := New("tok", WithAccountBase(srv.URL))
	result, err := c.Account.APIKeys.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 || result[0].Name != "ci" {
		t.Errorf("got %+v", result)
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

func TestAccountAPIKeysRevoke(t *testing.T) {
	srv, routes, _ := newTestServer(t)
	routes["DELETE /api/v1/account/apikeys/1"] = envelopeJSON(t, map[string]any{})

	c := New("tok", WithAccountBase(srv.URL))
	if err := c.Account.APIKeys.Revoke(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
}
