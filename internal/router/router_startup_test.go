package router

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRouterAppDoesNotPanicOnAdminRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("RouterApp panicked while registering routes: %v", recovered)
		}
	}()

	RouterApp(nil)
}
