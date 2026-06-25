package router

import (
	"log"
	"net/http"

	"cgwm/battle/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func animaCurrent(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := currentUserID(c)
		log.Printf("[ANIMA_CURRENT] route=/api/v1/anima/current user=%d", userID)

		result, err := service.NewAnimaCurrentService(database).GetCurrent(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}