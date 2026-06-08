package seeds

import (
	"context"
	"fmt"

	"cgwm/battle/internal/nexus_game/models"
	"cgwm/battle/internal/nexus_game/services"

	"gorm.io/gorm"
)

// SeedInitialWorlds - call on startup or admin to create initial world if none.
// Creates 1 world with 5 continents. Factions/players will assign proportionally.
func SeedInitialWorlds(db *gorm.DB, worldSvc *services.WorldService) error {
	var count int64
	db.Model(&models.World{}).Count(&count)
	if count > 0 {
		fmt.Println("[SEED] Worlds already exist, skipping initial seed.")
		return nil
	}

	w, err := worldSvc.CreateWorld(context.Background())
	if err != nil {
		return err
	}
	fmt.Printf("[SEED] Created initial world %d with 5 continents.\n", w.ID)

	// Seed some example prompts for IA serveur (CRUD will allow modification later).
	prompts := []models.Prompt{
		{
			PromptID:     "quest_seed_generation",
			Version:      "v1.2-optimized-2026",
			Domain:       "quest_seed_generation",
			Purpose:      "Transform player reports into controlled quest seeds.",
			SystemPrompt: "You are Nexus server AI. Output ONLY JSON. Optimized for cost/speed. Enriching lore hooks. Respect max rewards, no bypass.",
			IsActive:     true,
		},
		{
			PromptID:     "event_generation",
			Version:      "v1.1-world-event-optimized",
			Domain:       "event_generation",
			Purpose:      "Propose world events based on state.",
			SystemPrompt: "Generate enriching event. Max 4/day. Linked to region/faction. Detailed narrative.",
			IsActive:     true,
		},
	}
	for _, p := range prompts {
		db.Create(&p)
	}
	fmt.Println("[SEED] Seeded example IA prompts (modifiable via /prompts).")
	return nil
}
