package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type chiRouterFactory struct{}

func (f *chiRouterFactory) Build(handler *Handler) http.Handler {
	r := chi.NewRouter()

	r.Get("/health", handler.Health)

	r.Post("/api/cards", handler.CreateCard)
	r.Get("/api/cards/{number}", adaptChi(handler.GetCard))
	r.Post("/api/cards/{number}/top-up", adaptChi(handler.TopUpCard))
	r.Get("/api/cards/{number}/transactions", adaptChi(handler.GetCardTransactions))

	r.Get("/api/stations", handler.ListStations)
	r.Get("/api/stations/{id}", adaptChi(handler.GetStation))

	r.Get("/api/routes", handler.ListRoutes)
	r.Get("/api/routes/connectivity", handler.CheckRouteConnectivity)
	r.Get("/api/routes/{id}", adaptChi(handler.GetRoute))

	r.Post("/api/gate/tap-in", handler.TapIn)
	r.Post("/api/gate/tap-out", handler.TapOut)

	r.Handle("/docs/*", http.StripPrefix("/docs/", http.FileServer(http.Dir("api/openapi"))))
	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/", http.StatusPermanentRedirect)
	})

	return r
}

func adaptChi(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		if rctx != nil {
			for i, key := range rctx.URLParams.Keys {
				r.SetPathValue(key, rctx.URLParams.Values[i])
			}
		}
		h(w, r)
	}
}
