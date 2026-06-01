package economy

import (
	"context"
	"time"

	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

// CityEconomy is the authoritative economy snapshot.
type CityEconomy struct {
	GoldBalance    float64   `json:"goldBalance"`
	HourlyIncome   float64   `json:"hourlyIncome"`
	HourlyExpenses float64   `json:"hourlyExpenses"`
	NetPerHour     float64   `json:"netPerHour"`
	TaxRate        float64   `json:"taxRate"`
	Debt           float64   `json:"debt"`
	DebtDueAt      time.Time `json:"debtDueAt"`
}

type Engine struct {
	db *gorm.DB
	// TODO: dependencies on resources engine, research resolver, army, etc.
}

func NewEngine(db *gorm.DB) *Engine {
	return &Engine{db: db}
}

func (e *Engine) GetEconomy(ctx context.Context, playerID uint) (CityEconomy, error) {
	// TODO: real calculation from PlayerSave + research + buildings + army upkeep + active policies
	return CityEconomy{
		GoldBalance:    45230,
		HourlyIncome:   890.5,
		HourlyExpenses: 340.2,
		NetPerHour:     550.3,
		TaxRate:        0.45,
		Debt:           0,
	}, nil
}

// SetTaxRate implements POST /api/v1/city/economy/tax-rate
func (e *Engine) SetTaxRate(ctx context.Context, playerID uint, rate float64) error {
	if e.db != nil {
		e.db.Model(&models.PlayerSave{}).Where("player_id = ?", playerID).Update("tax_rate", rate)
	}
	// TODO(full): validate 0-1, trigger happiness recalc via population engine
	return nil
}

// RequestLoan implements the loan system from the spec.
func (e *Engine) RequestLoan(ctx context.Context, playerID uint, amount float64) error {
	if e.db != nil {
		now := time.Now().UTC()
		due := now.Add(24 * time.Hour)
		e.db.Model(&models.PlayerSave{}).Where("player_id = ?", playerID).Updates(map[string]any{
			"debt":       amount * 1.15, // 15% interest
			"debt_due_at": due,
		})
	}
	// TODO(full): max check = HourlyIncome * 48, proper debt record
	return nil
}

// RepayLoan forces or voluntary repayment (15% interest already applied at request time).
func (e *Engine) RepayLoan(ctx context.Context, playerID uint) error {
	if e.db != nil {
		e.db.Model(&models.PlayerSave{}).Where("player_id = ?", playerID).Updates(map[string]any{
			"debt":       0,
			"debt_due_at": nil,
		})
	}
	return nil
}
