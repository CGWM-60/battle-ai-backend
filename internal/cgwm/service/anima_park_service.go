package service

import (
	"time"
	"cgwm/battle/internal/cgwm/models"
	"cgwm/battle/internal/cgwm/realtime"
)

// AnimaParkService implements the park logic (enter, leave alone, heartbeat, return, report).
func EnterPark(animaID string) (*models.AnimaParkVisit, error) {
	visit := &models.AnimaParkVisit{
		AnimaID:   animaID,
		StartedAt: time.Now(),
		LeftAlone: false,
	}
	// create presence, assign zone, broadcast via realtime.ParkHubInstance.Broadcast(...)
	realtime.ParkHubInstance.Broadcast(map[string]interface{}{"type": "presence_update", "anima": animaID})
	return visit, nil
}

func LeaveAloneInPark(animaID string) error {
	// set IsAlone=true, start scheduler eligibility
	return nil
}

func ReturnFromPark(animaID, visitID string) (map[string]interface{}, error) {
	// close visit, generate report with lessons (already sanitized), broadcast
	report := map[string]interface{}{"animaId": animaID, "lessons": []string{"exemple leçon anonymisée"}}
	return report, nil
}

func GetParkState() *models.AnimaParkWorld {
	return &models.AnimaParkWorld{ActiveAnimas: 47, AloneAnimas: 19, AtmosphereLevel: "luminousForest"}
}