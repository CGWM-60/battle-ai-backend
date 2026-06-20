package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/provider"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type RolePlayQuestVisualService struct {
	db *gorm.DB
}

func NewRolePlayQuestVisualService(db *gorm.DB) *RolePlayQuestVisualService {
	return &RolePlayQuestVisualService{db: db}
}

type RolePlaySceneInput struct {
	SceneKey       string
	ArcID          *uint
	ChapterID      *uint
	ChapterIndex   int
	ArcIndex       int
	Title          string
	Summary        string
	Prompt         string
	ImagePrompt    string
	NegativePrompt string
	SceneType      string
	RoomType       string
	Atmosphere     string
	DangerLevel    string
	VisualTags     []string
	RpgMetadata    map[string]any
}

type BackfillImagePromptsInput struct {
	OnlyMissing     bool   `json:"onlyMissing"`
	ForceRegenerate bool   `json:"forceRegenerate"`
	SceneMode       string `json:"sceneMode"`
	SceneCount      int    `json:"sceneCount"`
}

type BackfillImagePromptsResult struct {
	UpdatedQuests  int      `json:"updatedQuests"`
	CreatedScenes  int      `json:"createdScenes"`
	UpdatedPrompts int      `json:"updatedPrompts"`
	Errors         []string `json:"errors"`
}

type GenerateImagePromptsInput struct {
	OnlyMissing     bool   `json:"onlyMissing"`
	ForceRegenerate bool   `json:"forceRegenerate"`
	SceneMode       string `json:"sceneMode"`
	SceneCount      int    `json:"sceneCount"`
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	APIKey          string `json:"apiKey"`
}

type RolePlayQuestPromptGenerationResult struct {
	QuestID        uint   `json:"questId"`
	UpdatedPrompts int    `json:"updatedPrompts"`
	CreatedScenes  int    `json:"createdScenes"`
	Skipped        bool   `json:"skipped"`
	Error          string `json:"error,omitempty"`
}

type PublishActionResult struct {
	UpdatedCount int       `json:"updatedCount"`
	At           time.Time `json:"at"`
	AdminUser    string    `json:"adminUser,omitempty"`
}

func RolePlayScenePublicDir() string {
	if dir := strings.TrimSpace(os.Getenv("ROLEPLAY_SCENE_PUBLIC_DIR")); dir != "" {
		return dir
	}
	baseDir := strings.TrimSpace(os.Getenv("NEXUS_ASSETS_BASE_DIR"))
	if baseDir == "" {
		baseDir = "uploads"
	}
	return filepath.Join(baseDir, "roleplay", "quests")
}

func RolePlayScenePublicBaseURL() string {
	return strings.TrimRight(defaultText(os.Getenv("ROLEPLAY_SCENE_PUBLIC_BASE_URL"), "/uploads/roleplay/quests"), "/")
}

func (s *RolePlayQuestVisualService) PublishQuest(ctx context.Context, id uint, adminUser string) (*PublishActionResult, error) {
	now := time.Now()
	fields := map[string]any{
		"is_published":   true,
		"status":         constants.QuestStatusPublished,
		"published_at":   now,
		"unpublished_at": nil,
	}
	if err := s.db.WithContext(ctx).Model(&models.RolePlayQuestTemplate{}).Where("id = ?", id).Updates(fields).Error; err != nil {
		return nil, err
	}
	return &PublishActionResult{UpdatedCount: 1, At: now, AdminUser: adminUser}, nil
}

func (s *RolePlayQuestVisualService) UnpublishQuest(ctx context.Context, id uint, adminUser string) (*PublishActionResult, error) {
	now := time.Now()
	fields := map[string]any{
		"is_published":   false,
		"status":         constants.QuestStatusDraft,
		"unpublished_at": now,
	}
	if err := s.db.WithContext(ctx).Model(&models.RolePlayQuestTemplate{}).Where("id = ?", id).Updates(fields).Error; err != nil {
		return nil, err
	}
	return &PublishActionResult{UpdatedCount: 1, At: now, AdminUser: adminUser}, nil
}

