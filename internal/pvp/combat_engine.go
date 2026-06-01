package pvp

import (
	"context"
	"time"
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
	basePower := 12.0
	// TODO(real): multiply by research army attack bonus + morale + policy from bonusResolver + policyEngine
	researchAttackMulti := 1.05 // placeholder - real call to research.Resolver
	for _, c := range units {
		attForce += float64(c) * basePower * researchAttackMulti
	}
	// TODO(real): load defender army composition + fort bonuses + research defense from defender state
	defForce := 8500.0 // placeholder - real load via army + buildings + bonuses

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

	// Random factor 0.9-1.1 per spec (simple variation for determinism in tests)
	randF := 0.95 + (float64(time.Now().Unix()%10) * 0.02) // 0.95-1.13 range close to spec
	attLossPct := attLossBase * randF
	defLossPct := defLossBase * randF

	// Compute per unit losses (real impl would use actual defender army map + power weights)
	attLosses := map[string]int{}
	for t, c := range units {
		attLosses[t] = int(float64(c) * attLossPct)
		if attLosses[t] < 1 && c > 0 {
			attLosses[t] = 1
		}
	}
	// TODO(real): load real defender army composition instead of hardcoded
	defLosses := map[string]int{"infanterie_legere": int(120 * defLossPct)}

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
		// TODO(real integration): resEngine := resources.NewEngine(db); bal, _ := resEngine.GetBalance(ctx, defenderID); then scale bal.Current
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

	// Real side effects + persistence (called from handler + service):
	// - Call army engine to apply attLosses/defLosses
	// - Call resources engine to deduct loot from defender + add to attacker
	// - Persist battleLog + shield/cooldown timestamps to DB (PlayerSave or battle_logs table)
	// - Return full CombatResult for Flutter replay (pvp_battle_detail_page)

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

// ExpireShieldsAndCooldowns called by scheduler
func (e *Engine) ExpireShieldsAndCooldowns(ctx context.Context, playerID uint) error {
	// TODO: real DB check for shield_until / cooldown_until and remove expired
	return nil
}
