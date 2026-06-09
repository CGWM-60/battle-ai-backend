package balance

import (
	"math"
)

// ContentFormulas implements the v2.0 balance rules from NEXUS GAME CONTENT REFERENCE.
// All values rounded to nearest integer on server side.
// cost(level) = costLevel1 × 1.18^(level − 1)
// duration(level) = durationLevel1 × 1.16^(level − 1)
// etc.

func Cost(level int, base float64) int {
	if level < 1 {
		level = 1
	}
	val := base * math.Pow(1.18, float64(level-1))
	return int(val + 0.5)
}

func DurationSeconds(level int, baseSeconds float64) int {
	if level < 1 {
		level = 1
	}
	val := baseSeconds * math.Pow(1.16, float64(level-1))
	return int(val + 0.5)
}

func Output(level int, base float64) int {
	if level < 1 {
		level = 1
	}
	val := base * math.Pow(1.13, float64(level-1))
	return int(val + 0.5)
}

func Power(level int, base float64) int {
	if level < 1 {
		level = 1
	}
	val := base * math.Pow(1.14, float64(level-1))
	return int(val + 0.5)
}

func Upkeep(level int, base float64) int {
	if level < 1 {
		level = 1
	}
	val := base * math.Pow(1.10, float64(level-1))
	return int(val + 0.5)
}

func Health(level int, base float64) int {
	if level < 1 {
		level = 1
	}
	val := base * math.Pow(1.12, float64(level-1))
	return int(val + 0.5)
}

func Defense(level int, base float64) int {
	if level < 1 {
		level = 1
	}
	val := base * math.Pow(1.11, float64(level-1))
	return int(val + 0.5)
}

// ApplyRarityMultiplier applies the rarity cost/duration modifiers.
func ApplyRarityMultiplier(base int, rarity string, isDuration bool) int {
	var mult float64 = 1.0
	switch rarity {
	case "uncommon":
		mult = 1.4
		if isDuration {
			mult = 1.2
		}
	case "rare":
		mult = 2.1
		if isDuration {
			mult = 1.5
		}
	case "epic":
		mult = 3.5
		if isDuration {
			mult = 2.0
		}
	case "legendary":
		mult = 6.0
		if isDuration {
			mult = 3.0
		}
	case "nexus":
		mult = 12.0
		if isDuration {
			mult = 5.0
		}
	default: // common
		mult = 1.0
	}
	return int(float64(base)*mult + 0.5)
}

// TierForLevel returns the visual tier for assetsByTier (1-4).
func TierForLevel(level int) int {
	if level >= 30 {
		return 4
	}
	if level >= 20 {
		return 3
	}
	if level >= 10 {
		return 2
	}
	return 1
}
