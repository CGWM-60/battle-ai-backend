package services

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cgwm/battle/internal/nexus_game/content/balance"
	"cgwm/battle/internal/nexus_game/models"

	"gorm.io/gorm"
)

// ContentService handles definitions (catalog) CRUD, level calculations, asset management, and basic player construction logic.
// Designed to be extensible. All major items (buildings, later units/research) follow similar pattern.
// Assets uploaded here are stored under the persistent Nexus assets volume.

type ContentService struct {
	db            *gorm.DB
	assetsBaseDir string // e.g. /nexus_game/assets/content
}

type ContentTranslationStatusRow struct {
	ContentType    string   `json:"contentType"`
	ContentID      string   `json:"contentId"`
	Field          string   `json:"field"`
	Key            string   `json:"key"`
	Exists         bool     `json:"exists"`
	Locales        []string `json:"locales"`
	MissingLocales []string `json:"missingLocales"`
}

func NewContentService(db *gorm.DB, assetsBaseDir string) *ContentService {
	if assetsBaseDir == "" {
		assetsBaseDir = filepath.Join(os.Getenv("NEXUS_ASSETS_BASE_DIR"), "content")
		if assetsBaseDir == "content" {
			assetsBaseDir = "/nexus_game/assets/content"
		}
	}
	_ = os.MkdirAll(assetsBaseDir, 0755)
	return &ContentService{db: db, assetsBaseDir: assetsBaseDir}
}

func (s *ContentService) TranslationStatus(locales []string) ([]ContentTranslationStatusRow, error) {
	if len(locales) == 0 {
		locales = []string{"fr", "en", "de"}
	}

	rows := []ContentTranslationStatusRow{}
	seen := map[string]bool{}
	addKey := func(contentType, contentID, field, key string, required bool) {
		if key == "" && !required {
			return
		}
		rowID := contentType + "\x00" + contentID + "\x00" + field
		if seen[rowID] {
			return
		}
		seen[rowID] = true
		rows = append(rows, ContentTranslationStatusRow{
			ContentType: contentType,
			ContentID:   contentID,
			Field:       field,
			Key:         key,
		})
	}
	addLevelKeys := func(contentType, contentID string, maxLevel int, keys map[string]string) {
		if maxLevel <= 0 {
			maxLevel = 30
		}
		for level := 1; level <= maxLevel; level++ {
			levelKey := fmt.Sprintf("%d", level)
			addKey(contentType, contentID, "levelDescriptionKeys."+levelKey, keys[levelKey], true)
		}
	}

	buildings, err := s.ListBuildings(false)
	if err != nil {
		return nil, err
	}
	for _, item := range buildings {
		addKey("building", item.ContentID, "nameKey", item.NameKey, true)
		addKey("building", item.ContentID, "descriptionKey", item.DescriptionKey, true)
		addKey("building", item.ContentID, "flavorTextKey", item.FlavorTextKey, false)
		addLevelKeys("building", item.ContentID, item.MaxLevel, item.LevelDescriptionKeys)
	}

	units, err := s.ListUnits(false)
	if err != nil {
		return nil, err
	}
	for _, item := range units {
		addKey("unit", item.ContentID, "nameKey", item.NameKey, true)
		addKey("unit", item.ContentID, "descriptionKey", item.DescriptionKey, true)
		addKey("unit", item.ContentID, "flavorTextKey", item.FlavorTextKey, false)
		addLevelKeys("unit", item.ContentID, item.MaxLevel, item.LevelDescriptionKeys)
	}

	research, err := s.ListResearch(false)
	if err != nil {
		return nil, err
	}
	for _, item := range research {
		addKey("research", item.ContentID, "nameKey", item.NameKey, true)
		addKey("research", item.ContentID, "descriptionKey", item.DescriptionKey, true)
		addKey("research", item.ContentID, "flavorTextKey", item.FlavorTextKey, false)
		addLevelKeys("research", item.ContentID, 30, item.LevelDescriptionKeys)
	}

	keySet := map[string]bool{}
	keys := []string{}
	for _, row := range rows {
		if row.Key == "" || keySet[row.Key] {
			continue
		}
		keySet[row.Key] = true
		keys = append(keys, row.Key)
	}
	if len(keys) == 0 {
		return rows, nil
	}

	type foundRow struct {
		Key    string
		Locale string
	}
	found := []foundRow{}
	if err := s.db.Table("nexus_translation_keys as k").
		Select("k.`key` as `key`, v.locale as locale").
		Joins("LEFT JOIN nexus_translation_values as v ON v.key_id = k.id").
		Where("k.`key` IN ?", keys).
		Scan(&found).Error; err != nil {
		return nil, err
	}

	existing := map[string]bool{}
	availableLocales := map[string]map[string]bool{}
	for _, item := range found {
		existing[item.Key] = true
		if item.Locale == "" {
			continue
		}
		if availableLocales[item.Key] == nil {
			availableLocales[item.Key] = map[string]bool{}
		}
		availableLocales[item.Key][item.Locale] = true
	}

	for i := range rows {
		if rows[i].Key == "" {
			rows[i].MissingLocales = append([]string{}, locales...)
			continue
		}
		rows[i].Exists = existing[rows[i].Key]
		for _, locale := range locales {
			if availableLocales[rows[i].Key][locale] {
				rows[i].Locales = append(rows[i].Locales, locale)
			} else {
				rows[i].MissingLocales = append(rows[i].MissingLocales, locale)
			}
		}
	}

	return rows, nil
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
	if def.EffectsJSON == "" {
		def.EffectsJSON = "[]"
	}
	now := time.Now()
	var existing models.BuildingDefinition
	err := s.db.Where("content_id = ?", def.ContentID).First(&existing).Error
	if err == nil {
		def.ID = existing.ID
		def.CreatedAt = existing.CreatedAt
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	} else if def.CreatedAt.IsZero() {
		def.CreatedAt = now
	}
	def.UpdatedAt = now
	return s.db.Save(def).Error
}

