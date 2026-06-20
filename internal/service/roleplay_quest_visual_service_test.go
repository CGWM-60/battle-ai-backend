package service

import (
	"testing"

	"cgwm/battle/internal/models"
)

func TestIsQuestVisualComplete(t *testing.T) {
	completeQuest := models.RolePlayQuestTemplate{
		ImagePrompt:         "prompt",
		ImageNegativePrompt: "negative",
	}
	completeScenes := []models.RolePlayQuestScene{
		{ImagePrompt: "a", NegativePrompt: "b"},
	}
	if !isQuestVisualComplete(completeQuest, completeScenes) {
		t.Fatal("expected quest to be complete")
	}

	incompleteQuest := models.RolePlayQuestTemplate{ImagePrompt: "prompt"}
	if isQuestVisualComplete(incompleteQuest, completeScenes) {
		t.Fatal("expected missing negative prompt to be incomplete")
	}
}

func TestValidateRolePlayImageUploadRejectsInvalidMime(t *testing.T) {
	_, _, err := validateRolePlayImageUpload("test.txt", []byte("not-an-image"))
	if err == nil {
		t.Fatal("expected invalid mime rejection")
	}
}

func TestLegacySceneCoversNewChapterScene(t *testing.T) {
	chapterID := uint(42)
	quest := models.RolePlayQuestTemplate{
		Arcs: []models.RolePlayQuestArc{{
			Id:       7,
			Position: 1,
			Chapters: []models.RolePlayQuestChapter{{Id: chapterID, Position: 1, Title: "Entrée"}},
		}},
	}
	existing := []models.RolePlayQuestScene{{
		SceneKey:     "scene_01_entry",
		ArcIndex:     1,
		ChapterIndex: 1,
		ImageURL:     "/uploads/roleplay/quests/1/scenes/1/entry.webp",
	}}

	if missing := missingChapterSceneInputs(quest, existing); len(missing) != 0 {
		t.Fatalf("legacy scene should cover the new chapter structure, got %d missing", len(missing))
	}
}

func TestRolePlaySceneMatchSupportsBothKeyGenerations(t *testing.T) {
	chapterID := uint(42)
	input := RolePlaySceneInput{
		SceneKey:     "arc_01_chapter_01",
		ChapterID:    &chapterID,
		ArcIndex:     1,
		ChapterIndex: 1,
	}
	legacy := models.RolePlayQuestScene{SceneKey: "scene_01_entry", ChapterIndex: 1}
	modern := models.RolePlayQuestScene{SceneKey: "arc_01_chapter_01", ChapterIndex: 1}

	if rolePlaySceneMatchScore(legacy, input) <= 0 {
		t.Fatal("legacy scene key should match the chapter input")
	}
	if rolePlaySceneMatchScore(modern, input) <= rolePlaySceneMatchScore(legacy, input) {
		t.Fatal("exact modern key should be preferred over positional legacy match")
	}
}

func TestRolePlaySceneUploadBatchLimitExceedsPerFileLimit(t *testing.T) {
	t.Setenv("ROLEPLAY_SCENE_MAX_BYTES", "3145728")
	t.Setenv("ROLEPLAY_SCENE_MAX_REQUEST_BYTES", "33554432")

	if RolePlaySceneMaxUploadRequestBytes() <= RolePlaySceneMaxUploadBytes() {
		t.Fatal("multipart batch limit must allow several valid files")
	}
}
