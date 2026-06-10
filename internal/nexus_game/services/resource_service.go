package services

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"cgwm/battle/internal/nexus_game/models"
	"cgwm/battle/internal/nexus_game/repositories"

	"gorm.io/gorm"
)

const (
	DefaultStorageCapacity = int64(1000)
	NexusCoreContentID     = "building_nexus_core"
)

type ResourceDefinition struct {
	Code             string
	Name             string
	Category         string
	IsConsumable     bool
	IsRare           bool
	IsStorageLimited bool
	InitialAmount    int64
	DailyGrantAmount int64
	SortOrder        int
}

var officialResourceDefinitions = []ResourceDefinition{
	{Code: "population", Name: "Population", Category: "city", IsStorageLimited: false, InitialAmount: 0, SortOrder: 10},
	{Code: "credits", Name: "Credits", Category: "basic", IsConsumable: true, IsStorageLimited: true, InitialAmount: 450, DailyGrantAmount: 120, SortOrder: 15},
	{Code: "food", Name: "Food", Category: "basic", IsConsumable: true, IsStorageLimited: true, InitialAmount: 500, DailyGrantAmount: 150, SortOrder: 20},
	{Code: "energy", Name: "Energy", Category: "basic", IsConsumable: true, IsStorageLimited: true, InitialAmount: 300, DailyGrantAmount: 100, SortOrder: 30},
	{Code: "metal", Name: "Metal", Category: "basic", IsConsumable: true, IsStorageLimited: true, InitialAmount: 800, DailyGrantAmount: 120, SortOrder: 40},
	{Code: "components", Name: "Components", Category: "basic", IsConsumable: true, IsStorageLimited: true, InitialAmount: 120, DailyGrantAmount: 20, SortOrder: 50},
	{Code: "data", Name: "Data", Category: "ai", IsConsumable: true, IsStorageLimited: true, InitialAmount: 100, DailyGrantAmount: 25, SortOrder: 60},
	{Code: "influence", Name: "Influence", Category: "social", IsConsumable: true, IsStorageLimited: true, InitialAmount: 25, DailyGrantAmount: 5, SortOrder: 70},
	{Code: "guild_marks", Name: "Guild Marks", Category: "guild", IsConsumable: true, IsRare: true, IsStorageLimited: true, InitialAmount: 0, DailyGrantAmount: 0, SortOrder: 80},
	{Code: "tokens", Name: "Tokens", Category: "ai", IsConsumable: true, IsStorageLimited: false, InitialAmount: 50, DailyGrantAmount: 25, SortOrder: 90},
	{Code: "provider_budget", Name: "Provider Budget", Category: "ai", IsConsumable: true, IsStorageLimited: false, InitialAmount: 0, DailyGrantAmount: 0, SortOrder: 100},
	{Code: "inference_credit", Name: "Inference Credit", Category: "ai", IsConsumable: true, IsStorageLimited: true, InitialAmount: 20, DailyGrantAmount: 10, SortOrder: 110},
	{Code: "local_compute", Name: "Local Compute", Category: "ai", IsConsumable: true, IsStorageLimited: true, InitialAmount: 10, DailyGrantAmount: 5, SortOrder: 120},
	{Code: "agent_focus", Name: "Agent Focus", Category: "ai", IsConsumable: true, IsStorageLimited: false, InitialAmount: 5, DailyGrantAmount: 3, SortOrder: 130},
	{Code: "quantum_core", Name: "Quantum Core", Category: "rare", IsConsumable: true, IsRare: true, IsStorageLimited: true, InitialAmount: 0, DailyGrantAmount: 0, SortOrder: 140},
	{Code: "neural_fiber", Name: "Neural Fiber", Category: "rare", IsConsumable: true, IsRare: true, IsStorageLimited: true, InitialAmount: 0, DailyGrantAmount: 0, SortOrder: 150},
	{Code: "void_fragment", Name: "Void Fragment", Category: "rare", IsConsumable: true, IsRare: true, IsStorageLimited: true, InitialAmount: 0, DailyGrantAmount: 0, SortOrder: 160},
}

func OfficialResourceDefinitions() []ResourceDefinition {
	defs := make([]ResourceDefinition, len(officialResourceDefinitions))
	copy(defs, officialResourceDefinitions)
	return defs
}

// Level1StarterAllocation returns the starter resources granted once when a profile
// receives its initial allocation. Only positive starter amounts are exposed.
func Level1StarterAllocation() map[string]int64 {
	starter := map[string]int64{}
	for _, def := range officialResourceDefinitions {
		if def.InitialAmount > 0 {
			starter[def.Code] = def.InitialAmount
		}
	}
	return starter
}

