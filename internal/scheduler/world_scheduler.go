package scheduler

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"cgwm/battle/internal/service"

	"gorm.io/gorm"
)

type WorldCronSnapshot struct {
	Enabled             bool   `json:"enabled"`
	Started             bool   `json:"started"`
	LightInterval       string `json:"lightInterval"`
	ContinentalInterval string `json:"continentalInterval"`
	DailyInterval       string `json:"dailyInterval"`
	RoutineInterval     string `json:"routineInterval"`
	LastChangedBy       string `json:"lastChangedBy"`
	LastChangedAt       string `json:"lastChangedAt"`
	LastRunType         string `json:"lastRunType"`
	LastRunStatus       string `json:"lastRunStatus"`
	LastRunAt           string `json:"lastRunAt"`
	LastError           string `json:"lastError"`
}

var worldCronState = struct {
	sync.RWMutex
	snapshot WorldCronSnapshot
}{
	snapshot: WorldCronSnapshot{},
}

func StartWorldSimulationCron(db *gorm.DB) {
	light := worldSimulationInterval("WORLD_SIMULATION_LIGHT_INTERVAL_MINUTES", 15, time.Minute)
	hourly := worldSimulationInterval("WORLD_SIMULATION_CONTINENT_INTERVAL_MINUTES", 60, time.Minute)
	daily := worldSimulationInterval("WORLD_SIMULATION_DAILY_INTERVAL_HOURS", 24, time.Hour)
	routine := worldSimulationInterval("WORLD_ROUTINE_4_PAGES_INTERVAL_SECONDS", 60, time.Second)
	enabled := !strings.EqualFold(os.Getenv("WORLD_SIMULATION_CRON_ENABLED"), "false")
	setWorldCronRuntime(enabled, "boot", light, hourly, daily, routine)
	if !enabled {
		log.Printf("[world-sim] step=boot status=disabled reason=WORLD_SIMULATION_CRON_ENABLED=false loops=paused")
	} else {
		log.Printf("[world-sim] step=boot status=enabled light=%s continental=%s daily=%s routine4pages=%s", light, hourly, daily, routine)
	}
	go runWorldSimulationLoop(db, service.SimulationCycleLight, light)
	go runWorldSimulationLoop(db, service.SimulationCycleHourly, hourly)
	go runWorldSimulationLoop(db, service.SimulationCycleDaily, daily)
	go runWorldRoutineLoop(db, routine)
}

func runWorldSimulationLoop(db *gorm.DB, cycleType string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		if !WorldSimulationCronEnabled() {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), worldSimulationTimeout())
		worldService := service.NewWorldGameService(db)
		_, err := worldService.SimulateWorldCycle(ctx, 0, "cron-"+cycleType, cycleType)
		if err == nil {
			_ = worldService.RunWorldMaintenanceTick(ctx)
		}
		cancel()
		if err != nil {
			recordWorldCronRun(cycleType, "failed", err.Error())
			log.Printf("[world-sim] step=run cycle=%s status=failed err=%v", cycleType, err)
			continue
		}
		recordWorldCronRun(cycleType, "completed", "")
		log.Printf("[world-sim] step=run cycle=%s status=completed", cycleType)
	}
}

func runWorldRoutineLoop(db *gorm.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		if !WorldSimulationCronEnabled() {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), worldSimulationTimeout())
		var worlds []struct {
			Id uint
		}
		err := db.WithContext(ctx).Model(&struct {
			Id uint
		}{}).Table("worlds").Select("id").Where("status = ?", service.WorldStatusActive).Find(&worlds).Error
		if err == nil {
			worldService := service.NewWorldGameService(db)
			failedWorlds := 0
			for _, world := range worlds {
				worldCtx, worldCancel := context.WithTimeout(context.Background(), worldRoutinePerWorldTimeout())
				_, _, worldErr := worldService.GenerateWorldFourPageRoutine(worldCtx, world.Id, "cron-routine-4-pages")
				worldCancel()
				if worldErr != nil {
					failedWorlds++
					recordWorldCronRun("routine_4_pages", "failed", worldErr.Error())
					log.Printf("[world-sim] step=routine4pages world_id=%d status=failed err=%v", world.Id, worldErr)
				}
			}
			if failedWorlds == 0 {
				err = nil
			} else {
				err = nil
				log.Printf("[world-sim] step=routine4pages status=partial worlds=%d failed=%d", len(worlds), failedWorlds)
			}
		}
		cancel()
		if err != nil {
			recordWorldCronRun("routine_4_pages", "failed", err.Error())
			log.Printf("[world-sim] step=routine4pages status=failed err=%v", err)
			continue
		}
		recordWorldCronRun("routine_4_pages", "completed", "")
		log.Printf("[world-sim] step=routine4pages status=completed worlds=%d", len(worlds))
	}
}

func WorldSimulationCronEnabled() bool {
	worldCronState.RLock()
	defer worldCronState.RUnlock()
	return worldCronState.snapshot.Enabled
}

func WorldSimulationCronSnapshot() WorldCronSnapshot {
	worldCronState.RLock()
	defer worldCronState.RUnlock()
	return worldCronState.snapshot
}

func SetWorldSimulationCronEnabled(enabled bool, changedBy string) WorldCronSnapshot {
	worldCronState.Lock()
	defer worldCronState.Unlock()
	worldCronState.snapshot.Enabled = enabled
	worldCronState.snapshot.LastChangedBy = changedBy
	worldCronState.snapshot.LastChangedAt = time.Now().UTC().Format(time.RFC3339)
	return worldCronState.snapshot
}

func setWorldCronRuntime(enabled bool, changedBy string, light time.Duration, hourly time.Duration, daily time.Duration, routine time.Duration) {
	worldCronState.Lock()
	defer worldCronState.Unlock()
	worldCronState.snapshot.Enabled = enabled
	worldCronState.snapshot.Started = true
	worldCronState.snapshot.LightInterval = light.String()
	worldCronState.snapshot.ContinentalInterval = hourly.String()
	worldCronState.snapshot.DailyInterval = daily.String()
	worldCronState.snapshot.RoutineInterval = routine.String()
	worldCronState.snapshot.LastChangedBy = changedBy
	worldCronState.snapshot.LastChangedAt = time.Now().UTC().Format(time.RFC3339)
}

func recordWorldCronRun(runType string, status string, errorMessage string) {
	worldCronState.Lock()
	defer worldCronState.Unlock()
	worldCronState.snapshot.LastRunType = runType
	worldCronState.snapshot.LastRunStatus = status
	worldCronState.snapshot.LastRunAt = time.Now().UTC().Format(time.RFC3339)
	worldCronState.snapshot.LastError = errorMessage
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
		seconds = 120
	}
	return time.Duration(seconds) * time.Second
}

func worldRoutinePerWorldTimeout() time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(os.Getenv("WORLD_ROUTINE_PER_WORLD_TIMEOUT_SECONDS")))
	if err != nil || seconds <= 0 {
		seconds = 45
	}
	return time.Duration(seconds) * time.Second
}
