package api

import (
	"net/http"
)

type netHTTPRouterFactory struct{}

func (f *netHTTPRouterFactory) Build(handler *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handler.Health)

	mux.HandleFunc("POST /api/cards", handler.CreateCard)
	mux.HandleFunc("GET /api/cards/{number}", handler.GetCard)
	mux.HandleFunc("POST /api/cards/{number}/top-up", handler.TopUpCard)
	mux.HandleFunc("GET /api/cards/{number}/transactions", handler.GetCardTransactions)

	mux.HandleFunc("GET /api/stations", handler.ListStations)
	mux.HandleFunc("GET /api/stations/{id}", handler.GetStation)

	mux.HandleFunc("GET /api/routes", handler.ListRoutes)
	mux.HandleFunc("GET /api/routes/connectivity", handler.CheckRouteConnectivity)
	mux.HandleFunc("GET /api/routes/{id}", handler.GetRoute)

	mux.HandleFunc("POST /api/gate/tap-in", handler.TapIn)
	mux.HandleFunc("POST /api/gate/tap-out", handler.TapOut)

	mux.Handle("GET /docs/", http.StripPrefix("/docs/", http.FileServer(http.Dir("api/openapi"))))
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/", http.StatusPermanentRedirect)
	})

	return mux
}
