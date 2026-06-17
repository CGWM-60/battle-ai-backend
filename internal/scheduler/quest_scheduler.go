package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

type providerCallError struct {
	Provider string
	Model    string
	URL      string
	Err      error
}

func (e providerCallError) Error() string {
	return fmt.Sprintf("provider=%s model=%s url=%s err=%v", e.Provider, e.Model, e.URL, e.Err)
}

func (e providerCallError) Unwrap() error {
	return e.Err
}

func (e providerCallError) TimedOut() bool {
	return errors.Is(e.Err, context.DeadlineExceeded) || strings.Contains(strings.ToLower(e.Err.Error()), "context deadline exceeded")
}

type providerAttemptFailure struct {
	Provider string
	Model    string
	URL      string
	Message  string
	TimedOut bool
}

type providerAttemptsError struct {
	Primary  aiProviderConfig
	Failures []providerAttemptFailure
	LastErr  error
}

func (e providerAttemptsError) Error() string {
	parts := make([]string, 0, len(e.Failures))
	for _, failure := range e.Failures {
		timeout := ""
		if failure.TimedOut {
			timeout = " timeout=true"
		}
		parts = append(parts, fmt.Sprintf("%s/%s url=%s%s err=%s", failure.Provider, failure.Model, failure.URL, timeout, failure.Message))
	}
	return fmt.Sprintf(
		"all provider attempts failed primary=%s/%s attempts=[%s]",
		e.Primary.Name,
		e.Primary.Model,
		strings.Join(parts, " | "),
	)
}

