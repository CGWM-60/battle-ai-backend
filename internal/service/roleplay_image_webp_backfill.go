package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cgwm/battle/internal/models"
)

type ConvertExistingRolePlayImagesResult struct {
	Total     int      `json:"total"`
	Converted int      `json:"converted"`
	Skipped   int      `json:"skipped"`
	Failed    int      `json:"failed"`
	Errors    []string `json:"errors"`
}

func (s *RolePlayQuestVisualService) ConvertExistingImagesToWebP(ctx context.Context) (*ConvertExistingRolePlayImagesResult, error) {
	var images []models.RolePlayQuestSceneImage
	if err := s.db.WithContext(ctx).Order("id ASC").Find(&images).Error; err != nil {
		return nil, err
	}

	result := &ConvertExistingRolePlayImagesResult{
		Total:  len(images),
		Errors: []string{},
	}

	for _, image := range images {
		if strings.EqualFold(strings.TrimSpace(image.MimeType), rolePlayImageWebPMime) &&
			strings.HasSuffix(strings.ToLower(strings.TrimSpace(image.Filename)), ".webp") {
			result.Skipped++
			continue
		}

		storageKey := strings.TrimSpace(image.StorageKey)
		if storageKey == "" || strings.Contains(storageKey, "..") {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("image %d: invalid storage key", image.Id))
			continue
		}

		fullPath := filepath.Join(RolePlayScenePublicDir(), filepath.FromSlash(storageKey))
		input, err := os.ReadFile(fullPath)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("image %d: %v", image.Id, err))
			continue
		}

		converted, _, originalMime, err := ConvertUploadedImageBytesToWebP(input, image.Filename)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("image %d: %v", image.Id, err))
			continue
		}

		newFilename := strings.TrimSuffix(filepath.Base(image.Filename), filepath.Ext(image.Filename)) + ".webp"
		if !strings.HasSuffix(newFilename, ".webp") {
			newFilename = fmt.Sprintf("scene_%d_%d.webp", image.SceneID, image.Id)
		}
		newRel := filepath.ToSlash(filepath.Join(filepath.Dir(filepath.FromSlash(storageKey)), newFilename))
		newFullPath := filepath.Join(RolePlayScenePublicDir(), filepath.FromSlash(newRel))
		if err := os.WriteFile(newFullPath, converted.Data, 0o644); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("image %d: %v", image.Id, err))
			continue
		}

		publicBase := RolePlayScenePublicBaseURL()
		newURL := publicBase + "/" + newRel
		updates := map[string]any{
			"filename":           newFilename,
			"mime_type":          rolePlayImageWebPMime,
			"size":               len(converted.Data),
			"width":              converted.Width,
			"height":             converted.Height,
			"url":                newURL,
			"storage_key":        newRel,
			"original_filename":  image.Filename,
			"original_mime_type": originalMime,
		}
		if err := s.db.WithContext(ctx).Model(&models.RolePlayQuestSceneImage{}).Where("id = ?", image.Id).Updates(updates).Error; err != nil {
			_ = os.Remove(newFullPath)
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("image %d: %v", image.Id, err))
			continue
		}

		if newFullPath != fullPath {
			_ = os.Remove(fullPath)
		}

		var scene models.RolePlayQuestScene
		if err := s.db.WithContext(ctx).Where("id = ?", image.SceneID).First(&scene).Error; err == nil {
			if scene.ImageStorageKey == storageKey {
				_ = s.db.WithContext(ctx).Model(&scene).Updates(map[string]any{
					"image_url":         newURL,
					"image_storage_key": newRel,
					"image_status":      "uploaded",
				}).Error
			}
		}

		result.Converted++
	}

	return result, nil
}