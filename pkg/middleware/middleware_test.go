package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestIdempotencyMiddleware(t *testing.T) {
	store := NewIdempotencyStore()
	var executionCount int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&executionCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	})

	wrapped := store.Middleware(1 * time.Minute)(handler)

	// First request with key
	req1 := httptest.NewRequest(http.MethodPost, "/api/test", bytes.NewBufferString(`{}`))
	req1.Header.Set("Idempotency-Key", "key-12345")
	rr1 := httptest.NewRecorder()
	wrapped.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr1.Code)
	}
	if atomic.LoadInt32(&executionCount) != 1 {
		t.Fatalf("expected 1 execution, got %d", atomic.LoadInt32(&executionCount))
	}

	// Second request with same key
	req2 := httptest.NewRequest(http.MethodPost, "/api/test", bytes.NewBufferString(`{}`))
	req2.Header.Set("Idempotency-Key", "key-12345")
	rr2 := httptest.NewRecorder()
	wrapped.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr2.Code)
	}
	if atomic.LoadInt32(&executionCount) != 1 {
		t.Fatalf("expected still 1 execution due to idempotency cache, got %d", atomic.LoadInt32(&executionCount))
	}
	if rr2.Header().Get("X-Cache-Lookup") != "HIT" {
		t.Errorf("expected X-Cache-Lookup: HIT")
	}
}

func TestIdempotencyMiddleware_ConcurrentInFlight(t *testing.T) {
	store := NewIdempotencyStore()
	var executionCount int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&executionCount, 1)
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	})

	wrapped := store.Middleware(1 * time.Minute)(handler)

	const concurrency = 10
	startGate := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(concurrency)

	var okCount int32
	var conflictCount int32

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			<-startGate

			req := httptest.NewRequest(http.MethodPost, "/api/test", bytes.NewBufferString(`{}`))
			req.Header.Set("Idempotency-Key", "stampede-key-1")
			rr := httptest.NewRecorder()
			wrapped.ServeHTTP(rr, req)

			switch rr.Code {
			case http.StatusOK:
				atomic.AddInt32(&okCount, 1)
			case http.StatusConflict:
				atomic.AddInt32(&conflictCount, 1)
			default:
				t.Errorf("unexpected status code: %d", rr.Code)
			}
		}()
	}

	close(startGate)
	wg.Wait()

	if got := atomic.LoadInt32(&executionCount); got != 1 {
		t.Fatalf("expected downstream handler to execute exactly once, got %d", got)
	}
	if got := atomic.LoadInt32(&okCount); got != 1 {
		t.Errorf("expected 1 OK response, got %d", got)
	}
	if got := atomic.LoadInt32(&conflictCount); got != concurrency-1 {
		t.Errorf("expected %d 409 Conflict responses, got %d", concurrency-1, got)
	}

	// Subsequent request after completion must return cached 200 OK with X-Cache-Lookup: HIT
	reqAfter := httptest.NewRequest(http.MethodPost, "/api/test", bytes.NewBufferString(`{}`))
	reqAfter.Header.Set("Idempotency-Key", "stampede-key-1")
	rrAfter := httptest.NewRecorder()
	wrapped.ServeHTTP(rrAfter, reqAfter)

	if rrAfter.Code != http.StatusOK {
		t.Fatalf("expected 200 on post-flight replay, got %d", rrAfter.Code)
	}
	if rrAfter.Header().Get("X-Cache-Lookup") != "HIT" {
		t.Errorf("expected X-Cache-Lookup: HIT on post-flight replay, got %q", rrAfter.Header().Get("X-Cache-Lookup"))
	}
	if got := atomic.LoadInt32(&executionCount); got != 1 {
		t.Errorf("expected execution count to remain 1, got %d", got)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	panickingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went critically wrong")
	})

	wrapped := RecoveryMiddleware(logger)(panickingHandler)

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rr.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := CORSMiddleware(baseHandler)

	// Preflight OPTIONS
	reqOpt := httptest.NewRequest(http.MethodOptions, "/api/data", nil)
	rrOpt := httptest.NewRecorder()
	wrapped.ServeHTTP(rrOpt, reqOpt)

	if rrOpt.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS, got %d", rrOpt.Code)
	}
	if rrOpt.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("missing or invalid Access-Control-Allow-Origin")
	}

	// Regular GET
	reqGet := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rrGet := httptest.NewRecorder()
	wrapped.ServeHTTP(rrGet, reqGet)

	if rrGet.Code != http.StatusOK {
		t.Errorf("expected 200 for GET, got %d", rrGet.Code)
	}
	if rrGet.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("missing or invalid Access-Control-Allow-Origin on GET")
	}
}
