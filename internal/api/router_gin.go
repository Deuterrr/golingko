package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ginRouterFactory struct{}

func (f *ginRouterFactory) Build(handler *Handler) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	engine.GET("/health", adaptGin(handler.Health))

	engine.POST("/api/cards", adaptGin(handler.CreateCard))
	engine.GET("/api/cards/:number", adaptGin(handler.GetCard))
	engine.POST("/api/cards/:number/top-up", adaptGin(handler.TopUpCard))
	engine.GET("/api/cards/:number/transactions", adaptGin(handler.GetCardTransactions))

	engine.GET("/api/stations", adaptGin(handler.ListStations))
	engine.GET("/api/stations/:id", adaptGin(handler.GetStation))

	engine.GET("/api/routes", adaptGin(handler.ListRoutes))
	engine.GET("/api/routes/connectivity", adaptGin(handler.CheckRouteConnectivity))
	engine.GET("/api/routes/:id", adaptGin(handler.GetRoute))

	engine.POST("/api/gate/tap-in", adaptGin(handler.TapIn))
	engine.POST("/api/gate/tap-out", adaptGin(handler.TapOut))

	engine.GET("/docs", func(c *gin.Context) {
		c.Redirect(http.StatusPermanentRedirect, "/docs/")
	})

	docsFS := http.StripPrefix("/docs/", http.FileServer(http.Dir("api/openapi")))
	engine.GET("/docs/*filepath", func(c *gin.Context) {
		docsFS.ServeHTTP(c.Writer, c.Request)
	})

	return engine
}

func adaptGin(h http.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, p := range c.Params {
			c.Request.SetPathValue(p.Key, p.Value)
		}
		h(c.Writer, c.Request)
	}
}