func (s *RolePlayQuestVisualService) UnpublishAllQuests(ctx context.Context, adminUser string) (*PublishActionResult, error) {
	now := time.Now()
	result := s.db.WithContext(ctx).
		Model(&models.RolePlayQuestTemplate{}).
		Where("is_published = ? OR status = ?", true, constants.QuestStatusPublished).
		Updates(map[string]any{
			"is_published":   false,
			"status":         constants.QuestStatusDraft,
			"unpublished_at": now,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	return &PublishActionResult{UpdatedCount: int(result.RowsAffected), At: now, AdminUser: adminUser}, nil
}

func (s *RolePlayQuestVisualService) ListScenes(ctx context.Context, questID uint) ([]models.RolePlayQuestScene, error) {
	var scenes []models.RolePlayQuestScene
	err := s.db.WithContext(ctx).
		Preload("Images", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("is_main DESC").Order("id ASC")
		}).
		Where("quest_id = ?", questID).
		Order("chapter_index ASC").
		Order("id ASC").
		Find(&scenes).Error
	return scenes, err
}

func (s *RolePlayQuestVisualService) CreateScenes(ctx context.Context, questID uint, inputs []RolePlaySceneInput) (int, error) {
	if len(inputs) == 0 {
		return 0, nil
	}
	created := 0
	for _, input := range inputs {
		tags, _ := json.Marshal(input.VisualTags)
		rpg, _ := json.Marshal(input.RpgMetadata)
		scene := models.RolePlayQuestScene{
			QuestID:        questID,
			ArcID:          input.ArcID,
			ChapterID:      input.ChapterID,
			SceneKey:       defaultString(input.SceneKey, fmt.Sprintf("scene_%02d", input.ChapterIndex)),
			ChapterIndex:   input.ChapterIndex,
			ArcIndex:       input.ArcIndex,
			Title:          input.Title,
			Summary:        input.Summary,
			Prompt:         input.Prompt,
			ImagePrompt:    input.ImagePrompt,
			NegativePrompt: input.NegativePrompt,
			SceneType:      defaultString(input.SceneType, "exploration"),
			RoomType:       defaultString(input.RoomType, "generic"),
			Atmosphere:     defaultString(input.Atmosphere, "mysterious"),
			DangerLevel:    defaultString(input.DangerLevel, "medium"),
			VisualTags:     datatypes.JSON(tags),
			RpgMetadata:    datatypes.JSON(rpg),
			ImageStatus:    "prompt_only",
		}
		if err := s.db.WithContext(ctx).Create(&scene).Error; err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}

func (s *RolePlayQuestVisualService) ApplyGeneratedVisuals(
	ctx context.Context,
	questID uint,
	theme string,
	level string,
	title string,
	summary string,
	imagePrompt string,
	imageNegativePrompt string,
	visualStyle string,
	visualTags []string,
	rpgMetadata map[string]any,
	scenes []RolePlaySceneInput,
) error {
	tagsJSON, _ := json.Marshal(visualTags)
	rpgJSON, _ := json.Marshal(rpgMetadata)
	fields := map[string]any{
		"image_prompt":          imagePrompt,
		"image_negative_prompt": imageNegativePrompt,
		"visual_style":          defaultString(visualStyle, "dark fantasy mobile RPG"),
		"visual_tags":           datatypes.JSON(tagsJSON),
		"rpg_metadata":          datatypes.JSON(rpgJSON),
	}
	if err := s.db.WithContext(ctx).Model(&models.RolePlayQuestTemplate{}).Where("id = ?", questID).Updates(fields).Error; err != nil {
		return err
	}
	if len(scenes) == 0 {
		return nil
	}
	_, err := s.CreateScenes(ctx, questID, scenes)
	return err
}

func (s *RolePlayQuestVisualService) BackfillImagePrompts(ctx context.Context, input BackfillImagePromptsInput) (*BackfillImagePromptsResult, error) {
	genInput := GenerateImagePromptsInput{
		OnlyMissing:     input.OnlyMissing,
		ForceRegenerate: input.ForceRegenerate,
		SceneMode:       input.SceneMode,
		SceneCount:      input.SceneCount,
	}
	result := &BackfillImagePromptsResult{Errors: []string{}}
	var quests []models.RolePlayQuestTemplate
	if err := s.db.WithContext(ctx).Order("id ASC").Find(&quests).Error; err != nil {
		return nil, err
	}
	for _, quest := range quests {
		item, err := s.GenerateImagePromptsForQuest(ctx, quest.Id, genInput)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("quest %d: %v", quest.Id, err))
			continue
		}
		if item.Skipped {
			continue
		}
		result.UpdatedPrompts += item.UpdatedPrompts
		result.CreatedScenes += item.CreatedScenes
		if item.UpdatedPrompts > 0 || item.CreatedScenes > 0 {
			result.UpdatedQuests++
		}
	}
	return result, nil
}

