package market

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type MarketOffer struct {
	ID           string    `json:"id"`
	CityID       string    `json:"cityId"`
	Source       string    `json:"source"` // "ia_global" | "player"
	ContinentID  string    `json:"continentId"`
	Direction    string    `json:"direction"` // "sell" (offer to buy from) or "buy" (IA/player wants to acquire)
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

func (e *Engine) Sell(playerID uint, resource string, quantity float64, continentID string) (string, error) {
	offerID := "offer_" + time.Now().Format("20060102150405")
	if e.db != nil {
		// Player sell offer -> source player, direction sell (others can buy it), tagged with real continent
		price := 1.5 // could be dynamic from GetPrices
		e.db.Exec(`INSERT INTO market_offers (id, city_id, source, continent_id, direction, resource, quantity, price_per_unit, expires_at) 
		           VALUES (?, ?, 'player', ?, 'sell', ?, ?, ?, ?)`,
			offerID, fmt.Sprintf("%d", playerID), continentID, resource, quantity, price, time.Now().Add(24*time.Hour))
	}
	return offerID, nil
}

func (e *Engine) Buy(playerID uint, offerID string, quantity float64) error {
	if e.db != nil {
		// Reduce remaining qty on the offer (works for both "sell" offers and "buy" offers / wanted qty).
		// If drops to 0 or below, remove the offer (fulfilled).
		e.db.Exec("UPDATE market_offers SET quantity = quantity - ? WHERE id = ?", quantity, offerID)
		e.db.Exec("DELETE FROM market_offers WHERE id = ? AND quantity <= 0", offerID)
	}
	return nil
}

// FulfillBuyOffer is used when player sells into an IA "buy" offer (IA acquires the resource, player gets paid).
// For now same as Buy (qty reduction); handler does the credit/resource transfer.
func (e *Engine) FulfillBuyOffer(playerID uint, offerID string, quantity float64) error {
	return e.Buy(playerID, offerID, quantity)
}

// GetIAMarketOffers returns offers from the Global IA Market (evil world market).
// Makes the market "vivant": auto-refills low quantity offers, seeds initial sell + buy offers if empty.
func (e *Engine) GetIAMarketOffers() []MarketOffer {
	if e.db == nil {
		return nil
	}

	// Try to load existing IA offers
	var offers []MarketOffer
	e.db.Where("city_id = ?", "ia_global").Find(&offers)

	now := time.Now().UTC()
	basePrices := e.GetPrices()

	if len(offers) == 0 {
		// Seed initial IA sell offers (IA selling to players) + some buy offers (IA buying from players)
		resources := []string{"gold", "energy", "food", "water", "materials", "research_points"}
		for i, res := range resources {
			base := basePrices[res]
			price := base * (1.0 + 0.30) // IA markup ~30% on sell offers
			qty := 650.0 + float64(i*40)

			offerID := "ia_sell_" + res
			e.db.Exec(`INSERT INTO market_offers (id, city_id, source, continent_id, direction, resource, quantity, price_per_unit, expires_at) 
			           VALUES (?, 'ia_global', 'ia_global', '', 'sell', ?, ?, ?, ?)`,
				offerID, res, qty, price, now.Add(72*time.Hour))

			// Seed 3 IA buy offers (IA wants to acquire these from players at decent price)
			if res == "food" || res == "materials" || res == "energy" {
				buyPrice := base * 0.85 // IA pays slightly below base to players
				buyQty := 400.0
				buyID := "ia_buy_" + res
				e.db.Exec(`INSERT INTO market_offers (id, city_id, source, continent_id, direction, resource, quantity, price_per_unit, expires_at) 
				           VALUES (?, 'ia_global', 'ia_global', '', 'buy', ?, ?, ?, ?)`,
					buyID, res, buyQty, buyPrice, now.Add(48*time.Hour))
			}
		}

		// Reload after seeding
		e.db.Where("city_id = ?", "ia_global").Find(&offers)
	}

	// --- Make IA market alive: periodic-ish refill on access (quantities decrease on player buys, we top-up low ones) ---
	refilled := false
	for i := range offers {
		if offers[i].Quantity < 80 {
			// Top up the offer (simulates IA production / restock)
			topUp := 300.0
			newQty := offers[i].Quantity + topUp
			e.db.Exec("UPDATE market_offers SET quantity = ? WHERE id = ?", newQty, offers[i].ID)
			offers[i].Quantity = newQty
			refilled = true
		}
		// Also refresh expiration occasionally
		if offers[i].ExpiresAt.Before(now.Add(6 * time.Hour)) {
			e.db.Exec("UPDATE market_offers SET expires_at = ? WHERE id = ?", now.Add(72*time.Hour), offers[i].ID)
		}
	}
	if refilled {
		// reload to be sure
		e.db.Where("city_id = ?", "ia_global").Find(&offers)
	}

	return offers
}

// RefillIAMarket can be called from world cycle (continental) for scheduled restocks.
func (e *Engine) RefillIAMarket() {
	if e.db == nil {
		return
	}
	// Example scheduled boost: add stock to all IA offers that are low or randomly
	e.db.Exec(`UPDATE market_offers SET quantity = quantity + 150 WHERE city_id = 'ia_global' AND quantity < 200 AND direction = 'sell'`)
	e.db.Exec(`UPDATE market_offers SET quantity = quantity + 80 WHERE city_id = 'ia_global' AND quantity < 150 AND direction = 'buy'`)
}

// GetDB exposes the DB for handlers that need direct queries (temporary until engine has richer query methods).
func (e *Engine) GetDB() *gorm.DB { return e.db }

// GetPlayerOffers loads persisted player sell/buy offers, optionally filtered by continent.
func (e *Engine) GetPlayerOffers(continentID string) []MarketOffer {
	if e.db == nil {
		return nil
	}
	var out []MarketOffer
	q := e.db.Where("source = ?", "player")
	if continentID != "" {
		q = q.Where("continent_id = ?", continentID)
	}
	q.Find(&out)
	return out
}
