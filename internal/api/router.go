package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// Selects the multiplexer or framework driver used to serve transit HTTP endpoints.
//
// Callers configure this at boot time to switch between the Go standard library
// and third-party web frameworks without altering domain route definitions.
type RouterEngine string

const (
	RouterEngineNetHTTP RouterEngine = "nethttp"
	RouterEngineChi     RouterEngine = "chi"
	RouterEngineGin     RouterEngine = "gin"
)

type RouterFactory interface {
	Build(handler *Handler) http.Handler // Builds an HTTP handler
}

var routerFactories = map[RouterEngine]RouterFactory{
	RouterEngineNetHTTP: &netHTTPRouterFactory{},
	RouterEngineChi:     &chiRouterFactory{},
	RouterEngineGin:     &ginRouterFactory{},
}

// Normalizes configuration strings into recognized router engine constants.
// It accepts case-insensitive aliases and defaults to the standard library multiplexer when unspecified.
func ParseRouterEngine(raw string) (RouterEngine, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "nethttp", "net/http", "std", "stdlib", "":
		return RouterEngineNetHTTP, nil
	case "chi":
		return RouterEngineChi, nil
	case "gin":
		return RouterEngineGin, nil
	default:
		return "", fmt.Errorf("unsupported router engine: %q (supported: nethttp, chi, gin)", raw)
	}
}

// Instantiates the default transit HTTP router with the standard middleware pipeline.
// It configures telemetry, recovery, and CORS headers for public API consumers.
func SetupRouter(handler *Handler, logger *slog.Logger) http.Handler {
	router, err := SetupRouterWithEngine(RouterEngineNetHTTP, handler, logger)
	if err != nil {
		panic(err)
	}
	return router
}

// Assembles the transit HTTP router using the specified framework engine driver.
// It resolves the engine factory and wraps the route tree with standard production middlewares.
func SetupRouterWithEngine(engine RouterEngine, handler *Handler, logger *slog.Logger) (http.Handler, error) {
	factory, ok := routerFactories[engine]
	if !ok {
		return nil, fmt.Errorf("router engine %q not registered", engine)
	}

	raw := factory.Build(handler)
	return wrapMiddlewares(raw, logger), nil
}

// Extends the router factory registry with a custom engine implementation.
// It allows third-party multiplexers to be registered without altering core route definitions.
func RegisterRouterFactory(engine RouterEngine, factory RouterFactory) {
	routerFactories[engine] = factory
}
