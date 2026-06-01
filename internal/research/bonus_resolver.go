package research

// ResearchBonuses is the central multiplier object returned by the resolver.
// Every engine (resources, economy, army, construction, etc.) must call this.
type ResearchBonuses struct {
	ProductionMultipliers map[string]float64 `json:"productionMultipliers"`
	ConstructionSpeed     float64            `json:"constructionSpeed"`
	ArmyAttack            float64            `json:"armyAttack"`
	ArmyDefense           float64            `json:"armyDefense"`
	EconomyBonus          float64            `json:"economyBonus"`
	StorageCapacity       float64            `json:"storageCapacity"`
}

// Resolver computes the aggregated bonuses from unlocked research nodes.
// Formula (per original spec): product of all unlocked node multipliers per domain.
type Resolver struct {
	// nodeEffects defines the multiplier contribution of each research node key.
	// These are the canonical effects used by every engine (resources, pvp, economy, population, etc.).
	nodeEffects map[string]ResearchBonuses
}

func NewResolver() *Resolver {
	return &Resolver{
		nodeEffects: map[string]ResearchBonuses{
			// Army domain (examples from arbre_de_recherche)
			"army_attack_1":   {ArmyAttack: 1.05},
			"army_attack_2":   {ArmyAttack: 1.08},
			"army_defense_1":  {ArmyDefense: 1.05},
			"army_defense_2":  {ArmyDefense: 1.08},
			"army_morale_1":   {ArmyAttack: 1.03, ArmyDefense: 1.03},
			// Production domain
			"prod_gold_1":     {ProductionMultipliers: map[string]float64{"gold": 1.06}},
			"prod_energy_1":   {ProductionMultipliers: map[string]float64{"energy": 1.07}},
			"prod_food_1":     {ProductionMultipliers: map[string]float64{"food": 1.05}},
			"prod_materials_1": {ProductionMultipliers: map[string]float64{"materials": 1.06}},
			"prod_all_1":      {ProductionMultipliers: map[string]float64{"gold": 1.02, "energy": 1.02, "food": 1.02, "materials": 1.02}},
			// Economy / storage / construction
			"econ_tax_1":      {EconomyBonus: 1.04},
			"storage_1":       {StorageCapacity: 1.10},
			"build_speed_1":   {ConstructionSpeed: 1.08},
			"build_speed_2":   {ConstructionSpeed: 1.12},
		},
	}
}

// Compute takes the list of unlocked node keys and returns the final multipliers.
// The formula is: product of all node multipliers in the same domain (exact per plan).
func (r *Resolver) Compute(unlockedNodeKeys []string) ResearchBonuses {
	res := ResearchBonuses{
		ProductionMultipliers: map[string]float64{
			"gold": 1.0, "energy": 1.0, "food": 1.0, "water": 1.0, "materials": 1.0, "research_points": 1.0,
		},
		ConstructionSpeed: 1.0,
		ArmyAttack:        1.0,
		ArmyDefense:       1.0,
		EconomyBonus:      1.0,
		StorageCapacity:   1.0,
	}

	for _, key := range unlockedNodeKeys {
		effect, ok := r.nodeEffects[key]
		if !ok {
			continue
		}
		// Product per domain
		res.ArmyAttack *= effect.ArmyAttack
		res.ArmyDefense *= effect.ArmyDefense
		res.ConstructionSpeed *= effect.ConstructionSpeed
		res.EconomyBonus *= effect.EconomyBonus
		res.StorageCapacity *= effect.StorageCapacity

		for resName, m := range effect.ProductionMultipliers {
			if cur, exists := res.ProductionMultipliers[resName]; exists {
				res.ProductionMultipliers[resName] = cur * m
			} else {
				res.ProductionMultipliers[resName] = m
			}
		}
	}
	return res
}
