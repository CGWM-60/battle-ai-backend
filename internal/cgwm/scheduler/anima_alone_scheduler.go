package scheduler

import (
	"log"
	"time"
	"cgwm/battle/internal/cgwm/service"
)

// AnimaAloneLearningScheduler runs every X minutes for Animas left alone in the park.
// Only processes consented Animas, produces safe anonymized social cards via the privacy filter.
// No private data ever leaves the system.

func StartAloneLearningScheduler() {
	ticker := time.NewTicker(5 * time.Minute) // configurable
	go func() {
		for range ticker.C {
			err := service.RunAloneParkLearningTick()
			if err != nil {
				log.Printf("[CGWM Scheduler] alone learning tick error: %v", err)
			}
		}
	}()
	log.Println("[CGWM] Alone-in-park learning scheduler started")
}