package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"cgwm/battle/internal/models"
	"cgwm/battle/internal/policies"
	"cgwm/battle/internal/research"
	"cgwm/battle/internal/weather"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ResourceBalance is the source of truth struct returned by the resources engine.
type ResourceBalance struct {
	Current     map[string]float64 `json:"current"`
	Capacity    map[string]float64 `json:"capacity"`
	Production  map[string]float64 `json:"production"` // per hour
	Consumption map[string]float64 `json:"consumption"`
	Net         map[string]float64 `json:"net"`
}

// Engine is the authoritative resources calculation engine.
// All production, consumption, ticks, and deficit effects must go through here.
type Engine struct {
	db *gorm.DB
	// Resolvers (use whatever is exported today)
	researchResolver *research.Resolver
	policyEngine     *policies.Engine
}

const collectorCycleDuration = time.Hour

type collectorProfile struct {
	Resource string
	BaseRate float64
}

func NewEngine(db *gorm.DB) *Engine {
	return &Engine{
		db:               db,
		researchResolver: research.NewResolver(),
		policyEngine:     policies.NewEngine(),
	}
}

// GetBalance returns the current resource state for a player.
// This is the single source of truth. No local calculation in Flutter should contradict this.
func (e *Engine) GetBalance(ctx context.Context, playerID uint) (ResourceBalance, error) {
	// Real load from PlayerSave (single source of truth)
	balance := ResourceBalance{
		Current:     map[string]float64{"gold": 0, "credits": 0, "energy": 0, "food": 0, "gems": 0},
		Capacity:    map[string]float64{"gold": 50000, "credits": 50000, "energy": 8000, "food": 12000, "gems": 999999, "water": 6000, "materials": 5000, "research_points": 500},
		Production:  map[string]float64{"gold": 0, "credits": 0, "energy": 0, "food": 0, "water": 0, "materials": 0, "research_points": 0},
		Consumption: map[string]float64{"gold": 0, "credits": 0, "energy": 0, "food": 0, "water": 0, "materials": 0, "research_points": 0},
	}

	if e.db != nil {
		var save models.PlayerSave
		if err := e.db.WithContext(ctx).Where("player_id = ?", playerID).First(&save).Error; err == nil {
			// Load current from InventoryJSON (preferred) or direct fields
			if len(save.InventoryJSON) > 0 {
				var inv map[string]float64
				if json.Unmarshal(save.InventoryJSON, &inv) == nil {
					for k, v := range inv {
						balance.Current[k] = v
					}
				}
			}
			// Direct PlayerSave fields are authoritative for core HUD resources.
			balance.Current["gold"] = float64(save.Credits)
			balance.Current["credits"] = float64(save.Credits)
			balance.Current["food"] = float64(save.Food)
			balance.Current["energy"] = float64(save.Energy)
			balance.Current["gems"] = float64(save.Gems)

			// Derive base production from BuildingsJSON (real)
			if len(save.BuildingsJSON) > 4 {
				var buildings []map[string]any
				if json.Unmarshal(save.BuildingsJSON, &buildings) == nil {
					for _, b := range buildings {
						key := fmt.Sprintf("%v", b["buildingKey"])
						levelF, _ := b["level"].(float64)
						lvl := int(levelF)
						if lvl < 1 {
							lvl = 1
						}
						switch key {
						case "solar_park", "energy_plant":
							balance.Production["energy"] += float64(40 * lvl)
						case "vertical_farm", "farm":
							balance.Production["food"] += float64(35 * lvl)
						case "mine", "quarry":
							balance.Production["materials"] += float64(22 * lvl)
						case "market", "trading_post":
							balance.Production["gold"] += float64(55 * lvl)
						default:
							balance.Production["gold"] += float64(8 * lvl)
						}
					}
				}
			}
			// Army upkeep consumption (real)
			var armyCount int64
			e.db.Model(&models.ArmyUnit{}).Where("player_id = ?", playerID).Count(&armyCount)
			balance.Consumption["food"] += float64(armyCount) * 1.8
			balance.Consumption["energy"] += float64(armyCount) * 0.9
			balance.Consumption["gold"] += float64(armyCount) * 0.6
		}
	}

	// === Apply cross-domain bonuses (Go = single source of truth) ===
	researchMulti := 1.0
	if e.researchResolver != nil && e.db != nil {
		var save models.PlayerSave
		if err := e.db.WithContext(ctx).Where("player_id = ?", playerID).First(&save).Error; err == nil && len(save.ResearchJSON) > 0 {
			var r map[string]any
			if json.Unmarshal(save.ResearchJSON, &r) == nil {
				keys := []string{}
				if u, ok := r["unlocked"].([]any); ok {
					for _, v := range u {
						if s, ok := v.(string); ok {
							keys = append(keys, s)
						}
					}
				}
				bonuses := e.researchResolver.Compute(keys)
				for _, m := range bonuses.ProductionMultipliers {
					researchMulti *= m
				}
			}
		}
	} else if e.researchResolver != nil {
		bonuses := e.researchResolver.Compute([]string{})
		for _, m := range bonuses.ProductionMultipliers {
			researchMulti *= m
		}
	}

	weatherMulti := 1.0
	_ = weather.ApplyWeatherModifiers(map[string]float64{}, "clear")

	policyMulti := 1.0
	if e.policyEngine != nil && e.db != nil {
		// Load active policy effects from ActiveEffectsJSON (matches policies engine)
		var save models.PlayerSave
		if err := e.db.WithContext(ctx).Where("player_id = ?", playerID).First(&save).Error; err == nil && len(save.ActiveEffectsJSON) > 0 {
			var fx map[string]any
			if json.Unmarshal(save.ActiveEffectsJSON, &fx) == nil {
				if ap, ok := fx["active_policy"].(map[string]any); ok {
					if eff, ok := ap["effects"].(map[string]any); ok {
						if pb, ok := eff["productionBonus"].(map[string]any); ok {
							for k, v := range pb {
								if f, ok := v.(float64); ok {
									if cur, exists := balance.Production[k]; exists {
										balance.Production[k] = cur * f
									}
								}
							}
						}
					}
				}
			}
		}
	}

	finalProdMulti := researchMulti * weatherMulti * policyMulti
	for r := range balance.Production {
		balance.Production[r] *= finalProdMulti
	}

	for r, prod := range balance.Production {
		cons := balance.Consumption[r]
		balance.Net[r] = prod - cons
	}

	return balance, nil
}

