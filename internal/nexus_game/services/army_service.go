package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"cgwm/battle/internal/nexus_game/models"
	"cgwm/battle/internal/nexus_game/repositories"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ArmyService struct {
	db          *gorm.DB
	contentSvc  *ContentService
	resourceSvc *ResourceService
}

type ArmyCapacitySnapshot struct {
	Capacity       int `json:"capacity"`
	Used           int `json:"used"`
	Reserved       int `json:"reserved"`
	Available      int `json:"available"`
	TrainingSlots  int `json:"trainingSlots"`
	TrainingActive int `json:"trainingActive"`
}

type ArmyTrainRequest struct {
	ProfileGamerID uint   `json:"profileGamerId"`
	UnitCode       string `json:"unitCode"`
	Quantity       int    `json:"quantity"`
}

type ArmyAssignmentRequest struct {
	ProfileGamerID uint   `json:"profileGamerId"`
	UnitCode       string `json:"unitCode"`
	Quantity       int    `json:"quantity"`
}

type ArmyFormationDetail struct {
	Formation   models.ArmyFormation        `json:"formation"`
	Slots       []models.ArmyFormationSlot  `json:"slots"`
	Assignments []models.ArmySlotAssignment `json:"assignments"`
}

func NewArmyService(db *gorm.DB, contentSvc *ContentService) *ArmyService {
	return &ArmyService{db: db, contentSvc: contentSvc, resourceSvc: NewResourceService(db)}
}

func (s *ArmyService) Catalog(ctx context.Context, publishedOnly bool) ([]models.UnitDefinition, error) {
	return s.contentSvc.ListUnits(publishedOnly)
}

func (s *ArmyService) PlayerUnits(ctx context.Context, profileID uint) ([]models.PlayerUnit, error) {
	if err := s.resourceSvc.EnsureInitialAllocation(ctx, profileID); err != nil {
		return nil, err
	}
	return repositories.NewPlayerUnitRepository(s.db).List(ctx, profileID)
}

func (s *ArmyService) TrainingQueue(ctx context.Context, profileID uint, includeDone bool) ([]models.UnitTrainingQueue, error) {
	if err := s.RefreshTrainingQueue(ctx, profileID); err != nil {
		return nil, err
	}
	return repositories.NewUnitTrainingQueueRepository(s.db).List(ctx, profileID, includeDone)
}

func (s *ArmyService) Capacity(ctx context.Context, profileID uint) (ArmyCapacitySnapshot, error) {
	var profile models.ProfileGamer
	if err := s.db.WithContext(ctx).First(&profile, profileID).Error; err != nil {
		return ArmyCapacitySnapshot{}, err
	}
	barracks := s.playerBuildingLevel(ctx, profileID, "building_barracks")
	guild := s.playerBuildingLevel(ctx, profileID, "building_guild_hq")
	aiCenter := s.playerBuildingLevel(ctx, profileID, "building_ai_center")
	capacity := 20 + barracks*10 + guild*15 + aiCenter*10
	if capacity < 20 {
		capacity = 20
	}

	units, err := repositories.NewPlayerUnitRepository(s.db).List(ctx, profileID)
	if err != nil {
		return ArmyCapacitySnapshot{}, err
	}
	used := 0
	for _, unit := range units {
		def, err := s.contentSvc.GetUnit(unit.ContentID)
		if err != nil {
			continue
		}
		used += unit.Count * capacityCostForUnit(def)
	}

	queue, err := repositories.NewUnitTrainingQueueRepository(s.db).List(ctx, profileID, false)
	if err != nil {
		return ArmyCapacitySnapshot{}, err
	}
	reserved := 0
	active := 0
	for _, row := range queue {
		if row.Status == "training" {
			active++
			reserved += row.ReservedCapacity
		}
	}
	slots := 1 + minInt(2, barracks/5+aiCenter/8)
	if s.playerBuildingLevel(ctx, profileID, "building_drone_factory") > 0 && slots < 3 {
		slots++
	}
	if slots > 3 {
		slots = 3
	}
	return ArmyCapacitySnapshot{
		Capacity: capacity, Used: used, Reserved: reserved,
		Available: capacity - used - reserved, TrainingSlots: slots, TrainingActive: active,
	}, nil
}

