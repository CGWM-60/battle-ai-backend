package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"cgwm/battle/internal/models"
)

const (
	SceneModePerChapter = "per_chapter"
	SceneModeMinimum3   = "minimum_3"
	SceneModePerArc     = "per_arc"
)

func normalizeSceneMode(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case SceneModeMinimum3, SceneModePerArc:
		return strings.TrimSpace(strings.ToLower(mode))
	default:
		return SceneModePerChapter
	}
}

func countQuestChapters(quest models.RolePlayQuestTemplate) int {
	total := 0
	for _, arc := range quest.Arcs {
		total += len(arc.Chapters)
	}
	return total
}

func scenesCoverAllChapters(scenes []RolePlaySceneInput, quest *models.RolePlayQuestTemplate) bool {
	if quest == nil {
		return false
	}
	expected := countQuestChapters(*quest)
	if expected == 0 || len(scenes) < expected {
		return false
	}
	covered := map[uint]bool{}
	for _, scene := range scenes {
		if scene.ChapterID != nil && *scene.ChapterID > 0 {
			covered[*scene.ChapterID] = true
		}
	}
	for _, arc := range quest.Arcs {
		for _, chapter := range arc.Chapters {
			if !covered[chapter.Id] {
				return false
			}
		}
	}
	return true
}

func BuildScenesFromQuestStructure(quest models.RolePlayQuestTemplate) []RolePlaySceneInput {
	if len(quest.Arcs) == 0 {
		return buildDefaultScenes(quest.Theme, quest.Level, quest.Title, quest.Summary)
	}

	outputs := make([]RolePlaySceneInput, 0, countQuestChapters(quest))
	globalChapterIndex := 0

	for arcIdx, arc := range quest.Arcs {
		arcIndex := arc.Position
		if arcIndex <= 0 {
			arcIndex = arcIdx + 1
		}
		arcID := arc.Id

		for chapIdx, chapter := range arc.Chapters {
			globalChapterIndex++
			chapterPosition := chapter.Position
			if chapterPosition <= 0 {
				chapterPosition = chapIdx + 1
			}
			chapterID := chapter.Id

			summary := strings.TrimSpace(chapter.Summary)
			if summary == "" {
				summary = strings.TrimSpace(chapter.Objective)
			}
			prompt := strings.TrimSpace(chapter.IntroPrompt)
			if prompt == "" {
				prompt = strings.TrimSpace(chapter.Objective)
			}

			meta := decodeChapterMetadata(chapter.Metadata)
			sceneType := stringFromMetadata(meta, "sceneType")
			if sceneType == "" {
				sceneType = inferChapterSceneType(chapter)
			}
			roomType := stringFromMetadata(meta, "roomType")
			if roomType == "" {
				roomType = inferChapterRoomType(quest.Theme, chapter, globalChapterIndex)
			}
			atmosphere := stringFromMetadata(meta, "atmosphere")
			if atmosphere == "" {
				atmosphere = inferChapterAtmosphere(sceneType)
			}
			dangerLevel := stringFromMetadata(meta, "dangerLevel")
			if dangerLevel == "" {
				dangerLevel = inferChapterDangerLevel(quest.Level, chapter.IsBoss)
			}
			imagePrompt := stringFromMetadata(meta, "imagePrompt")
			if imagePrompt == "" {
				imagePrompt = buildSceneImagePrompt(quest.Theme, roomType, chapter.Title, summary)
			}
			negativePrompt := stringFromMetadata(meta, "imageNegativePrompt")
			if negativePrompt == "" {
				negativePrompt = buildDefaultNegativePrompt()
			}
			visualTags := stringSliceFromMetadata(meta, "visualTags")
			if len(visualTags) == 0 {
				visualTags = buildDefaultVisualTags(quest.Theme)
			}

			arcIDCopy := arcID
			chapterIDCopy := chapterID

			outputs = append(outputs, RolePlaySceneInput{
				SceneKey:       fmt.Sprintf("arc_%02d_chapter_%02d", arcIndex, chapterPosition),
				ArcID:          &arcIDCopy,
				ChapterID:      &chapterIDCopy,
				ArcIndex:       arcIndex,
				ChapterIndex:   globalChapterIndex,
				Title:          chapter.Title,
				Summary:        summary,
				Prompt:         prompt,
				ImagePrompt:    imagePrompt,
				NegativePrompt: negativePrompt,
				SceneType:      sceneType,
				RoomType:       roomType,
				Atmosphere:     atmosphere,
				DangerLevel:    dangerLevel,
				VisualTags:     visualTags,
			})
		}
	}
	return outputs
}

