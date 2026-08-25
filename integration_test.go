//go:build integration

// Real integration tests: this file talks to an actual running
// AlicerceLabs instance (production by default) over the network, using
// the public SDK surface exactly as a real caller would. It's opt-in
// (build tag `integration`) because it registers and deletes a real
// throwaway account, creates and deletes real resources (KV keys, a
// queue, an Edge DB, a cron job, an uptime monitor, a deployed function,
// API keys), and makes real outbound calls (RDAP, BrasilAPI, DNS, SMTP
// probes) that a normal `go test ./...` run has no business doing.
//
//	go test -tags integration -run TestIntegration -v .
//
// Override ALICERCELABS_API_BASE / ALICERCELABS_ACCOUNT_BASE to point at
// a self-hosted instance instead of production.
//
// Deliberately NOT covered: Cron.WorkerStart/WorkerStop and
// UpTime.WorkerStart/WorkerStop. Those control a daemon shared by every
// account on the instance, not something scoped to the test account —
// stopping it here would affect real users. WorkerStatus (read-only) is
// covered instead.
package alicercelabs_test

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	alicercelabs "github.com/alicercelabs/alicercesdk-go"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// buildEchoWASM compiles a tiny WASI guest that echoes the request body
// back, so Functions.Invoke has something real to call. Skips the whole
// test (not a failure) if the `go` toolchain isn't on PATH — CI running
// this suite is expected to have it, since it's the Go SDK's own suite,
// but a plain dev machine running `go test -tags integration` without
// network shouldn't get a confusing failure over a missing compiler.
func buildEchoWASM(t *testing.T) []byte {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not on PATH, can't build the WASI test fixture")
	}

	dir := t.TempDir()
	src := `package main

import (
	"encoding/json"
	"io"
	"os"
)

type req struct {
	Body string ` + "`json:\"body\"`" + `
}
type resp struct {
	Status int    ` + "`json:\"status\"`" + `
	Body   string ` + "`json:\"body\"`" + `
}

func main() {
	data, _ := io.ReadAll(os.Stdin)
	var r req
	json.Unmarshal(data, &r)
	out, _ := json.Marshal(resp{Status: 200, Body: "eco: " + r.Body})
	os.Stdout.Write(out)
}
`
	srcPath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write guest source: %v", err)
	}

	wasmPath := filepath.Join(dir, "echo.wasm")
	cmd := exec.Command("go", "build", "-o", wasmPath, srcPath)
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build echo.wasm: %v\n%s", err, out)
	}

	data, err := os.ReadFile(wasmPath)
	if err != nil {
		t.Fatalf("read echo.wasm: %v", err)
	}
	return data
}

