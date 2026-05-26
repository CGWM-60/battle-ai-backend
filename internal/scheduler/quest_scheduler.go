package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/provider"
	"cgwm/battle/internal/repository"
	"cgwm/battle/internal/service"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const cronQuestLimit = 10
const cronLogLimit = 300

var cronScheduleHours = []int{8, 12, 20}

type CronLogEntry struct {
	CreatedAt string
	RunID     string
	Job       string
	Hour      string
	Provider  string
	Model     string
	Step      string
	Status    string
	Message   string
}

type CronJobState struct {
	Job            string
	LastRunID      string
	LastHour       string
	LastProvider   string
	LastModel      string
	LastStatus     string
	LastStep       string
	LastMessage    string
	LastStartedAt  string
	LastFinishedAt string
	LastDurationMS int64
	LastError      string
}

type CronSnapshot struct {
	Enabled  bool
	Timezone string
	Window   string
	Limit    int
	NextRun  string
	Battle   CronJobState
	RolePlay CronJobState
	Logs     []CronLogEntry
}

var cronMemory = struct {
	sync.Mutex
	enabled  bool
	timezone string
	logs     []CronLogEntry
	states   map[string]CronJobState
}{
	timezone: env("APP_TIMEZONE", "Europe/Paris"),
	states: map[string]CronJobState{
		"battle":   {Job: "battle"},
		"roleplay": {Job: "roleplay"},
	},
}

type cronTrace struct {
	RunID    string
	Job      string
	Hour     string
	Provider string
	Model    string
}

type aiProviderConfig struct {
	Name   string
	APIKey string
	Model  string
}

type generatedBattleQuest struct {
	Title   string         `json:"title"`
	Content string         `json:"content"`
	Level   string         `json:"level"`
	Theme   string         `json:"theme"`
	Point   int            `json:"point"`
	Xp      int            `json:"xp"`
	Coin    int            `json:"coin"`
	Meta    map[string]any `json:"metadata"`
}

type generatedRolePlayQuest struct {
	Title   string                 `json:"title"`
	Summary string                 `json:"summary"`
	Prompt  string                 `json:"prompt"`
	Theme   string                 `json:"theme"`
	Level   string                 `json:"level"`
	Xp      int                    `json:"xp"`
	Coin    int                    `json:"coin"`
	Meta    map[string]any         `json:"metadata"`
	Arcs    []generatedRolePlayArc `json:"arcs"`
}

type generatedRolePlayArc struct {
	Title     string                     `json:"title"`
	Summary   string                     `json:"summary"`
	Objective string                     `json:"objective"`
	Prompt    string                     `json:"prompt"`
	Meta      map[string]any             `json:"metadata"`
	Chapters  []generatedRolePlayChapter `json:"chapters"`
}

type generatedRolePlayChapter struct {
	Title         string         `json:"title"`
	Summary       string         `json:"summary"`
	Objective     string         `json:"objective"`
	IntroPrompt   string         `json:"introPrompt"`
	SuccessPrompt string         `json:"successPrompt"`
	FailurePrompt string         `json:"failurePrompt"`
	IsBoss        bool           `json:"isBoss"`
	Xp            int            `json:"xp"`
	Coin          int            `json:"coin"`
	Meta          map[string]any `json:"metadata"`
}

func StartQuestGenerationCron(db *gorm.DB) {
	if strings.EqualFold(env("AI_QUEST_CRON_ENABLED", "true"), "false") {
		log.Printf("[quest-cron] step=boot status=disabled reason=AI_QUEST_CRON_ENABLED=false")
		setCronRuntime(false, env("APP_TIMEZONE", "Europe/Paris"))
		recordCronLog(CronLogEntry{Step: "boot", Status: "disabled", Message: "reason=AI_QUEST_CRON_ENABLED=false"})
		return
	}

	location := loadLocation()
	setCronRuntime(true, location.String())
	log.Printf(
		"[quest-cron] step=boot status=enabled timezone=%s hours=08,12,20 limit=%d battle_job=true roleplay_job=true",
		location.String(),
		cronQuestLimit,
	)
	recordCronLog(CronLogEntry{Step: "boot", Status: "enabled", Message: fmt.Sprintf("timezone=%s hours=08,12,20 limit=%d battle_job=true roleplay_job=true", location.String(), cronQuestLimit)})

	go runHourlyJob(db, location, "battle", runBattleQuestJob)
	go runHourlyJob(db, location, "roleplay", runRolePlayQuestJob)
}

