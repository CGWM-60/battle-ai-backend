package prompts

import "fmt"

// BuildNarrativeCasesPrompt returns strict prompt for exactly 10 fully scenarized
// narrative Tribunal cases (Phoenix-like, original cyberpunk IA, no license copy).
// Output: {"cases": [ {title, synopsis, level, ..., cast, acts, scenes, progressionRules, failureRules, crisisMoment, possibleVerdicts, epilogue, nexusBridgeHints } ]}
func BuildNarrativeCasesPrompt(count int) (system string, user string) {
	if count <= 0 {
		count = 10
	}
	system = "Tu es un generateur expert d'affaires de Tribunal IA cyberpunk original et scenarisees (style Phoenix Wright mais 100% original, aucun nom/personnage/musique/UI de licence existante). Tu produis des cas JOUABLES avec trame complete : plusieurs actes, scenes, contradictions, fausses pistes, moments de crise, grandes objections, revelations, verdicts nuances. Reponds UNIQUEMENT en JSON valide strict, pas de markdown, pas de texte hors JSON."

	user = fmt.Sprintf(`Genere exactement %d affaires Tribunal IA NARRATIVES ET SCENARISEES (niveaux 1 a 10 uniques, difficulte progressive, tout en francais cyberpunk).

Contraintes de variete obligatoires:
- pas deux fois le meme theme principal
- pas deux fois le meme cast (juge/procureur/avocat)
- minimum 2 affaires moral/philosophical
- minimum 2 guild/faction
- minimum 2 world_event
- minimum 1 absurde/satirique
- minimum 1 affaire Nexus integre niveau 10 avec bridge hints

Chaque affaire DOIT contenir:
- title, synopsis, realTruth, publicTruth, finalReveal
- level (1-10), difficulty, estimatedDurationMinutes, caseType, mode:"full_case", tone
- cast: tableau de 4-7 acteurs (judge, prosecutor, defense_attorney, au moins 2 witnesses de types differents, option expert, jury_*, assistant, clerk, nexus_faction_representative). Chaque avec actorType, name, personality, avatarAssetId (utilise tribunal.character.xxx)
- acts: au moins 1 pour niveau<4, 3+ pour >=4. Chaque act: actIndex, title, objective, summary
- scenes: 4 a 12 scenes par affaire. Chaque scene: sceneId (ex "act1_mira_testimony"), actIndex, sceneIndex, sceneType (intro|briefing|witness_testimony|investigation|cross_examination|crisis|reveal|final_plea|jury_vote|verdict|epilogue), title, objective, narrativeText, activeWitnessId (nullable), activeActorIds, availableEvidenceIds, visibleStatementIds, allowedActions (ex ["press","present_evidence","objection","ai_analysis","ask_hint","expose_lie"]), nextSceneId nullable
- progressionRules: au moins 1 par scene critique. TriggerAction + (requiredEvidenceId ou requiredStatementId ou requiredChoice), resultType, isCritical, unlock*, narrativeResult, scoreEffects (defenseScoreDelta, witnessCredibilityDelta, tribunalPressureDelta, accusationScoreDelta)
- failureRules: pour mauvaises actions (mauvaise preuve etc). penaltyType, maxFailuresBeforeHint, judgeWarningText, hintText, scoreEffects, stayOnScene:true
- crisisMoment (pour niveau >=6): sceneId, trigger, effect
- possibleVerdicts: ["defense_win","partial_defense","partial_guilty","guilty","hidden_truth"] au minimum
- epilogue
- nexusBridgeHints (surtout niveau 10): tableau de consequences proposees (type, target, delta)

Format STRICT:
{
  "cases": [
    {
      "title": "Le verrouillage d’ORA-7",
      "synopsis": "...",
      "level": 4,
      "difficulty": "intermediate",
      "estimatedDurationMinutes": 18,
      "caseType": "moral",
      "mode": "full_case",
      "tone": "cyberpunk_serious",
      "realTruth": "...",
      "publicTruth": "...",
      "finalReveal": "...",
      "cast": [ {"actorType":"judge","name":"Juge Veyra","personality":"calme, severe, rationnelle","avatarAssetId":"tribunal.character.judge_ai"} , ... ],
      "acts": [ {"actIndex":1,"title":"Le verrouillage","objective":"..."} ],
      "scenes": [ {"sceneId":"act1_intro","actIndex":1,"sceneIndex":0,"sceneType":"intro","title":"Ouverture","objective":"Comprendre l accusation","narrativeText":"...","activeWitnessId":null,"activeActorIds":["veyra"],"availableEvidenceIds":[],"visibleStatementIds":[],"allowedActions":["continue_story"],"nextSceneId":"act1_briefing"} , ... ],
      "progressionRules": [ {"sceneId":"act1_mira_testimony","triggerAction":"present_evidence","requiredEvidenceId":"biometric_log","requiredStatementId":"mira_s2","resultType":"minor_contradiction","isCritical":true,"unlockSceneId":"act2_admin_access","narrativeResult":"Le journal prouve intervention humaine.","scoreEffects":{"defenseScoreDelta":10,"witnessCredibilityDelta":-15}} ],
      "failureRules": [ {"sceneId":"...","triggerAction":"present_evidence","penaltyType":"score_down","maxFailuresBeforeHint":2,"judgeWarningText":"Preuve non pertinente. Reflechis.","hintText":"Cherche le log horodate.","scoreEffects":{"defenseScoreDelta":-4},"stayOnScene":true} ],
      "crisisMoment": {"sceneId":"act3_crisis","trigger":"3e objection ratee ou preuve cle manquante","effect":"Pression max, temoin s effondre"},
      "possibleVerdicts": ["defense_win","partial_defense","partial_guilty","guilty"],
      "epilogue": "Une nouvelle enquete s ouvre sur l admin responsable.",
      "nexusBridgeHints": [{"type":"guild_reputation_delta","targetId":"12","delta":-5},{"type":"create_lore_entry","visibility":"regional"}]
    }
    // exactement %d entrees, niveaux 1-10 uniques
  ]
}

Contraintes absolues: exactement %d; niveaux 1-10 croissants; tout en francais cyberpunk original; contradictions exploitables; pas de licence; assets avatarAssetId valides (tribunal.character.*); scenes jouables (pas descriptives seulement); au moins 1 progressionRule critique par affaire niveau>=3; crisis pour >=6; finalReveal pour >=5.`, count, count)
	return system, user
}
