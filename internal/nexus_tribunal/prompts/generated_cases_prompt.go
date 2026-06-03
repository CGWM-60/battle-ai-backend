package prompts

import "fmt"

// BuildGeneratedCasesPrompt returns the system + user prompt to generate exactly 10 Tribunal cases
// with increasing difficulty per the spec (level 1-10).
// Output must be strict JSON: {"cases": [ ... ]}
func BuildGeneratedCasesPrompt() (system string, user string) {
	system = "Tu es un generateur d'affaires de Tribunal IA cyberpunk original. Tu produis des cas jouables, coherents, avec contradictions exploitables. Reponds UNIQUEMENT en JSON valide strict (pas de markdown, pas de texte hors JSON)."

	user = fmt.Sprintf(`Genere exactement 10 affaires Tribunal IA avec les niveaux 1 a 10 decrits ci-dessous.
Chaque affaire doit etre complete, coherente, et contenir les champs demandes.
Niveaux (respecte les specs de complexite):
1. Initiation: 1 temoin, 2 preuves, contradiction simple, 5-8min
2. Facile: 1 temoin, 3 preuves, 1 contradiction claire, 8-10min
3. Standard: 2 temoins, 3 preuves, 2 contradictions, 10-15min
4. Intermediaire: 2 temoins, 4 preuves, temoignage partiellement fiable, 15-20min
5. Confirme: 2-3 temoins, 5 preuves, contradictions croisees, 20-25min
6. Difficile: 3 temoins, 5 preuves, une fausse piste, 25-30min
7. Expert: 3 temoins, 6 preuves, contradictions indirectes, 30-35min
8. Maitre: 4 temoins, 7 preuves, plusieurs issues possibles, 35-40min
9. Legendaire: 4 temoins, 8 preuves, conflit moral complexe, verdict nuance, 40-50min
10. Nexus: 5 temoins, 10 preuves, lien possible avec Nexus Games, consequences Living Lore possibles, 50-60min

Reponds STRICTEMENT au format:
{
  "cases": [
    {
      "title": "titre court original",
      "summary": "resume 1-2 phrases",
      "caseType": "moral|political|guild_conflict|player_conflict|world_event|quest_consequence|roleplay|absurd|custom",
      "level": 1,
      "difficulty": "initiation|easy|standard|intermediate|confirmed|hard|expert|master|legendary|nexus",
      "estimatedDurationMinutes": 7,
      "mode": "quick_trial|full_case",
      "tone": "cyberpunk_serious|cyberpunk_ironic|dark|neutral",
      "playerRoleSuggestion": "defense|prosecutor|neutral",
      "accusationPosition": "phrase accusation",
      "defensePosition": "phrase defense",
      "tags": ["ia","ville"],
      "witnesses": [{"name":"...","role":"...","credibility":65,"bias":"neutral","personality":"...","knowledge":"..."}],
      "evidence": [{"title":"...","description":"...","evidenceType":"document|surveillance_log|...","strength":60,"reliability":70,"tags":["..."],"supportsSide":"defense|accusation|neutral"}],
      "testimonyStatements": [{"witnessName":"...","content":"phrase attackable courte","tags":["..."],"isAttackable":true}],
      "expectedContradictions": [{"statementContent":"...","evidenceTitle":"...","contradictionType":"time|fact|..."}]
    }
    // ... exactement 10 entrees, level 1 a 10 distincts
  ]
}

Contraintes absolues: exactement 10; niveaux uniques 1-10; tout en francais cyberpunk; titres/summaries courts; contradictions exploitables dans les phrases vs preuves; pas de licence existante, pas de nom connu, pas de contenu interdit.`)
	return system, user
}
