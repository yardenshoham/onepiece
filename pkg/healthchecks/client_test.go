package healthchecks

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type pingRecord struct {
	method string
	path   string
	query  string
	body   string
}

func newTestServer(t *testing.T, records *[]pingRecord, mu *sync.Mutex, statusCode int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		*records = append(*records, pingRecord{
			method: r.Method,
			path:   r.URL.Path,
			query:  r.URL.RawQuery,
			body:   string(body),
		})
		mu.Unlock()
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte("OK"))
	}))
}

func newClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	c := NewClient(slog.Default(), "test-uuid")
	c.baseURL = serverURL
	c.retryDelay = 0
	return c
}

func TestPing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		call       func(*Client, context.Context)
		wantMethod string
		wantPath   string
		wantQuery  string
		wantBody   string
	}{
		{
			name:       "start",
			call:       func(c *Client, ctx context.Context) { c.Start(ctx, "rid-1") },
			wantMethod: http.MethodHead,
			wantPath:   "/test-uuid/start",
			wantQuery:  "rid=rid-1",
		},
		{
			name:       "success with body",
			call:       func(c *Client, ctx context.Context) { c.Success(ctx, "rid-2", "episodes=42") },
			wantMethod: http.MethodPost,
			wantPath:   "/test-uuid",
			wantQuery:  "rid=rid-2",
			wantBody:   "episodes=42",
		},
		{
			name:       "fail with body",
			call:       func(c *Client, ctx context.Context) { c.Fail(ctx, "rid-3", "something broke") },
			wantMethod: http.MethodPost,
			wantPath:   "/test-uuid/fail",
			wantQuery:  "rid=rid-3",
			wantBody:   "something broke",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var records []pingRecord
			var mu sync.Mutex
			srv := newTestServer(t, &records, &mu, http.StatusOK)
			defer srv.Close()

			c := newClient(t, srv.URL)
			tt.call(c, t.Context())

			mu.Lock()
			defer mu.Unlock()
			if len(records) != 1 {
				t.Fatalf("expected 1 request, got %d", len(records))
			}
			r := records[0]
			if r.method != tt.wantMethod {
				t.Errorf("method = %s, want %s", r.method, tt.wantMethod)
			}
			if r.path != tt.wantPath {
				t.Errorf("path = %s, want %s", r.path, tt.wantPath)
			}
			if r.query != tt.wantQuery {
				t.Errorf("query = %q, want %q", r.query, tt.wantQuery)
			}
			if tt.wantBody != "" && !strings.Contains(r.body, tt.wantBody) {
				t.Errorf("body = %q, want it to contain %q", r.body, tt.wantBody)
			}
		})
	}
}

func TestRetriesOnServerError(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var count int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		count++
		n := count
		mu.Unlock()
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newClient(t, srv.URL)
	c.Start(t.Context(), "rid-retry")

	mu.Lock()
	defer mu.Unlock()
	if count != 3 {
		t.Errorf("expected 3 attempts, got %d", count)
	}
}

func TestCancelledContextStopsRetries(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var count int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		count++
		mu.Unlock()
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel immediately

	c := newClient(t, srv.URL)
	c.Start(ctx, "rid-cancel")

	// Should not panic or hang; must not complete all retries
	mu.Lock()
	defer mu.Unlock()
	if count >= maxRetries {
		t.Errorf("expected fewer than %d attempts with cancelled context, got %d", maxRetries, count)
	}
}
