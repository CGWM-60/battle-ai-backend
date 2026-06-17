package service

import (
	"testing"

	"cgwm/battle/internal/models"
)

func TestBuildLocalArcMapFallback(t *testing.T) {
	arcMap, err := BuildLocalArcMapFallback(1, 1, "Les bas-fonds huilés", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if arcMap.Name != "Les bas-fonds huilés" {
		t.Fatalf("expected arc title, got %s", arcMap.Name)
	}

	nodes, edges, objectives := parseArcMap(arcMap)
	if len(nodes) < 12 {
		t.Fatalf("expected at least 12 nodes, got %d", len(nodes))
	}
	if len(edges) < 10 {
		t.Fatalf("expected at least 10 edges, got %d", len(edges))
	}
	if len(objectives) < 3 {
		t.Fatalf("expected at least 3 objectives, got %d", len(objectives))
	}

	entrance := findEntranceNode(nodes)
	if entrance == "" {
		t.Fatal("expected entrance node")
	}

	bossFound := false
	leverFound := false
	for _, n := range nodes {
		if n.IsBossRoom {
			bossFound = true
		}
		if n.IsLever {
			leverFound = true
		}
	}
	if !bossFound {
		t.Fatal("expected boss room")
	}
	if !leverFound {
		t.Fatal("expected lever node")
	}
}

func TestIsAdjacent(t *testing.T) {
	edges := []models.RPGMapEdge{
		{From: "a", To: "b"},
		{From: "b", To: "c", Locked: true, RequiredFlag: "door_unlocked"},
	}
	flags := map[string]any{}
	if !isAdjacent(edges, "a", "b", []string{"a", "b"}, flags) {
		t.Fatal("a->b should be adjacent")
	}
	if isAdjacent(edges, "b", "c", []string{"b", "c"}, flags) {
		t.Fatal("b->c should be locked")
	}
	flags["door_unlocked"] = true
	if !isAdjacent(edges, "b", "c", []string{"b", "c"}, flags) {
		t.Fatal("b->c should be unlocked")
	}
}

func TestAllEnemiesDefeated(t *testing.T) {
	enemies := []models.RPGMapEnemy{
		{ID: "e1", Defeated: true},
		{ID: "e2", Defeated: false},
	}
	if allEnemiesDefeated(enemies) {
		t.Fatal("not all defeated")
	}
	enemies[1].Defeated = true
	if !allEnemiesDefeated(enemies) {
		t.Fatal("all should be defeated")
	}
}