package admin

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/provider"
	"cgwm/battle/internal/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const adminCookieName = "go_battle_admin"

type Server struct {
	db        *gorm.DB
	templates *template.Template
}

type dashboardData struct {
	AdminUsername string
	Flash         string
	Error         string
	Health        healthData
	Stats         statsData
	Config        configData
	Recent        recentData
}

type healthData struct {
	DatabaseOK bool
	Database   string
	Now        string
}

type statsData struct {
	Users          int64
	BattleQuests   int64
	RolePlayQuests int64
	Battles        int64
	LiveSessions   int64
	LiveStreaming  int64
	LiveEnded      int64
}

type configData struct {
	AppPort               string
	GinMode               string
	MaxConcurrentRequests string
	QueueTimeoutSeconds   string
	MaxBodyBytes          string
	DBMaxOpenConns        string
	DBMaxIdleConns        string
}

type recentData struct {
	BattleQuests   []models.QuestIaBattle
	RolePlayQuests []models.RolePlayQuestTemplate
	Battles        []models.BattleSave
	LiveSessions   []models.LiveSession
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

func Register(router *gin.Engine, db *gorm.DB) {
	server := &Server{
		db:        db,
		templates: template.Must(template.New("admin").Funcs(template.FuncMap{"json": toJSON}).Parse(adminHTML)),
	}

	router.GET("/admin/login", server.loginPage)
	router.POST("/admin/login", server.login)
	router.POST("/admin/logout", server.requireAdmin(), server.logout)

	group := router.Group("/admin")
	group.Use(server.requireAdmin())
	group.GET("", server.dashboard)
	group.POST("/quests/battle", server.createBattleQuest)
	group.POST("/quests/rp", server.createRolePlayQuest)
	group.POST("/generate/battle", server.generateBattleQuests)
	group.POST("/generate/rp", server.generateRolePlayQuests)
	group.POST("/live/:id/end", server.endLiveSession)
}

func (s *Server) loginPage(c *gin.Context) {
	s.render(c, "login", gin.H{
		"Error": c.Query("error"),
	})
}

func (s *Server) login(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	password := c.PostForm("password")

	if !adminPasswordConfigured() {
		s.render(c, "login", gin.H{
			"Error": "ADMIN_PASSWORD ou ADMIN_PASSWORD_BCRYPT doit etre configure pour activer l'admin.",
		})
		return
	}

	if username != adminUsername() || !checkAdminPassword(password) {
		s.render(c, "login", gin.H{"Error": "Identifiants invalides."})
		return
	}

	token, err := makeAdminSession(username)
	if err != nil {
		s.render(c, "login", gin.H{"Error": "Impossible de creer la session admin."})
		return
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     adminCookieName,
		Value:    token,
		Path:     "/admin",
		MaxAge:   int((8 * time.Hour).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   c.Request.TLS != nil,
	})
	c.Redirect(http.StatusSeeOther, "/admin")
}

func (s *Server) logout(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     adminCookieName,
		Value:    "",
		Path:     "/admin",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   c.Request.TLS != nil,
	})
	c.Redirect(http.StatusSeeOther, "/admin/login")
}

func (s *Server) dashboard(c *gin.Context) {
	data := s.dashboardData(c)
	data.Flash = c.Query("flash")
	data.Error = c.Query("error")
	s.render(c, "dashboard", data)
}

func (s *Server) createBattleQuest(c *gin.Context) {
	metadata := parseMetadata(c.PostForm("metadata"))
	quest := models.QuestIaBattle{
		Slug:     defaultSlug(c.PostForm("slug"), c.PostForm("title")),
		Title:    c.PostForm("title"),
		Content:  c.PostForm("content"),
		Level:    c.PostForm("level"),
		Point:    formInt(c, "point"),
		Theme:    c.PostForm("theme"),
		Xp:       formInt(c, "xp"),
		Coin:     formInt(c, "coin"),
		Mode:     defaultValue(c.PostForm("mode"), constants.ModeBattleIA),
		Source:   "admin",
		Status:   defaultValue(c.PostForm("status"), constants.QuestStatusPublished),
		Metadata: datatypes.JSON(metadata),
	}
	if quest.Title == "" || quest.Content == "" {
		s.redirectError(c, "Titre et contenu requis pour la quete battle.")
		return
	}
	if err := s.db.WithContext(c.Request.Context()).Create(&quest).Error; err != nil {
		s.redirectError(c, "Impossible de creer la quete battle: "+err.Error())
		return
	}
	s.redirectFlash(c, "Quete battle creee.")
}

