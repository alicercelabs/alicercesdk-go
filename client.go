// Package alicercelabs is the official Go SDK for AlicerceLabs
// (https://alicercelabs.com.br) — basic API infrastructure for building in
// Brazil: IP, CEP, DNS, email, queues, an edge database, WASM execution
// and more, all behind one auth scheme and one response envelope.
//
//	client := alicercelabs.New("alk_...")
//	endereco, err := client.CEP.Get(ctx, "01310100", nil)
//
// Every product API is a field on Client (Client.IP, Client.CEP, ...),
// plus Client.Auth (register/login/profile) and Client.Account (your own
// API keys and usage analytics).
package alicercelabs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultAPIBase is the product API host — every resource except Auth
	// (register/login/me) and Account (api keys, usage) lives here.
	DefaultAPIBase = "https://api.alicercelabs.com.br"
	// DefaultAccountBase is the panel/account API host — Client.Account's
	// endpoints (API keys, usage analytics) live here.
	DefaultAccountBase = "https://app.alicercelabs.com.br"
)

// Client is the SDK entry point. Construct one with New and reuse it —
// it holds no per-request state beyond your credentials.
type Client struct {
	APIKey      string
	APIBase     string
	AccountBase string
	HTTPClient  *http.Client

	IP         *IPService
	CEP        *CEPService
	CNPJ       *CNPJService
	CPF        *CPFService
	Feriados   *FeriadosService
	DiasUteis  *DiasUteisService
	ISBN       *ISBNService
	IBGE       *IBGEService
	Bancos     *BancosService
	NCM        *NCMService
	OMS        *OMSService
	Cambio     *CambioService
	Taxas      *TaxasService
	RegistroBR *RegistroBRService
	PIX        *PIXService
	DNS        *DNSService
	Email      *EmailService
	SSL        *SSLService
	Maps       *MapsService
	Trust      *TrustService
	KV         *KVService
	Queue      *QueueService
	EdgeDB     *EdgeDBService
	Cron       *CronService
	UpTime     *UpTimeService
	QRCode     *QRCodeService
	Imagem     *ImagemService
	Templating *TemplatingService
	Functions  *FunctionsService
	Auth       *AuthService
	Account    *AccountService
}

// Option configures a Client at construction time. See WithAPIBase,
// WithAccountBase, WithHTTPClient and WithTimeout.
type Option func(*Client)

// WithAPIBase overrides the product API host — only needed for local
// development against a self-hosted instance.
func WithAPIBase(base string) Option {
	return func(c *Client) { c.APIBase = strings.TrimRight(base, "/") }
}

// WithAccountBase overrides the panel/account API host.
func WithAccountBase(base string) Option {
	return func(c *Client) { c.AccountBase = strings.TrimRight(base, "/") }
}

// WithHTTPClient replaces the underlying *http.Client entirely — for
// custom transports, proxies, or test doubles.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.HTTPClient = hc }
}

// WithTimeout sets the HTTP client's timeout. Ignored if combined with
// WithHTTPClient (that option replaces the client wholesale) — pass it
// before WithHTTPClient if you want both, or set Timeout on your own
// *http.Client instead.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.HTTPClient.Timeout = d }
}

// New builds a Client. apiKey is either a JWT (from Auth.Login/Register)
// or a static API key (alk_...) — both are sent the same way, so you can
// pass either. Leave it empty if you're about to call Auth.Register or
// Auth.Login: both store the resulting token on the client automatically.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		APIKey:      apiKey,
		APIBase:     DefaultAPIBase,
		AccountBase: DefaultAccountBase,
		HTTPClient:  &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}

	c.IP = &IPService{c}
	c.CEP = &CEPService{c}
	c.CNPJ = &CNPJService{c}
	c.CPF = &CPFService{c}
	c.Feriados = &FeriadosService{c}
	c.DiasUteis = &DiasUteisService{c}
	c.ISBN = &ISBNService{c}
	c.IBGE = &IBGEService{c}
	c.Bancos = &BancosService{c}
	c.NCM = &NCMService{c}
	c.OMS = &OMSService{c}
	c.Cambio = &CambioService{c}
	c.Taxas = &TaxasService{c}
	c.RegistroBR = &RegistroBRService{c}
	c.PIX = &PIXService{c}
	c.DNS = &DNSService{c}
	c.Email = &EmailService{c}
	c.SSL = &SSLService{c}
	c.Maps = &MapsService{c}
	c.Trust = &TrustService{c}
	c.KV = &KVService{c}
	c.Queue = &QueueService{c}
	c.EdgeDB = &EdgeDBService{c}
	c.Cron = &CronService{c}
	c.UpTime = &UpTimeService{c}
	c.QRCode = &QRCodeService{c}
	c.Imagem = &ImagemService{c}
	c.Templating = &TemplatingService{c}
	c.Functions = &FunctionsService{c}
	c.Auth = &AuthService{c}
	c.Account = &AccountService{c: c, APIKeys: &APIKeysService{c}}
	return c
}

