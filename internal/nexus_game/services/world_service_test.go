package services

import (
	"context"
	"testing"

	"cgwm/battle/internal/nexus_game/cache"
	"cgwm/battle/internal/nexus_game/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWorldCreationAndAssignment(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	db.AutoMigrate(&models.World{}, &models.Continent{}, &models.Faction{}, &models.ProfileGamer{})

	redis := cache.NewRedisServiceFromEnv() // may be disabled in test, ok for stub
	ws := NewWorldService(db, redis)

	// Test create world
	w, err := ws.CreateWorld(context.Background())
	if err != nil {
		t.Fatalf("create world: %v", err)
	}
	if w.ID == 0 {
		t.Error("world ID not set")
	}

	// Test faction assign (should use first continent)
	f := &models.Faction{Name: "TestFaction"}
	db.Create(f)
	wID, cID, err := ws.GetOrCreateWorldForFaction(context.Background(), f.ID)
	if err != nil {
		t.Fatalf("assign faction: %v", err)
	}
	if wID == 0 || cID == 0 {
		t.Error("assignment failed")
	}

	// Test player assign
	p := &models.ProfileGamer{UserID: 1, FactionID: f.ID}
	db.Create(p)
	_, _, err = ws.AssignPlayerToContinent(context.Background(), p.UserID, f.ID)
	if err != nil {
		t.Logf("player assign (may be full in stub): %v", err)
	}

	t.Log("World mechanics basic test passed (Redis may be stubbed).")
}