func (s *ContentService) DeleteBuilding(contentID string) error {
	return s.db.Where("content_id = ?", contentID).Delete(&models.BuildingDefinition{}).Error
}

func (s *ContentService) DeleteBuildingByID(id uint) error {
	return s.db.Delete(&models.BuildingDefinition{}, id).Error
}

// UploadAsset saves the file for a content item and updates the definition's asset fields.
// Called from handler (multipart form "file", "tier" optional "tier1|2|3|4", "assetId").
// Images are served from /nexus-assets/content/{domain}/{filename} and the
// canonical public URL stored in DB.

func (s *ContentService) UploadAsset(domain, contentID, tier, originalFilename string, data []byte, publicContentBaseURL string) (string, string, error) {
	if domain == "" || contentID == "" {
		return "", "", errors.New("domain and contentID required")
	}

	ext := filepath.Ext(originalFilename)
	if ext == "" {
		ext = ".png"
	}
	slot := tier
	if slot == "" {
		slot = "main"
	}
	safeName := fmt.Sprintf("%s_%s%s", safeAssetNameSegment(contentID), safeAssetNameSegment(slot), ext)
	folder := domain + "s"
	switch domain {
	case "research":
		folder = "research"
	case "building":
		folder = "buildings"
	case "unit":
		folder = "units"
	}
	dir := filepath.Join(s.assetsBaseDir, folder)
	_ = os.MkdirAll(dir, 0755)
	fullPath := filepath.Join(dir, safeName)
	publicURL := strings.TrimRight(publicContentBaseURL, "/") + "/" + folder + "/" + safeName
	assetRef := publicURL
	if strings.TrimSpace(publicContentBaseURL) == "" {
		assetRef = safeName
	}

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", "", err
	}

	// Update definition
	switch domain {
	case "building":
		b, err := s.GetBuilding(contentID)
		if err != nil {
			return "", "", err
		}
		if b.AssetsByTier == nil {
			b.AssetsByTier = map[string]string{}
		}
		key := "tier" + tier
		if tier == "" {
			key = "main"
			b.AssetID = assetRef
		}
		b.AssetsByTier[key] = assetRef
		return safeName, publicURL, s.CreateOrUpdateBuilding(b)
	case "unit":
		u, err := s.GetUnit(contentID)
		if err != nil {
			return "", "", err
		}
		if u.AssetsByTier == nil {
			u.AssetsByTier = map[string]string{}
		}
		key := "tier" + tier
		if tier == "" {
			key = "main"
			u.AssetID = assetRef
		}
		u.AssetsByTier[key] = assetRef
		return safeName, publicURL, s.CreateOrUpdateUnit(u)
	case "research":
		r, err := s.GetResearch(contentID)
		if err != nil {
			return "", "", err
		}
		if r.AssetsByTier == nil {
			r.AssetsByTier = map[string]string{}
		}
		key := "tier" + tier
		if tier == "" {
			key = "main"
			r.AssetID = assetRef
		}
		r.AssetsByTier[key] = assetRef
		return safeName, publicURL, s.CreateOrUpdateResearch(r)
	default:
		return safeName, publicURL, nil
	}
}

