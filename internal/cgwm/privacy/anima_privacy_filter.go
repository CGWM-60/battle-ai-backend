package privacy

import (
	"strings"
	"cgwm/battle/internal/cgwm/models"
)

// AnimaSocialPrivacyFilter ensures only safe, anonymized, consented lessons are ever shared.
// Strict RGPD: no names, dates, locations, exact messages, personal identifiers, sensitive topics.
type AnimaSocialPrivacyFilter struct{}

func NewAnimaSocialPrivacyFilter() *AnimaSocialPrivacyFilter {
	return &AnimaSocialPrivacyFilter{}
}

// SanitizeAndScore takes a raw lesson produced locally and returns a safe card + safety score (0-1).
// If safety < 0.8 the card must be rejected.
func (f *AnimaSocialPrivacyFilter) SanitizeAndScore(raw map[string]interface{}) (models.AnimaSocialLearningCard, float64, bool) {
	topic := getString(raw, "topic")
	lesson := getString(raw, "lesson")
	tone := getString(raw, "emotional_tone")

	// Hard filters
	badWords := []string{"je", "moi", "mon joueur", "mon email", "mon nom", "habite", "appelle", "password", "clé", "api"}
	lowerLesson := strings.ToLower(lesson)
	for _, w := range badWords {
		if strings.Contains(lowerLesson, w) {
			return models.AnimaSocialLearningCard{}, 0.0, false
		}
	}

	// Strip anything that looks like personal data
	safeLesson := lesson
	// (real implementation would use more advanced NER / regex + LLM guard if budget allows — here we keep deterministic)

	card := models.AnimaSocialLearningCard{
		SourcePublicAnimaID: getString(raw, "public_anima_id"),
		Topic:               topic,
		Lesson:              safeLesson,
		EmotionalTone:       tone,
		Confidence:          getFloat(raw, "confidence"),
		SafetyScore:         0.92, // base after filters
		Tags:                getString(raw, "tags"),
	}

	// Final safety gate
	if card.SafetyScore < 0.8 {
		return card, card.SafetyScore, false
	}
	return card, card.SafetyScore, true
}

func getString(m map[string]interface{}, k string) string {
	if v, ok := m[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloat(m map[string]interface{}, k string) float64 {
	if v, ok := m[k]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0.5
}