func (s *Server) createRolePlayQuest(c *gin.Context) {
	metadata := parseMetadata(c.PostForm("metadata"))
	quest := models.RolePlayQuestTemplate{
		Slug:     defaultSlug(c.PostForm("slug"), c.PostForm("title")),
		Title:    c.PostForm("title"),
		Summary:  c.PostForm("summary"),
		Prompt:   c.PostForm("prompt"),
		Theme:    c.PostForm("theme"),
		Level:    c.PostForm("level"),
		Xp:       formInt(c, "xp"),
		Coin:     formInt(c, "coin"),
		Source:   "admin",
		Status:   defaultValue(c.PostForm("status"), constants.QuestStatusPublished),
		Metadata: datatypes.JSON(metadata),
	}
	if quest.Title == "" || quest.Prompt == "" {
		s.redirectError(c, "Titre et prompt requis pour la quete RP.")
		return
	}
	if err := s.db.WithContext(c.Request.Context()).Create(&quest).Error; err != nil {
		s.redirectError(c, "Impossible de creer la quete RP: "+err.Error())
		return
	}
	s.redirectFlash(c, "Quete RP creee.")
}

func (s *Server) generateBattleQuests(c *gin.Context) {
	apiKey := c.PostForm("api_key")
	providerName := c.PostForm("provider")
	model := c.PostForm("model")
	count := formInt(c, "count")
	if count <= 0 || count > 20 {
		count = 10
	}
	url, err := service.ProviderURL(providerName)
	if err != nil || apiKey == "" || model == "" {
		s.redirectError(c, "Provider, modele et cle API sont requis pour generer.")
		return
	}

	result, err := generateBattleQuestPayload(c.Request.Context(), url, apiKey, model, count)
	if err != nil {
		s.redirectError(c, "Generation battle impossible: "+err.Error())
		return
	}

	created := 0
	for _, item := range result {
		if item.Title == "" || item.Content == "" {
			continue
		}
		meta, _ := json.Marshal(item.Meta)
		quest := models.QuestIaBattle{
			Slug:     defaultSlug("", item.Title),
			Title:    item.Title,
			Content:  item.Content,
			Level:    item.Level,
			Point:    item.Point,
			Theme:    item.Theme,
			Xp:       item.Xp,
			Coin:     item.Coin,
			Mode:     constants.ModeBattleIA,
			Source:   "admin_ai",
			Status:   constants.QuestStatusPublished,
			Metadata: datatypes.JSON(meta),
		}
		if err := s.db.WithContext(c.Request.Context()).Create(&quest).Error; err == nil {
			created++
		}
	}

	s.redirectFlash(c, fmt.Sprintf("%d quetes battle generees.", created))
}

func (s *Server) generateRolePlayQuests(c *gin.Context) {
	apiKey := c.PostForm("api_key")
	providerName := c.PostForm("provider")
	model := c.PostForm("model")
	count := formInt(c, "count")
	if count <= 0 || count > 20 {
		count = 10
	}
	url, err := service.ProviderURL(providerName)
	if err != nil || apiKey == "" || model == "" {
		s.redirectError(c, "Provider, modele et cle API sont requis pour generer.")
		return
	}

	result, err := generateRolePlayQuestPayload(c.Request.Context(), url, apiKey, model, count)
	if err != nil {
		s.redirectError(c, "Generation RP impossible: "+err.Error())
		return
	}

	created := 0
	for _, item := range result {
		if item.Title == "" || item.Prompt == "" {
			continue
		}
		meta, _ := json.Marshal(item.Meta)
		quest := models.RolePlayQuestTemplate{
			Slug:     defaultSlug("", item.Title),
			Title:    item.Title,
			Summary:  item.Summary,
			Prompt:   item.Prompt,
			Theme:    item.Theme,
			Level:    item.Level,
			Xp:       item.Xp,
			Coin:     item.Coin,
			Source:   "admin_ai",
			Status:   constants.QuestStatusPublished,
			Metadata: datatypes.JSON(meta),
		}
		if err := s.db.WithContext(c.Request.Context()).Create(&quest).Error; err == nil {
			created++
		}
	}

	s.redirectFlash(c, fmt.Sprintf("%d quetes RP generees.", created))
}

