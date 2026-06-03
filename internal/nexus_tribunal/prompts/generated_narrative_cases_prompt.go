package prompts

import "fmt"

// BuildSingleNarrativeCasePrompt generates a prompt for **exactly one** fully scenarized
// narrative Tribunal case for a specific level (1-10).
// This is called in a loop (1 by 1) to avoid huge prompts/timeouts.
// Output MUST be ONLY the case object as valid JSON (no wrapper, no extra text).
func BuildSingleNarrativeCasePrompt(level int) (system string, user string) {
	if level < 1 || level > 10 {
		level = 5
	}

	system = "Tu es un générateur expert d'affaires Tribunal IA cyberpunk originales (style enquête interactive type Phoenix Wright mais 100% original, sans aucune référence à des licences existantes). Tu génères des cas COMPLETS et JOUABLES avec actes, scènes, contradictions, fausses pistes, crises, objections, révélations et verdicts. Réponds UNIQUEMENT par du JSON valide strict (l'objet de l'affaire directement), sans markdown, sans explication, sans texte hors JSON."

	user = fmt.Sprintf(`Génère EXACTEMENT UNE affaire Tribunal IA NARRATIVE ET SCÉNARISÉE pour le niveau %d (difficulté progressive).

Règles:
- Niveau exact: %d
- Thème, cast et contradictions différents des affaires typiques de ce niveau.
- Français cyberpunk.
- Structure complète et jouable (Phoenix-like mais original).

Structure EXACTE de l'affaire (objet JSON racine, pas de "cases"):
{
  "title": "titre court original",
  "synopsis": "1-2 phrases",
  "level": %d,
  "difficulty": "initiation|easy|standard|intermediate|confirmed|hard|expert|master|legendary|nexus",
  "estimatedDurationMinutes": 8,
  "caseType": "moral|guild_conflict|world_event|absurd|player_conflict|...",
  "mode": "full_case",
  "tone": "cyberpunk_serious",
  "realTruth": "la vérité cachée",
  "publicTruth": "ce que l'opinion publique croit",
  "finalReveal": "la révélation finale",
  "cast": [
    {"actorType":"judge|prosecutor|defense_attorney|witness|expert_witness|assistant|clerk|jury_logic|...", "name":"...", "personality":"...", "avatarAssetId":"tribunal.character.xxx"}
  ],
  "acts": [{"actIndex":1,"title":"...","objective":"...","summary":"..."}],
  "scenes": [
    {
      "sceneId": "act1_xxx",
      "actIndex":1,
      "sceneIndex":0,
      "sceneType": "intro|briefing|witness_testimony|cross_examination|crisis|reveal|final_plea|verdict",
      "title": "...",
      "objective": "...",
      "narrativeText": "...",
      "activeWitnessId": "nom du témoin ou null",
      "activeActorIds": ["..."],
      "availableEvidenceIds": ["..."],
      "visibleStatementIds": ["..."],
      "allowedActions": ["press","present_evidence","objection","ai_analysis","ask_hint","expose_lie","continue_story"],
      "nextSceneId": "..." ou null
    }
  ],
  "progressionRules": [
    {
      "sceneId": "...",
      "triggerAction": "present_evidence|objection|expose_lie|...",
      "requiredEvidenceId": "..." ou null,
      "requiredStatementId": "..." ou null,
      "resultType": "minor_contradiction|major_reveal|crisis_trigger|...",
      "isCritical": true,
      "unlockSceneId": "..." ou null,
      "narrativeResult": "...",
      "scoreEffects": {"defenseScoreDelta": 8, "witnessCredibilityDelta": -12, "tribunalPressureDelta": 10}
    }
  ],
  "failureRules": [
    {
      "sceneId": "...",
      "triggerAction": "...",
      "penaltyType": "score_down",
      "maxFailuresBeforeHint": 2,
      "judgeWarningText": "...",
      "hintText": "...",
      "scoreEffects": {"defenseScoreDelta":-3},
      "stayOnScene": true
    }
  ],
  "crisisMoment": {"sceneId":"...","trigger":"...","effect":"..."} ou null,
  "possibleVerdicts": ["defense_win","partial_defense","partial_guilty","guilty","hidden_truth"],
  "epilogue": "...",
  "nexusBridgeHints": [{"type":"...","targetId":"...","delta":-5}]
}

Génère l'objet JSON complet pour le niveau %d. JSON strict uniquement, rien d'autre.`, level, level, level, level)

	return system, user
}

// BuildNarrativeCasesPrompt is kept for backward compatibility but now internally
// recommends using BuildSingleNarrativeCasePrompt + loop (1 by 1) to avoid timeouts.
func BuildNarrativeCasesPrompt(count int) (system string, user string) {
	if count <= 0 {
		count = 10
	}
	// For compatibility, still generate a prompt, but the generation code should
	// prefer calling BuildSingleNarrativeCasePrompt in a loop for reliability.
	return BuildSingleNarrativeCasePrompt(5) // fallback example
}