func (s *RolePlayQuestVisualService) GenerateImagePromptsForQuest(
	ctx context.Context,
	questID uint,
	input GenerateImagePromptsInput,
) (*RolePlayQuestPromptGenerationResult, error) {
	sceneMode := normalizeSceneMode(input.SceneMode)
	sceneCount := input.SceneCount
	var quest models.RolePlayQuestTemplate
	if err := s.db.WithContext(ctx).
		Preload("Arcs", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("position ASC").Order("id ASC")
		}).
		Preload("Arcs.Chapters", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("position ASC").Order("id ASC")
		}).
		First(&quest, questID).Error; err != nil {
		return nil, err
	}
	var scenes []models.RolePlayQuestScene
	_ = s.db.WithContext(ctx).Where("quest_id = ?", questID).Order("chapter_index ASC").Find(&scenes).Error

	if input.OnlyMissing && !input.ForceRegenerate && isQuestVisualComplete(quest, scenes) {
		return &RolePlayQuestPromptGenerationResult{QuestID: questID, Skipped: true}, nil
	}

	result := &RolePlayQuestPromptGenerationResult{QuestID: questID}
	if strings.TrimSpace(input.Provider) != "" && strings.TrimSpace(input.APIKey) != "" && strings.TrimSpace(input.Model) != "" {
		if err := s.applyAIVisualsForQuest(ctx, quest, scenes, input, sceneMode, sceneCount, result); err == nil {
			return result, nil
		}
	}
	return s.applyHeuristicVisualsForQuest(ctx, quest, scenes, input, sceneMode, sceneCount, result)
}

func (s *RolePlayQuestVisualService) GenerateImagePromptForScene(
	ctx context.Context,
	questID uint,
	sceneID uint,
	input GenerateImagePromptsInput,
) (*RolePlayQuestPromptGenerationResult, error) {
	var quest models.RolePlayQuestTemplate
	if err := s.db.WithContext(ctx).First(&quest, questID).Error; err != nil {
		return nil, err
	}
	var scene models.RolePlayQuestScene
	if err := s.db.WithContext(ctx).Where("id = ? AND quest_id = ?", sceneID, questID).First(&scene).Error; err != nil {
		return nil, err
	}
	result := &RolePlayQuestPromptGenerationResult{QuestID: questID}
	if !input.ForceRegenerate &&
		strings.TrimSpace(scene.ImagePrompt) != "" &&
		strings.TrimSpace(scene.NegativePrompt) != "" {
		result.Skipped = true
		return result, nil
	}
	prompt := buildSceneImagePrompt(quest.Theme, scene.RoomType, scene.Title, scene.Summary)
	if err := s.db.WithContext(ctx).Model(&scene).Updates(map[string]any{
		"image_prompt":    prompt,
		"negative_prompt": buildDefaultNegativePrompt(),
	}).Error; err != nil {
		return nil, err
	}
	result.UpdatedPrompts = 1
	return result, nil
}

func isQuestVisualComplete(quest models.RolePlayQuestTemplate, scenes []models.RolePlayQuestScene) bool {
	expected := countQuestChapters(quest)
	if expected > 0 && len(scenes) < expected {
		return false
	}
	if strings.TrimSpace(quest.ImagePrompt) == "" ||
		strings.TrimSpace(quest.ImageNegativePrompt) == "" ||
		len(scenes) == 0 {
		return false
	}
	for _, scene := range scenes {
		if strings.TrimSpace(scene.ImagePrompt) == "" || strings.TrimSpace(scene.NegativePrompt) == "" {
			return false
		}
	}
	return true
}

func (s *RolePlayQuestVisualService) CreateOrUpdateScenesForChapters(
	ctx context.Context,
	questID uint,
	inputs []RolePlaySceneInput,
) (int, error) {
	if len(inputs) == 0 {
		return 0, nil
	}
	var existing []models.RolePlayQuestScene
	if err := s.db.WithContext(ctx).Where("quest_id = ?", questID).Find(&existing).Error; err != nil {
		return 0, err
	}
	created := 0
	for _, input := range inputs {
		if matchIndex := bestMatchingRolePlaySceneIndex(existing, input); matchIndex >= 0 {
			matched := &existing[matchIndex]
			updates := map[string]any{}
			if matched.ArcID == nil && input.ArcID != nil {
				updates["arc_id"] = *input.ArcID
				matched.ArcID = input.ArcID
			}
			if matched.ChapterID == nil && input.ChapterID != nil {
				updates["chapter_id"] = *input.ChapterID
				matched.ChapterID = input.ChapterID
			}
			if matched.ArcIndex <= 0 && input.ArcIndex > 0 {
				updates["arc_index"] = input.ArcIndex
				matched.ArcIndex = input.ArcIndex
			}
			if matched.ChapterIndex <= 0 && input.ChapterIndex > 0 {
				updates["chapter_index"] = input.ChapterIndex
				matched.ChapterIndex = input.ChapterIndex
			}
			if len(updates) > 0 {
				if err := s.db.WithContext(ctx).Model(&models.RolePlayQuestScene{}).Where("id = ?", matched.Id).Updates(updates).Error; err != nil {
					return created, err
				}
			}
			continue
		}
		count, err := s.CreateScenes(ctx, questID, []RolePlaySceneInput{input})
		if err != nil {
			return created, err
		}
		created += count
	}
	return created, nil
}