func (s *Server) endLiveSession(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		s.redirectError(c, "Live session invalide.")
		return
	}

	now := time.Now()
	err = s.db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var live models.LiveSession
		if err := tx.Where("id = ?", uint(id)).First(&live).Error; err != nil {
			return err
		}

		var last models.LiveEvent
		nextSequence := 1
		if err := tx.Where("live_session_id = ?", live.Id).Order("sequence DESC").First(&last).Error; err == nil {
			nextSequence = last.Sequence + 1
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		payload, _ := json.Marshal(gin.H{"status": constants.LiveStatusEnded, "message": "ended from admin"})
		if err := tx.Create(&models.LiveEvent{
			LiveSessionID: live.Id,
			Sequence:      nextSequence,
			EventType:     "status",
			AuthorType:    constants.AuthorTypeSystem,
			AuthorName:    "admin",
			Payload:       datatypes.JSON(payload),
		}).Error; err != nil {
			return err
		}

		return tx.Model(&models.LiveSession{}).Where("id = ?", live.Id).Updates(map[string]any{
			"status":        constants.LiveStatusEnded,
			"ended_at":      &now,
			"last_event_at": &now,
		}).Error
	})
	if err != nil {
		s.redirectError(c, "Impossible de terminer le live: "+err.Error())
		return
	}
	s.redirectFlash(c, "Live termine.")
}

func (s *Server) dashboardData(c *gin.Context) dashboardData {
	data := dashboardData{
		AdminUsername: adminUsername(),
		Health: healthData{
			Now: time.Now().Format(time.RFC3339),
		},
		Config: configData{
			AppPort:               env("APP_PORT", "8080"),
			GinMode:               env("GIN_MODE", "debug"),
			MaxConcurrentRequests: env("APP_MAX_CONCURRENT_REQUESTS", "8"),
			QueueTimeoutSeconds:   env("APP_QUEUE_TIMEOUT_SECONDS", "10"),
			MaxBodyBytes:          env("APP_MAX_BODY_BYTES", "1048576"),
			DBMaxOpenConns:        env("DB_MAX_OPEN_CONNS", "25"),
			DBMaxIdleConns:        env("DB_MAX_IDLE_CONNS", "10"),
		},
	}

	sqlDB, err := s.db.DB()
	if err != nil {
		data.Health.Database = err.Error()
	} else if err := sqlDB.PingContext(c.Request.Context()); err != nil {
		data.Health.Database = err.Error()
	} else {
		data.Health.DatabaseOK = true
		data.Health.Database = "ok"
	}

	countModel(c.Request.Context(), s.db, &models.Users{}, &data.Stats.Users)
	countModel(c.Request.Context(), s.db, &models.QuestIaBattle{}, &data.Stats.BattleQuests)
	countModel(c.Request.Context(), s.db, &models.RolePlayQuestTemplate{}, &data.Stats.RolePlayQuests)
	countModel(c.Request.Context(), s.db, &models.BattleSave{}, &data.Stats.Battles)
	countModel(c.Request.Context(), s.db, &models.LiveSession{}, &data.Stats.LiveSessions)
	s.db.WithContext(c.Request.Context()).Model(&models.LiveSession{}).Where("status = ?", constants.LiveStatusStreaming).Count(&data.Stats.LiveStreaming)
	s.db.WithContext(c.Request.Context()).Model(&models.LiveSession{}).Where("status = ?", constants.LiveStatusEnded).Count(&data.Stats.LiveEnded)

	s.db.WithContext(c.Request.Context()).Order("created_at DESC").Limit(8).Find(&data.Recent.BattleQuests)
	s.db.WithContext(c.Request.Context()).Order("created_at DESC").Limit(8).Find(&data.Recent.RolePlayQuests)
	s.db.WithContext(c.Request.Context()).Order("updated_at DESC").Limit(8).Find(&data.Recent.Battles)
	s.db.WithContext(c.Request.Context()).Order("updated_at DESC").Limit(8).Find(&data.Recent.LiveSessions)

	return data
}