func resourceDefinitionByCode(code string) (ResourceDefinition, bool) {
	for _, def := range officialResourceDefinitions {
		if def.Code == code {
			return def, true
		}
	}
	return ResourceDefinition{}, false
}

type ResourceService struct {
	db *gorm.DB
}

type buildingEffect struct {
	EffectType    string  `json:"effectType"`
	Target        string  `json:"target"`
	Stat          string  `json:"stat"`
	ValueBase     float64 `json:"valueBase"`
	ValuePerLevel float64 `json:"valuePerLevel"`
	Value         float64 `json:"value"`
	IsPercentage  bool    `json:"isPercentage"`
}

type productionAccumulator struct {
	ResourceProduction  map[string]float64
	ResourceConsumption map[string]float64
	PopulationCapacity  int
	EnergyProduction    int
	EnergyConsumption   int
}

type researchModifiers struct {
	BuildingBonuses           map[string][]buildingEffect
	StorageCapacityMultiplier float64
}

func NewResourceService(db *gorm.DB) *ResourceService {
	return &ResourceService{db: db}
}

func newProductionAccumulator() productionAccumulator {
	return productionAccumulator{
		ResourceProduction:  make(map[string]float64),
		ResourceConsumption: make(map[string]float64),
	}
}

func normalizeResourceCode(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	key = strings.ReplaceAll(key, "-", "")
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, " ", "")

	switch key {
	case "food":
		return "food"
	case "energy", "energyproduction":
		return "energy"
	case "metal":
		return "metal"
	case "credits", "credit":
		return "credits"
	case "data", "dataproduction":
		return "data"
	case "components", "component":
		return "components"
	case "influence":
		return "influence"
	case "tokens", "token":
		return "tokens"
	case "population":
		return "population"
	case "providerbudget":
		return "provider_budget"
	case "inferencecredit":
		return "inference_credit"
	case "localcompute":
		return "local_compute"
	case "agentfocus":
		return "agent_focus"
	case "guildmarks":
		return "guild_marks"
	case "quantumcore", "quantumcoreproduction":
		return "quantum_core"
	case "neuralfiber":
		return "neural_fiber"
	case "voidfragment":
		return "void_fragment"
	default:
		return ""
	}
}

func resolveEffectValue(effect buildingEffect, level int) float64 {
	if level < 1 {
		level = 1
	}
	if effect.Value != 0 {
		return effect.Value
	}
	return effect.ValueBase + float64(level-1)*effect.ValuePerLevel
}

func percentToFraction(value float64) float64 {
	if value > 1 || value < -1 {
		return value / 100.0
	}
	return value
}

func applyPercentBonus(base float64, percent float64) float64 {
	return base + (base * percentToFraction(percent))
}

func applyScaledDelta(base float64, delta float64, isPercentage bool) float64 {
	if !isPercentage {
		return base + delta
	}
	if base <= 0 {
		return base
	}
	return applyPercentBonus(base, delta)
}

func applyStarterFallbackProduction(acc *productionAccumulator, building models.PlayerBuilding) {
	if building.Level <= 0 {
		return
	}
	levelFactor := float64(building.Level)
	switch building.ContentID {
	case "building_modular_habitat":
		acc.PopulationCapacity += 50 * building.Level
	case "building_solar_plant":
		acc.EnergyProduction += 80 * building.Level
		acc.ResourceProduction["energy"] += 80 * levelFactor
	case "building_vertical_farm":
		acc.ResourceProduction["food"] += 70 * levelFactor
	case "building_nexus_market":
		acc.ResourceProduction["credits"] += 25 * levelFactor
	}
}

func mergeProductionAccumulator(dst *productionAccumulator, src productionAccumulator) {
	for code, value := range src.ResourceProduction {
		dst.ResourceProduction[code] += value
	}
	for code, value := range src.ResourceConsumption {
		dst.ResourceConsumption[code] += value
	}
	dst.PopulationCapacity += src.PopulationCapacity
	dst.EnergyProduction += src.EnergyProduction
	dst.EnergyConsumption += src.EnergyConsumption
}

