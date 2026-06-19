package service

import (
	"encoding/json"
	"fmt"
)

func UintFromSnapshot(snapshot map[string]any, key string) uint {
	return uintFromSnapshot(snapshot, key)
}

func uintFromSnapshot(snapshot map[string]any, key string) uint {
	if snapshot == nil {
		return 0
	}
	raw, ok := snapshot[key]
	if !ok {
		return 0
	}
	if value, ok := uintFromAny(raw); ok {
		return value
	}
	return 0
}

func uintFromAny(value any) (uint, bool) {
	switch v := value.(type) {
	case uint:
		return v, true
	case uint64:
		return uint(v), true
	case int:
		if v < 0 {
			return 0, false
		}
		return uint(v), true
	case int64:
		if v < 0 {
			return 0, false
		}
		return uint(v), true
	case float64:
		if v < 0 {
			return 0, false
		}
		return uint(v), true
	case json.Number:
		if n, err := v.Int64(); err == nil && n >= 0 {
			return uint(n), true
		}
	}
	return 0, false
}

func validateRolePlaySnapshotQuestID(snapshot map[string]any, expectedQuestID uint) error {
	if expectedQuestID == 0 {
		return nil
	}
	snapshotQuestID := uintFromSnapshot(snapshot, "questId")
	if snapshotQuestID != 0 && snapshotQuestID != expectedQuestID {
		return fmt.Errorf("roleplay snapshot quest mismatch")
	}
	return nil
}

func EnsureRolePlaySnapshotQuestID(snapshot map[string]any, questID uint) {
	ensureRolePlaySnapshotQuestID(snapshot, questID)
}

func ensureRolePlaySnapshotQuestID(snapshot map[string]any, questID uint) {
	if snapshot == nil || questID == 0 {
		return
	}
	if uintFromSnapshot(snapshot, "questId") == 0 {
		snapshot["questId"] = questID
	}
}

func stampRolePlaySnapshotFromTemplate(snapshot map[string]any, templateID uint, title, slug string) {
	if snapshot == nil || templateID == 0 {
		return
	}
	snapshot["questId"] = templateID
	if title != "" {
		snapshot["questTitle"] = title
		snapshot["serverQuestTitle"] = title
	}
	if slug != "" {
		snapshot["questSlug"] = slug
		snapshot["serverQuestSlug"] = slug
	}
}