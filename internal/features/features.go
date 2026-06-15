package features

import (
	"os"
	"strings"
)

const NexusGameDisabledStatus = "deprecated_disabled"

// NexusGameEnabled is intentionally opt-in. The MMO/Nexus module is deprecated
// and must not boot, seed, expose routes, or run server-AI jobs unless enabled
// explicitly for maintenance.
func NexusGameEnabled() bool {
	value := strings.TrimSpace(os.Getenv("NEXUS_GAME_ENABLED"))
	if value == "" {
		value = strings.TrimSpace(os.Getenv("ENABLE_NEXUS_GAME"))
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "y", "on", "enabled":
		return true
	default:
		return false
	}
}

func NexusGameDisabledPayload() map[string]any {
	return map[string]any{
		"success": false,
		"status":  NexusGameDisabledStatus,
		"module":  "nexus_game",
		"message": "Nexus Games / MMO est deprecie et desactive sur ce backend.",
		"hint":    "Definir NEXUS_GAME_ENABLED=true uniquement pour une maintenance temporaire.",
	}
}
