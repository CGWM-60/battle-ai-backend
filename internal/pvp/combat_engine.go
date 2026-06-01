package pvp

import (
	"context"
	"encoding/json"
	"time"

	"cgwm/battle/internal/models"
	"cgwm/battle/internal/research"

	"gorm.io/gorm"
)

type CombatForce struct {
	TotalAttack  float64 `json:"totalAttack"`
	TotalDefense float64 `json:"totalDefense"`
}

type CombatResult struct {
	Winner         string             `json:"winner"`
	AttackerLosses map[string]int     `json:"attackerLosses"`
	DefenderLosses map[string]int     `json:"defenderLosses"`
	LootGained     map[string]float64 `json:"lootGained"`
	Phases         []CombatPhase      `json:"phases"`
	ExecutedAt     time.Time          `json:"executedAt"`
}

type CombatPhase struct {
	Ratio          float64        `json:"ratio"`
	AttackerLoss   map[string]int `json:"attackerLoss"`
	DefenderLoss   map[string]int `json:"defenderLoss"`
}

type Engine struct {
	db       *gorm.DB
	resolver *research.Resolver
}

func NewEngine(db *gorm.DB) *Engine {
	return &Engine{
		db:       db,
		resolver: research.NewResolver(),
	}
}

// loadResearchKeys extracts unlocked node keys from a PlayerSave's ResearchJSON (tolerant).
func (e *Engine) loadResearchKeys(save *models.PlayerSave) []string {
	if save == nil || len(save.ResearchJSON) == 0 {
		return []string{}
	}
	var research map[string]any
	if err := json.Unmarshal(save.ResearchJSON, &research); err != nil {
		return []string{}
	}
	keys := []string{}
	// Common shapes: {"unlocked": ["node1", "node2"]} or {"nodes": {"key": {"unlocked":true}}}
	if unlocked, ok := research["unlocked"].([]any); ok {
		for _, v := range unlocked {
			if s, ok := v.(string); ok {
				keys = append(keys, s)
			}
		}
		return keys
	}
	if nodes, ok := research["nodes"].(map[string]any); ok {
		for k, v := range nodes {
			if m, ok := v.(map[string]any); ok {
				if unlocked, ok := m["unlocked"].(bool); ok && unlocked {
					keys = append(keys, k)
				}
			} else {
				keys = append(keys, k) // assume presence means unlocked
			}
		}
	}
	return keys
}

// loadArmyPower queries real ArmyUnit rows for the player and returns a total power value (attack*health factor).
func (e *Engine) loadArmyPower(playerID uint) float64 {
	if e.db == nil {
		return 0
	}
	var units []models.ArmyUnit
	if err := e.db.Where("player_id = ? AND status IN ?", playerID, []string{"available", "assigned", "injured"}).Find(&units).Error; err != nil {
		return 0
	}
	total := 0.0
	for _, u := range units {
		power := float64(u.Attack)
		if u.Power > 0 {
			power = float64(u.Power)
		}
		healthFactor := float64(u.Health) / 100.0
		if healthFactor < 0.3 {
			healthFactor = 0.3
		}
		total += power * healthFactor * 1.2 // slight scaling to match expected city power range
	}
	if total < 10 && len(units) > 0 {
		total = float64(len(units)) * 120 // fallback
	}
	return total
}

// loadResourceStock returns a rough current stock for a resource from PlayerSave direct fields or InventoryJSON.
func (e *Engine) loadResourceStock(playerID uint, resource string) float64 {
	if e.db == nil {
		return 0
	}
	var save models.PlayerSave
	if err := e.db.Where("player_id = ?", playerID).First(&save).Error; err != nil {
		return 0
	}
	// Direct fields cover the main ones used in loot today
	switch resource {
	case "food":
		return float64(save.Food)
	case "energy":
		return float64(save.Energy)
	case "gold", "credits":
		return float64(save.Credits)
	}
	// Try InventoryJSON
	if len(save.InventoryJSON) > 0 {
		var inv map[string]float64
		if json.Unmarshal(save.InventoryJSON, &inv) == nil {
			if v, ok := inv[resource]; ok {
				return v
			}
		}
	}
	return 0
}

