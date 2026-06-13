package repositories

import (
	"context"
	"errors"

	"cgwm/battle/internal/nexus_game/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PlayerUnitRepository struct {
	db *gorm.DB
}

func NewPlayerUnitRepository(db *gorm.DB) *PlayerUnitRepository {
	return &PlayerUnitRepository{db: db}
}

func (r *PlayerUnitRepository) List(ctx context.Context, profileID uint) ([]models.PlayerUnit, error) {
	var units []models.PlayerUnit
	err := r.db.WithContext(ctx).Where("profile_gamer_id = ?", profileID).Order("content_id ASC").Find(&units).Error
	return units, err
}

func (r *PlayerUnitRepository) Get(ctx context.Context, profileID uint, code string) (*models.PlayerUnit, error) {
	var unit models.PlayerUnit
	err := r.db.WithContext(ctx).Where("profile_gamer_id = ? AND content_id = ?", profileID, code).First(&unit).Error
	if err != nil {
		return nil, err
	}
	return &unit, nil
}

func (r *PlayerUnitRepository) GetOrCreate(ctx context.Context, profileID uint, code string) (*models.PlayerUnit, error) {
	unit, err := r.Get(ctx, profileID, code)
	if err == nil {
		return unit, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	unit = &models.PlayerUnit{ProfileGamerID: profileID, ContentID: code}
	return unit, r.db.WithContext(ctx).Create(unit).Error
}

func (r *PlayerUnitRepository) Save(ctx context.Context, unit *models.PlayerUnit) error {
	unit.Count = unit.ReserveQuantity + unit.AssignedQuantity + unit.WoundedQuantity + unit.DamagedQuantity + unit.InactiveQuantity
	return r.db.WithContext(ctx).Save(unit).Error
}

type UnitTrainingQueueRepository struct {
	db *gorm.DB
}

func NewUnitTrainingQueueRepository(db *gorm.DB) *UnitTrainingQueueRepository {
	return &UnitTrainingQueueRepository{db: db}
}

func (r *UnitTrainingQueueRepository) List(ctx context.Context, profileID uint, includeDone bool) ([]models.UnitTrainingQueue, error) {
	var rows []models.UnitTrainingQueue
	q := r.db.WithContext(ctx).Where("profile_gamer_id = ?", profileID)
	if !includeDone {
		q = q.Where("status IN ?", []string{"training", "ready"})
	}
	err := q.Order("created_at DESC, id DESC").Find(&rows).Error
	return rows, err
}

func (r *UnitTrainingQueueRepository) ListAll(ctx context.Context, limit int) ([]models.UnitTrainingQueue, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var rows []models.UnitTrainingQueue
	err := r.db.WithContext(ctx).Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *UnitTrainingQueueRepository) Get(ctx context.Context, id uint) (*models.UnitTrainingQueue, error) {
	var row models.UnitTrainingQueue
	err := r.db.WithContext(ctx).First(&row, id).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *UnitTrainingQueueRepository) Create(ctx context.Context, row *models.UnitTrainingQueue) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *UnitTrainingQueueRepository) Save(ctx context.Context, row *models.UnitTrainingQueue) error {
	return r.db.WithContext(ctx).Save(row).Error
}

type ArmyFormationRepository struct {
	db *gorm.DB
}

func NewArmyFormationRepository(db *gorm.DB) *ArmyFormationRepository {
	return &ArmyFormationRepository{db: db}
}

func (r *ArmyFormationRepository) List(ctx context.Context, profileID uint) ([]models.ArmyFormation, error) {
	var rows []models.ArmyFormation
	err := r.db.WithContext(ctx).Where("profile_gamer_id = ?", profileID).Order("type ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (r *ArmyFormationRepository) ListAll(ctx context.Context, limit int) ([]models.ArmyFormation, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var rows []models.ArmyFormation
	err := r.db.WithContext(ctx).Order("updated_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *ArmyFormationRepository) Get(ctx context.Context, id uint) (*models.ArmyFormation, error) {
	var row models.ArmyFormation
	err := r.db.WithContext(ctx).First(&row, id).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ArmyFormationRepository) UpsertDefault(ctx context.Context, row *models.ArmyFormation) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "profile_gamer_id"}, {Name: "type"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "status", "world_id", "updated_at"}),
	}).Create(row).Error
}

