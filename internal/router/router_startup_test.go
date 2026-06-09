package router

import (
	"testing"

	"cgwm/battle/internal/admin"
	nexusroutes "cgwm/battle/internal/nexus_game/routes"
	"github.com/gin-gonic/gin"
)

func TestAdminStaticDoesNotConflictWithLoginRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("admin route registration panicked: %v", recovered)
		}
	}()

	router := gin.New()
	nexusroutes.RegisterAdminStatic(router)
	admin.Register(router, nil)
}