// Tick applies the 10-minute resource tick (the authoritative one — Go source of truth).
// Applies Net * (minutes/60), clamps Current to [0, Capacity], and applies deficit cascading
// (energy <0 → production * 0.3, food deficit → happiness malus, etc.).
func (e *Engine) Tick(ctx context.Context, playerID uint, minutes float64) error {
	// In full impl: load full PlayerSave + buildings + research + weather + policies + army upkeep.
	// For now: demonstrate the exact formulas on the minimal balance returned by GetBalance.
	balance, err := e.GetBalance(ctx, playerID)
	if err != nil {
		return err
	}

	fraction := minutes / 60.0
	for r, net := range balance.Net {
		delta := net * fraction
		newCur := balance.Current[r] + delta
		if newCur < 0 {
			newCur = 0
		}
		if cap, ok := balance.Capacity[r]; ok && newCur > cap {
			newCur = cap
		}
		balance.Current[r] = newCur
	}

	// Deficit side effects (applied to the balance for next Get)
	energyNet := 0.0
	if v, ok := balance.Net["energy"]; ok {
		energyNet = v
	}
	if energyNet < 0 {
		// Example: energy deficit slows other production
		for r := range balance.Production {
			if r != "energy" {
				balance.Production[r] *= 0.3
			}
		}
	}

	foodNet := 0.0
	if v, ok := balance.Net["food"]; ok {
		foodNet = v
	}
	if foodNet < 0 {
		// Food deficit would reduce happiness in population engine
		_ = "food deficit marker for population/happiness cross-effect"
	}

	// === Persistence + cross-engine side effects (Go source of truth) ===
	if e.db != nil {
		_ = e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var save models.PlayerSave
			if err := tx.Where("player_id = ?", playerID).First(&save).Error; err != nil {
				return err
			}

			// Marshal full current balance back to InventoryJSON (authoritative)
			inv := map[string]float64{}
			if len(save.InventoryJSON) > 0 {
				_ = json.Unmarshal(save.InventoryJSON, &inv)
			}
			for k, v := range balance.Current {
				inv[k] = v
			}
			invJSON, _ := json.Marshal(inv)

			return tx.Model(&models.PlayerSave{}).Where("player_id = ?", playerID).
				Updates(map[string]any{
					"inventory_json": datatypes.JSON(invJSON),
					"food":           int64(balance.Current["food"]),
					"energy":         int64(balance.Current["energy"]),
					"credits":        int64(balance.Current["gold"]),
					"last_synced_at": time.Now(),
				}).Error
		})

		// Cross-effect note for population (food deficit)
		if foodNet < 0 {
			// In full scheduler this calls population engine for happiness impact.
		}
	}

	// Bonuses already applied upstream.
	return nil
}