func (s *RolePlayQuestVisualService) ResolveOrCreateSceneForChapter(
	ctx context.Context,
	questID uint,
	chapterID uint,
) (*models.RolePlayQuestScene, error) {
	var scene models.RolePlayQuestScene
	err := s.db.WithContext(ctx).
		Where("quest_id = ? AND chapter_id = ?", questID, chapterID).
		First(&scene).Error
	if err == nil {
		return &scene, nil
	}

	var quest models.RolePlayQuestTemplate
	if err := s.db.WithContext(ctx).
		Preload("Arcs", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("position ASC").Order("id ASC")
		}).
		Preload("Arcs.Chapters", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("position ASC").Order("id ASC")
		}).
		First(&quest, questID).Error; err != nil {
		return nil, err
	}

	for _, input := range BuildScenesFromQuestStructure(quest) {
		if input.ChapterID != nil && *input.ChapterID == chapterID {
			var existing []models.RolePlayQuestScene
			if err := s.db.WithContext(ctx).Where("quest_id = ?", questID).Order("id ASC").Find(&existing).Error; err != nil {
				return nil, err
			}
			if matchIndex := bestMatchingRolePlaySceneIndex(existing, input); matchIndex >= 0 {
				scene = existing[matchIndex]
				updates := map[string]any{
					"arc_id":        input.ArcID,
					"chapter_id":    input.ChapterID,
					"arc_index":     input.ArcIndex,
					"chapter_index": input.ChapterIndex,
				}
				if err := s.db.WithContext(ctx).Model(&models.RolePlayQuestScene{}).Where("id = ?", scene.Id).Updates(updates).Error; err != nil {
					return nil, err
				}
				scene.ArcID = input.ArcID
				scene.ChapterID = input.ChapterID
				scene.ArcIndex = input.ArcIndex
				scene.ChapterIndex = input.ChapterIndex
				return &scene, nil
			}
			if _, err := s.CreateScenes(ctx, questID, []RolePlaySceneInput{input}); err != nil {
				return nil, err
			}
			if err := s.db.WithContext(ctx).
				Where("quest_id = ? AND chapter_id = ?", questID, chapterID).
				First(&scene).Error; err != nil {
				return nil, err
			}
			return &scene, nil
		}
	}
	return nil, fmt.Errorf("chapter %d not found for quest %d", chapterID, questID)
}

func (s *RolePlayQuestVisualService) applyHeuristicVisualsForQuest(
	ctx context.Context,
	quest models.RolePlayQuestTemplate,
	scenes []models.RolePlayQuestScene,
	input GenerateImagePromptsInput,
	sceneMode string,
	sceneCount int,
	result *RolePlayQuestPromptGenerationResult,
) (*RolePlayQuestPromptGenerationResult, error) {
	updated := false
	if strings.TrimSpace(quest.ImagePrompt) == "" || input.ForceRegenerate {
		imagePrompt := buildQuestImagePrompt(quest.Theme, quest.Level, quest.Title, quest.Summary)
		tagsJSON, _ := json.Marshal(buildDefaultVisualTags(quest.Theme))
		rpgJSON, _ := json.Marshal(buildDefaultRpgMetadata(quest.Level, quest.Theme))
		if err := s.db.WithContext(ctx).Model(&models.RolePlayQuestTemplate{}).Where("id = ?", quest.Id).Updates(map[string]any{
			"image_prompt":          imagePrompt,
			"image_negative_prompt": buildDefaultNegativePrompt(),
			"visual_style":          "dark fantasy mobile RPG",
			"visual_tags":           datatypes.JSON(tagsJSON),
			"rpg_metadata":          datatypes.JSON(rpgJSON),
		}).Error; err != nil {
			return nil, err
		}
		result.UpdatedPrompts++
		updated = true
	}
	missingScenes := missingChapterSceneInputs(quest, scenes)
	if len(missingScenes) == 0 && len(scenes) == 0 {
		missingScenes = BuildScenesForQuestMode(quest, sceneMode, sceneCount)
	}
	if sceneCount > 0 && len(missingScenes) > sceneCount {
		missingScenes = missingScenes[:sceneCount]
	}
	if len(missingScenes) > 0 {
		count, err := s.CreateOrUpdateScenesForChapters(ctx, quest.Id, missingScenes)
		if err != nil {
			return nil, err
		}
		result.CreatedScenes += count
		result.UpdatedPrompts += count
		updated = true
		if count > 0 {
			_ = s.db.WithContext(ctx).Where("quest_id = ?", quest.Id).Order("chapter_index ASC").Find(&scenes).Error
		}
	} else if input.ForceRegenerate || !input.OnlyMissing {
		for _, scene := range scenes {
			if strings.TrimSpace(scene.ImagePrompt) != "" && !input.ForceRegenerate {
				continue
			}
			prompt := buildSceneImagePrompt(quest.Theme, scene.RoomType, scene.Title, scene.Summary)
			if err := s.db.WithContext(ctx).Model(&models.RolePlayQuestScene{}).Where("id = ?", scene.Id).Updates(map[string]any{
				"image_prompt":    prompt,
				"negative_prompt": buildDefaultNegativePrompt(),
			}).Error; err != nil {
				return nil, err
			}
			result.UpdatedPrompts++
			updated = true
		}
	}
	if updated && (result.UpdatedPrompts > 0 || result.CreatedScenes > 0) {
		return result, nil
	}
	if !updated {
		result.Skipped = true
	}
	return result, nil
}

