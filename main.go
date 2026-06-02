package main

import (
	"cgwm/battle/internal/db"
	"cgwm/battle/internal/router"
	"cgwm/battle/internal/scheduler"
)

func main() {
	database := db.DbConnect()
	scheduler.StartQuestGenerationCron(database)
	// Monde IA desactive: ne pas demarrer les boucles cron world simulation/routine.
	// scheduler.StartWorldSimulationCron(database)
	router.RouterApp(database)
}
