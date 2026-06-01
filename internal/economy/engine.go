package economy

import (
	"context"
	"time"
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
	// TODO: dependencies on resources engine, research resolver, army, etc.
}

func NewEngine() *Engine {
	return &Engine{}
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
	// TODO: validate 0-1, update PlayerSave, trigger happiness recalc
	return nil
}

// RequestLoan implements the loan system from the spec.
func (e *Engine) RequestLoan(ctx context.Context, playerID uint, amount float64) error {
	// TODO: check max = HourlyIncome * 48, create debt record with 15% interest, 24h due
	return nil
}

// RepayLoan forces or voluntary repayment (15% interest already applied at request time).
func (e *Engine) RepayLoan(ctx context.Context, playerID uint) error {
	// TODO: deduct from gold, clear Debt/DebtDueAt on PlayerSave
	return nil
}
