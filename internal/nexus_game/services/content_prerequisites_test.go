package services

import "testing"

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
