package services

import (
	"cgwm/battle/internal/features"
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

var (
	serverAIJobSchedulerOnce sync.Once
	serverAIJobSchedulerMu   sync.Mutex
)

func StartServerAIJobScheduler(db *gorm.DB) {
	if db == nil {
		return
	}
	if !features.NexusGameEnabled() {
		log.Printf("[nexus-server-ai-cron] status=deprecated_disabled reason=NEXUS_GAME_ENABLED is not true")
		return
	}
	if strings.EqualFold(os.Getenv("NEXUS_SERVER_AI_CRON_ENABLED"), "false") {
		log.Printf("[nexus-server-ai-cron] status=disabled reason=NEXUS_SERVER_AI_CRON_ENABLED=false")
		return
	}
	interval := serverAIJobSchedulerInterval()
	serverAIJobSchedulerOnce.Do(func() {
		log.Printf("[nexus-server-ai-cron] status=started interval=%s", interval)
		go func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for range ticker.C {
				runServerAIJobSchedulerTick(db)
			}
		}()
	})
}

func serverAIJobSchedulerInterval() time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(os.Getenv("NEXUS_SERVER_AI_CRON_INTERVAL_SECONDS")))
	if err != nil || seconds <= 0 {
		seconds = 60
	}
	if seconds < 15 {
		seconds = 15
	}
	return time.Duration(seconds) * time.Second
}

func runServerAIJobSchedulerTick(db *gorm.DB) {
	if !serverAIJobSchedulerMu.TryLock() {
		log.Printf("[nexus-server-ai-cron] status=skipped reason=previous_tick_running")
		return
	}
	defer serverAIJobSchedulerMu.Unlock()

	timeout := 10 * time.Minute
	if seconds, err := strconv.Atoi(strings.TrimSpace(os.Getenv("NEXUS_SERVER_AI_CRON_TIMEOUT_SECONDS"))); err == nil && seconds > 0 {
		timeout = time.Duration(seconds) * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	results, err := NewService(db).RunDueJobs(ctx, "cron", 0)
	if err != nil {
		log.Printf("[nexus-server-ai-cron] status=failed error=%q completed=%d", err.Error(), len(results))
		return
	}
	if len(results) > 0 {
		log.Printf("[nexus-server-ai-cron] status=success jobs=%d", len(results))
	}
}
