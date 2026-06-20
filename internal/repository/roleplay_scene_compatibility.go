package repository

import (
	"strings"

	"cgwm/battle/internal/models"
)

// ApplyRolePlayQuestImageCompatibility preserves both scene key generations in
// API responses. Equivalent legacy/new scenes share their available visual,
// and the first scene visual becomes the quest cover when no explicit cover is
// configured. This lets Flutter render old and newly generated quests alike.
func ApplyRolePlayQuestImageCompatibility(quest *models.RolePlayQuestTemplate) {
	if quest == nil || len(quest.Scenes) == 0 {
		return
	}

	byChapterIndex := make(map[int][]int)
	for index := range quest.Scenes {
		scene := &quest.Scenes[index]
		ensureSceneImageURL(scene)
		if scene.ChapterIndex > 0 {
			byChapterIndex[scene.ChapterIndex] = append(byChapterIndex[scene.ChapterIndex], index)
		}
	}

	for _, indexes := range byChapterIndex {
		var source *models.RolePlayQuestScene
		for _, index := range indexes {
			candidate := &quest.Scenes[index]
			if rolePlaySceneImageURL(candidate) != "" {
				source = candidate
				break
			}
		}
		if source == nil {
			continue
		}
		for _, index := range indexes {
			applySceneImageFallback(&quest.Scenes[index], source)
		}
	}

	if strings.TrimSpace(quest.ImageURL) == "" {
		for index := range quest.Scenes {
			if imageURL := rolePlaySceneImageURL(&quest.Scenes[index]); imageURL != "" {
				quest.ImageURL = imageURL
				break
			}
		}
	}
}

func ensureSceneImageURL(scene *models.RolePlayQuestScene) {
	if scene == nil || strings.TrimSpace(scene.ImageURL) != "" {
		return
	}
	for _, image := range scene.Images {
		if image.IsMain && strings.TrimSpace(image.URL) != "" {
			scene.ImageURL = image.URL
			return
		}
	}
	for _, image := range scene.Images {
		if strings.TrimSpace(image.URL) != "" {
			scene.ImageURL = image.URL
			return
		}
	}
}

func rolePlaySceneImageURL(scene *models.RolePlayQuestScene) string {
	if scene == nil {
		return ""
	}
	ensureSceneImageURL(scene)
	return strings.TrimSpace(scene.ImageURL)
}

func applySceneImageFallback(target, source *models.RolePlayQuestScene) {
	if target == nil || source == nil || target == source || rolePlaySceneImageURL(target) != "" {
		return
	}
	target.ImageURL = rolePlaySceneImageURL(source)
	target.ImageAlt = source.ImageAlt
	target.ImageStatus = source.ImageStatus
	target.ImageStorageKey = source.ImageStorageKey
	if len(target.Images) == 0 && len(source.Images) > 0 {
		target.Images = append([]models.RolePlayQuestSceneImage(nil), source.Images...)
	}
}

func ApplyRolePlaySessionImageCompatibility(session *models.RolePlaySession) {
	if session == nil {
		return
	}
	for runIndex := range session.QuestRuns {
		ApplyRolePlayQuestImageCompatibility(session.QuestRuns[runIndex].Template)
	}
}

func ApplyCoopPartyImageCompatibility(party *models.CoopParty) {
	if party != nil {
		ApplyRolePlaySessionImageCompatibility(party.RolePlaySession)
	}
}

func ApplyLiveSessionImageCompatibility(session *models.LiveSession) {
	if session == nil {
		return
	}
	ApplyRolePlaySessionImageCompatibility(session.RolePlaySession)
	if session.CoopParty != nil {
		ApplyCoopPartyImageCompatibility(session.CoopParty)
	}
}
