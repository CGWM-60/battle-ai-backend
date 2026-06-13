package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cgwm/battle/internal/nexus_game/models"
	"cgwm/battle/internal/nexus_game/repositories"

	"gorm.io/gorm"
)

const (
	armyRuleSlotUnlock    = "slot_unlock"
	armyRuleCapacityBonus = "capacity_bonus"
)

type ArmyProgressionService struct {
	db         *gorm.DB
	contentSvc *ContentService
}

type ArmyProgressionChange struct {
	FormationType  string `json:"formationType"`
	SlotType       string `json:"slotType"`
	SlotIndex      int    `json:"slotIndex"`
	SourceCode     string `json:"sourceCode"`
	ChangeType     string `json:"changeType"`
	BeforeLocked   bool   `json:"beforeLocked"`
	AfterLocked    bool   `json:"afterLocked"`
	BeforeCapacity int    `json:"beforeCapacity"`
	AfterCapacity  int    `json:"afterCapacity"`
}

type ArmyFormationProgressionSummary struct {
	Type          string                     `json:"type"`
	MaxSlots      int                        `json:"maxSlots"`
	UnlockedSlots int                        `json:"unlockedSlots"`
	LockedSlots   int                        `json:"lockedSlots"`
	NextUnlocks   []map[string]any           `json:"nextUnlocks"`
	Slots         []models.ArmyFormationSlot `json:"slots"`
}

type ArmyProgressionSnapshot struct {
	ArmyCapacity     int                               `json:"armyCapacity"`
	ArmyCapacityUsed int                               `json:"armyCapacityUsed"`
	TrainingSlots    int                               `json:"trainingSlots"`
	Formations       []ArmyFormationProgressionSummary `json:"formations"`
}

type armyProgressionState struct {
	Profile       models.ProfileGamer
	BuildingLevel map[string]int
	ResearchDone  map[string]bool
}

func NewArmyProgressionService(db *gorm.DB, contentSvc *ContentService) *ArmyProgressionService {
	return &ArmyProgressionService{db: db, contentSvc: contentSvc}
}

func (s *ArmyProgressionService) RecalculateArmyCapacity(ctx context.Context, profileID uint) (ArmyCapacitySnapshot, error) {
	return NewArmyService(s.db, s.contentSvc).Capacity(ctx, profileID)
}

func (s *ArmyProgressionService) RecalculateTrainingSlots(ctx context.Context, profileID uint) (int, error) {
	capacity, err := s.RecalculateArmyCapacity(ctx, profileID)
	return capacity.TrainingSlots, err
}

func (s *ArmyProgressionService) RecalculateFormationSlots(ctx context.Context, profileID uint) ([]ArmyProgressionChange, error) {
	return s.syncFormationSlots(ctx, profileID)
}

func (s *ArmyProgressionService) UnlockFormationSlots(ctx context.Context, profileID uint) ([]ArmyProgressionChange, error) {
	return s.syncFormationSlots(ctx, profileID)
}

func (s *ArmyProgressionService) UpgradeSlotCapacities(ctx context.Context, profileID uint) ([]ArmyProgressionChange, error) {
	return s.syncFormationSlots(ctx, profileID)
}

func (s *ArmyProgressionService) ValidateFormation(ctx context.Context, profileID uint, formationID uint) (*ArmyFormationDetail, error) {
	if _, err := s.syncFormationSlots(ctx, profileID); err != nil {
		return nil, err
	}
	detail, err := NewArmyService(s.db, s.contentSvc).Formation(ctx, profileID, formationID)
	if err != nil {
		return nil, err
	}
	usedBySlot := map[uint]int{}
	for _, assignment := range detail.Assignments {
		usedBySlot[assignment.SlotID] += assignment.CapacityUsed
	}
	for _, slot := range detail.Slots {
		if slot.IsLocked && usedBySlot[slot.ID] > 0 {
			return nil, codedError("slot_locked_with_assignment", "Une formation contient une unite dans un slot verrouille.")
		}
		if usedBySlot[slot.ID] > effectiveSlotCapacity(slot) {
			return nil, codedError("slot_capacity_exceeded", "La capacite du slot est depassee.")
		}
	}
	return detail, nil
}

