package alicercelabs

import (
	"context"
	"net/url"
	"strconv"
)

// KVService is the client's key-value store — client.KV.
type KVService struct{ c *Client }

// KVListResult is List's answer.
type KVListResult struct {
	Keys       []string `json:"keys"`
	NextCursor uint64   `json:"next_cursor"`
	Total      int64    `json:"total"`
}

// List returns a page of the client's key names (not values).
func (s *KVService) List(ctx context.Context, cursor uint64, count int) (*KVListResult, error) {
	q := url.Values{}
	if cursor > 0 {
		q.Set("cursor", strconv.FormatUint(cursor, 10))
	}
	if count > 0 {
		q.Set("count", strconv.Itoa(count))
	}
	var out KVListResult
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/kv", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get reads one key's value. Returns an *APIError with StatusCode 404
// (check with IsNotFound) if it doesn't exist or expired.
func (s *KVService) Get(ctx context.Context, key string) (string, error) {
	var out struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/kv/"+key, nil, nil, &out); err != nil {
		return "", err
	}
	return out.Value, nil
}

// Put writes a key. ttlSeconds=0 means no expiry.
func (s *KVService) Put(ctx context.Context, key, value string, ttlSeconds int) error {
	body, err := jsonBody(map[string]any{"value": value, "ttl_seconds": ttlSeconds})
	if err != nil {
		return err
	}
	return s.c.doJSON(ctx, "PUT", s.c.APIBase, "/api/v1/kv/"+key, nil, body, nil)
}

// Delete removes a key. Idempotent — deleting a key that doesn't exist is
// not an error.
func (s *KVService) Delete(ctx context.Context, key string) error {
	return s.c.doJSON(ctx, "DELETE", s.c.APIBase, "/api/v1/kv/"+key, nil, nil, nil)
}

// QueueService is the client's FIFO queue API — client.Queue.
type QueueService struct{ c *Client }

// QueueStats is Stats's answer.
type QueueStats struct {
	Name  string `json:"name"`
	Depth int64  `json:"depth"`
}

// Push appends a message to the end of a FIFO queue (created on first
// use).
func (s *QueueService) Push(ctx context.Context, name, message string) error {
	body, err := jsonBody(map[string]any{"message": message})
	if err != nil {
		return err
	}
	return s.c.doJSON(ctx, "POST", s.c.APIBase, "/api/v1/queue/"+name+"/push", nil, body, nil)
}

// Pull pulls the oldest message. waitSeconds (0-5) turns this into a short
// long-poll instead of an immediate empty check. Returns ("", nil) if the
// queue is empty — check the second value (ok) to tell "empty" apart from
// a message that happens to be an empty string.
func (s *QueueService) Pull(ctx context.Context, name string, waitSeconds int) (message string, ok bool, err error) {
	q := url.Values{}
	if waitSeconds > 0 {
		q.Set("wait", strconv.Itoa(waitSeconds))
	}
	var out struct {
		Message string `json:"message"`
	}
	err = s.c.doJSON(ctx, "POST", s.c.APIBase, "/api/v1/queue/"+name+"/pull", q, nil, &out)
	if err != nil {
		if IsNotFound(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return out.Message, true, nil
}

// Stats returns a queue's current depth.
func (s *QueueService) Stats(ctx context.Context, name string) (*QueueStats, error) {
	var out QueueStats
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/queue/"+name+"/stats", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete removes a queue entirely. Idempotent.
func (s *QueueService) Delete(ctx context.Context, name string) error {
	return s.c.doJSON(ctx, "DELETE", s.c.APIBase, "/api/v1/queue/"+name, nil, nil, nil)
}

// EdgeDBService is the client's per-client SQLite database API —
// client.EdgeDB. See https://alicercelabs.com.br/apis/edgedb.
type EdgeDBService struct{ c *Client }

// EdgeDBInfo describes one database — part of List's answer.
type EdgeDBInfo struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
	UpdatedAt string `json:"updated_at"`
}

// EdgeDBQueryResult is Query's answer. For SELECT/PRAGMA/EXPLAIN/WITH,
// Columns/Rows are populated; for everything else, RowsAffected/
// LastInsertID are.
type EdgeDBQueryResult struct {
	Columns      []string `json:"columns,omitempty"`
	Rows         [][]any  `json:"rows,omitempty"`
	RowsAffected int64    `json:"rows_affected,omitempty"`
	LastInsertID int64    `json:"last_insert_id,omitempty"`
}

// List returns every database this client has.
func (s *EdgeDBService) List(ctx context.Context) ([]EdgeDBInfo, error) {
	var out struct {
		Databases []EdgeDBInfo `json:"databases"`
	}
	if err := s.c.doJSON(ctx, "GET", s.c.APIBase, "/api/v1/edgedb", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Databases, nil
}

// Query runs one SQL statement against name (created on first query). args
// are positional values for the statement's "?" placeholders.
func (s *EdgeDBService) Query(ctx context.Context, name, sql string, args []any) (*EdgeDBQueryResult, error) {
	body, err := jsonBody(map[string]any{"sql": sql, "args": args})
	if err != nil {
		return nil, err
	}
	var out EdgeDBQueryResult
	if err := s.c.doJSON(ctx, "POST", s.c.APIBase, "/api/v1/edgedb/"+name+"/query", nil, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete removes a database file entirely. Idempotent.
func (s *EdgeDBService) Delete(ctx context.Context, name string) error {
	return s.c.doJSON(ctx, "DELETE", s.c.APIBase, "/api/v1/edgedb/"+name, nil, nil, nil)
}
