package models

import (
	"time"

	"gorm.io/datatypes"
)

// RPGArcMap = carte persistante d'un arc de quête RP.
type RPGArcMap struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	QuestID   uint           `gorm:"index" json:"questId"`
	ArcID     uint           `gorm:"index" json:"arcId"`
	Name      string         `gorm:"size:160" json:"name"`
	Theme     string         `gorm:"size:80" json:"theme"`
	Width     int            `json:"width"`
	Height    int            `json:"height"`
	NodesJSON datatypes.JSON `gorm:"type:json" json:"nodesJson"`
	EdgesJSON datatypes.JSON `gorm:"type:json" json:"edgesJson"`
	LoreJSON  datatypes.JSON `gorm:"type:json" json:"loreJson"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

// RPGSessionMapState = état de carte pour une session solo.
type RPGSessionMapState struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	SessionID         uint           `gorm:"uniqueIndex" json:"sessionId"`
	ArcMapID          uint           `gorm:"index" json:"arcMapId"`
	CurrentNodeID     string         `gorm:"size:64" json:"currentNodeId"`
	ActiveChapterID   uint           `json:"activeChapterId"`
	ActiveObjectiveID string         `gorm:"size:64" json:"activeObjectiveId"`
	DiscoveredJSON    datatypes.JSON `gorm:"type:json" json:"discoveredJson"`
	CompletedJSON     datatypes.JSON `gorm:"type:json" json:"completedJson"`
	LockedJSON        datatypes.JSON `gorm:"type:json" json:"lockedJson"`
	FlagsJSON         datatypes.JSON `gorm:"type:json" json:"flagsJson"`
	NPCStateJSON      datatypes.JSON `gorm:"type:json" json:"npcStateJson"`
	EnemyStateJSON    datatypes.JSON `gorm:"type:json" json:"enemyStateJson"`
	LootStateJSON     datatypes.JSON `gorm:"type:json" json:"lootStateJson"`
	TrapStateJSON     datatypes.JSON `gorm:"type:json" json:"trapStateJson"`
	CombatStateJSON   datatypes.JSON `gorm:"type:json" json:"combatStateJson"`
	HistoryJSON       datatypes.JSON `gorm:"type:json" json:"historyJson"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

// RPGCoopMapState = état carte partagée coop.
type RPGCoopMapState struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	CoopPartyID       uint           `gorm:"uniqueIndex" json:"coopPartyId"`
	ArcMapID          uint           `gorm:"index" json:"arcMapId"`
	CurrentNodeID     string         `gorm:"size:64" json:"currentNodeId"`
	ActiveChapterID   uint           `json:"activeChapterId"`
	ActiveObjectiveID string         `gorm:"size:64" json:"activeObjectiveId"`
	PlayerPositions   datatypes.JSON `gorm:"type:json" json:"playerPositionsJson"`
	DiscoveredJSON    datatypes.JSON `gorm:"type:json" json:"discoveredJson"`
	CompletedJSON     datatypes.JSON `gorm:"type:json" json:"completedJson"`
	LockedJSON        datatypes.JSON `gorm:"type:json" json:"lockedJson"`
	FlagsJSON         datatypes.JSON `gorm:"type:json" json:"flagsJson"`
	NPCStateJSON      datatypes.JSON `gorm:"type:json" json:"npcStateJson"`
	EnemyStateJSON    datatypes.JSON `gorm:"type:json" json:"enemyStateJson"`
	LootStateJSON     datatypes.JSON `gorm:"type:json" json:"lootStateJson"`
	TrapStateJSON     datatypes.JSON `gorm:"type:json" json:"trapStateJson"`
	CombatStateJSON   datatypes.JSON `gorm:"type:json" json:"combatStateJson"`
	PendingVoteJSON   datatypes.JSON `gorm:"type:json" json:"pendingVoteJson"`
	HistoryJSON       datatypes.JSON `gorm:"type:json" json:"historyJson"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

// RPGMapNode = lieu sur la carte.
type RPGMapNode struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	X            int            `json:"x"`
	Y            int            `json:"y"`
	Description  string         `json:"description"`
	DangerLevel  string         `json:"dangerLevel"`
	ChapterIDs   []uint         `json:"chapterIds"`
	ObjectiveIDs []string       `json:"objectiveIds"`
	NPCs         []RPGMapNPC    `json:"npcs"`
	Enemies      []RPGMapEnemy  `json:"enemies"`
	Loot         []RPGMapLoot   `json:"loot"`
	Traps        []RPGMapTrap   `json:"traps"`
	IsEntrance   bool           `json:"isEntrance"`
	IsExit       bool           `json:"isExit"`
	IsBossRoom   bool           `json:"isBossRoom"`
	IsSafeRoom   bool           `json:"isSafeRoom"`
	IsLever      bool           `json:"isLever"`
	LeverTarget  string         `json:"leverTarget,omitempty"`
	Flags        map[string]any `json:"flags"`
}

// RPGMapEdge = connexion entre nodes.
type RPGMapEdge struct {
	From         string         `json:"from"`
	To           string         `json:"to"`
	Locked       bool           `json:"locked"`
	Hidden       bool           `json:"hidden"`
	LockType     string         `json:"lockType"`
	RequiredKey  string         `json:"requiredKey"`
	RequiredFlag string         `json:"requiredFlag"`
	Description  string         `json:"description"`
	Flags        map[string]any `json:"flags"`
}

// RPGMapObjective = objectif de chapitre lié à la carte.
type RPGMapObjective struct {
	ID           string `json:"id"`
	ChapterID    uint   `json:"chapterId"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	TargetNodeID string `json:"targetNodeId"`
	RequiredFlag string `json:"requiredFlag"`
	Completed    bool   `json:"completed"`
}

