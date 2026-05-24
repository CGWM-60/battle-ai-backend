package scenarios

import (
	"cgwm/battle/internal/provider"
	"context"
	"fmt"
)



const promptScenarioQueteRp=`
Tu es un créateur de scénarios de jeu de rôle expérimenté. 
Génère exactement 10 scénarios RPG complets, chacun avec un thème UNIQUE et radicalement différent des autres.

Pour CHAQUE scénario, fournis obligatoirement les informations suivantes dans cet ordre:

## SCÉNARIO [N°] - [TITRE ACCROCHEUR]

**THÈME:** [Mentionner le thème unique]

**UNIVERS:** [Décrire le monde, l'époque, la réalité. 2-3 phrases]

**TON:** [Fantasy épique / Horreur cosmic / Comédie / Noir détective / etc. 1 phrase]

**RÈGLES:** [Système utilisé: D&D 5e / Pathfinder / Appel de Cthulhu / Custom / etc. Règles spéciales si nécessaire]

**PERSONNAGES IMPORTANTS:**
- [Nom] - [Rôle et description courte]
- [Nom] - [Rôle et description courte]
- [Nom] - [Rôle et description courte]

**LIEUX CLÉS:**
- [Lieu 1] - [Description rapide]
- [Lieu 2] - [Description rapide]
- [Lieu 3] - [Description rapide]

**BUT PRINCIPAL:** [L'objectif global clair que les PJ doivent accomplir]

**GRANDS ARCS NARRATIFS (3-5):**
1. [Arc 1] - [Résumé 1-2 lignes]
2. [Arc 2] - [Résumé 1-2 lignes]
3. [Arc 3] - [Résumé 1-2 lignes]
4. [Arc 4 optionnel] - [Résumé 1-2 lignes]
5. [Arc 5 optionnel] - [Résumé 1-2 lignes]

**NIVEAU:** [Difficulté: Facile / Moyen / Difficile / Très Difficile]
**XP À LA FIN:** [Nombre exact de points d'expérience]
**RÉCOMPENSE EN OR:** [Nombre exact de pièces d'or]

---

CONTRAINTES IMPORTANTES:
✓ Chaque scénario DOIT avoir un thème radicalement différent (pas 2 fois du "fantasy classique")
✓ Les univers doivent être variés: Medieval, SF, Horreur, Modern, Urbain, Fantastique, Historique, Post-Apo, Steampunk, Autre
✓ Les tons doivent contraster: alternez entre épique, comique, sombre, mysterious, dramatique, etc.
✓ Sois créatif et immersif dans les descriptions
✓ Les récompenses (XP/Or) varient selon la difficulté
✓ Chaque arc doit être jouable et avoir des conséquences
✓ Les personnages importants doivent avoir des motivations claires

Génère ces 10 scénarios MAINTENANT, complets et détaillés:
`


func GenerateScenarioRp(){
			mistral := provider.NewsProvider(
			"6846mKmwFymVwu0XWafPZXRfqJ9beYga",
			"https://api.mistral.ai/v1/chat/completions",
			"mistral-large-latest",
		)

				messages := []provider.ProviderMessage{
			{
				Role:    "system",
				Content: promptScenarioQueteRp,
			},
			
		}

		response, err := mistral.Chat(context.Background(), messages)
		if err != nil {
			panic(err)
		}
		fmt.Print(response)

}