package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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

type ContentAssetStatusRow struct {
	ContentType string `json:"contentType"`
	ContentID   string `json:"contentId"`
	Field       string `json:"field"`
	Reference   string `json:"reference"`
	PublicURL   string `json:"publicUrl"`
	Exists      bool   `json:"exists"`
}

type ContentCatalog struct {
	Buildings []models.BuildingDefinition `json:"buildings"`
	Units     []models.UnitDefinition     `json:"units"`
	Research  []models.ResearchDefinition `json:"research"`
	Counts    map[string]int              `json:"counts"`
}

type RequirementStatus struct {
	Type          string `json:"type"`
	ContentID     string `json:"contentId,omitempty"`
	RequiredLevel int    `json:"requiredLevel,omitempty"`
	CurrentLevel  int    `json:"currentLevel,omitempty"`
	Required      bool   `json:"required"`
	Satisfied     bool   `json:"satisfied"`
	Message       string `json:"message,omitempty"`
}

type PrerequisiteValidation struct {
	ProfileGamerID uint                `json:"profileGamerId"`
	Domain         string              `json:"domain"`
	ContentID      string              `json:"contentId"`
	Allowed        bool                `json:"allowed"`
	Requirements   []RequirementStatus `json:"requirements"`
	Missing        []RequirementStatus `json:"missing"`
}

type PrerequisiteError struct {
	Validation PrerequisiteValidation
}