type generatedQuestVisuals struct {
	ImagePrompt         string                      `json:"imagePrompt"`
	ImageNegativePrompt string                      `json:"imageNegativePrompt"`
	VisualStyle         string                      `json:"visualStyle"`
	VisualTags          []string                    `json:"visualTags"`
	RpgMetadata         map[string]any              `json:"rpgMetadata"`
	Scenes              []generatedQuestVisualScene `json:"scenes"`
}

type generatedQuestVisualScene struct {
	SceneKey            string   `json:"sceneKey"`
	ChapterIndex        int      `json:"chapterIndex"`
	Title               string   `json:"title"`
	Summary             string   `json:"summary"`
	SceneType           string   `json:"sceneType"`
	RoomType            string   `json:"roomType"`
	Atmosphere          string   `json:"atmosphere"`
	DangerLevel         string   `json:"dangerLevel"`
	ImagePrompt         string   `json:"imagePrompt"`
	ImageNegativePrompt string   `json:"imageNegativePrompt"`
	VisualTags          []string `json:"visualTags"`
}

func (s *RolePlayQuestVisualService) applyAIVisualsForQuest(
	ctx context.Context,
	quest models.RolePlayQuestTemplate,
	existingScenes []models.RolePlayQuestScene,
	input GenerateImagePromptsInput,
	sceneMode string,
	sceneCount int,
	result *RolePlayQuestPromptGenerationResult,
) error {
	prompt := fmt.Sprintf(`Genere uniquement les metadonnees visuelles pour cette quete RP existante.
Ne modifie PAS le titre, le resume, le prompt principal, les arcs ou chapitres.
Reponds uniquement en JSON valide, sans markdown.

Quete:
- title: %s
- summary: %s
- theme: %s
- level: %s

Format:
{"imagePrompt":"...","imageNegativePrompt":"...","visualStyle":"dark fantasy mobile RPG","visualTags":["crypt","fog"],"rpgMetadata":{"recommendedPartySize":1,"supportsCoop":true,"recommendedLevel":1,"difficultyClass":12,"mainThreat":"...","mainLocation":"...","rpgTags":[],"suggestedSkills":[]},"scenes":[{"sceneKey":"scene_01_entry","chapterIndex":1,"title":"...","summary":"...","sceneType":"exploration","roomType":"entrance","atmosphere":"mysterious","dangerLevel":"low","imagePrompt":"...","imageNegativePrompt":"...","visualTags":[]}]}

Regles image:
- prompts en anglais, utilisables directement dans un generateur d'image
- pas de texte visible, pas de logo, pas de watermark
- mobile game background, vertical ou cinematic
- une scene par chapitre si possible (%d chapitres)
- coherence visuelle entre les scenes`, quest.Title, quest.Summary, quest.Theme, quest.Level, countQuestChapters(quest))

	raw, err := s.callProvider(ctx, input.Provider, input.APIKey, input.Model, prompt)
	if err != nil {
		return err
	}
	visuals, err := decodeGeneratedQuestVisuals(raw)
	if err != nil {
		return err
	}
	if strings.TrimSpace(visuals.ImagePrompt) == "" && len(visuals.Scenes) == 0 {
		return fmt.Errorf("provider returned empty visuals")
	}

	if strings.TrimSpace(visuals.ImagePrompt) != "" && (input.ForceRegenerate || strings.TrimSpace(quest.ImagePrompt) == "") {
		tagsJSON, _ := json.Marshal(visuals.VisualTags)
		rpgJSON, _ := json.Marshal(visuals.RpgMetadata)
		if err := s.db.WithContext(ctx).Model(&models.RolePlayQuestTemplate{}).Where("id = ?", quest.Id).Updates(map[string]any{
			"image_prompt":          visuals.ImagePrompt,
			"image_negative_prompt": defaultString(visuals.ImageNegativePrompt, buildDefaultNegativePrompt()),
			"visual_style":          defaultString(visuals.VisualStyle, "dark fantasy mobile RPG"),
			"visual_tags":           datatypes.JSON(tagsJSON),
			"rpg_metadata":          datatypes.JSON(rpgJSON),
		}).Error; err != nil {
			return err
		}
		result.UpdatedPrompts++
	}

	missingScenes := missingChapterSceneInputs(quest, existingScenes)
	if len(missingScenes) == 0 && len(existingScenes) == 0 {
		if len(visuals.Scenes) > 0 {
			for index, item := range visuals.Scenes {
				chapterIndex := item.ChapterIndex
				if chapterIndex <= 0 {
					chapterIndex = index + 1
				}
				missingScenes = append(missingScenes, RolePlaySceneInput{
					SceneKey:       defaultString(item.SceneKey, fmt.Sprintf("scene_%02d", index+1)),
					ChapterIndex:   chapterIndex,
					Title:          item.Title,
					Summary:        item.Summary,
					SceneType:      item.SceneType,
					RoomType:       item.RoomType,
					Atmosphere:     item.Atmosphere,
					DangerLevel:    item.DangerLevel,
					ImagePrompt:    item.ImagePrompt,
					NegativePrompt: item.ImageNegativePrompt,
					VisualTags:     item.VisualTags,
				})
			}
		} else {
			missingScenes = BuildScenesForQuestMode(quest, sceneMode, sceneCount)
		}
	}
	if sceneCount > 0 && len(missingScenes) > sceneCount {
		missingScenes = missingScenes[:sceneCount]
	}
	if len(missingScenes) > 0 {
		count, err := s.CreateOrUpdateScenesForChapters(ctx, quest.Id, missingScenes)
		if err != nil {
			return err
		}
		result.CreatedScenes += count
		result.UpdatedPrompts += count
	} else if input.ForceRegenerate {
		for _, scene := range existingScenes {
			promptText := buildSceneImagePrompt(quest.Theme, scene.RoomType, scene.Title, scene.Summary)
			for _, generated := range visuals.Scenes {
				if generated.SceneKey == scene.SceneKey || generated.Title == scene.Title {
					if strings.TrimSpace(generated.ImagePrompt) != "" {
						promptText = generated.ImagePrompt
					}
					break
				}
			}
			if err := s.db.WithContext(ctx).Model(&models.RolePlayQuestScene{}).Where("id = ?", scene.Id).Updates(map[string]any{
				"image_prompt":    promptText,
				"negative_prompt": buildDefaultNegativePrompt(),
			}).Error; err != nil {
				return err
			}
			result.UpdatedPrompts++
		}
	}
	return nil
}