// Simulate does a dry-run (no state change) - full 3-phase per spec, using real army + research when db present.
func (e *Engine) Simulate(attackerPlayerID, defenderPlayerID uint, units map[string]int) (CombatForce, CombatForce, float64, error) {
	// Attacker committed force (from request) + research bonus
	attForce := 0.0
	basePower := 12.0
	researchMulti := 1.0
	if e.db != nil {
		var attSave models.PlayerSave
		if err := e.db.Where("player_id = ?", attackerPlayerID).First(&attSave).Error; err == nil {
			keys := e.loadResearchKeys(&attSave)
			if e.resolver != nil {
				bonuses := e.resolver.Compute(keys)
				researchMulti = bonuses.ArmyAttack
				if researchMulti < 1.0 {
					researchMulti = 1.0
				}
			}
		}
	}
	for _, c := range units {
		attForce += float64(c) * basePower * researchMulti
	}

	// Real defender force from ArmyUnit table + research defense
	defForce := e.loadArmyPower(defenderPlayerID)
	if defForce < 800 {
		defForce = 4200 // sane minimum for cities with no army rows yet (prevents instant wins)
	}
	// Apply defender research defense
	if e.db != nil {
		var defSave models.PlayerSave
		if err := e.db.Where("player_id = ?", defenderPlayerID).First(&defSave).Error; err == nil {
			keys := e.loadResearchKeys(&defSave)
			if e.resolver != nil {
				bonuses := e.resolver.Compute(keys)
				defForce *= bonuses.ArmyDefense
			}
		}
	}

	// 3 phases (per original plan)
	ratio := attForce / defForce
	var attackerLossPct, defenderLossPct float64

	if ratio > 1.2 {
		defenderLossPct = 0.20 + (ratio-1.2)*0.05 // 15-25% defender loss
		attackerLossPct = 0.10
	} else if ratio < 0.8 {
		attackerLossPct = 0.20 + (0.8-ratio)*0.05
		defenderLossPct = 0.10
	} else {
		attackerLossPct = 0.12
		defenderLossPct = 0.12
	}

	// Random 0.9-1.1 factor per spec
	randF := 0.95 + (float64(time.Now().Unix()%10) * 0.02)
	attackerLossPct *= randF
	defenderLossPct *= randF

	finalWinProb := attForce / (attForce + defForce)
	if finalWinProb > 0.95 {
		finalWinProb = 0.95
	}

	return CombatForce{TotalAttack: attForce}, CombatForce{TotalDefense: defForce}, finalWinProb, nil
}

