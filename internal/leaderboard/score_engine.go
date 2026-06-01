package leaderboard

type LeaderboardEntry struct {
	PlayerID  string `json:"playerId"`
	CityName  string `json:"cityName"`
	Score     int    `json:"score"`
	Rank      int    `json:"rank"`
	Delta24h  int    `json:"delta24h"`
}

type Engine struct{}

func NewEngine() *Engine { return &Engine{} }

func (e *Engine) ComputeScore(playerID uint) int {
	// Exact spec formula (Go = single source of truth)
	// buildings*level*10 + researches*50 + pop*0.5 + army*5 + pvp_wins*100 - losses*20 + quests*30 + guild*0.1
	buildingsScore := 0   // TODO: load real player buildings sum(level*10)
	researchesScore := 0  // TODO: count unlocked researches * 50
	popScore := 0         // TODO: current population * 0.5
	armyScore := 0        // TODO: total army units * 5
	pvpScore := 0         // TODO: wins*100 - losses*20
	questsScore := 0      // TODO: completed quests * 30
	guildScore := 0       // TODO: guild contribution * 0.1

	return buildingsScore + researchesScore + int(float64(popScore)*0.5) + armyScore + pvpScore + questsScore + guildScore
}

func (e *Engine) GetGlobal(limit int) []LeaderboardEntry {
	// TODO: real DB query
	return []LeaderboardEntry{
		{PlayerID: "1", CityName: "Valoria", Score: 24500, Rank: 1, Delta24h: 320},
		{PlayerID: "2", CityName: "Dravon", Score: 21800, Rank: 2, Delta24h: -150},
	}
}
