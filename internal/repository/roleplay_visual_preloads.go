package repository

import "gorm.io/gorm"

// PreloadRolePlayQuestVisuals loads the quest catalogue data needed by RP,
// coop and live clients to render the quest and its scene images.
// rolePlaySessionPath is the association path to a RolePlaySession.
func PreloadRolePlayQuestVisuals(db *gorm.DB, rolePlaySessionPath string) *gorm.DB {
	questRuns := "QuestRuns"
	if rolePlaySessionPath != "" {
		db = db.Preload(rolePlaySessionPath)
		questRuns = rolePlaySessionPath + ".QuestRuns"
	}
	template := questRuns + ".Template"
	scenes := template + ".Scenes"

	return db.
		Preload(questRuns, func(tx *gorm.DB) *gorm.DB {
			return tx.Order("created_at DESC").Order("id DESC")
		}).
		Preload(template).
		Preload(scenes, func(tx *gorm.DB) *gorm.DB {
			return tx.Order("chapter_index ASC").Order("id ASC")
		}).
		Preload(scenes+".Images", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("is_main DESC").Order("id ASC")
		})
}

// PreloadLiveRolePlayQuestVisuals supports both a live attached directly to an
// RP session and a coop live whose RP session is attached through the party.
func PreloadLiveRolePlayQuestVisuals(db *gorm.DB) *gorm.DB {
	db = PreloadRolePlayQuestVisuals(db, "RolePlaySession")
	db = db.Preload("CoopParty")
	return PreloadRolePlayQuestVisuals(db, "CoopParty.RolePlaySession")
}
