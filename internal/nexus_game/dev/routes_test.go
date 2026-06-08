package dev

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDevBackdoorDisabledByDefault(t *testing.T) {
	t.Setenv(envEnabled, "")
	t.Setenv(envToken, "")
	gin.SetMode(gin.TestMode)

	router := gin.New()
	RegisterRoutes(router)

	status := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/nexus-game/dev/status", nil)
	router.ServeHTTP(status, req)
	if status.Code != http.StatusOK {
		t.Fatalf("status code=%d, want 200", status.Code)
	}

	action := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/nexus-game/dev/login", nil)
	router.ServeHTTP(action, req)
	if action.Code != http.StatusForbidden {
		t.Fatalf("disabled action code=%d, want 403", action.Code)
	}
}

func TestDevBackdoorRequiresTokenWhenEnabled(t *testing.T) {
	t.Setenv(envEnabled, "true")
	t.Setenv(envToken, "secret")
	gin.SetMode(gin.TestMode)

	router := gin.New()
	RegisterRoutes(router)

	unauthorized := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/nexus-game/dev/login", nil)
	router.ServeHTTP(unauthorized, req)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("missing token code=%d, want 401", unauthorized.Code)
	}

	authorized := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/nexus-game/dev/login", nil)
	req.Header.Set("X-Nexus-Dev-Token", "secret")
	router.ServeHTTP(authorized, req)
	if authorized.Code != http.StatusOK {
		t.Fatalf("valid token code=%d, want 200 body=%s", authorized.Code, authorized.Body.String())
	}
}
