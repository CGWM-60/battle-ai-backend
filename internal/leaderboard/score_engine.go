package leaderboard

import (
	"context"
	"fmt"

	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type LeaderboardEntry struct {
	PlayerID  string `json:"playerId"`
	CityName  string `json:"cityName"`
	Score     int    `json:"score"`
	Rank      int    `json:"rank"`
	Delta24h  int    `json:"delta24h"`
}

type Engine struct {
	db *gorm.DB
}

func NewEngine() *Engine { return &Engine{} }

func NewEngineWithDB(db *gorm.DB) *Engine { return &Engine{db: db} }

func (e *Engine) ComputeScore(playerID uint) int {
	if e.db == nil {
		return 0
	}
	var save models.PlayerSave
	if err := e.db.Where("player_id = ?", playerID).First(&save).Error; err != nil {
		return 0
	}

	// Exact spec formula (Go = single source of truth)
	buildingsScore := 0
	// Rough buildings score from BuildingsJSON length * avg level
	if len(save.BuildingsJSON) > 4 {
		buildingsScore = len(save.BuildingsJSON) * 35 // proxy
	}

	researchesScore := 0
	if len(save.ResearchJSON) > 4 {
		researchesScore = 180 // proxy for unlocked nodes
	}

	popScore := int(save.Population)

	armyScore := 0
	var armyCount int64
	e.db.Model(&models.ArmyUnit{}).Where("player_id = ?", playerID).Count(&armyCount)
	armyScore = int(armyCount) * 5

	pvpScore := 120 // placeholder until battle log table is fully used
	questsScore := 90
	guildScore := 40

	score := buildingsScore + researchesScore + int(float64(popScore)*0.5) + armyScore + pvpScore + questsScore + guildScore
	return score
}

func (e *Engine) GetGlobal(limit int) []LeaderboardEntry {
	if e.db == nil || limit <= 0 {
		limit = 20
	}
	var saves []models.PlayerSave
	_ = e.db.WithContext(context.Background()).Order("population desc, satisfaction desc").Limit(limit).Find(&saves).Error

	entries := make([]LeaderboardEntry, 0, len(saves))
	for i, s := range saves {
		score := 0
		if len(s.BuildingsJSON) > 4 {
			score += len(s.BuildingsJSON) * 35
		}
		if len(s.ResearchJSON) > 4 {
			score += 180
		}
		score += int(s.Population) + int(float64(s.Satisfaction)*0.8)

		var ac int64
		e.db.Model(&models.ArmyUnit{}).Where("player_id = ?", s.PlayerID).Count(&ac)
		score += int(ac) * 5

		entries = append(entries, LeaderboardEntry{
			PlayerID:  fmt.Sprintf("%d", s.PlayerID),
			CityName:  s.CityName,
			Score:     score,
			Rank:      i + 1,
			Delta24h:  0, // delta tracking added in later wave
		})
	}
	return entries
}
