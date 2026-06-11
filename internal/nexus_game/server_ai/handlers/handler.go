package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"cgwm/battle/internal/nexus_game/models"
	saiservices "cgwm/battle/internal/nexus_game/server_ai/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	svc *saiservices.Service
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{svc: saiservices.NewService(db)}
}

func (h *Handler) Dashboard(c *gin.Context) {
	data, err := h.svc.Dashboard(c.Request.Context())
	write(c, data, err)
}

func (h *Handler) Jobs(c *gin.Context) {
	jobs, err := h.svc.ListJobs(c.Request.Context())
	write(c, gin.H{"jobs": jobs}, err)
}

func (h *Handler) RunDueJobs(c *gin.Context) {
	worldID := uintFromQuery(c, "world_id")
	results, err := h.svc.RunDueJobs(c.Request.Context(), "admin_due", worldID)
	write(c, gin.H{"results": results}, err)
}

func (h *Handler) RunAllJobs(c *gin.Context) {
	worldID := uintFromQuery(c, "world_id")
	results, err := h.svc.RunAllJobs(c.Request.Context(), "admin_all", worldID)
	write(c, gin.H{"results": results}, err)
}

func (h *Handler) RunJob(c *gin.Context) {
	var body struct {
		WorldID uint `json:"worldId"`
	}
	_ = c.ShouldBindJSON(&body)
	worldID := body.WorldID
	if worldID == 0 {
		worldID = uintFromQuery(c, "world_id")
	}
	result, err := h.svc.RunJob(c.Request.Context(), c.Param("jobKey"), "admin_manual", worldID)
	write(c, gin.H{"result": result}, err)
}

func (h *Handler) EnsureWorldCities(c *gin.Context) {
	worldID := uintFromParam(c, "worldId")
	if worldID == 0 {
		worldID = uintFromQuery(c, "world_id")
	}
	write(c, gin.H{"ok": true}, h.svc.EnsureCitiesForWorld(c.Request.Context(), worldID))
}

func (h *Handler) PublicCities(c *gin.Context) {
	cities, err := h.svc.ListCities(c.Request.Context(), true)
	write(c, gin.H{"cities": cities}, err)
}

func (h *Handler) AdminCities(c *gin.Context) {
	cities, err := h.svc.ListCities(c.Request.Context(), false)
	write(c, gin.H{"cities": cities}, err)
}

func (h *Handler) City(c *gin.Context) {
	city, err := h.svc.GetCity(c.Request.Context(), uintFromParam(c, "id"))
	write(c, gin.H{"city": city}, err)
}

func (h *Handler) UpdateCity(c *gin.Context) {
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	write(c, gin.H{"ok": true}, h.svc.UpdateCity(c.Request.Context(), uintFromParam(c, "id"), body))
}

func (h *Handler) DeleteCity(c *gin.Context) {
	write(c, gin.H{"ok": true}, h.svc.DeleteCity(c.Request.Context(), uintFromParam(c, "id")))
}

func (h *Handler) ThreatLevel(c *gin.Context) {
	data, err := h.svc.ThreatLevel(c.Request.Context(), uintFromQuery(c, "profile_gamer_id"))
	write(c, data, err)
}

func (h *Handler) PublicAttacks(c *gin.Context) {
	attacks, err := h.svc.ListAttacks(c.Request.Context(), uintFromQuery(c, "user_id"))
	write(c, gin.H{"attacks": attacks}, err)
}

func (h *Handler) PublicAttack(c *gin.Context) {
	attacks, err := h.svc.ListAttacks(c.Request.Context(), 0)
	if err != nil {
		write(c, nil, err)
		return
	}
	id := uintFromParam(c, "id")
	for _, attack := range attacks {
		if attack.ID == id {
			write(c, gin.H{"attack": attack}, nil)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "attack not found"})
}

func (h *Handler) AdminAttacks(c *gin.Context) {
	attacks, err := h.svc.ListAttacks(c.Request.Context(), uintFromQuery(c, "target_user_id"))
	write(c, gin.H{"attacks": attacks}, err)
}