func (s *ArmyService) TrainUnit(ctx context.Context, req ArmyTrainRequest) (*models.UnitTrainingQueue, ArmyCapacitySnapshot, error) {
	req.UnitCode = strings.TrimSpace(req.UnitCode)
	if req.ProfileGamerID == 0 {
		return nil, ArmyCapacitySnapshot{}, codedError("profile_required", "profileGamerId est requis.")
	}
	if req.UnitCode == "" {
		return nil, ArmyCapacitySnapshot{}, codedError("unit_required", "unitCode est requis.")
	}
	if req.Quantity <= 0 {
		return nil, ArmyCapacitySnapshot{}, codedError("invalid_quantity", "La quantite doit etre superieure a 0.")
	}
	if req.Quantity > 500 {
		return nil, ArmyCapacitySnapshot{}, codedError("quantity_too_high", "La quantite depasse la limite de securite.")
	}
	var profile models.ProfileGamer
	if err := s.db.WithContext(ctx).First(&profile, req.ProfileGamerID).Error; err != nil {
		return nil, ArmyCapacitySnapshot{}, err
	}
	def, err := s.contentSvc.GetUnit(req.UnitCode)
	if err != nil {
		return nil, ArmyCapacitySnapshot{}, codedError("unknown_unit", "Unite inconnue.")
	}
	if !def.IsPublished {
		return nil, ArmyCapacitySnapshot{}, codedError("unit_disabled", "Cette unite est desactivee par l'administration.")
	}
	if validation, err := s.contentSvc.ValidatePrerequisites(req.ProfileGamerID, "unit", req.UnitCode); err != nil || !validation.Allowed {
		return nil, ArmyCapacitySnapshot{}, codedError("missing_prerequisite", firstMissingRequirementMessage(validation, "Les prerequis de cette unite ne sont pas remplis."))
	}
	capacity, err := s.Capacity(ctx, req.ProfileGamerID)
	if err != nil {
		return nil, ArmyCapacitySnapshot{}, err
	}
	neededCapacity := req.Quantity * capacityCostForUnit(def)
	if neededCapacity > capacity.Available {
		return nil, capacity, codedError("army_capacity_insufficient", "Capacite militaire insuffisante.")
	}
	if capacity.TrainingActive >= capacity.TrainingSlots {
		return nil, capacity, codedError("training_queue_full", "La file d'entrainement est pleine.")
	}
	cost := trainingCostForUnit(def, req.Quantity)
	if err := s.ensureResources(ctx, req.ProfileGamerID, cost); err != nil {
		return nil, capacity, err
	}
	now := time.Now().UTC()
	queue := &models.UnitTrainingQueue{
		ProfileGamerID:   req.ProfileGamerID,
		WorldID:          profile.WorldID,
		UnitCode:         req.UnitCode,
		Quantity:         req.Quantity,
		Status:           "training",
		StartedAt:        now,
		CompletedAt:      now.Add(time.Duration(trainingDurationForUnit(def, req.Quantity)) * time.Second),
		CostJSON:         mustArmyJSON(cost),
		ReservedCapacity: neededCapacity,
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rs := NewResourceService(tx)
		for code, amount := range cost {
			if amount > 0 {
				if _, err := rs.ApplyResourceDelta(ctx, req.ProfileGamerID, code, -amount, "unit_training_start", "army"); err != nil {
					return err
				}
			}
		}
		if err := repositories.NewUnitTrainingQueueRepository(tx).Create(ctx, queue); err != nil {
			return err
		}
		return repositories.NewArmyTransactionLogRepository(tx).Create(ctx, &models.ArmyTransactionLog{
			ProfileGamerID: req.ProfileGamerID, WorldID: profile.WorldID, ActionType: "training_start",
			UnitCode: req.UnitCode, Quantity: req.Quantity, AfterJSON: mustArmyJSON(queue),
			Reason: "player", LinkedType: "unit_training_queue", LinkedID: queue.ID, CreatedAt: now,
		})
	})
	return queue, capacity, err
}

