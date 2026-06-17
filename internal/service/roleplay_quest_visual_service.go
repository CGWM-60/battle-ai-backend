package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"

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
	SceneKey            string
	ChapterIndex        int
	ArcIndex            int
	Title               string
	Summary             string
	Prompt              string
	ImagePrompt         string
	NegativePrompt      string
	SceneType           string
	RoomType            string
	Atmosphere          string
	DangerLevel         string
	VisualTags          []string
	RpgMetadata         map[string]any
}

type BackfillImagePromptsInput struct {
	OnlyMissing       bool `json:"onlyMissing"`
	ForceRegenerate   bool `json:"forceRegenerate"`
	SceneCount        int  `json:"sceneCount"`
}

type BackfillImagePromptsResult struct {
	UpdatedQuests  int      `json:"updatedQuests"`
	CreatedScenes  int      `json:"createdScenes"`
	UpdatedPrompts int      `json:"updatedPrompts"`
	Errors         []string `json:"errors"`
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
			QuestID:             questID,
			SceneKey:            defaultString(input.SceneKey, fmt.Sprintf("scene_%02d", input.ChapterIndex)),
			ChapterIndex:        input.ChapterIndex,
			ArcIndex:            input.ArcIndex,
			Title:               input.Title,
			Summary:             input.Summary,
			Prompt:              input.Prompt,
			ImagePrompt:         input.ImagePrompt,
			NegativePrompt:      input.NegativePrompt,
			SceneType:           defaultString(input.SceneType, "exploration"),
			RoomType:            defaultString(input.RoomType, "generic"),
			Atmosphere:          defaultString(input.Atmosphere, "mysterious"),
			DangerLevel:         defaultString(input.DangerLevel, "medium"),
			VisualTags:          datatypes.JSON(tags),
			RpgMetadata:         datatypes.JSON(rpg),
			ImageStatus:         "prompt_only",
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
		scenes = buildDefaultScenes(theme, level, title, summary)
	}
	_, err := s.CreateScenes(ctx, questID, scenes)
	return err
}

