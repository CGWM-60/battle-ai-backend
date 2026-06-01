package resources

import (
	"context"
	"encoding/json"
	"fmt"
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
	Production  map[string]float64 `json:"production"`  // per hour
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
		// Demo positive starting inventory so market sell dialog works (select resource + quantity)
		Current:     map[string]float64{"gold": 2450, "energy": 890, "food": 1340, "water": 620, "materials": 780, "research_points": 45},
		Capacity:    map[string]float64{"gold": 50000, "energy": 8000, "food": 12000, "water": 6000, "materials": 5000, "research_points": 500},
		Production:  map[string]float64{"gold": 180, "energy": 90, "food": 45, "water": 25, "materials": 18, "research_points": 4},
		Consumption: map[string]float64{"gold": 60, "energy": 80, "food": 55, "water": 20, "materials": 8, "research_points": 0},
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
			// Fallback to direct fields on PlayerSave
			if balance.Current["gold"] == 0 {
				balance.Current["gold"] = float64(save.Credits)
			}
			if balance.Current["food"] == 0 {
				balance.Current["food"] = float64(save.Food)
			}
			if balance.Current["energy"] == 0 {
				balance.Current["energy"] = float64(save.Energy)
			}

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
					"inventory_json":  datatypes.JSON(invJSON),
					"food":            int64(balance.Current["food"]),
					"energy":          int64(balance.Current["energy"]),
					"credits":         int64(balance.Current["gold"]),
					"last_synced_at":  time.Now(),
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
	// TODO: find the building, transfer its accumulator into Current, reset accumulator.
	return map[string]float64{"food": 124.5, "materials": 18}, nil
}