func (s *ArmyService) RefreshTrainingQueue(ctx context.Context, profileID uint) error {
	rows, err := repositories.NewUnitTrainingQueueRepository(s.db).List(ctx, profileID, false)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for i := range rows {
		if rows[i].Status == "training" && !rows[i].CompletedAt.After(now) {
			rows[i].Status = "ready"
			if err := repositories.NewUnitTrainingQueueRepository(s.db).Save(ctx, &rows[i]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *ArmyService) ClaimTraining(ctx context.Context, profileID uint, queueID uint) (*models.PlayerUnit, error) {
	if err := s.RefreshTrainingQueue(ctx, profileID); err != nil {
		return nil, err
	}
	queue, err := repositories.NewUnitTrainingQueueRepository(s.db).Get(ctx, queueID)
	if err != nil {
		return nil, err
	}
	if queue.ProfileGamerID != profileID {
		return nil, codedError("not_owner", "Cette queue ne correspond pas au joueur.")
	}
	if queue.Status != "ready" {
		return nil, codedError("training_not_ready", "L'entrainement n'est pas termine.")
	}
	now := time.Now().UTC()
	var updated *models.PlayerUnit
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		unitRepo := repositories.NewPlayerUnitRepository(tx)
		unit, err := unitRepo.GetOrCreate(ctx, profileID, queue.UnitCode)
		if err != nil {
			return err
		}
		before := *unit
		unit.ReserveQuantity += queue.Quantity
		unit.Count += queue.Quantity
		if err := unitRepo.Save(ctx, unit); err != nil {
			return err
		}
		queue.Status = "claimed"
		queue.ClaimedAt = &now
		if err := repositories.NewUnitTrainingQueueRepository(tx).Save(ctx, queue); err != nil {
			return err
		}
		updated = unit
		return repositories.NewArmyTransactionLogRepository(tx).Create(ctx, &models.ArmyTransactionLog{
			ProfileGamerID: profileID, WorldID: queue.WorldID, ActionType: "training_claim",
			UnitCode: queue.UnitCode, Quantity: queue.Quantity, BeforeJSON: mustArmyJSON(before), AfterJSON: mustArmyJSON(unit),
			LinkedType: "unit_training_queue", LinkedID: queue.ID, CreatedAt: now,
		})
	})
	return updated, err
}

func (s *ArmyService) CancelTraining(ctx context.Context, profileID uint, queueID uint) (*models.UnitTrainingQueue, error) {
	queue, err := repositories.NewUnitTrainingQueueRepository(s.db).Get(ctx, queueID)
	if err != nil {
		return nil, err
	}
	if queue.ProfileGamerID != profileID {
		return nil, codedError("not_owner", "Cette queue ne correspond pas au joueur.")
	}
	if queue.Status != "training" {
		return nil, codedError("training_not_cancellable", "Seul un entrainement en cours peut etre annule.")
	}
	cost := map[string]int64{}
	_ = json.Unmarshal(queue.CostJSON, &cost)
	refund := map[string]int64{}
	for code, amount := range cost {
		refund[code] = int64(math.Round(float64(amount) * 0.60))
	}
	now := time.Now().UTC()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rs := NewResourceService(tx)
		for code, amount := range refund {
			if amount > 0 {
				if _, err := rs.ApplyResourceDelta(ctx, profileID, code, amount, "unit_training_cancel_refund", "army"); err != nil {
					return err
				}
			}
		}
		queue.Status = "cancelled"
		queue.CancelledAt = &now
		queue.RefundJSON = mustArmyJSON(refund)
		if err := repositories.NewUnitTrainingQueueRepository(tx).Save(ctx, queue); err != nil {
			return err
		}
		return repositories.NewArmyTransactionLogRepository(tx).Create(ctx, &models.ArmyTransactionLog{
			ProfileGamerID: profileID, WorldID: queue.WorldID, ActionType: "training_cancel",
			UnitCode: queue.UnitCode, Quantity: queue.Quantity, AfterJSON: mustArmyJSON(refund),
			LinkedType: "unit_training_queue", LinkedID: queue.ID, CreatedAt: now,
		})
	})
	return queue, err
}

func (s *ArmyService) EnsureDefaultFormations(ctx context.Context, profileID uint) error {
	var profile models.ProfileGamer
	if err := s.db.WithContext(ctx).First(&profile, profileID).Error; err != nil {
		return err
	}
	formationRepo := repositories.NewArmyFormationRepository(s.db)
	for _, spec := range defaultFormationSpecs() {
		row := &models.ArmyFormation{ProfileGamerID: profileID, WorldID: profile.WorldID, Type: spec.Type, Name: spec.Name, Status: "active"}
		if err := formationRepo.UpsertDefault(ctx, row); err != nil {
			return err
		}
	}
	formations, err := formationRepo.List(ctx, profileID)
	if err != nil {
		return err
	}
	slotRepo := repositories.NewArmySlotRepository(s.db)
	for _, formation := range formations {
		slots, err := slotRepo.ListSlots(ctx, formation.ID)
		if err != nil {
			return err
		}
		if len(slots) > 0 {
			continue
		}
		for _, slot := range defaultSlotsForFormation(formation.Type) {
			row := models.ArmyFormationSlot{
				FormationID: formation.ID, SlotIndex: slot.Index, SlotType: slot.Type,
				Capacity: slot.Capacity, AllowedUnitTypesJSON: mustArmyJSON(allowedTypesForSlot(slot.Type)),
			}
			if err := slotRepo.CreateSlot(ctx, &row); err != nil {
				return err
			}
		}
	}
	_, err = repositories.NewArmyAutomationRepository(s.db).GetOrCreate(ctx, profileID, profile.WorldID)
	return err
}

