package main

import (
	"cgwm/battle/internal/db"
	"cgwm/battle/internal/router"
)

func main() {
	database := db.DbConnect()
	router.RouterApp(database)
}
