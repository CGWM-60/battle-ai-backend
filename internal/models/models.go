package models

import (
	"cgwm/battle/internal/provider"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Users = joueur applicatif.
// Cette table porte les infos de compte et toutes les relations utiles
// vers les battles, arenes, quetes roleplay, coop et live.
type Users struct {
	Id        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Pseudo visible dans les parties, l'arene, le live et le coop.
	Pseudo string `gorm:"size:80;index"`
	// Password = mot de passe hashé du joueur.
	Password string `gorm:"size:255"`
	// BirthdayDate = date de naissance brute si tu veux la garder telle quelle.
	BirthdayDate string `gorm:"size:32"`
	// Email unique pour la connexion et la verification compte.
	Email string `gorm:"size:190;uniqueIndex"`
	// Avatar = url ou chemin du visuel profil.
	Avatar string `gorm:"size:255"`
	// Xp / Coin = progression globale du joueur.
	Xp   int
	Coin int

	// Runtime only: provider IA choisi en memoire, pas en base.
	KeyApiProvider []provider.Provider `gorm:"-"`

	// Relations de persistance.
	BattleSaves        []BattleSave        `gorm:"foreignKey:OwnerID"`
	HostedArenas       []BattleArena       `gorm:"foreignKey:HostUserID"`
	ArenaMemberships   []BattleArenaMember `gorm:"foreignKey:UserID"`
	CoopParties        []CoopParty         `gorm:"foreignKey:HostUserID"`
	CoopMemberships    []CoopPartyMember   `gorm:"foreignKey:UserID"`
	RolePlayRuns       []RolePlayQuestRun  `gorm:"foreignKey:UserID"`
	RolePlaySessions   []RolePlaySession   `gorm:"foreignKey:OwnerID"`
	RolePlayCharacters []RolePlayCharacter `gorm:"foreignKey:UserID"`
	IAProfiles         []IAProfile         `gorm:"foreignKey:OwnerID"`
}

// IAProfile = profil persistant cree par un joueur pour reutiliser une IA.
// Il stocke la personnalite et le modele, mais pas les cles API.
type IAProfile struct {
	Id        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	OwnerID uint  `gorm:"index"`
	Owner   Users `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	Name         string `gorm:"size:120;index"`
	ProviderName string `gorm:"size:40;index"`
	ModelName    string `gorm:"size:120"`
	Personality  string `gorm:"type:text"`
	Mindset      string `gorm:"type:text"`
	Style        string `gorm:"type:text"`
	Goal         string `gorm:"type:text"`
	Weakness     string `gorm:"type:text"`
}

// QuestIaBattle = banque systeme des questions / quetes pour les Battle IA.
// Elle sert a generer des parties, alimenter la rotation et attribuer des recompenses.
type QuestIaBattle struct {
	Id        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Slug = identifiant stable pour retrouver une quete meme si le titre change.
	Slug string `gorm:"size:120;uniqueIndex"`
	// Title = nom court visible dans l'admin ou le lobby.
	Title string `gorm:"size:160;index"`
	// Content = vraie question / prompt de battle.
	Content string `gorm:"type:text"`
	// Level = difficulte affichee au joueur.
	Level string `gorm:"size:32;index"`
	// Point = score brut si tu veux noter la valeur de la quete.
	Point int
	// Theme = theme de debat pour filtrer la rotation.
	Theme string `gorm:"size:80;index"`
	// Xp = recompense d'xp si la quete est completee.
	Xp int
	// Coin = recompense monetaire optionnelle.
	Coin int
	// Mode = solo / arena / coop si tu veux limiter l'usage de la quete.
	Mode string `gorm:"size:32;index"`
	// Source = system / manuel / event.
	Source string `gorm:"size:32;index"`
	// Status = draft / published / archived.
	Status string `gorm:"size:32;index"`
	// Metadata = reserve pour tags, contraintes, ids externes, etc.
	Metadata datatypes.JSON `gorm:"type:json"`
}

// BattleSave = sauvegarde progressive d'une battle IA.
// C'est la table principale pour reprendre une partie plus tard.
type BattleSave struct {
	Id        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// OwnerID = joueur qui a cree la partie.
	OwnerID uint  `gorm:"index"`
	Owner   Users `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// QuestID = quete source utilisee pour cette battle si elle vient du systeme.
	QuestID *uint          `gorm:"index"`
	Quest   *QuestIaBattle `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	// Title = etiquette lisible dans la liste des sauvegardes.
	Title string `gorm:"size:160"`
	// Question = sujet exact de la battle.
	Question string `gorm:"type:text"`
	// Status = draft / live / paused / finished / abandoned.
	Status string `gorm:"size:32;index"`
	// Visibility = private / public. Public permet de rattacher une arene.
	Visibility string `gorm:"size:32;index"`
	// CurrentRound / TotalRounds = reprise precise de l'avancement.
	CurrentRound int
	TotalRounds  int
	// DebateRoundSeconds = duree cible d'un round de debat apres le round 1.
	DebateRoundSeconds int
	// PublicVote = indique si l'application gere un vote a chaque round.
	PublicVote bool
	// WinnerName = gagnant final si la partie est terminee.
	WinnerName string `gorm:"size:120"`
	// PromptTokens / CompletionTokens / TotalTokens = consommation IA cumulee.
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	// EstimatedCostMicros = cout estime en micro-USD, si les prix env sont configures.
	EstimatedCostMicros int64

	// IASnapshot = config complete des IA jouees au moment du lancement.
	// Cela evite qu'une reprise casse si la config runtime change.
	IASnapshot datatypes.JSON `gorm:"type:json"`
	// Context = etat courant utile pour reprendre le scenario.
	Context BattleMessageContext `gorm:"type:json;serializer:json"`

	// StartedAt / LastActivityAt / FinishedAt pilotent la reprise et le live.
	StartedAt      *time.Time `gorm:"index"`
	LastActivityAt *time.Time `gorm:"index"`
	FinishedAt     *time.Time `gorm:"index"`

	// Relations.
	Turns        []BattleSaveTurn `gorm:"foreignKey:BattleSaveID"`
	Arena        *BattleArena     `gorm:"foreignKey:BattleSaveID"`
	LiveSessions []LiveSession    `gorm:"foreignKey:BattleSaveID"`
}

// BattleSaveTurn = historique detaille d'une battle IA.
// Chaque message / round / phase est persiste pour la reprise et le live.
type BattleSaveTurn struct {
	Id        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time

	// BattleSaveID = session parente.
	BattleSaveID uint       `gorm:"index;uniqueIndex:idx_battle_turn_sequence"`
	BattleSave   BattleSave `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// Round = numero de round.
	Round int `gorm:"index"`
	// Phase = ouverture / debat / conclusion / system.
	Phase string `gorm:"size:32;index"`
	// AuthorType = ia / player / system.
	AuthorType string `gorm:"size:32;index"`
	// AuthorName = nom de l'IA ou du joueur qui parle.
	AuthorName string `gorm:"size:120;index"`
	// Content = texte genere ou saisi.
	Content string `gorm:"type:longtext"`
	// Payload = stockage brut pour chunks live, stats, score, etc.
	Payload datatypes.JSON `gorm:"type:json"`
	// Sequence = ordre strict des evenements dans une meme battle.
	Sequence int `gorm:"uniqueIndex:idx_battle_turn_sequence"`
}

// BattleArena = salle publique d'une battle en cours.
// Un autre joueur peut la rejoindre pendant son execution.
type BattleArena struct {
	Id        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Code = code de partage / url courte pour rejoindre une arene.
	Code string `gorm:"size:64;uniqueIndex"`
	// Name = nom lisible dans le lobby.
	Name string `gorm:"size:120;index"`
	// Status = waiting / running / paused / finished / closed.
	Status string `gorm:"size:32;index"`

	// HostUserID = joueur qui a ouvert l'arene.
	HostUserID uint  `gorm:"index"`
	HostUser   Users `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// BattleSaveID = battle publique associee a l'arene.
	BattleSaveID uint       `gorm:"uniqueIndex"`
	BattleSave   BattleSave `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// MaxPlayers = nombre de joueurs humains autorises.
	MaxPlayers int
	// AllowSpectators = si des viewers peuvent suivre sans jouer.
	AllowSpectators bool
	// LastHeartbeatAt = utile pour savoir si l'arene est toujours active.
	LastHeartbeatAt *time.Time `gorm:"index"`

	Members      []BattleArenaMember `gorm:"foreignKey:ArenaID"`
	LiveSessions []LiveSession       `gorm:"foreignKey:ArenaID"`
}

// BattleArenaMember = inscription d'un joueur dans une arene battle.
// Sert pour rejoindre, quitter, savoir qui joue et qui spectate.
type BattleArenaMember struct {
	Id        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	ArenaID uint        `gorm:"index:idx_arena_user,unique"`
	Arena   BattleArena `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	UserID uint  `gorm:"index:idx_arena_user,unique"`
	User   Users `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// Role = host / challenger / spectator.
	Role string `gorm:"size:32;index"`
	// Status = invited / joined / left / kicked / disconnected.
	Status     string     `gorm:"size:32;index"`
	JoinedAt   *time.Time `gorm:"index"`
	LastSeenAt *time.Time `gorm:"index"`
}

// RolePlayQuestTemplate = quete roleplay generee pour le systeme.
// Elle sert de modele reutilisable pour lancer des parties roleplay IA.
type RolePlayQuestTemplate struct {
	Id        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"index:idx_rpq_catalog,priority:5"`
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Slug string `gorm:"size:120;uniqueIndex"`
	// Title = titre administratif ou visible dans un catalogue.
	Title string `gorm:"size:160;index"`
	// Summary = resume court de la quete.
	Summary string `gorm:"type:text"`
	// Prompt = texte complet servant a initialiser le roleplay.
	Prompt string `gorm:"type:longtext"`
	// Theme / Level = filtres de catalogue.
	Theme string `gorm:"size:80;index;index:idx_rpq_catalog,priority:3"`
	Level string `gorm:"size:32;index;index:idx_rpq_catalog,priority:4"`
	// Xp / Coin = recompenses si la quete est finie.
	Xp   int
	Coin int
	// Source = system / manuel / event.
	Source string `gorm:"size:32;index"`
	// Status = draft / published / archived.
	Status   string         `gorm:"size:32;index;index:idx_rpq_catalog,priority:1"`
	Metadata datatypes.JSON `gorm:"type:json"`

	// Publication explicite (sync avec Status pour retrocompat).
	IsPublished   bool       `gorm:"index;index:idx_rpq_catalog,priority:2;default:false" json:"isPublished"`
	PublishedAt   *time.Time `json:"publishedAt"`
	UnpublishedAt *time.Time `json:"unpublishedAt"`

	// Visuels globaux de quete.
	ImagePrompt         string         `gorm:"type:text" json:"imagePrompt"`
	ImageNegativePrompt string         `gorm:"type:text" json:"imageNegativePrompt"`
	VisualStyle         string         `gorm:"size:120" json:"visualStyle"`
	ImageURL            string         `gorm:"size:512" json:"imageUrl"`
	ImageStorageKey     string         `gorm:"size:512" json:"imageStorageKey"`
	VisualTags          datatypes.JSON `gorm:"type:json" json:"visualTags"`
	RpgMetadata         datatypes.JSON `gorm:"type:json" json:"rpgMetadata"`

	Arcs   []RolePlayQuestArc   `gorm:"foreignKey:TemplateID"`
	Scenes []RolePlayQuestScene `gorm:"foreignKey:QuestID"`
	Runs   []RolePlayQuestRun   `gorm:"foreignKey:TemplateID"`
}

// RolePlayQuestScene = scene visuelle / salle narrative d'une quete RP.
type RolePlayQuestScene struct {
	Id        uint `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	QuestID uint                  `gorm:"index;index:idx_rpq_scene_order,priority:1" json:"questId"`
	Quest   RolePlayQuestTemplate `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	ArcID     *uint `gorm:"index" json:"arcId"`
	ChapterID *uint `gorm:"index" json:"chapterId"`

	SceneKey     string `gorm:"size:120;index" json:"sceneKey"`
	ChapterIndex int    `gorm:"index;index:idx_rpq_scene_order,priority:3" json:"chapterIndex"`
	ArcIndex     int    `gorm:"index;index:idx_rpq_scene_order,priority:2" json:"arcIndex"`
	Title        string `gorm:"size:160" json:"title"`
	Summary      string `gorm:"type:text" json:"summary"`
	Prompt       string `gorm:"type:text" json:"prompt"`

	ImagePrompt     string         `gorm:"type:text" json:"imagePrompt"`
	NegativePrompt  string         `gorm:"type:text" json:"imageNegativePrompt"`
	ImageURL        string         `gorm:"size:512" json:"imageUrl"`
	ImageAlt        string         `gorm:"size:255" json:"imageAlt"`
	ImageStatus     string         `gorm:"size:32" json:"imageStatus"`
	ImageStorageKey string         `gorm:"size:512" json:"imageStorageKey"`
	SceneType       string         `gorm:"size:64" json:"sceneType"`
	RoomType        string         `gorm:"size:64" json:"roomType"`
	Atmosphere      string         `gorm:"size:64" json:"atmosphere"`
	DangerLevel     string         `gorm:"size:32" json:"dangerLevel"`
	VisualTags      datatypes.JSON `gorm:"type:json" json:"visualTags"`
	RpgMetadata     datatypes.JSON `gorm:"type:json" json:"rpgMetadata"`

	Images []RolePlayQuestSceneImage `gorm:"foreignKey:SceneID" json:"images"`
}

// RolePlayImagePromptJob = job batch admin pour generer les prompts image RP.
type RolePlayImagePromptJob struct {
	Id         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time

	Status string `gorm:"size:32;index"` // pending/running/completed/failed/cancelled/interrupted
	Scope  string `gorm:"size:32"`

	TotalQuests       int
	ProcessedQuests   int
	UpdatedQuests     int
	CreatedScenes     int
	UpdatedPrompts    int
	FailedQuests      int
	CurrentQuestID    *uint
	CurrentQuestTitle string `gorm:"size:160"`

	OnlyMissing     bool
	ForceRegenerate bool
	SceneCount      int
	BatchSize       int
	Provider        string         `gorm:"size:80"`
	Model           string         `gorm:"size:160"`
	QuestIDs        datatypes.JSON `gorm:"type:json"`

	Errors datatypes.JSON `gorm:"type:json"`
}

// RolePlayImagePromptJobItem = progression par quete dans un job batch.
type RolePlayImagePromptJobItem struct {
	Id        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	JobID      uint   `gorm:"index"`
	QuestID    uint   `gorm:"index"`
	QuestTitle string `gorm:"size:160"`
	Status     string `gorm:"size:32;index"` // pending/running/completed/failed/skipped

	CreatedScenes  int
	UpdatedPrompts int
	Error          string `gorm:"type:text"`
}

// RolePlayQuestSceneImage = image uploadee pour une scene RP.
type RolePlayQuestSceneImage struct {
	Id        uint `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	SceneID uint               `gorm:"index;index:idx_rpq_scene_image_order,priority:1" json:"sceneId"`
	Scene   RolePlayQuestScene `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	QuestID uint               `gorm:"index" json:"questId"`

	URL              string `gorm:"size:512" json:"url"`
	StorageKey       string `gorm:"size:512" json:"storageKey"`
	Filename         string `gorm:"size:255" json:"filename"`
	MimeType         string `gorm:"size:64" json:"mimeType"`
	Size             int64  `json:"size"`
	Width            int    `json:"width"`
	Height           int    `json:"height"`
	IsMain           bool   `gorm:"index;index:idx_rpq_scene_image_order,priority:2" json:"isMain"`
	Alt              string `gorm:"size:255" json:"alt"`
	Source           string `gorm:"size:32" json:"source"`
	OriginalFilename string `gorm:"size:255" json:"originalFilename,omitempty"`
	OriginalMimeType string `gorm:"size:64" json:"originalMimeType,omitempty"`
}

// RolePlayQuestArc = grande partie narrative d'une quete RP.
// Un template doit pouvoir porter plusieurs arcs, chacun decoupe en chapitres.
type RolePlayQuestArc struct {
	Id        uint `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	TemplateID uint                  `gorm:"index;index:idx_rpq_arc_order,priority:1" json:"templateId"`
	Template   RolePlayQuestTemplate `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	Position  int            `gorm:"index;index:idx_rpq_arc_order,priority:2" json:"position"`
	Slug      string         `gorm:"size:120;index" json:"slug"`
	Title     string         `gorm:"size:160;index" json:"title"`
	Summary   string         `gorm:"type:text" json:"summary"`
	Objective string         `gorm:"type:text" json:"objective"`
	Prompt    string         `gorm:"type:longtext" json:"prompt"`
	Metadata  datatypes.JSON `gorm:"type:json" json:"metadata"`

	Chapters []RolePlayQuestChapter `gorm:"foreignKey:ArcID" json:"chapters"`
}

// RolePlayQuestChapter = checkpoint jouable dans un arc RP.
// C'est l'unite que Flutter peut afficher et que le backend peut suivre.
type RolePlayQuestChapter struct {
	Id        uint `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	TemplateID uint                  `gorm:"index;index:idx_rpq_chapter_template_order,priority:1" json:"templateId"`
	Template   RolePlayQuestTemplate `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	ArcID      uint                  `gorm:"index;index:idx_rpq_chapter_arc_order,priority:1" json:"arcId"`
	Arc        RolePlayQuestArc      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	Position      int            `gorm:"index;index:idx_rpq_chapter_template_order,priority:2;index:idx_rpq_chapter_arc_order,priority:2" json:"position"`
	Slug          string         `gorm:"size:120;index" json:"slug"`
	Title         string         `gorm:"size:160;index" json:"title"`
	Summary       string         `gorm:"type:text" json:"summary"`
	Objective     string         `gorm:"type:text" json:"objective"`
	IntroPrompt   string         `gorm:"type:longtext" json:"introPrompt"`
	SuccessPrompt string         `gorm:"type:longtext" json:"successPrompt"`
	FailurePrompt string         `gorm:"type:longtext" json:"failurePrompt"`
	IsBoss        bool           `json:"isBoss"`
	Xp            int            `json:"xp"`
	Coin          int            `json:"coin"`
	Metadata      datatypes.JSON `gorm:"type:json" json:"metadata"`
}

// RolePlayQuestRun = progression d'un joueur sur une quete roleplay.
// Cette table sert a enregistrer les quetes roleplay en cours ou terminees.
type RolePlayQuestRun struct {
	Id        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	TemplateID *uint                  `gorm:"index"`
	Template   *RolePlayQuestTemplate `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	UserID uint  `gorm:"index"`
	User   Users `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// SessionID = partie roleplay qui porte la progression concrete.
	SessionID       *uint            `gorm:"index"`
	RolePlaySession *RolePlaySession `gorm:"foreignKey:SessionID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	// Title = photo du titre au moment du lancement.
	Title string `gorm:"size:160"`
	// Status = draft / live / paused / finished / failed / abandoned.
	Status string `gorm:"size:32;index"`
	// CurrentStep / TotalSteps = checkpoints fonctionnels de la quete.
	CurrentStep      int
	TotalSteps       int
	CurrentArcID     *uint `gorm:"index"`
	CurrentChapterID *uint `gorm:"index"`
	// CompletedChapters = ids des chapitres termines, conserve en JSON pour avancer vite cote API.
	CompletedChapters datatypes.JSON `gorm:"type:json"`
	// Journal = resume evolutif des faits marquants.
	Journal string `gorm:"type:longtext"`
	// State = etat libre de la quete, inventaire, flags, objectifs, etc.
	State datatypes.JSON `gorm:"type:json"`

	CharacterID *uint              `gorm:"index"`
	Character   *RolePlayCharacter `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	StartedAt      *time.Time `gorm:"index"`
	LastActivityAt *time.Time `gorm:"index"`
	FinishedAt     *time.Time `gorm:"index"`
}

// RolePlayCharacter = fiche de heros creee/validee avant une quete RP ou Coop Live.
// Elle stocke la fiche jouable, jamais les cles API utilisees pour la generer.
type RolePlayCharacter struct {
	Id        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID uint  `gorm:"index" json:"userId"`
	User   Users `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`

	Name         string `gorm:"size:120;index" json:"name"`
	Class        string `gorm:"column:class;size:120;index" json:"class"`
	Origin       string `gorm:"size:180" json:"origin"`
	Race         string `gorm:"size:120" json:"race"`
	Alignment    string `gorm:"size:120" json:"alignment"`
	Personality  string `gorm:"type:text" json:"personality"`
	Background   string `gorm:"type:text" json:"background"`
	PersonalGoal string `gorm:"type:text" json:"personalGoal"`
	Level        int    `json:"level"`
	HeroImageID  *uint  `gorm:"index" json:"heroImageId,omitempty"`
	ImageURL     string `gorm:"size:500" json:"imageUrl"`

	Attributes datatypes.JSON `gorm:"type:json" json:"attributes"`
	Skills     datatypes.JSON `gorm:"type:json" json:"skills"`
	Traits     datatypes.JSON `gorm:"type:json" json:"traits"`
	Inventory  datatypes.JSON `gorm:"type:json" json:"inventory"`

	Health    int `json:"health"`
	MaxHealth int `json:"maxHealth"`
	Stress    int `json:"stress"`
	Fatigue   int `json:"fatigue"`
	Morale    int `json:"morale"`

	RolePlayQuestRunID *uint             `gorm:"index" json:"rolePlayQuestRunId,omitempty"`
	RolePlayQuestRun   *RolePlayQuestRun `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	RolePlaySessionID  *uint             `gorm:"index" json:"rolePlaySessionId,omitempty"`
	RolePlaySession    *RolePlaySession  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	CoopLiveSessionID  *uint             `gorm:"index" json:"coopLiveSessionId,omitempty"`
	CoopLiveSession    *LiveSession      `gorm:"foreignKey:CoopLiveSessionID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	CoopPartyID        *uint             `gorm:"index" json:"coopPartyId,omitempty"`
	CoopParty          *CoopParty        `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
}

// RolePlayHeroImage = catalogue d'images selectionnables pour les heros RP.
type RolePlayHeroImage struct {
	Id        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name      string `gorm:"size:160;index" json:"name"`
	Sex       string `gorm:"size:1;index" json:"sex"`
	ImageURL  string `gorm:"size:500" json:"imageUrl"`
	ImageHash string `gorm:"size:128;index" json:"imageHash"`
	ImageSize int64  `json:"imageSize"`
	ImageData []byte `gorm:"type:longblob" json:"-"`
	Version   int    `gorm:"index" json:"version"`
	IsActive  bool   `gorm:"index" json:"isActive"`
}

// RolePlaySession = sauvegarde d'une partie roleplay IA.
// Sert a reprendre une aventure, a la lier au coop et au live.
type RolePlaySession struct {
	Id        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	OwnerID uint  `gorm:"index;index:idx_roleplay_owner_updated,priority:1"`
	Owner   Users `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// Mode = solo / coop / live.
	Mode string `gorm:"size:32;index"`
	// Title = nom de la campagne ou de la scene.
	Title string `gorm:"size:160;index"`
	// Status = draft / live / paused / finished / abandoned.
	Status string `gorm:"size:32;index"`
	// ScenarioPrompt = brief initial de la partie.
	ScenarioPrompt string `gorm:"type:longtext"`
	// CurrentScene = scene ou etape active.
	CurrentScene string `gorm:"type:text"`
	// CurrentTurn = numero de tour pour reprise et live.
	CurrentTurn int
	// PromptTokens / CompletionTokens / TotalTokens = consommation IA cumulee.
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	// EstimatedCostMicros = cout estime en micro-USD, si les prix env sont configures.
	EstimatedCostMicros int64
	// Snapshot = etat libre de la session roleplay.
	Snapshot datatypes.JSON `gorm:"type:json"`

	ActiveCharacterID *uint              `gorm:"index"`
	ActiveCharacter   *RolePlayCharacter `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	StartedAt      *time.Time `gorm:"index"`
	LastActivityAt *time.Time `gorm:"index;index:idx_roleplay_owner_updated,priority:2"`
	FinishedAt     *time.Time `gorm:"index"`

	Turns        []RolePlaySessionTurn `gorm:"foreignKey:SessionID"`
	QuestRuns    []RolePlayQuestRun    `gorm:"foreignKey:SessionID"`
	LiveSessions []LiveSession         `gorm:"foreignKey:RolePlaySessionID"`
	Characters   []RolePlayCharacter   `gorm:"foreignKey:RolePlaySessionID"`
}

// RolePlaySessionTurn = journal incremental d'une partie roleplay.
// Sert a la reprise, au replay et au stream live.
type RolePlaySessionTurn struct {
	Id        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time

	SessionID       uint            `gorm:"index;uniqueIndex:idx_roleplay_turn_sequence"`
	RolePlaySession RolePlaySession `gorm:"foreignKey:SessionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// Turn = ordre logique des messages.
	Turn int `gorm:"index"`
	// AuthorType = player / ia / narrateur / system.
	AuthorType string `gorm:"size:32;index"`
	// AuthorName = nom du personnage ou du systeme.
	AuthorName string `gorm:"size:120;index"`
	// Content = narration ou action.
	Content string `gorm:"type:longtext"`
	// Payload = reserve pour actions structurees, jets, choix, etc.
	Payload datatypes.JSON `gorm:"type:json"`
	// Sequence = ordre strict dans la session pour le live.
	Sequence int `gorm:"uniqueIndex:idx_roleplay_turn_sequence"`
}

// CoopParty = salon de coop.
// Il peut porter soit une battle IA, soit une session roleplay.
type CoopParty struct {
	Id        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// Code = code d'invitation pour rejoindre la coop.
	Code string `gorm:"size:64;uniqueIndex"`
	// Mode = battle_ia / roleplay_ia.
	Mode string `gorm:"size:32;index"`
	// Status = waiting / running / paused / finished / closed.
	Status string `gorm:"size:32;index;index:idx_coop_host_updated,priority:2"`

	HostUserID uint  `gorm:"index;index:idx_coop_host_updated,priority:1"`
	HostUser   Users `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// BattleSaveID = rempli si la coop porte une battle IA.
	BattleSaveID *uint       `gorm:"index"`
	BattleSave   *BattleSave `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	// RolePlaySessionID = rempli si la coop porte une partie roleplay.
	RolePlaySessionID *uint            `gorm:"index"`
	RolePlaySession   *RolePlaySession `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	MaxMembers     int
	SharedState    datatypes.JSON `gorm:"type:json"`
	LastActivityAt *time.Time     `gorm:"index;index:idx_coop_host_updated,priority:3"`

	Members      []CoopPartyMember `gorm:"foreignKey:CoopPartyID"`
	LiveSessions []LiveSession     `gorm:"foreignKey:CoopPartyID"`
}

// CoopPartyMember = membre humain du salon coop.
// Sert pour savoir qui participe, qui attend, qui s'est deconnecte.
type CoopPartyMember struct {
	Id        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	CoopPartyID uint      `gorm:"index:idx_coop_user,unique;index:idx_coop_members_status,priority:1"`
	CoopParty   CoopParty `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	UserID uint  `gorm:"index:idx_coop_user,unique"`
	User   Users `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// Role = host / player / spectator.
	Role string `gorm:"size:32;index"`
	// Status = invited / joined / ready / left / disconnected.
	Status     string     `gorm:"size:32;index;index:idx_coop_members_status,priority:2"`
	JoinedAt   *time.Time `gorm:"index"`
	LastSeenAt *time.Time `gorm:"index"`

	CharacterID *uint              `gorm:"index"`
	Character   *RolePlayCharacter `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

// LiveSession = canal live generique.
// Il sert autant pour Battle IA que pour RolePlay IA.
type LiveSession struct {
	Id        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	OwnerID uint  `gorm:"index;index:idx_live_owner_updated,priority:1"`
	Owner   Users `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// ChannelKey = identifiant de stream / SSE / websocket.
	ChannelKey string `gorm:"size:120;uniqueIndex;index:idx_live_channel_owner,priority:1"`
	// Mode = battle_ia / roleplay_ia.
	Mode string `gorm:"size:32;index"`
	// Status = waiting / streaming / paused / ended.
	Status string `gorm:"size:32;index;index:idx_live_owner_updated,priority:2"`

	// Ces FK permettent de rattacher le live a la bonne source.
	BattleSaveID      *uint            `gorm:"index"`
	BattleSave        *BattleSave      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	RolePlaySessionID *uint            `gorm:"index"`
	RolePlaySession   *RolePlaySession `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	ArenaID           *uint            `gorm:"index"`
	Arena             *BattleArena     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	CoopPartyID       *uint            `gorm:"index"`
	CoopParty         *CoopParty       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	ViewerCount int
	LastEventAt *time.Time `gorm:"index;index:idx_live_owner_updated,priority:3"`
	StartedAt   *time.Time `gorm:"index"`
	EndedAt     *time.Time `gorm:"index"`
	AllowReplay bool

	Events []LiveEvent `gorm:"foreignKey:LiveSessionID"`
}

// LiveEvent = evenement unitaire du stream live.
// Un event peut etre un message IA, une action joueur, un score ou un etat systeme.
type LiveEvent struct {
	Id        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time

	LiveSessionID uint        `gorm:"index;uniqueIndex:idx_live_event_sequence"`
	LiveSession   LiveSession `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// Sequence = ordre strict dans le canal.
	Sequence int `gorm:"uniqueIndex:idx_live_event_sequence"`
	// EventType = message / chunk / score / join / leave / status.
	EventType string `gorm:"size:32;index"`
	// AuthorType = ia / player / narrateur / system.
	AuthorType string `gorm:"size:32;index"`
	// AuthorName = nom a afficher dans le flux.
	AuthorName string `gorm:"size:120;index"`
	// Payload = contenu complet de l'evenement.
	Payload datatypes.JSON `gorm:"type:json"`
}

// BattleIA = DTO runtime, pas un modele GORM.
type BattleIA struct {
	Name     string             `json:"name"`
	Provider *provider.Provider `json:"provider"`
}

// BattleIAConfig = config runtime d'une IA.
// Elle peut etre serialisee dans IASnapshot, mais n'est pas migree en table.
type BattleIAConfig struct {
	Name         string                     `json:"name"`
	Provider     *provider.Provider         `json:"-"`
	Fallbacks    []BattleIAProviderFallback `json:"-"`
	ProviderName string                     `json:"providerName"`
	ModelName    string                     `json:"modelName"`
	Personality  string                     `json:"personality"`
	Mindset      string                     `json:"mindset"`
	Style        string                     `json:"style"`
	Goal         string                     `json:"goal"`
	Weakness     string                     `json:"weakness"`
}

type BattleIAProviderFallback struct {
	Provider     *provider.Provider `json:"-"`
	ProviderName string             `json:"providerName"`
	ModelName    string             `json:"modelName"`
}

// BattleRoundMessage = morceau d'historique runtime.
type BattleRoundMessage struct {
	IA      string `json:"ia"`
	Round   int    `json:"round"`
	Content string `json:"content"`
}

// BattleIAProfile = snapshot runtime d'une IA dans un contexte de battle.
type BattleIAProfile struct {
	Name        string `json:"name"`
	Personality string `json:"personality"`
	Mindset     string `json:"mindset"`
	Style       string `json:"style"`
	Goal        string `json:"goal"`
	Weakness    string `json:"weakness"`
}

// BattleMessageContext = etat JSON persiste dans BattleSave.Context.
type BattleMessageContext struct {
	Question             string               `json:"question"`
	Round                int                  `json:"round"`
	TotalRounds          int                  `json:"totalRounds"`
	RoundDurationSeconds int                  `json:"roundDurationSeconds"`
	JudgeName            string               `json:"judgeName,omitempty"`
	JudgeSlot            int                  `json:"judgeSlot,omitempty"`
	JudgeWinnerSlot      int                  `json:"judgeWinnerSlot,omitempty"`
	JudgeScoreOne        int                  `json:"judgeScoreOne,omitempty"`
	JudgeScoreTwo        int                  `json:"judgeScoreTwo,omitempty"`
	JudgeReason          string               `json:"judgeReason,omitempty"`
	CurrentIA            string               `json:"currentIa"`
	DebatePosition       string               `json:"debatePosition"`
	OpponentPosition     string               `json:"opponentPosition"`
	ConflictInstruction  string               `json:"conflictInstruction"`
	Instruction          string               `json:"instruction"`
	IAProfile            BattleIAProfile      `json:"iaProfile"`
	MyPreviousMessages   []BattleRoundMessage `json:"myPreviousMessages"`
	OpponentMessages     []BattleRoundMessage `json:"opponentMessages"`
	AllPreviousRounds    []BattleRoundMessage `json:"allPreviousRounds"`
}

// AIUsageRecord = journal comptable des appels IA par session de jeu.
// Les tokens peuvent venir du provider ou d'une estimation locale si le provider ne remonte pas usage.
type AIUsageRecord struct {
	Id        uint `gorm:"primaryKey"`
	CreatedAt time.Time

	OwnerID uint `gorm:"index"`

	// SessionMode = battle_ia / roleplay_ia.
	SessionMode       string           `gorm:"size:32;index"`
	BattleSaveID      *uint            `gorm:"index"`
	BattleSave        *BattleSave      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	RolePlaySessionID *uint            `gorm:"index"`
	RolePlaySession   *RolePlaySession `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	// BillingSource = client_key / platform_key. Permet de distinguer gratuit vs credits plateforme.
	BillingSource string `gorm:"size:32;index"`
	ProviderName  string `gorm:"size:64;index"`
	ProviderHost  string `gorm:"size:160"`
	ModelName     string `gorm:"size:160;index"`
	Operation     string `gorm:"size:64;index"`
	Phase         string `gorm:"size:64;index"`
	Round         int    `gorm:"index"`
	ActorName     string `gorm:"size:120;index"`

	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	InputChars       int
	OutputChars      int
	Stream           bool
	Fallback         bool
	Estimated        bool

	Currency             string `gorm:"size:8"`
	InputUSDPer1MToken   float64
	OutputUSDPer1MToken  float64
	EstimatedCostMicros  int64
	PricingConfiguration string `gorm:"size:120"`
}

// NexusCoinPlan = pack commercial de credits IA expose plus tard au client Flutter.
// Les prix sont stockes en micro-USD pour rester alignes avec AIUsageRecord.EstimatedCostMicros.
type NexusCoinPlan struct {
	Id        uint `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Slug        string `gorm:"size:80;uniqueIndex" json:"slug"`
	Position    int    `gorm:"index" json:"position"`
	Name        string `gorm:"size:120" json:"name"`
	Subtitle    string `gorm:"size:180" json:"subtitle"`
	Description string `gorm:"type:text" json:"description"`
	Status      string `gorm:"size:32;index" json:"status"`

	TokenBudget            int64 `json:"tokenBudget"`
	NexusCoins             int64 `json:"nexusCoins"`
	BaseCostMicros         int64 `json:"baseCostMicros"`
	MarginPercent          int   `json:"marginPercent"`
	PriceMicros            int64 `json:"priceMicros"`
	EstimatedCalls         int64 `json:"estimatedCalls"`
	EstimatedTokensPerCall int64 `json:"estimatedTokensPerCall"`
}
