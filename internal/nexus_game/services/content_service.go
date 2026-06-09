package services

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cgwm/battle/internal/nexus_game/content/balance"
	"cgwm/battle/internal/nexus_game/models"

	"gorm.io/gorm"
)

// ContentService handles definitions (catalog) CRUD, level calculations, asset management, and basic player construction logic.
// Designed to be extensible. All major items (buildings, later units/research) follow similar pattern.
// Assets uploaded here are served statically by the main server (configure /nexus-assets/ to point to content/assets).

type ContentService struct {
	db           *gorm.DB
	assetsBaseDir string // e.g. "./content/assets" or absolute
}

func NewContentService(db *gorm.DB, assetsBaseDir string) *ContentService {
	if assetsBaseDir == "" {
		assetsBaseDir = "./content/assets"
	}
	_ = os.MkdirAll(assetsBaseDir, 0755)
	return &ContentService{db: db, assetsBaseDir: assetsBaseDir}
}

// === Building Definitions CRUD (example for "chaque grand item") ===

func (s *ContentService) ListBuildings(publishedOnly bool) ([]models.BuildingDefinition, error) {
	var list []models.BuildingDefinition
	q := s.db.Model(&models.BuildingDefinition{})
	if publishedOnly {
		q = q.Where("is_published = ?", true)
	}
	if err := q.Order("content_id").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *ContentService) GetBuilding(contentID string) (*models.BuildingDefinition, error) {
	var b models.BuildingDefinition
	if err := s.db.Where("content_id = ?", contentID).First(&b).Error; err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *ContentService) CreateOrUpdateBuilding(def *models.BuildingDefinition) error {
	def.UpdatedAt = time.Now()
	if def.CreatedAt.IsZero() {
		def.CreatedAt = time.Now()
	}
	return s.db.Save(def).Error
}

func (s *ContentService) DeleteBuilding(contentID string) error {
	return s.db.Where("content_id = ?", contentID).Delete(&models.BuildingDefinition{}).Error
}

// UploadAsset saves the file for a content item and updates the definition's asset fields.
// Called from handler (multipart form "file", "tier" optional "tier1|2|3|4", "assetId").
// Images served later via static /nexus-assets/content/buildings/{filename}

func (s *ContentService) UploadAsset(domain, contentID, tier, originalFilename string, data []byte) (string, error) {
	if domain == "" || contentID == "" {
		return "", errors.New("domain and contentID required")
	}

	ext := filepath.Ext(originalFilename)
	if ext == "" {
		ext = ".png"
	}
	safeName := fmt.Sprintf("%s_%s%s", contentID, tier, ext)
	dir := filepath.Join(s.assetsBaseDir, domain+"s") // buildings, units...
	_ = os.MkdirAll(dir, 0755)
	fullPath := filepath.Join(dir, safeName)

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", err
	}

	// Update definition
	switch domain {
	case "building":
		b, err := s.GetBuilding(contentID)
		if err != nil {
			return "", err
		}
		if b.AssetsByTier == nil {
			b.AssetsByTier = map[string]string{}
		}
		key := "tier" + tier
		if tier == "" {
			key = "main"
			b.AssetID = safeName
		}
		b.AssetsByTier[key] = safeName
		return safeName, s.CreateOrUpdateBuilding(b)
	// TODO: units, research similar
	default:
		return safeName, nil
	}
}

// === Formulas usage (open for all content) ===

func (s *ContentService) CalculateBuildingCostAtLevel(def *models.BuildingDefinition, level int, rarity string) map[string]int {
	credits := balance.ApplyRarityMultiplier(balance.Cost(level, float64(def.CostBaseCredits)), rarity, false)
	metal := balance.ApplyRarityMultiplier(balance.Cost(level, float64(def.CostBaseMetal)), rarity, false)
	data := balance.ApplyRarityMultiplier(balance.Cost(level, float64(def.CostBaseData)), rarity, false)
	return map[string]int{
		"credits": credits,
		"metal":   metal,
		"data":    data,
	}
}

func (s *ContentService) CalculateBuildingDurationAtLevel(def *models.BuildingDefinition, level int, rarity string) int {
	base := float64(def.DurationBaseSeconds)
	return balance.ApplyRarityMultiplier(balance.DurationSeconds(level, base), rarity, true)
}

// === Player construction (depends on buildings) ===
// Simplified MVP: start construction, check completion on tick or on demand.

func (s *ContentService) StartConstruction(profileID uint, contentID string, targetLevel int) (*models.PlayerBuilding, error) {
	def, err := s.GetBuilding(contentID)
	if err != nil {
		return nil, err
	}
	if targetLevel > def.MaxLevel {
		return nil, errors.New("max level exceeded")
	}

	// TODO: check resources, prerequisites, workers using city service / profile

	now := time.Now()
	ends := now.Add(time.Duration(s.CalculateBuildingDurationAtLevel(def, targetLevel, "common")) * time.Second)

	pb := &models.PlayerBuilding{
		ProfileGamerID:        profileID,
		ContentID:             contentID,
		Level:                 targetLevel - 1, // current before finish
		IsConstructing:        true,
		ConstructionStartedAt: &now,
		ConstructionEndsAt:    &ends,
	}
	if err := s.db.Create(pb).Error; err != nil {
		return nil, err
	}
	return pb, nil
}

func (s *ContentService) CompleteConstructionIfReady(pb *models.PlayerBuilding) (bool, error) {
	if !pb.IsConstructing || pb.ConstructionEndsAt == nil || pb.ConstructionEndsAt.After(time.Now()) {
		return false, nil
	}
	pb.Level++
	pb.IsConstructing = false
	pb.ConstructionStartedAt = nil
	pb.ConstructionEndsAt = nil
	if err := s.db.Save(pb).Error; err != nil {
		return false, err
	}
	// TODO: apply effects to profile (pop capacity, energy prod etc), notify
	return true, nil
}

// ListPlayerBuildings for a profile (used by city dashboard / construction UI)
func (s *ContentService) ListPlayerBuildings(profileID uint) ([]models.PlayerBuilding, error) {
	var list []models.PlayerBuilding
	err := s.db.Where("profile_gamer_id = ?", profileID).Find(&list).Error
	return list, err
}
