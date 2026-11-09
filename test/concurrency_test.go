package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"golingko/internal/transit"
)

func TestDoubleTap_RaceCondition(t *testing.T) {
	router, repo := SetupPartitionedApp()

	card, err := repo.CreateCard(context.Background(), nil, "1000999988887777", 50000, "active")
	if err != nil {
		t.Fatalf("failed to create card: %v", err)
	}

	concurrency := 10
	var successCount int64
	var conflictCount int64

	startGate := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			<-startGate

			payload := fmt.Sprintf(`{"card_number": "%s", "station_id": "MRT-DKA"}`, card.CardNumber)
			req := httptest.NewRequest(http.MethodPost, "/api/gate/tap-in", bytes.NewBufferString(payload))
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code == http.StatusOK {
				atomic.AddInt64(&successCount, 1)
			} else if rr.Code == http.StatusConflict {
				atomic.AddInt64(&conflictCount, 1)
			}
		}()
	}

	close(startGate)
	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful tap-in, got %d", successCount)
	}
	if conflictCount != int64(concurrency-1) {
		t.Fatalf("expected %d conflict rejections, got %d", concurrency-1, conflictCount)
	}
}

func TestConcurrentTopUp(t *testing.T) {
	router, repo := SetupPartitionedApp()

	card, err := repo.CreateCard(context.Background(), nil, "1000123456789012", 10000, "active")
	if err != nil {
		t.Fatalf("failed to create card: %v", err)
	}

	concurrency := 10
	topUpAmount := int64(10000)

	startGate := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			<-startGate

			payload := fmt.Sprintf(`{"amount": %d}`, topUpAmount)
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/cards/%s/top-up", card.CardNumber), bytes.NewBufferString(payload))
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("top-up failed with status %d", rr.Code)
			}
		}()
	}

	close(startGate)
	wg.Wait()

	updatedCard, err := repo.GetCardByNumber(context.Background(), nil, card.CardNumber)
	if err != nil {
		t.Fatalf("failed to get card: %v", err)
	}

	expectedBalance := int64(10000) + int64(concurrency)*topUpAmount
	if updatedCard.Balance != expectedBalance {
		t.Fatalf("expected balance %d, got %d", expectedBalance, updatedCard.Balance)
	}
}

func TestTransferWindow_IntegratedFare(t *testing.T) {
	router, _ := SetupPartitionedApp()

	reqCard := httptest.NewRequest(http.MethodPost, "/api/cards", bytes.NewBufferString(`{"initial_balance": 50000}`))
	rrCard := httptest.NewRecorder()
	router.ServeHTTP(rrCard, reqCard)
	if rrCard.Code != http.StatusCreated {
		t.Fatalf("failed to create card: %s", rrCard.Body.String())
	}
	var card transit.Card
	_ = json.Unmarshal(rrCard.Body.Bytes(), &card)

	doTapIn := func(stationID string) {
		payload := fmt.Sprintf(`{"card_number": "%s", "station_id": "%s"}`, card.CardNumber, stationID)
		req := httptest.NewRequest(http.MethodPost, "/api/gate/tap-in", bytes.NewBufferString(payload))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("tap in at %s failed: status %d, body %s", stationID, rr.Code, rr.Body.String())
		}
	}

	doTapOut := func(stationID string) transit.TapOutResult {
		payload := fmt.Sprintf(`{"card_number": "%s", "station_id": "%s"}`, card.CardNumber, stationID)
		req := httptest.NewRequest(http.MethodPost, "/api/gate/tap-out", bytes.NewBufferString(payload))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("tap out at %s failed: status %d, body %s", stationID, rr.Code, rr.Body.String())
		}
		var res transit.TapOutResult
		_ = json.Unmarshal(rr.Body.Bytes(), &res)
		return res
	}

	// Leg 1: TransJakarta Kota to Monas (Flat Rp 3,500)
	doTapIn("TJ-KOTA")
	leg1 := doTapOut("TJ-MNAS")
	if leg1.Fare.FareAmount != 3500 {
		t.Fatalf("leg 1: expected Rp 3,500, got %d", leg1.Fare.FareAmount)
	}

	// Leg 2: MRT Dukuh Atas to Lebak Bulus within transfer window (Integrated tariff)
	doTapIn("MRT-DKA")
	leg2 := doTapOut("MRT-BLB")
	if !leg2.Fare.IsIntegrated {
		t.Fatalf("leg 2: expected integrated fare flag")
	}

	// Leg 3: Further trip within transfer window that hits the Rp 10,000 cap
	doTapIn("MRT-BLB")
	leg3 := doTapOut("MRT-BHI")
	if !leg3.Fare.CapApplied {
		t.Fatalf("leg 3: expected cap applied flag")
	}

	// Leg 4: Fourth leg within window - cap already reached (Rp 0 free transfer)
	doTapIn("TJ-MNAS")
	leg4 := doTapOut("TJ-HRMN")
	if leg4.Fare.FareAmount != 0 {
		t.Fatalf("leg 4: expected Rp 0 after cap reached, got %d", leg4.Fare.FareAmount)
	}

	// Verify total deductions in ledger
	reqTxs := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/cards/%s/transactions", card.CardNumber), nil)
	rrTxs := httptest.NewRecorder()
	router.ServeHTTP(rrTxs, reqTxs)
	var txs []transit.LedgerTransaction
	_ = json.Unmarshal(rrTxs.Body.Bytes(), &txs)

	var totalDeductions int64
	for _, tx := range txs {
		if tx.TransactionType == transit.TxTypeFareDeduction {
			totalDeductions += -tx.Amount
		}
	}

	if totalDeductions > 10000 {
		t.Fatalf("expected total deductions capped at Rp 10,000, got %d", totalDeductions)
	}
}

