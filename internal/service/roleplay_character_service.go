package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cgwm/battle/internal/models"
	"cgwm/battle/internal/provider"
	"cgwm/battle/internal/repository"

	"gorm.io/datatypes"
)

type RolePlayCharacterInput struct {
	Name         string         `json:"name"`
	Class        string         `json:"class"`
	Archetype    string         `json:"archetype"`
	Origin       string         `json:"origin"`
	Race         string         `json:"race"`
	Species      string         `json:"species"`
	Alignment    string         `json:"alignment"`
	Personality  string         `json:"personality"`
	Background   string         `json:"background"`
	PersonalGoal string         `json:"personal_goal"`
	Goal         string         `json:"goal"`
	Level        int            `json:"level"`
	HeroImageID  *uint          `json:"heroImageId"`
	ImageURL     string         `json:"imageUrl"`
	Attributes   map[string]int `json:"attributes"`
	Skills       map[string]int `json:"skills"`
	Traits       []string       `json:"traits"`
	Inventory    []string       `json:"inventory"`
	Health       int            `json:"health"`
	MaxHealth    int            `json:"max_health"`
	Stress       int            `json:"stress"`
	Fatigue      int            `json:"fatigue"`
	Morale       int            `json:"morale"`
}

type CharacterGenerationInput struct {
	ProviderName string
	ModelName    string
	APIKey       string
	PlayerPrompt string
	QuestContext string
	Mode         string
	QuestID      uint
	CoopPartyID  uint
	LocalLLM     bool
}

type RolePlayCharacterService struct {
	characters *repository.RolePlayCharacterRepository
	quests     *repository.QuestRepository
}

func NewRolePlayCharacterService(characters *repository.RolePlayCharacterRepository, quests *repository.QuestRepository) *RolePlayCharacterService {
	return &RolePlayCharacterService{characters: characters, quests: quests}
}

func (s *RolePlayCharacterService) List(ctx context.Context, userID uint, limit int) ([]models.RolePlayCharacter, error) {
	return s.characters.ListByUser(ctx, userID, limit)
}

func (s *RolePlayCharacterService) Get(ctx context.Context, userID uint, id uint) (*models.RolePlayCharacter, error) {
	return s.characters.GetOwnedByID(ctx, id, userID)
}

func (s *RolePlayCharacterService) Create(ctx context.Context, userID uint, input RolePlayCharacterInput) (*models.RolePlayCharacter, error) {
	character, err := normalizeRolePlayCharacter(userID, input)
	if err != nil {
		return nil, err
	}
	if err := s.characters.Create(ctx, character); err != nil {
		return nil, err
	}
	return character, nil
}

func (s *RolePlayCharacterService) Update(ctx context.Context, userID uint, id uint, input RolePlayCharacterInput) (*models.RolePlayCharacter, error) {
	character, err := normalizeRolePlayCharacter(userID, input)
	if err != nil {
		return nil, err
	}
	fields := map[string]any{
		"name":          character.Name,
		"class":         character.Class,
		"origin":        character.Origin,
		"race":          character.Race,
		"alignment":     character.Alignment,
		"personality":   character.Personality,
		"background":    character.Background,
		"personal_goal": character.PersonalGoal,
		"level":         character.Level,
		"hero_image_id": character.HeroImageID,
		"image_url":     character.ImageURL,
		"attributes":    character.Attributes,
		"skills":        character.Skills,
		"traits":        character.Traits,
		"inventory":     character.Inventory,
		"health":        character.Health,
		"max_health":    character.MaxHealth,
		"stress":        character.Stress,
		"fatigue":       character.Fatigue,
		"morale":        character.Morale,
	}
	if err := s.characters.Update(ctx, id, userID, fields); err != nil {
		return nil, err
	}
	return s.characters.GetOwnedByID(ctx, id, userID)
}

func (s *RolePlayCharacterService) Delete(ctx context.Context, userID uint, id uint) error {
	return s.characters.Delete(ctx, id, userID)
}

func (s *RolePlayCharacterService) ValidateDraft(userID uint, input RolePlayCharacterInput) (*models.RolePlayCharacter, error) {
	return normalizeRolePlayCharacter(userID, input)
}