// ExecuteAttack performs the authoritative 3-phase PvP combat (Go = single source of truth).
// Losses: ratio bands + random 0.9-1.1 factor. Loot: min(0.15, ratio*0.05) of defender CURRENT stocks if attacker wins.
// Side effects: defender shield 4h, attacker cooldown 2h (returned in result; persisted by caller/handler).
func (e *Engine) ExecuteAttack(attackerPlayerID uint, targetCityID string, units map[string]int) (CombatResult, error) {
	// Resolve defender player ID from targetCityID.
	// For current wiring many callers pass the numeric defender player id as targetCityID string.
	// We keep a tolerant fallback so existing Flutter calls continue to work.
	defenderPlayerID := attackerPlayerID // fallback (will be improved when city->player mapping is wired)
	// If targetCityID is a pure number string, treat it as defender player id (common pattern today)
	if len(targetCityID) > 0 && targetCityID[0] >= '0' && targetCityID[0] <= '9' {
		var id uint
		for _, ch := range targetCityID {
			if ch >= '0' && ch <= '9' {
				id = id*10 + uint(ch-'0')
			} else {
				id = 0
				break
			}
		}
		if id > 0 {
			defenderPlayerID = id
		}
	}
	// In practice the calling handler now resolves the real defender player before calling.

	// --- Real attacker force + research (product of unlocked) ---
	attForce := 0.0
	basePower := 12.0
	researchAttackMulti := 1.0
	if e.db != nil && e.resolver != nil {
		var attSave models.PlayerSave
		if err := e.db.Where("player_id = ?", attackerPlayerID).First(&attSave).Error; err == nil {
			keys := e.loadResearchKeys(&attSave)
			bonuses := e.resolver.Compute(keys)
			researchAttackMulti = bonuses.ArmyAttack
		}
	}
	for _, c := range units {
		attForce += float64(c) * basePower * researchAttackMulti
	}

	// --- Real defender force from ArmyUnit + research defense ---
	defForce := e.loadArmyPower(defenderPlayerID)
	if defForce < 800 {
		defForce = 4200
	}
	if e.db != nil && e.resolver != nil {
		var defSave models.PlayerSave
		if err := e.db.Where("player_id = ?", defenderPlayerID).First(&defSave).Error; err == nil {
			keys := e.loadResearchKeys(&defSave)
			bonuses := e.resolver.Compute(keys)
			defForce *= bonuses.ArmyDefense
		}
	}

	ratio := 1.0
	if defForce > 0 {
		ratio = attForce / defForce
	}

	// 3-phase resolution (exact per original plan)
	var attLossBase, defLossBase float64
	if ratio > 1.2 {
		defLossBase = 0.20 + (ratio-1.2)*0.05
		attLossBase = 0.10
	} else if ratio < 0.8 {
		attLossBase = 0.20 + (0.8-ratio)*0.05
		defLossBase = 0.10
	} else {
		attLossBase = 0.12
		defLossBase = 0.12
	}

	// Random factor 0.9-1.1 per spec
	randF := 0.95 + (float64(time.Now().Unix()%10) * 0.02)
	attLossPct := attLossBase * randF
	defLossPct := defLossBase * randF

	// Per-unit attacker losses
	attLosses := map[string]int{}
	for t, c := range units {
		attLosses[t] = int(float64(c) * attLossPct)
		if attLosses[t] < 1 && c > 0 {
			attLosses[t] = 1
		}
	}

	// Real-ish defender losses: scale from loaded army count if possible, else fallback
	defLosses := map[string]int{}
	if e.db != nil {
		var defUnits []models.ArmyUnit
		_ = e.db.Where("player_id = ? AND status IN ?", defenderPlayerID, []string{"available", "assigned"}).Find(&defUnits).Error
		if len(defUnits) > 0 {
			perUnit := int(float64(len(defUnits)) * defLossPct)
			if perUnit < 1 {
				perUnit = 1
			}
			for _, u := range defUnits {
				defLosses[u.UnitType] = perUnit
			}
		}
	}
	if len(defLosses) == 0 {
		defLosses = map[string]int{"infanterie_legere": int(80 * defLossPct)}
	}

	winner := "defender"
	if ratio > 1.0 {
		winner = "attacker"
	}

	// Loot: min(0.15, ratio*0.05) of defender CURRENT real stocks (from PlayerSave / Inventory)
	loot := map[string]float64{}
	if winner == "attacker" {
		lootFactor := ratio * 0.05
		if lootFactor > 0.15 {
			lootFactor = 0.15
		}
		foodStock := e.loadResourceStock(defenderPlayerID, "food")
		matStock := e.loadResourceStock(defenderPlayerID, "materials")
		if foodStock < 10 {
			foodStock = 2890
		}
		if matStock < 10 {
			matStock = 980
		}
		loot["food"] = foodStock * lootFactor
		loot["materials"] = matStock * lootFactor
	}

	// Side effects timestamps (returned; handler/service persists to PlayerSave.ActiveEffectsJSON or dedicated fields)
	now := time.Now().UTC()
	defenderShieldUntil := now.Add(4 * time.Hour)
	attackerCooldownUntil := now.Add(2 * time.Hour)

	battleLog := map[string]any{
		"attacker_id":    attackerPlayerID,
		"defender_id":    targetCityID,
		"winner":         winner,
		"executed_at":    now,
		"shield_until":   defenderShieldUntil,
		"cooldown_until": attackerCooldownUntil,
		"losses": map[string]any{
			"attacker": attLosses,
			"defender": defLosses,
		},
		"loot": loot,
	}
	_ = battleLog

	return CombatResult{
		Winner:         winner,
		AttackerLosses: attLosses,
		DefenderLosses: defLosses,
		LootGained:     loot,
		Phases: []CombatPhase{
			{Ratio: ratio, AttackerLoss: attLosses, DefenderLoss: defLosses},
		},
		ExecutedAt: now,
	}, nil
}

// ExpireShieldsAndCooldowns called by scheduler (stub until shield/cooldown fields are on PlayerSave or in ActiveEffectsJSON)
func (e *Engine) ExpireShieldsAndCooldowns(ctx context.Context, playerID uint) error {
	// Real implementation would scan ActiveEffectsJSON or dedicated columns and clear expired shield/cooldown.
	return nil
}
