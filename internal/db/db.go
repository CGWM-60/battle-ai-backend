package db

import (
	"cgwm/battle/internal/models"
	"fmt"
	"os"
	"strconv"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func DbConnect() *gorm.DB {
	host := getEnv("DB_HOST", "127.0.0.1")
	port := getEnv("DB_PORT", "3306")
	name := getEnv("DB_NAME", "battleia")
	user := getEnv("DB_USER", "battleia")
	password := getEnv("DB_PASSWORD", "battleia")

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user,
		password,
		host,
		port,
		name,
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("failed to connect database: %v", err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(fmt.Sprintf("failed to get database pool: %v", err))
	}
	sqlDB.SetMaxOpenConns(getEnvInt("DB_MAX_OPEN_CONNS", 25))
	sqlDB.SetMaxIdleConns(getEnvInt("DB_MAX_IDLE_CONNS", 10))
	sqlDB.SetConnMaxLifetime(time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 30)) * time.Minute)

	db.AutoMigrate(
		&models.Users{},
		&models.IAProfile{},
		&models.QuestIaBattle{},
		&models.BattleSave{},
		&models.BattleSaveTurn{},
		&models.BattleArena{},
		&models.BattleArenaMember{},
		&models.RolePlayQuestTemplate{},
		&models.RolePlayQuestArc{},
		&models.RolePlayQuestChapter{},
		&models.RolePlayQuestRun{},
		&models.RolePlaySession{},
		&models.RolePlaySessionTurn{},
		&models.AIUsageRecord{},
		&models.CoopParty{},
		&models.CoopPartyMember{},
		&models.LiveSession{},
		&models.LiveEvent{},
	)
	return db
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}

func getEnvInt(key string, fallback int) int {
	value, err := strconv.Atoi(getEnv(key, strconv.Itoa(fallback)))
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}
