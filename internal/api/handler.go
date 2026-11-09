// Package api exposes HTTP transport endpoints.
// It translates client transport requests into transit domain operations while enforcing protocol safety and response contracts.
package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golingko/internal/transit"
	"golingko/pkg/httputil"
)

// Dispatches HTTP transport requests to transit domain services.
// It serves as the transport gateway translating client JSON payloads and responses for domain services.
type Handler struct {
	cards    transit.CardService
	stations transit.StationService
	routes   transit.RouteService
	journeys transit.JourneyOrchestrator
	ledger   transit.LedgerService
	db       *sql.DB
	logger   *slog.Logger
}

func NewHandler(
	cards transit.CardService,
	stations transit.StationService,
	routes transit.RouteService,
	journeys transit.JourneyOrchestrator,
	ledger transit.LedgerService,
	db *sql.DB,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		cards:    cards,
		stations: stations,
		routes:   routes,
		journeys: journeys,
		ledger:   ledger,
		db:       db,
		logger:   logger,
	}
}

type HealthResponse struct {
	Status    string `json:"status"`
	Database  string `json:"database"`
	Timestamp string `json:"timestamp"`
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	dbStatus := "disconnected"
	overallStatus := "ok"

	if h.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := h.db.PingContext(ctx); err != nil {
			dbStatus = "unreachable"
			overallStatus = "degraded"
		} else {
			dbStatus = "connected"
		}
	}

	writeJSON(w, http.StatusOK, HealthResponse{
		Status:    overallStatus,
		Database:  dbStatus,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

type CreateCardRequest struct {
	// Optional 16-digit primary account number (PAN); auto-generated if left empty.
	CardNumber string `json:"card_number"`
	// Opening balance in IDR (Rp) credited upon card creation.
	InitialBalance int64 `json:"initial_balance"`
}

func (h *Handler) CreateCard(w http.ResponseWriter, r *http.Request) {
	if h.cards == nil {
		writeError(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	var req CreateCardRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request payload")
			return
		}
	}

	card, err := h.cards.CreateCard(r.Context(), req.CardNumber, req.InitialBalance)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, card)
}

func (h *Handler) GetCard(w http.ResponseWriter, r *http.Request) {
	if h.cards == nil {
		writeError(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	cardNumber := r.PathValue("number")
	if cardNumber == "" {
		writeError(w, http.StatusBadRequest, "card number is required")
		return
	}

	card, err := h.cards.GetCard(r.Context(), cardNumber)
	if err != nil {
		if errors.Is(err, transit.ErrCardNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, card)
}

type TopUpRequest struct {
	// Nominal value in IDR (Rp) to deposit into the card balance.
	Amount int64 `json:"amount"`
}

func (h *Handler) TopUpCard(w http.ResponseWriter, r *http.Request) {
	if h.cards == nil {
		writeError(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	cardNumber := r.PathValue("number")
	if cardNumber == "" {
		writeError(w, http.StatusBadRequest, "card number is required")
		return
	}

	var req TopUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var idempotencyKey *string
	if key := r.Header.Get("Idempotency-Key"); key != "" {
		idempotencyKey = &key
	}

	card, txRecord, err := h.cards.TopUp(r.Context(), cardNumber, req.Amount, idempotencyKey)
	if err != nil {
		if errors.Is(err, transit.ErrCardNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, transit.ErrCardInactive) || errors.Is(err, transit.ErrInsufficientAmount) {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"card":        card,
		"transaction": txRecord,
	})
}

func (h *Handler) GetCardTransactions(w http.ResponseWriter, r *http.Request) {
	if h.cards == nil || h.ledger == nil {
		writeError(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	cardNumber := r.PathValue("number")
	card, err := h.cards.GetCard(r.Context(), cardNumber)
	if err != nil {
		if errors.Is(err, transit.ErrCardNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	transactions, err := h.ledger.ListTransactions(r.Context(), card.ID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, transactions)
}

func (h *Handler) ListStations(w http.ResponseWriter, r *http.Request) {
	if h.stations == nil {
		writeError(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	var filter transit.StationFilter
	if m := strings.TrimSpace(r.URL.Query().Get("mode")); m != "" {
		tm := transit.TransitMode(strings.ToLower(m))
		filter.Mode = &tm
	}
	if c := strings.TrimSpace(r.URL.Query().Get("city")); c != "" {
		lowerCity := strings.ToLower(c)
		filter.City = &lowerCity
	}
	if d := strings.TrimSpace(r.URL.Query().Get("district")); d != "" {
		lowerDistrict := strings.ToLower(d)
		filter.District = &lowerDistrict
	}
	if route := strings.TrimSpace(r.URL.Query().Get("route_id")); route != "" {
		filter.RouteID = &route
	}

	stations, err := h.stations.ListStations(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stations)
}

func (h *Handler) GetStation(w http.ResponseWriter, r *http.Request) {
	if h.stations == nil {
		writeError(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "station id is required")
		return
	}

	station, err := h.stations.GetStation(r.Context(), id)
	if err != nil {
		if errors.Is(err, transit.ErrStationNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, transit.ErrStationInactive) {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, station)
}

func (h *Handler) ListRoutes(w http.ResponseWriter, r *http.Request) {
	if h.routes == nil {
		writeError(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	var filter transit.RouteFilter
	if m := strings.TrimSpace(r.URL.Query().Get("mode")); m != "" {
		tm := transit.TransitMode(strings.ToLower(m))
		filter.Mode = &tm
	}
	if c := strings.TrimSpace(r.URL.Query().Get("city")); c != "" {
		lowerCity := strings.ToLower(c)
		filter.City = &lowerCity
	}

	routes, err := h.routes.ListRoutes(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, routes)
}

func (h *Handler) GetRoute(w http.ResponseWriter, r *http.Request) {
	if h.routes == nil {
		writeError(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "route id is required")
		return
	}

	route, stops, err := h.routes.GetRoute(r.Context(), id)
	if err != nil {
		if errors.Is(err, transit.ErrRouteNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, transit.ErrRouteInactive) {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"route": route,
		"stops": stops,
	})
}

func (h *Handler) CheckRouteConnectivity(w http.ResponseWriter, r *http.Request) {
	if h.routes == nil {
		writeError(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	origin := strings.TrimSpace(r.URL.Query().Get("origin"))
	destination := strings.TrimSpace(r.URL.Query().Get("destination"))
	if origin == "" || destination == "" {
		writeError(w, http.StatusBadRequest, "origin and destination query parameters are required")
		return
	}

	conn, err := h.routes.CheckConnectivity(r.Context(), origin, destination)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, conn)
}

type TapInRequest struct {
	// 16-digit PAN scanned from the commuter's transit card.
	CardNumber string `json:"card_number"`
	// Unique identifier of the entry station or gate turnstile (e.g. "MRT-DKA").
	StationID string `json:"station_id"`
	// Optional operational transit corridor or vehicle run identifier.
	RouteID *string `json:"route_id,omitempty"`
}

func (h *Handler) TapIn(w http.ResponseWriter, r *http.Request) {
	if h.journeys == nil {
		writeError(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	var req TapInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	journey, err := h.journeys.TapIn(r.Context(), req.CardNumber, req.StationID, req.RouteID)
	if err != nil {
		if errors.Is(err, transit.ErrCardNotFound) || errors.Is(err, transit.ErrStationNotFound) || errors.Is(err, transit.ErrRouteNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, transit.ErrInsufficientBalance) {
			writeError(w, http.StatusPaymentRequired, err.Error())
			return
		}
		if errors.Is(err, transit.ErrAlreadyInJourney) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, transit.ErrCardInactive) || errors.Is(err, transit.ErrStationInactive) || errors.Is(err, transit.ErrRouteInactive) || errors.Is(err, transit.ErrStationNotInRoute) {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"gate_opened": true,
		"journey":     journey,
	})
}

type TapOutRequest struct {
	// 16-digit PAN scanned from the commuter's transit card.
	CardNumber string `json:"card_number"`
	// Unique identifier of the destination exit station or turnstile barrier.
	StationID string `json:"station_id"`
}

func (h *Handler) TapOut(w http.ResponseWriter, r *http.Request) {
	if h.journeys == nil {
		writeError(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}

	var req TapOutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	allowOverdraft := false
	result, err := h.journeys.TapOut(r.Context(), req.CardNumber, req.StationID, allowOverdraft)
	if err != nil {
		if errors.Is(err, transit.ErrCardNotFound) || errors.Is(err, transit.ErrStationNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, transit.ErrNoActiveJourney) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, transit.ErrInsufficientBalanceForExit) {
			writeError(w, http.StatusPaymentRequired, err.Error())
			return
		}
		if errors.Is(err, transit.ErrNoConnectingRoute) || errors.Is(err, transit.ErrStationNotInRoute) {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	httputil.WriteJSON(w, status, data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	httputil.WriteError(w, status, message)
}
