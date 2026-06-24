package main

import (
	"context"
	"fmt"
	"log"

	"cgwm/battle/internal/db"
	"cgwm/battle/internal/features"
	translations "cgwm/battle/internal/nexus_game/translations"
	"cgwm/battle/internal/router"
	"cgwm/battle/internal/scheduler"
)

func main() {
	database := db.DbConnect()
	if err := translations.RepairInvalidTranslationTagsJSON(context.Background(), database); err != nil {
		panic(fmt.Sprintf("failed to repair translation tags json: %v", err))
	}
	if err := translations.RepairTranslationDomains(context.Background(), database); err != nil {
		panic(fmt.Sprintf("failed to repair translation domains: %v", err))
	}
	if _, err := translations.SeedInitialImport(context.Background(), database); err != nil {
		panic(fmt.Sprintf("failed to seed initial translations: %v", err))
	}
	missing, err := translations.CheckStartupKeysInDatabase(context.Background(), database, "fr")
	if err != nil {
		panic(fmt.Sprintf("failed to check startup translation keys: %v", err))
	}
	if len(missing) > 0 {
		panic(fmt.Sprintf("missing startup translation keys after seed: %v", missing))
	}
	log.Printf("[translations] startup keys ok count=%d", len(translations.RequiredStartupKeys))

	if !features.NexusGameEnabled() {
		log.Printf("[nexus-game] status=deprecated_disabled step=boot reason=NEXUS_GAME_ENABLED is not true")
	}
	scheduler.StartQuestGenerationCron(database)
	// Monde IA desactive: ne pas demarrer les boucles cron world simulation/routine.
	// scheduler.StartWorldSimulationCron(database)

	// ANIMA CGWM registration (full cloud + park + social + realtime + schedulers)
	// router.RouterApp already sets up; to enable CGWM routes:
	// import cgwm "cgwm/battle/internal/cgwm"
	// After router.RouterApp or inside it: cgwm.RegisterCGWMRoutes(theRouter)
	// (See internal/cgwm/cgwm_routes.go for the complete wiring of the prompt spec)
	router.RouterApp(database)
}
