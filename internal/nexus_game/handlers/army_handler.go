package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"cgwm/battle/internal/nexus_game/models"
	"cgwm/battle/internal/nexus_game/services"

	"github.com/gin-gonic/gin"
)

type ArmyHandler struct {
	svc *services.ArmyService
}

func NewArmyHandler(svc *services.ArmyService) *ArmyHandler {
	return &ArmyHandler{svc: svc}
}

func (h *ArmyHandler) Catalog(c *gin.Context) {
	catalog, err := h.svc.Catalog(c.Request.Context(), true)
	writeArmy(c, gin.H{"units": catalog, "count": len(catalog)}, err)
}

func (h *ArmyHandler) PlayerUnits(c *gin.Context) {
	profileID, ok := armyProfileID(c)
	if !ok {
		return
	}
	units, err := h.svc.PlayerUnits(c.Request.Context(), profileID)
	writeArmy(c, gin.H{"units": units}, err)
}

func (h *ArmyHandler) TrainingQueue(c *gin.Context) {
	profileID, ok := armyProfileID(c)
	if !ok {
		return
	}
	queue, err := h.svc.TrainingQueue(c.Request.Context(), profileID, c.Query("includeDone") == "true")
	writeArmy(c, gin.H{"trainingQueue": queue}, err)
}

func (h *ArmyHandler) Train(c *gin.Context) {
	var req services.ArmyTrainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "errorCode": "invalid_payload", "message": err.Error()})
		return
	}
	if req.ProfileGamerID == 0 {
		req.ProfileGamerID = armyProfileIDFromQuery(c)
	}
	queue, capacity, err := h.svc.TrainUnit(c.Request.Context(), req)
	writeArmy(c, gin.H{"success": true, "trainingQueue": queue, "capacity": capacity}, err)
}

func (h *ArmyHandler) CancelTraining(c *gin.Context) {
	profileID, ok := armyProfileID(c)
	if !ok {
		return
	}
	queue, err := h.svc.CancelTraining(c.Request.Context(), profileID, uintFromParam(c, "id"))
	writeArmy(c, gin.H{"success": true, "trainingQueue": queue}, err)
}

func (h *ArmyHandler) ClaimTraining(c *gin.Context) {
	profileID, ok := armyProfileID(c)
	if !ok {
		return
	}
	unit, err := h.svc.ClaimTraining(c.Request.Context(), profileID, uintFromParam(c, "id"))
	writeArmy(c, gin.H{"success": true, "unit": unit}, err)
}

func (h *ArmyHandler) Formations(c *gin.Context) {
	profileID, ok := armyProfileID(c)
	if !ok {
		return
	}
	formations, err := h.svc.Formations(c.Request.Context(), profileID)
	writeArmy(c, gin.H{"formations": formations}, err)
}

func (h *ArmyHandler) Formation(c *gin.Context) {
	profileID, ok := armyProfileID(c)
	if !ok {
		return
	}
	detail, err := h.svc.Formation(c.Request.Context(), profileID, uintFromParam(c, "id"))
	writeArmy(c, gin.H{"formation": detail}, err)
}

func (h *ArmyHandler) Assign(c *gin.Context) {
	var req services.ArmyAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "errorCode": "invalid_payload", "message": err.Error()})
		return
	}
	profileID := req.ProfileGamerID
	if profileID == 0 {
		profileID = armyProfileIDFromQuery(c)
	}
	if profileID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "errorCode": "profile_required", "message": "profileGamerId est requis."})
		return
	}
	detail, err := h.svc.AssignUnit(c.Request.Context(), profileID, uintFromParam(c, "id"), uintFromParam(c, "slotId"), req)
	writeArmy(c, gin.H{"success": true, "formation": detail}, err)
}

func (h *ArmyHandler) Remove(c *gin.Context) {
	var req services.ArmyAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "errorCode": "invalid_payload", "message": err.Error()})
		return
	}
	profileID := req.ProfileGamerID
	if profileID == 0 {
		profileID = armyProfileIDFromQuery(c)
	}
	if profileID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "errorCode": "profile_required", "message": "profileGamerId est requis."})
		return
	}
	detail, err := h.svc.RemoveUnit(c.Request.Context(), profileID, uintFromParam(c, "id"), uintFromParam(c, "slotId"), req)
	writeArmy(c, gin.H{"success": true, "formation": detail}, err)
}

