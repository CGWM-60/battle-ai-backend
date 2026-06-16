package repository

import "cgwm/battle/internal/cgwm/models"

func SaveCloudSnapshot(s *models.AnimaCloudSnapshot) error { return nil }
func GetCloudSnapshot(id string) (*models.AnimaCloudSnapshot, error) { return &models.AnimaCloudSnapshot{}, nil }

// Add other repo funcs (GetParkState, SavePresence, SaveCard, etc.) as needed.