func (e *PrerequisiteError) Error() string {
	if len(e.Validation.Missing) == 0 {
		return "requirements not satisfied"
	}
	first := e.Validation.Missing[0]
	if first.Message != "" {
		return first.Message
	}
	return "requirements not satisfied"
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

func (s *ContentService) Catalog(publishedOnly bool) (*ContentCatalog, error) {
	buildings, err := s.ListBuildings(publishedOnly)
	if err != nil {
		return nil, err
	}
	units, err := s.ListUnits(publishedOnly)
	if err != nil {
		return nil, err
	}
	research, err := s.ListResearch(publishedOnly)
	if err != nil {
		return nil, err
	}
	return &ContentCatalog{
		Buildings: buildings,
		Units:     units,
		Research:  research,
		Counts: map[string]int{
			"buildings": len(buildings),
			"units":     len(units),
			"research":  len(research),
			"total":     len(buildings) + len(units) + len(research),
		},
	}, nil
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

func (s *ContentService) AssetStatus(publicContentBaseURL string) ([]ContentAssetStatusRow, error) {
	rows := []ContentAssetStatusRow{}
	addRef := func(contentType, contentID, folder, field, ref string) {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return
		}
		filename := assetFilenameFromReference(ref)
		publicURL := strings.TrimRight(publicContentBaseURL, "/") + "/" + folder + "/" + filename
		fullPath := filepath.Join(s.assetsBaseDir, folder, filename)
		_, err := os.Stat(fullPath)
		rows = append(rows, ContentAssetStatusRow{
			ContentType: contentType,
			ContentID:   contentID,
			Field:       field,
			Reference:   ref,
			PublicURL:   publicURL,
			Exists:      err == nil,
		})
	}

	buildings, err := s.ListBuildings(false)
	if err != nil {
		return nil, err
	}
	for _, item := range buildings {
		addRef("building", item.ContentID, "buildings", "assetId", item.AssetID)
		for key, ref := range item.AssetsByTier {
			addRef("building", item.ContentID, "buildings", "assetsByTier."+key, ref)
		}
	}

	units, err := s.ListUnits(false)
	if err != nil {
		return nil, err
	}
	for _, item := range units {
		addRef("unit", item.ContentID, "units", "assetId", item.AssetID)
		for key, ref := range item.AssetsByTier {
			addRef("unit", item.ContentID, "units", "assetsByTier."+key, ref)
		}
	}

	research, err := s.ListResearch(false)
	if err != nil {
		return nil, err
	}
	for _, item := range research {
		addRef("research", item.ContentID, "research", "assetId", item.AssetID)
		for key, ref := range item.AssetsByTier {
			addRef("research", item.ContentID, "research", "assetsByTier."+key, ref)
		}
	}

	return rows, nil
}

func assetFilenameFromReference(ref string) string {
	ref = strings.TrimSpace(ref)
	if parsed, err := url.Parse(ref); err == nil && parsed.Path != "" {
		ref = parsed.Path
	}
	ref = strings.TrimPrefix(ref, "/")
	parts := strings.Split(ref, "/")
	if len(parts) == 0 {
		return ref
	}
	return parts[len(parts)-1]
}

// === Building Definitions CRUD (example for "chaque grand item") ===

func (s *ContentService) ListBuildings(publishedOnly bool) ([]models.BuildingDefinition, error) {
	var list []models.BuildingDefinition
	q := s.db.Model(&models.BuildingDefinition{}).Where("content_id <> ''")
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
	if targetLevel <= 0 {
		targetLevel = 1
	}
	if targetLevel > def.MaxLevel {
		return nil, errors.New("max level exceeded")
	}

	if _, err := s.ValidatePrerequisites(profileID, "building", contentID); err != nil {
		return nil, err
	}

	var existing models.PlayerBuilding
	err = s.db.Where("profile_gamer_id = ? AND content_id = ?", profileID, contentID).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	currentLevel := 0
	hasExisting := err == nil
	if hasExisting {
		if existing.IsConstructing {
			return nil, errors.New("BUILDING_ALREADY_CONSTRUCTING: Une construction est deja en cours pour ce batiment.")
		}
		currentLevel = existing.Level
	}
	if targetLevel != currentLevel+1 {
		return nil, fmt.Errorf("BUILDING_LEVEL_SEQUENCE_REQUIRED: Le prochain niveau attendu est %d.", currentLevel+1)
	}

	now := time.Now()
	ends := now.Add(time.Duration(s.CalculateBuildingDurationAtLevel(def, targetLevel, "common")) * time.Second)

	if hasExisting {
		existing.Level = targetLevel - 1
		existing.IsConstructing = true
		existing.ConstructionStartedAt = &now
		existing.ConstructionEndsAt = &ends
		if err := s.db.Save(&existing).Error; err != nil {
			return nil, err
		}
		return &existing, nil
	}

	pb := &models.PlayerBuilding{
		ProfileGamerID:        profileID,
		ContentID:             contentID,
		Level:                 0,
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

// === v1 contract additions for buildings / construction ===

func (s *ContentService) CatalogVersion() map[string]interface{} {
	return map[string]interface{}{
		"version":        "v2.0.1",
		"balanceVersion": "ref-2.0",
		"updatedAt":      time.Now().Format(time.RFC3339),
	}
}

func (s *ContentService) GetBuildingResearchTree(contentID string) (map[string]interface{}, error) {
	def, err := s.GetBuilding(contentID)
	if err != nil {
		return nil, err
	}
	// Parse required research and return a simple tree stub (extend with full ResearchDefinition later)
	var required []map[string]interface{}
	if def.RequiredResearchJSON != "" {
		// simple parse, in real use json unmarshal to list
		required = []map[string]interface{}{{"contentId": def.RequiredResearchJSON, "level": 1}}
	}
	return map[string]interface{}{
		"buildingContentId": contentID,
		"linkedResearch":    required,
		"unlocks":           []string{}, // future
	}, nil
}

func (s *ContentService) BuildingsAssetsManifest() (map[string]interface{}, error) {
	// Use existing asset status logic but focused on buildings
	// For simplicity return list of assets from DB
	buildings, _ := s.ListBuildings(false)
	manifest := []map[string]string{}
	for _, b := range buildings {
		if b.AssetID != "" {
			manifest = append(manifest, map[string]string{"contentId": b.ContentID, "assetId": b.AssetID})
		}
		for k, v := range b.AssetsByTier {
			manifest = append(manifest, map[string]string{"contentId": b.ContentID, "tier": k, "asset": v})
		}
	}
	return map[string]interface{}{"manifest": manifest, "version": "v1"}, nil
}

func (s *ContentService) BuildingsAssetsUpdates(sinceVersion string) (map[string]interface{}, error) {
	// Stub: return full for now, in real compare versions or timestamps
	m, _ := s.BuildingsAssetsManifest()
	return map[string]interface{}{
		"updates": m["manifest"],
		"version": "v1",
		"since":   sinceVersion,
	}, nil
}

func (s *ContentService) ValidatePrerequisites(profileID uint, domain string, contentID string) (PrerequisiteValidation, error) {
	domain = normalizeDomain(domain)
	contentID = strings.TrimSpace(contentID)
	validation := PrerequisiteValidation{
		ProfileGamerID: profileID,
		Domain:         domain,
		ContentID:      contentID,
		Allowed:        true,
		Requirements:   []RequirementStatus{},
		Missing:        []RequirementStatus{},
	}
	if profileID == 0 {
		validation.Allowed = false
		missing := RequirementStatus{Type: "profile", Required: true, Satisfied: false, Message: "PROFILE_REQUIRED: profileGamerId is required"}
		validation.Requirements = append(validation.Requirements, missing)
		validation.Missing = append(validation.Missing, missing)
		return validation, &PrerequisiteError{Validation: validation}
	}

	switch domain {
	case "building":
		def, err := s.GetBuilding(contentID)
		if err != nil {
			return validation, err
		}
		validation.Requirements = append(validation.Requirements, nexusLevelRequirement(def.NexusLevelRequired))
		validation.Requirements = append(validation.Requirements, parseBuildingRequirements(def.RequiredBuildingsJSON)...)
		validation.Requirements = append(validation.Requirements, parseResearchRequirements(def.RequiredResearchJSON)...)
	case "unit":
		def, err := s.GetUnit(contentID)
		if err != nil {
			return validation, err
		}
		validation.Requirements = append(validation.Requirements, nexusLevelRequirement(def.NexusLevelRequired))
		validation.Requirements = append(validation.Requirements, parseBuildingRequirements(def.RequiredBuildingsJSON)...)
		validation.Requirements = append(validation.Requirements, parseResearchRequirements(def.RequiredResearchJSON)...)
	case "research":
		def, err := s.GetResearch(contentID)
		if err != nil {
			return validation, err
		}
		validation.Requirements = append(validation.Requirements, nexusLevelRequirement(def.NexusLevelRequired))
		validation.Requirements = append(validation.Requirements, parseBuildingRequirements(def.RequiredBuildingsJSON)...)
		validation.Requirements = append(validation.Requirements, parseResearchRequirements(def.PrerequisitesJSON)...)
	default:
		return validation, fmt.Errorf("unsupported requirement domain %q", domain)
	}

	profileLevel, buildingLevels, completedResearch, err := s.playerRequirementState(profileID)
	if err != nil {
		return validation, err
	}
	for i := range validation.Requirements {
		req := &validation.Requirements[i]
		if !req.Required {
			req.Satisfied = true
			continue
		}
		switch req.Type {
		case "nexus_level":
			req.CurrentLevel = profileLevel
			req.Satisfied = req.RequiredLevel <= 0 || profileLevel >= req.RequiredLevel
			if !req.Satisfied {
				req.Message = fmt.Sprintf("NEXUS_LEVEL_REQUIRED: Niveau Nexus %d requis.", req.RequiredLevel)
			}
		case "building":
			req.CurrentLevel = buildingLevels[req.ContentID]
			req.Satisfied = req.RequiredLevel <= 0 || req.CurrentLevel >= req.RequiredLevel
			if !req.Satisfied {
				req.Message = fmt.Sprintf("BUILDING_REQUIRED: %s niveau %d requis.", req.ContentID, req.RequiredLevel)
			}
		case "research":
			req.CurrentLevel = 0
			if completedResearch[req.ContentID] {
				req.CurrentLevel = 1
			}
			req.Satisfied = completedResearch[req.ContentID]
			if !req.Satisfied {
				req.Message = fmt.Sprintf("RESEARCH_REQUIRED: %s requis.", req.ContentID)
			}
		}
		if !req.Satisfied {
			validation.Missing = append(validation.Missing, *req)
		}
	}
	validation.Allowed = len(validation.Missing) == 0
	if !validation.Allowed {
		return validation, &PrerequisiteError{Validation: validation}
	}
	return validation, nil
}

func (s *ContentService) playerRequirementState(profileID uint) (int, map[string]int, map[string]bool, error) {
	var profile models.ProfileGamer
	if err := s.db.First(&profile, profileID).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil, nil, err
		}
	}
	profileLevel := profile.Level
	if profileLevel <= 0 {
		profileLevel = 1
	}

	var buildings []models.PlayerBuilding
	if err := s.db.Where("profile_gamer_id = ? AND is_constructing = ?", profileID, false).Find(&buildings).Error; err != nil {
		return 0, nil, nil, err
	}
	buildingLevels := map[string]int{}
	for _, building := range buildings {
		if building.Level > buildingLevels[building.ContentID] {
			buildingLevels[building.ContentID] = building.Level
		}
	}

	var research []models.PlayerResearch
	if err := s.db.Where("profile_gamer_id = ?", profileID).Find(&research).Error; err != nil {
		return 0, nil, nil, err
	}
	completedResearch := map[string]bool{}
	for _, item := range research {
		completedResearch[item.ContentID] = true
	}

	return profileLevel, buildingLevels, completedResearch, nil
}

func normalizeDomain(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimSuffix(domain, "s")
	switch domain {
	case "build", "construction":
		return "building"
	case "research_node", "tech":
		return "research"
	default:
		return domain
	}
}

func nexusLevelRequirement(level int) RequirementStatus {
	if level <= 0 {
		return RequirementStatus{Type: "nexus_level", RequiredLevel: 0, Required: false, Satisfied: true}
	}
	return RequirementStatus{Type: "nexus_level", RequiredLevel: level, Required: true}
}

func parseBuildingRequirements(raw string) []RequirementStatus {
	items := parseRequirementItems(raw)
	requirements := make([]RequirementStatus, 0, len(items))
	for _, item := range items {
		contentID := stringFromAny(firstPresent(item, "contentId", "buildingContentId", "buildingKey", "key"))
		if contentID == "" {
			continue
		}
		level := intFromAny(firstPresent(item, "level", "requiredLevel", "minLevel"), 1)
		requirements = append(requirements, RequirementStatus{Type: "building", ContentID: contentID, RequiredLevel: level, Required: true})
	}
	return requirements
}

func parseResearchRequirements(raw string) []RequirementStatus {
	items := parseRequirementItems(raw)
	requirements := make([]RequirementStatus, 0, len(items))
	for _, item := range items {
		contentID := stringFromAny(firstPresent(item, "contentId", "researchContentId", "researchKey", "key"))
		if contentID == "" {
			continue
		}
		requirements = append(requirements, RequirementStatus{Type: "research", ContentID: contentID, RequiredLevel: 1, Required: true})
	}
	return requirements
}

func parseRequirementItems(raw string) []map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" || raw == "{}" || raw == "null" {
		return nil
	}
	if !strings.HasPrefix(raw, "[") && !strings.HasPrefix(raw, "{") {
		return []map[string]any{{"contentId": raw}}
	}
	var asObjects []map[string]any
	if err := json.Unmarshal([]byte(raw), &asObjects); err == nil {
		return asObjects
	}
	var asStrings []string
	if err := json.Unmarshal([]byte(raw), &asStrings); err == nil {
		items := make([]map[string]any, 0, len(asStrings))
		for _, item := range asStrings {
			if strings.TrimSpace(item) != "" {
				items = append(items, map[string]any{"contentId": strings.TrimSpace(item)})
			}
		}
		return items
	}
	var asObject map[string]any
	if err := json.Unmarshal([]byte(raw), &asObject); err != nil {
		return nil
	}
	if nested, ok := asObject["buildings"]; ok {
		return mapsFromAny(nested)
	}
	if nested, ok := asObject["research"]; ok {
		return mapsFromAny(nested)
	}
	return []map[string]any{asObject}
}

