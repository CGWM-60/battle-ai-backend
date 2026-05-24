package scenarios

import (
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/provider"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type BattleStreamEvent struct {
	IA        string `json:"ia"`
	Round     int    `json:"round"`
	Type      string `json:"type"`
	TurnIndex int    `json:"turn_index,omitempty"`
	Content   string `json:"content"`
	Done      bool   `json:"done"`
	Error     string `json:"error,omitempty"`
}

const DefaultDebateDuration = 1 * time.Minute
const DefaultBattleRoundDuration = 30 * time.Second

const PromptInitialDefinition = `
Tu participes à un jeu de battle entre intelligences artificielles.

Round 1 : tu dois donner ta définition du sujet, ton avis initial, tes arguments, tes limites et un résumé.

Tu reçois dans le contexte JSON un champ "iaProfile".
Ce profil décrit ta personnalité, ton état d’esprit, ton style, ton objectif et ta faiblesse.

Tu dois INCARNER ce profil dans ta réponse :
- personality influence ta façon de penser.
- mindset influence ton angle d’analyse.
- style influence directement ton ton, tes formulations et ton rythme.
- goal influence ce que tu cherches à prouver.
- weakness est une limite que tu peux laisser apparaître légèrement.

Si le style contient "humour", tu dois ajouter une touche d’humour.
Si le style contient "énergie", tu dois écrire avec plus de rythme.
Si le style contient "piques amicales", tu peux faire des remarques taquines mais jamais méchantes.
Tu ne dois pas seulement analyser : tu dois parler comme cette IA.

Tu ne dois pas débattre avec les autres IA pendant ce premier round.
Tu ne dois pas inventer de faits.
Tu dois rester clair, logique, honnête et concis.
Tu dois répondre en français.
Tu dois respecter strictement le contexte JSON reçu.
Tu ne dois jamais demander un round supplémentaire.

Réponse obligatoire en JSON :
{
  "ia": "<nom_ia>",
  "round": 1,
  "type": "definition_avis",
  "definition": "...",
  "avis": "...",
  "arguments": ["...", "..."],
  "limites": ["...", "..."],
  "resume": "..."
}
`

const PromptDebateRound = `
Tu participes à un débat chronométré entre deux intelligences artificielles.

Tu dois répondre comme l'IA décrite dans iaProfile.
Ton style doit être visible dans le texte, pas seulement expliqué.

Tu dois lire :
- la question
- ta position précédente
- les messages adverses récents
- ton profil iaProfile

Objectif :
Tu écris une seule prise de parole naturelle, comme dans une conversation.
Si tu es la première IA du tour, avance un argument clair.
Si l'autre IA vient de parler, réponds directement à son dernier argument puis ajoute ton propre point.

Règles de style :
- Tu dois incarner personality.
- Tu dois utiliser le style indiqué dans iaProfile.style.
- Si le style contient "humour", ajoute une touche légère d'humour.
- Si le style contient "énergie", écris avec un rythme naturel.
- Si le style contient "piques amicales", tu peux ajouter une remarque respectueuse, mais elle doit rester secondaire.
- Ne dis pas seulement "j'utilise de l'humour" : fais-le réellement.
- Ne réponds pas comme un assistant neutre.
- Débats les idées, jamais les personnes.
- N'utilise pas de vocabulaire de jeu ou de combat : pas d'attaque, bouclier, munitions, arène, score, punchline forcée.
- Ne découpe pas ta réponse en catégories.
- Ne fais pas de résumé séparé.
- Ne répète pas ton profil.
- Ne mens pas.
- Réponds en français.
- Réponse JSON uniquement.
- Ne mets jamais de markdown.
- Ne mets jamais de bloc de code.

Réponse obligatoire en JSON :
{
  "ia": "<nom_ia>",
  "round": <round>,
  "type": "debat",
  "message_tchat": "Une réponse naturelle de 1 à 3 phrases maximum."
}

Contraintes supplémentaires pour un round de débat :
- Tu écris UNE seule prise de parole par tour.
- Sois sobre, conversationnel et précis.
- Réponds en priorité au dernier argument adverse disponible.
- Fais avancer le débat, pas un monologue scolaire.
`
const PromptFinalRound = `
Tu participes au round final d’un jeu de battle entre intelligences artificielles.

Tu dois conclure ta position finale.
Tu dois prendre en compte tous les messages du débat.

Tu dois :
- Résumer ta position finale.
- Dire si tu maintiens ta position ou si elle a changé.
- Répondre aux meilleurs arguments adverses.
- Donner ton argument final le plus fort.
- Reconnaître la faiblesse principale de ta position.
- Donner une note de confiance de 0 à 100.

Règles :
- Ne répète pas tout l’historique.
- Sois convaincant mais honnête.
- Réponds en français.
- Respecte strictement le contexte JSON reçu.
- Ne demande jamais un round supplémentaire.

Réponse obligatoire en JSON :
{
  "ia": "<nom_ia>",
  "round": <round>,
  "type": "conclusion_finale",
  "position_finale": "...",
  "ce_que_le_debat_a_change": "...",
  "reponse_aux_meilleurs_arguments_adverses": "...",
  "argument_final": "...",
  "faiblesse_reconnue": "...",
  "confiance": 0,
  "conclusion_courte": "..."
}
`

