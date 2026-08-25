package alicercelabs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"strconv"
)

// QRCodeService is the client's QR code generator — client.QRCode.
type QRCodeService struct{ c *Client }

// Generate returns a QR code PNG for arbitrary text or a URL. size is in
// pixels; pass 0 for the API's default.
func (s *QRCodeService) Generate(ctx context.Context, data string, size int) (*BinaryResponse, error) {
	q := url.Values{"data": {data}}
	if size > 0 {
		q.Set("size", strconv.Itoa(size))
	}
	return s.c.doRaw(ctx, "GET", s.c.APIBase, "/api/v1/qrcode", q, nil, nil)
}

// PixParams is Pix's input. Chave, Nome and Cidade are required; the rest
// are optional. Nome/Cidade get uppercased and stripped of accents
// server-side, same as a card terminal, so send them however you like.
type PixParams struct {
	// Chave is the recipient's Pix key: CPF, CNPJ, email, phone or random
	// key.
	Chave string
	// Nome is the recipient's name, up to 25 characters.
	Nome string
	// Cidade is the recipient's city, up to 15 characters.
	Cidade string
	// Valor is the charge amount in reais (e.g. 10.50). Zero means the
	// payer types in the amount themselves.
	Valor float64
	// TxID identifies the transaction, letters and digits only, up to 25
	// characters. Empty means "***" (the standard's own "no identifier"
	// convention).
	TxID string
	// Descricao is free text embedded in the payload, up to 72 characters.
	Descricao string
	// Size is the image side in pixels, 64-1024. Zero uses the API's
	// default.
	Size int
}

// PixQRCode is Pix's answer: the QR code PNG, plus the same payload as
// plain text (the "copia e cola" code). Most integrations show both.
type PixQRCode struct {
	*BinaryResponse
	// CopiaCola is the Pix BR Code payload as text, straight from the
	// X-Pix-Copia-Cola response header.
	CopiaCola string
}

// Pix generates a static Pix QR code: builds the EMV/BR Code payload from
// params and renders it, same encoder and cache as Generate. Doesn't
// validate that Chave actually exists, that needs the Central Bank's
// DICT, out of scope here, only its format and length.
func (s *QRCodeService) Pix(ctx context.Context, params PixParams) (*PixQRCode, error) {
	q := url.Values{"chave": {params.Chave}, "nome": {params.Nome}, "cidade": {params.Cidade}}
	if params.Valor > 0 {
		q.Set("valor", strconv.FormatFloat(params.Valor, 'f', 2, 64))
	}
	if params.TxID != "" {
		q.Set("txid", params.TxID)
	}
	if params.Descricao != "" {
		q.Set("descricao", params.Descricao)
	}
	if params.Size > 0 {
		q.Set("size", strconv.Itoa(params.Size))
	}
	resp, err := s.c.doRaw(ctx, "GET", s.c.APIBase, "/api/v1/qrcode/pix", q, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PixQRCode{BinaryResponse: resp, CopiaCola: resp.Header.Get("X-Pix-Copia-Cola")}, nil
}

// ImagemService is the client's image-transform API — client.Imagem. See
// https://alicercelabs.com.br/apis/imagem for the full list of query
// parameters (resize, crop, rotate, format, quality, watermark_text,
// grayscale, blur, round, and many more, added over several phases of the
// API) — Params is passed straight through as the query string so this
// SDK doesn't go stale as new ones ship server-side.
type ImagemService struct{ c *Client }

// ErrExactlyOneSource is returned by Imagem's Transform/Analyze when
// neither or both of image/url are given.
var ErrExactlyOneSource = errors.New("alicercelabs: pass exactly one of image or url")

// Transform transforms an image, either uploaded directly (pass image,
// leave url empty) or fetched server-side (pass url, leave image nil).
func (s *ImagemService) Transform(ctx context.Context, image []byte, imgURL string, params url.Values) (*BinaryResponse, error) {
	if (len(image) == 0) == (imgURL == "") {
		return nil, ErrExactlyOneSource
	}
	q := cloneValues(params)
	if imgURL != "" {
		q.Set("url", imgURL)
	}
	var body io.Reader
	if len(image) > 0 {
		body = bytes.NewReader(image)
	}
	return s.c.doRaw(ctx, "POST", s.c.APIBase, "/api/v1/imagem/transform", q, body, nil)
}

// ImagemAnalysis is Analyze's answer.
type ImagemAnalysis struct {
	Width         int      `json:"width"`
	Height        int      `json:"height"`
	Format        string   `json:"format"`
	DominantColor string   `json:"dominant_color"`
	Palette       []string `json:"palette"`
	BlurHash      string   `json:"blurhash"`
}

// Analyze returns metadata about an image (dimensions, dominant color,
// palette, BlurHash) without transforming it.
func (s *ImagemService) Analyze(ctx context.Context, image []byte, imgURL string) (*ImagemAnalysis, error) {
	if (len(image) == 0) == (imgURL == "") {
		return nil, ErrExactlyOneSource
	}
	q := url.Values{}
	if imgURL != "" {
		q.Set("url", imgURL)
	}
	var body io.Reader
	if len(image) > 0 {
		body = bytes.NewReader(image)
	}
	var out ImagemAnalysis
	if err := s.c.doJSON(ctx, "POST", s.c.APIBase, "/api/v1/imagem/analyze", q, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TemplatingService is the client's invoice-generation API —
// client.Templating.
type TemplatingService struct{ c *Client }

// InvoiceParty is an invoice's issuer or recipient.
type InvoiceParty struct {
	Name     string `json:"name"`
	Document string `json:"document,omitempty"`
	Address  string `json:"address,omitempty"`
	Email    string `json:"email,omitempty"`
}

// InvoiceItem is one invoice line item. Quantity/UnitPrice drive the
// server-computed totals — never trust a total you compute client-side.
type InvoiceItem struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
}

// InvoiceRequest is Invoice's input.
type InvoiceRequest struct {
	InvoiceNumber string        `json:"invoice_number,omitempty"`
	Currency      string        `json:"currency,omitempty"`
	Issuer        InvoiceParty  `json:"issuer"`
	Recipient     InvoiceParty  `json:"recipient"`
	Items         []InvoiceItem `json:"items"`
	Notes         string        `json:"notes,omitempty"`
}

// Invoice generates an invoice PDF. Totals are always computed
// server-side from each item's Quantity/UnitPrice.
func (s *TemplatingService) Invoice(ctx context.Context, req InvoiceRequest) (*BinaryResponse, error) {
	body, err := jsonBody(req)
	if err != nil {
		return nil, err
	}
	return s.c.doRaw(ctx, "POST", s.c.APIBase, "/api/v1/templating/invoice", nil, body, map[string]string{"Content-Type": "application/json"})
}

func cloneValues(v url.Values) url.Values {
	out := url.Values{}
	for k, vals := range v {
		for _, val := range vals {
			out.Add(k, val)
		}
	}
	return out
}
