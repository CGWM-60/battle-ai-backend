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
    {"actorId":"acteur_slug_unique", "actorType":"judge|prosecutor|defense_attorney|witness|expert_witness|assistant|clerk|jury_logic|jury_emotional|jury_expert|...", "name":"...", "personality":"...", "avatarAssetId":"ID_MANIFEST_AUTORISE_UNIQUEMENT"}
  ],
	  "evidence": [
	    {"evidenceId":"preuve_slug_unique", "title":"titre français", "description":"description claire en français", "details":"analyse exploitable en 2-4 phrases", "origin":"origine précise dans l'enquête", "contradictionHint":"quelle déclaration cette preuve peut contredire", "chainOfCustody":"traçabilité courte", "evidenceType":"document|log|image|audio|video|biometric_log", "strength":70, "reliability":80, "supportsSide":"defense|accusation|neutral", "assetId":"tribunal.evidence.document"}
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
      "narrativeText": "texte de dialogue/narration en français, jamais en anglais",
      "activeWitnessId": "actorId du témoin actif ou null",
      "activeActorIds": ["actorId exact depuis cast"],
      "availableEvidenceIds": ["evidenceId exact depuis evidence"],
      "visibleStatementIds": ["stmt_..."],
	      "visibleStatements": [
	        {"statementId":"stmt_...", "speakerActorId":"actorId exact depuis cast", "text":"déclaration complète en français, unique, différente de narrativeText et différente des autres déclarations"}
	      ],
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
	  "verdictSummary": "résumé final détaillé: verdict, preuve clé, contradiction majeure, conséquence morale",
	  "nexusBridgeHints": [{"type":"...","targetId":"...","delta":-5}]
	}

VERROU ASSETS PERSONNAGES:
- Interdiction absolue d'inventer un avatarAssetId depuis le nom du personnage.
- avatarAssetId doit être exactement un des IDs suivants, aucun autre:
  tribunal.character.judge_ai
  tribunal.character.prosecutor_ai
  tribunal.character.defense_ai
  tribunal.character.witness_default
  tribunal.character.clerk_ai
  tribunal.character.fact_checker_ai
  tribunal.character.jury_logic
  tribunal.character.jury_emotional
  tribunal.character.jury_expert
  tribunal.character.assistant_ai
  tribunal.character.expert_witness
  tribunal.character.witness_civilian
  tribunal.character.witness_agent
  tribunal.character.witness_hacker
  tribunal.character.witness_guild_master
  tribunal.character.witness_faction_envoy
  tribunal.character.witness_android
  tribunal.character.witness_corrupted_ai
- Mapping obligatoire:
  judge -> tribunal.character.judge_ai
  prosecutor -> tribunal.character.prosecutor_ai
  defense_attorney -> tribunal.character.defense_ai
  assistant -> tribunal.character.assistant_ai
  clerk -> tribunal.character.clerk_ai
  expert_witness -> tribunal.character.expert_witness
  jury_logic -> tribunal.character.jury_logic
  jury_emotional -> tribunal.character.jury_emotional
  jury_expert -> tribunal.character.jury_expert
  witness -> un witness_* adapté; si doute -> tribunal.character.witness_default

VERROU INTERLOCUTEUR / DIALOGUE:
- Chaque entree de cast doit avoir "actorId" stable, court, unique, sans espace.
- Chaque scene doit avoir "activeActorIds" avec au moins un "actorId" exact du cast.
- "activeWitnessId" doit etre un "actorId", pas un nom libre.
- "narrativeText" doit correspondre a ce que dit ou presente l'interlocuteur actif de la scene.
- Tous les textes visibles par le joueur doivent etre en francais. Pas d'anglais dans titres, declarations, preuves, objectifs, resultats.
- Chaque "availableEvidenceIds[]" doit exister dans "evidence[].evidenceId".
- Chaque "visibleStatementIds[]" doit exister dans "visibleStatements[].statementId".
- Chaque "visibleStatements[].speakerActorId" doit etre un "actorId" exact du cast.
- Chaque "visibleStatements[].text" doit etre une declaration jouable en francais, pas un identifiant technique.
- Interdiction de recopier "narrativeText" dans les declarations. Si une scene a 3 declarations, les 3 textes doivent contenir 3 informations differentes.
- Chaque preuve doit contenir "details", "origin", "contradictionHint" et "chainOfCustody" pour affichage en popin.
- Les progressionRules doivent couvrir au minimum: press sur une declaration importante, present_evidence avec preuve cible, objection avec preuve cible, expose_lie pour la contradiction majeure, continue_story seulement apres contradiction critique.
- Les scenes finales doivent fournir un "narrativeText" conclusif et l'affaire doit fournir "epilogue" et "verdictSummary" utilisables sans renvoyer directement aux archives.

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