func (e providerAttemptsError) Unwrap() error {
	return e.LastErr
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

type generatedRolePlayScene struct {
	SceneKey            string   `json:"sceneKey"`
	ChapterIndex        int      `json:"chapterIndex"`
	Title               string   `json:"title"`
	Summary             string   `json:"summary"`
	SceneType           string   `json:"sceneType"`
	RoomType            string   `json:"roomType"`
	Atmosphere          string   `json:"atmosphere"`
	DangerLevel         string   `json:"dangerLevel"`
	ImagePrompt         string   `json:"imagePrompt"`
	ImageNegativePrompt string   `json:"imageNegativePrompt"`
	VisualTags          []string `json:"visualTags"`
}

type generatedRolePlayQuest struct {
	Title               string                   `json:"title"`
	Summary             string                   `json:"summary"`
	Prompt              string                   `json:"prompt"`
	Theme               string                   `json:"theme"`
	Level               string                   `json:"level"`
	Xp                  int                      `json:"xp"`
	Coin                int                      `json:"coin"`
	ImagePrompt         string                   `json:"imagePrompt"`
	ImageNegativePrompt string                   `json:"imageNegativePrompt"`
	VisualStyle         string                   `json:"visualStyle"`
	VisualTags          []string                 `json:"visualTags"`
	RpgMetadata         map[string]any           `json:"rpgMetadata"`
	Meta                map[string]any           `json:"metadata"`
	Arcs                []generatedRolePlayArc   `json:"arcs"`
	Scenes              []generatedRolePlayScene `json:"scenes"`
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
Pour chaque quete, ajoute aussi des visuels exploitables par un generateur d'image:
- imagePrompt et imageNegativePrompt globaux (anglais, detailles, sans texte visible)
- visualStyle (ex: dark fantasy mobile RPG)
- visualTags (3 a 6 mots)
- rpgMetadata (recommendedPartySize, supportsCoop, difficultyClass, mainThreat, mainLocation, rpgTags, suggestedSkills)
- scenes: au minimum 3 scenes (entree, conflit, climax) avec sceneKey, chapterIndex, title, summary, sceneType, roomType, atmosphere, dangerLevel, imagePrompt, imageNegativePrompt, visualTags
Format:
[{"title":"...","summary":"resume court","prompt":"prompt global de la quete","theme":"fantasy|sf|horreur|steampunk|modern","level":"facile|moyen|difficile","xp":80,"coin":30,"imagePrompt":"...","imageNegativePrompt":"...","visualStyle":"dark fantasy mobile RPG","visualTags":["crypt","fog"],"rpgMetadata":{"recommendedPartySize":1,"supportsCoop":true,"difficultyClass":12},"metadata":{"ton":"..."},"scenes":[{"sceneKey":"scene_01_entry","chapterIndex":1,"title":"Entree","summary":"...","sceneType":"exploration","roomType":"entrance","atmosphere":"mysterious","dangerLevel":"low","imagePrompt":"...","imageNegativePrompt":"...","visualTags":["statues"]}],"arcs":[{"title":"Arc 1","summary":"...","objective":"...","prompt":"brief de l'arc","metadata":{"tone":"..."},"chapters":[{"title":"Chapitre 1","summary":"...","objective":"objectif jouable","introPrompt":"situation initiale du chapitre","successPrompt":"consequence en cas de reussite","failurePrompt":"consequence en cas d'echec","isBoss":false,"xp":20,"coin":8,"metadata":{"stakes":"..."}}]}]}]`, count)

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
		Slug:                slug,
		Title:               item.Title,
		Summary:             item.Summary,
		Prompt:              item.Prompt,
		Theme:               item.Theme,
		Level:               item.Level,
		Xp:                  item.Xp,
		Coin:                item.Coin,
		Source:              source,
		Status:              constants.QuestStatusDraft,
		Metadata:            ensureMetadata(item.Meta, cfg, runAt),
		Arcs:                generatedRolePlayArcInputs(item.Arcs, item.Level),
		ImagePrompt:         item.ImagePrompt,
		ImageNegativePrompt: item.ImageNegativePrompt,
		VisualStyle:         item.VisualStyle,
		VisualTags:          item.VisualTags,
		RpgMetadata:         item.RpgMetadata,
		Scenes:              generatedRolePlaySceneInputs(item.Scenes),
	}
}

func generatedRolePlaySceneInputs(items []generatedRolePlayScene) []service.RolePlaySceneInput {
	if len(items) == 0 {
		return nil
	}
	out := make([]service.RolePlaySceneInput, 0, len(items))
	for index, item := range items {
		chapterIndex := item.ChapterIndex
		if chapterIndex <= 0 {
			chapterIndex = index + 1
		}
		sceneKey := strings.TrimSpace(item.SceneKey)
		if sceneKey == "" {
			sceneKey = fmt.Sprintf("scene_%02d", index+1)
		}
		out = append(out, service.RolePlaySceneInput{
			SceneKey:       sceneKey,
			ChapterIndex:   chapterIndex,
			ArcIndex:       1,
			Title:          item.Title,
			Summary:        item.Summary,
			SceneType:      item.SceneType,
			RoomType:       item.RoomType,
			Atmosphere:     item.Atmosphere,
			DangerLevel:    item.DangerLevel,
			ImagePrompt:    item.ImagePrompt,
			NegativePrompt: item.ImageNegativePrompt,
			VisualTags:     item.VisualTags,
		})
	}
	return out
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
		return "", providerCallError{
			Provider: cfg.Name,
			Model:    cfg.Model,
			URL:      url,
			Err:      err,
		}
	}
	trace.log("provider_call", "completed", "duration_ms=%d response_chars=%d", time.Since(startedAt).Milliseconds(), len(response))
	return response, nil
}

func callProviderWithFallbacks(ctx context.Context, primary aiProviderConfig, prompt string, trace cronTrace) (string, aiProviderConfig, error) {
	return callProviderWithFallbacksExcluding(ctx, primary, prompt, trace, nil)
}

func callProviderWithFallbacksExcluding(ctx context.Context, primary aiProviderConfig, prompt string, trace cronTrace, excludedProviders map[string]bool) (string, aiProviderConfig, error) {
	attempts := providerAttemptsWithExclusions(primary, excludedProviders)
	var lastErr error
	failures := make([]providerAttemptFailure, 0, len(attempts))
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
		failure := providerAttemptFailure{
			Provider: cfg.Name,
			Model:    cfg.Model,
			Message:  err.Error(),
		}
		var callErr providerCallError
		if errors.As(err, &callErr) {
			failure.URL = callErr.URL
			failure.TimedOut = callErr.TimedOut()
		}
		failures = append(failures, failure)
		if ctx.Err() != nil {
			break
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no configured provider available")
	}
	return "", aiProviderConfig{}, providerAttemptsError{Primary: primary, Failures: failures, LastErr: lastErr}
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
	return providerAttemptsWithExclusions(primary, nil)
}

func providerAttemptsWithExclusions(primary aiProviderConfig, excludedProviders map[string]bool) []aiProviderConfig {
	attempts := []aiProviderConfig{primary}
	primaryName := strings.ToLower(strings.TrimSpace(primary.Name))
	attempts = nil
	seen := map[string]bool{primaryName: true}
	if primary.Name != "" && !excludedProviders[primaryName] {
		attempts = append(attempts, primary)
	}
	for _, name := range cronProviderRotation() {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" || seen[name] || excludedProviders[name] {
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
	Synopsis                 string           `json:"synopsis"`
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
			Summary:                    firstNonEmpty(c.Summary, c.Synopsis),
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
	disabledProviders := map[string]bool{}

	for lvl := 1; lvl <= tribunalCasesPerCron; lvl++ {
		if seen[lvl] {
			continue
		}

		sys, user := tribunalprompts.BuildSingleNarrativeCasePrompt(lvl)
		trace.log("prompt_build", "completed", "level=%d chars=%d", lvl, len(user))

		// callProvider puts the prompt as system message
		fullPrompt := sys + "\n\n" + user
		trace.log("single_case_request", "started", "request_index=%d total=%d level=%d disabled_providers=%s", lvl, tribunalCasesPerCron, lvl, strings.Join(disabledProviderNames(disabledProviders), ","))
		response, usedProvider, err := callProviderWithFallbacksExcluding(ctx, cfg, fullPrompt, trace, disabledProviders)
		if err != nil {
			trace.log("single_case", "failed", "level=%d err=%v", lvl, err)
			disableTimedOutProviders(err, disabledProviders, trace)
			continue // don't fail the whole batch
		}
		if usedProvider.Name != cfg.Name || usedProvider.Model != cfg.Model {
			trace.log("single_case", "provider_fallback_used", "level=%d provider=%s model=%s", lvl, usedProvider.Name, usedProvider.Model)
		}
		cleaned := cleanJSON(response)
		trace.log("json_clean", "completed", "level=%d", lvl)

		single, uerr := parseGeneratedTribunalCasePayload(cleaned, lvl)
		if uerr != nil {
			trace.log("json_parse", "failed", "level=%d err=%v preview=%s", lvl, uerr, preview(cleaned, 400))
			single = fallbackGeneratedTribunalCase(lvl, fmt.Sprintf("parse:%s:%d:%d", cfg.Name, lvl, time.Now().UnixNano()))
			trace.log("single_case", "fallback", "level=%d reason=json_parse title=%s", lvl, single.Title)
		}

		// force correct level
		single.Level = lvl
		single = completeGeneratedTribunalCase(single, lvl, fmt.Sprintf("%s:%d:%d", cfg.Name, lvl, time.Now().UnixNano()))

		lv := clampInt(single.Level, 1, 10)
		if seen[lv] {
			continue
		}
		seen[lv] = true

		if single.EstimatedDurationMinutes <= 0 {
			single.EstimatedDurationMinutes = 5 + lv*5
		}
		out = append(out, single)
		trace.log("single_case_request", "completed", "request_index=%d total=%d level=%d provider=%s model=%s", lvl, tribunalCasesPerCron, lv, usedProvider.Name, usedProvider.Model)
		trace.log("single_case", "success", "level=%d title=%s", lv, single.Title)
	}

	if len(out) == 0 {
		for lvl := 1; lvl <= tribunalCasesPerCron; lvl++ {
			out = append(out, fallbackGeneratedTribunalCase(lvl, fmt.Sprintf("zero:%s:%d:%d", cfg.Name, lvl, time.Now().UnixNano())))
		}
		trace.log("generate", "fallback", "provider returned 0 valid cases; generated deterministic fallback cases=%d", len(out))
	}
	return out, nil
}

func parseGeneratedTribunalCasePayload(cleaned string, level int) (generatedTribunalCase, error) {
	var direct generatedTribunalCase
	if err := json.Unmarshal([]byte(cleaned), &direct); err == nil && strings.TrimSpace(direct.Title) != "" {
		return direct, nil
	}

	var raw any
	if err := json.Unmarshal([]byte(cleaned), &raw); err != nil {
		return generatedTribunalCase{}, err
	}
	candidates := generatedTribunalCaseCandidates(raw)
	if len(candidates) == 0 {
		return generatedTribunalCase{}, fmt.Errorf("no tribunal case object found")
	}
	index := clampInt(level-1, 0, len(candidates)-1)
	return candidates[index], nil
}

func generatedTribunalCaseCandidates(raw any) []generatedTribunalCase {
	out := []generatedTribunalCase{}
	switch v := raw.(type) {
	case []any:
		for _, item := range v {
			out = append(out, generatedTribunalCaseCandidates(item)...)
		}
	case map[string]any:
		for _, key := range []string{"cases", "case", "data", "result", "generatedCase"} {
			if nested, ok := v[key]; ok {
				out = append(out, generatedTribunalCaseCandidates(nested)...)
			}
		}
		if _, hasTitle := v["title"]; hasTitle {
			if b, err := json.Marshal(v); err == nil {
				var c generatedTribunalCase
				if err := json.Unmarshal(b, &c); err == nil && strings.TrimSpace(c.Title) != "" {
					out = append(out, c)
				}
			}
		}
	}
	return out
}

func completeGeneratedTribunalCase(c generatedTribunalCase, level int, seed string) generatedTribunalCase {
	fallback := fallbackGeneratedTribunalCase(level, seed)
	if isGenericTribunalTitle(c.Title) {
		c.Title = fallback.Title
	}
	if strings.TrimSpace(c.Synopsis) == "" && strings.TrimSpace(c.Summary) == "" {
		c.Synopsis = fallback.Synopsis
	}
	if strings.TrimSpace(c.CaseType) == "" {
		c.CaseType = fallback.CaseType
	}
	if strings.TrimSpace(c.Difficulty) == "" {
		c.Difficulty = fallback.Difficulty
	}
	if c.EstimatedDurationMinutes <= 0 {
		c.EstimatedDurationMinutes = fallback.EstimatedDurationMinutes
	}
	if strings.TrimSpace(c.Mode) == "" {
		c.Mode = fallback.Mode
	}
	if strings.TrimSpace(c.Tone) == "" {
		c.Tone = fallback.Tone
	}
	if strings.TrimSpace(c.PlayerRoleSuggestion) == "" {
		c.PlayerRoleSuggestion = fallback.PlayerRoleSuggestion
	}
	if strings.TrimSpace(c.AccusationPosition) == "" {
		c.AccusationPosition = fallback.AccusationPosition
	}
	if strings.TrimSpace(c.DefensePosition) == "" {
		c.DefensePosition = fallback.DefensePosition
	}
	if len(c.Tags) == 0 {
		c.Tags = fallback.Tags
	}
	if len(c.Cast) == 0 {
		c.Cast = fallback.Cast
	}
	if len(c.Witnesses) == 0 {
		c.Witnesses = fallback.Witnesses
	}
	if len(c.Evidence) == 0 {
		c.Evidence = fallback.Evidence
	}
	if len(c.TestimonyStatements) == 0 {
		c.TestimonyStatements = fallback.TestimonyStatements
	}
	if len(c.ExpectedContradictions) == 0 {
		c.ExpectedContradictions = fallback.ExpectedContradictions
	}
	if strings.TrimSpace(c.RealTruth) == "" {
		c.RealTruth = fallback.RealTruth
	}
	if strings.TrimSpace(c.PublicTruth) == "" {
		c.PublicTruth = fallback.PublicTruth
	}
	if strings.TrimSpace(c.FinalReveal) == "" {
		c.FinalReveal = fallback.FinalReveal
	}
	if len(c.Acts) == 0 {
		c.Acts = fallback.Acts
	}
	if len(c.Scenes) == 0 {
		c.Scenes = fallback.Scenes
	}
	if len(c.ProgressionRules) == 0 {
		c.ProgressionRules = fallback.ProgressionRules
	}
	if len(c.FailureRules) == 0 {
		c.FailureRules = fallback.FailureRules
	}
	if len(c.PossibleVerdicts) == 0 {
		c.PossibleVerdicts = fallback.PossibleVerdicts
	}
	if strings.TrimSpace(c.Epilogue) == "" {
		c.Epilogue = fallback.Epilogue
	}
	if len(c.NexusBridgeHints) == 0 {
		c.NexusBridgeHints = fallback.NexusBridgeHints
	}
	c.Level = level
	return c
}

func isGenericTribunalTitle(title string) bool {
	t := strings.ToLower(strings.TrimSpace(title))
	if t == "" || t == "<nil>" {
		return true
	}
	generic := []string{"affaire nexus", "tribunal nexus", "le jugement", "le proces de l'ia", "le procès de l'ia", "l'affaire du protocole", "affaire du protocole"}
	for _, value := range generic {
		if t == value {
			return true
		}
	}
	return len([]rune(t)) < 8
}

func fallbackGeneratedTribunalCase(level int, seed string) generatedTribunalCase {
	level = clampInt(level, 1, 10)
	themes := []struct {
		title    string
		caseType string
		place    string
		object   string
		secret   string
	}{
		{"Les minutes manquantes du relais Kappa", "world_event", "relais Kappa", "un journal de maintenance", "le delai de vingt-deux minutes cache une intervention humaine"},
		{"Le contrat fantome de la serre Helix", "guild_conflict", "serre Helix", "un contrat de rationnement", "la clause signee n'est valide qu'apres activation d'un badge vole"},
		{"L'alibi froid de Sira Voss", "player_conflict", "clinique Nocturne", "un scan biométrique", "le scan prouve une presence mais pas l'identite du corps"},
		{"La dette rouge du marche orbital", "commercial", "marche orbital", "un reçu de transfert", "la dette a ete reglee par un tiers qui voulait provoquer le proces"},
		{"Le silence du temoin synthétique", "moral", "tour de mediation", "un enregistrement audio", "le temoin a ete bride pour proteger une victime"},
		{"La signature impossible du bastion Aster", "faction_conflict", "bastion Aster", "une signature cryptographique", "la signature est authentique mais rejouee depuis une sauvegarde"},
		{"La pluie noire sur le quai 17", "absurd", "quai 17", "un capteur meteo", "la pluie sert de couverture a une livraison interdite"},
		{"Le verdict avant l'audience", "tribunal_integrity", "greffe central", "un brouillon de verdict", "le document est une simulation, pas une decision finale"},
		{"Le témoin qui n'avait pas d'ombre", "identity", "district holographique", "une video de securite", "l'ombre absente vient d'un projecteur coupe au bon moment"},
		{"La clé morte du coffre Nexus", "data_theft", "coffre Nexus", "une cle d'acces expiree", "la cle morte ouvre seulement le journal des tentatives"},
	}
	idx := (level - 1) % len(themes)
	theme := themes[idx]
	shortSeed := strconv.FormatInt(int64(len(seed)+level*37), 36)
	title := fmt.Sprintf("%s [%s]", theme.title, shortSeed)
	witnessID := fmt.Sprintf("temoin_%d", level)
	evidenceID := fmt.Sprintf("preuve_%d", level)
	statementID := fmt.Sprintf("stmt_%d", level)
	nextSceneID := fmt.Sprintf("act1_confrontation_%d", level)
	return generatedTribunalCase{
		Title:                    title,
		Synopsis:                 fmt.Sprintf("Dans %s, une preuve trop parfaite accuse la mauvaise personne. Le tribunal doit separer trace technique, mobile et intention.", theme.place),
		CaseType:                 theme.caseType,
		Level:                    level,
		Difficulty:               []string{"initiation", "easy", "standard", "intermediate", "confirmed", "hard", "expert", "master", "legendary", "nexus"}[level-1],
		EstimatedDurationMinutes: 10 + level*4,
		Mode:                     "full_case",
		Tone:                     "cyberpunk_serious",
		PlayerRoleSuggestion:     "defense",
		AccusationPosition:       fmt.Sprintf("L'accusation affirme que %s etablit une responsabilite directe et volontaire.", theme.object),
		DefensePosition:          "La defense doit montrer que la preuve principale est vraie mais interpretee trop vite.",
		Tags:                     []string{"fallback", "cyberpunk", theme.caseType, fmt.Sprintf("niveau_%d", level)},
		Witnesses: []map[string]any{{
			"name":        fmt.Sprintf("Temoin %d du %s", level, theme.place),
			"role":        "temoin cle",
			"credibility": 62 + level,
			"bias":        "protege sa reputation",
			"personality": "precis, nerveux, evasif quand on parle des horaires",
		}},
		Evidence: []map[string]any{{
			"evidenceId":        evidenceID,
			"title":             strings.Title(theme.object),
			"description":       fmt.Sprintf("Piece centrale recuperee dans %s.", theme.place),
			"details":           fmt.Sprintf("La piece semble accuser directement l'accuse, mais %s.", theme.secret),
			"origin":            "registre Tribunal de secours",
			"contradictionHint": fmt.Sprintf("A confronter avec la declaration %s sur l'heure et le lieu.", statementID),
			"chainOfCustody":    "saisie automatique, scelle numerique, verification partielle",
			"evidenceType":      "document",
			"strength":          70,
			"reliability":       76,
			"supportsSide":      "neutral",
			"assetId":           "tribunal.evidence.document",
		}},
		TestimonyStatements: []map[string]any{{
			"statementId":    statementID,
			"speakerActorId": witnessID,
			"text":           fmt.Sprintf("Je n'ai vu personne manipuler %s apres la fermeture du %s.", theme.object, theme.place),
		}},
		ExpectedContradictions: []map[string]any{{
			"statementContent":  fmt.Sprintf("Je n'ai vu personne manipuler %s apres la fermeture du %s.", theme.object, theme.place),
			"evidenceTitle":     strings.Title(theme.object),
			"contradictionType": "time",
		}},
		RealTruth:   theme.secret,
		PublicTruth: "La ville croit que l'accuse a simplement ete pris par une preuve technique incontestable.",
		FinalReveal: fmt.Sprintf("La contradiction finale montre que %s.", theme.secret),
		Cast: []map[string]any{
			{"actorId": "judge_ai", "actorType": "judge", "name": "Magistrat Ada-7", "personality": "calme, strict, sensible aux contradictions temporelles", "avatarAssetId": "tribunal.character.judge_ai"},
			{"actorId": "prosecutor_ai", "actorType": "prosecutor", "name": "Procureur Nyx", "personality": "incisif, convaincu par les logs", "avatarAssetId": "tribunal.character.prosecutor_ai"},
			{"actorId": witnessID, "actorType": "witness", "name": fmt.Sprintf("Ilan Trace-%d", level), "personality": "temoin technique sous pression", "avatarAssetId": "tribunal.character.witness_agent"},
		},
		Acts: []map[string]any{{"actIndex": 1, "title": "Audience sous tension", "objective": "Identifier la faille de la preuve principale.", "summary": "Le tribunal ouvre sur une preuve forte mais incomplete."}},
		Scenes: []map[string]any{
			{"sceneId": fmt.Sprintf("act1_intro_%d", level), "actIndex": 1, "sceneIndex": 0, "sceneType": "intro", "title": "Ouverture du dossier", "objective": "Comprendre l'accusation.", "narrativeText": fmt.Sprintf("Le tribunal examine %s. Le procureur presente %s comme une preuve decisive.", theme.place, theme.object), "activeWitnessId": nil, "activeActorIds": []string{"judge_ai", "prosecutor_ai"}, "availableEvidenceIds": []string{evidenceID}, "visibleStatementIds": []string{}, "visibleStatements": []map[string]any{}, "allowedActions": []string{"continue_story", "ai_analysis"}, "nextSceneId": nextSceneID},
			{"sceneId": nextSceneID, "actIndex": 1, "sceneIndex": 1, "sceneType": "cross_examination", "title": "Contre-interrogatoire", "objective": "Trouver la contradiction horaire.", "narrativeText": "Le temoin confirme sa version. Sa certitude repose sur une horloge secondaire.", "activeWitnessId": witnessID, "activeActorIds": []string{witnessID}, "availableEvidenceIds": []string{evidenceID}, "visibleStatementIds": []string{statementID}, "visibleStatements": []map[string]any{{"statementId": statementID, "speakerActorId": witnessID, "text": fmt.Sprintf("Je n'ai vu personne manipuler %s apres la fermeture du %s.", theme.object, theme.place)}}, "allowedActions": []string{"press", "present_evidence", "objection", "ai_analysis", "expose_lie"}, "nextSceneId": fmt.Sprintf("act1_reveal_%d", level)},
			{"sceneId": fmt.Sprintf("act1_reveal_%d", level), "actIndex": 1, "sceneIndex": 2, "sceneType": "reveal", "title": "La preuve se retourne", "objective": "Formuler la verite cachee.", "narrativeText": fmt.Sprintf("La salle comprend que %s. Le verdict ne peut plus suivre la lecture publique.", theme.secret), "activeWitnessId": nil, "activeActorIds": []string{"judge_ai"}, "availableEvidenceIds": []string{evidenceID}, "visibleStatementIds": []string{}, "visibleStatements": []map[string]any{}, "allowedActions": []string{"continue_story"}, "nextSceneId": nil},
		},
		ProgressionRules: []map[string]any{{
			"sceneId":             nextSceneID,
			"triggerAction":       "present_evidence",
			"requiredEvidenceId":  evidenceID,
			"requiredStatementId": statementID,
			"resultType":          "major_reveal",
			"isCritical":          true,
			"unlockSceneId":       fmt.Sprintf("act1_reveal_%d", level),
			"narrativeResult":     fmt.Sprintf("La preuve confirme la trace, mais revele surtout que %s.", theme.secret),
			"scoreEffects":        map[string]any{"defenseScoreDelta": 10, "witnessCredibilityDelta": -8, "tribunalPressureDelta": 6},
		}},
		FailureRules:     []map[string]any{{"sceneId": nextSceneID, "triggerAction": "objection", "penaltyType": "score_down", "maxFailuresBeforeHint": 2, "judgeWarningText": "Le tribunal exige un lien direct entre preuve et declaration.", "hintText": "La contradiction se trouve dans l'heure et la conservation de la piece.", "scoreEffects": map[string]any{"defenseScoreDelta": -3}, "stayOnScene": true}},
		CrisisMoment:     map[string]any{"sceneId": nextSceneID, "trigger": "wrong_objection", "effect": "la pression du tribunal augmente"},
		PossibleVerdicts: []string{"defense_win", "partial_defense", "hidden_truth"},
		Epilogue:         "Le tribunal ordonne une verification des protocoles de preuve avant toute sanction definitive.",
		NexusBridgeHints: []map[string]any{{"type": "trust_delta", "targetId": theme.caseType, "delta": -2}},
	}
}

func disableTimedOutProviders(err error, disabledProviders map[string]bool, trace cronTrace) {
	var attemptsErr providerAttemptsError
	if !errors.As(err, &attemptsErr) {
		return
	}
	for _, failure := range attemptsErr.Failures {
		if !failure.TimedOut || failure.Provider == "" {
			continue
		}
		providerName := strings.ToLower(strings.TrimSpace(failure.Provider))
		if providerName == "" || disabledProviders[providerName] {
			continue
		}
		disabledProviders[providerName] = true
		trace.log("provider_batch_disable", "timeout", "provider=%s model=%s reason=timeout_for_remaining_single_case_requests", failure.Provider, failure.Model)
	}
}

func disabledProviderNames(disabledProviders map[string]bool) []string {
	names := make([]string, 0, len(disabledProviders))
	for name, disabled := range disabledProviders {
		if disabled {
			names = append(names, name)
		}
	}
	return names
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