func (s *ArmyProgressionService) SyncFormationSlotsAfterBuildingUpgrade(ctx context.Context, profileID uint, buildingCode string, buildingLevel int) ([]ArmyProgressionChange, error) {
	return s.syncFormationSlots(ctx, profileID)
}

func (s *ArmyProgressionService) SyncFormationSlotsAfterResearchCompleted(ctx context.Context, profileID uint, researchCode string) ([]ArmyProgressionChange, error) {
	return s.syncFormationSlots(ctx, profileID)
}

func (s *ArmyProgressionService) Progression(ctx context.Context, profileID uint) (*ArmyProgressionSnapshot, error) {
	if _, err := s.syncFormationSlots(ctx, profileID); err != nil {
		return nil, err
	}
	capacity, err := NewArmyService(s.db, s.contentSvc).Capacity(ctx, profileID)
	if err != nil {
		return nil, err
	}
	formations, err := repositories.NewArmyFormationRepository(s.db).List(ctx, profileID)
	if err != nil {
		return nil, err
	}
	repo := repositories.NewArmySlotRepository(s.db)
	summaries := make([]ArmyFormationProgressionSummary, 0, len(formations))
	for _, formation := range formations {
		slots, err := repo.ListSlots(ctx, formation.ID)
		if err != nil {
			return nil, err
		}
		summary := ArmyFormationProgressionSummary{Type: formation.Type, Slots: slots}
		for _, slot := range slots {
			summary.MaxSlots++
			if slot.IsLocked {
				summary.LockedSlots++
				summary.NextUnlocks = append(summary.NextUnlocks, map[string]any{
					"slotType":  slot.SlotType,
					"slotIndex": slot.SlotIndex,
					"condition": slot.LockedReason,
				})
			} else {
				summary.UnlockedSlots++
			}
		}
		summaries = append(summaries, summary)
	}
	return &ArmyProgressionSnapshot{
		ArmyCapacity: capacity.Capacity, ArmyCapacityUsed: capacity.Used,
		TrainingSlots: capacity.TrainingSlots, Formations: summaries,
	}, nil
}

func (s *ArmyProgressionService) ProgressionRules(ctx context.Context, activeOnly bool) ([]models.ArmyFormationProgressionRule, error) {
	return repositories.NewArmyProgressionRuleRepository(s.db).List(ctx, activeOnly)
}

func (s *ArmyProgressionService) SaveProgressionRule(ctx context.Context, rule *models.ArmyFormationProgressionRule) error {
	normalizeProgressionRule(rule)
	return repositories.NewArmyProgressionRuleRepository(s.db).Save(ctx, rule)
}

func (s *ArmyProgressionService) DeleteProgressionRule(ctx context.Context, id uint) error {
	return repositories.NewArmyProgressionRuleRepository(s.db).Delete(ctx, id)
}

func (s *ArmyProgressionService) SeedPreview(ctx context.Context) (map[string]any, error) {
	existing, err := repositories.NewArmyProgressionRuleRepository(s.db).List(ctx, false)
	if err != nil {
		return nil, err
	}
	bySource := map[string]bool{}
	for _, row := range existing {
		bySource[row.SourceCode] = true
	}
	defaults := DefaultArmyProgressionRules()
	missing := 0
	for _, row := range defaults {
		if !bySource[row.SourceCode] {
			missing++
		}
	}
	return map[string]any{"defaults": len(defaults), "existing": len(existing), "missing": missing}, nil
}

func (s *ArmyProgressionService) SeedDefaultRules(ctx context.Context) (map[string]any, error) {
	repo := repositories.NewArmyProgressionRuleRepository(s.db)
	count := 0
	for _, rule := range DefaultArmyProgressionRules() {
		row := rule
		normalizeProgressionRule(&row)
		if err := repo.UpsertBySourceCode(ctx, &row); err != nil {
			return nil, err
		}
		count++
	}
	return map[string]any{"seeded": count}, nil
}