func (h *Handler) ScheduleAttack(c *gin.Context) {
	var req saiservices.ScheduleAttackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	attack, err := h.svc.ScheduleAttack(c.Request.Context(), req)
	write(c, gin.H{"attack": attack}, err)
}

func (h *Handler) CancelAttack(c *gin.Context) {
	write(c, gin.H{"ok": true}, h.svc.CancelAttack(c.Request.Context(), uintFromParam(c, "id")))
}

func (h *Handler) ResolveAttack(c *gin.Context) {
	var body struct {
		Result string `json:"result"`
	}
	_ = c.ShouldBindJSON(&body)
	attack, err := h.svc.ResolveAttack(c.Request.Context(), uintFromParam(c, "id"), body.Result)
	write(c, gin.H{"attack": attack}, err)
}

func (h *Handler) DeleteAttack(c *gin.Context) {
	write(c, gin.H{"ok": true}, h.svc.CancelAttack(c.Request.Context(), uintFromParam(c, "id")))
}

func (h *Handler) Sabotages(c *gin.Context) {
	rows, err := h.svc.ListSabotages(c.Request.Context())
	write(c, gin.H{"sabotages": rows}, err)
}

func (h *Handler) CancelSabotage(c *gin.Context) {
	write(c, gin.H{"ok": true}, h.svc.CancelSabotage(c.Request.Context(), uintFromParam(c, "id")))
}

func (h *Handler) Espionage(c *gin.Context) {
	rows, err := h.svc.ListEspionage(c.Request.Context())
	write(c, gin.H{"espionage": rows}, err)
}

func (h *Handler) DeleteEspionage(c *gin.Context) {
	write(c, gin.H{"ok": true}, h.svc.DeleteEspionage(c.Request.Context(), uintFromParam(c, "id")))
}

func (h *Handler) DailyBroadcast(c *gin.Context) {
	broadcast, err := h.svc.DailyBroadcast(c.Request.Context())
	write(c, gin.H{"broadcast": broadcast}, err)
}

func (h *Handler) Broadcasts(c *gin.Context) {
	broadcasts, err := h.svc.ListBroadcasts(c.Request.Context())
	write(c, gin.H{"broadcasts": broadcasts}, err)
}

func (h *Handler) GenerateBroadcast(c *gin.Context) {
	broadcast, err := h.svc.GenerateBroadcast(c.Request.Context(), uintFromQuery(c, "world_id"))
	write(c, gin.H{"broadcast": broadcast}, err)
}

func (h *Handler) UpdateBroadcast(c *gin.Context) {
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	write(c, gin.H{"ok": true}, h.svc.UpdateBroadcast(c.Request.Context(), uintFromParam(c, "id"), body))
}

func (h *Handler) PublishBroadcast(c *gin.Context) {
	broadcast, err := h.svc.PublishBroadcast(c.Request.Context(), uintFromParam(c, "id"))
	write(c, gin.H{"broadcast": broadcast}, err)
}

func (h *Handler) DeleteBroadcast(c *gin.Context) {
	write(c, gin.H{"ok": true}, h.svc.DeleteBroadcast(c.Request.Context(), uintFromParam(c, "id")))
}

func (h *Handler) SeasonalProposals(c *gin.Context) {
	events, err := h.svc.ListSeasonalEvents(c.Request.Context(), []string{"draft", "proposed", "review"})
	write(c, gin.H{"events": events}, err)
}

func (h *Handler) ProposeSeasonalByAI(c *gin.Context) {
	event, err := h.svc.ProposeSeasonalEvent(c.Request.Context(), uintFromQuery(c, "world_id"))
	write(c, gin.H{"event": event}, err)
}

func (h *Handler) SeasonalActive(c *gin.Context) {
	events, err := h.svc.ListSeasonalEvents(c.Request.Context(), []string{"active"})
	write(c, gin.H{"events": events}, err)
}

func (h *Handler) SeasonalUpcoming(c *gin.Context) {
	events, err := h.svc.ListSeasonalEvents(c.Request.Context(), []string{"approved", "scheduled"})
	write(c, gin.H{"events": events}, err)
}

