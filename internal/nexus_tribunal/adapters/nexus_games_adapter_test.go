package adapters

import "testing"

func TestBuildCaseFromGuildConflictKeepsNexusContext(t *testing.T) {
	draft := BuildCaseFromGuildConflict(NexusSource{
		ID:          "conflict-42",
		Title:       "Sanction de guilde contestee",
		Description: "Une guilde refuse une sanction regionale.",
		WorldID:     "world-1",
		RegionID:    "region-7",
		GuildID:     "guild-9",
		Severity:    80,
	})

	if draft.Mode != "nexus_integrated" {
		t.Fatalf("expected nexus_integrated mode, got %q", draft.Mode)
	}
	if draft.NexusContext["guildId"] != "guild-9" {
		t.Fatalf("expected guild id in context, got %#v", draft.NexusContext)
	}
	if draft.Title != "Sanction de guilde contestee" {
		t.Fatalf("unexpected title %q", draft.Title)
	}
}

func TestImportNexusEvidenceClampsScores(t *testing.T) {
	items := ImportNexusEvidence(NexusSource{
		Type:        "quest",
		ID:          "quest-1",
		Title:       "Rapport de quete",
		Description: "Rapport structure par le serveur.",
		Severity:    300,
	})

	if len(items) != 1 {
		t.Fatalf("expected one evidence, got %d", len(items))
	}
	if items[0].Strength != 95 || items[0].Reliability != 95 {
		t.Fatalf("expected clamped strength/reliability 95, got %d/%d", items[0].Strength, items[0].Reliability)
	}
	if items[0].SourceID != "quest-1" {
		t.Fatalf("expected source id quest-1, got %q", items[0].SourceID)
	}
}

func TestProposeWorldConsequencesDoesNotApplyWorldChanges(t *testing.T) {
	consequences := ProposeWorldConsequences(VerdictProposal{
		CaseID:  12,
		Source:  "tribunal",
		Verdict: "partial_guilty",
		Summary: "Responsabilite partielle reconnue.",
		KeyContradictions: []Contradiction{
			{StatementID: 3, EvidenceID: 4, Type: "time_conflict"},
		},
	})

	if len(consequences) != 3 {
		t.Fatalf("expected 3 proposed consequences, got %d", len(consequences))
	}
	for _, consequence := range consequences {
		if consequence.Type == "" || consequence.Reason == "" {
			t.Fatalf("consequence must stay declarative, got %#v", consequence)
		}
	}
}

func TestSendVerdictProposalRequiresPolicy(t *testing.T) {
	proposal := VerdictProposal{
		CaseID:  1,
		Verdict: "guilty",
		ProposedConsequences: []WorldConsequence{
			{Type: "create_rumor", Reason: "test"},
		},
	}

	response := SendVerdictProposalToNexusGames(proposal, nil)
	if response.Accepted {
		t.Fatal("nil policy must not accept consequences")
	}
	if len(response.RejectedConsequences) != 1 {
		t.Fatalf("expected rejected consequence, got %#v", response)
	}

	response = SendVerdictProposalToNexusGames(proposal, func(input VerdictProposal) NexusPolicyResponse {
		return NexusPolicyResponse{Accepted: true, AppliedConsequences: input.ProposedConsequences, LoreEntryID: 88}
	})
	if !response.Accepted || response.LoreEntryID != 88 {
		t.Fatalf("expected policy response to be returned, got %#v", response)
	}
}