// RPGMapNPC = personnage non-joueur.
type RPGMapNPC struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Role           string `json:"role"`
	Mood           string `json:"mood"`
	CurrentNode    string `json:"currentNode"`
	Memory         string `json:"memory"`
	DialogueIntent string `json:"dialogueIntent"`
}

// RPGMapEnemy = ennemi sur la carte ou en combat.
type RPGMapEnemy struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Level     int    `json:"level"`
	Health    int    `json:"health"`
	MaxHealth int    `json:"maxHealth"`
	Phase     int    `json:"phase"`
	MaxPhase  int    `json:"maxPhase"`
	Boss      bool   `json:"boss"`
	Defeated  bool   `json:"defeated"`
}

// RPGMapLoot = butin.
type RPGMapLoot struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Rarity       string `json:"rarity"`
	Collected    bool   `json:"collected"`
	RequiredRoll bool   `json:"requiredRoll"`
}

// RPGMapTrap = piège.
type RPGMapTrap struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Difficulty int    `json:"difficulty"`
	Triggered  bool   `json:"triggered"`
	Disarmed   bool   `json:"disarmed"`
	Damage     int    `json:"damage"`
	Detected   bool   `json:"detected"`
}

// RPGMapAction = action disponible pour le joueur.
type RPGMapAction struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Type         string `json:"type"`
	TargetNodeID string `json:"targetNodeId,omitempty"`
	TargetID     string `json:"targetId,omitempty"`
	RequiresRoll bool   `json:"requiresRoll"`
	Difficulty   int    `json:"difficulty,omitempty"`
	Attribute    string `json:"attribute,omitempty"`
	Skill        string `json:"skill,omitempty"`
}

// RPGMapDialogueLine = ligne de dialogue.
type RPGMapDialogueLine struct {
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
	Intent  string `json:"intent,omitempty"`
}

// RPGCombatState = état de combat actif.
type RPGCombatState struct {
	ID           string        `json:"id"`
	Active       bool          `json:"active"`
	NodeID       string        `json:"nodeId"`
	Turn         int           `json:"turn"`
	CurrentActor string        `json:"currentActor"`
	Enemies      []RPGMapEnemy `json:"enemies"`
	Log          []string      `json:"log"`
	Phase        int           `json:"phase"`
	MaxPhase     int           `json:"maxPhase"`
	Victory      bool          `json:"victory"`
	Fled         bool          `json:"fled"`
}

// RPGMapRollResult = résultat d'un jet de dé.
type RPGMapRollResult struct {
	ActionID  string `json:"actionId"`
	Attribute string `json:"attribute"`
	Skill     string `json:"skill"`
	Roll      int    `json:"roll"`
	Modifier  int    `json:"modifier"`
	Total     int    `json:"total"`
	Difficulty int   `json:"difficulty"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
}

// RPGMapResponse = réponse standard des endpoints carte.
type RPGMapResponse struct {
	ArcMap          *RPGArcMap          `json:"arcMap"`
	MapState        *RPGSessionMapState `json:"mapState"`
	CurrentNode     *RPGMapNode         `json:"currentNode"`
	ActiveChapter   map[string]any      `json:"activeChapter,omitempty"`
	ActiveObjective *RPGMapObjective    `json:"activeObjective,omitempty"`
	Objectives      []RPGMapObjective   `json:"objectives,omitempty"`
	Narration       string              `json:"narration"`
	Dialogues       []RPGMapDialogueLine `json:"dialogues"`
	AvailableActions []RPGMapAction     `json:"availableActions"`
	InCombat        bool                `json:"inCombat"`
	Combat          *RPGCombatState     `json:"combat,omitempty"`
	RollResult      *RPGMapRollResult   `json:"rollResult,omitempty"`
	DiscoveredNodes []string            `json:"discoveredNodes,omitempty"`
	AdjacentNodes   []RPGMapNode        `json:"adjacentNodes,omitempty"`
}

// RPGCoopMapResponse = réponse carte coop.
type RPGCoopMapResponse struct {
	RPGMapResponse
	CoopMapState    *RPGCoopMapState     `json:"coopMapState,omitempty"`
	PlayerPositions map[string]string    `json:"playerPositions,omitempty"`
	PendingVote     map[string]any       `json:"pendingVote,omitempty"`
	Members         []map[string]any     `json:"members,omitempty"`
}