func loadResearchModifiers(ctx context.Context, tx *gorm.DB, profileID uint) (researchModifiers, error) {
	mods := researchModifiers{
		BuildingBonuses:           map[string][]buildingEffect{},
		StorageCapacityMultiplier: 1.0,
	}

	var playerResearch []models.PlayerResearch
	if err := tx.WithContext(ctx).Where("profile_gamer_id = ?", profileID).Find(&playerResearch).Error; err != nil {
		return mods, err
	}
	if len(playerResearch) == 0 {
		return mods, nil
	}

	contentIDs := make([]string, 0, len(playerResearch))
	for _, pr := range playerResearch {
		if pr.ContentID != "" {
			contentIDs = append(contentIDs, pr.ContentID)
		}
	}
	if len(contentIDs) == 0 {
		return mods, nil
	}

	var defs []models.ResearchDefinition
	if err := tx.WithContext(ctx).Where("content_id IN ?", contentIDs).Find(&defs).Error; err != nil {
		return mods, err
	}

	for _, def := range defs {
		var effects []buildingEffect
		if err := json.Unmarshal([]byte(def.EffectsJSON), &effects); err != nil {
			continue
		}
		for _, effect := range effects {
			effectType := strings.ToLower(strings.TrimSpace(effect.EffectType))
			stat := strings.ToLower(strings.TrimSpace(effect.Stat))
			target := strings.ToLower(strings.TrimSpace(effect.Target))
			value := resolveEffectValue(effect, 1)
			switch effectType {
			case "buildingbonus":
				mods.BuildingBonuses[target] = append(mods.BuildingBonuses[target], effect)
			case "resourcebonus":
				if stat == "storagecapacity" {
					mods.StorageCapacityMultiplier = applyPercentBonus(mods.StorageCapacityMultiplier, value)
				}
			case "statbonus":
				if stat == "storagecapacity" {
					mods.StorageCapacityMultiplier = applyPercentBonus(mods.StorageCapacityMultiplier, value)
				}
			}
		}
	}

	return mods, nil
}

func applyResearchBuildingBonuses(acc *productionAccumulator, bonuses []buildingEffect, level int) {
	for _, effect := range bonuses {
		value := resolveEffectValue(effect, level)
		if value == 0 {
			continue
		}
		effectType := strings.ToLower(strings.TrimSpace(effect.EffectType))
		if effectType != "buildingbonus" && effectType != "statbonus" {
			continue
		}
		stat := strings.ToLower(strings.TrimSpace(effect.Stat))
		if stat == "" {
			stat = strings.ToLower(strings.TrimSpace(effect.Target))
		}
		switch stat {
		case "populationcapacity":
			acc.PopulationCapacity = int(math.Round(applyPercentBonus(float64(acc.PopulationCapacity), value)))
		case "energyproduction":
			acc.EnergyProduction = int(math.Round(applyPercentBonus(float64(acc.EnergyProduction), value)))
			acc.ResourceProduction["energy"] = float64(acc.EnergyProduction)
		case "dataproduction":
			acc.ResourceProduction["data"] = applyPercentBonus(acc.ResourceProduction["data"], value)
		case "tradebonus", "credits":
			acc.ResourceProduction["credits"] = applyPercentBonus(acc.ResourceProduction["credits"], value)
		case "globalresourceproduction":
			for resourceCode, current := range acc.ResourceProduction {
				acc.ResourceProduction[resourceCode] = applyPercentBonus(current, value)
			}
		default:
			if code := normalizeResourceCode(stat); code != "" {
				acc.ResourceProduction[code] = applyPercentBonus(acc.ResourceProduction[code], value)
			}
		}
	}
}

func applyBuildingEffects(acc *productionAccumulator, effects []buildingEffect, level int) {
	for _, effect := range effects {
		value := resolveEffectValue(effect, level)
		if value == 0 {
			continue
		}

		effectType := strings.ToLower(strings.TrimSpace(effect.EffectType))
		target := strings.TrimSpace(effect.Target)
		stat := strings.TrimSpace(effect.Stat)

		switch effectType {
		case "resourceproduction":
			code := normalizeResourceCode(target)
			if code == "" {
				code = normalizeResourceCode(stat)
			}
			if code == "" {
				continue
			}
			acc.ResourceProduction[code] = applyScaledDelta(acc.ResourceProduction[code], value, effect.IsPercentage)
			if code == "energy" {
				acc.EnergyProduction = int(math.Round(acc.ResourceProduction["energy"]))
			}
		case "statbonus":
			metric := strings.ToLower(strings.TrimSpace(stat))
			if metric == "" {
				metric = strings.ToLower(strings.TrimSpace(target))
			}
			switch metric {
			case "populationcapacity":
				base := float64(acc.PopulationCapacity)
				acc.PopulationCapacity = int(math.Round(applyScaledDelta(base, value, effect.IsPercentage)))
			case "energyproduction":
				base := float64(acc.EnergyProduction)
				acc.EnergyProduction = int(math.Round(applyScaledDelta(base, value, effect.IsPercentage)))
				acc.ResourceProduction["energy"] = float64(acc.EnergyProduction)
			case "dataproduction":
				acc.ResourceProduction["data"] = applyScaledDelta(acc.ResourceProduction["data"], value, effect.IsPercentage)
			case "tradebonus":
				acc.ResourceProduction["credits"] = applyScaledDelta(acc.ResourceProduction["credits"], value, effect.IsPercentage)
			case "globalresourceproduction":
				if effect.IsPercentage {
					for resourceCode, current := range acc.ResourceProduction {
						acc.ResourceProduction[resourceCode] = applyScaledDelta(current, value, true)
					}
				}
			}
		}
	}
}

