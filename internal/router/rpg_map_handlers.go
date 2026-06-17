package router

import (
	"net/http"

	"cgwm/battle/internal/repository"
	"cgwm/battle/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type rpgMapMoveRequest struct {
	TargetNodeID string `json:"targetNodeId"`
}

type rpgMapActionRequest struct {
	ActionID   string `json:"actionId"`
	ActionType string `json:"actionType"`
	NodeID     string `json:"nodeId"`
}

type rpgMapRollRequest struct {
	ActionID   string `json:"actionId"`
	Attribute  string `json:"attribute"`
	Skill      string `json:"skill"`
	Difficulty int    `json:"difficulty"`
}

type rpgMapResolveRequest struct {
	ActionID    string `json:"actionId"`
	RollSuccess bool   `json:"rollSuccess"`
}

type rpgMapNPCRequest struct {
	NpcID string `json:"npcId"`
}

type rpgMapLootRequest struct {
	LootID string `json:"lootId"`
}

type rpgMapTrapRequest struct {
	TrapID string `json:"trapId"`
}

type rpgMapCombatRequest struct {
	CombatID string `json:"combatId"`
	ActionID string `json:"actionId"`
	TargetID string `json:"targetId"`
}

type rpgCoopVoteRequest struct {
	Accept bool `json:"accept"`
}

type rpgCoopSeparateRequest struct {
	TargetA string `json:"targetA"`
	TargetB string `json:"targetB"`
}

type rpgCoopMapActionRequest struct {
	ActionID   string `json:"actionId"`
	ActionType string `json:"actionType"`
	TargetID   string `json:"targetId"`
}

func newRPGMapService(database *gorm.DB) *service.RPGMapService {
	return service.NewRPGMapService(
		repository.NewRPGMapRepository(database),
		repository.NewRolePlayRepository(database),
		repository.NewQuestRepository(database),
		repository.NewCoopRepository(database),
	)
}

func getRPGMap(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		resp, err := newRPGMapService(database).GetMap(c.Request.Context(), sessionID, currentUserID(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func generateRPGMap(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		arcMap, state, err := newRPGMapService(database).GenerateArcMapForSession(c.Request.Context(), sessionID, currentUserID(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"arcMap": arcMap, "mapState": state})
	}
}

func enterRPGMap(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		resp, err := newRPGMapService(database).EnterMap(c.Request.Context(), sessionID, currentUserID(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func moveRPGMap(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		var req rpgMapMoveRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid move payload"})
			return
		}
		resp, err := newRPGMapService(database).MoveToNode(c.Request.Context(), sessionID, currentUserID(c), req.TargetNodeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func actionRPGMap(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		var req rpgMapActionRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action payload"})
			return
		}
		resp, err := newRPGMapService(database).RunAction(c.Request.Context(), sessionID, currentUserID(c), req.ActionID, req.ActionType, req.NodeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func rollRPGMap(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		var req rpgMapRollRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid roll payload"})
			return
		}
		if req.Difficulty <= 0 {
			req.Difficulty = 12
		}
		resp, err := newRPGMapService(database).Roll(c.Request.Context(), sessionID, currentUserID(c), req.ActionID, req.Attribute, req.Skill, req.Difficulty)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func resolveRPGMapAction(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		var req rpgMapResolveRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resolve payload"})
			return
		}
		resp, err := newRPGMapService(database).ResolveAction(c.Request.Context(), sessionID, currentUserID(c), req.ActionID, req.RollSuccess)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func interactRPGMapNPC(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		var req rpgMapNPCRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid npc payload"})
			return
		}
		resp, err := newRPGMapService(database).InteractNPC(c.Request.Context(), sessionID, currentUserID(c), req.NpcID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func lootRPGMap(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		var req rpgMapLootRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid loot payload"})
			return
		}
		resp, err := newRPGMapService(database).CollectLoot(c.Request.Context(), sessionID, currentUserID(c), req.LootID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func disarmRPGMapTrap(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		var req rpgMapTrapRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trap payload"})
			return
		}
		resp, err := newRPGMapService(database).DisarmTrap(c.Request.Context(), sessionID, currentUserID(c), req.TrapID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func combatActionRPGMap(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		var req rpgMapCombatRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid combat payload"})
			return
		}
		resp, err := newRPGMapService(database).CombatAction(c.Request.Context(), sessionID, currentUserID(c), req.CombatID, req.ActionID, req.TargetID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func fleeRPGMapCombat(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := parseUintParam(c, "id")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
			return
		}
		var req rpgMapCombatRequest
		_ = bindPayload(c, &req)
		resp, err := newRPGMapService(database).FleeCombat(c.Request.Context(), sessionID, currentUserID(c), req.CombatID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// --- Coop map handlers ---

func getCoopRPGMap(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		resp, err := newRPGMapService(database).GetOrCreateCoopMap(c.Request.Context(), code)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func enterCoopRPGMap(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		resp, err := newRPGMapService(database).EnterCoopMap(c.Request.Context(), code)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func coopRPGMapMove(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		var req rpgMapMoveRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid move payload"})
			return
		}
		resp, err := newRPGMapService(database).CoopProposeMove(c.Request.Context(), code, currentUserID(c), req.TargetNodeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func coopRPGMapVote(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		var req rpgCoopVoteRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid vote payload"})
			return
		}
		resp, err := newRPGMapService(database).CoopVote(c.Request.Context(), code, currentUserID(c), req.Accept)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func coopRPGMapSeparate(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		var req rpgCoopSeparateRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid separate payload"})
			return
		}
		resp, err := newRPGMapService(database).CoopProposeSeparate(c.Request.Context(), code, currentUserID(c), req.TargetA, req.TargetB)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func coopRPGMapLever(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		resp, err := newRPGMapService(database).CoopProposeLever(c.Request.Context(), code, currentUserID(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func coopRPGMapRegroup(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		resp, err := newRPGMapService(database).CoopProposeRegroup(c.Request.Context(), code, currentUserID(c))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func coopRPGMapAction(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		var req rpgCoopMapActionRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action payload"})
			return
		}
		resp, err := newRPGMapService(database).CoopMapAction(
			c.Request.Context(),
			code,
			currentUserID(c),
			req.ActionID,
			req.ActionType,
			req.TargetID,
		)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func coopRPGMapCombatAction(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Param("code")
		var req rpgMapCombatRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid combat payload"})
			return
		}
		resp, err := newRPGMapService(database).CoopCombatAction(c.Request.Context(), code, req.ActionID, req.TargetID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}