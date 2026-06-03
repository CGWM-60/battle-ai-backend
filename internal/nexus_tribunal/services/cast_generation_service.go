package services

import "cgwm/battle/internal/nexus_tribunal/models"

// CastGenerationService generates varied cast for narrative cases.
type CastGenerationService struct{}

func NewCastGenerationService() *CastGenerationService { return &CastGenerationService{} }

func (s *CastGenerationService) GenerateCastForCase(level int, caseType string) []models.TribunalGeneratedActor {
	// Stub returns minimal valid cast; real uses IA or templates.
	return []models.TribunalGeneratedActor{
		{ActorType: "judge", Name: "Juge Veyra", Personality: "calme, severe", AvatarAssetID: "tribunal.character.judge_ai"},
		{ActorType: "prosecutor", Name: "Kael Orthis", Personality: "methodique", AvatarAssetID: "tribunal.character.prosecutor_ai"},
		{ActorType: "defense_attorney", Name: "Nara Silex", Personality: "strategique", AvatarAssetID: "tribunal.character.defense_ai"},
		{ActorType: "witness", Name: "Mira", Personality: "nerveuse", AvatarAssetID: "tribunal.character.witness_default"},
	}
}

func (s *CastGenerationService) GenerateJudge() models.TribunalGeneratedActor { return models.TribunalGeneratedActor{ActorType: "judge"} }
func (s *CastGenerationService) GenerateDefenseAttorney() models.TribunalGeneratedActor { return models.TribunalGeneratedActor{ActorType: "defense_attorney"} }
func (s *CastGenerationService) GenerateProsecutor() models.TribunalGeneratedActor { return models.TribunalGeneratedActor{ActorType: "prosecutor"} }
func (s *CastGenerationService) GenerateWitnesses(count int) []models.TribunalGeneratedActor { return nil }
func (s *CastGenerationService) GenerateExpertWitnesses() []models.TribunalGeneratedActor { return nil }
func (s *CastGenerationService) AssignAvatarAssets(actors []models.TribunalGeneratedActor) {}
func (s *CastGenerationService) ValidateCastVariety(cast []models.TribunalGeneratedActor) bool { return true }