func (s *ArmyProgressionService) SeedStatus(ctx context.Context) (map[string]any, error) {
	return s.SeedPreview(ctx)
}

func (s *ArmyProgressionService) syncFormationSlots(ctx context.Context, profileID uint) ([]ArmyProgressionChange, error) {
	if err := NewArmyService(s.db, s.contentSvc).ensureFormationRows(ctx, profileID); err != nil {
		return nil, err
	}
	state, err := s.loadState(ctx, profileID)
	if err != nil {
		return nil, err
	}
	formations, err := repositories.NewArmyFormationRepository(s.db).List(ctx, profileID)
	if err != nil {
		return nil, err
	}
	rules, err := repositories.NewArmyProgressionRuleRepository(s.db).List(ctx, true)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		if _, err := s.SeedDefaultRules(ctx); err != nil {
			return nil, err
		}
		rules, err = repositories.NewArmyProgressionRuleRepository(s.db).List(ctx, true)
		if err != nil {
			return nil, err
		}
	}
	byFormation := map[string][]models.ArmyFormationProgressionRule{}
	for _, rule := range rules {
		byFormation[rule.FormationType] = append(byFormation[rule.FormationType], rule)
	}
	changes := []ArmyProgressionChange{}
	for _, formation := range formations {
		next, err := s.syncFormation(ctx, state, formation, byFormation[formation.Type])
		if err != nil {
			return nil, err
		}
		changes = append(changes, next...)
		if err := NewArmyService(s.db, s.contentSvc).RecalculateFormation(ctx, formation.ID); err != nil {
			return nil, err
		}
	}
	return changes, nil
}

func (s *ArmyProgressionService) syncFormation(ctx context.Context, state armyProgressionState, formation models.ArmyFormation, rules []models.ArmyFormationProgressionRule) ([]ArmyProgressionChange, error) {
	slotRepo := repositories.NewArmySlotRepository(s.db)
	changes := []ArmyProgressionChange{}
	slotRules := make([]models.ArmyFormationProgressionRule, 0)
	bonusRules := make([]models.ArmyFormationProgressionRule, 0)
	for _, rule := range rules {
		switch rule.RuleType {
		case armyRuleCapacityBonus:
			bonusRules = append(bonusRules, rule)
		default:
			slotRules = append(slotRules, rule)
		}
	}
	if len(slotRules) > 12 {
		slotRules = slotRules[:12]
	}
	for _, rule := range slotRules {
		unlocked, reason := s.ruleUnlocked(state, rule)
		base := maxInt(0, rule.BaseCapacity)
		maxCap := rule.MaxCapacity
		if maxCap <= 0 {
			maxCap = base
		}
		bonusTotal := 0
		modifiers := []map[string]any{}
		for _, bonus := range bonusRules {
			ok, _ := s.ruleUnlocked(state, bonus)
			if !ok || !capacityBonusApplies(bonus, rule.SlotType) {
				continue
			}
			bonusTotal += bonus.CapacityBonus
			modifiers = append(modifiers, map[string]any{"sourceCode": bonus.SourceCode, "bonus": bonus.CapacityBonus})
		}
		current := base + bonusTotal
		if maxCap > 0 && current > maxCap {
			current = maxCap
		}
		slot, err := slotRepo.FindSlotBySource(ctx, formation.ID, rule.SourceCode)
		if errorsIsNotFound(err) {
			slot, err = slotRepo.FindSlotByIndexType(ctx, formation.ID, rule.SlotIndex, rule.SlotType)
		}
		if err != nil && !errorsIsNotFound(err) {
			return nil, err
		}
		now := time.Now().UTC()
		beforeLocked := true
		beforeCapacity := 0
		if slot == nil || errorsIsNotFound(err) {
			slot = &models.ArmyFormationSlot{
				FormationID: formation.ID,
				SlotIndex:   rule.SlotIndex,
				SlotType:    rule.SlotType,
				CreatedAt:   now,
			}
		} else {
			beforeLocked = slot.IsLocked
			beforeCapacity = effectiveSlotCapacity(*slot)
		}
		slot.SlotLevel = slotLevelForCapacity(base, current)
		slot.BaseCapacity = base
		slot.CurrentCapacity = current
		slot.MaxCapacity = maxCap
		slot.Capacity = current
		slot.AllowedUnitTypesJSON = mustArmyJSON(allowedTypesForSlot(rule.SlotType))
		slot.UnlockRequirementJSON = mustArmyJSON(unlockRequirementPayload(rule))
		slot.CapacityModifiersJSON = mustArmyJSON(modifiers)
		slot.SourceType = firstNonEmptyString(rule.SourceType, "progression_rule")
		slot.SourceCode = rule.SourceCode
		slot.IsLocked = !unlocked
		if unlocked {
			slot.Status = "unlocked"
			slot.LockedReason = ""
			if slot.UnlockedAt == nil {
				slot.UnlockedAt = &now
			}
		} else {
			slot.Status = "locked"
			slot.LockedReason = reason
		}
		var saveErr error
		if slot.ID == 0 {
			saveErr = slotRepo.CreateSlot(ctx, slot)
		} else {
			saveErr = slotRepo.SaveSlot(ctx, slot)
		}
		if saveErr != nil {
			return nil, saveErr
		}
		afterCapacity := effectiveSlotCapacity(*slot)
		if beforeLocked != slot.IsLocked || beforeCapacity != afterCapacity {
			changes = append(changes, ArmyProgressionChange{
				FormationType: formation.Type, SlotType: slot.SlotType, SlotIndex: slot.SlotIndex, SourceCode: slot.SourceCode,
				ChangeType:   changeTypeForSlot(beforeLocked, slot.IsLocked, beforeCapacity, afterCapacity),
				BeforeLocked: beforeLocked, AfterLocked: slot.IsLocked, BeforeCapacity: beforeCapacity, AfterCapacity: afterCapacity,
			})
		}
	}
	return changes, nil
}

