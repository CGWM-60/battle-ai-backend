package main

import (
	"context"
	"fmt"

	"cgwm/battle/internal/db"
	translations "cgwm/battle/internal/nexus_game/translations"
	"cgwm/battle/internal/router"
	"cgwm/battle/internal/scheduler"
)

func main() {
	database := db.DbConnect()
	if _, err := translations.SeedInitialImport(context.Background(), database); err != nil {
		panic(fmt.Sprintf("failed to seed initial nexus translations: %v", err))
	}
	scheduler.StartQuestGenerationCron(database)
	// Monde IA desactive: ne pas demarrer les boucles cron world simulation/routine.
	// scheduler.StartWorldSimulationCron(database)
	router.RouterApp(database)
}