func (r *ArmyFormationRepository) Save(ctx context.Context, row *models.ArmyFormation) error {
	return r.db.WithContext(ctx).Save(row).Error
}

type ArmySlotRepository struct {
	db *gorm.DB
}

func NewArmySlotRepository(db *gorm.DB) *ArmySlotRepository {
	return &ArmySlotRepository{db: db}
}

func (r *ArmySlotRepository) ListSlots(ctx context.Context, formationID uint) ([]models.ArmyFormationSlot, error) {
	var rows []models.ArmyFormationSlot
	err := r.db.WithContext(ctx).Where("formation_id = ?", formationID).Order("slot_index ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (r *ArmySlotRepository) GetSlot(ctx context.Context, id uint) (*models.ArmyFormationSlot, error) {
	var row models.ArmyFormationSlot
	err := r.db.WithContext(ctx).First(&row, id).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ArmySlotRepository) CreateSlot(ctx context.Context, row *models.ArmyFormationSlot) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *ArmySlotRepository) ListAssignments(ctx context.Context, formationID uint) ([]models.ArmySlotAssignment, error) {
	var rows []models.ArmySlotAssignment
	err := r.db.WithContext(ctx).Where("formation_id = ?", formationID).Order("slot_id ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (r *ArmySlotRepository) GetAssignment(ctx context.Context, slotID uint, unitCode string) (*models.ArmySlotAssignment, error) {
	var row models.ArmySlotAssignment
	err := r.db.WithContext(ctx).Where("slot_id = ? AND unit_code = ?", slotID, unitCode).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *ArmySlotRepository) SaveAssignment(ctx context.Context, row *models.ArmySlotAssignment) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *ArmySlotRepository) DeleteAssignment(ctx context.Context, row *models.ArmySlotAssignment) error {
	return r.db.WithContext(ctx).Delete(row).Error
}

type ArmyAutomationRepository struct {
	db *gorm.DB
}

func NewArmyAutomationRepository(db *gorm.DB) *ArmyAutomationRepository {
	return &ArmyAutomationRepository{db: db}
}

func (r *ArmyAutomationRepository) GetOrCreate(ctx context.Context, profileID uint, worldID uint) (*models.ArmyAutomationSettings, error) {
	var row models.ArmyAutomationSettings
	err := r.db.WithContext(ctx).Where("profile_gamer_id = ?", profileID).First(&row).Error
	if err == nil {
		return &row, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	row = models.ArmyAutomationSettings{
		ProfileGamerID:            profileID,
		WorldID:                   worldID,
		AutoDefenseEnabled:        true,
		MaxAutoSpendPercent:       20,
		MinFoodKeep:               200,
		MinEnergyKeep:             150,
		MinDefensePower:           300,
		MaxUnitsOnMissionPercent:  40,
		AllowRareResourceSpend:    false,
		AutoComposeDefenseEnabled: false,
	}
	return &row, r.db.WithContext(ctx).Create(&row).Error
}

func (r *ArmyAutomationRepository) Save(ctx context.Context, row *models.ArmyAutomationSettings) error {
	return r.db.WithContext(ctx).Save(row).Error
}

type ArmyReportRepository struct {
	db *gorm.DB
}

func NewArmyReportRepository(db *gorm.DB) *ArmyReportRepository {
	return &ArmyReportRepository{db: db}
}

func (r *ArmyReportRepository) ListReports(ctx context.Context, profileID uint, limit int) ([]models.ArmyCombatReport, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var rows []models.ArmyCombatReport
	err := r.db.WithContext(ctx).
		Where("attacker_user_id = ? OR defender_user_id = ?", profileID, profileID).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

func (r *ArmyReportRepository) ListAllReports(ctx context.Context, limit int) ([]models.ArmyCombatReport, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var rows []models.ArmyCombatReport
	err := r.db.WithContext(ctx).Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

type ArmyTransactionLogRepository struct {
	db *gorm.DB
}

func NewArmyTransactionLogRepository(db *gorm.DB) *ArmyTransactionLogRepository {
	return &ArmyTransactionLogRepository{db: db}
}

func (r *ArmyTransactionLogRepository) Create(ctx context.Context, row *models.ArmyTransactionLog) error {
	return r.db.WithContext(ctx).Create(row).Error
}
