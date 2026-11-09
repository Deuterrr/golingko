package test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func BenchmarkCreateCard(b *testing.B) {
	router, _ := SetupPartitionedApp()
	payload := []byte(`{"initial_balance": 10000}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/cards", bytes.NewReader(payload))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			b.Fatalf("expected 201 Created, got %d", rr.Code)
		}
	}
}

func BenchmarkTopUp(b *testing.B) {
	router, repo := SetupPartitionedApp()
	card, err := repo.CreateCard(b.Context(), nil, "1000111122223333", 10000, "active")
	if err != nil {
		b.Fatalf("failed to create card: %v", err)
	}
	payload := []byte(`{"amount": 25000}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/cards/%s/top-up", card.CardNumber), bytes.NewReader(payload))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			b.Fatalf("expected 200 OK, got %d", rr.Code)
		}
	}
}

func BenchmarkListStations(b *testing.B) {
	router, _ := SetupPartitionedApp()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/stations?mode=mrt", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			b.Fatalf("expected 200 OK, got %d", rr.Code)
		}
	}
}

func BenchmarkTapIn(b *testing.B) {
	router, repo := SetupPartitionedApp()
	var cardCounter int64

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		id := atomic.AddInt64(&cardCounter, 1)
		cardNum := fmt.Sprintf("1000%012d", id)
		_, _ = repo.CreateCard(b.Context(), nil, cardNum, 50000, "active")
		payload := fmt.Sprintf(`{"card_number": "%s", "station_id": "MRT-DKA"}`, cardNum)
		b.StartTimer()

		req := httptest.NewRequest(http.MethodPost, "/api/gate/tap-in", bytes.NewBufferString(payload))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			b.Fatalf("expected 200 OK, got %d", rr.Code)
		}
	}
}

func BenchmarkTapOut(b *testing.B) {
	router, repo := SetupPartitionedApp()
	var cardCounter int64

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		id := atomic.AddInt64(&cardCounter, 1)
		cardNum := fmt.Sprintf("1000%012d", id)
		card, _ := repo.CreateCard(b.Context(), nil, cardNum, 50000, "active")
		_, _ = repo.CreateJourney(b.Context(), nil, card.ID, "MRT-DKA", nil)
		payload := fmt.Sprintf(`{"card_number": "%s", "station_id": "MRT-BLB"}`, cardNum)
		b.StartTimer()

		req := httptest.NewRequest(http.MethodPost, "/api/gate/tap-out", bytes.NewBufferString(payload))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			b.Fatalf("expected 200 OK, got %d", rr.Code)
		}
	}
}