func (s *ArmyProgressionService) loadState(ctx context.Context, profileID uint) (armyProgressionState, error) {
	var profile models.ProfileGamer
	if err := s.db.WithContext(ctx).First(&profile, profileID).Error; err != nil {
		return armyProgressionState{}, err
	}
	var buildings []models.PlayerBuilding
	if err := s.db.WithContext(ctx).Where("profile_gamer_id = ?", profileID).Find(&buildings).Error; err != nil {
		return armyProgressionState{}, err
	}
	var research []models.PlayerResearch
	if err := s.db.WithContext(ctx).Where("profile_gamer_id = ?", profileID).Find(&research).Error; err != nil {
		return armyProgressionState{}, err
	}
	state := armyProgressionState{
		Profile:       profile,
		BuildingLevel: map[string]int{},
		ResearchDone:  map[string]bool{},
	}
	for _, building := range buildings {
		if building.Level > state.BuildingLevel[building.ContentID] {
			state.BuildingLevel[building.ContentID] = building.Level
		}
	}
	for _, row := range research {
		state.ResearchDone[row.ContentID] = true
	}
	return state, nil
}

func (s *ArmyProgressionService) ruleUnlocked(state armyProgressionState, rule models.ArmyFormationProgressionRule) (bool, string) {
	missing := []string{}
	if rule.UnlockBuildingCode != "" && state.BuildingLevel[rule.UnlockBuildingCode] < rule.UnlockBuildingLevel {
		missing = append(missing, fmt.Sprintf("%s niveau %d", rule.UnlockBuildingCode, maxInt(1, rule.UnlockBuildingLevel)))
	}
	if rule.RequiredGuildLevel > 0 && state.BuildingLevel["building_guild_hq"] < rule.RequiredGuildLevel {
		missing = append(missing, fmt.Sprintf("building_guild_hq niveau %d", rule.RequiredGuildLevel))
	}
	if rule.UnlockResearchCode != "" && !state.ResearchDone[rule.UnlockResearchCode] {
		missing = append(missing, rule.UnlockResearchCode)
	}
	if rule.UnlockPlayerLevel > 0 && state.Profile.Level < rule.UnlockPlayerLevel {
		missing = append(missing, fmt.Sprintf("niveau joueur %d", rule.UnlockPlayerLevel))
	}
	// Current model has only ProfileGamer.Level. Use it as city level until a separate city level exists.
	if rule.UnlockCityLevel > 0 && state.Profile.Level < rule.UnlockCityLevel {
		missing = append(missing, fmt.Sprintf("niveau ville %d", rule.UnlockCityLevel))
	}
	if len(missing) > 0 {
		return false, "Requis: " + strings.Join(missing, " + ")
	}
	return true, ""
}

