package repository

import (
	"testing"

	"cgwm/battle/internal/models"
)

func TestApplyRolePlayQuestImageCompatibilityBridgesLegacyAndModernScenes(t *testing.T) {
	legacyURL := "/uploads/roleplay/quests/9/scenes/1/entry.webp"
	quest := models.RolePlayQuestTemplate{
		Scenes: []models.RolePlayQuestScene{
			{
				Id:           1,
				SceneKey:     "scene_01_entry",
				ChapterIndex: 1,
				ImageURL:     legacyURL,
				Images: []models.RolePlayQuestSceneImage{{
					Id: 4, SceneID: 1, URL: legacyURL, IsMain: true,
				}},
			},
			{Id: 2, SceneKey: "arc_01_chapter_01", ChapterIndex: 1},
		},
	}

	ApplyRolePlayQuestImageCompatibility(&quest)

	if quest.ImageURL != legacyURL {
		t.Fatalf("expected quest cover fallback %q, got %q", legacyURL, quest.ImageURL)
	}
	if quest.Scenes[1].ImageURL != legacyURL {
		t.Fatalf("expected modern scene image fallback %q, got %q", legacyURL, quest.Scenes[1].ImageURL)
	}
	if len(quest.Scenes[1].Images) != 1 || quest.Scenes[1].Images[0].URL != legacyURL {
		t.Fatal("expected legacy image collection in modern scene response")
	}
}

func TestApplyRolePlayQuestImageCompatibilityUsesMainImageAsCover(t *testing.T) {
	mainURL := "/uploads/roleplay/quests/9/scenes/1/main.webp"
	quest := models.RolePlayQuestTemplate{
		Scenes: []models.RolePlayQuestScene{{
			SceneKey: "arc_01_chapter_01", ChapterIndex: 1,
			Images: []models.RolePlayQuestSceneImage{{URL: mainURL, IsMain: true}},
		}},
	}

	ApplyRolePlayQuestImageCompatibility(&quest)

	if quest.ImageURL != mainURL || quest.Scenes[0].ImageURL != mainURL {
		t.Fatalf("expected main image to feed scene and quest URLs, got quest=%q scene=%q", quest.ImageURL, quest.Scenes[0].ImageURL)
	}
}