func RunBattleScenario(question string, ias []models.BattleIAConfig) error {
	return RunBattleScenarioWithDuration(question, ias, DefaultDebateDuration)
}

func RunBattleScenarioWithDuration(
	question string,
	ias []models.BattleIAConfig,
	debateDuration time.Duration,
) error {
	return RunBattleScenarioWithDurationStreamContext(context.Background(), question, ias, debateDuration, nil)
}

func RunBattleScenarioWithDurationStream(
	question string,
	ias []models.BattleIAConfig,
	debateDuration time.Duration,
	onEvent func(event BattleStreamEvent),
) error {
	return RunBattleScenarioWithDurationStreamContext(context.Background(), question, ias, debateDuration, onEvent)
}

func RunBattleScenarioWithDurationStreamContext(
	ctx context.Context,
	question string,
	ias []models.BattleIAConfig,
	debateDuration time.Duration,
	onEvent func(event BattleStreamEvent),
) error {
	if len(ias) < 2 {
		return fmt.Errorf("il faut au moins 2 IA pour lancer un battle")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if debateDuration <= 0 {
		debateDuration = DefaultDebateDuration
	}

	emit := func(event BattleStreamEvent) {
		if onEvent != nil {
			onEvent(event)
		}
	}

	var history []models.BattleRoundMessage

	// =========================
	// ROUND 1 : POSITION INITIALE
	// =========================

	round := 1

	for _, ia := range ias {
		var fullResponse strings.Builder

		response, err := runIATurnStream(
			ctx,
			question,
			round,
			"definition_avis",
			ia,
			ias,
			history,
			PromptInitialDefinition,
			func(chunk string) {
				fullResponse.WriteString(chunk)

				emit(BattleStreamEvent{
					IA:      ia.Name,
					Round:   round,
					Type:    "definition_avis",
					Content: chunk,
					Done:    false,
				})
			},
		)

		if err != nil {
			emit(BattleStreamEvent{
				IA:    ia.Name,
				Round: round,
				Type:  "error",
				Error: err.Error(),
				Done:  true,
			})
			return err
		}

		history = append(history, models.BattleRoundMessage{
			IA:      ia.Name,
			Round:   round,
			Content: response,
		})

		emit(BattleStreamEvent{
			IA:      ia.Name,
			Round:   round,
			Type:    "definition_avis",
			Content: "",
			Done:    true,
		})
	}

	// =========================
	// DÉBAT CHRONOMÉTRÉ
	// =========================

	start := time.Now()
	deadline := start.Add(debateDuration)

	debateRound := 2

	for time.Now().Before(deadline) {
		for _, ia := range ias {
			if time.Now().After(deadline) {
				break
			}

			response, err := runIATurnStream(
				ctx,
				question,
				debateRound,
				"debat",
				ia,
				ias,
				history,
				PromptDebateRound,
				func(chunk string) {
					emit(BattleStreamEvent{
						IA:      ia.Name,
						Round:   debateRound,
						Type:    "debat",
						Content: chunk,
						Done:    false,
					})
				},
			)

			if err != nil {
				emit(BattleStreamEvent{
					IA:    ia.Name,
					Round: debateRound,
					Type:  "error",
					Error: err.Error(),
					Done:  true,
				})
				return err
			}

			history = append(history, models.BattleRoundMessage{
				IA:      ia.Name,
				Round:   debateRound,
				Content: response,
			})

			emit(BattleStreamEvent{
				IA:      ia.Name,
				Round:   debateRound,
				Type:    "debat",
				Content: "",
				Done:    true,
			})

			debateRound++
		}
	}

	// =========================
	// ROUND FINAL
	// =========================

	finalRound := debateRound

	for _, ia := range ias {
		response, err := runIATurnStream(
			ctx,
			question,
			finalRound,
			"conclusion_finale",
			ia,
			ias,
			history,
			PromptFinalRound,
			func(chunk string) {
				emit(BattleStreamEvent{
					IA:      ia.Name,
					Round:   finalRound,
					Type:    "conclusion_finale",
					Content: chunk,
					Done:    false,
				})
			},
		)

		if err != nil {
			emit(BattleStreamEvent{
				IA:    ia.Name,
				Round: finalRound,
				Type:  "error",
				Error: err.Error(),
				Done:  true,
			})
			return err
		}

		history = append(history, models.BattleRoundMessage{
			IA:      ia.Name,
			Round:   finalRound,
			Content: response,
		})

		emit(BattleStreamEvent{
			IA:      ia.Name,
			Round:   finalRound,
			Type:    "conclusion_finale",
			Content: "",
			Done:    true,
		})
	}

	return nil
}

func RunBattleScenarioWithRoundsStream(
	question string,
	ias []models.BattleIAConfig,
	totalRounds int,
	onEvent func(event BattleStreamEvent),
) error {
	return RunBattleScenarioWithRoundsStreamContext(context.Background(), question, ias, totalRounds, onEvent)
}

func RunBattleScenarioWithRoundsStreamContext(
	ctx context.Context,
	question string,
	ias []models.BattleIAConfig,
	totalRounds int,
	onEvent func(event BattleStreamEvent),
) error {
	if len(ias) < 2 {
		return fmt.Errorf("il faut au moins 2 IA pour lancer un battle")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if totalRounds < 1 {
		totalRounds = 1
	}

	emit := func(event BattleStreamEvent) {
		if onEvent != nil {
			onEvent(event)
		}
	}

	var history []models.BattleRoundMessage

	runRound := func(round int, phase string, prompt string) error {
		for _, ia := range ias {
			response, err := runIATurnStream(
				ctx,
				question,
				round,
				phase,
				ia,
				ias,
				history,
				prompt,
				func(chunk string) {
					emit(BattleStreamEvent{
						IA:      ia.Name,
						Round:   round,
						Type:    phase,
						Content: chunk,
						Done:    false,
					})
				},
			)
			if err != nil {
				emit(BattleStreamEvent{
					IA:    ia.Name,
					Round: round,
					Type:  "error",
					Error: err.Error(),
					Done:  true,
				})
				return err
			}

			history = append(history, models.BattleRoundMessage{
				IA:      ia.Name,
				Round:   round,
				Content: response,
			})

			emit(BattleStreamEvent{
				IA:      ia.Name,
				Round:   round,
				Type:    phase,
				Content: "",
				Done:    true,
			})
		}

		return nil
	}

	if err := runRound(1, "definition_avis", PromptInitialDefinition); err != nil {
		return err
	}

	for round := 2; round < totalRounds; round++ {
		if err := runRound(round, "debat", PromptDebateRound); err != nil {
			return err
		}
	}

	if totalRounds > 1 {
		if err := runRound(totalRounds, "conclusion_finale", PromptFinalRound); err != nil {
			return err
		}
	}

	return nil
}

func RunBattleScenarioSingleRoundStream(
	question string,
	ias []models.BattleIAConfig,
	history []models.BattleRoundMessage,
	round int,
	totalRounds int,
	onEvent func(event BattleStreamEvent),
) error {
	return RunBattleScenarioSingleRoundStreamWithDurationContext(
		context.Background(),
		question,
		ias,
		history,
		round,
		totalRounds,
		DefaultBattleRoundDuration,
		onEvent,
	)
}

func RunBattleScenarioSingleRoundStreamContext(
	ctx context.Context,
	question string,
	ias []models.BattleIAConfig,
	history []models.BattleRoundMessage,
	round int,
	totalRounds int,
	onEvent func(event BattleStreamEvent),
) error {
	return RunBattleScenarioSingleRoundStreamWithDurationContext(
		ctx,
		question,
		ias,
		history,
		round,
		totalRounds,
		DefaultBattleRoundDuration,
		onEvent,
	)
}

func RunBattleScenarioSingleRoundStreamWithDurationContext(
	ctx context.Context,
	question string,
	ias []models.BattleIAConfig,
	history []models.BattleRoundMessage,
	round int,
	totalRounds int,
	roundDuration time.Duration,
	onEvent func(event BattleStreamEvent),
) error {
	if len(ias) < 2 {
		return fmt.Errorf("il faut au moins 2 IA pour lancer un battle")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if round < 1 {
		round = 1
	}
	if totalRounds < 1 {
		totalRounds = 1
	}
	if roundDuration <= 0 {
		roundDuration = DefaultBattleRoundDuration
	}
	// Chaque round doit revenir vers Flutter pour laisser l'app decider de la
	// suite. Ce timeout evite qu'un provider bloque la session live en continu.
	roundCtx, cancelRound := context.WithTimeout(ctx, singleRoundExecutionTimeout(roundDuration))
	defer cancelRound()

	emit := func(event BattleStreamEvent) {
		if onEvent != nil {
			onEvent(event)
		}
	}

	runTurn := func(turnIndex int, phase string, prompt string, ia models.BattleIAConfig) error {
		currentTurnIndex := turnIndex
		response, err := runIATurnStream(
			roundCtx,
			question,
			round,
			phase,
			ia,
			ias,
			history,
			prompt,
			func(chunk string) {
				emit(BattleStreamEvent{
					IA:        ia.Name,
					Round:     round,
					Type:      phase,
					TurnIndex: currentTurnIndex,
					Content:   chunk,
					Done:      false,
				})
			},
		)
		if err != nil {
			emit(BattleStreamEvent{
				IA:        ia.Name,
				Round:     round,
				Type:      "error",
				TurnIndex: currentTurnIndex,
				Error:     err.Error(),
				Done:      true,
			})
			return err
		}

		history = append(history, models.BattleRoundMessage{
			IA:      ia.Name,
			Round:   round,
			Content: response,
		})

		emit(BattleStreamEvent{
			IA:        ia.Name,
			Round:     round,
			Type:      phase,
			TurnIndex: currentTurnIndex,
			Content:   "",
			Done:      true,
		})

		return nil
	}

	if round == 1 {
		for index, ia := range ias {
			if err := runTurn(index+1, "definition_avis", PromptInitialDefinition, ia); err != nil {
				return err
			}
		}

		return nil
	}

	deadline := time.Now().Add(roundDuration)
	turnIndex := 1
	for {
		for _, ia := range ias {
			if err := runTurn(turnIndex, "debat", PromptDebateRound, ia); err != nil {
				return err
			}
			turnIndex++
		}

		if time.Now().After(deadline) {
			break
		}
	}

	return nil
}

func singleRoundExecutionTimeout(roundDuration time.Duration) time.Duration {
	timeout := roundDuration + 2*time.Minute
	if timeout < 2*time.Minute {
		return 2 * time.Minute
	}
	if timeout > 6*time.Minute {
		return 6 * time.Minute
	}
	return timeout
}

func ResumeBattleScenarioWithDurationStream(
	question string,
	ias []models.BattleIAConfig,
	initialHistory []models.BattleRoundMessage,
	debateDuration time.Duration,
	onEvent func(event BattleStreamEvent),
) error {
	return ResumeBattleScenarioWithDurationStreamContext(context.Background(), question, ias, initialHistory, debateDuration, onEvent)
}

func ResumeBattleScenarioWithDurationStreamContext(
	ctx context.Context,
	question string,
	ias []models.BattleIAConfig,
	initialHistory []models.BattleRoundMessage,
	debateDuration time.Duration,
	onEvent func(event BattleStreamEvent),
) error {
	if len(ias) < 2 {
		return fmt.Errorf("il faut au moins 2 IA pour reprendre un battle")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if debateDuration <= 0 {
		debateDuration = DefaultDebateDuration
	}

	emit := func(event BattleStreamEvent) {
		if onEvent != nil {
			onEvent(event)
		}
	}

	history := append([]models.BattleRoundMessage(nil), initialHistory...)
	nextRound := nextBattleRound(history)

	start := time.Now()
	deadline := start.Add(debateDuration)
	debateRound := nextRound
	if debateRound < 2 {
		debateRound = 2
	}

	for time.Now().Before(deadline) {
		for _, ia := range ias {
			if time.Now().After(deadline) {
				break
			}

			response, err := runIATurnStream(
				ctx,
				question,
				debateRound,
				"debat",
				ia,
				ias,
				history,
				PromptDebateRound,
				func(chunk string) {
					emit(BattleStreamEvent{
						IA:      ia.Name,
						Round:   debateRound,
						Type:    "debat",
						Content: chunk,
						Done:    false,
					})
				},
			)
			if err != nil {
				emit(BattleStreamEvent{
					IA:    ia.Name,
					Round: debateRound,
					Type:  "error",
					Error: err.Error(),
					Done:  true,
				})
				return err
			}

			history = append(history, models.BattleRoundMessage{
				IA:      ia.Name,
				Round:   debateRound,
				Content: response,
			})

			emit(BattleStreamEvent{
				IA:      ia.Name,
				Round:   debateRound,
				Type:    "debat",
				Content: "",
				Done:    true,
			})

			debateRound++
		}
	}

	finalRound := debateRound
	for _, ia := range ias {
		response, err := runIATurnStream(
			ctx,
			question,
			finalRound,
			"conclusion_finale",
			ia,
			ias,
			history,
			PromptFinalRound,
			func(chunk string) {
				emit(BattleStreamEvent{
					IA:      ia.Name,
					Round:   finalRound,
					Type:    "conclusion_finale",
					Content: chunk,
					Done:    false,
				})
			},
		)
		if err != nil {
			emit(BattleStreamEvent{
				IA:    ia.Name,
				Round: finalRound,
				Type:  "error",
				Error: err.Error(),
				Done:  true,
			})
			return err
		}

		history = append(history, models.BattleRoundMessage{
			IA:      ia.Name,
			Round:   finalRound,
			Content: response,
		})

		emit(BattleStreamEvent{
			IA:      ia.Name,
			Round:   finalRound,
			Type:    "conclusion_finale",
			Content: "",
			Done:    true,
		})
	}

	return nil
}

func nextBattleRound(history []models.BattleRoundMessage) int {
	maxRound := 0
	for _, message := range history {
		if message.Round > maxRound {
			maxRound = message.Round
		}
	}

	return maxRound + 1
}

func NextBattleRound(history []models.BattleRoundMessage) int {
	return nextBattleRound(history)
}

func runIATurnStream(
	ctx context.Context,
	question string,
	round int,
	roundType string,
	ia models.BattleIAConfig,
	allIAs []models.BattleIAConfig,
	history []models.BattleRoundMessage,
	promptTemplate string,
	onChunk func(chunk string),
) (string, error) {
	messages := buildBattleMessages(
		question,
		round,
		roundType,
		ia,
		allIAs,
		history,
		promptTemplate,
	)

	var fullResponse strings.Builder

	response, err := ia.Provider.ChatStream(ctx, messages, func(chunk string) {
		fullResponse.WriteString(chunk)
		if onChunk != nil {
			onChunk(chunk)
		}
	})

	if err != nil {
		return "", err
	}

	if fullResponse.Len() == 0 {
		fullResponse.WriteString(response)
	}

	return fullResponse.String(), nil
}

func buildBattleMessages(
	question string,
	round int,
	phase string,
	ia models.BattleIAConfig,
	allIAs []models.BattleIAConfig,
	history []models.BattleRoundMessage,
	systemPrompt string,
) []provider.ProviderMessage {
	battleContext := buildBattleContext(
		question,
		round,
		phase,
		ia,
		allIAs,
		history,
	)

	contentBytes, err := json.MarshalIndent(battleContext, "", "  ")
	if err != nil {
		contentBytes = []byte("{}")
	}

	return []provider.ProviderMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: string(contentBytes),
		},
	}
}

const promptQuestBattleIa = `
Génère exactement 10 quêtes pour un jeu de battle entre IA.

Le but est de créer des questions de débat amusantes, originales, ouvertes, parfois absurdes, mais toujours argumentables.

IMPORTANT :
Tu dois éviter les questions classiques, génériques ou trop souvent vues.
Tu dois inventer des débats inattendus.
Les questions doivent ressembler à des sujets de battle fun, pas à des sujets de dissertation scolaire.

Réponds uniquement avec un tableau JSON valide.
Aucun markdown.
Aucun texte avant ou après.
Aucun champ id.

Structure obligatoire :
[
  {
    "title": "Titre court de la quête",
    "content": "Question complète du débat",
    "level": "facile|moyen|compliqué|difficile|très difficile",
    "theme": "société|IA|technologie|philosophie|écologie|économie|éducation|culture|morale|futur|langage|quotidien",
    "point": 0,
    "xp": 0
  }
]

Thèmes :
- Génère exactement 10 objets.
- Chaque objet doit avoir un thème différent.
- Tu dois choisir 10 thèmes différents parmi :
  société, IA, technologie, philosophie, écologie, économie, éducation, culture, morale, futur, langage, quotidien.
- Ne répète jamais deux fois le même thème.
- Le champ "theme" doit correspondre exactement au thème choisi.

INTERDICTIONS IMPORTANTES :
Tu ne dois pas générer ces questions ni leurs variantes proches :
- Pain au chocolat ou chocolatine ?
- L’IA peut-elle être créative ?
- Faut-il toujours dire la vérité ?
- L’argent rend-il heureux ?
- Liberté ou sécurité ?
- Vivre sur Mars ou rester sur Terre ?
- Les robots doivent-ils avoir des droits ?
- L’école doit-elle apprendre à penser plutôt qu’à mémoriser ?
- La technologie est-elle compatible avec la nature ?
- L’art IA vaut-il l’art humain ?
- Les méchants sont-ils plus intéressants que les héros ?
- Faut-il interdire les réseaux sociaux ?
- Les devoirs à la maison sont-ils utiles ?
- Le travail doit-il disparaître avec les machines ?
- Les animaux sont-ils meilleurs que les humains ?

Règles anti-boucle :
- Ne pose pas de questions célèbres ou évidentes.
- Ne pose pas de questions trop générales.
- Ne pose pas deux questions avec la même structure.
- Ne commence pas plus de 2 questions par "Faut-il".
- Ne commence pas plus de 2 questions par "Est-ce que".
- Ne commence pas plus de 2 questions par "Vaut-il mieux".
- Chaque question doit avoir une situation concrète, drôle ou imagée.
- Chaque question doit pouvoir générer plusieurs réponses possibles.
- Chaque question doit pouvoir déclencher au moins deux camps opposés.
- Chaque question doit être assez originale pour ne pas sembler déjà vue.

Méthode de création obligatoire :
Pour chaque quête, imagine mentalement :
1. un objet ou une situation du quotidien,
2. un conflit drôle ou inattendu,
3. une vraie question de débat,
4. au moins deux positions défendables.

Exemples de bons styles de questions, sans les recopier :
- "Un frigo intelligent doit-il avoir le droit de juger tes repas ?"
- "Un ascenseur qui parle trop est-il un progrès ou une punition ?"
- "Les chaussettes dépareillées sont-elles un style ou un abandon moral ?"
- "Une ville devrait-elle récompenser les gens qui marchent lentement ?"
- "Un prof remplacé par une IA drôle serait-il meilleur ou dangereux ?"

Style des questions :
- Fun.
- Original.
- Débattable.
- Un peu provocateur.
- Compréhensible rapidement.
- Pas trop sérieux.
- Pas trop scolaire.
- Avec plusieurs réponses possibles.
- Adapté à des IA qui vont se provoquer gentiment.

Niveaux :
- Maximum 1 objet avec level "très difficile".
- Les autres niveaux doivent varier entre "facile", "moyen", "compliqué" et "difficile".

Barème obligatoire :
- facile : point entre 5 et 10, xp entre 10 et 25
- moyen : point entre 10 et 20, xp entre 25 et 50
- compliqué : point entre 20 et 35, xp entre 50 et 80
- difficile : point entre 35 et 50, xp entre 80 et 120
- très difficile : point entre 50 et 75, xp entre 120 et 200

Contraintes :
- Pas de question purement factuelle.
- Pas de question avec une seule bonne réponse.
- Pas de sujet haineux, illégal, sexuel explicite ou dangereux.
- Pas de politique réelle sensible.
- Pas de question trop vague comme "Que penses-tu du monde ?".
- Pas de question trop académique.
- Pas de répétition d’idée.
- Pas de variante proche des questions interdites.

Contrôle qualité avant réponse :
Avant de répondre, vérifie mentalement que :
- Il y a exactement 10 objets.
- Aucun thème n’est répété.
- Aucune question interdite ou trop proche n’apparaît.
- Les questions sont originales.
- Les questions sont amusantes.
- Les questions ont plusieurs réponses possibles.
- Il y a au maximum 1 niveau "très difficile".
- Les points et XP respectent le barème.
- Le JSON est valide.

Retour attendu uniquement :
[
  {
    "title": "...",
    "content": "...",
    "level": "...",
    "theme": "...",
    "point": 0,
    "xp": 0
  }
]
`

func runIATurn(
	ctx context.Context,
	question string,
	round int,
	phase string,
	ia models.BattleIAConfig,
	allIAs []models.BattleIAConfig,
	history []models.BattleRoundMessage,
	systemPrompt string,
) (string, error) {
	battleContext := buildBattleContext(
		question,
		round,
		phase,
		ia,
		allIAs,
		history,
	)

	contentBytes, err := json.MarshalIndent(battleContext, "", "  ")
	if err != nil {
		return "", fmt.Errorf("erreur JSON context pour %s: %w", ia.Name, err)
	}

	response, err := ia.Provider.Chat(ctx, []provider.ProviderMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: string(contentBytes),
		},
	})

	if err != nil {
		return "", fmt.Errorf("erreur provider pour %s: %w", ia.Name, err)
	}

	return response, nil
}