func runHourlyJob(
	db *gorm.DB,
	location *time.Location,
	jobName string,
	job func(context.Context, *gorm.DB, time.Time, aiProviderConfig, cronTrace) error,
) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	log.Printf("[quest-cron] job=%s step=worker_start status=running", jobName)
	recordCronLog(CronLogEntry{Job: jobName, Step: "worker_start", Status: "running"})

	var lastRun string
	for {
		now := <-ticker.C
		localNow := now.In(location)
		if localNow.Minute() != 0 {
			continue
		}
		if !isScheduledCronHour(localNow.Hour()) {
			continue
		}

		runKey := localNow.Format("2006010215")
		if runKey == lastRun {
			continue
		}
		lastRun = runKey
		runID := fmt.Sprintf("%s-%s", jobName, runKey)

		cfg, ok := providerForHour(localNow)
		if !ok {
			expectedProvider := providerNameForHour(localNow)
			log.Printf(
				"[quest-cron] run_id=%s job=%s step=provider_config status=skipped hour=%s expected_provider=%s reason=missing_api_key_or_model",
				runID,
				jobName,
				runKey,
				expectedProvider,
			)
			recordCronLog(CronLogEntry{
				RunID:    runID,
				Job:      jobName,
				Hour:     runKey,
				Provider: expectedProvider,
				Step:     "provider_config",
				Status:   "skipped",
				Message:  "reason=missing_api_key_or_model",
			})
			continue
		}

		trace := cronTrace{
			RunID:    runID,
			Job:      jobName,
			Hour:     runKey,
			Provider: cfg.Name,
			Model:    cfg.Model,
		}
		startedAt := time.Now()
		trace.log(
			"trigger",
			"started",
			"local_time=%s utc_time=%s key_len=%d",
			localNow.Format(time.RFC3339),
			now.UTC().Format(time.RFC3339),
			len(cfg.APIKey),
		)

		ctx, cancel := context.WithTimeout(context.Background(), questCronTimeout())
		err := withMySQLLock(ctx, db, "go_battle_ia_"+jobName+"_"+runKey, trace, func(ctx context.Context) error {
			return job(ctx, db, localNow, cfg, trace)
		})
		cancel()
		if err != nil {
			trace.log("run", "failed", "duration_ms=%d err=%v", time.Since(startedAt).Milliseconds(), err)
			continue
		}

		trace.log("run", "completed", "duration_ms=%d", time.Since(startedAt).Milliseconds())
	}
}

func runBattleQuestJob(ctx context.Context, db *gorm.DB, runAt time.Time, cfg aiProviderConfig, trace cronTrace) error {
	trace.log("generate", "started", "limit=%d", cronQuestLimit)
	quests, err := generateBattleQuests(ctx, cfg, trace)
	if err != nil {
		return fmt.Errorf("generate battle quests: %w", err)
	}
	trace.log("generate", "completed", "items=%d", len(quests))

	created := 0
	skipped := 0
	failed := 0
	for index, item := range quests {
		if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.Content) == "" {
			skipped++
			trace.log("insert", "skipped", "index=%d reason=missing_title_or_content title=%q", index, item.Title)
			continue
		}
		meta, _ := json.Marshal(ensureMetadata(item.Meta, cfg, runAt))
		slug := uniqueSlug(item.Title, runAt, index)
		quest := models.QuestIaBattle{
			Slug:     slug,
			Title:    item.Title,
			Content:  item.Content,
			Level:    item.Level,
			Point:    item.Point,
			Theme:    item.Theme,
			Xp:       item.Xp,
			Coin:     item.Coin,
			Mode:     constants.ModeBattleIA,
			Source:   "cron_ai",
			Status:   constants.QuestStatusPublished,
			Metadata: datatypes.JSON(meta),
		}
		if err := db.WithContext(ctx).Create(&quest).Error; err == nil {
			created++
			trace.log("insert", "created", "index=%d id=%d slug=%s title=%q theme=%s level=%s", index, quest.Id, slug, item.Title, item.Theme, item.Level)
		} else {
			failed++
			trace.log("insert", "failed", "index=%d slug=%s title=%q err=%v", index, slug, item.Title, err)
		}
	}
	trace.log("insert", "summary", "received=%d created=%d skipped=%d failed=%d", len(quests), created, skipped, failed)

	if created == 0 {
		return fmt.Errorf("no battle quest created")
	}
	return nil
}

