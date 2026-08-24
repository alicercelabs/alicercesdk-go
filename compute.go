package alicercelabs

import (
	"bytes"
	"context"
)

// FunctionsService is the client's WASM-execution API — client.Functions.
// See https://alicercelabs.com.br/apis/functions for the sandbox's exact
// guarantees (no network, no filesystem, bounded memory/time).
type FunctionsService struct{ c *Client }

// FunctionInfo is one deployed function's metadata.
type FunctionInfo struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// List returns every function this client has deployed.
func (s *FunctionsService) List(ctx context.Context) ([]FunctionInfo, error) {
	var out struct {
		Functions []FunctionInfo `json:"functions"`
	}
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/functions", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Functions, nil
}

// Get returns one function's metadata.
func (s *FunctionsService) Get(ctx context.Context, name string) (*FunctionInfo, error) {
	var out FunctionInfo
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/functions/"+name, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Deploy uploads a compiled WASM binary — any language that compiles to
// WASI works, including plain Go (GOOS=wasip1 GOARCH=wasm go build).
func (s *FunctionsService) Deploy(ctx context.Context, name string, wasm []byte) (*FunctionInfo, error) {
	var out FunctionInfo
	if err := s.c.doJSON(ctx, "PUT", s.c.APIBase, "/api/v1/functions/"+name, nil, bytes.NewReader(wasm), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete removes a function. Idempotent.
func (s *FunctionsService) Delete(ctx context.Context, name string) error {
	return s.c.doJSON(ctx, "DELETE", s.c.APIBase, "/api/v1/functions/"+name, nil, nil, nil)
}

// Invoke runs a deployed function. The response is exactly what the
// function itself produced (status, headers, body) — not the usual
// envelope, since it's the client's own code's output, not ours.
func (s *FunctionsService) Invoke(ctx context.Context, name string, body []byte) (*BinaryResponse, error) {
	return s.c.doRaw(ctx, "POST", s.c.APIBase, "/api/v1/functions/"+name+"/invoke", nil, bytes.NewReader(body), nil)
}
