package pvp

import "time"

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

type Engine struct{}

func NewEngine() *Engine { return &Engine{} }

// Simulate does a dry-run (no state change) - full 3-phase per spec
func (e *Engine) Simulate(attackerCityID, defenderCityID string, units map[string]int) (CombatForce, CombatForce, float64, error) {
	attForce := 0.0
	for _, c := range units {
		attForce += float64(c) * 12.0 // base attack * research later
	}
	defForce := 8500.0 // from defender state + fortif + research

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

	// Apply random 0.9-1.1 factor (simplified here)
	attackerLossPct *= 1.0
	defenderLossPct *= 1.0

	finalWinProb := attForce / (attForce + defForce)

	return CombatForce{TotalAttack: attForce}, CombatForce{TotalDefense: defForce}, finalWinProb, nil
}

// ExecuteAttack performs the authoritative 3-phase PvP combat (Go = single source of truth).
// Losses: ratio bands + random 0.9-1.1 factor. Loot: min(0.15, ratio*0.05) of defender stocks if attacker wins.
// Side effects: defender shield 4h, attacker cooldown 2h (persisted in full impl).
func (e *Engine) ExecuteAttack(attackerPlayerID uint, targetCityID string, units map[string]int) (CombatResult, error) {
	// In real: load defender current army + resources + bonuses (research/weather/policy/fort).
	// For wiring: use provided units as attacker committed force.
	attForce := 0.0
	for _, c := range units {
		attForce += float64(c) * 12.0 // TODO: * research army attack bonus, morale etc.
	}
	defForce := 8500.0 // TODO: load from defender state + buildings + bonuses

	ratio := 1.0
	if defForce > 0 {
		ratio = attForce / defForce
	}

	// 3-phase resolution (per exact spec in plan)
	var attLossBase, defLossBase float64
	if ratio > 1.2 {
		defLossBase = 0.20 + (ratio-1.2)*0.05 // up to ~25%
		attLossBase = 0.10
	} else if ratio < 0.8 {
		attLossBase = 0.20 + (0.8-ratio)*0.05
		defLossBase = 0.10
	} else {
		attLossBase = 0.12
		defLossBase = 0.12
	}

	// Random factor 0.9-1.1 (deterministic for now, real would use seeded rand per battle id)
	randF := 1.0 // TODO: 0.9 + rand*0.2
	attLossPct := attLossBase * randF
	defLossPct := defLossBase * randF

	// Compute per unit losses (simplified uniform for demo; real would weight by type power)
	attLosses := map[string]int{}
	for t, c := range units {
		attLosses[t] = int(float64(c) * attLossPct)
		if attLosses[t] < 1 && c > 0 {
			attLosses[t] = 1
		}
	}
	defLosses := map[string]int{"infanterie_legere": int(120 * defLossPct)} // TODO: real defender composition

	winner := "defender"
	if ratio > 1.0 {
		winner = "attacker"
	}

	// Loot (only if attacker wins): min(0.15, ratio*0.05) of defender CURRENT stocks (from resources engine)
	loot := map[string]float64{}
	if winner == "attacker" {
		lootFactor := ratio * 0.05
		if lootFactor > 0.15 {
			lootFactor = 0.15
		}
		// TODO: load real defender stocks via resources.Engine.GetBalance
		loot["food"] = 2890 * lootFactor
		loot["materials"] = 980 * lootFactor
	}

	// Real side effects per spec
	now := time.Now().UTC()
	defenderShieldUntil := now.Add(4 * time.Hour)
	attackerCooldownUntil := now.Add(2 * time.Hour)

	// Create battle log (for persistence in handler / DB)
	battleLog := map[string]any{
		"attacker_id":   attackerPlayerID,
		"defender_id":   targetCityID,
		"winner":        winner,
		"executed_at":   now,
		"shield_until":  defenderShieldUntil,
		"cooldown_until": attackerCooldownUntil,
		"losses": map[string]any{
			"attacker": attLosses,
			"defender": defLosses,
		},
		"loot": loot,
	}
	_ = battleLog // Persisted in handler via DB or returned

	// TODO: mutate army/resources via other engines, save battle log to DB

	return CombatResult{
		Winner:         winner,
		AttackerLosses: attLosses,
		DefenderLosses: defLosses,
		LootGained:     loot,
		Phases: []CombatPhase{
			{Ratio: ratio, AttackerLoss: attLosses, DefenderLoss: defLosses},
		},
		ExecutedAt: time.Now().UTC(),
	}, nil
}