func runRolePlayQuestJob(ctx context.Context, db *gorm.DB, runAt time.Time, cfg aiProviderConfig, trace cronTrace) error {
	trace.log("generate", "started", "limit=%d", cronQuestLimit)
	quests, err := generateRolePlayQuests(ctx, cfg, trace)
	if err != nil {
		return fmt.Errorf("generate roleplay quests: %w", err)
	}
	trace.log("generate", "completed", "items=%d", len(quests))

	created := 0
	skipped := 0
	failed := 0
	for index, item := range quests {
		if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.Prompt) == "" {
			skipped++
			trace.log("insert", "skipped", "index=%d reason=missing_title_or_prompt title=%q", index, item.Title)
			continue
		}
		if !hasGeneratedRolePlayStructure(item) {
			skipped++
			trace.log("insert", "skipped", "index=%d reason=invalid_arc_chapter_structure title=%q arcs=%d", index, item.Title, len(item.Arcs))
			continue
		}
		slug := uniqueSlug(item.Title, runAt, index)
		input := generatedRolePlayQuestInput(item, cfg, runAt, slug, "cron_ai")
		quest, err := service.NewQuestService(repository.NewQuestRepository(db)).CreateRolePlay(ctx, input)
		if err == nil {
			created++
			trace.log("insert", "created", "index=%d id=%d slug=%s title=%q theme=%s level=%s", index, quest.Id, slug, item.Title, item.Theme, item.Level)
		} else {
			failed++
			trace.log("insert", "failed", "index=%d slug=%s title=%q err=%v", index, slug, item.Title, err)
		}
	}
	trace.log("insert", "summary", "received=%d created=%d skipped=%d failed=%d", len(quests), created, skipped, failed)

	if created == 0 {
		return fmt.Errorf("no roleplay quest created")
	}
	return nil
}

func generateBattleQuests(ctx context.Context, cfg aiProviderConfig, trace cronTrace) ([]generatedBattleQuest, error) {
	prompt := fmt.Sprintf(`Genere exactement %d quetes pour un jeu de battle entre IA.
Reponds uniquement en JSON valide, sans markdown.
Les IA sont seulement les participantes du debat: les sujets ne doivent pas tourner principalement autour de l'IA.
Les sujets doivent etre varies, amusants, originaux, ouverts, parfois absurdes, mais toujours debatables.
Au moins 6 quetes doivent etre non technologiques.
Au moins 5 quetes doivent parler de vie quotidienne, culture generale, sport, cuisine, famille, travail, ecole, cinema, musique, ville, voyage, morale ou societe.
Maximum 1 quete peut parler d'IA, de robots ou de technologie numerique.
Au minimum 4 quetes doivent avoir un angle clairement humoristique.
Ces 4 quetes humoristiques doivent rester argumentables et opposer au moins deux camps.
Evite les questions classiques, scolaires, trop generales ou deja vues.
Evite les themes IA/robots/algorithmes sauf exception unique.
Ne commence pas plus de 2 questions par "Faut-il".
Ne commence pas plus de 2 questions par "Est-ce que".
Format:
[{"title":"...","content":"question debat complete","level":"facile|moyen|difficile","theme":"...","point":10,"xp":25,"coin":5,"metadata":{"angle":"...","humour":true}}]`, cronQuestLimit)

	response, err := callProvider(ctx, cfg, prompt, trace)
	if err != nil {
		return nil, fmt.Errorf("call provider: %w", err)
	}

	cleaned := cleanJSON(response)
	trace.log("json_clean", "completed", "raw_chars=%d cleaned_chars=%d", len(response), len(cleaned))

	var quests []generatedBattleQuest
	if err := json.Unmarshal([]byte(cleaned), &quests); err != nil {
		trace.log("json_parse", "failed", "err=%v preview=%q", err, preview(cleaned, 500))
		return nil, fmt.Errorf("parse battle JSON: %w", err)
	}
	return quests, nil
}