func (s *RolePlayQuestVisualService) callProvider(ctx context.Context, providerName, apiKey, model, prompt string) (string, error) {
	url, err := ProviderURL(providerName)
	if err != nil {
		return "", err
	}
	ai := provider.NewsProvider(apiKey, url, model)
	return ai.Chat(ctx, []provider.ProviderMessage{{Role: "system", Content: prompt}})
}

func decodeGeneratedQuestVisuals(raw string) (generatedQuestVisuals, error) {
	var visuals generatedQuestVisuals
	source := strings.TrimSpace(raw)
	start := strings.Index(source, "{")
	end := strings.LastIndex(source, "}")
	if start >= 0 && end > start {
		source = source[start : end+1]
	}
	if err := json.Unmarshal([]byte(source), &visuals); err != nil {
		return visuals, err
	}
	return visuals, nil
}

func (s *RolePlayQuestVisualService) SaveSceneImageUpload(
	ctx context.Context,
	questID uint,
	sceneID uint,
	originalName string,
	reader io.Reader,
	alt string,
) (*models.RolePlayQuestSceneImage, error) {
	var scene models.RolePlayQuestScene
	if err := s.db.WithContext(ctx).Where("id = ? AND quest_id = ?", sceneID, questID).First(&scene).Error; err != nil {
		return nil, err
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	maxSize := RolePlaySceneMaxUploadBytes()
	if int64(len(data)) > maxSize {
		return nil, fmt.Errorf("image file is too large")
	}
	converted, originalExt, originalMime, err := ConvertUploadedImageBytesToWebP(data, originalName)
	if err != nil {
		return nil, err
	}
	assetDir := RolePlayScenePublicDir()
	publicBase := RolePlayScenePublicBaseURL()
	relDir := filepath.Join(fmt.Sprintf("%d", questID), "scenes", fmt.Sprintf("%d", sceneID))
	fullDir := filepath.Join(assetDir, relDir)
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		return nil, err
	}
	filename := fmt.Sprintf("scene_%d_%d.webp", sceneID, time.Now().UnixNano())
	fullPath := filepath.Join(fullDir, filename)
	if err := os.WriteFile(fullPath, converted.Data, 0o644); err != nil {
		return nil, err
	}
	relURL := filepath.ToSlash(filepath.Join(relDir, filename))
	url := publicBase + "/" + relURL
	var existingCount int64
	_ = s.db.WithContext(ctx).Model(&models.RolePlayQuestSceneImage{}).Where("scene_id = ?", sceneID).Count(&existingCount).Error
	isMain := existingCount == 0

	image := &models.RolePlayQuestSceneImage{
		SceneID:          sceneID,
		QuestID:          questID,
		URL:              url,
		StorageKey:       relURL,
		Filename:         filename,
		MimeType:         rolePlayImageWebPMime,
		Size:             int64(len(converted.Data)),
		Width:            converted.Width,
		Height:           converted.Height,
		IsMain:           isMain,
		Alt:              defaultString(alt, scene.Title),
		Source:           "admin_upload",
		OriginalFilename: strings.TrimSpace(originalName),
		OriginalMimeType: strings.TrimSpace(originalMime),
	}
	_ = originalExt
	return image, s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if isMain {
			if err := tx.Model(&models.RolePlayQuestSceneImage{}).
				Where("scene_id = ?", sceneID).
				Update("is_main", false).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(image).Error; err != nil {
			_ = os.Remove(fullPath)
			return err
		}
		if isMain {
			return tx.Model(&models.RolePlayQuestScene{}).Where("id = ?", sceneID).Updates(map[string]any{
				"image_url":         url,
				"image_storage_key": relURL,
				"image_status":      "uploaded",
				"image_alt":         image.Alt,
			}).Error
		}
		return nil
	})
}

