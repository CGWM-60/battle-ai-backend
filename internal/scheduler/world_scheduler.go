package scheduler

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"cgwm/battle/internal/service"

	"gorm.io/gorm"
)

func StartWorldSimulationCron(db *gorm.DB) {
	if strings.EqualFold(os.Getenv("WORLD_SIMULATION_CRON_ENABLED"), "false") {
		log.Printf("[world-sim] step=boot status=disabled reason=WORLD_SIMULATION_CRON_ENABLED=false")
		return
	}

	interval := worldSimulationInterval()
	log.Printf("[world-sim] step=boot status=enabled interval=%s", interval)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), worldSimulationTimeout())
			_, err := service.NewWorldGameService(db).SimulateWorld(ctx, 0, "cron")
			cancel()
			if err != nil {
				log.Printf("[world-sim] step=run status=failed err=%v", err)
				continue
			}
			log.Printf("[world-sim] step=run status=completed")
		}
	}()
}

func worldSimulationInterval() time.Duration {
	minutes, err := strconv.Atoi(strings.TrimSpace(os.Getenv("WORLD_SIMULATION_INTERVAL_MINUTES")))
	if err != nil || minutes <= 0 {
		minutes = 15
	}
	return time.Duration(minutes) * time.Minute
}

func worldSimulationTimeout() time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(os.Getenv("WORLD_SIMULATION_TIMEOUT_SECONDS")))
	if err != nil || seconds <= 0 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}
