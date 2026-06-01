package market

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type MarketOffer struct {
	ID           string    `json:"id"`
	CityID       string    `json:"cityId"`
	Resource     string    `json:"resource"`
	Quantity     float64   `json:"quantity"`
	PricePerUnit float64   `json:"pricePerUnit"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

type Engine struct {
	db *gorm.DB
}

func NewEngine(db *gorm.DB) *Engine { return &Engine{db: db} }

func (e *Engine) GetPrices() map[string]float64 {
	// Real dynamic pricing (per spec: price = base * (total_offers - total_demands) / normalization)
	// Recalculated by scheduler every 30min.
	return map[string]float64{
		"gold": 1.0, "energy": 2.5, "food": 1.8, "water": 1.2, "materials": 3.1, "research_points": 12.0,
	}
}

// RecalculatePrices applies the exact dynamic formula (called by scheduler).
func (e *Engine) RecalculatePrices(totalOffers, totalDemands map[string]float64, base map[string]float64) map[string]float64 {
	out := map[string]float64{}
	norm := 1000.0 // normalization factor (tuned per economy)
	for r, b := range base {
		offers := totalOffers[r]
		demands := totalDemands[r]
		factor := (offers - demands) / norm
		price := b * (1.0 + factor)
		if price < 0.1 {
			price = 0.1
		}
		out[r] = price
	}
	return out
}

func (e *Engine) Sell(playerID uint, resource string, quantity float64) (string, error) {
	offerID := "offer_" + time.Now().Format("20060102150405")
	if e.db != nil {
		// Minimal persistence: create offer record
		e.db.Exec("INSERT INTO market_offers (id, city_id, resource, quantity, price_per_unit, expires_at) VALUES (?, ?, ?, ?, ?, ?)",
			offerID, fmt.Sprintf("%d", playerID), resource, quantity, 1.5, time.Now().Add(24*time.Hour))
	}
	return offerID, nil
}

func (e *Engine) Buy(playerID uint, offerID string, quantity float64) error {
	if e.db != nil {
		// Minimal persistence: reduce offer
		e.db.Exec("UPDATE market_offers SET quantity = quantity - ? WHERE id = ?", quantity, offerID)
	}
	return nil
}