func mapsFromAny(value any) []map[string]any {
	switch typed := value.(type) {
	case []any:
		items := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			switch v := item.(type) {
			case map[string]any:
				items = append(items, v)
			case string:
				items = append(items, map[string]any{"contentId": v})
			}
		}
		return items
	case []map[string]any:
		return typed
	case string:
		return []map[string]any{{"contentId": typed}}
	case map[string]any:
		return []map[string]any{typed}
	default:
		return nil
	}
}

func firstPresent(item map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := item[key]; ok {
			return value
		}
	}
	return nil
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func intFromAny(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		if i, err := typed.Int64(); err == nil {
			return int(i)
		}
	case string:
		var parsed int
		if _, err := fmt.Sscanf(strings.TrimSpace(typed), "%d", &parsed); err == nil {
			return parsed
		}
	}
	return fallback
}

func (s *ContentService) ConstructionQueue(profileID uint) ([]map[string]interface{}, error) {
	pbs, err := s.ListPlayerBuildings(profileID)
	if err != nil {
		return nil, err
	}
	queue := []map[string]interface{}{}
	for _, pb := range pbs {
		if !pb.IsConstructing || pb.ConstructionEndsAt == nil {
			continue
		}
		remaining := int(time.Until(*pb.ConstructionEndsAt).Seconds())
		if remaining < 0 {
			remaining = 0
		}
		queue = append(queue, map[string]interface{}{
			"id":              pb.ID,
			"contentId":       pb.ContentID,
			"level":           pb.Level,
			"startedAt":       pb.ConstructionStartedAt,
			"endsAt":          pb.ConstructionEndsAt,
			"remainingSec":    remaining,
			"assignedWorkers": pb.AssignedWorkers,
		})
	}
	return queue, nil
}

