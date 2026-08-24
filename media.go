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
