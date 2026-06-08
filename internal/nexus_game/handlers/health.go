package handlers

import (
	"net/http"
	"runtime"

	"cgwm/battle/internal/nexus_game/cache"
	"cgwm/battle/internal/nexus_game/dto"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db    *gorm.DB
	redis *cache.RedisService
}

func NewHealthHandler(db *gorm.DB, redis *cache.RedisService) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

func (h *HealthHandler) Health(c *gin.Context) {
	databaseOK := h.databaseOK()
	statusCode := http.StatusOK
	if !databaseOK {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, dto.OK(gin.H{
		"module":   "nexus_game",
		"status":   statusText(databaseOK),
		"database": databaseOK,
		"redis":    h.redis.Status(c.Request.Context()),
	}))
}

func (h *HealthHandler) DebugStatus(c *gin.Context) {
	c.JSON(http.StatusOK, dto.OK(gin.H{
		"module":      "nexus_game",
		"status":      statusText(h.databaseOK()),
		"database":    h.databaseOK(),
		"redis":       h.redis.Status(c.Request.Context()),
		"go_version":  runtime.Version(),
		"go_os":       runtime.GOOS,
		"go_arch":     runtime.GOARCH,
		"routes":      []string{"/api/nexus-game/health", "/api/nexus-game/debug/status"},
		"translation": "server_driven",
	}))
}

func (h *HealthHandler) databaseOK() bool {
	if h.db == nil {
		return false
	}
	sqlDB, err := h.db.DB()
	if err != nil {
		return false
	}
	return sqlDB.Ping() == nil
}

func statusText(ok bool) string {
	if ok {
		return "ok"
	}
	return "degraded"
}