func (s *ResourceService) computeProductionState(ctx context.Context, tx *gorm.DB, profile models.ProfileGamer) (productionAccumulator, error) {
	acc := newProductionAccumulator()
	mods, err := loadResearchModifiers(ctx, tx, profile.ID)
	if err != nil {
		return acc, err
	}

	var buildings []models.PlayerBuilding
	if err := tx.WithContext(ctx).
		Where("profile_gamer_id = ? AND level > 0 AND is_constructing = ?", profile.ID, false).
		Find(&buildings).Error; err != nil {
		return acc, err
	}

	contentIDs := make([]string, 0, len(buildings))
	seen := map[string]bool{}
	for _, b := range buildings {
		if b.ContentID == "" || seen[b.ContentID] {
			continue
		}
		seen[b.ContentID] = true
		contentIDs = append(contentIDs, b.ContentID)
	}

	definitionsByContentID := map[string]models.BuildingDefinition{}
	if len(contentIDs) > 0 {
		var defs []models.BuildingDefinition
		if err := tx.WithContext(ctx).Where("content_id IN ?", contentIDs).Find(&defs).Error; err != nil {
			return acc, err
		}
		for _, def := range defs {
			definitionsByContentID[def.ContentID] = def
		}
	}

	for _, building := range buildings {
		local := newProductionAccumulator()
		def, ok := definitionsByContentID[building.ContentID]
		if !ok || strings.TrimSpace(def.EffectsJSON) == "" {
			applyStarterFallbackProduction(&local, building)
			mergeProductionAccumulator(&acc, local)
			continue
		}
		var effects []buildingEffect
		if err := json.Unmarshal([]byte(def.EffectsJSON), &effects); err != nil {
			applyStarterFallbackProduction(&local, building)
			mergeProductionAccumulator(&acc, local)
			continue
		}
		applyBuildingEffects(&local, effects, building.Level)
		if bonusEffects := mods.BuildingBonuses[strings.ToLower(building.ContentID)]; len(bonusEffects) > 0 {
			applyResearchBuildingBonuses(&local, bonusEffects, building.Level)
		}
		if bonusEffects := mods.BuildingBonuses["all"]; len(bonusEffects) > 0 {
			applyResearchBuildingBonuses(&local, bonusEffects, building.Level)
		}
		mergeProductionAccumulator(&acc, local)
	}

	if acc.EnergyProduction <= 0 {
		acc.EnergyProduction = int(math.Round(acc.ResourceProduction["energy"]))
	} else {
		acc.ResourceProduction["energy"] = float64(acc.EnergyProduction)
	}

	foodConsumption := math.Max(0, float64(profile.Population)*0.08)
	energyConsumption := 5 + max(0, profile.Population/20)
	acc.EnergyConsumption = energyConsumption
	acc.ResourceConsumption["food"] = foodConsumption
	acc.ResourceConsumption["energy"] = float64(energyConsumption)

	if acc.PopulationCapacity < profile.Population {
		acc.PopulationCapacity = profile.Population
	}

	return acc, nil
}