func (s *ArmyService) Formations(ctx context.Context, profileID uint) ([]ArmyFormationDetail, error) {
	if err := s.EnsureDefaultFormations(ctx, profileID); err != nil {
		return nil, err
	}
	formations, err := repositories.NewArmyFormationRepository(s.db).List(ctx, profileID)
	if err != nil {
		return nil, err
	}
	details := make([]ArmyFormationDetail, 0, len(formations))
	for _, formation := range formations {
		if err := s.RecalculateFormation(ctx, formation.ID); err != nil {
			return nil, err
		}
		detail, err := s.Formation(ctx, profileID, formation.ID)
		if err != nil {
			return nil, err
		}
		details = append(details, *detail)
	}
	return details, nil
}

func (s *ArmyService) Formation(ctx context.Context, profileID uint, formationID uint) (*ArmyFormationDetail, error) {
	formation, err := repositories.NewArmyFormationRepository(s.db).Get(ctx, formationID)
	if err != nil {
		return nil, err
	}
	if formation.ProfileGamerID != profileID {
		return nil, codedError("not_owner", "Formation non disponible pour ce joueur.")
	}
	slots, err := repositories.NewArmySlotRepository(s.db).ListSlots(ctx, formationID)
	if err != nil {
		return nil, err
	}
	assignments, err := repositories.NewArmySlotRepository(s.db).ListAssignments(ctx, formationID)
	if err != nil {
		return nil, err
	}
	return &ArmyFormationDetail{Formation: *formation, Slots: slots, Assignments: assignments}, nil
}

func (s *ArmyService) AssignUnit(ctx context.Context, profileID uint, formationID uint, slotID uint, req ArmyAssignmentRequest) (*ArmyFormationDetail, error) {
	req.UnitCode = strings.TrimSpace(req.UnitCode)
	if req.Quantity <= 0 {
		return nil, codedError("invalid_quantity", "La quantite doit etre superieure a 0.")
	}
	formation, err := repositories.NewArmyFormationRepository(s.db).Get(ctx, formationID)
	if err != nil {
		return nil, err
	}
	if formation.ProfileGamerID != profileID {
		return nil, codedError("not_owner", "Formation non disponible pour ce joueur.")
	}
	slot, err := repositories.NewArmySlotRepository(s.db).GetSlot(ctx, slotID)
	if err != nil {
		return nil, err
	}
	if slot.FormationID != formationID {
		return nil, codedError("slot_invalid", "Slot invalide.")
	}
	if slot.IsLocked {
		return nil, codedError("slot_locked", "Ce slot est verrouille.")
	}
	def, err := s.contentSvc.GetUnit(req.UnitCode)
	if err != nil {
		return nil, codedError("unknown_unit", "Unite inconnue.")
	}
	if !slotAllowsUnit(slot.SlotType, def) {
		return nil, codedError("slot_incompatible", "Cette unite n'est pas compatible avec ce slot.")
	}
	capacityUsed := capacityCostForUnit(def) * req.Quantity
	assignments, err := repositories.NewArmySlotRepository(s.db).ListAssignments(ctx, formationID)
	if err != nil {
		return nil, err
	}
	currentSlotUsed := 0
	for _, assignment := range assignments {
		if assignment.SlotID == slotID {
			currentSlotUsed += assignment.CapacityUsed
		}
	}
	if currentSlotUsed+capacityUsed > slot.Capacity {
		return nil, codedError("slot_capacity_exceeded", "La capacite du slot est depassee.")
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		unitRepo := repositories.NewPlayerUnitRepository(tx)
		unit, err := unitRepo.Get(ctx, profileID, req.UnitCode)
		if err != nil {
			return codedError("unit_not_owned", "Le joueur ne possede pas cette unite.")
		}
		if unit.ReserveQuantity < req.Quantity {
			return codedError("reserve_insufficient", "Quantite en reserve insuffisante.")
		}
		before := *unit
		unit.ReserveQuantity -= req.Quantity
		unit.AssignedQuantity += req.Quantity
		if err := unitRepo.Save(ctx, unit); err != nil {
			return err
		}
		slotRepo := repositories.NewArmySlotRepository(tx)
		assignment, err := slotRepo.GetAssignment(ctx, slotID, req.UnitCode)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if assignment == nil {
			assignment = &models.ArmySlotAssignment{FormationID: formationID, SlotID: slotID, UnitCode: req.UnitCode}
		}
		assignment.Quantity += req.Quantity
		assignment.CapacityUsed += capacityUsed
		assignment.Power += unitPower(def) * req.Quantity
		if err := slotRepo.SaveAssignment(ctx, assignment); err != nil {
			return err
		}
		return repositories.NewArmyTransactionLogRepository(tx).Create(ctx, &models.ArmyTransactionLog{
			ProfileGamerID: profileID, WorldID: formation.WorldID, ActionType: "slot_assign",
			UnitCode: req.UnitCode, Quantity: req.Quantity, BeforeJSON: mustArmyJSON(before), AfterJSON: mustArmyJSON(unit),
			LinkedType: "army_formation", LinkedID: formationID, CreatedAt: time.Now().UTC(),
		})
	})
	if err != nil {
		return nil, err
	}
	if err := s.RecalculateFormation(ctx, formationID); err != nil {
		return nil, err
	}
	return s.Formation(ctx, profileID, formationID)
}