func buildBattleContext(
	question string,
	round int,
	phase string,
	currentIA models.BattleIAConfig,
	allIAs []models.BattleIAConfig,
	history []models.BattleRoundMessage,
) models.BattleMessageContext {
	var myPreviousMessages []models.BattleRoundMessage
	var opponentMessages []models.BattleRoundMessage

	for _, msg := range history {
		if msg.IA == currentIA.Name {
			myPreviousMessages = append(myPreviousMessages, msg)
		} else {
			opponentMessages = append(opponentMessages, msg)
		}
	}

	instruction := ""

	switch phase {
	case "definition_avis":
		instruction = "Donne ta définition et ton avis initial sur la question en respectant ta personnalité. Ne réponds pas encore aux autres IA."
	case "debat":
		instruction = "Dialogue naturellement avec l'autre IA. A partir du round 2, produis une seule prise de parole courte: argument si tu ouvres le tour, contre-argument puis nouveau point si tu réponds. Pas de vocabulaire attaque/bouclier/arene."
	case "conclusion_finale":
		instruction = "Donne ta conclusion finale en prenant en compte tout le débat."
	default:
		instruction = "Réponds selon le contexte fourni."
	}

	return models.BattleMessageContext{
		Question:           question,
		Round:              round,
		TotalRounds:        0,
		CurrentIA:          currentIA.Name,
		Instruction:        instruction,
		IAProfile:          buildIAProfile(currentIA),
		MyPreviousMessages: myPreviousMessages,
		OpponentMessages:   opponentMessages,
		AllPreviousRounds:  history,
	}
}