// SyncBuildingProduction recomputes production/consumption from completed buildings and
// updates per-tick balances. When applyAccrual is true, elapsed production since the last
// sync is applied directly to resource amounts.
func (s *ResourceService) SyncBuildingProduction(ctx context.Context, profileID uint, applyAccrual bool) error {
	if s.db == nil || profileID == 0 {
		return nil
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.ensureMissingResourceRows(ctx, tx, profileID, false); err != nil {
			return err
		}

		var profile models.ProfileGamer
		if err := tx.WithContext(ctx).First(&profile, profileID).Error; err != nil {
			return err
		}

		statsRepo := repositories.NewPlayerCityStatsRepository(tx)
		stats, err := statsRepo.GetOrCreate(ctx, profileID)
		if err != nil {
			return err
		}

		mods, err := loadResearchModifiers(ctx, tx, profileID)
		if err != nil {
			return err
		}

		acc, err := s.computeProductionState(ctx, tx, profile)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		elapsedSeconds := 0.0
		if applyAccrual {
			lastSync := stats.LastProductionSyncAt
			if !lastSync.IsZero() {
				elapsedSeconds = now.Sub(lastSync).Seconds()
				if elapsedSeconds < 0 {
					elapsedSeconds = 0
				}
				if elapsedSeconds > 21600 {
					elapsedSeconds = 21600
				}
			}
		}

		resourceRepo := repositories.NewPlayerResourceRepository(tx)
		resources, err := resourceRepo.List(ctx, profileID)
		if err != nil {
			return err
		}

		for _, resource := range resources {
			prod := acc.ResourceProduction[resource.ResourceCode]
			cons := acc.ResourceConsumption[resource.ResourceCode]
			balance := prod - cons

			resource.ProductionPerTick = prod
			resource.ConsumptionPerTick = cons
			resource.BalancePerTick = balance

			if applyAccrual && elapsedSeconds > 0 && balance != 0 {
				delta := int64(math.Round(balance * elapsedSeconds))
				if delta != 0 {
					nextAmount := resource.Amount + delta
					if nextAmount < 0 {
						nextAmount = 0
					}
					if resource.Capacity > 0 && nextAmount > resource.Capacity {
						nextAmount = resource.Capacity
					}
					resource.Amount = nextAmount
				}
			}

			if resource.Capacity == 0 {
				if def, ok := resourceDefinitionByCode(resource.ResourceCode); ok {
					if def.IsStorageLimited {
						resource.Capacity = int64(math.Round(float64(DefaultStorageCapacity) * mods.StorageCapacityMultiplier))
					}
				}
			} else if def, ok := resourceDefinitionByCode(resource.ResourceCode); ok && def.IsStorageLimited {
				resource.Capacity = int64(math.Round(float64(DefaultStorageCapacity) * mods.StorageCapacityMultiplier))
			}

			if err := resourceRepo.Save(ctx, &resource); err != nil {
				return err
			}
		}

		stats.FoodProduction = acc.ResourceProduction["food"]
		stats.FoodConsumption = acc.ResourceConsumption["food"]
		stats.FoodBalance = stats.FoodProduction - stats.FoodConsumption
		stats.LastProductionSyncAt = now
		if err := statsRepo.Save(ctx, stats); err != nil {
			return err
		}

		updates := map[string]any{
			"population_capacity": acc.PopulationCapacity,
			"energy_production":   acc.EnergyProduction,
			"energy_consumption":  acc.EnergyConsumption,
			"energy_balance":      acc.EnergyProduction - acc.EnergyConsumption,
			"updated_at":          now,
		}
		return tx.WithContext(ctx).Model(&models.ProfileGamer{}).Where("id = ?", profileID).Updates(updates).Error
	})
}