func (s *ArmyService) RemoveUnit(ctx context.Context, profileID uint, formationID uint, slotID uint, req ArmyAssignmentRequest) (*ArmyFormationDetail, error) {
	if req.Quantity <= 0 {
		return nil, codedError("invalid_quantity", "La quantite doit etre superieure a 0.")
	}
	formation, err := repositories.NewArmyFormationRepository(s.db).Get(ctx, formationID)
	if err != nil {
		return nil, err
	}
	if formation.ProfileGamerID != profileID {
		return nil, codedError("not_owner", "Formation non disponible pour ce joueur.")
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		slotRepo := repositories.NewArmySlotRepository(tx)
		assignment, err := slotRepo.GetAssignment(ctx, slotID, req.UnitCode)
		if err != nil {
			return codedError("assignment_not_found", "Assignation introuvable.")
		}
		if assignment.Quantity < req.Quantity {
			return codedError("assignment_quantity_insufficient", "Quantite assignee insuffisante.")
		}
		def, err := s.contentSvc.GetUnit(req.UnitCode)
		if err != nil {
			return err
		}
		unitRepo := repositories.NewPlayerUnitRepository(tx)
		unit, err := unitRepo.Get(ctx, profileID, req.UnitCode)
		if err != nil {
			return err
		}
		before := *unit
		capacityReleased := capacityCostForUnit(def) * req.Quantity
		powerReleased := unitPower(def) * req.Quantity
		assignment.Quantity -= req.Quantity
		assignment.CapacityUsed -= capacityReleased
		assignment.Power -= powerReleased
		if assignment.Quantity <= 0 {
			if err := slotRepo.DeleteAssignment(ctx, assignment); err != nil {
				return err
			}
		} else if err := slotRepo.SaveAssignment(ctx, assignment); err != nil {
			return err
		}
		unit.AssignedQuantity -= req.Quantity
		if unit.AssignedQuantity < 0 {
			unit.AssignedQuantity = 0
		}
		unit.ReserveQuantity += req.Quantity
		if err := unitRepo.Save(ctx, unit); err != nil {
			return err
		}
		return repositories.NewArmyTransactionLogRepository(tx).Create(ctx, &models.ArmyTransactionLog{
			ProfileGamerID: profileID, WorldID: formation.WorldID, ActionType: "slot_remove",
			UnitCode: req.UnitCode, Quantity: req.Quantity, BeforeJSON: mustArmyJSON(before), AfterJSON: mustArmyJSON(unit),
			LinkedType: "army_formation", LinkedID: formationID, CreatedAt: time.Now().UTC(),
		})
	})
	if err != nil {
		return nil, err
	}
	if err := s.RecalculateFormation(ctx, formationID); err != nil {
		return nil, err
	}
	return s.Formation(ctx, profileID, formationID)
}

func (s *ArmyService) RecalculateFormation(ctx context.Context, formationID uint) error {
	formation, err := repositories.NewArmyFormationRepository(s.db).Get(ctx, formationID)
	if err != nil {
		return err
	}
	assignments, err := repositories.NewArmySlotRepository(s.db).ListAssignments(ctx, formationID)
	if err != nil {
		return err
	}
	total := 0
	attack := 0
	defense := 0
	scouting := 0
	antiSabotage := 0
	support := 0
	upkeep := map[string]int64{}
	for _, assignment := range assignments {
		def, err := s.contentSvc.GetUnit(assignment.UnitCode)
		if err != nil {
			continue
		}
		power := unitPower(def) * assignment.Quantity
		total += power
		attack += def.AttackBase * assignment.Quantity
		defense += def.DefenseBase * assignment.Quantity
		if strings.Contains(def.ContentID, "eclaireur") || strings.Contains(def.ContentID, "agent") || strings.Contains(def.Type, "drone") {
			scouting += power / 2
		}
		if strings.Contains(def.ContentID, "hacker") || strings.Contains(def.ContentID, "sentinelle") || strings.Contains(def.ContentID, "anti") {
			antiSabotage += power / 2
		}
		if def.Type == "support" || strings.Contains(def.ContentID, "medic") {
			support += power / 2
		}
		for code, amount := range upkeepForUnit(def, assignment.Quantity) {
			upkeep[code] += amount
		}
	}
	now := time.Now().UTC()
	formation.TotalPower = total
	formation.AttackPower = attack
	formation.DefensePower = defense
	formation.ScoutingPower = scouting
	formation.AntiSabotagePower = antiSabotage
	formation.SupportPower = support
	formation.UpkeepJSON = mustArmyJSON(upkeep)
	formation.LastCalculatedAt = &now
	return repositories.NewArmyFormationRepository(s.db).Save(ctx, formation)
}

