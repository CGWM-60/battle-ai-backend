package main

import (
	"cgwm/battle/internal/db"
	"cgwm/battle/internal/router"
	"cgwm/battle/internal/scheduler"
)

func main() {
	database := db.DbConnect()
	scheduler.StartQuestGenerationCron(database)
	router.RouterApp(database)
}