// envelope mirrors web/shared/response.go's APIResponse — the shape every
// JSON endpoint answers with.
type envelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
	Meta    *struct {
		ElapsedMs int64  `json:"elapsed_ms,omitempty"`
		RequestID string `json:"request_id,omitempty"`
	} `json:"meta,omitempty"`
}

// BinaryResponse is what the raw-bytes endpoints return: QRCode.Generate,
// Imagem.Transform, Templating.Invoice, Functions.Invoke. StatusCode and
// Header matter most for Functions.Invoke — its response is whatever the
// client's own deployed function set, not a fixed content type like the
// other three.
type BinaryResponse struct {
	Content     []byte
	ContentType string
	StatusCode  int
	Header      http.Header
}

// Save writes the response body to path — the common thing you want to do
// with a generated image or PDF.
func (b *BinaryResponse) Save(path string) error {
	return os.WriteFile(path, b.Content, 0o644)
}

func (c *Client) authHeader(req *http.Request) {
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
}

// doJSON calls a JSON-envelope endpoint and decodes its "data" field into
// out (a pointer). out may be nil for endpoints with no meaningful
// response body (a 200 with just {"message": "..."} you don't care about).
func (c *Client) doJSON(ctx context.Context, method, base, path string, query url.Values, body io.Reader, out any) error {
	resp, err := c.rawDo(ctx, method, base, path, query, body, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("alicercelabs: read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return apiErrorFromBody(resp, respBody)
	}
	if out == nil || len(respBody) == 0 {
		return nil
	}
	var env envelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return fmt.Errorf("alicercelabs: decode response envelope: %w", err)
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("alicercelabs: decode response data: %w", err)
	}
	return nil
}

// doRaw calls one of the raw-bytes endpoints. Errors from these still come
// back as the normal JSON envelope, so error handling is unchanged — only
// the success path is bytes instead of JSON.
func (c *Client) doRaw(ctx context.Context, method, base, path string, query url.Values, body io.Reader, headers map[string]string) (*BinaryResponse, error) {
	resp, err := c.rawDo(ctx, method, base, path, query, body, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("alicercelabs: read response body: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, apiErrorFromBody(resp, respBody)
	}
	return &BinaryResponse{
		Content:     respBody,
		ContentType: resp.Header.Get("Content-Type"),
		StatusCode:  resp.StatusCode,
		Header:      resp.Header,
	}, nil
}

func (c *Client) rawDo(ctx context.Context, method, base, path string, query url.Values, body io.Reader, headers map[string]string) (*http.Response, error) {
	u := base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, fmt.Errorf("alicercelabs: build request: %w", err)
	}
	c.authHeader(req)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("alicercelabs: request failed: %w", err)
	}
	return resp, nil
}

func apiErrorFromBody(resp *http.Response, body []byte) *APIError {
	apiErr := &APIError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("unexpected response (status %d)", resp.StatusCode)}
	var env envelope
	if json.Unmarshal(body, &env) == nil {
		if env.Error != "" {
			apiErr.Message = env.Error
		}
		if env.Meta != nil {
			apiErr.RequestID = env.Meta.RequestID
		}
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if n, err := strconv.Atoi(ra); err == nil {
				apiErr.RetryAfter = n
			}
		}
	}
	return apiErr
}

// jsonBody marshals v to a *bytes.Reader for use as a request body — a
// small helper so resource methods don't each repeat the
// marshal-then-wrap dance.
func jsonBody(v any) (io.Reader, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("alicercelabs: encode request body: %w", err)
	}
	return bytes.NewReader(b), nil
}