func generateRolePlayQuests(ctx context.Context, cfg aiProviderConfig, trace cronTrace) ([]generatedRolePlayQuest, error) {
	quests := make([]generatedRolePlayQuest, 0, cronQuestLimit)
	for len(quests) < cronQuestLimit {
		batchCount := minInt(rolePlayGenerationBatchSize(), cronQuestLimit-len(quests))
		trace.log("generate_batch", "started", "batch_size=%d current_total=%d target=%d", batchCount, len(quests), cronQuestLimit)
		batch, err := generateRolePlayQuestsBatch(ctx, cfg, trace, batchCount)
		if err != nil {
			if len(quests) > 0 {
				trace.log("generate_batch", "partial", "items=%d err=%v", len(quests), err)
				return quests, nil
			}
			return nil, err
		}
		if len(batch) == 0 {
			if len(quests) > 0 {
				trace.log("generate_batch", "partial", "items=%d err=empty_batch", len(quests))
				return quests, nil
			}
			return nil, fmt.Errorf("provider returned no roleplay quest")
		}
		quests = append(quests, batch...)
		trace.log("generate_batch", "completed", "batch_items=%d total=%d", len(batch), len(quests))
	}
	return quests, nil
}

func generateRolePlayQuestsBatch(ctx context.Context, cfg aiProviderConfig, trace cronTrace, count int) ([]generatedRolePlayQuest, error) {
	prompt := fmt.Sprintf(`Genere exactement %d quetes de jeu de role.
Reponds uniquement en JSON valide, sans markdown.
Les quetes doivent etre variees, jouables, immersives et differentes les unes des autres.
Les quetes doivent avoir une longueur variable selon leur niveau.
Tu dois varier les niveaux dans le batch: environ 35%% facile, 40%% moyen, 25%% difficile quand le count le permet.
Regles de structure obligatoires:
- level "facile": 2 ou 3 arcs; chaque arc a 2 ou 3 chapitres; total 4 a 7 chapitres.
- level "moyen": 3 ou 4 arcs; chaque arc a 2 a 4 chapitres; total 7 a 12 chapitres.
- level "difficile": 4 a 6 arcs; chaque arc a 3 a 5 chapitres; total 12 a 24 chapitres.
Ne genere pas toujours la structure minimale. Dans un meme batch, alterne le nombre d'arcs et de chapitres.
Ajoute exactement un chapitre boss pour les quetes difficiles, et zero ou un chapitre boss pour les autres.
Les arcs et chapitres doivent etre clairement identifies et ordonnes par leur position dans le tableau JSON.
Format:
[{"title":"...","summary":"resume court","prompt":"prompt global de la quete","theme":"fantasy|sf|horreur|steampunk|modern","level":"facile|moyen|difficile","xp":80,"coin":30,"metadata":{"ton":"..."},"arcs":[{"title":"Arc 1","summary":"...","objective":"...","prompt":"brief de l'arc","metadata":{"tone":"..."},"chapters":[{"title":"Chapitre 1","summary":"...","objective":"objectif jouable","introPrompt":"situation initiale du chapitre","successPrompt":"consequence en cas de reussite","failurePrompt":"consequence en cas d'echec","isBoss":false,"xp":20,"coin":8,"metadata":{"stakes":"..."}}]}]}]`, count)

	response, err := callProvider(ctx, cfg, prompt, trace)
	if err != nil {
		return nil, fmt.Errorf("call provider: %w", err)
	}

	cleaned := cleanJSON(response)
	trace.log("json_clean", "completed", "raw_chars=%d cleaned_chars=%d", len(response), len(cleaned))

	var quests []generatedRolePlayQuest
	if err := json.Unmarshal([]byte(cleaned), &quests); err != nil {
		trace.log("json_parse", "failed", "err=%v preview=%q", err, preview(cleaned, 500))
		return nil, fmt.Errorf("parse roleplay JSON: %w", err)
	}
	return quests, nil
}

