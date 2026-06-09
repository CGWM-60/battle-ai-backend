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
		"/api/nexus-game/world-players",
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

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/nexus-game/worlds/repair-player-assignments", nil)
	router.ServeHTTP(recorder, req)
	if recorder.Code == http.StatusNotFound {
		t.Fatal("/api/nexus-game/worlds/repair-player-assignments was not mounted")
	}
}

func TestRegisterRoutesMountsContentCRUDRoutes(t *testing.T) {
	t.Setenv("REDIS_URL", "")
	gin.SetMode(gin.TestMode)

	router := gin.New()
	RegisterRoutes(router, nil)

	mounted := make(map[string]bool)
	for _, route := range router.Routes() {
		mounted[route.Method+" "+route.Path] = true
	}

	expected := []string{
		"GET /api/nexus-game/admin/content/buildings",
		"GET /api/nexus-game/admin/content/buildings/page",
		"GET /api/nexus-game/admin/content/buildings/:contentId",
		"GET /api/nexus-game/admin/content/translations/status",
		"POST /api/nexus-game/admin/content/buildings",
		"PUT /api/nexus-game/admin/content/buildings/:contentId",
		"DELETE /api/nexus-game/admin/content/buildings/:contentId",
		"POST /api/nexus-game/admin/content/buildings/:contentId/delete",
		"GET /api/nexus-game/admin/content/units",
		"GET /api/nexus-game/admin/content/units/page",
		"GET /api/nexus-game/admin/content/units/:contentId",
		"POST /api/nexus-game/admin/content/units",
		"PUT /api/nexus-game/admin/content/units/:contentId",
		"DELETE /api/nexus-game/admin/content/units/:contentId",
		"POST /api/nexus-game/admin/content/units/:contentId/delete",
		"GET /api/nexus-game/admin/content/research",
		"GET /api/nexus-game/admin/content/research/page",
		"GET /api/nexus-game/admin/content/research/:contentId",
		"POST /api/nexus-game/admin/content/research",
		"PUT /api/nexus-game/admin/content/research/:contentId",
		"DELETE /api/nexus-game/admin/content/research/:contentId",
		"POST /api/nexus-game/admin/content/research/:contentId/delete",
	}

	for _, route := range expected {
		if !mounted[route] {
			t.Fatalf("%s was not mounted", route)
		}
	}
}