// StartConstructionV1 for /construction/start body {profileGamerId, contentId, targetLevel?}
func (s *ContentService) StartConstructionV1(profileID uint, contentID string, targetLevel int) (*models.PlayerBuilding, error) {
	return s.StartConstruction(profileID, contentID, targetLevel)
}

// Similar for upgrade (reuse start or separate logic)
func (s *ContentService) StartUpgrade(profileID uint, playerBuildingID uint, targetLevel int) (*models.PlayerBuilding, error) {
	// Find the pb
	var pb models.PlayerBuilding
	if err := s.db.First(&pb, playerBuildingID).Error; err != nil {
		return nil, err
	}
	if pb.ProfileGamerID != profileID {
		return nil, errors.New("not owner")
	}
	if pb.IsConstructing {
		return nil, errors.New("BUILDING_ALREADY_CONSTRUCTING: Une construction est deja en cours pour ce batiment.")
	}
	def, err := s.GetBuilding(pb.ContentID)
	if err != nil {
		return nil, err
	}
	if targetLevel <= 0 {
		targetLevel = pb.Level + 1
	}
	if def.MaxLevel > 0 && targetLevel > def.MaxLevel {
		return nil, errors.New("max level exceeded")
	}
	if targetLevel != pb.Level+1 {
		return nil, fmt.Errorf("BUILDING_LEVEL_SEQUENCE_REQUIRED: Le prochain niveau attendu est %d.", pb.Level+1)
	}
	if _, err := s.ValidatePrerequisites(profileID, "building", pb.ContentID); err != nil {
		return nil, err
	}
	now := time.Now()
	ends := now.Add(time.Duration(s.CalculateBuildingDurationAtLevel(def, targetLevel, "common")) * time.Second)
	pb.IsConstructing = true
	pb.ConstructionStartedAt = &now
	pb.ConstructionEndsAt = &ends
	pb.Level = targetLevel - 1
	if err := s.db.Save(&pb).Error; err != nil {
		return nil, err
	}
	return &pb, nil
}

