package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golingko/internal/api"
	"golingko/pkg/logger"
)

func BenchmarkRouter_NetHTTP(b *testing.B) {
	router, _ := SetupPartitionedAppWithEngine(api.RouterEngineNetHTTP, logger.NewNoopLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/stations", nil)
	rr := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rr.Body.Reset()
		router.ServeHTTP(rr, req)
	}
}

func BenchmarkRouter_Chi(b *testing.B) {
	router, _ := SetupPartitionedAppWithEngine(api.RouterEngineChi, logger.NewNoopLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/stations", nil)
	rr := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rr.Body.Reset()
		router.ServeHTTP(rr, req)
	}
}

func BenchmarkRouter_Gin(b *testing.B) {
	router, _ := SetupPartitionedAppWithEngine(api.RouterEngineGin, logger.NewNoopLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/stations", nil)
	rr := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rr.Body.Reset()
		router.ServeHTTP(rr, req)
	}
}