func (s *ResourceService) SeedDefaults(ctx context.Context) error {
	if s.db == nil {
		return nil
	}
	catalogRepo := repositories.NewResourceCatalogRepository(s.db)
	configRepo := repositories.NewDailyGrantConfigRepository(s.db)
	now := time.Now().UTC()
	for _, def := range officialResourceDefinitions {
		if err := catalogRepo.Upsert(ctx, &models.ResourceCatalog{
			Code:             def.Code,
			Name:             def.Name,
			Category:         def.Category,
			IsConsumable:     def.IsConsumable,
			IsRare:           def.IsRare,
			IsStorageLimited: def.IsStorageLimited,
			BaseStorage:      DefaultStorageCapacity,
			IsActive:         true,
			SortOrder:        def.SortOrder,
			UpdatedAt:        now,
		}); err != nil {
			return err
		}
		if err := configRepo.Upsert(ctx, &models.DailyGrantConfig{
			ResourceCode: def.Code,
			BaseAmount:   def.DailyGrantAmount,
			IsEnabled:    true,
			UpdatedAt:    now,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *ResourceService) EnsureInitialAllocation(ctx context.Context, profileID uint) error {
	if s.db == nil || profileID == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var log models.InitialAllocationLog
		err := tx.Where("profile_gamer_id = ?", profileID).First(&log).Error
		if err == nil {
			if err := s.ensureMissingResourceRows(ctx, tx, profileID, true); err != nil {
				return err
			}
			return s.ensureNexusCore(ctx, tx, profileID)
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := s.ensureInitialCityStats(ctx, tx, profileID); err != nil {
			return err
		}
		grantedResources, err := s.ensureStarterAllocation(ctx, tx, profileID)
		if err != nil {
			return err
		}
		if err := s.ensureNexusCore(ctx, tx, profileID); err != nil {
			return err
		}

		return tx.Create(&models.InitialAllocationLog{
			ProfileGamerID: profileID,
			Resources:      grantedResources,
			CreatedAt:      time.Now().UTC(),
		}).Error
	})
}

func (s *ResourceService) ensureStarterAllocation(ctx context.Context, tx *gorm.DB, profileID uint) (map[string]int64, error) {
	resourceRepo := repositories.NewPlayerResourceRepository(tx)
	transactionRepo := repositories.NewResourceTransactionRepository(tx)
	now := time.Now().UTC()
	granted := map[string]int64{}

	for _, def := range officialResourceDefinitions {
		capacity := DefaultStorageCapacity
		if !def.IsStorageLimited {
			capacity = 0
		}

		resource, err := resourceRepo.Get(ctx, profileID, def.Code)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			resource = &models.PlayerResource{
				ProfileGamerID: profileID,
				ResourceCode:   def.Code,
				Amount:         def.InitialAmount,
				Capacity:       capacity,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if err := resourceRepo.Create(ctx, resource); err != nil {
				return nil, err
			}
			granted[def.Code] = def.InitialAmount
			if def.InitialAmount > 0 {
				if err := transactionRepo.Create(ctx, &models.ResourceTransaction{
					ProfileGamerID:  profileID,
					ResourceCode:    def.Code,
					AmountDelta:     def.InitialAmount,
					BalanceAfter:    resource.Amount,
					TransactionType: "initial_allocation",
					Source:          "system",
					CreatedAt:       now,
				}); err != nil {
					return nil, err
				}
			}
			continue
		}

		capacityChanged := false
		if resource.Capacity != capacity {
			resource.Capacity = capacity
			capacityChanged = true
		}

		delta := int64(0)
		if def.InitialAmount > 0 && resource.Amount < def.InitialAmount {
			delta = def.InitialAmount - resource.Amount
			resource.Amount = def.InitialAmount
		}
		if delta > 0 || capacityChanged {
			if err := resourceRepo.Save(ctx, resource); err != nil {
				return nil, err
			}
		}
		if delta > 0 {
			granted[def.Code] = delta
			if err := transactionRepo.Create(ctx, &models.ResourceTransaction{
				ProfileGamerID:  profileID,
				ResourceCode:    def.Code,
				AmountDelta:     delta,
				BalanceAfter:    resource.Amount,
				TransactionType: "initial_allocation_topup",
				Source:          "system",
				CreatedAt:       now,
			}); err != nil {
				return nil, err
			}
		}
	}

	return granted, nil
}

func (s *ResourceService) ensureInitialCityStats(ctx context.Context, tx *gorm.DB, profileID uint) error {
	now := time.Now().UTC()
	if err := tx.WithContext(ctx).Model(&models.ProfileGamer{}).Where("id = ?", profileID).Updates(map[string]any{
		"population":          0,
		"population_capacity": 0,
		"morale":              50,
		"energy_production":   0,
		"energy_consumption":  0,
		"energy_balance":      0,
		"energy_stored":       0,
		"security":            50,
		"updated_at":          now,
	}).Error; err != nil {
		return err
	}
	statsRepo := repositories.NewPlayerCityStatsRepository(tx)
	stats, err := statsRepo.GetOrCreate(ctx, profileID)
	if err != nil {
		return err
	}
	stats.StorageCapacity = DefaultStorageCapacity
	stats.FoodProduction = 0
	stats.FoodConsumption = 0
	stats.FoodBalance = 0
	return statsRepo.Save(ctx, stats)
}

func (s *ResourceService) ensureMissingResourceRows(ctx context.Context, tx *gorm.DB, profileID uint, grantInitial bool) error {
	resourceRepo := repositories.NewPlayerResourceRepository(tx)
	transactionRepo := repositories.NewResourceTransactionRepository(tx)
	now := time.Now().UTC()
	for _, def := range officialResourceDefinitions {
		_, err := resourceRepo.Get(ctx, profileID, def.Code)
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		amount := int64(0)
		if grantInitial {
			amount = def.InitialAmount
		}
		capacity := DefaultStorageCapacity
		if !def.IsStorageLimited {
			capacity = 0
		}
		resource := &models.PlayerResource{
			ProfileGamerID: profileID,
			ResourceCode:   def.Code,
			Amount:         amount,
			Capacity:       capacity,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := resourceRepo.Create(ctx, resource); err != nil {
			return err
		}
		if grantInitial {
			if err := transactionRepo.Create(ctx, &models.ResourceTransaction{
				ProfileGamerID:  profileID,
				ResourceCode:    def.Code,
				AmountDelta:     amount,
				BalanceAfter:    resource.Amount,
				TransactionType: "initial_allocation",
				Source:          "system",
				CreatedAt:       now,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *ResourceService) ensureNexusCore(ctx context.Context, tx *gorm.DB, profileID uint) error {
	var building models.PlayerBuilding
	err := tx.WithContext(ctx).Where("profile_gamer_id = ? AND content_id = ?", profileID, NexusCoreContentID).First(&building).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return tx.WithContext(ctx).Create(&models.PlayerBuilding{
		ProfileGamerID: profileID,
		ContentID:      NexusCoreContentID,
		Level:          1,
		IsConstructing: false,
	}).Error
}

func (s *ResourceService) PlayerSnapshot(ctx context.Context, profileID uint) (map[string]any, error) {
	if err := s.EnsureInitialAllocation(ctx, profileID); err != nil {
		return nil, err
	}
	if err := s.SyncBuildingProduction(ctx, profileID, true); err != nil {
		return nil, err
	}
	catalogRepo := repositories.NewResourceCatalogRepository(s.db)
	resourceRepo := repositories.NewPlayerResourceRepository(s.db)
	transactionRepo := repositories.NewResourceTransactionRepository(s.db)
	statsRepo := repositories.NewPlayerCityStatsRepository(s.db)

	catalog, err := catalogRepo.List(ctx, true)
	if err != nil {
		return nil, err
	}
	resources, err := resourceRepo.List(ctx, profileID)
	if err != nil {
		return nil, err
	}
	stats, err := statsRepo.GetOrCreate(ctx, profileID)
	if err != nil {
		return nil, err
	}
	transactions, err := transactionRepo.List(ctx, profileID, 20)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"catalog":      catalog,
		"resources":    resources,
		"cityStats":    stats,
		"transactions": transactions,
	}, nil
}

func (s *ResourceService) ApplyResourceDelta(ctx context.Context, profileID uint, code string, delta int64, transactionType string, source string) (*models.PlayerResource, error) {
	def, ok := resourceDefinitionByCode(code)
	if !ok {
		return nil, errors.New("unknown resource")
	}
	var updated *models.PlayerResource
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		resourceRepo := repositories.NewPlayerResourceRepository(tx)
		transactionRepo := repositories.NewResourceTransactionRepository(tx)
		resource, err := resourceRepo.Add(ctx, profileID, code, delta, DefaultStorageCapacity, def.IsStorageLimited)
		if err != nil {
			return err
		}
		updated = resource
		return transactionRepo.Create(ctx, &models.ResourceTransaction{
			ProfileGamerID:  profileID,
			ResourceCode:    code,
			AmountDelta:     delta,
			BalanceAfter:    resource.Amount,
			TransactionType: transactionType,
			Source:          source,
			CreatedAt:       time.Now().UTC(),
		})
	})
	return updated, err
}

type DailyGrantStatus struct {
	Status         string                  `json:"status"`
	CanClaim       bool                    `json:"canClaim"`
	ServerDate     string                  `json:"serverDate"`
	NextResetAt    time.Time               `json:"nextResetAt"`
	StreakDay      int                     `json:"streakDay"`
	StreakCycleDay int                     `json:"streakCycleDay"`
	Rewards        map[string]int64        `json:"rewards"`
	LastClaim      *models.DailyGrantClaim `json:"lastClaim,omitempty"`
}

type DailyGrantService struct {
	db              *gorm.DB
	resourceService *ResourceService
	now             func() time.Time
}

func NewDailyGrantService(db *gorm.DB) *DailyGrantService {
	return &DailyGrantService{
		db:              db,
		resourceService: NewResourceService(db),
		now:             func() time.Time { return time.Now().UTC() },
	}
}

func (s *DailyGrantService) Status(ctx context.Context, profileID uint) (*DailyGrantStatus, error) {
	if err := s.resourceService.EnsureInitialAllocation(ctx, profileID); err != nil {
		return nil, err
	}
	claimRepo := repositories.NewDailyGrantClaimRepository(s.db)
	today := s.serverDate()
	nextReset := s.nextResetAt()
	if claim, err := claimRepo.FindByDate(ctx, profileID, today); err == nil {
		return &DailyGrantStatus{
			Status:         "already_claimed",
			CanClaim:       false,
			ServerDate:     today,
			NextResetAt:    nextReset,
			StreakDay:      claim.StreakDay,
			StreakCycleDay: claim.StreakCycleDay,
			Rewards:        claim.RewardResources,
			LastClaim:      claim,
		}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	streakDay, cycleDay, err := s.nextStreak(ctx, profileID)
	if err != nil {
		return nil, err
	}
	return &DailyGrantStatus{
		Status:         "available",
		CanClaim:       true,
		ServerDate:     today,
		NextResetAt:    nextReset,
		StreakDay:      streakDay,
		StreakCycleDay: cycleDay,
		Rewards:        s.rewardsForCycleDay(ctx, cycleDay),
	}, nil
}

func (s *DailyGrantService) Claim(ctx context.Context, profileID uint) (*DailyGrantStatus, error) {
	status, err := s.Status(ctx, profileID)
	if err != nil {
		return nil, err
	}
	if !status.CanClaim {
		return status, nil
	}

	claim := &models.DailyGrantClaim{}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		claimRepo := repositories.NewDailyGrantClaimRepository(tx)
		if existing, err := claimRepo.FindByDate(ctx, profileID, status.ServerDate); err == nil {
			claim = existing
			status.Status = "already_claimed"
			status.CanClaim = false
			status.LastClaim = existing
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		resourceRepo := repositories.NewPlayerResourceRepository(tx)
		transactionRepo := repositories.NewResourceTransactionRepository(tx)
		now := s.now()
		for code, amount := range status.Rewards {
			if amount == 0 {
				continue
			}
			def, ok := resourceDefinitionByCode(code)
			if !ok {
				continue
			}
			resource, err := resourceRepo.Add(ctx, profileID, code, amount, DefaultStorageCapacity, def.IsStorageLimited)
			if err != nil {
				return err
			}
			if err := transactionRepo.Create(ctx, &models.ResourceTransaction{
				ProfileGamerID:  profileID,
				ResourceCode:    code,
				AmountDelta:     amount,
				BalanceAfter:    resource.Amount,
				TransactionType: "daily_grant",
				Source:          "system",
				CreatedAt:       now,
			}); err != nil {
				return err
			}
		}
		claim = &models.DailyGrantClaim{
			ProfileGamerID:  profileID,
			ClaimedDate:     status.ServerDate,
			StreakDay:       status.StreakDay,
			StreakCycleDay:  status.StreakCycleDay,
			RewardResources: status.Rewards,
			CreatedAt:       now,
		}
		return claimRepo.Create(ctx, claim)
	})
	if err != nil {
		return nil, err
	}
	status.Status = "claimed"
	status.CanClaim = false
	status.LastClaim = claim
	return status, nil
}

func (s *DailyGrantService) History(ctx context.Context, profileID uint, limit int) ([]models.DailyGrantClaim, error) {
	return repositories.NewDailyGrantClaimRepository(s.db).List(ctx, profileID, limit)
}

func (s *DailyGrantService) nextStreak(ctx context.Context, profileID uint) (int, int, error) {
	last, err := repositories.NewDailyGrantClaimRepository(s.db).Last(ctx, profileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 1, 1, nil
		}
		return 0, 0, err
	}
	lastDate, err := time.Parse("2006-01-02", last.ClaimedDate)
	if err != nil {
		return 1, 1, nil
	}
	today, _ := time.Parse("2006-01-02", s.serverDate())
	days := int(today.Sub(lastDate).Hours() / 24)
	if days == 1 {
		streak := last.StreakDay + 1
		return streak, ((streak - 1) % 7) + 1, nil
	}
	return 1, 1, nil
}

func (s *DailyGrantService) rewardsForCycleDay(ctx context.Context, cycleDay int) map[string]int64 {
	rewards := map[string]int64{}
	configs, err := repositories.NewDailyGrantConfigRepository(s.db).List(ctx, true)
	configByCode := map[string]int64{}
	if err == nil {
		for _, config := range configs {
			configByCode[config.ResourceCode] = config.BaseAmount
		}
	}
	for _, def := range officialResourceDefinitions {
		amount, ok := configByCode[def.Code]
		if !ok {
			amount = def.DailyGrantAmount
		}
		rewards[def.Code] = applyStreakMultiplier(amount, cycleDay)
	}
	if cycleDay == 7 {
		rewards["tokens"] += 50
		rewards["inference_credit"] += 25
		rewards["neural_fiber"] += 1
	}
	return rewards
}

func applyStreakMultiplier(amount int64, cycleDay int) int64 {
	if amount == 0 {
		return 0
	}
	multipliers := map[int]float64{
		1: 1.00,
		2: 1.05,
		3: 1.10,
		4: 1.15,
		5: 1.20,
		6: 1.25,
		7: 1.00,
	}
	multiplier := multipliers[cycleDay]
	if multiplier == 0 {
		multiplier = 1
	}
	return int64(math.Round(float64(amount) * multiplier))
}

func (s *DailyGrantService) serverDate() string {
	return s.now().UTC().Format("2006-01-02")
}

func (s *DailyGrantService) nextResetAt() time.Time {
	now := s.now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
}
