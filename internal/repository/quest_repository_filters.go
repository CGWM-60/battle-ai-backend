package repository

import (
	"log"
	"strings"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type battleQuestStatusFilterPlan struct {
	ApplyStatus      bool
	StatusValue      string
	ApplyIsPublished bool
	WarnMessage      string
}

func hasQuestBattleColumn(db *gorm.DB, columnName string) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasColumn(&models.QuestIaBattle{}, columnName)
}

func hasRolePlayQuestColumn(db *gorm.DB, columnName string) bool {
	if db == nil {
		return false
	}
	return db.Migrator().HasColumn(&models.RolePlayQuestTemplate{}, columnName)
}

func resolveBattleQuestStatusFilter(status string, hasStatusCol bool, hasIsPublishedCol bool) battleQuestStatusFilterPlan {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" || status == "all" {
		return battleQuestStatusFilterPlan{}
	}

	if status == constants.QuestStatusPublished {
		if hasStatusCol {
			return battleQuestStatusFilterPlan{
				ApplyStatus: true,
				StatusValue: constants.QuestStatusPublished,
			}
		}
		if hasIsPublishedCol {
			return battleQuestStatusFilterPlan{ApplyIsPublished: true}
		}
		return battleQuestStatusFilterPlan{
			WarnMessage: "[quest-repository] published filter skipped: columns status/is_published missing on quest_ia_battles",
		}
	}

	if hasStatusCol {
		return battleQuestStatusFilterPlan{
			ApplyStatus: true,
			StatusValue: status,
		}
	}

	return battleQuestStatusFilterPlan{
		WarnMessage: "[quest-repository] status filter skipped: status column missing on quest_ia_battles status=" + status,
	}
}

func applyBattleQuestStatusFilter(db *gorm.DB, query *gorm.DB, status string) *gorm.DB {
	plan := resolveBattleQuestStatusFilter(
		status,
		hasQuestBattleColumn(db, "status"),
		hasQuestBattleColumn(db, "is_published"),
	)
	if plan.WarnMessage != "" {
		log.Printf("%s", plan.WarnMessage)
		return query
	}
	if plan.ApplyStatus {
		return query.Where("status = ?", plan.StatusValue)
	}
	if plan.ApplyIsPublished {
		return query.Where("is_published = ?", true)
	}
	return query
}

func applyBattleQuestQuery(db *gorm.DB, query *gorm.DB, status string, theme string, level string) *gorm.DB {
	if status == "" {
		status = constants.QuestStatusPublished
	}
	query = applyBattleQuestStatusFilter(db, query, status)
	if theme != "" {
		query = query.Where("theme = ?", theme)
	}
	if level != "" {
		query = query.Where("level = ?", level)
	}
	return query
}

func applyRolePlayQuestStatusFilter(db *gorm.DB, query *gorm.DB, status string) *gorm.DB {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" || status == "all" {
		return query
	}

	hasStatus := hasRolePlayQuestColumn(db, "status")
	hasIsPublished := hasRolePlayQuestColumn(db, "is_published")

	if status == constants.QuestStatusPublished {
		if hasStatus {
			query = query.Where("status = ?", constants.QuestStatusPublished)
		}
		if hasIsPublished {
			query = query.Where("is_published = ?", true)
		}
		if !hasStatus && !hasIsPublished {
			log.Printf("[quest-repository] published filter skipped: columns status/is_published missing on role_play_quest_templates")
		}
		return query
	}

	if hasStatus {
		return query.Where("status = ?", status)
	}

	log.Printf("[quest-repository] status filter skipped: status column missing on role_play_quest_templates status=%s", status)
	return query
}

func applyRolePlayQuestQuery(db *gorm.DB, query *gorm.DB, status string, theme string, level string) *gorm.DB {
	if status == "" {
		status = constants.QuestStatusPublished
	}
	query = applyRolePlayQuestStatusFilter(db, query, status)
	if theme != "" {
		query = query.Where("theme = ?", theme)
	}
	if level != "" {
		query = query.Where("level = ?", level)
	}
	return query
}