func DefaultArmyProgressionRules() []models.ArmyFormationProgressionRule {
	rules := []models.ArmyFormationProgressionRule{}
	addSlot := func(formation string, index int, slotType string, capacity int, source string, order int, opts ...func(*models.ArmyFormationProgressionRule)) {
		rule := models.ArmyFormationProgressionRule{
			FormationType: formation, RuleType: armyRuleSlotUnlock, SlotType: slotType, SlotIndex: index,
			BaseCapacity: capacity, MaxCapacity: capacity + 50, SourceType: "default_seed", SourceCode: source,
			SortOrder: order, IsActive: true,
		}
		for _, opt := range opts {
			opt(&rule)
		}
		rules = append(rules, rule)
	}
	addBonus := func(formation string, target string, bonus int, source string, order int, opts ...func(*models.ArmyFormationProgressionRule)) {
		rule := models.ArmyFormationProgressionRule{
			FormationType: formation, RuleType: armyRuleCapacityBonus, TargetSlotType: target,
			CapacityBonus: bonus, SourceType: "default_seed", SourceCode: source, SortOrder: order, IsActive: true,
		}
		for _, opt := range opts {
			opt(&rule)
		}
		rules = append(rules, rule)
	}
	building := func(code string, level int) func(*models.ArmyFormationProgressionRule) {
		return func(rule *models.ArmyFormationProgressionRule) {
			rule.UnlockBuildingCode, rule.UnlockBuildingLevel = code, level
		}
	}
	research := func(code string) func(*models.ArmyFormationProgressionRule) {
		return func(rule *models.ArmyFormationProgressionRule) { rule.UnlockResearchCode = code }
	}
	guild := func(level int) func(*models.ArmyFormationProgressionRule) {
		return func(rule *models.ArmyFormationProgressionRule) { rule.RequiredGuildLevel = level }
	}

	addSlot("defense", 1, "infantry_slot", 10, "defense_base_infantry_1", 10)
	addSlot("defense", 2, "drone_slot", 10, "defense_base_drone_2", 20)
	addSlot("defense", 3, "support_slot", 5, "defense_base_support_3", 30)
	addSlot("defense", 4, "infantry_slot", 10, "defense_command_infantry_4", 40, building("building_nexus_core", 2))
	addSlot("defense", 5, "anti_sabotage_slot", 8, "defense_surveillance_anti_sabotage_5", 50, building("building_surveillance_tower", 2))
	addSlot("defense", 6, "heavy_slot", 15, "defense_wall_heavy_6", 60, building("building_holo_wall", 3))
	addBonus("defense", "all", 5, "defense_urban_defense_1_bonus", 70, research("research_urban_defense_1"))
	addSlot("defense", 7, "cyber_slot", 5, "defense_counter_espionage_cyber_7", 80, research("research_counter_espionage_1"))
	addSlot("defense", 8, "commander_slot", 1, "defense_command_commander_8", 90, building("building_nexus_core", 5))
	addBonus("defense", "infantry_slot", 10, "defense_urban_defense_2_infantry_bonus", 100, research("research_urban_defense_2"))
	addBonus("defense", "drone_slot", 10, "defense_urban_defense_2_drone_bonus", 101, research("research_urban_defense_2"))
	addSlot("defense", 9, "free_any_slot", 20, "defense_command_free_any_9", 110, building("building_nexus_core", 10))

	addSlot("attack", 1, "infantry_slot", 10, "attack_base_infantry_1", 10)
	addSlot("attack", 2, "drone_slot", 10, "attack_base_drone_2", 20)
	addSlot("attack", 3, "free_light_slot", 10, "attack_base_light_3", 30)
	addSlot("attack", 4, "infantry_slot", 10, "attack_barracks_infantry_4", 40, building("building_barracks", 2))
	addSlot("attack", 5, "drone_slot", 10, "attack_drone_factory_drone_5", 50, building("building_drone_factory", 2))
	addSlot("attack", 6, "heavy_slot", 15, "attack_mechanized_heavy_6", 60, building("building_mechanized_workshop", 1))
	addBonus("attack", "heavy_slot", 10, "attack_mechanized_assault_1_bonus", 70, research("research_mechanized_assault_1"))
	addSlot("attack", 7, "commander_slot", 1, "attack_command_commander_7", 80, building("building_nexus_core", 5))
	addBonus("attack", "all", 5, "attack_offensive_tactics_1_bonus", 90, research("research_offensive_tactics_1"))
	addSlot("attack", 8, "free_any_slot", 25, "attack_command_free_any_8", 100, building("building_nexus_core", 10))

	addSlot("mission", 1, "infantry_slot", 10, "mission_base_infantry_1", 10)
	addSlot("mission", 2, "support_slot", 5, "mission_base_support_2", 20)
	addSlot("mission", 3, "scout_slot", 8, "mission_barracks_scout_3", 30, building("building_barracks", 2))
	addSlot("mission", 4, "cyber_slot", 5, "mission_lab_cyber_4", 40, building("building_research_lab", 2))
	addSlot("mission", 5, "drone_slot", 10, "mission_tactical_exploration_drone_5", 50, research("research_tactical_exploration_1"))
	addBonus("mission", "all", 5, "mission_command_capacity_1_bonus", 60, building("building_nexus_core", 4))
	addSlot("mission", 6, "support_slot", 8, "mission_logistics_support_6", 70, research("research_mission_logistics_1"))
	addSlot("mission", 7, "free_light_slot", 15, "mission_command_light_7", 80, building("building_nexus_core", 8))

	addSlot("exploration", 1, "scout_slot", 10, "exploration_base_scout_1", 10)
	addSlot("exploration", 2, "drone_slot", 10, "exploration_base_drone_2", 20)
	addSlot("exploration", 3, "drone_slot", 10, "exploration_surveillance_drone_3", 30, building("building_surveillance_tower", 2))
	addSlot("exploration", 4, "support_slot", 5, "exploration_cartography_support_4", 40, research("research_cartography_1"))
	addBonus("exploration", "drone_slot", 10, "exploration_drone_recon_1_bonus", 50, research("research_drone_reconnaissance_1"))
	addSlot("exploration", 5, "cyber_slot", 5, "exploration_ai_center_cyber_5", 60, building("building_ai_center", 3))
	addSlot("exploration", 6, "free_light_slot", 15, "exploration_command_light_6", 70, building("building_nexus_core", 6))

	addSlot("anti_sabotage", 1, "drone_slot", 10, "anti_sabotage_base_drone_1", 10)
	addSlot("anti_sabotage", 2, "cyber_slot", 5, "anti_sabotage_base_cyber_2", 20)
	addSlot("anti_sabotage", 3, "drone_slot", 10, "anti_sabotage_surveillance_drone_3", 30, building("building_surveillance_tower", 2))
	addSlot("anti_sabotage", 4, "cyber_slot", 5, "anti_sabotage_ai_center_cyber_4", 40, building("building_ai_center", 2))
	addBonus("anti_sabotage", "cyber_slot", 5, "anti_sabotage_counter_espionage_1_bonus", 50, research("research_counter_espionage_1"))
	addSlot("anti_sabotage", 5, "support_slot", 5, "anti_sabotage_network_security_support_5", 60, research("research_network_security_1"))
	addSlot("anti_sabotage", 6, "anti_sabotage_slot", 12, "anti_sabotage_surveillance_special_6", 70, building("building_surveillance_tower", 5))
	addSlot("anti_sabotage", 7, "commander_slot", 1, "anti_sabotage_command_commander_7", 80, building("building_nexus_core", 8))

	addSlot("guild_raid", 1, "infantry_slot", 20, "guild_raid_base_infantry_1", 10, guild(1))
	addSlot("guild_raid", 2, "drone_slot", 20, "guild_raid_base_drone_2", 20, guild(1))
	addSlot("guild_raid", 3, "support_slot", 10, "guild_raid_base_support_3", 30, guild(1))
	addSlot("guild_raid", 4, "infantry_slot", 20, "guild_raid_hq_infantry_4", 40, guild(2))
	addSlot("guild_raid", 5, "heavy_slot", 20, "guild_raid_hq_heavy_5", 50, guild(3))
	addSlot("guild_raid", 6, "commander_slot", 1, "guild_raid_hq_commander_6", 60, guild(5))
	addBonus("guild_raid", "all", 10, "guild_raid_coordination_1_bonus", 70, research("research_guild_coordination_1"))
	addSlot("guild_raid", 7, "cyber_slot", 10, "guild_raid_ai_raid_cyber_7", 80, research("research_ai_raid_1"))
	addSlot("guild_raid", 8, "free_any_slot", 30, "guild_raid_hq_free_any_8", 90, guild(10))

	return rules
}

