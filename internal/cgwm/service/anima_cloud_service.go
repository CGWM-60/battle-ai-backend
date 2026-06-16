package service

import (
	"cgwm/battle/internal/cgwm/models"
	"cgwm/battle/internal/cgwm/repository" // placeholder
)

func UploadSnapshot(snap *models.AnimaCloudSnapshot) error {
	// validate checksum, owner, schema
	// store in repository
	return repository.SaveCloudSnapshot(snap)
}

func DownloadSnapshot(animaID string) (*models.AnimaCloudSnapshot, error) {
	return repository.GetCloudSnapshot(animaID)
}

func RunSyncCleanup() {
	// periodic old snapshot / consent cleanup per retention policy
}