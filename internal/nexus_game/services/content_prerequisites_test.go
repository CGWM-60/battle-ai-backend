package services

import (
	"testing"

	"cgwm/battle/internal/nexus_game/models"
)

func TestParseBuildingRequirementsAcceptsFlutterFriendlyFormats(t *testing.T) {
	cases := []struct {
		name        string
		raw         string
		wantContent string
		wantLevel   int
	}{
		{name: "array objects", raw: `[{"contentId":"building_barracks","level":2}]`, wantContent: "building_barracks", wantLevel: 2},
		{name: "legacy key", raw: `[{"buildingKey":"building_research_lab","minLevel":3}]`, wantContent: "building_research_lab", wantLevel: 3},
		{name: "nested object", raw: `{"buildings":[{"contentId":"building_ai_center","requiredLevel":4}]}`, wantContent: "building_ai_center", wantLevel: 4},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseBuildingRequirements(tc.raw)
			if len(got) != 1 {
				t.Fatalf("expected one requirement, got %d: %#v", len(got), got)
			}
			if got[0].Type != "building" || got[0].ContentID != tc.wantContent || got[0].RequiredLevel != tc.wantLevel || !got[0].Required {
				t.Fatalf("unexpected building requirement: %#v", got[0])
			}
		})
	}
}

func TestParseResearchRequirementsAcceptsStringsObjectsAndNestedPayloads(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{name: "array strings", raw: `["research_basic_tactics"]`, want: "research_basic_tactics"},
		{name: "array objects", raw: `[{"researchContentId":"research_swarm_control"}]`, want: "research_swarm_control"},
		{name: "nested object", raw: `{"research":["research_ai_summary_engine"]}`, want: "research_ai_summary_engine"},
		{name: "plain string", raw: `research_nexus_world_awareness`, want: "research_nexus_world_awareness"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseResearchRequirements(tc.raw)
			if len(got) != 1 {
				t.Fatalf("expected one requirement, got %d: %#v", len(got), got)
			}
			if got[0].Type != "research" || got[0].ContentID != tc.want || got[0].RequiredLevel != 1 || !got[0].Required {
				t.Fatalf("unexpected research requirement: %#v", got[0])
			}
		})
	}
}

func TestNormalizeDomainForPrerequisiteValidation(t *testing.T) {
	cases := map[string]string{
		"build":         "building",
		"buildings":     "building",
		"construction":  "building",
		"units":         "unit",
		"research_node": "research",
		"tech":          "research",
	}

	for raw, want := range cases {
		if got := normalizeDomain(raw); got != want {
			t.Fatalf("normalizeDomain(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestBuildingDurationUsesMMOMultiplierAndMilestoneReduction(t *testing.T) {
	svc := NewContentService(nil, "")
	def := &models.BuildingDefinition{
		DurationBaseSeconds: 600,
		DurationMultiplier:  1.28,
		MilestoneReduction:  0.15,
	}

	if got := svc.CalculateBuildingDurationAtLevel(def, 1, "common"); got != 600 {
		t.Fatalf("level 1 duration = %d, want 600", got)
	}
	if got := svc.CalculateBuildingDurationAtLevel(def, 5, "common"); got != 1369 {
		t.Fatalf("level 5 milestone duration = %d, want 1369", got)
	}
	if got := svc.CalculateBuildingDurationAtLevel(def, 6, "common"); got != 2062 {
		t.Fatalf("level 6 duration = %d, want 2062", got)
	}
}

func TestApplyBuildingOrUnlockRemovesMissingBuildingRequirementsWhenOnePathIsSatisfied(t *testing.T) {
	requirements := []RequirementStatus{
		{Type: "building", ContentID: "building_diplomatic_center", Required: true, Satisfied: false},
		{Type: "building", ContentID: "building_nexus_market", Required: true, Satisfied: true},
		{Type: "research", ContentID: "research_guild_charter", Required: true, Satisfied: false},
	}
	missing := []RequirementStatus{requirements[0], requirements[2]}

	filtered := applyBuildingOrUnlock(requirements, missing)
	if len(filtered) != 1 {
		t.Fatalf("expected only non-building missing requirement, got %#v", filtered)
	}
	if filtered[0].Type != "research" {
		t.Fatalf("expected research requirement to remain, got %#v", filtered[0])
	}
}