func generatedRolePlayQuestInput(item generatedRolePlayQuest, cfg aiProviderConfig, runAt time.Time, slug string, source string) service.RolePlayQuestInput {
	return service.RolePlayQuestInput{
		Slug:     slug,
		Title:    item.Title,
		Summary:  item.Summary,
		Prompt:   item.Prompt,
		Theme:    item.Theme,
		Level:    item.Level,
		Xp:       item.Xp,
		Coin:     item.Coin,
		Source:   source,
		Status:   constants.QuestStatusPublished,
		Metadata: ensureMetadata(item.Meta, cfg, runAt),
		Arcs:     generatedRolePlayArcInputs(item.Arcs),
	}
}

func generatedRolePlayArcInputs(items []generatedRolePlayArc) []service.RolePlayQuestArcInput {
	arcs := make([]service.RolePlayQuestArcInput, 0, len(items))
	for index, item := range items {
		arcs = append(arcs, service.RolePlayQuestArcInput{
			Position:  index + 1,
			Title:     item.Title,
			Summary:   item.Summary,
			Objective: item.Objective,
			Prompt:    item.Prompt,
			Metadata:  item.Meta,
			Chapters:  generatedRolePlayChapterInputs(item.Chapters),
		})
	}
	return arcs
}

func generatedRolePlayChapterInputs(items []generatedRolePlayChapter) []service.RolePlayQuestChapterInput {
	chapters := make([]service.RolePlayQuestChapterInput, 0, len(items))
	for index, item := range items {
		chapters = append(chapters, service.RolePlayQuestChapterInput{
			Position:      index + 1,
			Title:         item.Title,
			Summary:       item.Summary,
			Objective:     item.Objective,
			IntroPrompt:   item.IntroPrompt,
			SuccessPrompt: item.SuccessPrompt,
			FailurePrompt: item.FailurePrompt,
			IsBoss:        item.IsBoss,
			Xp:            item.Xp,
			Coin:          item.Coin,
			Metadata:      item.Meta,
		})
	}
	return chapters
}

func hasGeneratedRolePlayStructure(item generatedRolePlayQuest) bool {
	minArcs, maxArcs, minChapters, maxChapters := rolePlayStructureBounds(item.Level)
	if len(item.Arcs) < minArcs || len(item.Arcs) > maxArcs {
		return false
	}
	totalChapters := 0
	for _, arc := range item.Arcs {
		if strings.TrimSpace(arc.Title) == "" || len(arc.Chapters) < 2 {
			return false
		}
		totalChapters += len(arc.Chapters)
		for _, chapter := range arc.Chapters {
			if strings.TrimSpace(chapter.Title) == "" || strings.TrimSpace(chapter.Objective) == "" {
				return false
			}
		}
	}
	if totalChapters < minChapters || totalChapters > maxChapters {
		return false
	}
	return true
}

func rolePlayStructureBounds(level string) (minArcs int, maxArcs int, minChapters int, maxChapters int) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "difficile", "hard":
		return 4, 6, 12, 24
	case "moyen", "medium", "normal":
		return 3, 4, 7, 12
	default:
		return 2, 3, 4, 7
	}
}

func callProvider(ctx context.Context, cfg aiProviderConfig, prompt string, trace cronTrace) (string, error) {
	trace.log("provider_url", "started", "provider=%s", cfg.Name)
	url, err := service.ProviderURL(cfg.Name)
	if err != nil {
		trace.log("provider_url", "failed", "err=%v", err)
		return "", fmt.Errorf("resolve provider url: %w", err)
	}
	trace.log("provider_url", "completed", "url=%s", url)

	startedAt := time.Now()
	trace.log("provider_call", "started", "prompt_chars=%d key_len=%d", len(prompt), len(cfg.APIKey))
	ai := provider.NewsProvider(cfg.APIKey, url, cfg.Model)
	response, err := ai.Chat(ctx, []provider.ProviderMessage{{Role: "system", Content: prompt}})
	if err != nil {
		trace.log("provider_call", "failed", "duration_ms=%d err=%v", time.Since(startedAt).Milliseconds(), err)
		return "", err
	}
	trace.log("provider_call", "completed", "duration_ms=%d response_chars=%d", time.Since(startedAt).Milliseconds(), len(response))
	return response, nil
}

