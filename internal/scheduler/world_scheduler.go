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

	light := worldSimulationInterval("WORLD_SIMULATION_LIGHT_INTERVAL_MINUTES", 15, time.Minute)
	hourly := worldSimulationInterval("WORLD_SIMULATION_CONTINENT_INTERVAL_MINUTES", 60, time.Minute)
	daily := worldSimulationInterval("WORLD_SIMULATION_DAILY_INTERVAL_HOURS", 24, time.Hour)
	log.Printf("[world-sim] step=boot status=enabled light=%s continental=%s daily=%s", light, hourly, daily)
	go runWorldSimulationLoop(db, service.SimulationCycleLight, light)
	go runWorldSimulationLoop(db, service.SimulationCycleHourly, hourly)
	go runWorldSimulationLoop(db, service.SimulationCycleDaily, daily)
}

func runWorldSimulationLoop(db *gorm.DB, cycleType string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), worldSimulationTimeout())
		_, err := service.NewWorldGameService(db).SimulateWorldCycle(ctx, 0, "cron-"+cycleType, cycleType)
		cancel()
		if err != nil {
			log.Printf("[world-sim] step=run cycle=%s status=failed err=%v", cycleType, err)
			continue
		}
		log.Printf("[world-sim] step=run cycle=%s status=completed", cycleType)
	}
}

func worldSimulationInterval(envName string, fallback int, unit time.Duration) time.Duration {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(envName)))
	if err != nil || value <= 0 {
		value = fallback
	}
	return time.Duration(value) * unit
}

func worldSimulationTimeout() time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(os.Getenv("WORLD_SIMULATION_TIMEOUT_SECONDS")))
	if err != nil || seconds <= 0 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}