func TestIntegration(t *testing.T) {
	if os.Getenv("ALICERCELABS_INTEGRATION") == "" {
		t.Skip("set ALICERCELABS_INTEGRATION=1 to run against a real AlicerceLabs instance")
	}

	apiBase := envOr("ALICERCELABS_API_BASE", alicercelabs.DefaultAPIBase)
	accountBase := envOr("ALICERCELABS_ACCOUNT_BASE", alicercelabs.DefaultAccountBase)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	suffix := fmt.Sprintf("%d%03d", time.Now().Unix(), rand.Intn(1000))
	username := "sdk-go-it-" + suffix
	email := username + "@mailinator.com"
	password := "Senha-Forte-" + suffix + "!"

	client := alicercelabs.New("", alicercelabs.WithAPIBase(apiBase), alicercelabs.WithAccountBase(accountBase))

	if _, err := client.Auth.Register(ctx, username, email, password); err != nil {
		t.Fatalf("Auth.Register: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Auth.DeleteAccount(context.Background()); err != nil {
			t.Logf("cleanup: Auth.DeleteAccount: %v", err)
		}
	})

	t.Run("Auth", func(t *testing.T) {
		me, err := client.Auth.Me(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if me.Username != username {
			t.Errorf("got username=%q, want %q", me.Username, username)
		}
	})

	t.Run("IP", func(t *testing.T) {
		if _, err := client.IP.Lookup(ctx, "8.8.8.8", nil); err != nil {
			t.Errorf("Lookup: %v", err)
		}
		if _, err := client.IP.Self(ctx, nil); err != nil {
			t.Errorf("Self: %v", err)
		}
	})

	t.Run("CEP", func(t *testing.T) {
		endereco, err := client.CEP.Get(ctx, "01310100", nil)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		// Municipio (not "cidade") is the field the API actually returns —
		// asserting on it here is what caught the wrong field name before
		// this test suite existed.
		if endereco.Municipio != "São Paulo" {
			t.Errorf("got Municipio=%q, want São Paulo", endereco.Municipio)
		}
		if _, err := client.CEP.Search(ctx, "SP", "São Paulo", "Avenida Paulista"); err != nil {
			t.Errorf("Search: %v", err)
		}
		if _, err := client.CEP.Cities(ctx, "SP"); err != nil {
			t.Errorf("Cities: %v", err)
		}
		if _, err := client.CEP.Neighborhoods(ctx, "SP", "São Paulo"); err != nil {
			t.Errorf("Neighborhoods: %v", err)
		}
		// Same pair as the CEP docs page's own ?rota=true example — known
		// to geocode on both ends.
		if _, err := client.CEP.Distance(ctx, "01310100", "20040020", false); err != nil {
			t.Errorf("Distance: %v", err)
		}
		bulk, err := client.CEP.Bulk(ctx, []string{"01310100"})
		if err != nil {
			t.Fatalf("Bulk: %v", err)
		}
		if len(bulk) != 1 || bulk[0].Endereco == nil || bulk[0].Endereco.Municipio != "São Paulo" {
			t.Errorf("got %+v", bulk)
		}
	})

	t.Run("DNS", func(t *testing.T) {
		if _, err := client.DNS.Lookup(ctx, "alicercelabs.com.br"); err != nil {
			t.Errorf("Lookup: %v", err)
		}
	})

	t.Run("Email", func(t *testing.T) {
		if _, err := client.Email.Verify(ctx, email); err != nil {
			t.Errorf("Verify: %v", err)
		}
	})

	t.Run("SSL", func(t *testing.T) {
		if _, err := client.SSL.Check(ctx, "alicercelabs.com.br"); err != nil {
			t.Errorf("Check: %v", err)
		}
	})

	t.Run("Maps", func(t *testing.T) {
		if _, err := client.Maps.Geocode(ctx, "Avenida Paulista, 1000, São Paulo"); err != nil {
			t.Errorf("Geocode: %v", err)
		}
		if _, err := client.Maps.Reverse(ctx, -23.5613, -46.6558); err != nil {
			t.Errorf("Reverse: %v", err)
		}
		if _, err := client.Maps.Route(ctx, "-23.5613,-46.6558", "-23.5505,-46.6333"); err != nil {
			t.Errorf("Route: %v", err)
		}
	})

	t.Run("Trust", func(t *testing.T) {
		if _, err := client.Trust.Check(ctx, "alicercelabs.com.br", ""); err != nil {
			t.Errorf("Check: %v", err)
		}
	})

	t.Run("KV", func(t *testing.T) {
		key := "sdk-integration-" + suffix
		if err := client.KV.Put(ctx, key, "valor-de-teste", 300); err != nil {
			t.Fatalf("Put: %v", err)
		}
		defer client.KV.Delete(context.Background(), key)

		value, err := client.KV.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if value != "valor-de-teste" {
			t.Errorf("got value=%q", value)
		}
		if _, err := client.KV.List(ctx, 0, 50); err != nil {
			t.Errorf("List: %v", err)
		}
	})

	t.Run("Queue", func(t *testing.T) {
		name := "sdk-integration-" + suffix
		defer client.Queue.Delete(context.Background(), name)

		if err := client.Queue.Push(ctx, name, "mensagem de teste"); err != nil {
			t.Fatalf("Push: %v", err)
		}
		if _, err := client.Queue.Stats(ctx, name); err != nil {
			t.Errorf("Stats: %v", err)
		}
		message, ok, err := client.Queue.Pull(ctx, name, 0)
		if err != nil {
			t.Fatalf("Pull: %v", err)
		}
		if !ok || message != "mensagem de teste" {
			t.Errorf("got message=%q ok=%v", message, ok)
		}
	})

	t.Run("EdgeDB", func(t *testing.T) {
		name := "sdk-integration-" + suffix
		defer client.EdgeDB.Delete(context.Background(), name)

		if _, err := client.EdgeDB.Query(ctx, name, "CREATE TABLE t (id INTEGER PRIMARY KEY, nome TEXT)", nil); err != nil {
			t.Fatalf("Query (create): %v", err)
		}
		if _, err := client.EdgeDB.Query(ctx, name, "INSERT INTO t (nome) VALUES (?)", []any{"Fulano"}); err != nil {
			t.Fatalf("Query (insert): %v", err)
		}
		result, err := client.EdgeDB.Query(ctx, name, "SELECT * FROM t", nil)
		if err != nil {
			t.Fatalf("Query (select): %v", err)
		}
		if len(result.Rows) != 1 {
			t.Errorf("got %d rows, want 1", len(result.Rows))
		}
		if _, err := client.EdgeDB.List(ctx); err != nil {
			t.Errorf("List: %v", err)
		}
	})

	t.Run("Cron", func(t *testing.T) {
		job, err := client.Cron.Create(ctx, alicercelabs.CronJob{
			"name": "sdk-integration-" + suffix, "schedule": "0 0 1 1 *",
			"image_type": "image", "image_source": "alpine:latest", "command": "echo oi",
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		id := fmt.Sprintf("%v", job["id"])
		defer client.Cron.Delete(context.Background(), id)

		if _, err := client.Cron.List(ctx); err != nil {
			t.Errorf("List: %v", err)
		}
		if _, err := client.Cron.Get(ctx, id); err != nil {
			t.Errorf("Get: %v", err)
		}
		// Update is PUT semantics (whole-resource replace), not a partial
		// patch — every required field has to be present, not just the one
		// changing.
		if _, err := client.Cron.Update(ctx, id, alicercelabs.CronJob{
			"name": "sdk-integration-" + suffix, "schedule": "0 0 2 1 *",
			"image_type": "image", "image_source": "alpine:latest", "command": "echo oi",
		}); err != nil {
			t.Errorf("Update: %v", err)
		}
		// Not calling Trigger: it would actually run the job's command.
		if _, err := client.Cron.WorkerStatus(ctx); err != nil {
			t.Errorf("WorkerStatus: %v", err)
		}
	})

	t.Run("UpTime", func(t *testing.T) {
		monitor, err := client.UpTime.Create(ctx, "https://alicercelabs.com.br", alicercelabs.UpTimeMonitor{
			"name": "sdk-integration-" + suffix,
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		id := fmt.Sprintf("%v", monitor["id"])
		defer client.UpTime.Delete(context.Background(), id)

		if _, err := client.UpTime.List(ctx); err != nil {
			t.Errorf("List: %v", err)
		}
		if _, err := client.UpTime.Get(ctx, id); err != nil {
			t.Errorf("Get: %v", err)
		}
		// Same PUT-is-whole-resource-replace deal as Cron.Update.
		if _, err := client.UpTime.Update(ctx, id, alicercelabs.UpTimeMonitor{
			"name": "sdk-integration-" + suffix, "url": "https://alicercelabs.com.br", "interval_sec": float64(300),
		}); err != nil {
			t.Errorf("Update: %v", err)
		}
		if _, err := client.UpTime.Checks(ctx, id); err != nil {
			t.Errorf("Checks: %v", err)
		}
		if _, err := client.UpTime.WorkerStatus(ctx); err != nil {
			t.Errorf("WorkerStatus: %v", err)
		}
	})

	t.Run("QRCode", func(t *testing.T) {
		result, err := client.QRCode.Generate(ctx, "https://alicercelabs.com.br", 256)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Content) == 0 {
			t.Error("got empty QR code content")
		}

		pix, err := client.QRCode.Pix(ctx, alicercelabs.PixParams{
			Chave: "11999999999", Nome: "Fulano", Cidade: "Sao Paulo", Valor: 10.5,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(pix.Content) == 0 {
			t.Error("got empty Pix QR code content")
		}
		if !strings.Contains(pix.CopiaCola, "br.gov.bcb.pix") {
			t.Errorf("CopiaCola doesn't look like a Pix payload: %q", pix.CopiaCola)
		}
	})

	t.Run("Imagem", func(t *testing.T) {
		// Generate our own PNG instead of depending on an external image
		// URL staying up — self-contained and just as real.
		qr, err := client.QRCode.Generate(ctx, "https://alicercelabs.com.br", 256)
		if err != nil {
			t.Fatalf("QRCode.Generate (fixture): %v", err)
		}
		if _, err := client.Imagem.Transform(ctx, qr.Content, "", url.Values{"grayscale": {"true"}}); err != nil {
			t.Errorf("Transform: %v", err)
		}
		if _, err := client.Imagem.Analyze(ctx, qr.Content, ""); err != nil {
			t.Errorf("Analyze: %v", err)
		}
	})

	t.Run("Templating", func(t *testing.T) {
		result, err := client.Templating.Invoice(ctx, alicercelabs.InvoiceRequest{
			Issuer:    alicercelabs.InvoiceParty{Name: "SDK Integration Test"},
			Recipient: alicercelabs.InvoiceParty{Name: "Cliente Exemplo"},
			Items:     []alicercelabs.InvoiceItem{{Description: "Teste", Quantity: 1, UnitPrice: 1}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.ContentType != "application/pdf" {
			t.Errorf("got content type %q", result.ContentType)
		}
	})

	t.Run("Functions", func(t *testing.T) {
		wasm := buildEchoWASM(t)
		name := "sdk-integration-" + suffix
		defer client.Functions.Delete(context.Background(), name)

		if _, err := client.Functions.Deploy(ctx, name, wasm); err != nil {
			t.Fatalf("Deploy: %v", err)
		}
		if _, err := client.Functions.List(ctx); err != nil {
			t.Errorf("List: %v", err)
		}
		if _, err := client.Functions.Get(ctx, name); err != nil {
			t.Errorf("Get: %v", err)
		}
		result, err := client.Functions.Invoke(ctx, name, []byte("ola"))
		if err != nil {
			t.Fatalf("Invoke: %v", err)
		}
		if string(result.Content) != "eco: ola" {
			t.Errorf("got content=%q", result.Content)
		}
	})

	t.Run("Account", func(t *testing.T) {
		created, err := client.Account.APIKeys.Create(ctx, "sdk-integration-"+suffix)
		if err != nil {
			t.Fatalf("APIKeys.Create: %v", err)
		}
		defer client.Account.APIKeys.Revoke(context.Background(), created.APIKey.ID)

		if _, err := client.Account.APIKeys.List(ctx); err != nil {
			t.Errorf("APIKeys.List: %v", err)
		}
		if _, err := client.Account.Usage(ctx, 7); err != nil {
			t.Errorf("Usage: %v", err)
		}
		newPassword := password + "-novo"
		if err := client.Account.ChangePassword(ctx, password, newPassword); err != nil {
			t.Errorf("ChangePassword: %v", err)
		}
	})
}
