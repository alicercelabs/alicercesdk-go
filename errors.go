package alicercelabs

import (
	"errors"
	"fmt"
)

// APIError is returned for every non-2xx response from the API. The
// message always comes from the API's own JSON error envelope
// ({"success": false, "error": "..."}) — even the endpoints whose success
// response is raw bytes (QRCode, Imagem.Transform, Templating.Invoice,
// Functions.Invoke) still report errors this way.
type APIError struct {
	StatusCode int
	Message    string
	RequestID  string
	// RetryAfter is the value of the Retry-After header, in seconds — only
	// set when StatusCode is 429.
	RetryAfter int
}

func (e *APIError) Error() string {
	return fmt.Sprintf("alicercelabs: %d %s", e.StatusCode, e.Message)
}

// IsNotFound, IsAuthenticationError, IsRateLimit and IsValidationError are
// convenience checks for the status codes callers most often branch on —
// equivalent to checking apiErr.StatusCode directly after an
// errors.As(err, &apiErr), but reads better at the call site.

func IsNotFound(err error) bool { return statusIs(err, 404) }

func IsAuthenticationError(err error) bool { return statusIs(err, 401) }

func IsRateLimit(err error) bool { return statusIs(err, 429) }

func IsValidationError(err error) bool { return statusIs(err, 400) }

func IsServiceUnavailable(err error) bool { return statusIs(err, 503) }

func statusIs(err error, code int) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == code
	}
	return false
}
