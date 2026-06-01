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
type Resolver struct {
	// TODO: load node definitions from DB or JSON (building_rules)
}

func NewResolver() *Resolver {
	return &Resolver{}
}

// Compute takes the list of unlocked node keys and returns the final multipliers.
// The formula is: product of all node multipliers in the same domain.
func (r *Resolver) Compute(unlockedNodeKeys []string) ResearchBonuses {
	// TODO: real implementation
	// For now return sane defaults so the system can boot.
	return ResearchBonuses{
		ProductionMultipliers: map[string]float64{
			"gold": 1.0, "energy": 1.0, "food": 1.0, "water": 1.0, "materials": 1.0, "research_points": 1.0,
		},
		ConstructionSpeed: 1.0,
		ArmyAttack:        1.0,
		ArmyDefense:       1.0,
		EconomyBonus:      1.0,
		StorageCapacity:   1.0,
	}
}
