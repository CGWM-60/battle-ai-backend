package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeadersCacheNextStaticAssets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(securityHeaders())
	engine.GET("/*path", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "next static asset",
			path: "/admin/_next/static/chunks/app.js",
			want: "public, max-age=31536000, immutable",
		},
		{
			name: "admin html",
			path: "/admin/game/",
			want: "no-store",
		},
		{
			name: "api response",
			path: "/api/v1/world",
			want: "no-store",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			res := httptest.NewRecorder()

			engine.ServeHTTP(res, req)

			if got := res.Header().Get("Cache-Control"); got != tt.want {
				t.Fatalf("Cache-Control = %q, want %q", got, tt.want)
			}
		})
	}
}
