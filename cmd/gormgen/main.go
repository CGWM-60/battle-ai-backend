package main

import (
	"cgwm/battle/internal/models"

	"gorm.io/gen"
)

func main() {
	g := gen.NewGenerator(gen.Config{
		OutPath: "./internal/generated",
		Mode:    gen.WithDefaultQuery,
	})

	g.ApplyBasic(
		models.Users{},
		models.QuestIaBattle{},
		models.BattleSave{},
		models.BattleSaveTurn{},
		models.BattleArena{},
		models.BattleArenaMember{},
		models.RolePlayQuestTemplate{},
		models.RolePlayQuestRun{},
		models.RolePlaySession{},
		models.RolePlaySessionTurn{},
		models.NexusCoinPlan{},
		models.CoopParty{},
		models.CoopPartyMember{},
		models.LiveSession{},
		models.LiveEvent{},
	)

	g.Execute()
}
