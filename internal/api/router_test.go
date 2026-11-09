package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golingko/internal/transit"
)

type mockCardService struct {
	getCardFunc func(ctx context.Context, cardNumber string) (*transit.Card, error)
	topUpFunc   func(ctx context.Context, cardNumber string, amount int64, idempotencyKey *string) (*transit.Card, *transit.LedgerTransaction, error)
}

func (m *mockCardService) CreateCard(ctx context.Context, cardNumber string, initialBalance int64) (*transit.Card, error) {
	return nil, nil
}

func (m *mockCardService) GetCard(ctx context.Context, cardNumber string) (*transit.Card, error) {
	if m.getCardFunc != nil {
		return m.getCardFunc(ctx, cardNumber)
	}
	return nil, transit.ErrCardNotFound
}

func (m *mockCardService) TopUp(ctx context.Context, cardNumber string, amount int64, idempotencyKey *string) (*transit.Card, *transit.LedgerTransaction, error) {
	if m.topUpFunc != nil {
		return m.topUpFunc(ctx, cardNumber, amount, idempotencyKey)
	}
	return nil, nil, nil
}

func TestParseRouterEngine(t *testing.T) {
	tests := []struct {
		input    string
		expected RouterEngine
		wantErr  bool
	}{
		{"nethttp", RouterEngineNetHTTP, false},
		{"net/http", RouterEngineNetHTTP, false},
		{"std", RouterEngineNetHTTP, false},
		{"stdlib", RouterEngineNetHTTP, false},
		{"", RouterEngineNetHTTP, false},
		{"  nethttp  ", RouterEngineNetHTTP, false},
		{"chi", RouterEngineChi, false},
		{"CHI", RouterEngineChi, false},
		{" gin ", RouterEngineGin, false},
		{"GIN", RouterEngineGin, false},
		{"fiber", "", true},
		{"echo", "", true},
	}

	for _, tc := range tests {
		actual, err := ParseRouterEngine(tc.input)
		if tc.wantErr && err == nil {
			t.Errorf("ParseRouterEngine(%q) expected error, got nil", tc.input)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("ParseRouterEngine(%q) unexpected error: %v", tc.input, err)
		}
		if actual != tc.expected {
			t.Errorf("ParseRouterEngine(%q) = %q, expected %q", tc.input, actual, tc.expected)
		}
	}
}

func TestSetupRouterWithEngine_Unknown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(nil, nil, nil, nil, nil, nil, logger)

	_, err := SetupRouterWithEngine("unknown-engine", handler, logger)
	if err == nil {
		t.Fatal("expected error when registering unknown router engine, got nil")
	}
}

func TestRouterEngines_Parity(t *testing.T) {
	engines := []RouterEngine{
		RouterEngineNetHTTP,
		RouterEngineChi,
		RouterEngineGin,
	}

	for _, engine := range engines {
		t.Run(string(engine), func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			handler := NewHandler(nil, nil, nil, nil, nil, nil, logger)

			router, err := SetupRouterWithEngine(engine, handler, logger)
			if err != nil {
				t.Fatalf("failed to build router for engine %s: %v", engine, err)
			}

			t.Run("GET /health", func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/health", nil)
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)
				if rr.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d", rr.Code)
				}

				var resp HealthResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if resp.Status != "ok" {
					t.Errorf("expected status 'ok', got %q", resp.Status)
				}
				if resp.Database != "disconnected" {
					t.Errorf("expected database 'disconnected', got %q", resp.Database)
				}
			})

			t.Run("GET /docs redirect", func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/docs", nil)
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)
				if rr.Code != http.StatusPermanentRedirect {
					t.Fatalf("expected 308, got %d", rr.Code)
				}
			})

			t.Run("OPTIONS CORS preflight", func(t *testing.T) {
				req := httptest.NewRequest(http.MethodOptions, "/health", nil)
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)
				if rr.Code != http.StatusNoContent {
					t.Fatalf("expected 204, got %d", rr.Code)
				}
				if origin := rr.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
					t.Errorf("expected Access-Control-Allow-Origin '*', got %q", origin)
				}
			})

			t.Run("POST /api/cards service unavailable", func(t *testing.T) {
				body := bytes.NewBufferString(`{"initial_balance": 50000}`)
				req := httptest.NewRequest(http.MethodPost, "/api/cards", body)
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)
				if rr.Code != http.StatusServiceUnavailable {
					t.Fatalf("expected 503, got %d", rr.Code)
				}
			})

			t.Run("POST /api/gate/tap-in service unavailable", func(t *testing.T) {
				body := bytes.NewBufferString(`{"card_number": "1000123456789012", "station_id": "MRT-DKA"}`)
				req := httptest.NewRequest(http.MethodPost, "/api/gate/tap-in", body)
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)
				if rr.Code != http.StatusServiceUnavailable {
					t.Fatalf("expected 503, got %d", rr.Code)
				}
			})

			t.Run("POST /api/gate/tap-out service unavailable", func(t *testing.T) {
				body := bytes.NewBufferString(`{"card_number": "1000123456789012", "station_id": "MRT-BHI"}`)
				req := httptest.NewRequest(http.MethodPost, "/api/gate/tap-out", body)
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)
				if rr.Code != http.StatusServiceUnavailable {
					t.Fatalf("expected 503, got %d", rr.Code)
				}
			})

			t.Run("Idempotency Middleware", func(t *testing.T) {
				req1 := httptest.NewRequest(http.MethodGet, "/health", nil)
				req1.Header.Set("Idempotency-Key", "test-key-1")
				rr1 := httptest.NewRecorder()

				router.ServeHTTP(rr1, req1)
				if rr1.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d", rr1.Code)
				}

				req2 := httptest.NewRequest(http.MethodGet, "/health", nil)
				req2.Header.Set("Idempotency-Key", "test-key-1")
				rr2 := httptest.NewRecorder()

				router.ServeHTTP(rr2, req2)
				if rr2.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d", rr2.Code)
				}
				if rr2.Body.String() != rr1.Body.String() {
					t.Errorf("expected identical body from idempotency cache")
				}
			})
		})
	}
}

