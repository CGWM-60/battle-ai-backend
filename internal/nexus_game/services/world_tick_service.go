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

	// Persist log or update world.
	// TODO: full production/consumption using existing engines if integrated.

	fmt.Printf("[WORLD_TICK] Tick for world %d completed.\n", worldID)
	return nil
}