func (s *RolePlayCharacterService) PrepareGenerationPrompt(ctx context.Context, input CharacterGenerationInput) (string, map[string]any) {
	contextText := strings.TrimSpace(input.QuestContext)
	if contextText == "" && input.QuestID > 0 && s.quests != nil {
		if quest, err := s.quests.GetRolePlayQuestByID(ctx, input.QuestID); err == nil && quest != nil {
			contextText = strings.TrimSpace(fmt.Sprintf("%s\n%s\n%s", quest.Title, quest.Summary, quest.Prompt))
		}
	}
	if contextText == "" {
		contextText = "Univers non specifie. Cree un personnage jouable et adaptable a une aventure narrative."
	}
	playerPrompt := strings.TrimSpace(input.PlayerPrompt)
	if playerPrompt == "" {
		playerPrompt = "Le joueur n'a pas donne de contrainte supplementaire."
	}

	prompt := fmt.Sprintf(`Tu es un generateur de fiche personnage pour un jeu de role narratif. Genere un personnage jouable, equilibre et coherent avec le contexte fourni.
Tu dois repondre uniquement en JSON valide selon le schema demande. Ne mets aucun texte avant ou apres le JSON.
Le personnage doit etre niveau 1, sans objet surpuissant, avec des attributs equilibres entre 8 et 16, maximum un attribut a 16, competences entre 0 et 3, inventaire de 2 a 4 objets simples, 2 a 4 traits de personnalite, un background court et un objectif personnel exploitable dans une quete RP.

Mode de lancement: %s
Contexte:
%s

Demande joueur:
%s

Schema JSON attendu:
{
  "name": "...",
  "class": "...",
  "origin": "...",
  "race": "...",
  "alignment": "...",
  "personality": "...",
  "background": "...",
  "personal_goal": "...",
  "level": 1,
  "attributes": {
    "force": 10,
    "agility": 14,
    "constitution": 12,
    "intelligence": 13,
    "perception": 15,
    "charisma": 11
  },
  "skills": {
    "persuasion": 2,
    "stealth": 3,
    "combat": 1,
    "observation": 2,
    "survival": 1,
    "investigation": 2,
    "technology": 3,
    "intimidation": 0
  },
  "traits": ["Mefiant", "Rapide", "Loyal"],
  "inventory": ["Lame pliable", "Scanner de poche"],
  "health": 100,
  "max_health": 100,
  "stress": 0,
  "fatigue": 0,
  "morale": 50
}`, defaultString(input.Mode, "rp_quest"), truncateText(contextText, 4000), truncateText(playerPrompt, 1200))

	schema := map[string]any{
		"required":       []string{"name", "class", "origin", "race", "background", "personal_goal", "level", "attributes", "skills", "traits", "inventory", "health", "max_health", "stress", "fatigue", "morale"},
		"attributeRange": map[string]int{"min": 8, "max": 16},
		"skillRange":     map[string]int{"min": 0, "max": 3},
		"inventoryCount": map[string]int{"min": 2, "max": 4},
		"traitsCount":    map[string]int{"min": 2, "max": 4},
	}
	return prompt, schema
}

