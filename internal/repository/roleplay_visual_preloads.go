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
			return tx.
				Select("id", "created_at", "updated_at", "template_id", "session_id", "title", "status", "current_step", "total_steps", "current_arc_id", "current_chapter_id").
				Order("created_at DESC").
				Order("id DESC")
		}).
		Preload(template, func(tx *gorm.DB) *gorm.DB {
			return tx.Select(
				"id", "title", "summary", "theme", "level", "status", "is_published",
				"image_url", "visual_style", "visual_tags", "rpg_metadata",
			)
		}).
		Preload(scenes, func(tx *gorm.DB) *gorm.DB {
			return tx.
				Select(
					"id", "quest_id", "arc_id", "chapter_id", "scene_key", "chapter_index", "arc_index",
					"title", "summary", "image_url", "image_alt", "image_status", "image_storage_key",
					"scene_type", "room_type", "atmosphere", "danger_level", "visual_tags", "rpg_metadata",
				).
				Order("chapter_index ASC").
				Order("id ASC")
		}).
		Preload(scenes+".Images", func(tx *gorm.DB) *gorm.DB {
			return tx.
				Select(
					"id", "scene_id", "quest_id", "url", "storage_key", "filename", "mime_type",
					"size", "width", "height", "is_main", "alt", "source",
				).
				Order("is_main DESC").
				Order("id ASC")
		})
}

// PreloadLiveRolePlayQuestVisuals supports both a live attached directly to an
// RP session and a coop live whose RP session is attached through the party.
func PreloadLiveRolePlayQuestVisuals(db *gorm.DB) *gorm.DB {
	db = PreloadRolePlayQuestVisuals(db, "RolePlaySession")
	db = db.Preload("CoopParty")
	return PreloadRolePlayQuestVisuals(db, "CoopParty.RolePlaySession")
}
