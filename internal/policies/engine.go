package policies

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cgwm/battle/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Policy struct {
	Key         string             `json:"key"`
	Name        string             `json:"name"`
	Cost        map[string]float64 `json:"cost"`
	Duration    int                `json:"duration"` // seconds
	Effects     PolicyEffects      `json:"effects"`
	ActiveUntil time.Time          `json:"activeUntil"`
}

type PolicyEffects struct {
	HappinessBonus    int                `json:"happinessBonus"`
	ProductionBonus   map[string]float64 `json:"productionBonus"`
	TaxRevenueBonus   float64            `json:"taxRevenueBonus"`
	ArmyMoraleBonus   int                `json:"armyMoraleBonus"`
	ConstructionBonus float64            `json:"constructionBonus"`
}

var DefinedPolicies = map[string]Policy{
	"city_festival": {Key: "city_festival", Name: "Festival de la ville", Cost: map[string]float64{"gold": 1000}, Duration: 4 * 3600, Effects: PolicyEffects{HappinessBonus: 20}},
	"austerity":     {Key: "austerity", Name: "Austérité", Cost: map[string]float64{}, Duration: 12 * 3600, Effects: PolicyEffects{TaxRevenueBonus: -0.2, HappinessBonus: -15}},
}

type Engine struct {
	db *gorm.DB
}

func NewEngine() *Engine { return &Engine{} }

// NewEngineWithDB allows wiring db for real cost deduction + persistence (used by service).
func NewEngineWithDB(db *gorm.DB) *Engine { return &Engine{db: db} }

func (e *Engine) GetAvailable(ctx context.Context, playerID uint) ([]Policy, error) {
	return []Policy{DefinedPolicies["city_festival"], DefinedPolicies["austerity"]}, nil
}

func (e *Engine) Activate(ctx context.Context, playerID uint, policyKey string) error {
	p, ok := DefinedPolicies[policyKey]
	if !ok {
		return fmt.Errorf("unknown policy")
	}

	if e.db != nil {
		return e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var save models.PlayerSave
			if err := tx.Where("player_id = ?", playerID).First(&save).Error; err != nil {
				return err
			}
			// Deduct cost (simple gold/credits)
			cost := p.Cost["gold"]
			if cost > 0 && float64(save.Credits) < cost {
				return fmt.Errorf("not enough gold for policy")
			}
			if cost > 0 {
				if err := tx.Model(&models.PlayerSave{}).Where("player_id = ?", playerID).
					Update("credits", int64(float64(save.Credits)-cost)).Error; err != nil {
					return err
				}
			}

			// Persist active policy into ActiveEffectsJSON (tolerant, non-breaking)
			var fx map[string]any
			if len(save.ActiveEffectsJSON) > 0 {
				_ = json.Unmarshal(save.ActiveEffectsJSON, &fx)
			}
			if fx == nil {
				fx = map[string]any{}
			}
			active := map[string]any{
				"key":         policyKey,
				"until":       time.Now().UTC().Add(time.Duration(p.Duration) * time.Second).Format(time.RFC3339),
				"duration":    p.Duration, // store original duration so UI can display it correctly
				"effects":     p.Effects,
			}
			fx["active_policy"] = active
			fxJSON, _ := json.Marshal(fx)

			return tx.Model(&models.PlayerSave{}).Where("player_id = ?", playerID).
				Updates(map[string]any{
					"active_effects_json": datatypes.JSON(fxJSON),
					"last_synced_at":      time.Now(),
				}).Error
		})
	}

	// No db: still succeed so UI flows (effects returned to caller)
	_ = p.Effects
	return nil
}

// ExpireActivePolicies removes expired policies from ActiveEffectsJSON.
func (e *Engine) ExpireActivePolicies(ctx context.Context, playerID uint) error {
	if e.db == nil {
		return nil
	}
	return e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var save models.PlayerSave
		if err := tx.Where("player_id = ?", playerID).First(&save).Error; err != nil {
			return err
		}
		if len(save.ActiveEffectsJSON) == 0 {
			return nil
		}
		var fx map[string]any
		if json.Unmarshal(save.ActiveEffectsJSON, &fx) != nil || fx == nil {
			return nil
		}
		if ap, ok := fx["active_policy"].(map[string]any); ok {
			if untilStr, ok := ap["until"].(string); ok {
				if until, err := time.Parse(time.RFC3339, untilStr); err == nil && time.Now().UTC().After(until) {
					delete(fx, "active_policy")
					fxJSON, _ := json.Marshal(fx)
					return tx.Model(&models.PlayerSave{}).Where("player_id = ?", playerID).
						Update("active_effects_json", datatypes.JSON(fxJSON)).Error
				}
			}
		}
		return nil
	})
}