func TestRouterEngines_PathParameterHandling(t *testing.T) {
	engines := []RouterEngine{
		RouterEngineNetHTTP,
		RouterEngineChi,
		RouterEngineGin,
	}

	const expectedCardNumber = "1000999988887777"

	for _, engine := range engines {
		t.Run(string(engine), func(t *testing.T) {
			var capturedCardNumber string
			mockCards := &mockCardService{
				getCardFunc: func(ctx context.Context, cardNumber string) (*transit.Card, error) {
					capturedCardNumber = cardNumber
					return &transit.Card{
						ID:         "card-123",
						CardNumber: cardNumber,
						Balance:    75000,
						Status:     transit.CardStatusActive,
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					}, nil
				},
				topUpFunc: func(ctx context.Context, cardNumber string, amount int64, idempotencyKey *string) (*transit.Card, *transit.LedgerTransaction, error) {
					capturedCardNumber = cardNumber
					return &transit.Card{
							ID:         "card-123",
							CardNumber: cardNumber,
							Balance:    75000 + amount,
							Status:     transit.CardStatusActive,
						}, &transit.LedgerTransaction{
							ID:     "tx-123",
							CardID: "card-123",
							Amount: amount,
						}, nil
				},
			}

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			handler := NewHandler(mockCards, nil, nil, nil, nil, nil, logger)

			router, err := SetupRouterWithEngine(engine, handler, logger)
			if err != nil {
				t.Fatalf("failed to build router for engine %s: %v", engine, err)
			}

			t.Run("GET /api/cards/{number}", func(t *testing.T) {
				capturedCardNumber = ""
				req := httptest.NewRequest(http.MethodGet, "/api/cards/"+expectedCardNumber, nil)
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)
				if rr.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d, body: %s", rr.Code, rr.Body.String())
				}

				if capturedCardNumber != expectedCardNumber {
					t.Fatalf("expected captured card number %q, got %q", expectedCardNumber, capturedCardNumber)
				}

				var card transit.Card
				if err := json.Unmarshal(rr.Body.Bytes(), &card); err != nil {
					t.Fatalf("failed to decode card json: %v", err)
				}
				if card.CardNumber != expectedCardNumber {
					t.Errorf("expected card number %q in body, got %q", expectedCardNumber, card.CardNumber)
				}
			})

			t.Run("POST /api/cards/{number}/top-up", func(t *testing.T) {
				capturedCardNumber = ""
				body := bytes.NewBufferString(`{"amount": 25000}`)
				req := httptest.NewRequest(http.MethodPost, "/api/cards/"+expectedCardNumber+"/top-up", body)
				rr := httptest.NewRecorder()

				router.ServeHTTP(rr, req)
				if rr.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d, body: %s", rr.Code, rr.Body.String())
				}

				if capturedCardNumber != expectedCardNumber {
					t.Fatalf("expected captured card number %q in top-up, got %q", expectedCardNumber, capturedCardNumber)
				}
			})
		})
	}
}