func (s *ArmyService) Automation(ctx context.Context, profileID uint) (*models.ArmyAutomationSettings, error) {
	var profile models.ProfileGamer
	if err := s.db.WithContext(ctx).First(&profile, profileID).Error; err != nil {
		return nil, err
	}
	return repositories.NewArmyAutomationRepository(s.db).GetOrCreate(ctx, profileID, profile.WorldID)
}

func (s *ArmyService) SaveAutomation(ctx context.Context, profileID uint, patch models.ArmyAutomationSettings) (*models.ArmyAutomationSettings, error) {
	settings, err := s.Automation(ctx, profileID)
	if err != nil {
		return nil, err
	}
	settings.AutoDefenseEnabled = patch.AutoDefenseEnabled
	settings.AutoRepairEnabled = patch.AutoRepairEnabled
	settings.AutoHealEnabled = patch.AutoHealEnabled
	settings.AutoTrainEnabled = patch.AutoTrainEnabled
	settings.AutoComposeDefenseEnabled = patch.AutoComposeDefenseEnabled
	settings.MaxAutoSpendPercent = clampInt(patch.MaxAutoSpendPercent, 0, 50, 20)
	settings.MinFoodKeep = maxInt64(0, patch.MinFoodKeep)
	settings.MinEnergyKeep = maxInt64(0, patch.MinEnergyKeep)
	settings.MinDefensePower = clampInt(patch.MinDefensePower, 0, 1000000, 300)
	settings.MaxUnitsOnMissionPercent = clampInt(patch.MaxUnitsOnMissionPercent, 0, 80, 40)
	settings.AllowRareResourceSpend = patch.AllowRareResourceSpend
	return settings, repositories.NewArmyAutomationRepository(s.db).Save(ctx, settings)
}

func (s *ArmyService) CommanderSuggest(ctx context.Context, profileID uint, formationID uint) (map[string]any, error) {
	detail, err := s.Formation(ctx, profileID, formationID)
	if err != nil {
		return nil, err
	}
	units, err := s.PlayerUnits(ctx, profileID)
	if err != nil {
		return nil, err
	}
	suggestions := []map[string]any{}
	for _, slot := range detail.Slots {
		if slot.IsLocked {
			continue
		}
		for _, unit := range units {
			if unit.ReserveQuantity <= 0 {
				continue
			}
			def, err := s.contentSvc.GetUnit(unit.ContentID)
			if err != nil || !slotAllowsUnit(slot.SlotType, def) {
				continue
			}
			qty := minInt(unit.ReserveQuantity, maxInt(1, slot.Capacity/capacityCostForUnit(def)))
			suggestions = append(suggestions, map[string]any{"slotId": slot.ID, "slotIndex": slot.SlotIndex, "unitCode": unit.ContentID, "quantity": qty})
			break
		}
	}
	return map[string]any{
		"summary":              "Proposition generee cote serveur. Le joueur doit valider; aucune assignation n'est appliquee automatiquement.",
		"suggestedAssignments": suggestions,
		"warnings": []string{
			"Ne retire pas toutes les unites de defense avant un raid IA.",
			"Le Commandant IA ne lance pas d'attaque seul.",
		},
	}, nil
}

func (s *ArmyService) CombatReports(ctx context.Context, profileID uint, limit int) ([]models.ArmyCombatReport, error) {
	return repositories.NewArmyReportRepository(s.db).ListReports(ctx, profileID, limit)
}

