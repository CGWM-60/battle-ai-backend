package services

import (
	"context"
	"fmt"
	"time"

	"cgwm/battle/internal/nexus_game/cache"
	"cgwm/battle/internal/nexus_game/models"

	"gorm.io/gorm"
)

// WorldTickService integrates into existing World Tick (or can be called from main tick).
// Uses Redis lock, performs mechanics (production, population, etc. - stubs here), calls Server AI for summaries/events.
// Per spec: IA serveur only for summary, proposal, enrichment - never bypass policies.
// Redis heavy for locks, state.
type WorldTickService struct {
	db    *gorm.DB
	redis *cache.RedisService
	ai    *ServerAIService
	ws    *WorldService
}

func NewWorldTickService(db *gorm.DB, redis *cache.RedisService) *WorldTickService {
	return &WorldTickService{
		db:    db,
		redis: redis,
		ai:    NewServerAIService(db, redis),
		ws:    NewWorldService(db, redis),
	}
}

// RunWorldTick - call this periodically (e.g. from cron/job).
// 1. Redis lock per world.
// 2. Basic mechanics (stub production/pop etc using current models).
// 3. Call IA serveur for summary (optional).
// 4. Generate event if needed (limited).
// 5. Persist.
func (s *WorldTickService) RunWorldTick(ctx context.Context, worldID uint) error {
	lockKey := fmt.Sprintf("nexus:world:tick:lock:%d", worldID)
	locked, err := s.redis.AcquireLock(ctx, lockKey, 5*time.Minute)
	if err != nil || !locked {
		return fmt.Errorf("could not lock world tick %d", worldID)
	}
	defer s.redis.ReleaseLock(ctx, lockKey)

	// Load world
	var world models.World
	if err := s.db.First(&world, worldID).Error; err != nil {
		return err
	}

	// Stub mechanics (in real: production from buildings, pop growth, research progress, etc.
	// Use ProfileGamer, Faction for counts. Redis for live capacities.
	// Example: aggregate players per continent.
	fmt.Printf("[WORLD_TICK] Tick for world %d started at %s\n", worldID, time.Now())

	// Call IA for summary (enriching, detailed).
	worldState := map[string]interface{}{
		"world_id": worldID,
		"day":      time.Now().Format("2006-01-02"),
		"tensions": 3, // from previous conflicts or Redis
	}
	summary, _ := s.ai.SummarizeWorldTick(ctx, worldState)
	fmt.Printf("[WORLD_TICK] IA Summary: %s\n", summary)

	// Optional: generate event (respect max 4/day - use Redis counter).
	eventKey := fmt.Sprintf("nexus:world:%d:events_today", worldID)
	countStr, _, _ := s.redis.GetString(ctx, eventKey)
	count := 0
	fmt.Sscanf(countStr, "%d", &count)
	if count < 4 {
		event, _ := s.ai.GenerateWorldEvent(ctx, worldState, worldID)
		_ = s.redis.SetString(ctx, eventKey, fmt.Sprintf("%d", count+1), 24*time.Hour)
		fmt.Printf("[WORLD_TICK] Proposed event: %+v\n", event)
		// In real: persist event, notify via notifications.
	}

	// Sync building production for all profiles in the world (accrual + per-tick rates)
	resourceSvc := NewResourceService(s.db)
	var profiles []models.ProfileGamer
	if err := s.db.Where("world_id = ?", worldID).Find(&profiles).Error; err == nil {
		for i := range profiles {
			_ = resourceSvc.SyncBuildingProduction(ctx, profiles[i].ID, true)
			// Re-fetch updated profile (production sync may have updated energy balance etc.)
			if err := s.db.First(&profiles[i], profiles[i].ID).Error; err != nil {
				continue
			}
			s.evolveCityStats(&profiles[i])
			s.db.Save(&profiles[i])
		}
	}

	fmt.Printf("[WORLD_TICK] Tick for world %d completed.\n", worldID)
	return nil
}