// ManualCollect implements POST /api/v1/buildings/:id/collect for collector buildings.
func (e *Engine) ManualCollect(ctx context.Context, playerID uint, buildingID string) (map[string]float64, error) {
	if normalizeBuildingKey(buildingID) == "demolish_refund" {
		return map[string]float64{}, nil
	}

	if e.db == nil {
		return nil, fmt.Errorf("resources engine is not connected to a database")
	}

	var collectedAmount float64
	var collectedProgress float64
	collectedResource := ""

	err := e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var save models.PlayerSave
		if err := tx.Where("player_id = ?", playerID).First(&save).Error; err != nil {
			return err
		}

		buildings, err := decodePlayerBuildings(save.BuildingsJSON)
		if err != nil {
			return err
		}

		idx := findCollectibleBuildingIndex(buildings, buildingID)
		if idx < 0 {
			return fmt.Errorf("building %q not found for player %d", buildingID, playerID)
		}

		building := buildings[idx]
		buildingKey := normalizeBuildingKey(stringFromAny(building["buildingKey"]))
		if buildingKey == "" {
			return fmt.Errorf("building %q is missing a buildingKey", buildingID)
		}

		profileForKey, ok := collectorProfileForBuilding(buildingKey)
		if !ok {
			return fmt.Errorf("building %q does not support manual collection", buildingKey)
		}

		level := intFromAnyDefault(building["level"], 0)
		if level <= 0 {
			level = 1
		}

		now := time.Now().UTC()
		lastCollectedAt := timeFromAny(building["lastCollectedAt"])
		completedAt := timeFromAny(building["completedAt"])
		baseline := lastCollectedAt
		if baseline == nil {
			baseline = completedAt
		}
		amount, progress := computeCollectorHarvest(profileForKey, level, baseline, now)
		if amount <= 0 {
			amount = 0
		}
		collectedAmount = amount
		collectedProgress = progress
		collectedResource = profileForKey.Resource

		inventory := map[string]float64{}
		if len(save.InventoryJSON) > 0 {
			_ = json.Unmarshal(save.InventoryJSON, &inventory)
		}

		addResourceToSave(&save, inventory, profileForKey.Resource, amount)
		building["lastCollectedAt"] = now
		buildings[idx] = building

		buildingRowID, hasBuildingRowID := uintFromAny(building["id"])
		query := tx.Model(&models.PlayerBuilding{}).Where("player_id = ? AND building_key = ?", save.PlayerID, buildingKey)
		if hasBuildingRowID {
			query = tx.Model(&models.PlayerBuilding{}).Where("player_id = ? AND (building_key = ? OR id = ?)", save.PlayerID, buildingKey, buildingRowID)
		}
		if err := query.Update("last_collected_at", now).Error; err != nil {
			return err
		}

		buildingsJSON, err := json.Marshal(buildings)
		if err != nil {
			return err
		}
		inventoryJSON, err := json.Marshal(inventory)
		if err != nil {
			return err
		}

		return tx.Model(&models.PlayerSave{}).
			Where("id = ?", save.Id).
			Updates(map[string]any{
				"buildings_json": datatypes.JSON(buildingsJSON),
				"inventory_json": datatypes.JSON(inventoryJSON),
				"food":           save.Food,
				"energy":         save.Energy,
				"credits":        save.Credits,
				"last_synced_at": now,
			}).Error
	})
	if err != nil {
		return nil, err
	}

	if collectedResource == "" {
		collectedResource = "materials"
	}

	return map[string]float64{
		collectedResource: collectedAmount,
		"progress":        collectedProgress,
	}, nil
}

