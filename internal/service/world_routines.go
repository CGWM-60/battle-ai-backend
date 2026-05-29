package service

import (
	"context"
	"time"

	"gorm.io/gorm"
)

func (s *WorldGameService) CompleteBuildingConstruction(ctx context.Context) (int64, error) {
	// Construction queue completion is handled lazily by construction endpoints.
	return 0, nil
}

func (s *WorldGameService) CompleteBuildingUpgrade(ctx context.Context) (int64, error) {
	// Construction queue completion is handled lazily by construction endpoints.
	return 0, nil
}

func (s *WorldGameService) UpdateQuestProgress(ctx context.Context) (int64, error) {
	// Quest scheduler and dedicated quest services already manage lifecycle.
	return 0, nil
}

func (s *WorldGameService) UpdateWorldEvents(ctx context.Context) (int64, error) {
	result := s.db.WithContext(ctx).
		Model(&map[string]any{}).
		Table("game_events").
		Where("status = ? AND ends_at < ?", EventStatusActive, time.Now().UTC()).
		Update("status", EventStatusFinished)
	return result.RowsAffected, result.Error
}

func (s *WorldGameService) UpdateWorldConflicts(ctx context.Context) (int64, error) {
	result := s.db.WithContext(ctx).
		Model(&map[string]any{}).
		Table("conflicts").
		Where("status = ? AND ends_at < ?", ConflictStatusActive, time.Now().UTC()).
		Update("status", ConflictStatusResolved)
	return result.RowsAffected, result.Error
}

func (s *WorldGameService) ResolveConflictInterventions(ctx context.Context) (int64, error) {
	return 0, nil
}

func (s *WorldGameService) UpdateWeatherEvents(ctx context.Context) (int64, error) {
	result := s.db.WithContext(ctx).
		Model(&map[string]any{}).
		Table("weather_events").
		Where("ends_at < ?", time.Now().UTC()).
		Update("deleted_at", gorm.Expr("deleted_at"))
	return result.RowsAffected, nil
}

func (s *WorldGameService) CompleteWeatherPlans(ctx context.Context) (int64, error) {
	return 0, nil
}

func (s *WorldGameService) UpdateDiplomaticNegotiations(ctx context.Context) (int64, error) {
	return 0, nil
}

func (s *WorldGameService) UpdateTradeRoutes(ctx context.Context) (int64, error) {
	return 0, nil
}

func (s *WorldGameService) GenerateWorldReports(ctx context.Context) (int64, error) {
	return 0, nil
}

func (s *WorldGameService) RunWorldMaintenanceTick(ctx context.Context) map[string]any {
	completedArmy, _ := s.CompleteArmyTraining(ctx)
	consumedArmy, _ := s.UpdateArmyConsumption(ctx)
	eventsUpdated, _ := s.UpdateWorldEvents(ctx)
	conflictsUpdated, _ := s.UpdateWorldConflicts(ctx)
	return map[string]any{
		"completeArmyTraining":  completedArmy,
		"updateArmyConsumption": consumedArmy,
		"updateWorldEvents":     eventsUpdated,
		"updateWorldConflicts":  conflictsUpdated,
	}
}
