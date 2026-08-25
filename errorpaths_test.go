package alicercelabs

// A single error-propagation sweep across every resource method. Every
// wrapper here does the same thing on failure (propagate doJSON/doRaw's
// error unchanged), so one 500 per endpoint proves the error path works
// without repeating client_test.go's deeper error-shape assertions for
// each of the 62 methods individually. Auth.Login isn't here because it
// has its own error path (basic auth, no doJSON) already covered by
// TestAuthLoginErrorPath in coverage_test.go.

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestErrorPropagatesFromEveryEndpoint(t *testing.T) {
	cases := []struct {
		route string
		call  func(ctx context.Context, c *Client) error
	}{
		{"GET /api/v1/ip/8.8.8.8", func(ctx context.Context, c *Client) error { _, err := c.IP.Lookup(ctx, "8.8.8.8"); return err }},
		{"GET /api/v1/ip/self", func(ctx context.Context, c *Client) error { _, err := c.IP.Self(ctx); return err }},
		{"GET /api/v1/cep/01310100", func(ctx context.Context, c *Client) error { _, err := c.CEP.Get(ctx, "01310100", nil); return err }},
		{"GET /api/v1/cep/busca", func(ctx context.Context, c *Client) error {
			_, err := c.CEP.Search(ctx, "SP", "São Paulo", "Rua")
			return err
		}},
		{"GET /api/v1/cep/cidades", func(ctx context.Context, c *Client) error { _, err := c.CEP.Cities(ctx, "SP"); return err }},
		{"GET /api/v1/cep/bairros", func(ctx context.Context, c *Client) error {
			_, err := c.CEP.Neighborhoods(ctx, "SP", "São Paulo")
			return err
		}},
		{"GET /api/v1/cep/distance/a/b", func(ctx context.Context, c *Client) error { _, err := c.CEP.Distance(ctx, "a", "b", false); return err }},
		{"POST /api/v1/cep/lote", func(ctx context.Context, c *Client) error {
			_, err := c.CEP.Bulk(ctx, []string{"01310100"})
			return err
		}},
		{"GET /api/v1/dns/exemplo.com", func(ctx context.Context, c *Client) error { _, err := c.DNS.Lookup(ctx, "exemplo.com"); return err }},
		{"GET /api/v1/email/verify", func(ctx context.Context, c *Client) error { _, err := c.Email.Verify(ctx, "a@b.com"); return err }},
		{"GET /api/v1/ssl/exemplo.com", func(ctx context.Context, c *Client) error { _, err := c.SSL.Check(ctx, "exemplo.com"); return err }},
		{"GET /api/v1/maps/geocode", func(ctx context.Context, c *Client) error { _, err := c.Maps.Geocode(ctx, "endereco"); return err }},
		{"GET /api/v1/maps/reverse", func(ctx context.Context, c *Client) error { _, err := c.Maps.Reverse(ctx, 0, 0); return err }},
		{"GET /api/v1/maps/route", func(ctx context.Context, c *Client) error { _, err := c.Maps.Route(ctx, "a", "b"); return err }},
		{"GET /api/v1/trust/exemplo.com", func(ctx context.Context, c *Client) error {
			_, err := c.Trust.Check(ctx, "exemplo.com", "")
			return err
		}},

		{"GET /api/v1/kv", func(ctx context.Context, c *Client) error { _, err := c.KV.List(ctx, 0, 0); return err }},
		{"GET /api/v1/kv/tema", func(ctx context.Context, c *Client) error { _, err := c.KV.Get(ctx, "tema"); return err }},
		{"PUT /api/v1/kv/tema", func(ctx context.Context, c *Client) error { return c.KV.Put(ctx, "tema", "v", 0) }},
		{"DELETE /api/v1/kv/tema", func(ctx context.Context, c *Client) error { return c.KV.Delete(ctx, "tema") }},
		{"POST /api/v1/queue/fila/push", func(ctx context.Context, c *Client) error { return c.Queue.Push(ctx, "fila", "m") }},
		{"GET /api/v1/queue/fila/stats", func(ctx context.Context, c *Client) error { _, err := c.Queue.Stats(ctx, "fila"); return err }},
		{"DELETE /api/v1/queue/fila", func(ctx context.Context, c *Client) error { return c.Queue.Delete(ctx, "fila") }},
		{"GET /api/v1/edgedb", func(ctx context.Context, c *Client) error { _, err := c.EdgeDB.List(ctx); return err }},
		{"POST /api/v1/edgedb/meubanco/query", func(ctx context.Context, c *Client) error {
			_, err := c.EdgeDB.Query(ctx, "meubanco", "SELECT 1", nil)
			return err
		}},
		{"DELETE /api/v1/edgedb/meubanco", func(ctx context.Context, c *Client) error { return c.EdgeDB.Delete(ctx, "meubanco") }},

		{"POST /api/v1/cron/jobs", func(ctx context.Context, c *Client) error {
			_, err := c.Cron.Create(ctx, CronJob{"name": "j"})
			return err
		}},
		{"GET /api/v1/cron/jobs", func(ctx context.Context, c *Client) error { _, err := c.Cron.List(ctx); return err }},
		{"GET /api/v1/cron/jobs/1", func(ctx context.Context, c *Client) error { _, err := c.Cron.Get(ctx, "1"); return err }},
		{"PUT /api/v1/cron/jobs/1", func(ctx context.Context, c *Client) error { _, err := c.Cron.Update(ctx, "1", CronJob{}); return err }},
		{"DELETE /api/v1/cron/jobs/1", func(ctx context.Context, c *Client) error { return c.Cron.Delete(ctx, "1") }},
		{"POST /api/v1/cron/jobs/1/trigger", func(ctx context.Context, c *Client) error { return c.Cron.Trigger(ctx, "1") }},
		{"GET /api/v1/cron/worker/status", func(ctx context.Context, c *Client) error { _, err := c.Cron.WorkerStatus(ctx); return err }},
		{"POST /api/v1/cron/worker/start", func(ctx context.Context, c *Client) error { return c.Cron.WorkerStart(ctx) }},
		{"POST /api/v1/cron/worker/stop", func(ctx context.Context, c *Client) error { return c.Cron.WorkerStop(ctx) }},

		{"POST /api/v1/uptime/monitors", func(ctx context.Context, c *Client) error {
			_, err := c.UpTime.Create(ctx, "https://a.com", nil)
			return err
		}},
		{"GET /api/v1/uptime/monitors", func(ctx context.Context, c *Client) error { _, err := c.UpTime.List(ctx); return err }},
		{"GET /api/v1/uptime/monitors/1", func(ctx context.Context, c *Client) error { _, err := c.UpTime.Get(ctx, "1"); return err }},
		{"PUT /api/v1/uptime/monitors/1", func(ctx context.Context, c *Client) error {
			_, err := c.UpTime.Update(ctx, "1", UpTimeMonitor{})
			return err
		}},
		{"DELETE /api/v1/uptime/monitors/1", func(ctx context.Context, c *Client) error { return c.UpTime.Delete(ctx, "1") }},
		{"GET /api/v1/uptime/monitors/1/checks", func(ctx context.Context, c *Client) error { _, err := c.UpTime.Checks(ctx, "1"); return err }},
		{"GET /api/v1/uptime/worker/status", func(ctx context.Context, c *Client) error { _, err := c.UpTime.WorkerStatus(ctx); return err }},
		{"POST /api/v1/uptime/worker/start", func(ctx context.Context, c *Client) error { return c.UpTime.WorkerStart(ctx) }},
		{"POST /api/v1/uptime/worker/stop", func(ctx context.Context, c *Client) error { return c.UpTime.WorkerStop(ctx) }},

		{"GET /api/v1/qrcode", func(ctx context.Context, c *Client) error { _, err := c.QRCode.Generate(ctx, "hello", 0); return err }},
		{"GET /api/v1/qrcode/pix", func(ctx context.Context, c *Client) error {
			_, err := c.QRCode.Pix(ctx, PixParams{Chave: "11999999999", Nome: "Fulano", Cidade: "Sao Paulo"})
			return err
		}},
		{"POST /api/v1/imagem/transform", func(ctx context.Context, c *Client) error {
			_, err := c.Imagem.Transform(ctx, nil, "https://a.com/x.jpg", nil)
			return err
		}},
		{"POST /api/v1/imagem/analyze", func(ctx context.Context, c *Client) error {
			_, err := c.Imagem.Analyze(ctx, nil, "https://a.com/x.jpg")
			return err
		}},
		{"POST /api/v1/templating/invoice", func(ctx context.Context, c *Client) error {
			_, err := c.Templating.Invoice(ctx, InvoiceRequest{
				Issuer: InvoiceParty{Name: "a"}, Recipient: InvoiceParty{Name: "b"},
				Items: []InvoiceItem{{Description: "x", Quantity: 1, UnitPrice: 1}},
			})
			return err
		}},

		{"GET /api/v1/functions", func(ctx context.Context, c *Client) error { _, err := c.Functions.List(ctx); return err }},
		{"GET /api/v1/functions/minha", func(ctx context.Context, c *Client) error { _, err := c.Functions.Get(ctx, "minha"); return err }},
		{"PUT /api/v1/functions/minha", func(ctx context.Context, c *Client) error {
			_, err := c.Functions.Deploy(ctx, "minha", []byte("x"))
			return err
		}},
		{"DELETE /api/v1/functions/minha", func(ctx context.Context, c *Client) error { return c.Functions.Delete(ctx, "minha") }},
		{"POST /api/v1/functions/minha/invoke", func(ctx context.Context, c *Client) error {
			_, err := c.Functions.Invoke(ctx, "minha", []byte("x"))
			return err
		}},

		{"POST /api/v1/user/register", func(ctx context.Context, c *Client) error {
			_, err := c.Auth.Register(ctx, "u", "u@a.com", "senha1234")
			return err
		}},
		{"GET /api/v1/user/me", func(ctx context.Context, c *Client) error { _, err := c.Auth.Me(ctx); return err }},
		{"DELETE /api/v1/user/me", func(ctx context.Context, c *Client) error { return c.Auth.DeleteAccount(ctx) }},

		{"GET /api/v1/account/usage", func(ctx context.Context, c *Client) error { _, err := c.Account.Usage(ctx, 0); return err }},
		{"POST /api/v1/account/password", func(ctx context.Context, c *Client) error { return c.Account.ChangePassword(ctx, "a", "b") }},
		{"GET /api/v1/account/apikeys", func(ctx context.Context, c *Client) error { _, err := c.Account.APIKeys.List(ctx); return err }},
		{"POST /api/v1/account/apikeys", func(ctx context.Context, c *Client) error { _, err := c.Account.APIKeys.Create(ctx, "n"); return err }},
		{"DELETE /api/v1/account/apikeys/1", func(ctx context.Context, c *Client) error { return c.Account.APIKeys.Revoke(ctx, 1) }},
	}

	for _, tc := range cases {
		t.Run(tc.route, func(t *testing.T) {
			srv, routes, _ := newTestServer(t)
			routes[tc.route] = errorJSON(http.StatusInternalServerError, "erro interno")

			c := New("tok", WithAPIBase(srv.URL), WithAccountBase(srv.URL))
			err := tc.call(context.Background(), c)
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected *APIError, got %v", err)
			}
			if apiErr.StatusCode != http.StatusInternalServerError {
				t.Errorf("got status %d", apiErr.StatusCode)
			}
		})
	}
}

func TestJSONBodyNil(t *testing.T) {
	r, err := jsonBody(nil)
	if err != nil || r != nil {
		t.Errorf("got reader=%v err=%v, want nil/nil", r, err)
	}
}