func (s *ArmyService) AdminGrantUnits(ctx context.Context, profileID uint, unitCode string, quantity int, reason string) (*models.PlayerUnit, error) {
	if quantity == 0 {
		return nil, codedError("invalid_quantity", "La quantite doit etre differente de 0.")
	}
	def, err := s.contentSvc.GetUnit(unitCode)
	if err != nil || def.ContentID == "" {
		return nil, codedError("unknown_unit", "Unite inconnue.")
	}
	var profile models.ProfileGamer
	if err := s.db.WithContext(ctx).First(&profile, profileID).Error; err != nil {
		return nil, err
	}
	var updated *models.PlayerUnit
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := repositories.NewPlayerUnitRepository(tx)
		unit, err := repo.GetOrCreate(ctx, profileID, unitCode)
		if err != nil {
			return err
		}
		before := *unit
		unit.ReserveQuantity += quantity
		if unit.ReserveQuantity < 0 {
			unit.ReserveQuantity = 0
		}
		if err := repo.Save(ctx, unit); err != nil {
			return err
		}
		updated = unit
		return repositories.NewArmyTransactionLogRepository(tx).Create(ctx, &models.ArmyTransactionLog{
			ProfileGamerID: profileID, WorldID: profile.WorldID, ActionType: "admin_units_delta",
			UnitCode: unitCode, Quantity: quantity, BeforeJSON: mustArmyJSON(before), AfterJSON: mustArmyJSON(unit),
			Reason: reason, CreatedAt: time.Now().UTC(),
		})
	})
	return updated, err
}

func (s *ArmyService) AdminSnapshot(ctx context.Context) (map[string]any, error) {
	var players []models.ProfileGamer
	if err := s.db.WithContext(ctx).Order("updated_at DESC, id DESC").Limit(100).Find(&players).Error; err != nil {
		return nil, err
	}
	catalog, err := s.Catalog(ctx, false)
	if err != nil {
		return nil, err
	}
	queues, err := repositories.NewUnitTrainingQueueRepository(s.db).ListAll(ctx, 100)
	if err != nil {
		return nil, err
	}
	formations, err := repositories.NewArmyFormationRepository(s.db).ListAll(ctx, 100)
	if err != nil {
		return nil, err
	}
	reports, err := repositories.NewArmyReportRepository(s.db).ListAllReports(ctx, 50)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"players": players, "unitCatalog": catalog, "trainingQueues": queues,
		"formations": formations, "combatReports": reports,
		"modules": []map[string]any{
			{"key": "resources", "label": "Ressources joueur", "crud": true, "endpoint": "/admin/api/nexus-system/resources/grant"},
			{"key": "units", "label": "Unites et training queue", "crud": true, "endpoint": "/api/nexus-game/admin/units/catalog"},
			{"key": "formations", "label": "Formations et slots", "crud": true, "endpoint": "/api/nexus-game/admin/army/formations"},
			{"key": "server_ai", "label": "IA serveur et crons", "crud": false, "endpoint": "/api/nexus-game/admin/ai-server/jobs"},
		},
	}, nil
}

func (s *ArmyService) playerBuildingLevel(ctx context.Context, profileID uint, contentID string) int {
	var building models.PlayerBuilding
	err := s.db.WithContext(ctx).Where("profile_gamer_id = ? AND content_id = ?", profileID, contentID).Order("level DESC").First(&building).Error
	if err != nil {
		return 0
	}
	return building.Level
}

func (s *ArmyService) ensureResources(ctx context.Context, profileID uint, cost map[string]int64) error {
	if err := s.resourceSvc.EnsureInitialAllocation(ctx, profileID); err != nil {
		return err
	}
	resources, err := repositories.NewPlayerResourceRepository(s.db).List(ctx, profileID)
	if err != nil {
		return err
	}
	current := map[string]int64{}
	for _, resource := range resources {
		current[resource.ResourceCode] = resource.Amount
	}
	for code, needed := range cost {
		if needed > current[code] {
			return codedError("resources_insufficient", fmt.Sprintf("Ressource insuffisante: %s requis %d, disponible %d.", code, needed, current[code]))
		}
	}
	return nil
}

type formationSpec struct{ Type, Name string }
type slotSpec struct {
	Index    int
	Type     string
	Capacity int
}

func defaultFormationSpecs() []formationSpec {
	return []formationSpec{
		{"defense", "Defense"},
		{"attack", "Attaque"},
		{"mission", "Mission"},
		{"guild_raid", "Raid guilde"},
		{"exploration", "Exploration"},
		{"anti_sabotage", "Anti-sabotage"},
	}
}

func defaultSlotsForFormation(kind string) []slotSpec {
	switch kind {
	case "defense":
		return []slotSpec{{1, "infantry_slot", 10}, {2, "drone_slot", 10}, {3, "support_slot", 5}}
	case "attack":
		return []slotSpec{{1, "infantry_slot", 10}, {2, "drone_slot", 10}, {3, "free_light_slot", 10}}
	case "exploration":
		return []slotSpec{{1, "scout_slot", 10}, {2, "drone_slot", 10}}
	case "anti_sabotage":
		return []slotSpec{{1, "drone_slot", 10}, {2, "cyber_slot", 5}}
	case "mission":
		return []slotSpec{{1, "infantry_slot", 10}, {2, "support_slot", 5}}
	case "guild_raid":
		return []slotSpec{{1, "infantry_slot", 20}, {2, "drone_slot", 20}, {3, "support_slot", 10}, {4, "heavy_slot", 20}}
	default:
		return []slotSpec{{1, "free_any_slot", 20}}
	}
}

