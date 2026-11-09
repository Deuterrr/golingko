package middleware

import (
	"bytes"
	"maps"
	"net/http"
	"sync"
	"time"
)

type cachedResponse struct {
	statusCode int
	body       []byte
	header     http.Header
	expiresAt  time.Time
}

// Concurrency-safe in-memory cache preventing duplicate execution of mutating operations.
// It replays cached responses for matching idempotency keys within the expiration duration.
type IdempotencyStore struct {
	mu    sync.RWMutex
	items map[string]cachedResponse
}

func NewIdempotencyStore() *IdempotencyStore {
	return &IdempotencyStore{
		items: make(map[string]cachedResponse),
	}
}

type bodyCaptureWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (w *bodyCaptureWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *bodyCaptureWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Intercepts mutating requests containing an idempotency key to prevent duplicate operations.
// It returns previously recorded responses for repeated submissions within the configured duration.
func (s *IdempotencyStore) Middleware(ttl time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Idempotency-Key")
			if key == "" || (r.Method != http.MethodPost && r.Method != http.MethodPut) {
				next.ServeHTTP(w, r)
				return
			}

			s.mu.RLock()
			cached, exists := s.items[key]
			s.mu.RUnlock()

			if exists && time.Now().Before(cached.expiresAt) {
				for h, vals := range cached.header {
					for _, v := range vals {
						w.Header().Add(h, v)
					}
				}
				w.Header().Set("X-Cache-Lookup", "HIT")
				w.WriteHeader(cached.statusCode)
				_, _ = w.Write(cached.body)
				return
			}

			capture := &bodyCaptureWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(capture, r)

			if capture.statusCode >= 200 && capture.statusCode < 300 {
				headerCopy := make(http.Header)
				maps.Copy(headerCopy, capture.Header())

				s.mu.Lock()
				s.items[key] = cachedResponse{
					statusCode: capture.statusCode,
					body:       capture.body.Bytes(),
					header:     headerCopy,
					expiresAt:  time.Now().Add(ttl),
				}
				s.mu.Unlock()
			}
		})
	}
}