func BuildScenesForQuestMode(quest models.RolePlayQuestTemplate, sceneMode string, sceneCount int) []RolePlaySceneInput {
	mode := normalizeSceneMode(sceneMode)
	switch mode {
	case SceneModeMinimum3:
		scenes := buildDefaultScenes(quest.Theme, quest.Level, quest.Title, quest.Summary)
		if sceneCount > 0 && sceneCount < len(scenes) {
			return scenes[:sceneCount]
		}
		return scenes
	case SceneModePerArc:
		outputs := make([]RolePlaySceneInput, 0, len(quest.Arcs))
		for arcIdx, arc := range quest.Arcs {
			arcIndex := arc.Position
			if arcIndex <= 0 {
				arcIndex = arcIdx + 1
			}
			arcIDCopy := arc.Id
			summary := strings.TrimSpace(arc.Summary)
			if summary == "" {
				summary = strings.TrimSpace(arc.Objective)
			}
			outputs = append(outputs, RolePlaySceneInput{
				SceneKey:       fmt.Sprintf("arc_%02d", arcIndex),
				ArcID:          &arcIDCopy,
				ArcIndex:       arcIndex,
				ChapterIndex:   arcIndex,
				Title:          arc.Title,
				Summary:        summary,
				Prompt:         strings.TrimSpace(arc.Prompt),
				ImagePrompt:    buildSceneImagePrompt(quest.Theme, "generic", arc.Title, summary),
				NegativePrompt: buildDefaultNegativePrompt(),
				SceneType:      "exploration",
				RoomType:       "generic",
				Atmosphere:     "mysterious",
				DangerLevel:    levelDanger(quest.Level),
				VisualTags:     buildDefaultVisualTags(quest.Theme),
			})
		}
		if sceneCount > 0 && sceneCount < len(outputs) {
			return outputs[:sceneCount]
		}
		return outputs
	default:
		scenes := BuildScenesFromQuestStructure(quest)
		if sceneCount > 0 && sceneCount < len(scenes) {
			return scenes[:sceneCount]
		}
		return scenes
	}
}

func missingChapterSceneInputs(
	quest models.RolePlayQuestTemplate,
	existing []models.RolePlayQuestScene,
) []RolePlaySceneInput {
	desired := BuildScenesFromQuestStructure(quest)
	if len(desired) == 0 {
		return nil
	}

	byChapterID := map[uint]bool{}
	byPosition := map[string]bool{}
	for _, scene := range existing {
		if scene.ChapterID != nil && *scene.ChapterID > 0 {
			byChapterID[*scene.ChapterID] = true
		}
		key := fmt.Sprintf("%d:%d", scene.ArcIndex, scene.ChapterIndex)
		byPosition[key] = true
	}

	missing := make([]RolePlaySceneInput, 0)
	for _, input := range desired {
		if input.ChapterID != nil && byChapterID[*input.ChapterID] {
			continue
		}
		key := fmt.Sprintf("%d:%d", input.ArcIndex, input.ChapterIndex)
		if byPosition[key] {
			continue
		}
		missing = append(missing, input)
	}
	return missing
}

func inferChapterSceneType(chapter models.RolePlayQuestChapter) string {
	if chapter.IsBoss {
		return "boss"
	}
	objective := strings.ToLower(strings.TrimSpace(chapter.Objective))
	if strings.Contains(objective, "combat") ||
		strings.Contains(objective, "affront") ||
		strings.Contains(objective, "confront") {
		return "combat"
	}
	return "exploration"
}

func inferChapterRoomType(theme string, chapter models.RolePlayQuestChapter, globalIndex int) string {
	if chapter.IsBoss {
		return "boss_room"
	}
	if globalIndex == 1 {
		return "entrance"
	}
	lower := strings.ToLower(strings.TrimSpace(chapter.Objective + " " + chapter.Summary + " " + theme))
	switch {
	case strings.Contains(lower, "forêt"), strings.Contains(lower, "forest"):
		return "forest"
	case strings.Contains(lower, "ville"), strings.Contains(lower, "city"):
		return "city"
	case strings.Contains(lower, "temple"):
		return "temple"
	case strings.Contains(lower, "station"), strings.Contains(lower, "orbital"):
		return "station"
	case strings.Contains(lower, "crypte"), strings.Contains(lower, "crypt"):
		return "crypt"
	case strings.Contains(lower, "ruine"), strings.Contains(lower, "ruin"):
		return "ruins"
	default:
		return "corridor"
	}
}

func inferChapterAtmosphere(sceneType string) string {
	switch sceneType {
	case "boss":
		return "dramatic"
	case "combat":
		return "tense"
	default:
		return "mysterious"
	}
}

func decodeChapterMetadata(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var meta map[string]any
	if err := json.Unmarshal(raw, &meta); err != nil {
		return map[string]any{}
	}
	return meta
}

func stringFromMetadata(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	value, ok := meta[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", value))
}

func stringSliceFromMetadata(meta map[string]any, key string) []string {
	if meta == nil {
		return nil
	}
	value, ok := meta[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprintf("%v", item))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func inferChapterDangerLevel(level string, isBoss bool) string {
	if isBoss {
		return levelDanger(level)
	}
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "difficile":
		return "high"
	case "moyen":
		return "medium"
	default:
		return "low"
	}
}