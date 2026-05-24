package constants

const (
	BattleStatusDraft     = "draft"
	BattleStatusLive      = "live"
	BattleStatusPaused    = "paused"
	BattleStatusFinished  = "finished"
	BattleStatusAbandoned = "abandoned"

	ArenaStatusWaiting  = "waiting"
	ArenaStatusRunning  = "running"
	ArenaStatusPaused   = "paused"
	ArenaStatusFinished = "finished"
	ArenaStatusClosed   = "closed"

	CoopStatusWaiting  = "waiting"
	CoopStatusRunning  = "running"
	CoopStatusPaused   = "paused"
	CoopStatusFinished = "finished"
	CoopStatusClosed   = "closed"

	RolePlayStatusDraft     = "draft"
	RolePlayStatusLive      = "live"
	RolePlayStatusPaused    = "paused"
	RolePlayStatusFinished  = "finished"
	RolePlayStatusFailed    = "failed"
	RolePlayStatusAbandoned = "abandoned"

	LiveStatusWaiting   = "waiting"
	LiveStatusStreaming = "streaming"
	LiveStatusPaused    = "paused"
	LiveStatusEnded     = "ended"

	QuestStatusDraft     = "draft"
	QuestStatusPublished = "published"
	QuestStatusArchived  = "archived"

	VisibilityPrivate = "private"
	VisibilityPublic  = "public"

	ModeBattleIA   = "battle_ia"
	ModeRolePlayIA = "roleplay_ia"
	ModeCoop       = "coop"

	AuthorTypeIA     = "ia"
	AuthorTypePlayer = "player"
	AuthorTypeSystem = "system"

	LiveEventTypeJoin    = "join"
	LiveEventTypeLeave   = "leave"
	LiveEventTypeMessage = "message"
	LiveEventTypeChunk   = "chunk"
	LiveEventTypeStatus  = "status"
	LiveEventTypeScore   = "score"
	LiveEventTypeError   = "error"
)
