package models

import (
	"time"
	"gorm.io/gorm"
)

// AnimaProfile represents a player's Anima in the CGWM system.
// PublicAnimaID is the only identifier safe to expose in the social park.
type AnimaProfile struct {
	ID                    string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	OwnerUserID           string         `gorm:"index;not null" json:"owner_user_id"`
	PublicAnimaID         string         `gorm:"uniqueIndex;not null" json:"public_anima_id"` // anonymized park/social id
	Name                  string         `json:"name"`
	Stage                 string         `json:"stage"`
	Mood                  string         `json:"mood"`
	AvatarPreview         string         `json:"avatar_preview"`
	CloudEnabled          bool           `json:"cloud_enabled"`
	ParkEnabled           bool           `json:"park_enabled"`
	SocialLearningEnabled bool           `json:"social_learning_enabled"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"-"`
}

// AnimaCloudSnapshot stores a privacy-filtered compressed snapshot for sync.
type AnimaCloudSnapshot struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	AnimaID       string         `gorm:"index;not null" json:"anima_id"`
	OwnerUserID   string         `gorm:"index;not null" json:"owner_user_id"`
	SchemaVersion int            `json:"schema_version"`
	SyncVersion   int            `json:"sync_version"`
	SnapshotJSON  string         `gorm:"type:jsonb" json:"snapshot_json"` // compressed safe data
	Checksum      string         `json:"checksum"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// AnimaSyncLog records every cloud operation for audit / conflict resolution.
type AnimaSyncLog struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	AnimaID       string    `gorm:"index" json:"anima_id"`
	OwnerUserID   string    `gorm:"index" json:"owner_user_id"`
	Direction     string    `json:"direction"` // upload, download, conflict, delete
	Status        string    `json:"status"`    // success, error, conflict
	ErrorMessage  string    `json:"error_message,omitempty"`
	ClientVersion string    `json:"client_version"`
	CreatedAt     time.Time `json:"created_at"`
}

// AnimaParkWorld is the living shared park state.
type AnimaParkWorld struct {
	ID                 string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name               string    `json:"name"`
	ActivePlayers      int       `json:"active_players"`
	ActiveAnimas       int       `json:"active_animas"`
	AloneAnimas        int       `json:"alone_animas"`
	CurrentMeetings    int       `json:"current_meetings"`
	SocialLearningEvents int     `json:"social_learning_events"`
	AtmosphereLevel    string    `json:"atmosphere_level"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// AnimaParkPresence is a live (or recently heartbeated) public presence in the park.
// Never contains owner email or real user id.
type AnimaParkPresence struct {
	ID              string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ParkID          string    `gorm:"index" json:"park_id"`
	AnimaID         string    `gorm:"index" json:"anima_id"` // internal
	OwnerUserID     string    `gorm:"index" json:"-"`        // never exposed
	PublicAnimaID   string    `gorm:"index" json:"public_anima_id"`
	DisplayName     string    `json:"display_name"` // filtered
	Stage           string    `json:"stage"`
	Mood            string    `json:"mood"`
	AuraColor       string    `json:"aura_color"`
	X               float64   `json:"x"`
	Y               float64   `json:"y"`
	IsAlone         bool      `json:"is_alone"`
	IsInMeeting     bool      `json:"is_in_meeting"`
	LastHeartbeatAt time.Time `json:"last_heartbeat_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// AnimaParkVisit records a visit (alone or accompanied) for later report generation.
type AnimaParkVisit struct {
	ID           string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ParkID       string         `json:"park_id"`
	AnimaID      string         `json:"anima_id"`
	OwnerUserID  string         `json:"-"`
	StartedAt    time.Time      `json:"started_at"`
	EndedAt      *time.Time     `json:"ended_at,omitempty"`
	LeftAlone    bool           `json:"left_alone"`
	ZonesVisited string         `gorm:"type:jsonb" json:"zones_visited"` // JSON array
	ReportJSON   string         `gorm:"type:jsonb" json:"report_json"`
	ReportRead   bool           `json:"report_read"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// AnimaSocialLearningCard is the only thing that can be shared between Animas (after heavy sanitization).
type AnimaSocialLearningCard struct {
	ID                  string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	SourcePublicAnimaID string    `gorm:"index" json:"source_public_anima_id"`
	Topic               string    `json:"topic"`
	Lesson              string    `json:"lesson"`
	EmotionalTone       string    `json:"emotional_tone"`
	Confidence          float64   `json:"confidence"`
	SafetyScore         float64   `json:"safety_score"`
	Tags                string    `gorm:"type:jsonb" json:"tags"`
	CreatedAt           time.Time `json:"created_at"`
}

// AnimaSocialEncounter records an anonymized meeting (for admin + return report).
type AnimaSocialEncounter struct {
	ID                  string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ParkVisitID         string    `gorm:"index" json:"park_visit_id"`
	SourcePublicAnimaID string    `json:"source_public_anima_id"`
	TargetPublicAnimaID string    `json:"target_public_anima_id"`
	EncounterType       string    `json:"encounter_type"`
	LessonCardID        string    `gorm:"index" json:"lesson_card_id"`
	ResultJSON          string    `gorm:"type:jsonb" json:"result_json"`
	CreatedAt           time.Time `json:"created_at"`
}

// AnimaPrivacyConsent stores explicit opt-ins. No pre-checked consents.
type AnimaPrivacyConsent struct {
	ID                     string         `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	OwnerUserID            string         `gorm:"uniqueIndex" json:"owner_user_id"`
	AnimaID                string         `gorm:"index" json:"anima_id"`
	CloudSyncEnabled       bool           `json:"cloud_sync_enabled"`
	ParkEnabled            bool           `json:"park_enabled"`
	SocialLearningEnabled  bool           `json:"social_learning_enabled"`
	AnonymousModeEnabled   bool           `json:"anonymous_mode_enabled"`
	ConsentVersion         int            `json:"consent_version"`
	AcceptedAt             time.Time      `json:"accepted_at"`
	RevokedAt              *time.Time     `json:"revoked_at,omitempty"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
}