func allowedTypesForSlot(slotType string) []string {
	switch slotType {
	case "infantry_slot":
		return []string{"infantry"}
	case "drone_slot":
		return []string{"drone"}
	case "heavy_slot":
		return []string{"mecha", "artillerie", "defense"}
	case "support_slot":
		return []string{"support", "defense"}
	case "cyber_slot":
		return []string{"support", "special", "drone"}
	case "commander_slot":
		return []string{"commander", "officer"}
	case "scout_slot":
		return []string{"infantry", "drone", "special"}
	case "free_light_slot":
		return []string{"infantry", "drone", "support", "special"}
	default:
		return []string{"infantry", "drone", "support", "special", "mecha", "artillerie", "defense"}
	}
}

func slotAllowsUnit(slotType string, def *models.UnitDefinition) bool {
	for _, allowed := range allowedTypesForSlot(slotType) {
		if def.Type == allowed {
			return true
		}
	}
	if slotType == "scout_slot" && (strings.Contains(def.ContentID, "eclaireur") || strings.Contains(def.ContentID, "agent")) {
		return true
	}
	return false
}

func capacityCostForUnit(def *models.UnitDefinition) int {
	switch def.Type {
	case "mecha":
		if strings.Contains(def.ContentID, "titan") {
			return 15
		}
		return 8
	case "artillerie":
		return 8
	case "defense":
		return 5
	case "support", "special":
		return 3
	case "drone":
		return 2
	default:
		return 1
	}
}

func trainingCostForUnit(def *models.UnitDefinition, quantity int) map[string]int64 {
	base := int64(maxInt(1, def.UpkeepBase))
	q := int64(quantity)
	switch def.Type {
	case "drone":
		return map[string]int64{"components": base * 8 * q, "energy": base * 5 * q, "data": base * 3 * q, "metal": base * 4 * q}
	case "mecha", "artillerie", "defense":
		return map[string]int64{"metal": base * 12 * q, "components": base * 8 * q, "energy": base * 6 * q, "data": base * 3 * q}
	case "support", "special":
		return map[string]int64{"food": base * 4 * q, "energy": base * 4 * q, "metal": base * 4 * q, "influence": int64(math.Ceil(float64(base*q) / 3))}
	default:
		return map[string]int64{"food": base * 6 * q, "energy": base * 3 * q, "metal": base * 5 * q}
	}
}

func upkeepForUnit(def *models.UnitDefinition, quantity int) map[string]int64 {
	base := int64(maxInt(1, def.UpkeepBase) * quantity)
	switch def.Type {
	case "drone":
		return map[string]int64{"energy": base, "components": maxInt64(1, base/2)}
	case "mecha", "artillerie", "defense":
		return map[string]int64{"energy": base * 2, "components": base}
	case "support", "special":
		return map[string]int64{"food": base, "energy": base, "data": maxInt64(1, base/3)}
	default:
		return map[string]int64{"food": base, "energy": maxInt64(1, base/2)}
	}
}

func trainingDurationForUnit(def *models.UnitDefinition, quantity int) int {
	base := def.TrainingTimeBaseSeconds
	if base <= 0 {
		base = 300
	}
	return base * quantity
}

func unitPower(def *models.UnitDefinition) int {
	power := float64(def.HealthBase)*0.2 + float64(def.AttackBase)*1.5 + float64(def.DefenseBase)*1.2 + float64(def.SpeedBase)*0.5
	if strings.Contains(def.Type, "drone") || def.Type == "support" || def.Type == "special" {
		power += 10
	}
	return int(math.Round(power))
}

func firstMissingRequirementMessage(validation PrerequisiteValidation, fallback string) string {
	for _, item := range validation.Missing {
		if item.Message != "" {
			return item.Message
		}
	}
	return fallback
}

func mustArmyJSON(value any) datatypes.JSON {
	raw, _ := json.Marshal(value)
	return datatypes.JSON(raw)
}

type ArmyCodedError struct {
	Code    string `json:"errorCode"`
	Message string `json:"message"`
}

func (e ArmyCodedError) Error() string {
	return e.Code + ": " + e.Message
}

func codedError(code string, message string) error {
	return ArmyCodedError{Code: code, Message: message}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampInt(value int, min int, max int, fallback int) int {
	if value == 0 {
		value = fallback
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