func safeAssetNameSegment(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, "..", "_")
	value = strings.ReplaceAll(value, " ", "_")
	if value == "" {
		return "asset"
	}
	return value
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
	// Basic effect application (expand with full formulas/effects from reference)
	// For demo: bump some profile stats based on known buildings
	var p models.ProfileGamer
	if err := s.db.First(&p, pb.ProfileGamerID).Error; err == nil {
		switch pb.ContentID {
		case "building_modular_habitat":
			p.PopulationCapacity += 50
			p.Morale = min(p.Morale+2, 100)
		case "building_solar_plant":
			p.EnergyProduction += 80
			p.EnergyBalance += 80
		case "building_vertical_farm":
			// food etc.
		}
		s.db.Save(&p)
	}
	// TODO: notify, full effects from EffectsJSON, prerequisites check in Start
	return true, nil
}

// ListPlayerBuildings for a profile (used by city dashboard / construction UI)
func (s *ContentService) ListPlayerBuildings(profileID uint) ([]models.PlayerBuilding, error) {
	var list []models.PlayerBuilding
	err := s.db.Where("profile_gamer_id = ?", profileID).Find(&list).Error
	return list, err
}

// === Units (full CRUD + catalog from reference §5) ===
func (s *ContentService) ListUnits(publishedOnly bool) ([]models.UnitDefinition, error) {
	var list []models.UnitDefinition
	q := s.db.Model(&models.UnitDefinition{})
	if publishedOnly {
		q = q.Where("is_published = ?", true)
	}
	if err := q.Order("content_id").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *ContentService) GetUnit(contentID string) (*models.UnitDefinition, error) {
	var u models.UnitDefinition
	if err := s.db.Where("content_id = ?", contentID).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *ContentService) CreateOrUpdateUnit(def *models.UnitDefinition) error {
	if def.EffectsJSON == "" {
		def.EffectsJSON = "[]"
	}
	now := time.Now()
	var existing models.UnitDefinition
	err := s.db.Where("content_id = ?", def.ContentID).First(&existing).Error
	if err == nil {
		def.ID = existing.ID
		def.CreatedAt = existing.CreatedAt
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	} else if def.CreatedAt.IsZero() {
		def.CreatedAt = now
	}
	def.UpdatedAt = now
	return s.db.Save(def).Error
}

func (s *ContentService) DeleteUnit(contentID string) error {
	return s.db.Where("content_id = ?", contentID).Delete(&models.UnitDefinition{}).Error
}

func (s *ContentService) DeleteUnitByID(id uint) error {
	return s.db.Delete(&models.UnitDefinition{}, id).Error
}

// === Research (full CRUD + 11 branches x 7 tiers per §6) ===
func (s *ContentService) ListResearch(publishedOnly bool) ([]models.ResearchDefinition, error) {
	var list []models.ResearchDefinition
	q := s.db.Model(&models.ResearchDefinition{})
	if publishedOnly {
		q = q.Where("is_published = ?", true)
	}
	if err := q.Order("content_id").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *ContentService) GetResearch(contentID string) (*models.ResearchDefinition, error) {
	var r models.ResearchDefinition
	if err := s.db.Where("content_id = ?", contentID).First(&r).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *ContentService) CreateOrUpdateResearch(def *models.ResearchDefinition) error {
	if def.EffectsJSON == "" {
		def.EffectsJSON = "[]"
	}
	now := time.Now()
	var existing models.ResearchDefinition
	err := s.db.Where("content_id = ?", def.ContentID).First(&existing).Error
	if err == nil {
		def.ID = existing.ID
		def.CreatedAt = existing.CreatedAt
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	} else if def.CreatedAt.IsZero() {
		def.CreatedAt = now
	}
	def.UpdatedAt = now
	return s.db.Save(def).Error
}

func (s *ContentService) DeleteResearch(contentID string) error {
	return s.db.Where("content_id = ?", contentID).Delete(&models.ResearchDefinition{}).Error
}

func (s *ContentService) DeleteResearchByID(id uint) error {
	return s.db.Delete(&models.ResearchDefinition{}, id).Error
}