func (s *RolePlayQuestVisualService) DeleteSceneImage(ctx context.Context, questID, sceneID, imageID uint) error {
	var image models.RolePlayQuestSceneImage
	if err := s.db.WithContext(ctx).
		Where("id = ? AND scene_id = ? AND quest_id = ?", imageID, sceneID, questID).
		First(&image).Error; err != nil {
		return err
	}
	storageKey := image.StorageKey
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&image).Error; err != nil {
			return err
		}
		if err := removeRolePlaySceneImageFile(storageKey); err != nil {
			log.Printf("roleplay scene image delete warning quest=%d scene=%d image=%d: %v", questID, sceneID, imageID, err)
		}
		if image.IsMain {
			var next models.RolePlayQuestSceneImage
			err := tx.Where("scene_id = ?", sceneID).Order("id DESC").First(&next).Error
			updates := map[string]any{"image_status": "prompt_only", "image_url": "", "image_storage_key": ""}
			if err == nil {
				_ = tx.Model(&models.RolePlayQuestSceneImage{}).Where("id = ?", next.Id).Update("is_main", true).Error
				updates["image_url"] = next.URL
				updates["image_storage_key"] = next.StorageKey
				updates["image_status"] = "uploaded"
			}
			return tx.Model(&models.RolePlayQuestScene{}).Where("id = ?", sceneID).Updates(updates).Error
		}
		return nil
	})
}

func (s *RolePlayQuestVisualService) SetMainSceneImage(ctx context.Context, questID, sceneID, imageID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var image models.RolePlayQuestSceneImage
		if err := tx.Where("id = ? AND scene_id = ? AND quest_id = ?", imageID, sceneID, questID).First(&image).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.RolePlayQuestSceneImage{}).Where("scene_id = ?", sceneID).Update("is_main", false).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.RolePlayQuestSceneImage{}).Where("id = ?", imageID).Update("is_main", true).Error; err != nil {
			return err
		}
		return tx.Model(&models.RolePlayQuestScene{}).Where("id = ?", sceneID).Updates(map[string]any{
			"image_url":         image.URL,
			"image_storage_key": image.StorageKey,
			"image_status":      "uploaded",
			"image_alt":         image.Alt,
		}).Error
	})
}

func buildDefaultScenes(theme, level, title, summary string) []RolePlaySceneInput {
	return []RolePlaySceneInput{
		{
			SceneKey: "scene_01_entry", ChapterIndex: 1, ArcIndex: 1,
			Title: "Entrée", Summary: fmt.Sprintf("Ouverture de %s", title),
			SceneType: "exploration", RoomType: "entrance", Atmosphere: "mysterious", DangerLevel: "low",
			ImagePrompt:    buildSceneImagePrompt(theme, "entrance", "Entrée", summary),
			NegativePrompt: buildDefaultNegativePrompt(),
			VisualTags:     []string{"entrance", "fog"},
		},
		{
			SceneKey: "scene_02_conflict", ChapterIndex: 2, ArcIndex: 1,
			Title: "Conflit", Summary: "Tension et exploration",
			SceneType: "exploration", RoomType: "corridor", Atmosphere: "tense", DangerLevel: "medium",
			ImagePrompt:    buildSceneImagePrompt(theme, "corridor", "Conflit", summary),
			NegativePrompt: buildDefaultNegativePrompt(),
			VisualTags:     []string{"corridor", "shadows"},
		},
		{
			SceneKey: "scene_03_climax", ChapterIndex: 3, ArcIndex: 1,
			Title: "Climax", Summary: "Révélation ou confrontation",
			SceneType: "boss", RoomType: "boss_room", Atmosphere: "dramatic", DangerLevel: levelDanger(level),
			ImagePrompt:    buildSceneImagePrompt(theme, "boss_room", "Climax", summary),
			NegativePrompt: buildDefaultNegativePrompt(),
			VisualTags:     []string{"boss", "dramatic_light"},
		},
	}
}

