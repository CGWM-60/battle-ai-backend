package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutesMountsHealthAndDebug(t *testing.T) {
	t.Setenv("REDIS_URL", "")
	gin.SetMode(gin.TestMode)

	router := gin.New()
	RegisterRoutes(router, nil)

	for _, path := range []string{
		"/api/nexus-game/health",
		"/api/nexus-game/debug/status",
		"/api/nexus-game/worlds/1",
		"/api/nexus-game/worlds/1/players",
	} {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		router.ServeHTTP(recorder, req)
		if recorder.Code == http.StatusNotFound {
			t.Fatalf("%s was not mounted", path)
		}
	}
}
