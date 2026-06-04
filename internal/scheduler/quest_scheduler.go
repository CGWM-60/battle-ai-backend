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
	tribunalmodels "cgwm/battle/internal/nexus_tribunal/models"
	tribunalprompts "cgwm/battle/internal/nexus_tribunal/prompts"
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
	Tribunal CronJobState
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
		"tribunal": {Job: "tribunal"},
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
		"[quest-cron] step=boot status=enabled timezone=%s hours=08,12,20 limit=%d battle_job=true roleplay_job=true tribunal_job=true",
		location.String(),
		cronQuestLimit,
	)
	recordCronLog(CronLogEntry{Step: "boot", Status: "enabled", Message: fmt.Sprintf("timezone=%s hours=08,12,20 limit=%d battle_job=true roleplay_job=true tribunal_job=true", location.String(), cronQuestLimit)})

	go runHourlyJob(db, location, "battle", runBattleQuestJob)
	go runHourlyJob(db, location, "roleplay", runRolePlayQuestJob)
	go runHourlyJob(db, location, "tribunal", runTribunalCaseJob)
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
		// Do not hard-fail the cron if the AI returned data but all were filtered by structure rules.
		// Log the situation instead of returning error (prevents the whole job from being marked failed).
		trace.log("insert", "warning", "received=%d but created=0 after structure filtering (rules may still be too strict)", len(quests))
		// Return nil so the cron doesn't treat this as total failure.
		return nil
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

	response, usedProvider, err := callProviderWithFallbacks(ctx, cfg, prompt, trace)
	if err != nil {
		return nil, fmt.Errorf("call provider: %w", err)
	}
	if usedProvider.Name != cfg.Name || usedProvider.Model != cfg.Model {
		trace.log("generate", "provider_fallback_used", "provider=%s model=%s", usedProvider.Name, usedProvider.Model)
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
- Un chapitre n'est jamais un seul tour joueur. Prevois un vrai rythme jouable:
  mini chapitre = 5 a 7 tours, chapitre standard = 8 a 12 tours, gros chapitre = 12 a 18 tours, boss/final = 15 a 25 tours.
- Dans metadata de chaque chapitre, ajoute "chapterType": "mini|standard|large|boss" et "turnRange": {"min":nombre,"max":nombre}.
- Utilise mini pour intro/tutoriel/rencontre rapide, standard pour exploration+choix+petit conflit, large pour enquete/donjon/mission importante, boss pour combat long/gros choix narratif/fin d'arc.
Ne genere pas toujours la structure minimale. Dans un meme batch, alterne le nombre d'arcs et de chapitres.
Ajoute exactement un chapitre boss pour les quetes difficiles, et zero ou un chapitre boss pour les autres.
Les arcs et chapitres doivent etre clairement identifies et ordonnes par leur position dans le tableau JSON.
Format:
[{"title":"...","summary":"resume court","prompt":"prompt global de la quete","theme":"fantasy|sf|horreur|steampunk|modern","level":"facile|moyen|difficile","xp":80,"coin":30,"metadata":{"ton":"..."},"arcs":[{"title":"Arc 1","summary":"...","objective":"...","prompt":"brief de l'arc","metadata":{"tone":"..."},"chapters":[{"title":"Chapitre 1","summary":"...","objective":"objectif jouable","introPrompt":"situation initiale du chapitre","successPrompt":"consequence en cas de reussite","failurePrompt":"consequence en cas d'echec","isBoss":false,"xp":20,"coin":8,"metadata":{"stakes":"..."}}]}]}]`, count)

	response, usedProvider, err := callProviderWithFallbacks(ctx, cfg, prompt, trace)
	if err != nil {
		return nil, fmt.Errorf("call provider: %w", err)
	}
	if usedProvider.Name != cfg.Name || usedProvider.Model != cfg.Model {
		trace.log("generate_batch", "provider_fallback_used", "provider=%s model=%s", usedProvider.Name, usedProvider.Model)
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
		Arcs:     generatedRolePlayArcInputs(item.Arcs, item.Level),
	}
}

func generatedRolePlayArcInputs(items []generatedRolePlayArc, level string) []service.RolePlayQuestArcInput {
	arcs := make([]service.RolePlayQuestArcInput, 0, len(items))
	for index, item := range items {
		arcs = append(arcs, service.RolePlayQuestArcInput{
			Position:  index + 1,
			Title:     item.Title,
			Summary:   item.Summary,
			Objective: item.Objective,
			Prompt:    item.Prompt,
			Metadata:  item.Meta,
			Chapters:  generatedRolePlayChapterInputs(item.Chapters, level),
		})
	}
	return arcs
}

func generatedRolePlayChapterInputs(items []generatedRolePlayChapter, level string) []service.RolePlayQuestChapterInput {
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
			Metadata:      rolePlayChapterPacingMetadata(item.Meta, level, item.IsBoss),
		})
	}
	return chapters
}

func rolePlayChapterPacingMetadata(meta map[string]any, level string, isBoss bool) map[string]any {
	out := map[string]any{}
	for key, value := range meta {
		out[key] = value
	}
	chapterType := strings.ToLower(strings.TrimSpace(fmt.Sprint(out["chapterType"])))
	if chapterType == "" {
		if isBoss {
			chapterType = "boss"
		} else {
			chapterType = "standard"
		}
		out["chapterType"] = chapterType
	}
	if _, ok := out["turnRange"]; !ok {
		minTurns, maxTurns := rolePlayChapterTurnRange(chapterType, level)
		out["turnRange"] = map[string]any{"min": minTurns, "max": maxTurns}
	}
	return out
}

func rolePlayChapterTurnRange(chapterType string, level string) (int, int) {
	difficultyOffset := 0
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "difficile", "hard":
		difficultyOffset = 2
	case "moyen", "medium", "normal":
		difficultyOffset = 1
	}
	switch strings.ToLower(strings.TrimSpace(chapterType)) {
	case "mini":
		return 5 + difficultyOffset, 7
	case "large", "gros":
		return 12 + difficultyOffset*2, 18
	case "boss", "final":
		return 15 + difficultyOffset*3, 25
	default:
		return 8 + difficultyOffset*2, 12
	}
}

func hasGeneratedRolePlayStructure(item generatedRolePlayQuest) bool {
	minArcs, _, minChapters, _ := rolePlayStructureBounds(item.Level)
	// Relaxed validation: only enforce minimums.
	// The previous strict min+max (especially 4-6 arcs / 12-24 chapters for "difficile")
	// was causing almost all AI generations to be rejected.
	if len(item.Arcs) < minArcs {
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
	if totalChapters < minChapters {
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

	callCtx, cancel := context.WithTimeout(ctx, questProviderCallTimeout())
	defer cancel()

	startedAt := time.Now()
	trace.log("provider_call", "started", "prompt_chars=%d key_len=%d timeout=%s", len(prompt), len(cfg.APIKey), questProviderCallTimeout())
	ai := provider.NewsProvider(cfg.APIKey, url, cfg.Model)
	response, err := ai.Chat(callCtx, []provider.ProviderMessage{{Role: "system", Content: prompt}})
	if err != nil {
		trace.log("provider_call", "failed", "duration_ms=%d err=%v", time.Since(startedAt).Milliseconds(), err)
		return "", err
	}
	trace.log("provider_call", "completed", "duration_ms=%d response_chars=%d", time.Since(startedAt).Milliseconds(), len(response))
	return response, nil
}

func callProviderWithFallbacks(ctx context.Context, primary aiProviderConfig, prompt string, trace cronTrace) (string, aiProviderConfig, error) {
	attempts := providerAttempts(primary)
	var lastErr error
	for index, cfg := range attempts {
		attemptTrace := trace
		attemptTrace.Provider = cfg.Name
		attemptTrace.Model = cfg.Model
		if index > 0 {
			attemptTrace.log("provider_fallback", "started", "from=%s attempt=%d", primary.Name, index+1)
		}
		response, err := callProvider(ctx, cfg, prompt, attemptTrace)
		if err == nil {
			if index > 0 {
				attemptTrace.log("provider_fallback", "completed", "from=%s attempt=%d", primary.Name, index+1)
			}
			return response, cfg, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			break
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no configured provider available")
	}
	return "", aiProviderConfig{}, lastErr
}

func providerForHour(now time.Time) (aiProviderConfig, bool) {
	return providerConfigForName(providerNameForHour(now))
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
	return providerKeyEnvForName(providerNameForHour(now))
}

func providerModelEnvForHour(now time.Time) string {
	return providerModelEnvForName(providerNameForHour(now))
}

func providerKeyEnvForName(name string) string {
	switch name {
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

func providerModelEnvForName(name string) string {
	switch name {
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

func providerConfigForName(name string) (aiProviderConfig, bool) {
	return providerConfig(name, providerKeyEnvForName(name), providerModelEnvForName(name))
}

func providerAttempts(primary aiProviderConfig) []aiProviderConfig {
	attempts := []aiProviderConfig{primary}
	seen := map[string]bool{strings.ToLower(strings.TrimSpace(primary.Name)): true}
	for _, name := range cronProviderRotation() {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		cfg, ok := providerConfigForName(name)
		if ok {
			attempts = append(attempts, cfg)
		}
	}
	return attempts
}

func questCronTimeout() time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(env("AI_QUEST_CRON_TIMEOUT_SECONDS", "900")))
	if err != nil || seconds <= 0 {
		seconds = 900
	}
	return time.Duration(seconds) * time.Second
}

func questProviderCallTimeout() time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(env("AI_QUEST_PROVIDER_CALL_TIMEOUT_SECONDS", "60")))
	if err != nil || seconds <= 0 {
		seconds = 60
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
		Tribunal: cronMemory.states["tribunal"],
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
	for _, job := range []string{"battle", "roleplay", "tribunal"} {
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

var tribunalCronAvatarAssetIDs = map[string]bool{
	"tribunal.character.judge_ai":              true,
	"tribunal.character.prosecutor_ai":         true,
	"tribunal.character.defense_ai":            true,
	"tribunal.character.witness_default":       true,
	"tribunal.character.clerk_ai":              true,
	"tribunal.character.fact_checker_ai":       true,
	"tribunal.character.jury_logic":            true,
	"tribunal.character.jury_emotional":        true,
	"tribunal.character.jury_expert":           true,
	"tribunal.character.assistant_ai":          true,
	"tribunal.character.expert_witness":        true,
	"tribunal.character.witness_civilian":      true,
	"tribunal.character.witness_agent":         true,
	"tribunal.character.witness_hacker":        true,
	"tribunal.character.witness_guild_master":  true,
	"tribunal.character.witness_faction_envoy": true,
	"tribunal.character.witness_android":       true,
	"tribunal.character.witness_corrupted_ai":  true,
}

func normalizeTribunalCronAvatarAsset(assetID string, actorType string) string {
	id := strings.TrimSpace(assetID)
	if tribunalCronAvatarAssetIDs[id] {
		return id
	}
	t := strings.ToLower(strings.TrimSpace(actorType))
	switch {
	case strings.Contains(t, "judge"):
		return "tribunal.character.judge_ai"
	case strings.Contains(t, "prosecut"):
		return "tribunal.character.prosecutor_ai"
	case strings.Contains(t, "defense"):
		return "tribunal.character.defense_ai"
	case strings.Contains(t, "assistant"):
		return "tribunal.character.assistant_ai"
	case strings.Contains(t, "clerk") || strings.Contains(t, "greff"):
		return "tribunal.character.clerk_ai"
	case strings.Contains(t, "fact"):
		return "tribunal.character.fact_checker_ai"
	case strings.Contains(t, "expert"):
		return "tribunal.character.expert_witness"
	case strings.Contains(t, "jury") && strings.Contains(t, "emotion"):
		return "tribunal.character.jury_emotional"
	case strings.Contains(t, "jury") && strings.Contains(t, "expert"):
		return "tribunal.character.jury_expert"
	case strings.Contains(t, "jury"):
		return "tribunal.character.jury_logic"
	case strings.Contains(t, "hacker"):
		return "tribunal.character.witness_hacker"
	case strings.Contains(t, "guild"):
		return "tribunal.character.witness_guild_master"
	case strings.Contains(t, "faction"):
		return "tribunal.character.witness_faction_envoy"
	case strings.Contains(t, "android"):
		return "tribunal.character.witness_android"
	case strings.Contains(t, "corrupt") || strings.Contains(t, "corrompu"):
		return "tribunal.character.witness_corrupted_ai"
	case strings.Contains(t, "agent"):
		return "tribunal.character.witness_agent"
	case strings.Contains(t, "civil"):
		return "tribunal.character.witness_civilian"
	default:
		return "tribunal.character.witness_default"
	}
}

func normalizeTribunalCronCastAssets(cast []map[string]any) []map[string]any {
	for i := range cast {
		cast[i]["avatarAssetId"] = normalizeTribunalCronAvatarAsset(
			fmt.Sprint(cast[i]["avatarAssetId"]),
			fmt.Sprint(cast[i]["actorType"]),
		)
	}
	return cast
}

// =============================================================================
// TRIBUNAL GENERATED CASES CRON JOB (reuses existing hourly + provider machinery)
// =============================================================================

const tribunalCasesPerCron = 10

type generatedTribunalCase struct {
	Title                    string           `json:"title"`
	Summary                  string           `json:"summary"`
	CaseType                 string           `json:"caseType"`
	Level                    int              `json:"level"`
	Difficulty               string           `json:"difficulty"`
	EstimatedDurationMinutes int              `json:"estimatedDurationMinutes"`
	Mode                     string           `json:"mode"`
	Tone                     string           `json:"tone"`
	PlayerRoleSuggestion     string           `json:"playerRoleSuggestion"`
	AccusationPosition       string           `json:"accusationPosition"`
	DefensePosition          string           `json:"defensePosition"`
	Tags                     []string         `json:"tags"`
	Witnesses                []map[string]any `json:"witnesses"`
	Evidence                 []map[string]any `json:"evidence"`
	TestimonyStatements      []map[string]any `json:"testimonyStatements"`
	ExpectedContradictions   []map[string]any `json:"expectedContradictions"`

	// Narrative Phoenix-like (correctif)
	RealTruth        string           `json:"realTruth"`
	PublicTruth      string           `json:"publicTruth"`
	FinalReveal      string           `json:"finalReveal"`
	Cast             []map[string]any `json:"cast"`
	Acts             []map[string]any `json:"acts"`
	Scenes           []map[string]any `json:"scenes"`
	ProgressionRules []map[string]any `json:"progressionRules"`
	FailureRules     []map[string]any `json:"failureRules"`
	CrisisMoment     map[string]any   `json:"crisisMoment"`
	PossibleVerdicts []string         `json:"possibleVerdicts"`
	Epilogue         string           `json:"epilogue"`
	NexusBridgeHints []map[string]any `json:"nexusBridgeHints"`
}

func runTribunalCaseJob(ctx context.Context, db *gorm.DB, runAt time.Time, cfg aiProviderConfig, trace cronTrace) error {
	trace.log("generate", "started", "limit=%d", tribunalCasesPerCron)

	batch := tribunalmodels.TribunalCaseGenerationBatch{
		StartedAt:      runAt,
		Source:         "cron_ai",
		TriggerType:    "scheduled",
		Status:         "running",
		RequestedCount: tribunalCasesPerCron,
		ProviderType:   cfg.Name,
		ProviderModel:  cfg.Model,
		CronSchedule:   "08:00,12:00,20:00",
	}
	if err := db.WithContext(ctx).Create(&batch).Error; err != nil {
		trace.log("batch_create", "failed", "err=%v", err)
		return fmt.Errorf("create batch: %w", err)
	}
	trace.log("batch_create", "completed", "batch_id=%d", batch.ID)

	cases, err := generateTribunalCases(ctx, cfg, trace)
	if err != nil {
		batch.Status = "failed"
		batch.ErrorMessage = err.Error()
		_ = db.WithContext(ctx).Save(&batch).Error
		return fmt.Errorf("generate tribunal cases: %w", err)
	}
	trace.log("generate", "completed", "items=%d", len(cases))

	generated := 0
	published := 0
	failed := 0
	for i, c := range cases {
		hasCast := len(c.Cast) > 0 || len(c.Witnesses) > 0
		if c.Title == "" || !hasCast {
			failed++
			trace.log("insert", "skipped", "index=%d reason=missing_title_or_cast level=%d", i, c.Level)
			continue
		}
		c.Cast = normalizeTribunalCronCastAssets(c.Cast)
		tagsJSON, _ := json.Marshal(c.Tags)
		witJSON, _ := json.Marshal(c.Witnesses)
		evJSON, _ := json.Marshal(c.Evidence)
		testJSON, _ := json.Marshal(c.TestimonyStatements)
		contrJSON, _ := json.Marshal(c.ExpectedContradictions)
		meta := map[string]any{"generatedBy": "tribunal-cron", "provider": cfg.Name, "model": cfg.Model, "runAt": runAt.Format(time.RFC3339)}
		metaJSON, _ := json.Marshal(meta)

		rec := tribunalmodels.TribunalGeneratedCase{
			GenerationBatchID:          batch.ID,
			Title:                      c.Title,
			Summary:                    c.Summary,
			CaseType:                   defaultText(c.CaseType, "custom"),
			Level:                      clampInt(c.Level, 1, 10),
			Difficulty:                 defaultText(c.Difficulty, "standard"),
			EstimatedDurationMinutes:   c.EstimatedDurationMinutes,
			Mode:                       defaultText(c.Mode, "full_case"),
			Tone:                       defaultText(c.Tone, "cyberpunk_serious"),
			PlayerRoleSuggestion:       defaultText(c.PlayerRoleSuggestion, "neutral"),
			AccusationPosition:         c.AccusationPosition,
			DefensePosition:            c.DefensePosition,
			TagsJSON:                   datatypes.JSON(tagsJSON),
			WitnessesJSON:              datatypes.JSON(witJSON),
			EvidenceJSON:               datatypes.JSON(evJSON),
			TestimonyJSON:              datatypes.JSON(testJSON),
			ExpectedContradictionsJSON: datatypes.JSON(contrJSON),
			Status:                     "ready",
			IsPlayable:                 true,
			IsPublished:                true,
			GeneratedByCron:            true,
			ProviderType:               cfg.Name,
			ProviderModel:              cfg.Model,
			MetadataJSON:               datatypes.JSON(metaJSON),
			// Narrative Phoenix-like fields (from corrective prompt)
			RealTruth:             c.RealTruth,
			PublicTruth:           c.PublicTruth,
			FinalReveal:           c.FinalReveal,
			IsNarrativePlayable:   len(c.Scenes) > 0 || len(c.ProgressionRules) > 0,
			HasCrisisMoment:       c.CrisisMoment != nil && len(c.CrisisMoment) > 0,
			HasFinalReveal:        c.FinalReveal != "",
			HasIntro:              true, // prompt always requires intro scene
			HasBriefing:           true, // prompt requires briefing
			ActsCount:             len(c.Acts),
			ScenesCount:           len(c.Scenes),
			WitnessesCount:        len(c.Witnesses) + len(c.Cast), // approximate from cast + old witnesses
			EvidenceCount:         len(c.Evidence),
			ProgressionRulesCount: len(c.ProgressionRules),
			HasPossibleVerdicts:   len(c.PossibleVerdicts) > 0,
			HasNexusBridge:        len(c.NexusBridgeHints) > 0,
		}
		if len(c.Cast) > 0 {
			if b, e := json.Marshal(c.Cast); e == nil {
				rec.CharacterCastJSON = datatypes.JSON(b)
			}
		}
		if len(c.Acts) > 0 {
			if b, e := json.Marshal(c.Acts); e == nil {
				rec.ActsJSON = datatypes.JSON(b)
			}
		}
		if len(c.Scenes) > 0 {
			if b, e := json.Marshal(c.Scenes); e == nil {
				rec.ScenesJSON = datatypes.JSON(b)
			}
		}
		if len(c.ProgressionRules) > 0 {
			if b, e := json.Marshal(c.ProgressionRules); e == nil {
				rec.ProgressionRulesJSON = datatypes.JSON(b)
			}
		}
		if len(c.FailureRules) > 0 {
			if b, e := json.Marshal(c.FailureRules); e == nil {
				rec.FailureRulesJSON = datatypes.JSON(b)
			}
		}
		if len(c.NexusBridgeHints) > 0 {
			if b, e := json.Marshal(c.NexusBridgeHints); e == nil {
				rec.NexusBridgeHintsJSON = datatypes.JSON(b)
			}
		}
		if err := db.WithContext(ctx).Create(&rec).Error; err != nil {
			failed++
			trace.log("insert", "failed", "index=%d level=%d err=%v", i, c.Level, err)
			continue
		}
		generated++
		published++
		trace.log("insert", "created", "id=%d level=%d title=%q", rec.ID, rec.Level, rec.Title)
	}

	batch.GeneratedCount = generated
	batch.PublishedCount = published
	batch.FailedCount = failed
	if generated == 0 {
		batch.Status = "failed"
		batch.ErrorMessage = "no cases created"
	} else if failed > 0 {
		batch.Status = "partial_success"
	} else {
		batch.Status = "success"
	}
	now := time.Now()
	batch.FinishedAt = &now
	batch.DurationMs = time.Since(runAt).Milliseconds()
	if err := db.WithContext(ctx).Save(&batch).Error; err != nil {
		trace.log("batch_finalize", "failed", "err=%v", err)
	}
	trace.log("insert", "summary", "received=%d created=%d published=%d failed=%d batch=%d", len(cases), generated, published, failed, batch.ID)

	if generated == 0 {
		return fmt.Errorf("no tribunal case created")
	}
	return nil
}

func generateTribunalCases(ctx context.Context, cfg aiProviderConfig, trace cronTrace) ([]generatedTribunalCase, error) {
	// Generate 1 by 1 (user requirement) — much more reliable than one huge prompt for 10 complex narrative cases.
	out := make([]generatedTribunalCase, 0, tribunalCasesPerCron)
	seen := map[int]bool{}

	for lvl := 1; lvl <= tribunalCasesPerCron; lvl++ {
		if seen[lvl] {
			continue
		}

		sys, user := tribunalprompts.BuildSingleNarrativeCasePrompt(lvl)
		trace.log("prompt_build", "completed", "level=%d chars=%d", lvl, len(user))

		// callProvider puts the prompt as system message
		fullPrompt := sys + "\n\n" + user
		response, usedProvider, err := callProviderWithFallbacks(ctx, cfg, fullPrompt, trace)
		if err != nil {
			trace.log("single_case", "failed", "level=%d err=%v", lvl, err)
			continue // don't fail the whole batch
		}
		if usedProvider.Name != cfg.Name || usedProvider.Model != cfg.Model {
			trace.log("single_case", "provider_fallback_used", "level=%d provider=%s model=%s", lvl, usedProvider.Name, usedProvider.Model)
		}
		cleaned := cleanJSON(response)
		trace.log("json_clean", "completed", "level=%d", lvl)

		var single generatedTribunalCase
		if uerr := json.Unmarshal([]byte(cleaned), &single); uerr != nil {
			trace.log("json_parse", "failed", "level=%d err=%v preview=%s", lvl, uerr, preview(cleaned, 400))
			continue
		}

		// force correct level
		single.Level = lvl

		lv := clampInt(single.Level, 1, 10)
		if seen[lv] {
			continue
		}
		seen[lv] = true

		if single.EstimatedDurationMinutes <= 0 {
			single.EstimatedDurationMinutes = 5 + lv*5
		}
		out = append(out, single)
		trace.log("single_case", "success", "level=%d title=%s", lv, single.Title)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("provider returned 0 cases (1-by-1 mode)")
	}
	return out, nil
}

func promptsForTribunalCases() (string, string) {
	// Kept for compatibility. Real generation now uses BuildSingleNarrativeCasePrompt in a loop (1 by 1).
	return tribunalprompts.BuildSingleNarrativeCasePrompt(5)
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func defaultText(v, fb string) string {
	if strings.TrimSpace(v) == "" {
		return fb
	}
	return strings.TrimSpace(v)
}
