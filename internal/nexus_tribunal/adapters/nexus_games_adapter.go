package adapters

import (
	"fmt"
	"strings"
	"time"
)

type NexusSource struct {
	Type        string
	ID          string
	Title       string
	Description string
	WorldID     string
	RegionID    string
	QuestID     string
	GuildID     string
	FactionID   string
	Severity    int
	Metadata    map[string]any
}

type TribunalCaseDraft struct {
	Title              string
	CaseType           string
	Description        string
	AccusationPosition string
	DefensePosition    string
	PlayerRole         string
	Mode               string
	Visibility         string
	NexusContext       map[string]any
}

type NexusEvidence struct {
	Title        string
	Description  string
	EvidenceType string
	SourceType   string
	SourceID     string
	Strength     int
	Reliability  int
	SupportsSide string
	Tags         []string
	AssetID      string
}

type VerdictProposal struct {
	CaseID               uint
	Source               string
	Verdict              string
	Summary              string
	KeyContradictions    []Contradiction
	ProposedConsequences []WorldConsequence
}

type Contradiction struct {
	StatementID uint
	EvidenceID  uint
	Type        string
}

type WorldConsequence struct {
	Type       string
	TargetID   string
	Delta      int
	Visibility string
	RegionID   string
	Reason     string
	Metadata   map[string]any
}

type NexusPolicyResponse struct {
	Accepted             bool
	AppliedConsequences  []WorldConsequence
	RejectedConsequences []WorldConsequence
	LoreEntryID          uint
	Message              string
}

func BuildCaseFromWorldEvent(source NexusSource) TribunalCaseDraft {
	source = normalizeSource(source, "world_event")
	title := defaultText(source.Title, "Litige evenement monde")
	return baseDraft(source, title, "world_event", "Une decision ou un evenement monde demande un arbitrage Tribunal.")
}

func BuildCaseFromQuest(source NexusSource) TribunalCaseDraft {
	source = normalizeSource(source, "quest")
	title := defaultText(source.Title, "Litige de quete Nexus")
	draft := baseDraft(source, title, "quest_dispute", "Une quete terminee produit un conflit de responsabilite.")
	draft.NexusContext["questId"] = source.QuestID
	return draft
}

func BuildCaseFromGuildConflict(source NexusSource) TribunalCaseDraft {
	source = normalizeSource(source, "guild_conflict")
	title := defaultText(source.Title, "Conflit de guilde")
	draft := baseDraft(source, title, "guild_conflict", "Deux guildes contestent une action ou une sanction.")
	draft.NexusContext["guildId"] = source.GuildID
	return draft
}

func BuildCaseFromFactionConflict(source NexusSource) TribunalCaseDraft {
	source = normalizeSource(source, "faction_conflict")
	title := defaultText(source.Title, "Conflit de faction")
	draft := baseDraft(source, title, "faction_conflict", "Une faction conteste une consequence politique ou territoriale.")
	draft.NexusContext["factionId"] = source.FactionID
	return draft
}

func ImportNexusEvidence(source NexusSource) []NexusEvidence {
	source = normalizeSource(source, "world_event")
	reliability := clamp(65+source.Severity/3, 45, 95)
	strength := clamp(55+source.Severity/2, 35, 95)
	return []NexusEvidence{
		{
			Title:        defaultText(source.Title, "Journal Nexus"),
			Description:  defaultText(source.Description, "Trace Nexus importee pour analyse Tribunal."),
			EvidenceType: "world_log",
			SourceType:   source.Type,
			SourceID:     source.ID,
			Strength:     strength,
			Reliability:  reliability,
			SupportsSide: "neutral",
			Tags:         compactTags("nexus", source.Type, source.RegionID, source.GuildID, source.FactionID),
			AssetID:      "tribunal.evidence.world_log",
		},
	}
}

func ProposeWorldConsequences(proposal VerdictProposal) []WorldConsequence {
	verdict := strings.ToLower(strings.TrimSpace(proposal.Verdict))
	consequences := make([]WorldConsequence, 0)
	if verdict == "" || verdict == "neutral" {
		consequences = append(consequences, WorldConsequence{
			Type:       "create_lore_entry",
			Visibility: "local",
			Reason:     "Verdict neutre archive sans sanction directe.",
		})
		return consequences
	}
	delta := 1
	if strings.Contains(verdict, "guilty") {
		delta = -3
	}
	consequences = append(consequences,
		WorldConsequence{
			Type:       "faction_reputation_delta",
			Delta:      delta,
			Visibility: "regional",
			Reason:     defaultText(proposal.Summary, "Reputation ajustee par proposition Tribunal."),
		},
		WorldConsequence{
			Type:       "create_rumor",
			Visibility: "regional",
			Reason:     "Rumeur creee depuis verdict Tribunal, a valider par policies Nexus Games.",
		},
	)
	if len(proposal.KeyContradictions) > 0 {
		consequences = append(consequences, WorldConsequence{
			Type:       "create_lore_entry",
			Visibility: "global",
			Reason:     fmt.Sprintf("%d contradiction(s) cle(s) reconnue(s) par le Tribunal.", len(proposal.KeyContradictions)),
		})
	}
	return consequences
}

func SendVerdictProposalToNexusGames(proposal VerdictProposal, policy func(VerdictProposal) NexusPolicyResponse) NexusPolicyResponse {
	if policy == nil {
		return NexusPolicyResponse{
			Accepted:             false,
			RejectedConsequences: proposal.ProposedConsequences,
			Message:              "Aucune policy Nexus Games fournie. Aucune consequence appliquee.",
		}
	}
	return policy(proposal)
}

func baseDraft(source NexusSource, title string, caseType string, fallbackDescription string) TribunalCaseDraft {
	description := defaultText(source.Description, fallbackDescription)
	return TribunalCaseDraft{
		Title:              title,
		CaseType:           caseType,
		Description:        description,
		AccusationPosition: "Le monde Nexus signale une responsabilite ou une contradiction a juger.",
		DefensePosition:    "La defense peut contester l'intention, le contexte ou la fiabilite des traces.",
		PlayerRole:         "defense",
		Mode:               "nexus_integrated",
		Visibility:         "private",
		NexusContext: map[string]any{
			"source":    source.Type,
			"sourceId":  source.ID,
			"worldId":   source.WorldID,
			"regionId":  source.RegionID,
			"createdAt": time.Now().UTC().Format(time.RFC3339),
			"metadata":  source.Metadata,
		},
	}
}

func normalizeSource(source NexusSource, fallbackType string) NexusSource {
	source.Type = defaultText(source.Type, fallbackType)
	source.ID = defaultText(source.ID, "unknown")
	if source.Metadata == nil {
		source.Metadata = map[string]any{}
	}
	return source
}

func compactTags(values ...string) []string {
	tags := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			tags = append(tags, value)
		}
	}
	return tags
}

func defaultText(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func clamp(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