func buildQuestImagePrompt(theme, level, title, summary string) string {
	style := "dark fantasy"
	if strings.Contains(strings.ToLower(theme), "cyber") || strings.Contains(strings.ToLower(theme), "sf") {
		style = "cyber fantasy"
	}
	return fmt.Sprintf("%s quest key art, %s, level %s, cinematic mobile game background, dramatic lighting, no text, no watermark, high detail", style, title, level)
}

func buildSceneImagePrompt(theme, roomType, title, summary string) string {
	style := "dark fantasy"
	if strings.Contains(strings.ToLower(theme), "cyber") || strings.Contains(strings.ToLower(theme), "sf") {
		style = "cyber fantasy"
	}
	return fmt.Sprintf("%s %s interior, %s, %s, cinematic mobile game background, dramatic lighting, no text, no watermark, high detail", style, roomType, title, strings.TrimSpace(summary))
}

func buildDefaultNegativePrompt() string {
	return "text, watermark, logo, blurry, low quality, deformed, extra limbs, known character, celebrity"
}

func buildDefaultVisualTags(theme string) []string {
	t := strings.ToLower(strings.TrimSpace(theme))
	switch {
	case strings.Contains(t, "horreur"):
		return []string{"crypt", "fog", "ruins"}
	case strings.Contains(t, "sf"), strings.Contains(t, "cyber"):
		return []string{"neon", "ruins", "hologram"}
	default:
		return []string{"dungeon", "runes", "fog"}
	}
}

func buildDefaultRpgMetadata(level, theme string) map[string]any {
	dc := 12
	switch strings.ToLower(level) {
	case "difficile":
		dc = 16
	case "moyen":
		dc = 14
	}
	return map[string]any{
		"recommendedPartySize": 1,
		"supportsCoop":         true,
		"recommendedLevel":     1,
		"difficultyClass":      dc,
		"mainThreat":           "unknown danger",
		"mainLocation":         theme,
		"rpgTags":              []string{"dungeon", "mystery"},
		"suggestedSkills":      []string{"Perception", "Investigation"},
	}
}

func levelDanger(level string) string {
	switch strings.ToLower(level) {
	case "difficile":
		return "high"
	case "moyen":
		return "medium"
	default:
		return "low"
	}
}

func mimeFromExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

func parseInt64(value string) (int64, error) {
	var out int64
	_, err := fmt.Sscan(value, &out)
	return out, err
}

func validateRolePlayImageUpload(originalName string, data []byte) (string, string, error) {
	if len(data) == 0 {
		return "", "", fmt.Errorf("empty image file")
	}
	ext := strings.ToLower(filepath.Ext(originalName))
	if !rolePlayAllowedUploadExt(ext) {
		return "", "", fmt.Errorf("unsupported image format")
	}
	mimeType := http.DetectContentType(data)
	switch mimeType {
	case "image/png", "image/jpeg", "image/webp":
	default:
		return "", "", fmt.Errorf("invalid image mime type")
	}
	return ext, mimeType, nil
}

func removeRolePlaySceneImageFile(storageKey string) error {
	key := strings.TrimSpace(storageKey)
	if key == "" || strings.Contains(key, "..") {
		return nil
	}
	fullPath := filepath.Join(RolePlayScenePublicDir(), filepath.FromSlash(key))
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func DeleteQuestUploadDir(questID uint) error {
	dir := filepath.Join(RolePlayScenePublicDir(), fmt.Sprintf("%d", questID))
	if strings.Contains(dir, "..") {
		return nil
	}
	return os.RemoveAll(dir)
}

func RolePlaySceneMaxUploadBytes() int64 {
	maxSize := int64(8 * 1024 * 1024)
	if v := strings.TrimSpace(os.Getenv("ROLEPLAY_SCENE_MAX_BYTES")); v != "" {
		if parsed, parseErr := parseInt64(v); parseErr == nil && parsed > 0 {
			maxSize = parsed
		}
	}
	return maxSize
}

// RolePlaySceneMaxUploadRequestBytes limits a complete multipart batch. It is
// deliberately separate from the per-file limit so several valid images can
// be uploaded together.
func RolePlaySceneMaxUploadRequestBytes() int64 {
	maxSize := int64(64 * 1024 * 1024)
	if v := strings.TrimSpace(os.Getenv("ROLEPLAY_SCENE_MAX_REQUEST_BYTES")); v != "" {
		if parsed, parseErr := parseInt64(v); parseErr == nil && parsed > 0 {
			maxSize = parsed
		}
	}
	if perFile := RolePlaySceneMaxUploadBytes(); maxSize < perFile {
		return perFile
	}
	return maxSize
}
