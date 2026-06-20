package repository

import (
	"context"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type QuestRepository struct {
	db *gorm.DB
}

func NewQuestRepository(db *gorm.DB) *QuestRepository {
	return &QuestRepository{db: db}
}

func (r *QuestRepository) DB() *gorm.DB {
	return r.db
}

func (r *QuestRepository) GetPublishedBattleQuestByID(ctx context.Context, id uint) (*models.QuestIaBattle, error) {
	var quest models.QuestIaBattle
	query := r.db.WithContext(ctx).Model(&models.QuestIaBattle{}).Where("id = ?", id)
	query = applyBattleQuestStatusFilter(r.db, query, constants.QuestStatusPublished)
	err := query.First(&quest).Error
	if err != nil {
		return nil, err
	}

	return &quest, nil
}

func (r *QuestRepository) ListBattleQuests(ctx context.Context, status string, theme string, level string, limit int) ([]models.QuestIaBattle, error) {
	quests, _, err := r.ListBattleQuestsPage(ctx, status, theme, level, limit, 0)
	return quests, err
}

func (r *QuestRepository) ListBattleQuestsPage(ctx context.Context, status string, theme string, level string, limit int, offset int) ([]models.QuestIaBattle, int64, error) {
	var quests []models.QuestIaBattle
	query := r.db.WithContext(ctx).Model(&models.QuestIaBattle{})
	query = applyBattleQuestQuery(r.db, query, status, theme, level)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("created_at DESC, id DESC").Limit(limit).Offset(offset).Find(&quests).Error
	return quests, total, err
}

func (r *QuestRepository) GetBattleQuestByID(ctx context.Context, id uint) (*models.QuestIaBattle, error) {
	var quest models.QuestIaBattle
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&quest).Error
	if err != nil {
		return nil, err
	}
	return &quest, nil
}

func (r *QuestRepository) CreateBattleQuest(ctx context.Context, quest *models.QuestIaBattle) error {
	return r.db.WithContext(ctx).Create(quest).Error
}

func (r *QuestRepository) UpdateBattleQuest(ctx context.Context, id uint, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&models.QuestIaBattle{}).Where("id = ?", id).Updates(fields).Error
}

func (r *QuestRepository) DeleteBattleQuest(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.QuestIaBattle{}, id).Error
}

func (r *QuestRepository) RandomBattleQuest(ctx context.Context, theme string, level string) (*models.QuestIaBattle, error) {
	var quest models.QuestIaBattle
	query := r.db.WithContext(ctx).Model(&models.QuestIaBattle{})
	query = applyBattleQuestQuery(r.db, query, constants.QuestStatusPublished, theme, level)
	err := query.Order("RAND()").First(&quest).Error
	if err != nil {
		return nil, err
	}
	return &quest, nil
}

func (r *QuestRepository) ListRolePlayQuests(ctx context.Context, status string, theme string, level string, limit int) ([]models.RolePlayQuestTemplate, error) {
	quests, _, err := r.ListRolePlayQuestsPage(ctx, status, theme, level, limit, 0)
	return quests, err
}

func (r *QuestRepository) ListRolePlayQuestsPage(ctx context.Context, status string, theme string, level string, limit int, offset int) ([]models.RolePlayQuestTemplate, int64, error) {
	var quests []models.RolePlayQuestTemplate
	query := r.db.WithContext(ctx).Model(&models.RolePlayQuestTemplate{})
	query = applyRolePlayQuestQuery(r.db, query, status, theme, level)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.
		Select("id", "created_at", "updated_at", "slug", "title", "summary", "theme", "level", "xp", "coin", "source", "status", "metadata", "is_published", "published_at", "unpublished_at", "image_url", "visual_style", "visual_tags", "rpg_metadata").
		Preload("Scenes", func(tx *gorm.DB) *gorm.DB {
			return tx.
				Select("id", "quest_id", "arc_id", "chapter_id", "scene_key", "chapter_index", "arc_index", "title", "summary", "image_url", "image_alt", "image_status", "image_storage_key", "scene_type", "room_type", "atmosphere", "danger_level", "visual_tags", "rpg_metadata").
				Order("chapter_index ASC").
				Order("id ASC")
		}).
		Preload("Scenes.Images", func(tx *gorm.DB) *gorm.DB {
			return tx.
				Select("id", "scene_id", "quest_id", "url", "storage_key", "filename", "mime_type", "size", "width", "height", "is_main", "alt", "source").
				Order("is_main DESC").
				Order("id ASC")
		}).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&quests).Error
	if err == nil {
		for index := range quests {
			ApplyRolePlayQuestImageCompatibility(&quests[index])
		}
	}
	return quests, total, err
}

