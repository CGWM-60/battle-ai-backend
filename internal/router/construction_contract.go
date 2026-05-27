package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ConstructionJobDTO is the canonical queue item returned by construction endpoints.
type ConstructionJobDTO struct {
	ID          string     `json:"id"`
	BuildingKey string     `json:"buildingKey"`
	BuildingID  string     `json:"buildingId,omitempty"`
	FromLevel   int        `json:"fromLevel"`
	TargetLevel int        `json:"targetLevel"`
	Type        string     `json:"type"`   // construct | upgrade
	Status      string     `json:"status"` // queued | in_progress | completed | cancelled
	QueuedAt    *time.Time `json:"queuedAt,omitempty"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// ConstructionQueueResponse is shared by all 6 endpoints to keep Flutter sync simple.
type ConstructionQueueResponse struct {
	Jobs        []ConstructionJobDTO `json:"jobs"`
	MaxTeams    int                  `json:"maxTeams"`
	ActiveTeams int                  `json:"activeTeams"`
	ServerNow   time.Time            `json:"serverNow"`
	Message     string               `json:"message,omitempty"`
	PlayerSave  map[string]any       `json:"playerSave"`
}

// StartConstructionRequest starts a new building construction.
type StartConstructionRequest struct {
	BuildingKey string `json:"buildingKey" binding:"required"`
	TargetLevel int    `json:"targetLevel" binding:"required,min=1"`
	PositionX   *int   `json:"positionX,omitempty"`
	PositionY   *int   `json:"positionY,omitempty"`
}

// ConstructionActionPathParams represent :id path usage for upgrade/speedup/cancel/complete.
type ConstructionActionPathParams struct {
	ID string `uri:"id" binding:"required"`
}

// registerConstructionContractRoutes exposes the 6 construction endpoints with exact JSON shape.
// You can replace the TODO handlers with service calls and keep the DTO contract unchanged.
func registerConstructionContractRoutes(private *gin.RouterGroup) {
	private.GET("/construction/queue", func(c *gin.Context) {
		c.JSON(http.StatusOK, contractNotImplementedResponse("queue"))
	})

	private.POST("/construction/start", func(c *gin.Context) {
		var input StartConstructionRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start construction payload"})
			return
		}
		c.JSON(http.StatusOK, contractNotImplementedResponse("start"))
	})

	private.POST("/construction/:id/upgrade", func(c *gin.Context) {
		var path ConstructionActionPathParams
		if err := c.ShouldBindUri(&path); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid construction id"})
			return
		}
		c.JSON(http.StatusOK, contractNotImplementedResponse("upgrade"))
	})

	private.POST("/construction/:id/speedup", func(c *gin.Context) {
		var path ConstructionActionPathParams
		if err := c.ShouldBindUri(&path); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid construction id"})
			return
		}
		c.JSON(http.StatusOK, contractNotImplementedResponse("speedup"))
	})

	private.POST("/construction/:id/cancel", func(c *gin.Context) {
		var path ConstructionActionPathParams
		if err := c.ShouldBindUri(&path); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid construction id"})
			return
		}
		c.JSON(http.StatusOK, contractNotImplementedResponse("cancel"))
	})

	private.POST("/construction/:id/complete", func(c *gin.Context) {
		var path ConstructionActionPathParams
		if err := c.ShouldBindUri(&path); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid construction id"})
			return
		}
		c.JSON(http.StatusOK, contractNotImplementedResponse("complete"))
	})
}

func contractNotImplementedResponse(action string) ConstructionQueueResponse {
	now := time.Now().UTC()
	return ConstructionQueueResponse{
		Jobs:        []ConstructionJobDTO{},
		MaxTeams:    3,
		ActiveTeams: 0,
		ServerNow:   now,
		Message:     "construction contract stub: " + action,
		PlayerSave: map[string]any{
			"constructionQueue": []any{},
			"buildings":         []any{},
			"activeEffects":     map[string]any{"maxTeams": 3},
		},
	}
}
