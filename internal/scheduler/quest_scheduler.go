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
		log.Printf("[quest-cron] disabled by AI_QUEST_CRON_ENABLED=false")
		return
	}

	location := loadLocation()
	log.Printf("[quest-cron] enabled timezone=%s hours=08-21 limit=%d", location.String(), cronQuestLimit)

	go runHourlyJob(db, location, "battle", runBattleQuestJob)
	go runHourlyJob(db, location, "roleplay", runRolePlayQuestJob)
}

func runHourlyJob(
	db *gorm.DB,
	location *time.Location,
	jobName string,
	job func(context.Context, *gorm.DB, time.Time, aiProviderConfig) error,
) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

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

		cfg, ok := providerForHour(localNow)
		if !ok {
			log.Printf("[quest-cron] %s skipped hour=%s provider env missing", jobName, runKey)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		err := withMySQLLock(ctx, db, "go_battle_ia_"+jobName+"_"+runKey, func(ctx context.Context) error {
			return job(ctx, db, localNow, cfg)
		})
		cancel()
		if err != nil {
			log.Printf("[quest-cron] %s failed provider=%s hour=%s err=%v", jobName, cfg.Name, runKey, err)
			continue
		}

		log.Printf("[quest-cron] %s completed provider=%s hour=%s", jobName, cfg.Name, runKey)
	}
}

func runBattleQuestJob(ctx context.Context, db *gorm.DB, runAt time.Time, cfg aiProviderConfig) error {
	quests, err := generateBattleQuests(ctx, cfg)
	if err != nil {
		return err
	}

	created := 0
	for _, item := range quests {
		if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.Content) == "" {
			continue
		}
		meta, _ := json.Marshal(ensureMetadata(item.Meta, cfg, runAt))
		quest := models.QuestIaBattle{
			Slug:     uniqueSlug(item.Title, runAt),
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
		} else {
			log.Printf("[quest-cron] battle insert skipped title=%q err=%v", item.Title, err)
		}
	}

	if created == 0 {
		return fmt.Errorf("no battle quest created")
	}
	return nil
}

func runRolePlayQuestJob(ctx context.Context, db *gorm.DB, runAt time.Time, cfg aiProviderConfig) error {
	quests, err := generateRolePlayQuests(ctx, cfg)
	if err != nil {
		return err
	}

	created := 0
	for _, item := range quests {
		if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.Prompt) == "" {
			continue
		}
		meta, _ := json.Marshal(ensureMetadata(item.Meta, cfg, runAt))
		quest := models.RolePlayQuestTemplate{
			Slug:     uniqueSlug(item.Title, runAt),
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
		} else {
			log.Printf("[quest-cron] roleplay insert skipped title=%q err=%v", item.Title, err)
		}
	}

	if created == 0 {
		return fmt.Errorf("no roleplay quest created")
	}
	return nil
}

func generateBattleQuests(ctx context.Context, cfg aiProviderConfig) ([]generatedBattleQuest, error) {
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

	response, err := callProvider(ctx, cfg, prompt)
	if err != nil {
		return nil, err
	}

	var quests []generatedBattleQuest
	if err := json.Unmarshal([]byte(cleanJSON(response)), &quests); err != nil {
		return nil, err
	}
	return quests, nil
}

func generateRolePlayQuests(ctx context.Context, cfg aiProviderConfig) ([]generatedRolePlayQuest, error) {
	prompt := fmt.Sprintf(`Genere exactement %d quetes de jeu de role.
Reponds uniquement en JSON valide, sans markdown.
Les quetes doivent etre variees, jouables, immersives et differentes les unes des autres.
Format:
[{"title":"...","summary":"resume court","prompt":"prompt complet jouable","theme":"fantasy|sf|horreur|steampunk|modern","level":"facile|moyen|difficile","xp":80,"coin":30,"metadata":{"ton":"..."}}]`, cronQuestLimit)

	response, err := callProvider(ctx, cfg, prompt)
	if err != nil {
		return nil, err
	}

	var quests []generatedRolePlayQuest
	if err := json.Unmarshal([]byte(cleanJSON(response)), &quests); err != nil {
		return nil, err
	}
	return quests, nil
}

func callProvider(ctx context.Context, cfg aiProviderConfig, prompt string) (string, error) {
	url, err := service.ProviderURL(cfg.Name)
	if err != nil {
		return "", err
	}

	ai := provider.NewsProvider(cfg.APIKey, url, cfg.Model)
	return ai.Chat(ctx, []provider.ProviderMessage{{Role: "system", Content: prompt}})
}

func providerForHour(now time.Time) (aiProviderConfig, bool) {
	if (now.Hour()-8)%2 == 0 {
		return providerConfig("mistral", "MISTRAL_AI_KEY", "MISTRAL_AI_MODEL")
	}
	return providerConfig("openai", "OPEN_AI_KEY", "OPEN_AI_MODEL")
}

func providerConfig(name string, keyEnv string, modelEnv string) (aiProviderConfig, bool) {
	cfg := aiProviderConfig{
		Name:   name,
		APIKey: strings.TrimSpace(os.Getenv(keyEnv)),
		Model:  strings.TrimSpace(os.Getenv(modelEnv)),
	}
	return cfg, cfg.APIKey != "" && cfg.Model != ""
}

func withMySQLLock(ctx context.Context, db *gorm.DB, name string, fn func(context.Context) error) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	acquired, err := acquireLock(ctx, conn, name)
	if err != nil {
		return err
	}
	if !acquired {
		log.Printf("[quest-cron] lock not acquired name=%s", name)
		return nil
	}
	defer releaseLock(ctx, conn, name)

	return fn(ctx)
}

func acquireLock(ctx context.Context, conn *sql.Conn, name string) (bool, error) {
	var acquired int
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", name).Scan(&acquired); err != nil {
		return false, err
	}
	return acquired == 1, nil
}

func releaseLock(ctx context.Context, conn *sql.Conn, name string) {
	var released int
	if err := conn.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?)", name).Scan(&released); err != nil {
		log.Printf("[quest-cron] release lock failed name=%s err=%v", name, err)
	}
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

func uniqueSlug(title string, runAt time.Time) string {
	return slugify(title) + "-" + runAt.Format("20060102150405")
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