// evolveCityStats applies the evolutionary rules for Population, Morale, Energy, Security.
// Formulas from spec: growth with factors, decline, deltas, clamp, priorities, relations.
// Called per tick for profiles in the world.
func (s *WorldTickService) evolveCityStats(p *models.ProfileGamer) {
	if p.PopulationCapacity > 0 {
		baseGrowth := 10
		housingFactor := 1.0
		if p.PopulationCapacity > 0 {
			free := float64(p.PopulationCapacity-p.Population) / float64(p.PopulationCapacity)
			if free > 0.5 {
				housingFactor = 1.2
			} else if free < 0.1 {
				housingFactor = 0.2
			} else if free <= 0 {
				housingFactor = 0
			}
		}
		foodFactor := 1.0
		if p.Morale < 25 {
			foodFactor = -1.0
		} else if p.Morale < 50 {
			foodFactor = 0.5
		}
		moraleFactor := 1.0
		if p.Morale >= 80 {
			moraleFactor = 1.2
		} else if p.Morale < 25 {
			moraleFactor = -0.8
		} else if p.Morale < 50 {
			moraleFactor = 0.4
		}
		securityFactor := 1.0
		if p.Security >= 80 {
			securityFactor = 1.1
		} else if p.Security < 25 {
			securityFactor = -1.0
		} else if p.Security < 50 {
			securityFactor = 0.5
		}
		energyFactor := 1.0
		if p.EnergyBalance < 0 {
			energyFactor = 0.6
		}
		if p.EnergyBalance < -20 {
			energyFactor = -0.7
		}
		weatherFactor := 1.0 // stub, integrate real weather later

		growth := int(float64(baseGrowth) * housingFactor * foodFactor * moraleFactor * securityFactor * energyFactor * weatherFactor)

		decline := 0
		if p.Morale < 25 {
			decline += 5
		}
		if p.Security < 25 {
			decline += 5
		}
		if p.EnergyBalance < -10 {
			decline += 3
		}

		newPop := p.Population + growth - decline
		if newPop < 0 {
			newPop = 0
		}
		if newPop > p.PopulationCapacity {
			newPop = p.PopulationCapacity
		}
		p.Population = newPop
	}

	// Morale delta
	moraleDelta := 0
	if p.EnergyBalance >= 0 {
		moraleDelta += 2
	} else {
		moraleDelta -= 4
	}
	if p.Security >= 70 {
		moraleDelta += 2
	} else if p.Security < 30 {
		moraleDelta -= 5
	}
	if p.Population > p.PopulationCapacity {
		moraleDelta -= 3
	}
	p.Morale = clamp(p.Morale+moraleDelta, 0, 100)

	// Energy - use real production/consumption from SyncBuildingProduction (already set on profile).
	// Just recompute balance and handle stored energy drain.
	if p.EnergyProduction > 0 || p.EnergyConsumption > 0 {
		p.EnergyBalance = p.EnergyProduction - p.EnergyConsumption
		if p.EnergyBalance < 0 && p.EnergyStored > 0 {
			drain := -p.EnergyBalance
			if drain > p.EnergyStored {
				drain = p.EnergyStored
			}
			p.EnergyStored -= drain
			p.EnergyBalance = 0
		}
	}

	// Security delta
	secDelta := 0
	if p.Morale >= 70 {
		secDelta += 1
	}
	if p.EnergyBalance >= 0 {
		secDelta += 1
	} else {
		secDelta -= 2
	}
	if p.Population > p.PopulationCapacity*2/3 {
		secDelta -= 1
	}
	p.Security = clamp(p.Security+secDelta, 0, 100)

	// Relations example (energy low affects others)
	if p.EnergyBalance < -10 {
		p.Morale = clamp(p.Morale-3, 0, 100)
		p.Security = clamp(p.Security-2, 0, 100)
		if p.Population > 0 {
			p.Population = clamp(p.Population-1, 0, p.PopulationCapacity)
		}
	}
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
