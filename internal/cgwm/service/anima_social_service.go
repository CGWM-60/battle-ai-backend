package service

import (
	"cgwm/battle/internal/cgwm/models"
	"cgwm/battle/internal/cgwm/privacy"
)

// AnimaSocialLearningService handles safe card submission and simulated encounters.
var privacyFilter = privacy.NewAnimaSocialPrivacyFilter()

func SubmitSocialCard(raw map[string]interface{}) (models.AnimaSocialLearningCard, bool) {
	card, _, ok := privacyFilter.SanitizeAndScore(raw)
	if !ok {
		return card, false
	}
	// persist card (repository)
	return card, true
}

func SimulateEncounterForAloneAnima(alicePublic, bobPublic string) *models.AnimaSocialEncounter {
	// pick safe cards, create encounter, store result
	return &models.AnimaSocialEncounter{
		SourcePublicAnimaID: alicePublic,
		TargetPublicAnimaID: bobPublic,
		EncounterType:       "learning",
	}
}

// RunAloneParkLearningTick is called by the scheduler for alone Animas.
func RunAloneParkLearningTick() error {
	// In full impl: query alone Animas with consent, simulate encounters via SimulateEncounterForAloneAnima, persist.
	// For now a no-op stub so it compiles.
	return nil
}