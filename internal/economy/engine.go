package economy

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"cgwm/battle/internal/models"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// CityEconomy is the authoritative economy snapshot.
type CityEconomy struct {
	GoldBalance    float64   `json:"goldBalance"`
	Credits        float64   `json:"credits"` // canonical main currency name for UI consistency
	HourlyIncome   float64   `json:"hourlyIncome"`
	HourlyExpenses float64   `json:"hourlyExpenses"`
	NetPerHour     float64   `json:"netPerHour"`
	TaxRate        float64   `json:"taxRate"`
	Debt           float64   `json:"debt"`
	DebtDueAt      time.Time `json:"debtDueAt"`
}

type Engine struct {
	db *gorm.DB
	// TODO: dependencies on resources engine, research resolver, army, etc. (wired at service level)
}

func NewEngine(db *gorm.DB) *Engine {
	return &Engine{db: db}
}

// loadBaseFromSave extracts economy-relevant fields from PlayerSave (real data source).
func (e *Engine) loadBaseFromSave(save *models.PlayerSave) CityEconomy {
	econ := CityEconomy{
		GoldBalance: float64(save.Credits),
		Credits:     float64(save.Credits),
		TaxRate:     0.25,
		Debt:        0,
	}

	// Calculate a simple "interest paid so far" concept for UI clarity
	if econ.Debt > 0 {
		// Rough: 15% was added at request time. We can expose original principal if we stored it, for now show current debt.
	}

	// Try to read debt / tax from ActiveEffectsJSON if present (non-breaking)
	if len(save.ActiveEffectsJSON) > 0 {
		var fx map[string]any
		if json.Unmarshal(save.ActiveEffectsJSON, &fx) == nil {
			if t, ok := fx["tax_rate"].(float64); ok {
				econ.TaxRate = math.Max(0, math.Min(1, t))
			}
			if d, ok := fx["debt"].(float64); ok {
				econ.Debt = d
			}
			if dueStr, ok := fx["debt_due_at"].(string); ok {
				if t, err := time.Parse(time.RFC3339, dueStr); err == nil {
					econ.DebtDueAt = t
				}
			}
		}
	}

	// Rough income/expense estimate (will be improved with buildings + army upkeep + research)
	baseIncome := float64(save.Population) * 0.8
	econ.HourlyIncome = baseIncome * (1.0 + (econ.TaxRate * 0.6))
	econ.HourlyExpenses = float64(save.Population)*0.25 + (float64(len(save.BuildingsJSON)) * 2.0)
	econ.NetPerHour = econ.HourlyIncome - econ.HourlyExpenses

	return econ
}

func (e *Engine) GetEconomy(ctx context.Context, playerID uint) (CityEconomy, error) {
	if e.db == nil {
		return CityEconomy{GoldBalance: 45230, Credits: 45230, HourlyIncome: 890.5, HourlyExpenses: 340.2, NetPerHour: 550.3, TaxRate: 0.25}, nil
	}
	var save models.PlayerSave
	if err := e.db.WithContext(ctx).Where("player_id = ?", playerID).First(&save).Error; err != nil {
		return CityEconomy{}, err
	}
	return e.loadBaseFromSave(&save), nil
}

// SetTaxRate implements POST /api/v1/city/economy/tax-rate (0-1 range).
// Persists into ActiveEffectsJSON (tolerant) + triggers downstream happiness recalc in full wiring.
func (e *Engine) SetTaxRate(ctx context.Context, playerID uint, rate float64) error {
	if e.db == nil {
		return nil
	}
	rate = math.Max(0, math.Min(1, rate))
	return e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var save models.PlayerSave
		if err := tx.Where("player_id = ?", playerID).First(&save).Error; err != nil {
			return err
		}
		var fx map[string]any
		if len(save.ActiveEffectsJSON) > 0 {
			_ = json.Unmarshal(save.ActiveEffectsJSON, &fx)
		}
		if fx == nil {
			fx = map[string]any{}
		}
		fx["tax_rate"] = rate
		fxJSON, _ := json.Marshal(fx)
		return tx.Model(&models.PlayerSave{}).Where("player_id = ?", playerID).
			Updates(map[string]any{
				"active_effects_json": datatypes.JSON(fxJSON),
				"last_synced_at":      time.Now(),
			}).Error
	})
}

// RequestLoan implements the loan system from the spec (max ~HourlyIncome*48, 15% interest, due 24h).
func (e *Engine) RequestLoan(ctx context.Context, playerID uint, amount float64) error {
	if e.db == nil {
		return nil
	}
	return e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var save models.PlayerSave
		if err := tx.Where("player_id = ?", playerID).First(&save).Error; err != nil {
			return err
		}
		econ := e.loadBaseFromSave(&save)
		maxLoan := econ.HourlyIncome * 48
		if amount > maxLoan {
			amount = maxLoan
		}
		if amount <= 0 {
			return nil
		}
		now := time.Now().UTC()
		due := now.Add(24 * time.Hour)
		debt := amount * 1.15

		var fx map[string]any
		if len(save.ActiveEffectsJSON) > 0 {
			_ = json.Unmarshal(save.ActiveEffectsJSON, &fx)
		}
		if fx == nil {
			fx = map[string]any{}
		}
		fx["debt"] = debt
		fx["debt_due_at"] = due.Format(time.RFC3339)

		fxJSON, _ := json.Marshal(fx)
		return tx.Model(&models.PlayerSave{}).Where("player_id = ?", playerID).
			Updates(map[string]any{
				"credits":             int64(float64(save.Credits) + amount),
				"active_effects_json": datatypes.JSON(fxJSON),
				"last_synced_at":      now,
			}).Error
	})
}

// RepayLoan forces or voluntary repayment.
func (e *Engine) RepayLoan(ctx context.Context, playerID uint) error {
	if e.db == nil {
		return nil
	}
	return e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var save models.PlayerSave
		if err := tx.Where("player_id = ?", playerID).First(&save).Error; err != nil {
			return err
		}
		var fx map[string]any
		if len(save.ActiveEffectsJSON) > 0 {
			_ = json.Unmarshal(save.ActiveEffectsJSON, &fx)
		}
		if fx == nil {
			fx = map[string]any{}
		}
		delete(fx, "debt")
		delete(fx, "debt_due_at")
		fxJSON, _ := json.Marshal(fx)
		return tx.Model(&models.PlayerSave{}).Where("player_id = ?", playerID).
			Updates(map[string]any{
				"active_effects_json": datatypes.JSON(fxJSON),
				"last_synced_at":      time.Now(),
			}).Error
	})
}