func providerForHour(now time.Time) (aiProviderConfig, bool) {
	return providerConfig(providerNameForHour(now), providerKeyEnvForHour(now), providerModelEnvForHour(now))
}

func providerNameForHour(now time.Time) string {
	rotation := cronProviderRotation()
	slotIndex, ok := scheduledCronSlotIndex(now.Hour())
	if !ok {
		slotIndex = 0
	}
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayIndex := int(midnight.Unix() / int64((24 * time.Hour).Seconds()))
	return rotation[(dayIndex*len(cronScheduleHours)+slotIndex)%len(rotation)]
}

func isScheduledCronHour(hour int) bool {
	_, ok := scheduledCronSlotIndex(hour)
	return ok
}

func scheduledCronSlotIndex(hour int) (int, bool) {
	for index, scheduledHour := range cronScheduleHours {
		if hour == scheduledHour {
			return index, true
		}
	}
	return 0, false
}

func providerKeyEnvForHour(now time.Time) string {
	switch providerNameForHour(now) {
	case "mistral":
		return "MISTRAL_AI_KEY"
	case "claude", "anthropic":
		return "ANTHROPIC_AI_KEY"
	case "gemini", "google", "google_ai", "google-ai":
		return "GEMINI_AI_KEY"
	default:
		return "OPEN_AI_KEY"
	}
}

func providerModelEnvForHour(now time.Time) string {
	switch providerNameForHour(now) {
	case "mistral":
		return "MISTRAL_AI_MODEL"
	case "claude", "anthropic":
		return "ANTHROPIC_AI_MODEL"
	case "gemini", "google", "google_ai", "google-ai":
		return "GEMINI_AI_MODEL"
	default:
		return "OPEN_AI_MODEL"
	}
}

func cronProviderRotation() []string {
	raw := strings.TrimSpace(env("AI_QUEST_PROVIDER_ROTATION", "mistral,openai"))
	parts := strings.Split(raw, ",")
	rotation := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.ToLower(strings.TrimSpace(part))
		switch name {
		case "mistral", "openai", "claude", "anthropic", "gemini", "google", "google_ai", "google-ai":
			if name == "anthropic" {
				name = "claude"
			}
			if name == "google" || name == "google_ai" || name == "google-ai" {
				name = "gemini"
			}
			rotation = append(rotation, name)
		}
	}
	if len(rotation) == 0 {
		return []string{"mistral", "openai"}
	}
	return rotation
}

func questCronTimeout() time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(env("AI_QUEST_CRON_TIMEOUT_SECONDS", "900")))
	if err != nil || seconds <= 0 {
		seconds = 900
	}
	return time.Duration(seconds) * time.Second
}