func (s *Server) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(adminCookieName)
		if err != nil || !verifyAdminSession(cookie) {
			c.Redirect(http.StatusSeeOther, "/admin/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *Server) render(c *gin.Context, name string, data any) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(c.Writer, name, data); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}
}

func (s *Server) redirectFlash(c *gin.Context, message string) {
	c.Redirect(http.StatusSeeOther, "/admin?flash="+urlQueryEscape(message))
}

func (s *Server) redirectError(c *gin.Context, message string) {
	c.Redirect(http.StatusSeeOther, "/admin?error="+urlQueryEscape(message))
}

func generateBattleQuestPayload(ctx context.Context, url string, apiKey string, model string, count int) ([]generatedBattleQuest, error) {
	prompt := fmt.Sprintf(`Genere exactement %d quetes pour un jeu de battle entre IA.
Reponds uniquement en JSON valide, sans markdown.
Format:
[{"title":"...","content":"question debat complete","level":"facile|moyen|difficile","theme":"...","point":10,"xp":25,"coin":5,"metadata":{"angle":"..."}}]`, count)
	response, err := callAdminProvider(ctx, url, apiKey, model, prompt)
	if err != nil {
		return nil, err
	}
	var quests []generatedBattleQuest
	if err := json.Unmarshal([]byte(cleanJSON(response)), &quests); err != nil {
		return nil, err
	}
	return quests, nil
}

func generateRolePlayQuestPayload(ctx context.Context, url string, apiKey string, model string, count int) ([]generatedRolePlayQuest, error) {
	prompt := fmt.Sprintf(`Genere exactement %d quetes de jeu de role.
Reponds uniquement en JSON valide, sans markdown.
Format:
[{"title":"...","summary":"resume court","prompt":"prompt complet jouable","theme":"fantasy|sf|horreur|steampunk|modern","level":"facile|moyen|difficile","xp":80,"coin":30,"metadata":{"ton":"..."}}]`, count)
	response, err := callAdminProvider(ctx, url, apiKey, model, prompt)
	if err != nil {
		return nil, err
	}
	var quests []generatedRolePlayQuest
	if err := json.Unmarshal([]byte(cleanJSON(response)), &quests); err != nil {
		return nil, err
	}
	return quests, nil
}

func callAdminProvider(ctx context.Context, url string, apiKey string, model string, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	ai := provider.NewsProvider(apiKey, url, model)
	return ai.Chat(ctx, []provider.ProviderMessage{{Role: "system", Content: prompt}})
}

func checkAdminPassword(password string) bool {
	if hash := os.Getenv("ADMIN_PASSWORD_BCRYPT"); hash != "" {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	}
	if plain := os.Getenv("ADMIN_PASSWORD"); plain != "" {
		return subtleEqual(plain, password)
	}
	if env("GIN_MODE", "debug") != "release" {
		return subtleEqual("admin", password)
	}
	return false
}

func adminPasswordConfigured() bool {
	return os.Getenv("ADMIN_PASSWORD") != "" || os.Getenv("ADMIN_PASSWORD_BCRYPT") != "" || env("GIN_MODE", "debug") != "release"
}

func makeAdminSession(username string) (string, error) {
	expires := time.Now().Add(8 * time.Hour).Unix()
	nonceBytes := make([]byte, 12)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", err
	}
	payload := fmt.Sprintf("%s|%d|%s", username, expires, hex.EncodeToString(nonceBytes))
	return payload + "|" + sign(payload), nil
}

func verifyAdminSession(token string) bool {
	parts := strings.Split(token, "|")
	if len(parts) != 4 {
		return false
	}
	payload := strings.Join(parts[:3], "|")
	if !subtleEqual(sign(payload), parts[3]) {
		return false
	}
	if parts[0] != adminUsername() {
		return false
	}
	expires, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false
	}
	return time.Now().Unix() < expires
}

func sign(payload string) string {
	mac := hmac.New(sha256.New, []byte(adminSessionSecret()))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func subtleEqual(a string, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}

func adminUsername() string {
	return env("ADMIN_USERNAME", "admin")
}

func adminSessionSecret() string {
	if secret := os.Getenv("ADMIN_SESSION_SECRET"); secret != "" {
		return secret
	}
	return env("JWT_SECRET", "dev-secret-change-me")
}

func countModel(ctx context.Context, db *gorm.DB, model any, out *int64) {
	db.WithContext(ctx).Model(model).Count(out)
}

func parseMetadata(value string) []byte {
	value = strings.TrimSpace(value)
	if value == "" {
		return []byte("{}")
	}
	if json.Valid([]byte(value)) {
		return []byte(value)
	}
	return []byte("{}")
}

func formInt(c *gin.Context, key string) int {
	value, _ := strconv.Atoi(c.PostForm(key))
	return value
}

func defaultSlug(slug string, title string) string {
	if strings.TrimSpace(slug) != "" {
		return slug
	}
	base := strings.ToLower(strings.TrimSpace(title))
	base = strings.ReplaceAll(base, " ", "-")
	base = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, base)
	if base == "" {
		base = "quest"
	}
	return fmt.Sprintf("%s-%d", base, time.Now().Unix())
}

func defaultValue(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func cleanJSON(response string) string {
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	return strings.TrimSpace(response)
}

func toJSON(value any) string {
	data, _ := json.MarshalIndent(value, "", "  ")
	return string(data)
}

func env(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func urlQueryEscape(value string) string {
	replacer := strings.NewReplacer(" ", "+", "\n", "+", "&", "%26", "?", "%3F", "=", "%3D")
	return replacer.Replace(value)
}
