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
	"time"
	_ "time/tzdata"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/provider"
	"cgwm/battle/internal/service"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const cronQuestLimit = 10

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
	Title   string         `json:"title"`
	Summary string         `json:"summary"`
	Prompt  string         `json:"prompt"`
	Theme   string         `json:"theme"`
	Level   string         `json:"level"`
	Xp      int            `json:"xp"`
	Coin    int            `json:"coin"`
	Meta    map[string]any `json:"metadata"`
}

func StartQuestGenerationCron(db *gorm.DB) {
	if strings.EqualFold(env("AI_QUEST_CRON_ENABLED", "true"), "false") {
		log.Printf("[quest-cron] step=boot status=disabled reason=AI_QUEST_CRON_ENABLED=false")
		return
	}

	location := loadLocation()
	log.Printf(
		"[quest-cron] step=boot status=enabled timezone=%s hours=08-21 limit=%d battle_job=true roleplay_job=true",
		location.String(),
		cronQuestLimit,
	)

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

	var lastRun string
	for {
		now := <-ticker.C
		localNow := now.In(location)
		if localNow.Minute() != 0 {
			continue
		}
		if localNow.Hour() < 8 || localNow.Hour() > 21 {
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

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
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
		meta, _ := json.Marshal(ensureMetadata(item.Meta, cfg, runAt))
		slug := uniqueSlug(item.Title, runAt, index)
		quest := models.RolePlayQuestTemplate{
			Slug:     slug,
			Title:    item.Title,
			Summary:  item.Summary,
			Prompt:   item.Prompt,
			Theme:    item.Theme,
			Level:    item.Level,
			Xp:       item.Xp,
			Coin:     item.Coin,
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
		return fmt.Errorf("no roleplay quest created")
	}
	return nil
}

func generateBattleQuests(ctx context.Context, cfg aiProviderConfig, trace cronTrace) ([]generatedBattleQuest, error) {
	prompt := fmt.Sprintf(`Genere exactement %d quetes pour un jeu de battle entre IA.
Reponds uniquement en JSON valide, sans markdown.
Les sujets doivent etre amusants, originaux, ouverts, parfois absurdes, mais toujours debatables.
Au minimum 2 quetes doivent avoir un angle clairement humoristique.
Ces 2 quetes humoristiques doivent rester argumentables et opposer au moins deux camps.
Evite les questions classiques, scolaires, trop generales ou deja vues.
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
	prompt := fmt.Sprintf(`Genere exactement %d quetes de jeu de role.
Reponds uniquement en JSON valide, sans markdown.
Les quetes doivent etre variees, jouables, immersives et differentes les unes des autres.
Format:
[{"title":"...","summary":"resume court","prompt":"prompt complet jouable","theme":"fantasy|sf|horreur|steampunk|modern","level":"facile|moyen|difficile","xp":80,"coin":30,"metadata":{"ton":"..."}}]`, cronQuestLimit)

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
	if (now.Hour()-8)%2 == 0 {
		return "mistral"
	}
	return "openai"
}

func providerKeyEnvForHour(now time.Time) string {
	if providerNameForHour(now) == "mistral" {
		return "MISTRAL_AI_KEY"
	}
	return "OPEN_AI_KEY"
}

func providerModelEnvForHour(now time.Time) string {
	if providerNameForHour(now) == "mistral" {
		return "MISTRAL_AI_MODEL"
	}
	return "OPEN_AI_MODEL"
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
		message = " " + fmt.Sprintf(format, args...)
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
		message,
	)
}

func preview(value string, maxLength int) string {
	clean := strings.ReplaceAll(value, "\n", " ")
	clean = strings.ReplaceAll(clean, "\r", " ")
	if len(clean) <= maxLength {
		return clean
	}
	return clean[:maxLength] + "...(truncated)"
}