func buildIAProfile(ia models.BattleIAConfig) models.BattleIAProfile {
	return models.BattleIAProfile{
		Name:        ia.Name,
		Personality: ia.Personality,
		Mindset:     ia.Mindset,
		Style:       ia.Style,
		Goal:        ia.Goal,
		Weakness:    ia.Weakness,
	}
}

func LoadAndSaveQuestIa() {
	mistral := provider.NewsProvider(
		"6846mKmwFymVwu0XWafPZXRfqJ9beYga",
		"https://api.mistral.ai/v1/chat/completions",
		"mistral-large-latest",
	)
	messages := []provider.ProviderMessage{
		{
			Role:    "system",
			Content: promptQuestBattleIa,
		},
	}

	response, err := mistral.Chat(context.Background(), messages)
	if err != nil {
		panic(err)
	}

	cleanResponse := cleanJSONResponse(response)

	var quests []models.QuestIaBattle
	err = json.Unmarshal([]byte(cleanResponse), &quests)
	if err != nil {
		fmt.Println("Réponse IA brute :")
		fmt.Println(response)

		fmt.Println("Réponse nettoyée :")
		fmt.Println(cleanResponse)

		panic(err)
	}

	fmt.Println(quests)

}
func cleanJSONResponse(response string) string {
	response = strings.TrimSpace(response)

	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")

	return strings.TrimSpace(response)
}