func TestTransitLifecycle(t *testing.T) {
	router, repo := SetupPartitionedApp()

	// 1. Create card
	reqCard := httptest.NewRequest(http.MethodPost, "/api/cards", bytes.NewBufferString(`{"initial_balance": 0}`))
	rrCard := httptest.NewRecorder()
	router.ServeHTTP(rrCard, reqCard)
	if rrCard.Code != http.StatusCreated {
		t.Fatalf("create card failed: %d", rrCard.Code)
	}
	var card transit.Card
	_ = json.Unmarshal(rrCard.Body.Bytes(), &card)

	// 2. Top up card with Rp 50,000
	reqTopUp := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/cards/%s/top-up", card.CardNumber), bytes.NewBufferString(`{"amount": 50000}`))
	rrTopUp := httptest.NewRecorder()
	router.ServeHTTP(rrTopUp, reqTopUp)
	if rrTopUp.Code != http.StatusOK {
		t.Fatalf("top up failed: %d", rrTopUp.Code)
	}

	// 3. Tap in at MRT-DKA
	reqTapIn := httptest.NewRequest(http.MethodPost, "/api/gate/tap-in", bytes.NewBufferString(fmt.Sprintf(`{"card_number": "%s", "station_id": "MRT-DKA"}`, card.CardNumber)))
	rrTapIn := httptest.NewRecorder()
	router.ServeHTTP(rrTapIn, reqTapIn)
	if rrTapIn.Code != http.StatusOK {
		t.Fatalf("tap in failed: %d", rrTapIn.Code)
	}

	// 4. Tap out at MRT-BHI
	reqTapOut := httptest.NewRequest(http.MethodPost, "/api/gate/tap-out", bytes.NewBufferString(fmt.Sprintf(`{"card_number": "%s", "station_id": "MRT-BHI"}`, card.CardNumber)))
	rrTapOut := httptest.NewRecorder()
	router.ServeHTTP(rrTapOut, reqTapOut)
	if rrTapOut.Code != http.StatusOK {
		t.Fatalf("tap out failed: %d", rrTapOut.Code)
	}

	// 5. Verify final balance
	updatedCard, err := repo.GetCardByNumber(context.Background(), nil, card.CardNumber)
	if err != nil {
		t.Fatalf("failed to get card: %v", err)
	}
	if updatedCard.Balance >= 50000 {
		t.Fatalf("expected balance to be deducted after journey, got %d", updatedCard.Balance)
	}
}
