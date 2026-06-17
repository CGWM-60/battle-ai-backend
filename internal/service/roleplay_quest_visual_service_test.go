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