func rolePlayGenerationBatchSize() int {
	size, err := strconv.Atoi(strings.TrimSpace(env("AI_RP_GENERATION_BATCH_SIZE", "2")))
	if err != nil || size <= 0 {
		return 2
	}
	if size > 5 {
		return 5
	}
	return size
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func providerConfig(name string, keyEnv string, modelEnv string) (aiProviderConfig, bool) {
	cfg := aiProviderConfig{
		Name:   name,
		APIKey: strings.TrimSpace(os.Getenv(keyEnv)),
		Model:  strings.TrimSpace(os.Getenv(modelEnv)),
	}
	ok := cfg.APIKey != "" && cfg.Model != ""
	if !ok {
		log.Printf(
			"[quest-cron] step=provider_config status=invalid provider=%s key_env=%s key_present=%t model_env=%s model_present=%t",
			name,
			keyEnv,
			cfg.APIKey != "",
			modelEnv,
			cfg.Model != "",
		)
		recordCronLog(CronLogEntry{
			Provider: name,
			Step:     "provider_config",
			Status:   "invalid",
			Message: fmt.Sprintf(
				"key_env=%s key_present=%t model_env=%s model_present=%t",
				keyEnv,
				cfg.APIKey != "",
				modelEnv,
				cfg.Model != "",
			),
		})
	}
	return cfg, ok
}

func withMySQLLock(ctx context.Context, db *gorm.DB, name string, trace cronTrace, fn func(context.Context) error) error {
	trace.log("mysql_lock", "started", "lock=%s", name)
	sqlDB, err := db.DB()
	if err != nil {
		trace.log("mysql_lock", "failed", "phase=db_pool err=%v", err)
		return fmt.Errorf("get DB pool: %w", err)
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		trace.log("mysql_lock", "failed", "phase=conn err=%v", err)
		return fmt.Errorf("get DB conn: %w", err)
	}
	defer conn.Close()

	acquired, err := acquireLock(ctx, conn, name)
	if err != nil {
		trace.log("mysql_lock", "failed", "phase=acquire err=%v", err)
		return fmt.Errorf("acquire mysql lock: %w", err)
	}
	if !acquired {
		trace.log("mysql_lock", "skipped", "lock=%s reason=already_locked", name)
		return nil
	}
	trace.log("mysql_lock", "acquired", "lock=%s", name)
	defer releaseLock(ctx, conn, name, trace)

	return fn(ctx)
}

func acquireLock(ctx context.Context, conn *sql.Conn, name string) (bool, error) {
	var acquired int
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", name).Scan(&acquired); err != nil {
		return false, err
	}
	return acquired == 1, nil
}

func releaseLock(ctx context.Context, conn *sql.Conn, name string, trace cronTrace) {
	var released int
	if err := conn.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?)", name).Scan(&released); err != nil {
		trace.log("mysql_lock", "release_failed", "lock=%s err=%v", name, err)
		return
	}
	trace.log("mysql_lock", "released", "lock=%s released=%d", name, released)
}

func ensureMetadata(meta map[string]any, cfg aiProviderConfig, runAt time.Time) map[string]any {
	if meta == nil {
		meta = map[string]any{}
	}
	meta["generatedBy"] = "quest-cron"
	meta["providerName"] = cfg.Name
	meta["modelName"] = cfg.Model
	meta["generatedAt"] = runAt.Format(time.RFC3339)
	return meta
}

func cleanJSON(value string) string {
	clean := strings.TrimSpace(value)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	start := strings.IndexAny(clean, "[{")
	endArray := strings.LastIndex(clean, "]")
	endObject := strings.LastIndex(clean, "}")
	end := endArray
	if endObject > end {
		end = endObject
	}
	if start >= 0 && end >= start {
		return clean[start : end+1]
	}
	return clean
}

func uniqueSlug(title string, runAt time.Time, index int) string {
	return fmt.Sprintf("%s-%s-%02d", slugify(title), runAt.Format("20060102150405"), index)
}

func slugify(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, slug)
	if slug == "" {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return slug
}

func loadLocation() *time.Location {
	name := env("APP_TIMEZONE", "Europe/Paris")
	location, err := time.LoadLocation(name)
	if err != nil {
		log.Printf("[quest-cron] invalid APP_TIMEZONE=%s err=%v, using Local", name, err)
		recordCronLog(CronLogEntry{Step: "timezone", Status: "invalid", Message: fmt.Sprintf("APP_TIMEZONE=%s err=%v using=Local", name, err)})
		return time.Local
	}
	return location
}