func normalizeProgressionRule(rule *models.ArmyFormationProgressionRule) {
	rule.FormationType = strings.TrimSpace(rule.FormationType)
	rule.RuleType = strings.TrimSpace(rule.RuleType)
	if rule.RuleType == "" {
		rule.RuleType = armyRuleSlotUnlock
	}
	rule.SlotType = strings.TrimSpace(rule.SlotType)
	rule.TargetSlotType = strings.TrimSpace(rule.TargetSlotType)
	rule.SourceCode = strings.TrimSpace(rule.SourceCode)
	if rule.SourceCode == "" {
		rule.SourceCode = fmt.Sprintf("%s_%s_%d_%s", rule.FormationType, rule.RuleType, rule.SortOrder, rule.SlotType)
	}
	if rule.SourceType == "" {
		rule.SourceType = "admin"
	}
}

func capacityBonusApplies(rule models.ArmyFormationProgressionRule, slotType string) bool {
	target := strings.TrimSpace(rule.TargetSlotType)
	if target == "all" && slotType == "commander_slot" {
		return false
	}
	return target == "" || target == "all" || target == slotType
}

func unlockRequirementPayload(rule models.ArmyFormationProgressionRule) map[string]any {
	return map[string]any{
		"buildingCode":       rule.UnlockBuildingCode,
		"buildingLevel":      rule.UnlockBuildingLevel,
		"researchCode":       rule.UnlockResearchCode,
		"playerLevel":        rule.UnlockPlayerLevel,
		"cityLevel":          rule.UnlockCityLevel,
		"requiredGuildLevel": rule.RequiredGuildLevel,
	}
}

func slotLevelForCapacity(base int, current int) int {
	switch {
	case current >= base+20:
		return 3
	case current >= base+5:
		return 2
	default:
		return 1
	}
}

func changeTypeForSlot(beforeLocked bool, afterLocked bool, beforeCapacity int, afterCapacity int) string {
	if beforeLocked && !afterLocked {
		return "slot_unlocked"
	}
	if beforeCapacity != afterCapacity {
		return "slot_capacity_changed"
	}
	return "slot_synced"
}

func effectiveSlotCapacity(slot models.ArmyFormationSlot) int {
	if slot.CurrentCapacity > 0 {
		return slot.CurrentCapacity
	}
	if slot.Capacity > 0 {
		return slot.Capacity
	}
	return slot.BaseCapacity
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func errorsIsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