func (s *RolePlayQuestVisualService) BackfillImagePrompts(ctx context.Context, input BackfillImagePromptsInput) (*BackfillImagePromptsResult, error) {
	sceneCount := input.SceneCount
	if sceneCount <= 0 {
		sceneCount = 3
	}
	result := &BackfillImagePromptsResult{Errors: []string{}}
	var quests []models.RolePlayQuestTemplate
	if err := s.db.WithContext(ctx).Order("id ASC").Find(&quests).Error; err != nil {
		return nil, err
	}
	for _, quest := range quests {
		updated := false
		imagePrompt := strings.TrimSpace(quest.ImagePrompt)
		if imagePrompt == "" || input.ForceRegenerate {
			imagePrompt = buildQuestImagePrompt(quest.Theme, quest.Level, quest.Title, quest.Summary)
			neg := buildDefaultNegativePrompt()
			tags := buildDefaultVisualTags(quest.Theme)
			rpg := buildDefaultRpgMetadata(quest.Level, quest.Theme)
			tagsJSON, _ := json.Marshal(tags)
			rpgJSON, _ := json.Marshal(rpg)
			if err := s.db.WithContext(ctx).Model(&models.RolePlayQuestTemplate{}).Where("id = ?", quest.Id).Updates(map[string]any{
				"image_prompt":          imagePrompt,
				"image_negative_prompt": neg,
				"visual_style":          "dark fantasy mobile RPG",
				"visual_tags":           datatypes.JSON(tagsJSON),
				"rpg_metadata":          datatypes.JSON(rpgJSON),
			}).Error; err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("quest %d: %v", quest.Id, err))
				continue
			}
			result.UpdatedPrompts++
			updated = true
		}
		var existingCount int64
		_ = s.db.WithContext(ctx).Model(&models.RolePlayQuestScene{}).Where("quest_id = ?", quest.Id).Count(&existingCount).Error
		if existingCount == 0 {
			scenes := buildDefaultScenes(quest.Theme, quest.Level, quest.Title, quest.Summary)
			if sceneCount < len(scenes) {
				scenes = scenes[:sceneCount]
			}
			count, err := s.CreateScenes(ctx, quest.Id, scenes)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("quest %d scenes: %v", quest.Id, err))
				continue
			}
			result.CreatedScenes += count
			updated = true
		} else if !input.OnlyMissing {
			var scenes []models.RolePlayQuestScene
			_ = s.db.WithContext(ctx).Where("quest_id = ?", quest.Id).Find(&scenes).Error
			for _, scene := range scenes {
				if strings.TrimSpace(scene.ImagePrompt) != "" && !input.ForceRegenerate {
					continue
				}
				prompt := buildSceneImagePrompt(quest.Theme, scene.RoomType, scene.Title, scene.Summary)
				if err := s.db.WithContext(ctx).Model(&models.RolePlayQuestScene{}).Where("id = ?", scene.Id).Updates(map[string]any{
					"image_prompt":    prompt,
					"negative_prompt": buildDefaultNegativePrompt(),
				}).Error; err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("scene %d: %v", scene.Id, err))
					continue
				}
				result.UpdatedPrompts++
				updated = true
			}
		}
		if updated {
			result.UpdatedQuests++
		}
	}
	return result, nil
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
	size := int64(len(data))
	maxSize := int64(8 * 1024 * 1024)
	if v := strings.TrimSpace(os.Getenv("ROLEPLAY_SCENE_MAX_BYTES")); v != "" {
		if parsed, parseErr := parseInt64(v); parseErr == nil && parsed > 0 {
			maxSize = parsed
		}
	}
	if size > maxSize {
		return nil, fmt.Errorf("image file is too large")
	}
	ext := strings.ToLower(filepath.Ext(originalName))
	if !allowedAssetExt(ext) {
		return nil, fmt.Errorf("unsupported image format")
	}
	assetDir := RolePlayScenePublicDir()
	publicBase := RolePlayScenePublicBaseURL()
	relDir := filepath.Join(fmt.Sprintf("%d", questID), "scenes", fmt.Sprintf("%d", sceneID))
	fullDir := filepath.Join(assetDir, relDir)
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		return nil, err
	}
	hasher := sha256.New()
	_, _ = hasher.Write(data)
	filename := fmt.Sprintf("scene_%d_%d%s", sceneID, time.Now().UnixNano(), ext)
	fullPath := filepath.Join(fullDir, filename)
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return nil, err
	}
	relURL := filepath.ToSlash(filepath.Join(relDir, filename))
	url := publicBase + "/" + relURL
	image := &models.RolePlayQuestSceneImage{
		SceneID:    sceneID,
		QuestID:    questID,
		URL:        url,
		StorageKey: relURL,
		Filename:   filename,
		MimeType:   mimeFromExt(ext),
		Size:       size,
		IsMain:     true,
		Alt:        defaultString(alt, scene.Title),
		Source:     "admin_upload",
	}
	return image, s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.RolePlayQuestSceneImage{}).
			Where("scene_id = ?", sceneID).
			Update("is_main", false).Error; err != nil {
			return err
		}
		if err := tx.Create(image).Error; err != nil {
			_ = os.Remove(fullPath)
			return err
		}
		return tx.Model(&models.RolePlayQuestScene{}).Where("id = ?", sceneID).Updates(map[string]any{
			"image_url":         url,
			"image_storage_key": relURL,
			"image_status":      "uploaded",
			"image_alt":         image.Alt,
		}).Error
	})
}

func (s *RolePlayQuestVisualService) DeleteSceneImage(ctx context.Context, questID, sceneID, imageID uint) error {
	var image models.RolePlayQuestSceneImage
	if err := s.db.WithContext(ctx).
		Where("id = ? AND scene_id = ? AND quest_id = ?", imageID, sceneID, questID).
		First(&image).Error; err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&image).Error; err != nil {
			return err
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
			ImagePrompt: buildSceneImagePrompt(theme, "entrance", "Entrée", summary),
			NegativePrompt: buildDefaultNegativePrompt(),
			VisualTags: []string{"entrance", "fog"},
		},
		{
			SceneKey: "scene_02_conflict", ChapterIndex: 2, ArcIndex: 1,
			Title: "Conflit", Summary: "Tension et exploration",
			SceneType: "exploration", RoomType: "corridor", Atmosphere: "tense", DangerLevel: "medium",
			ImagePrompt: buildSceneImagePrompt(theme, "corridor", "Conflit", summary),
			NegativePrompt: buildDefaultNegativePrompt(),
			VisualTags: []string{"corridor", "shadows"},
		},
		{
			SceneKey: "scene_03_climax", ChapterIndex: 3, ArcIndex: 1,
			Title: "Climax", Summary: "Révélation ou confrontation",
			SceneType: "boss", RoomType: "boss_room", Atmosphere: "dramatic", DangerLevel: levelDanger(level),
			ImagePrompt: buildSceneImagePrompt(theme, "boss_room", "Climax", summary),
			NegativePrompt: buildDefaultNegativePrompt(),
			VisualTags: []string{"boss", "dramatic_light"},
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