func (s *RolePlayCharacterService) Generate(ctx context.Context, userID uint, input CharacterGenerationInput) (*models.RolePlayCharacter, string, error) {
	if strings.TrimSpace(input.ProviderName) == "" || strings.TrimSpace(input.ModelName) == "" || strings.TrimSpace(input.APIKey) == "" {
		return nil, "", fmt.Errorf("providerName, modelName and apiKey are required")
	}
	url, err := ProviderURL(input.ProviderName)
	if err != nil {
		return nil, "", fmt.Errorf("providerName invalide")
	}
	prompt, _ := s.PrepareGenerationPrompt(ctx, input)
	callCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	client := provider.NewsProvider(strings.TrimSpace(input.APIKey), url, strings.TrimSpace(input.ModelName))
	raw, err := client.Chat(callCtx, []provider.ProviderMessage{
		{Role: "system", Content: "Tu generes des fiches personnage. Reponds uniquement avec un objet JSON valide, sans markdown."},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return nil, raw, err
	}

	draft, err := decodeRolePlayCharacterDraft(raw)
	if err != nil {
		return nil, raw, err
	}
	character, err := normalizeRolePlayCharacter(userID, draft)
	if err != nil {
		return nil, raw, err
	}
	return character, raw, nil
}

func normalizeRolePlayCharacter(userID uint, input RolePlayCharacterInput) (*models.RolePlayCharacter, error) {
	name := compactText(input.Name, 120)
	className := compactText(defaultString(input.Class, input.Archetype), 120)
	origin := compactText(input.Origin, 180)
	race := compactText(defaultString(input.Race, input.Species), 120)
	alignment := compactText(input.Alignment, 120)
	personality := compactText(input.Personality, 1000)
	background := compactText(input.Background, 1800)
	goal := compactText(defaultString(input.PersonalGoal, input.Goal), 1000)

	if name == "" || className == "" || origin == "" || race == "" || background == "" || goal == "" {
		return nil, fmt.Errorf("name, class, origin, race, background and personal_goal are required")
	}

	attributes := normalizeAttributes(input.Attributes)
	skills := normalizeSkills(input.Skills)
	traits := normalizeStringList(input.Traits, 2, 4, 80)
	inventory := normalizeStringList(input.Inventory, 2, 4, 100)

	if len(traits) == 0 {
		traits = []string{"Prudent", "Determiné"}
	}
	if len(inventory) == 0 {
		inventory = []string{"Carnet de notes", "Trousse de survie"}
	}

	attributesJSON, _ := json.Marshal(attributes)
	skillsJSON, _ := json.Marshal(skills)
	traitsJSON, _ := json.Marshal(traits)
	inventoryJSON, _ := json.Marshal(inventory)

	maxHealth := clampInt(input.MaxHealth, 1, 100, 100)
	health := clampInt(input.Health, 1, maxHealth, maxHealth)
	return &models.RolePlayCharacter{
		UserID:       userID,
		Name:         name,
		Class:        className,
		Origin:       origin,
		Race:         race,
		Alignment:    defaultString(alignment, "Neutre"),
		Personality:  personality,
		Background:   background,
		PersonalGoal: goal,
		Level:        clampInt(input.Level, 1, 1, 1),
		HeroImageID:  input.HeroImageID,
		ImageURL:     compactText(input.ImageURL, 500),
		Attributes:   datatypes.JSON(attributesJSON),
		Skills:       datatypes.JSON(skillsJSON),
		Traits:       datatypes.JSON(traitsJSON),
		Inventory:    datatypes.JSON(inventoryJSON),
		Health:       health,
		MaxHealth:    maxHealth,
		Stress:       clampInt(input.Stress, 0, 100, 0),
		Fatigue:      clampInt(input.Fatigue, 0, 100, 0),
		Morale:       clampInt(input.Morale, 0, 100, 50),
	}, nil
}

func decodeRolePlayCharacterDraft(raw string) (RolePlayCharacterInput, error) {
	var lastErr error
	for _, candidate := range jsonObjectCandidates(raw) {
		var input RolePlayCharacterInput
		if err := json.Unmarshal([]byte(candidate), &input); err == nil {
			if input.PersonalGoal == "" {
				var aliases map[string]any
				_ = json.Unmarshal([]byte(candidate), &aliases)
				input.PersonalGoal = stringAlias(aliases, "personalGoal", "personal_goal", "goal")
			}
			return input, nil
		} else {
			lastErr = err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no JSON object found")
	}
	return RolePlayCharacterInput{}, fmt.Errorf("invalid character JSON: %w", lastErr)
}

func CharacterPromptSummary(character *models.RolePlayCharacter) string {
	if character == nil {
		return ""
	}
	return fmt.Sprintf(`Heros joueur:
- Nom: %s
- Classe/archetype: %s
- Origine: %s
- Race/espece: %s
- Alignement/personnalite: %s / %s
- Objectif personnel: %s
- Background: %s
- Attributs: %s
- Competences: %s
- Traits: %s
- Inventaire: %s
- Etat: sante %d/%d, stress %d, fatigue %d, moral %d`,
		character.Name,
		character.Class,
		character.Origin,
		character.Race,
		character.Alignment,
		character.Personality,
		character.PersonalGoal,
		character.Background,
		string(character.Attributes),
		string(character.Skills),
		string(character.Traits),
		string(character.Inventory),
		character.Health,
		character.MaxHealth,
		character.Stress,
		character.Fatigue,
		character.Morale,
	)
}

func CharacterSnapshot(character *models.RolePlayCharacter) map[string]any {
	if character == nil {
		return nil
	}
	return map[string]any{
		"id":           character.Id,
		"name":         character.Name,
		"class":        character.Class,
		"origin":       character.Origin,
		"race":         character.Race,
		"alignment":    character.Alignment,
		"personality":  character.Personality,
		"background":   character.Background,
		"personalGoal": character.PersonalGoal,
		"level":        character.Level,
		"attributes":   decodeJSONMap(character.Attributes),
		"skills":       decodeJSONMap(character.Skills),
		"traits":       decodeJSONList(character.Traits),
		"inventory":    decodeJSONList(character.Inventory),
		"health":       character.Health,
		"maxHealth":    character.MaxHealth,
		"stress":       character.Stress,
		"fatigue":      character.Fatigue,
		"morale":       character.Morale,
	}
}

func normalizeAttributes(input map[string]int) map[string]int {
	keys := []string{"force", "agility", "constitution", "intelligence", "perception", "charisma"}
	aliases := map[string][]string{
		"force":        {"force", "FOR", "Force"},
		"agility":      {"agility", "agilite", "agilité", "AGI"},
		"constitution": {"constitution", "CON"},
		"intelligence": {"intelligence", "INT"},
		"perception":   {"perception", "PER"},
		"charisma":     {"charisma", "charisme", "CHA"},
	}
	out := map[string]int{}
	sixteenCount := 0
	for _, key := range keys {
		value := 10
		for _, alias := range aliases[key] {
			if raw, ok := input[alias]; ok {
				value = raw
				break
			}
		}
		value = clampInt(value, 8, 16, 10)
		if value == 16 {
			sixteenCount++
			if sixteenCount > 1 {
				value = 15
			}
		}
		out[key] = value
	}
	return out
}

func normalizeSkills(input map[string]int) map[string]int {
	keys := []string{"persuasion", "stealth", "combat", "observation", "survival", "investigation", "technology", "intimidation"}
	out := map[string]int{}
	for _, key := range keys {
		value := input[key]
		if value == 0 {
			value = input[strings.Title(key)]
		}
		out[key] = clampInt(value, 0, 3, 0)
	}
	return out
}

func normalizeStringList(input []string, min int, max int, maxLen int) []string {
	out := make([]string, 0, max)
	seen := map[string]struct{}{}
	for _, value := range input {
		clean := compactText(value, maxLen)
		if clean == "" {
			continue
		}
		key := strings.ToLower(clean)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, clean)
		if len(out) >= max {
			break
		}
	}
	if len(out) < min {
		return out
	}
	return out
}

func compactText(value string, maxLen int) string {
	clean := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if maxLen > 0 && len(clean) > maxLen {
		return clean[:maxLen]
	}
	return clean
}

func truncateText(value string, maxLen int) string {
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}

func clampInt(value int, min int, max int, fallback int) int {
	if value == 0 {
		value = fallback
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func jsonObjectCandidates(raw string) []string {
	clean := strings.TrimSpace(raw)
	if strings.HasPrefix(clean, "```json") {
		clean = strings.TrimSpace(strings.TrimPrefix(clean, "```json"))
	}
	if strings.HasPrefix(clean, "```") {
		clean = strings.TrimSpace(strings.TrimPrefix(clean, "```"))
	}
	if strings.HasSuffix(clean, "```") {
		clean = strings.TrimSpace(strings.TrimSuffix(clean, "```"))
	}
	candidates := []string{clean}
	start := strings.Index(clean, "{")
	end := strings.LastIndex(clean, "}")
	if start >= 0 && end > start {
		candidates = append(candidates, clean[start:end+1])
	}
	return candidates
}

func stringAlias(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			return fmt.Sprint(value)
		}
	}
	return ""
}

func decodeJSONMap(raw datatypes.JSON) map[string]any {
	var out map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &out) != nil || out == nil {
		return map[string]any{}
	}
	return out
}

func decodeJSONList(raw datatypes.JSON) []any {
	var out []any
	if len(raw) == 0 || json.Unmarshal(raw, &out) != nil || out == nil {
		return []any{}
	}
	return out
}