func (h *ArmyHandler) ValidateFormation(c *gin.Context) {
	profileID, ok := armyProfileID(c)
	if !ok {
		return
	}
	detail, err := h.svc.Formation(c.Request.Context(), profileID, uintFromParam(c, "id"))
	writeArmy(c, gin.H{"success": err == nil, "formation": detail}, err)
}

func (h *ArmyHandler) CommanderSuggest(c *gin.Context) {
	profileID, ok := armyProfileID(c)
	if !ok {
		return
	}
	result, err := h.svc.CommanderSuggest(c.Request.Context(), profileID, uintFromParam(c, "id"))
	writeArmy(c, result, err)
}

func (h *ArmyHandler) Automation(c *gin.Context) {
	profileID, ok := armyProfileID(c)
	if !ok {
		return
	}
	settings, err := h.svc.Automation(c.Request.Context(), profileID)
	writeArmy(c, gin.H{"automation": settings}, err)
}

func (h *ArmyHandler) SaveAutomation(c *gin.Context) {
	var patch models.ArmyAutomationSettings
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "errorCode": "invalid_payload", "message": err.Error()})
		return
	}
	profileID := patch.ProfileGamerID
	if profileID == 0 {
		profileID = armyProfileIDFromQuery(c)
	}
	if profileID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "errorCode": "profile_required", "message": "profileGamerId est requis."})
		return
	}
	settings, err := h.svc.SaveAutomation(c.Request.Context(), profileID, patch)
	writeArmy(c, gin.H{"automation": settings}, err)
}

func (h *ArmyHandler) CombatReports(c *gin.Context) {
	profileID, ok := armyProfileID(c)
	if !ok {
		return
	}
	reports, err := h.svc.CombatReports(c.Request.Context(), profileID, limitFromQuery(c, 50, 200))
	writeArmy(c, gin.H{"combatReports": reports}, err)
}

func (h *ArmyHandler) AdminCatalog(c *gin.Context) {
	catalog, err := h.svc.Catalog(c.Request.Context(), false)
	writeArmy(c, gin.H{"units": catalog, "count": len(catalog)}, err)
}

func (h *ArmyHandler) AdminPlayerUnits(c *gin.Context) {
	units, err := h.svc.PlayerUnits(c.Request.Context(), uintFromParam(c, "profileId"))
	writeArmy(c, gin.H{"units": units}, err)
}

func (h *ArmyHandler) AdminGrantUnits(c *gin.Context) {
	var body struct {
		UnitCode string `json:"unitCode"`
		Quantity int    `json:"quantity"`
		Reason   string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "errorCode": "invalid_payload", "message": err.Error()})
		return
	}
	unit, err := h.svc.AdminGrantUnits(c.Request.Context(), uintFromParam(c, "profileId"), body.UnitCode, body.Quantity, body.Reason)
	writeArmy(c, gin.H{"success": true, "unit": unit}, err)
}

func (h *ArmyHandler) AdminSnapshot(c *gin.Context) {
	data, err := h.svc.AdminSnapshot(c.Request.Context())
	writeArmy(c, data, err)
}

func writeArmy(c *gin.Context, payload any, err error) {
	if err == nil {
		c.JSON(http.StatusOK, payload)
		return
	}
	var coded services.ArmyCodedError
	if errors.As(err, &coded) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "errorCode": coded.Code, "message": coded.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"success": false, "errorCode": "server_error", "message": err.Error()})
}

func armyProfileID(c *gin.Context) (uint, bool) {
	id := armyProfileIDFromQuery(c)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "errorCode": "profile_required", "message": "profileGamerId est requis."})
		return 0, false
	}
	return id, true
}

func armyProfileIDFromQuery(c *gin.Context) uint {
	for _, key := range []string{"profileGamerId", "profile_gamer_id", "profileId", "profile_id"} {
		if raw := c.Query(key); raw != "" {
			if id, err := strconv.ParseUint(raw, 10, 64); err == nil {
				return uint(id)
			}
		}
	}
	return 0
}

func uintFromParam(c *gin.Context, key string) uint {
	raw := c.Param(key)
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}