func (h *Handler) SeasonalList(c *gin.Context) {
	status := strings.TrimSpace(c.Query("status"))
	statuses := []string{}
	if status != "" && status != "all" {
		statuses = strings.Split(status, ",")
	}
	events, err := h.svc.ListSeasonalEvents(c.Request.Context(), statuses)
	write(c, gin.H{"events": events}, err)
}

func (h *Handler) SeasonalGet(c *gin.Context) {
	event, err := h.svc.GetSeasonalEvent(c.Request.Context(), uintFromParam(c, "id"))
	write(c, gin.H{"event": event}, err)
}

func (h *Handler) SeasonalUpdate(c *gin.Context) {
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	write(c, gin.H{"ok": true}, h.svc.UpdateSeasonalEvent(c.Request.Context(), uintFromParam(c, "id"), body))
}

func (h *Handler) SeasonalDelete(c *gin.Context) {
	write(c, gin.H{"ok": true}, h.svc.DeleteSeasonalEvent(c.Request.Context(), uintFromParam(c, "id")))
}

func (h *Handler) SeasonalTransition(status string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Reason string `json:"reason"`
		}
		_ = c.ShouldBindJSON(&body)
		event, err := h.svc.TransitionSeasonalEvent(c.Request.Context(), uintFromParam(c, "id"), status, body.Reason)
		write(c, gin.H{"event": event}, err)
	}
}

func (h *Handler) PlayerMemory(c *gin.Context) {
	memories, err := h.svc.ListPlayerMemory(c.Request.Context())
	write(c, gin.H{"playerMemory": memories}, err)
}

func (h *Handler) GlobalMemory(c *gin.Context) {
	memories, err := h.svc.GlobalMemory(c.Request.Context())
	write(c, gin.H{"memory": memories}, err)
}

func (h *Handler) DeletePlayerMemory(c *gin.Context) {
	write(c, gin.H{"ok": true}, h.svc.DeletePlayerMemory(c.Request.Context(), uintFromParam(c, "id")))
}

func (h *Handler) Prompts(c *gin.Context) {
	prompts, err := h.svc.ListPrompts(c.Request.Context())
	write(c, gin.H{"prompts": prompts}, err)
}

func (h *Handler) CreatePrompt(c *gin.Context) {
	var prompt models.Prompt
	if err := c.ShouldBindJSON(&prompt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.svc.CreatePrompt(c.Request.Context(), &prompt); err != nil {
		write(c, nil, err)
		return
	}
	write(c, gin.H{"prompt": prompt}, nil)
}

func (h *Handler) UpdatePrompt(c *gin.Context) {
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	write(c, gin.H{"ok": true}, h.svc.UpdatePrompt(c.Request.Context(), uintFromParam(c, "id"), body))
}

func (h *Handler) DeletePrompt(c *gin.Context) {
	write(c, gin.H{"ok": true}, h.svc.DeletePrompt(c.Request.Context(), uintFromParam(c, "id")))
}

func (h *Handler) TestPrompt(c *gin.Context) {
	result, err := h.svc.TestPrompt(c.Request.Context(), uintFromParam(c, "id"))
	write(c, gin.H{"result": result}, err)
}

func (h *Handler) CallLogs(c *gin.Context) {
	logs, err := h.svc.ListCallLogs(c.Request.Context(), intFromQuery(c, "limit", 100))
	write(c, gin.H{"logs": logs}, err)
}

func (h *Handler) Costs(c *gin.Context) {
	write(c, h.svc.Costs(c.Request.Context()), nil)
}

func (h *Handler) SeedPrompts(c *gin.Context) {
	write(c, gin.H{"ok": true}, h.svc.SeedPrompts(c.Request.Context()))
}

func write(c *gin.Context, payload any, err error) {
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if payload == nil {
		payload = gin.H{"ok": true}
	}
	c.JSON(http.StatusOK, payload)
}

func uintFromParam(c *gin.Context, name string) uint {
	v, _ := strconv.ParseUint(c.Param(name), 10, 64)
	return uint(v)
}

func uintFromQuery(c *gin.Context, name string) uint {
	v, _ := strconv.ParseUint(c.Query(name), 10, 64)
	return uint(v)
}

func intFromQuery(c *gin.Context, name string, fallback int) int {
	v, err := strconv.Atoi(c.Query(name))
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