func (r *QuestRepository) GetRolePlayQuestByID(ctx context.Context, id uint) (*models.RolePlayQuestTemplate, error) {
	var quest models.RolePlayQuestTemplate
	err := r.rolePlayQuestScope(ctx).Where("id = ?", id).First(&quest).Error
	if err != nil {
		return nil, err
	}
	ApplyRolePlayQuestImageCompatibility(&quest)
	return &quest, nil
}

func (r *QuestRepository) CreateRolePlayQuest(ctx context.Context, quest *models.RolePlayQuestTemplate) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		arcs := quest.Arcs
		quest.Arcs = nil
		if err := tx.Create(quest).Error; err != nil {
			return err
		}
		return createRolePlayQuestStructure(tx, quest.Id, arcs)
	})
}

func (r *QuestRepository) UpdateRolePlayQuest(ctx context.Context, id uint, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&models.RolePlayQuestTemplate{}).Where("id = ?", id).Updates(fields).Error
}

func (r *QuestRepository) ReplaceRolePlayQuestStructure(ctx context.Context, id uint, arcs []models.RolePlayQuestArc) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("template_id = ?", id).Delete(&models.RolePlayQuestChapter{}).Error; err != nil {
			return err
		}
		if err := tx.Where("template_id = ?", id).Delete(&models.RolePlayQuestArc{}).Error; err != nil {
			return err
		}
		return createRolePlayQuestStructure(tx, id, arcs)
	})
}

func (r *QuestRepository) DeleteRolePlayQuest(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.RolePlayQuestTemplate{}, id).Error
}

func (r *QuestRepository) rolePlayQuestScope(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).
		Model(&models.RolePlayQuestTemplate{}).
		Preload("Arcs", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("position ASC").Order("id ASC")
		}).
		Preload("Arcs.Chapters", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("position ASC").Order("id ASC")
		}).
		Preload("Scenes", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("chapter_index ASC").Order("id ASC")
		}).
		Preload("Scenes.Images", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("is_main DESC").Order("id ASC")
		})
}

func createRolePlayQuestStructure(tx *gorm.DB, templateID uint, arcs []models.RolePlayQuestArc) error {
	if len(arcs) == 0 {
		return nil
	}

	// Prepare arcs for batch insert
	preparedArcs := make([]models.RolePlayQuestArc, 0, len(arcs))
	for i := range arcs {
		a := arcs[i]
		a.Id = 0
		a.TemplateID = templateID
		if a.Position <= 0 {
			a.Position = i + 1
		}
		a.Chapters = nil // chapters handled separately
		preparedArcs = append(preparedArcs, a)
	}

	// Batch insert arcs (much faster than individual Creates)
	if err := tx.CreateInBatches(preparedArcs, 100).Error; err != nil {
		return err
	}

	// Now prepare all chapters, mapping ArcID from the inserted arcs
	allChapters := make([]models.RolePlayQuestChapter, 0, 64)
	for arcIdx, arc := range preparedArcs {
		originalChapters := arcs[arcIdx].Chapters
		for chIdx := range originalChapters {
			ch := originalChapters[chIdx]
			ch.Id = 0
			ch.TemplateID = templateID
			ch.ArcID = arc.Id
			if ch.Position <= 0 {
				ch.Position = chIdx + 1
			}
			allChapters = append(allChapters, ch)
		}
	}

	if len(allChapters) > 0 {
		if err := tx.CreateInBatches(allChapters, 200).Error; err != nil {
			return err
		}
	}

	return nil
}
