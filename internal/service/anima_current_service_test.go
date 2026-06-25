package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	cgwmmodels "cgwm/battle/internal/cgwm/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAnimaCurrentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:anima_current_%d?mode=memory&cache=private", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE anima_profiles (
			id TEXT PRIMARY KEY,
			owner_user_id TEXT NOT NULL,
			public_anima_id TEXT NOT NULL,
			name TEXT,
			stage TEXT,
			mood TEXT,
			avatar_preview TEXT,
			cloud_enabled INTEGER,
			park_enabled INTEGER,
			social_learning_enabled INTEGER,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
		CREATE TABLE anima_cloud_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			anima_id TEXT NOT NULL,
			owner_user_id TEXT NOT NULL,
			schema_version INTEGER,
			sync_version INTEGER,
			snapshot_json TEXT,
			checksum TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);
	`).Error; err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestCurrentLivingAnimaEndpointMissingReturnsExistsFalse(t *testing.T) {
	svc := NewAnimaCurrentService(newAnimaCurrentTestDB(t))
	result, err := svc.GetCurrent(context.Background(), 99)
	if err != nil {
		t.Fatalf("get current: %v", err)
	}
	if result.Exists || result.Alive || result.Anima != nil {
		t.Fatalf("expected empty current anima, got %+v", result)
	}
}

func TestCurrentLivingAnimaDeadReturnsAliveFalse(t *testing.T) {
	db := newAnimaCurrentTestDB(t)
	profile := cgwmmodels.AnimaProfile{
		ID:            "anima-dead-1",
		OwnerUserID:     "42",
		PublicAnimaID: "public-dead-1",
		Name:          "Amina",
		Stage:         "adult",
		Mood:          "sad",
	}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}
	snapshot := cgwmmodels.AnimaCloudSnapshot{
		AnimaID:      profile.ID,
		OwnerUserID:  profile.OwnerUserID,
		SnapshotJSON: `{"isDead":true,"name":"Amina"}`,
	}
	if err := db.Create(&snapshot).Error; err != nil {
		t.Fatalf("create snapshot: %v", err)
	}

	result, err := NewAnimaCurrentService(db).GetCurrent(context.Background(), 42)
	if err != nil {
		t.Fatalf("get current: %v", err)
	}
	if !result.Exists || result.Alive {
		t.Fatalf("expected exists=true alive=false, got %+v", result)
	}
	if result.Anima == nil || result.Anima.Status != "inactive" {
		t.Fatalf("expected inactive anima view, got %+v", result.Anima)
	}
}

func TestCurrentLivingAnimaAliveReturnsAppearance(t *testing.T) {
	db := newAnimaCurrentTestDB(t)
	profile := cgwmmodels.AnimaProfile{
		ID:            "anima-alive-1",
		OwnerUserID:     "43",
		PublicAnimaID: "public-alive-1",
		Name:          "Amina",
		Stage:         "adult",
		Mood:          "curious",
		AvatarPreview: "aura://violet",
	}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}
	snapshot := cgwmmodels.AnimaCloudSnapshot{
		AnimaID:     profile.ID,
		OwnerUserID: profile.OwnerUserID,
		SnapshotJSON: `{
			"isDead": false,
			"energy": 72,
			"affection": 61,
			"visualGenome": {"aura":"violet","eyes":"amber"}
		}`,
	}
	if err := db.Create(&snapshot).Error; err != nil {
		t.Fatalf("create snapshot: %v", err)
	}

	result, err := NewAnimaCurrentService(db).GetCurrent(context.Background(), 43)
	if err != nil {
		t.Fatalf("get current: %v", err)
	}
	if !result.Exists || !result.Alive {
		t.Fatalf("expected alive anima, got %+v", result)
	}
	if result.Anima == nil || result.Anima.Energy != 72 || result.Anima.Affection != 61 {
		t.Fatalf("unexpected anima payload %+v", result.Anima)
	}
	if result.Anima.Appearance["aura"] != "violet" {
		t.Fatalf("expected appearance genome, got %+v", result.Anima.Appearance)
	}
}

