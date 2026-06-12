package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cgwm/battle/internal/models"

	"gorm.io/gorm"
)

type RolePlayHeroImageService struct {
	db *gorm.DB
}

func NewRolePlayHeroImageService(db *gorm.DB) *RolePlayHeroImageService {
	return &RolePlayHeroImageService{db: db}
}

func HeroImagePublicDir() string {
	if dir := strings.TrimSpace(os.Getenv("HERO_IMAGE_PUBLIC_DIR")); dir != "" {
		return dir
	}
	baseDir := strings.TrimSpace(os.Getenv("NEXUS_ASSETS_BASE_DIR"))
	if baseDir == "" {
		baseDir = "/nexus_game/assets"
	}
	return filepath.Join(baseDir, "heroes")
}

func HeroImagePublicBaseURL() string {
	return strings.TrimRight(defaultText(os.Getenv("HERO_IMAGE_PUBLIC_BASE_URL"), "/assets/heroes"), "/")
}

func (s *RolePlayHeroImageService) SaveUpload(ctx context.Context, name string, sex string, version int, originalName string, reader io.Reader) (*models.RolePlayHeroImage, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	sex = normalizeHeroSex(sex)
	if sex == "" {
		return nil, fmt.Errorf("sex must be h or f")
	}
	if version <= 0 {
		var maxVersion int
		_ = s.db.WithContext(ctx).Model(&models.RolePlayHeroImage{}).
			Where("name = ? AND sex = ?", name, sex).
			Select("COALESCE(MAX(version), 0)").
			Scan(&maxVersion).Error
		version = maxVersion + 1
	}

	assetDir := HeroImagePublicDir()
	publicBase := HeroImagePublicBaseURL()
	relDir := filepath.Join(sex, safeAssetSegment(name))
	fullDir := filepath.Join(assetDir, relDir)
	if err := os.MkdirAll(fullDir, 0o755); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	size := int64(len(data))
	if size > 10*1024*1024 {
		return nil, fmt.Errorf("asset file is too large")
	}

	ext := strings.ToLower(filepath.Ext(originalName))
	if !allowedAssetExt(ext) {
		ext = ".bin"
	}
	filename := fmt.Sprintf("hero_%s_v%d_%d%s", safeAssetSegment(name), version, time.Now().UnixNano(), ext)
	fullPath := filepath.Join(fullDir, filename)
	hasher := sha256.New()
	if _, err := hasher.Write(data); err != nil {
		return nil, err
	}
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return nil, err
	}

	relURL := filepath.ToSlash(filepath.Join(relDir, filename))
	image := &models.RolePlayHeroImage{
		Name:      name,
		Sex:       sex,
		ImageURL:  publicBase + "/" + relURL,
		ImageHash: hex.EncodeToString(hasher.Sum(nil)),
		ImageSize: size,
		ImageData: data,
		Version:   version,
		IsActive:  true,
	}
	if err := s.db.WithContext(ctx).Create(image).Error; err != nil {
		_ = os.Remove(fullPath)
		return nil, err
	}
	return image, nil
}

func normalizeHeroSex(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "h", "m", "male", "homme":
		return "h"
	case "f", "female", "femme":
		return "f"
	default:
		return ""
	}
}