func env(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func (t cronTrace) log(step string, status string, format string, args ...any) {
	message := ""
	if format != "" {
		message = fmt.Sprintf(format, args...)
	}
	log.Printf(
		"[quest-cron] run_id=%s job=%s hour=%s provider=%s model=%s step=%s status=%s%s",
		t.RunID,
		t.Job,
		t.Hour,
		t.Provider,
		t.Model,
		step,
		status,
		formatLogMessage(message),
	)
	recordCronLog(CronLogEntry{
		RunID:    t.RunID,
		Job:      t.Job,
		Hour:     t.Hour,
		Provider: t.Provider,
		Model:    t.Model,
		Step:     step,
		Status:   status,
		Message:  message,
	})
}

func Snapshot() CronSnapshot {
	cronMemory.Lock()
	defer cronMemory.Unlock()

	logs := make([]CronLogEntry, len(cronMemory.logs))
	copy(logs, cronMemory.logs)

	return CronSnapshot{
		Enabled:  cronMemory.enabled,
		Timezone: cronMemory.timezone,
		Window:   "08:00, 12:00, 20:00",
		Limit:    cronQuestLimit,
		NextRun:  nextRunLocked(),
		Battle:   cronMemory.states["battle"],
		RolePlay: cronMemory.states["roleplay"],
		Logs:     logs,
	}
}

func setCronRuntime(enabled bool, timezone string) {
	cronMemory.Lock()
	defer cronMemory.Unlock()
	cronMemory.enabled = enabled
	cronMemory.timezone = timezone
	if cronMemory.states == nil {
		cronMemory.states = map[string]CronJobState{}
	}
	for _, job := range []string{"battle", "roleplay"} {
		state := cronMemory.states[job]
		state.Job = job
		cronMemory.states[job] = state
	}
}

func recordCronLog(entry CronLogEntry) {
	now := time.Now()
	if entry.CreatedAt == "" {
		entry.CreatedAt = now.Format(time.RFC3339)
	}

	cronMemory.Lock()
	defer cronMemory.Unlock()

	cronMemory.logs = append([]CronLogEntry{entry}, cronMemory.logs...)
	if len(cronMemory.logs) > cronLogLimit {
		cronMemory.logs = cronMemory.logs[:cronLogLimit]
	}

	if entry.Job == "" {
		return
	}
	if cronMemory.states == nil {
		cronMemory.states = map[string]CronJobState{}
	}

	state := cronMemory.states[entry.Job]
	state.Job = entry.Job
	state.LastRunID = firstNonEmpty(entry.RunID, state.LastRunID)
	state.LastHour = firstNonEmpty(entry.Hour, state.LastHour)
	state.LastProvider = firstNonEmpty(entry.Provider, state.LastProvider)
	state.LastModel = firstNonEmpty(entry.Model, state.LastModel)
	state.LastStatus = entry.Status
	state.LastStep = entry.Step
	state.LastMessage = entry.Message

	if entry.Step == "trigger" && entry.Status == "started" {
		state.LastStartedAt = entry.CreatedAt
		state.LastFinishedAt = ""
		state.LastDurationMS = 0
		state.LastError = ""
	}
	if entry.Step == "run" && (entry.Status == "completed" || entry.Status == "failed") {
		state.LastFinishedAt = entry.CreatedAt
		state.LastDurationMS = parseLogInt(entry.Message, "duration_ms")
		if entry.Status == "failed" {
			state.LastError = entry.Message
		}
	}

	cronMemory.states[entry.Job] = state
}

func nextRunLocked() string {
	if !cronMemory.enabled {
		return "-"
	}
	location, err := time.LoadLocation(cronMemory.timezone)
	if err != nil {
		location = time.Local
	}
	next := time.Now().In(location).Truncate(time.Hour).Add(time.Hour)
	for i := 0; i < 72; i++ {
		if isScheduledCronHour(next.Hour()) {
			return next.Format(time.RFC3339)
		}
		next = next.Add(time.Hour)
	}
	return "-"
}

func formatLogMessage(message string) string {
	if message == "" {
		return ""
	}
	return " " + message
}

func firstNonEmpty(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func parseLogInt(message string, key string) int64 {
	prefix := key + "="
	for _, part := range strings.Fields(message) {
		if !strings.HasPrefix(part, prefix) {
			continue
		}
		value := strings.TrimPrefix(part, prefix)
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func preview(value string, maxLength int) string {
	clean := strings.ReplaceAll(value, "\n", " ")
	clean = strings.ReplaceAll(clean, "\r", " ")
	if len(clean) <= maxLength {
		return clean
	}
	return clean[:maxLength] + "...(truncated)"
}