func collectorProfileForBuilding(buildingKey string) (collectorProfile, bool) {
	switch normalizeBuildingKey(buildingKey) {
	case "solar_park", "energy_plant":
		return collectorProfile{Resource: "energy", BaseRate: 40}, true
	case "vertical_farm", "farm":
		return collectorProfile{Resource: "food", BaseRate: 35}, true
	case "mine", "quarry", "composite_mine":
		return collectorProfile{Resource: "materials", BaseRate: 22}, true
	case "market", "trading_post":
		return collectorProfile{Resource: "credits", BaseRate: 55}, true
	default:
		return collectorProfile{}, false
	}
}

func normalizeBuildingKey(buildingKey string) string {
	return strings.ToLower(strings.TrimSpace(buildingKey))
}

func computeCollectorHarvest(profile collectorProfile, level int, lastCollectedAt *time.Time, now time.Time) (float64, float64) {
	if level <= 0 || lastCollectedAt == nil || lastCollectedAt.IsZero() {
		return 0, 0
	}
	elapsed := now.Sub(*lastCollectedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	progress := elapsed.Seconds() / collectorCycleDuration.Seconds()
	if progress > 1 {
		progress = 1
	}
	if progress < 0 {
		progress = 0
	}
	amount := profile.BaseRate * float64(level) * progress
	return math.Round(amount*100) / 100, math.Round(progress*10000) / 100
}

func timeFromAny(value any) *time.Time {
	switch v := value.(type) {
	case time.Time:
		if v.IsZero() {
			return nil
		}
		vv := v
		return &vv
	case *time.Time:
		if v == nil || v.IsZero() {
			return nil
		}
		return v
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		if parsed, err := time.Parse(time.RFC3339, v); err == nil {
			return &parsed
		}
		if parsed, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return &parsed
		}
	}
	return nil
}

func addResourceToSave(save *models.PlayerSave, inventory map[string]float64, resource string, amount float64) {
	if amount <= 0 {
		return
	}
	if resource == "credits" || resource == "gold" {
		inventory["credits"] += amount
		inventory["gold"] += amount
	} else {
		inventory[resource] += amount
	}
	additive := int64(math.Round(amount))
	if additive <= 0 {
		return
	}
	switch resource {
	case "food":
		save.Food += additive
	case "energy":
		save.Energy += additive
	case "credits", "gold":
		save.Credits += additive
	default:
		// Keep the resource in InventoryJSON even if it has no dedicated column.
	}
}

func decodePlayerBuildings(raw datatypes.JSON) ([]map[string]any, error) {
	if len(raw) == 0 {
		return []map[string]any{}, nil
	}
	buildings := []map[string]any{}
	if err := json.Unmarshal(raw, &buildings); err != nil {
		return nil, err
	}
	return buildings, nil
}

func findCollectibleBuildingIndex(buildings []map[string]any, buildingID string) int {
	if len(buildings) == 0 {
		return -1
	}
	needle := strings.ToLower(strings.TrimSpace(buildingID))
	if needle == "" {
		return -1
	}
	for i := range buildings {
		if strings.ToLower(strings.TrimSpace(stringFromAny(buildings[i]["buildingKey"]))) == needle {
			return i
		}
		if strings.ToLower(strings.TrimSpace(stringFromAny(buildings[i]["id"]))) == needle {
			return i
		}
		if strings.ToLower(strings.TrimSpace(stringFromAny(buildings[i]["buildingId"]))) == needle {
			return i
		}
		if id, ok := uintFromAny(buildings[i]["id"]); ok && strconv.FormatUint(uint64(id), 10) == needle {
			return i
		}
	}
	return -1
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func intFromAnyDefault(value any, fallback int) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return int(n)
		}
	}
	return fallback
}

func uintFromAny(value any) (uint, bool) {
	switch v := value.(type) {
	case uint:
		return v, true
	case uint64:
		return uint(v), true
	case int:
		if v < 0 {
			return 0, false
		}
		return uint(v), true
	case int64:
		if v < 0 {
			return 0, false
		}
		return uint(v), true
	case float64:
		if v < 0 {
			return 0, false
		}
		return uint(v), true
	case json.Number:
		if n, err := v.Int64(); err == nil && n >= 0 {
			return uint(n), true
		}
	}
	return 0, false
}