func (s *ContentService) SpeedupConstruction(playerBuildingID uint, speedupSeconds int) error {
	var pb models.PlayerBuilding
	if err := s.db.First(&pb, playerBuildingID).Error; err != nil {
		return err
	}
	if pb.ConstructionEndsAt != nil {
		newEnd := pb.ConstructionEndsAt.Add(-time.Duration(speedupSeconds) * time.Second)
		pb.ConstructionEndsAt = &newEnd
		return s.db.Save(&pb).Error
	}
	return nil
}

func (s *ContentService) CancelConstruction(playerBuildingID uint) error {
	var pb models.PlayerBuilding
	if err := s.db.First(&pb, playerBuildingID).Error; err != nil {
		return err
	}
	pb.IsConstructing = false
	pb.ConstructionStartedAt = nil
	pb.ConstructionEndsAt = nil
	return s.db.Save(&pb).Error
}

func (s *ContentService) CompleteConstruction(playerBuildingID uint) (bool, error) {
	var pb models.PlayerBuilding
	if err := s.db.First(&pb, playerBuildingID).Error; err != nil {
		return false, err
	}
	return s.CompleteConstructionIfReady(&pb)
}

// === Units (full CRUD + catalog from reference §5) ===
func (s *ContentService) ListUnits(publishedOnly bool) ([]models.UnitDefinition, error) {
	var list []models.UnitDefinition
	q := s.db.Model(&models.UnitDefinition{}).Where("content_id <> ''")
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
	q := s.db.Model(&models.ResearchDefinition{}).Where("content_id <> ''")
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
