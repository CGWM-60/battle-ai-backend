package cgwm

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"cgwm/battle/internal/cgwm/handler"
	"cgwm/battle/internal/cgwm/realtime"
	"cgwm/battle/internal/cgwm/scheduler"
)

func RegisterCGWMRoutes(r *gin.Engine, database *gorm.DB) {
	// User protected routes (add auth middleware)
	cgwm := r.Group("/api/cgwm")
	{
		cgwm.PUT("/animas/:id/snapshot", handler.UploadSnapshot)
		cgwm.GET("/animas/:id/snapshot", handler.DownloadSnapshot)
		cgwm.POST("/park/enter", handler.EnterPark)
		cgwm.POST("/park/leave-alone", handler.LeaveAlone)
		cgwm.POST("/park/heartbeat", handler.Heartbeat)
		cgwm.POST("/park/return", handler.ReturnFromPark)
		cgwm.POST("/social/cards", handler.SubmitSocialCard)
	}

	// Admin
	admin := r.Group("/api/admin/cgwm")
	{
		admin.GET("/sync-state", handler.AdminSyncState)
		admin.GET("/park-state", func(c *gin.Context) { handler.AdminParkState(c, database) })
	}

	// Realtime
	r.GET("/ws/cgwm/park", realtime.ParkWSHandler) // implement WS upgrade using ParkHub

	// Start schedulers on init
	scheduler.StartAloneLearningScheduler()
}