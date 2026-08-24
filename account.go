package alicercelabs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// AuthService handles registration, login and the authenticated account's
// own profile — client.Auth. These talk to the product API host (the same
// one every other resource uses); AccountService below talks to the panel
// host instead, since that's where API keys and usage analytics live.
type AuthService struct{ c *Client }

// User is an account's profile.
type User struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// RegisterResult is Register's answer.
type RegisterResult struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

// Register creates an account. On success, the returned token is also
// stored on the Client — you can start calling other APIs right away
// without a separate Login call.
func (s *AuthService) Register(ctx context.Context, username, email, password string) (*RegisterResult, error) {
	body, err := jsonBody(map[string]string{"username": username, "email": email, "password": password})
	if err != nil {
		return nil, err
	}
	var out RegisterResult
	if err := s.c.doJSON(ctx, "POST", s.c.APIBase, "/api/v1/user/register", nil, body, &out); err != nil {
		return nil, err
	}
	s.c.APIKey = out.Token
	return &out, nil
}

// Login exchanges username/password (HTTP Basic Auth under the hood) for
// a token. Stores the token on the Client and also returns it, in case you
// want to persist it yourself for next time.
func (s *AuthService) Login(ctx context.Context, username, password string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", s.c.APIBase+"/api/v1/user/login", nil)
	if err != nil {
		return "", fmt.Errorf("alicercelabs: build request: %w", err)
	}
	req.SetBasicAuth(username, password)

	resp, err := s.c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("alicercelabs: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("alicercelabs: read response body: %w", err)
	}
	if resp.StatusCode >= 400 {
		return "", apiErrorFromBody(resp, respBody)
	}

	var env struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &env); err != nil {
		return "", fmt.Errorf("alicercelabs: decode response: %w", err)
	}
	s.c.APIKey = env.Data.Token
	return env.Data.Token, nil
}

// Me returns the authenticated account's own profile.
func (s *AuthService) Me(ctx context.Context) (*User, error) {
	var out User
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/user/me", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteAccount deactivates the authenticated account. Irreversible from
// the API's side — think before calling this.
func (s *AuthService) DeleteAccount(ctx context.Context) error {
	return s.c.doJSON(ctx, "DELETE", s.c.APIBase, "/api/v1/user/me", nil, nil, nil)
}

// AccountService is the client's self-service account data — client.Account.
// Talks to the panel/account API host (Client.AccountBase), which is where
// API keys and usage analytics live.
type AccountService struct {
	c       *Client
	APIKeys *APIKeysService
}

// APIKeysService manages the account's own API keys — client.Account.APIKeys.
type APIKeysService struct{ c *Client }

// APIKeyRecord is one API key's metadata (never the raw key value, except
// right after Create).
type APIKeyRecord struct {
	ID         uint64  `json:"id"`
	Name       string  `json:"name"`
	KeyPrefix  string  `json:"key_prefix"`
	Active     bool    `json:"active"`
	ExpiresAt  *string `json:"expires_at,omitempty"`
	LastUsedAt *string `json:"last_used_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

// CreateAPIKeyResult is Create's answer. Key is the raw value (alk_...) —
// it's only ever present in this one response, so save it now.
type CreateAPIKeyResult struct {
	Key    string       `json:"key"`
	APIKey APIKeyRecord `json:"api_key"`
}

// List returns every API key on the account.
func (s *APIKeysService) List(ctx context.Context) ([]APIKeyRecord, error) {
	var out []APIKeyRecord
	if err := s.c.doJSON(ctx, "GET", s.c.AccountBase, "/api/v1/account/apikeys", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Create creates a new API key.
func (s *APIKeysService) Create(ctx context.Context, name string) (*CreateAPIKeyResult, error) {
	body, err := jsonBody(map[string]string{"name": name})
	if err != nil {
		return nil, err
	}
	var out CreateAPIKeyResult
	if err := s.c.doJSON(ctx, "POST", s.c.AccountBase, "/api/v1/account/apikeys", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Revoke revokes an API key by id.
func (s *APIKeysService) Revoke(ctx context.Context, id uint64) error {
	return s.c.doJSON(ctx, "DELETE", s.c.AccountBase, fmt.Sprintf("/api/v1/account/apikeys/%d", id), nil, nil, nil)
}

// ChangePassword changes the authenticated account's password.
func (s *AccountService) ChangePassword(ctx context.Context, currentPassword, newPassword string) error {
	body, err := jsonBody(map[string]string{"current_password": currentPassword, "new_password": newPassword})
	if err != nil {
		return err
	}
	return s.c.doJSON(ctx, "POST", s.c.AccountBase, "/api/v1/account/password", nil, body, nil)
}

// UsageRow is one line of usage analytics: how many requests this account
// made to one API+operation on one day.
type UsageRow struct {
	API          string `json:"api"`
	Operation    string `json:"operation"`
	Day          string `json:"day"`
	StatusClass  int    `json:"status_class"`
	RequestCount int64  `json:"request_count"`
}

// Usage returns your own usage analytics: request counts per
// API/operation/day for the last `days` days (default 30 server-side if
// you pass 0).
func (s *AccountService) Usage(ctx context.Context, days int) ([]UsageRow, error) {
	q := url.Values{}
	if days > 0 {
		q.Set("days", fmt.Sprintf("%d", days))
	}
	var out []UsageRow
	if err := s.c.doJSON(ctx, "GET", s.c.AccountBase, "/api/v1/account/usage", q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
