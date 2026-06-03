package services

import "cgwm/battle/internal/nexus_tribunal/adapters"

// NexusBridgeService handles import from Nexus events and verdict consequences.
type NexusBridgeService struct{}

func NewNexusBridgeService() *NexusBridgeService { return &NexusBridgeService{} }

func (s *NexusBridgeService) BuildNarrativeCaseFromNexusEvent(src adapters.NexusSource) map[string]any {
	return map[string]any{"title": src.Title, "mode": "nexus_integrated"}
}

func (s *NexusBridgeService) ImportNexusLogsAsEvidence(src adapters.NexusSource) []adapters.NexusEvidence {
	return adapters.ImportNexusEvidence(src)
}

func (s *NexusBridgeService) ImportGuildConflictAsCase(src adapters.NexusSource) map[string]any { return nil }
func (s *NexusBridgeService) ImportFactionConflictAsCase(src adapters.NexusSource) map[string]any { return nil }
func (s *NexusBridgeService) ProposeConsequencesFromVerdict(verdict string, caseID uint) []adapters.WorldConsequence {
	return adapters.ProposeWorldConsequences(adapters.VerdictProposal{Verdict: verdict, CaseID: caseID})
}

func (s *NexusBridgeService) SendVerdictToNexusGames(proposal adapters.VerdictProposal) adapters.NexusPolicyResponse {
	return adapters.SendVerdictProposalToNexusGames(proposal, nil)
}
