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
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/provider"
	"cgwm/battle/internal/repository"
	"cgwm/battle/internal/scheduler"
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
	uiDir     string
}

var adminProcessStartedAt = time.Now()

var adminRequestStats struct {
	totalRequests  int64
	activeRequests int64
	status2xx      int64
	status3xx      int64
	status4xx      int64
	status5xx      int64
	totalLatencyNS int64
	maxLatencyNS   int64
}

type dashboardData struct {
	AdminUsername string
	Flash         string
	Error         string
	Health        healthData
	Stats         statsData
	Config        configData
	Cron          scheduler.CronSnapshot
	Usage         usageData
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

type usageData struct {
	Total       usageSummary
	Battle      usageSummary
	RolePlay    usageSummary
	Recent      []models.AIUsageRecord
	PricingHint string
}

type usageSummary struct {
	CallCount           int64
	PromptTokens        int64
	CompletionTokens    int64
	TotalTokens         int64
	EstimatedCostMicros int64
}

type adminAccountData struct {
	Id               uint      `json:"id"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	Pseudo           string    `json:"pseudo"`
	Email            string    `json:"email"`
	Avatar           string    `json:"avatar"`
	Xp               int       `json:"xp"`
	Coin             int       `json:"coin"`
	BattleCount      int64     `json:"battleCount"`
	RolePlayCount    int64     `json:"rolePlayCount"`
	IAProfileCount   int64     `json:"iaProfileCount"`
	LiveSessionCount int64     `json:"liveSessionCount"`
}

type accountSummaryData struct {
	TotalAccounts     int64 `json:"totalAccounts"`
	UpdatedLast7Days  int64 `json:"updatedLast7Days"`
	UpdatedLast30Days int64 `json:"updatedLast30Days"`
	TotalXP           int64 `json:"totalXp"`
	TotalCoins        int64 `json:"totalCoins"`
}

type adminAccountsResponse struct {
	Summary  accountSummaryData `json:"summary"`
	Accounts []adminAccountData `json:"accounts"`
}

type systemData struct {
	Health   healthData        `json:"health"`
	Config   configData        `json:"config"`
	Runtime  runtimeStatsData  `json:"runtime"`
	Requests requestStatsData  `json:"requests"`
	Database databaseStatsData `json:"database"`
	Network  networkStatsData  `json:"network"`
}

type runtimeStatsData struct {
	StartedAt      string `json:"startedAt"`
	UptimeSeconds  int64  `json:"uptimeSeconds"`
	GoVersion      string `json:"goVersion"`
	GOOS           string `json:"goos"`
	GOARCH         string `json:"goarch"`
	NumCPU         int    `json:"numCpu"`
	NumGoroutine   int    `json:"numGoroutine"`
	AllocBytes     uint64 `json:"allocBytes"`
	HeapAllocBytes uint64 `json:"heapAllocBytes"`
	SysBytes       uint64 `json:"sysBytes"`
	NumGC          uint32 `json:"numGc"`
}

type requestStatsData struct {
	TotalRequests    int64   `json:"totalRequests"`
	ActiveRequests   int64   `json:"activeRequests"`
	Status2xx        int64   `json:"status2xx"`
	Status3xx        int64   `json:"status3xx"`
	Status4xx        int64   `json:"status4xx"`
	Status5xx        int64   `json:"status5xx"`
	AverageLatencyMS float64 `json:"averageLatencyMs"`
	MaxLatencyMS     float64 `json:"maxLatencyMs"`
}

type databaseStatsData struct {
	MaxOpenConnections int           `json:"maxOpenConnections"`
	OpenConnections    int           `json:"openConnections"`
	InUse              int           `json:"inUse"`
	Idle               int           `json:"idle"`
	WaitCount          int64         `json:"waitCount"`
	WaitDuration       time.Duration `json:"waitDuration"`
	MaxIdleClosed      int64         `json:"maxIdleClosed"`
	MaxIdleTimeClosed  int64         `json:"maxIdleTimeClosed"`
	MaxLifetimeClosed  int64         `json:"maxLifetimeClosed"`
}

type networkStatsData struct {
	LiveSessions  int64 `json:"liveSessions"`
	LiveStreaming int64 `json:"liveStreaming"`
	LiveEnded     int64 `json:"liveEnded"`
	LiveViewers   int64 `json:"liveViewers"`
	Arenas        int64 `json:"arenas"`
	CoopParties   int64 `json:"coopParties"`
}

type nexusCoinResponse struct {
	Stats       nexusCoinStats         `json:"stats"`
	Estimations []nexusCoinEstimate    `json:"estimations"`
	Plans       []models.NexusCoinPlan `json:"plans"`
}

type nexusCoinStats struct {
	CallCount                 int64   `json:"callCount"`
	TotalTokens               int64   `json:"totalTokens"`
	TotalCostMicros           int64   `json:"totalCostMicros"`
	AverageTokensPerCall      int64   `json:"averageTokensPerCall"`
	AverageCostMicrosPerToken float64 `json:"averageCostMicrosPerToken"`
	MarginPercent             int     `json:"marginPercent"`
	CostSource                string  `json:"costSource"`
}

type nexusCoinEstimate struct {
	Slug                   string `json:"slug"`
	Name                   string `json:"name"`
	Subtitle               string `json:"subtitle"`
	Description            string `json:"description"`
	TokenBudget            int64  `json:"tokenBudget"`
	NexusCoins             int64  `json:"nexusCoins"`
	BaseCostMicros         int64  `json:"baseCostMicros"`
	MarginPercent          int    `json:"marginPercent"`
	PriceMicros            int64  `json:"priceMicros"`
	EstimatedCalls         int64  `json:"estimatedCalls"`
	EstimatedTokensPerCall int64  `json:"estimatedTokensPerCall"`
	CostSource             string `json:"costSource"`
}

type nexusCoinPlanInput struct {
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Subtitle      string `json:"subtitle"`
	Description   string `json:"description"`
	Status        string `json:"status"`
	Position      int    `json:"position"`
	TokenBudget   int64  `json:"tokenBudget"`
	NexusCoins    int64  `json:"nexusCoins"`
	MarginPercent int    `json:"marginPercent"`
}

type adminRolePlayQuestsResponse struct {
	Stats  adminRolePlayQuestStats  `json:"stats"`
	Quests []adminRolePlayQuestData `json:"quests"`
}

type adminRolePlayQuestStats struct {
	TotalQuests   int64 `json:"totalQuests"`
	Published     int64 `json:"published"`
	Draft         int64 `json:"draft"`
	Archived      int64 `json:"archived"`
	TotalArcs     int64 `json:"totalArcs"`
	TotalChapters int64 `json:"totalChapters"`
}

type adminRolePlayQuestData struct {
	Id           uint                   `json:"id"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
	Slug         string                 `json:"slug"`
	Title        string                 `json:"title"`
	Summary      string                 `json:"summary"`
	Prompt       string                 `json:"prompt"`
	Theme        string                 `json:"theme"`
	Level        string                 `json:"level"`
	Xp           int                    `json:"xp"`
	Coin         int                    `json:"coin"`
	Source       string                 `json:"source"`
	Status       string                 `json:"status"`
	ArcCount     int                    `json:"arcCount"`
	ChapterCount int                    `json:"chapterCount"`
	Arcs         []adminRolePlayArcData `json:"arcs"`
}

type adminRolePlayArcData struct {
	Id           uint                       `json:"id"`
	Position     int                        `json:"position"`
	Title        string                     `json:"title"`
	Summary      string                     `json:"summary"`
	Objective    string                     `json:"objective"`
	ChapterCount int                        `json:"chapterCount"`
	Chapters     []adminRolePlayChapterData `json:"chapters"`
}

type adminRolePlayChapterData struct {
	Id        uint   `json:"id"`
	Position  int    `json:"position"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	Objective string `json:"objective"`
	IsBoss    bool   `json:"isBoss"`
	Xp        int    `json:"xp"`
	Coin      int    `json:"coin"`
}

type adminRolePlayQuestUpdateInput struct {
	Xp     int    `json:"xp"`
	Coin   int    `json:"coin"`
	Status string `json:"status"`
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

func Register(router *gin.Engine, db *gorm.DB) {
	server := &Server{
		db:        db,
		templates: template.Must(template.New("admin").Funcs(adminTemplateFuncs()).Parse(adminHTML)),
		uiDir:     env("ADMIN_UI_DIR", "admin/out"),
	}

	router.GET("/admin/login", server.loginPage)
	router.POST("/admin/login", server.login)
	router.POST("/admin/logout", server.requireAdmin(), server.logout)

	api := router.Group("/admin/api")
	api.Use(server.requireAdminAPI())
	api.GET("/dashboard", server.dashboardAPI)
	api.GET("/accounts", server.accountsAPI)
	api.GET("/system", server.systemAPI)
	api.GET("/usage", server.usageAPI)
	api.GET("/quests", server.questsAPI)
	api.GET("/live", server.liveAPI)
	api.GET("/roleplay-quests", server.rolePlayQuestsAdminAPI)
	api.PATCH("/roleplay-quests/:id", server.updateRolePlayQuestAdminAPI)
	api.PUT("/roleplay-quests/:id", server.updateRolePlayQuestAdminAPI)
	api.DELETE("/roleplay-quests", server.clearRolePlayQuestsAdminAPI)
	api.DELETE("/roleplay-quests/:id", server.deleteRolePlayQuestAdminAPI)
	api.GET("/nexus-coin", server.nexusCoinAPI)
	api.POST("/nexus-coin/plans", server.createNexusCoinPlanAPI)
	api.PUT("/nexus-coin/plans/:id", server.updateNexusCoinPlanAPI)
	api.PATCH("/nexus-coin/plans/:id", server.updateNexusCoinPlanAPI)
	api.DELETE("/nexus-coin/plans/:id", server.deleteNexusCoinPlanAPI)

	router.GET("/api/v1/nexus-coin/plans", server.publicNexusCoinPlansAPI)

	group := router.Group("/admin")
	group.Use(server.requireAdmin())
	group.GET("", server.adminAppPage)
	group.POST("/quests/battle", server.createBattleQuest)
	group.POST("/quests/rp", server.createRolePlayQuest)
	group.POST("/generate/battle", server.generateBattleQuests)
	group.POST("/generate/rp", server.generateRolePlayQuests)
	group.POST("/live/:id/end", server.endLiveSession)

	router.NoRoute(server.adminNoRoute)
}

func RequestMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		atomic.AddInt64(&adminRequestStats.totalRequests, 1)
		atomic.AddInt64(&adminRequestStats.activeRequests, 1)
		defer atomic.AddInt64(&adminRequestStats.activeRequests, -1)

		c.Next()

		duration := time.Since(start).Nanoseconds()
		atomic.AddInt64(&adminRequestStats.totalLatencyNS, duration)
		for {
			current := atomic.LoadInt64(&adminRequestStats.maxLatencyNS)
			if duration <= current || atomic.CompareAndSwapInt64(&adminRequestStats.maxLatencyNS, current, duration) {
				break
			}
		}

		switch status := c.Writer.Status(); {
		case status >= 200 && status < 300:
			atomic.AddInt64(&adminRequestStats.status2xx, 1)
		case status >= 300 && status < 400:
			atomic.AddInt64(&adminRequestStats.status3xx, 1)
		case status >= 400 && status < 500:
			atomic.AddInt64(&adminRequestStats.status4xx, 1)
		case status >= 500:
			atomic.AddInt64(&adminRequestStats.status5xx, 1)
		}
	}
}

func adminTemplateFuncs() template.FuncMap {
	return template.FuncMap{"json": toJSON, "usdMicros": usdMicros}
}

func (s *Server) loginPage(c *gin.Context) {
	if s.serveAdminUI(c, "/admin/login") {
		return
	}
	s.render(c, "login", gin.H{
		"Error": c.Query("error"),
	})
}

func (s *Server) login(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	password := c.PostForm("password")

	adminAuthLog(
		"login attempt ip=%s host=%s tls=%t user_agent=%q username=%q expected_username=%q",
		c.ClientIP(),
		c.Request.Host,
		c.Request.TLS != nil,
		c.Request.UserAgent(),
		username,
		adminUsername(),
	)
	adminAuthLog(
		"env status gin_mode=%q admin_password_set=%t admin_password_bcrypt_set=%t admin_session_secret_set=%t jwt_secret_set=%t",
		env("GIN_MODE", "debug"),
		os.Getenv("ADMIN_PASSWORD") != "",
		os.Getenv("ADMIN_PASSWORD_BCRYPT") != "",
		os.Getenv("ADMIN_SESSION_SECRET") != "",
		os.Getenv("JWT_SECRET") != "",
	)

	if !adminPasswordConfigured() {
		adminAuthLog("login blocked: no admin password configured")
		s.render(c, "login", gin.H{
			"Error": "ADMIN_PASSWORD ou ADMIN_PASSWORD_BCRYPT doit etre configure pour activer l'admin.",
		})
		return
	}

	if username != adminUsername() {
		adminAuthLog("login rejected: username mismatch input=%q expected=%q", username, adminUsername())
		s.render(c, "login", gin.H{"Error": "Identifiants invalides."})
		return
	}

	if !checkAdminPassword(password) {
		adminAuthLog("login rejected: password validation failed for username=%q", username)
		s.render(c, "login", gin.H{"Error": "Identifiants invalides."})
		return
	}

	token, err := makeAdminSession(username)
	if err != nil {
		adminAuthLog("login failed: cannot create admin session token: %v", err)
		s.render(c, "login", gin.H{"Error": "Impossible de creer la session admin."})
		return
	}
	adminAuthLog("login success: session created for username=%q", username)

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

func (s *Server) adminAppPage(c *gin.Context) {
	if s.serveAdminUI(c, "/admin") {
		return
	}
	s.dashboard(c)
}

func (s *Server) adminNoRoute(c *gin.Context) {
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		c.Status(http.StatusNotFound)
		return
	}
	if !strings.HasPrefix(c.Request.URL.Path, "/admin/") {
		c.Status(http.StatusNotFound)
		return
	}

	path := c.Request.URL.Path
	publicAsset := strings.HasPrefix(path, "/admin/_next/") || strings.Contains(filepath.Base(path), ".")
	if path != "/admin/login" && !publicAsset {
		if !s.ensureAdminOrRedirect(c) {
			return
		}
	}
	if s.serveAdminUI(c, path) {
		return
	}
	c.Status(http.StatusNotFound)
}

func (s *Server) serveAdminUI(c *gin.Context, requestPath string) bool {
	if strings.TrimSpace(s.uiDir) == "" {
		return false
	}

	rel := strings.TrimPrefix(requestPath, "/admin")
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		rel = "index.html"
	} else if !strings.Contains(filepath.Base(rel), ".") {
		rel = strings.TrimSuffix(rel, "/") + "/index.html"
	}
	rel = filepath.Clean(filepath.FromSlash(rel))
	if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return false
	}

	fullPath := filepath.Join(s.uiDir, rel)
	if info, err := os.Stat(fullPath); err != nil || info.IsDir() {
		return false
	}
	http.ServeFile(c.Writer, c.Request, fullPath)
	return true
}

func (s *Server) dashboardAPI(c *gin.Context) {
	c.JSON(http.StatusOK, s.dashboardData(c))
}

func (s *Server) accountsAPI(c *gin.Context) {
	accounts, err := s.accountsData(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, accounts)
}

func (s *Server) systemAPI(c *gin.Context) {
	c.JSON(http.StatusOK, s.systemData(c))
}

func (s *Server) usageAPI(c *gin.Context) {
	c.JSON(http.StatusOK, s.usageDashboardData(c.Request.Context()))
}

func (s *Server) questsAPI(c *gin.Context) {
	data := s.dashboardData(c)
	c.JSON(http.StatusOK, gin.H{
		"stats":  data.Stats,
		"recent": gin.H{"battleQuests": data.Recent.BattleQuests, "rolePlayQuests": data.Recent.RolePlayQuests},
	})
}

func (s *Server) liveAPI(c *gin.Context) {
	data := s.dashboardData(c)
	c.JSON(http.StatusOK, gin.H{
		"stats":        data.Stats,
		"liveSessions": data.Recent.LiveSessions,
	})
}

func (s *Server) rolePlayQuestsAdminAPI(c *gin.Context) {
	data, err := s.rolePlayQuestsAdminData(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) updateRolePlayQuestAdminAPI(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid roleplay quest id"})
		return
	}
	var input adminRolePlayQuestUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid roleplay quest payload"})
		return
	}
	updates := map[string]any{
		"xp":   input.Xp,
		"coin": input.Coin,
	}
	if strings.TrimSpace(input.Status) != "" {
		updates["status"] = strings.TrimSpace(input.Status)
	}
	if err := s.db.WithContext(c.Request.Context()).
		Model(&models.RolePlayQuestTemplate{}).
		Where("id = ?", uint(id)).
		Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"updated": true})
}

func (s *Server) deleteRolePlayQuestAdminAPI(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid roleplay quest id"})
		return
	}
	if err := s.deleteRolePlayQuestTemplate(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) clearRolePlayQuestsAdminAPI(c *gin.Context) {
	if err := s.clearRolePlayQuestTemplates(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) nexusCoinAPI(c *gin.Context) {
	data, err := s.nexusCoinData(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) createNexusCoinPlanAPI(c *gin.Context) {
	var input nexusCoinPlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid nexus coin plan payload"})
		return
	}
	plan, err := s.saveNexusCoinPlan(c.Request.Context(), nil, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, plan)
}

func (s *Server) updateNexusCoinPlanAPI(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid nexus coin plan id"})
		return
	}
	var input nexusCoinPlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid nexus coin plan payload"})
		return
	}
	planID := uint(id)
	plan, err := s.saveNexusCoinPlan(c.Request.Context(), &planID, input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, plan)
}

func (s *Server) deleteNexusCoinPlanAPI(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid nexus coin plan id"})
		return
	}
	if err := s.db.WithContext(c.Request.Context()).Delete(&models.NexusCoinPlan{}, uint(id)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) publicNexusCoinPlansAPI(c *gin.Context) {
	if err := s.ensureDefaultNexusCoinPlans(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var plans []models.NexusCoinPlan
	if err := s.db.WithContext(c.Request.Context()).
		Where("status = ?", "active").
		Order("position ASC, id ASC").
		Find(&plans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"plans": plans})
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
		if !hasAdminGeneratedRolePlayStructure(item) {
			continue
		}
		input := adminGeneratedRolePlayQuestInput(item, defaultSlug("", item.Title))
		if _, err := service.NewQuestService(repository.NewQuestRepository(s.db)).CreateRolePlay(c.Request.Context(), input); err == nil {
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
		Cron: scheduler.Snapshot(),
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

	data.Usage = s.usageDashboardData(c.Request.Context())

	s.db.WithContext(c.Request.Context()).Order("created_at DESC").Limit(8).Find(&data.Recent.BattleQuests)
	s.db.WithContext(c.Request.Context()).Order("created_at DESC").Limit(8).Find(&data.Recent.RolePlayQuests)
	s.db.WithContext(c.Request.Context()).Order("updated_at DESC").Limit(8).Find(&data.Recent.Battles)
	s.db.WithContext(c.Request.Context()).Order("updated_at DESC").Limit(8).Find(&data.Recent.LiveSessions)

	return data
}

func (s *Server) accountsData(ctx context.Context) (adminAccountsResponse, error) {
	response := adminAccountsResponse{}
	db := s.db.WithContext(ctx)

	if err := db.Model(&models.Users{}).Count(&response.Summary.TotalAccounts).Error; err != nil {
		return response, err
	}
	now := time.Now()
	_ = db.Model(&models.Users{}).Where("updated_at >= ?", now.AddDate(0, 0, -7)).Count(&response.Summary.UpdatedLast7Days).Error
	_ = db.Model(&models.Users{}).Where("updated_at >= ?", now.AddDate(0, 0, -30)).Count(&response.Summary.UpdatedLast30Days).Error
	_ = db.Model(&models.Users{}).Select("COALESCE(SUM(xp), 0)").Scan(&response.Summary.TotalXP).Error
	_ = db.Model(&models.Users{}).Select("COALESCE(SUM(coin), 0)").Scan(&response.Summary.TotalCoins).Error

	var users []models.Users
	if err := db.Order("created_at DESC").Limit(200).Find(&users).Error; err != nil {
		return response, err
	}

	battleCounts := s.countByUser(ctx, &models.BattleSave{}, "owner_id")
	rolePlayCounts := s.countByUser(ctx, &models.RolePlaySession{}, "owner_id")
	profileCounts := s.countByUser(ctx, &models.IAProfile{}, "owner_id")
	liveCounts := s.countByUser(ctx, &models.LiveSession{}, "owner_id")

	response.Accounts = make([]adminAccountData, 0, len(users))
	for _, user := range users {
		response.Accounts = append(response.Accounts, adminAccountData{
			Id:               user.Id,
			CreatedAt:        user.CreatedAt,
			UpdatedAt:        user.UpdatedAt,
			Pseudo:           user.Pseudo,
			Email:            user.Email,
			Avatar:           user.Avatar,
			Xp:               user.Xp,
			Coin:             user.Coin,
			BattleCount:      battleCounts[user.Id],
			RolePlayCount:    rolePlayCounts[user.Id],
			IAProfileCount:   profileCounts[user.Id],
			LiveSessionCount: liveCounts[user.Id],
		})
	}

	return response, nil
}

func (s *Server) countByUser(ctx context.Context, model any, userColumn string) map[uint]int64 {
	type userCount struct {
		UserID uint
		Count  int64
	}
	rows := []userCount{}
	_ = s.db.WithContext(ctx).
		Model(model).
		Select(userColumn + " AS user_id, COUNT(*) AS count").
		Group(userColumn).
		Scan(&rows).Error

	counts := make(map[uint]int64, len(rows))
	for _, row := range rows {
		counts[row.UserID] = row.Count
	}
	return counts
}

func (s *Server) systemData(c *gin.Context) systemData {
	dashboard := s.dashboardData(c)
	data := systemData{
		Health:   dashboard.Health,
		Config:   dashboard.Config,
		Runtime:  runtimeSnapshot(),
		Requests: requestSnapshot(),
		Network: networkStatsData{
			LiveSessions:  dashboard.Stats.LiveSessions,
			LiveStreaming: dashboard.Stats.LiveStreaming,
			LiveEnded:     dashboard.Stats.LiveEnded,
		},
	}

	if sqlDB, err := s.db.DB(); err == nil {
		stats := sqlDB.Stats()
		data.Database = databaseStatsData{
			MaxOpenConnections: stats.MaxOpenConnections,
			OpenConnections:    stats.OpenConnections,
			InUse:              stats.InUse,
			Idle:               stats.Idle,
			WaitCount:          stats.WaitCount,
			WaitDuration:       stats.WaitDuration,
			MaxIdleClosed:      stats.MaxIdleClosed,
			MaxIdleTimeClosed:  stats.MaxIdleTimeClosed,
			MaxLifetimeClosed:  stats.MaxLifetimeClosed,
		}
	}

	db := s.db.WithContext(c.Request.Context())
	_ = db.Model(&models.LiveSession{}).Select("COALESCE(SUM(viewer_count), 0)").Scan(&data.Network.LiveViewers).Error
	_ = db.Model(&models.BattleArena{}).Count(&data.Network.Arenas).Error
	_ = db.Model(&models.CoopParty{}).Count(&data.Network.CoopParties).Error

	return data
}

func (s *Server) rolePlayQuestsAdminData(ctx context.Context) (adminRolePlayQuestsResponse, error) {
	response := adminRolePlayQuestsResponse{}
	db := s.db.WithContext(ctx)
	if err := db.Model(&models.RolePlayQuestTemplate{}).Count(&response.Stats.TotalQuests).Error; err != nil {
		return response, err
	}
	_ = db.Model(&models.RolePlayQuestTemplate{}).Where("status = ?", constants.QuestStatusPublished).Count(&response.Stats.Published).Error
	_ = db.Model(&models.RolePlayQuestTemplate{}).Where("status = ?", constants.QuestStatusDraft).Count(&response.Stats.Draft).Error
	_ = db.Model(&models.RolePlayQuestTemplate{}).Where("status = ?", constants.QuestStatusArchived).Count(&response.Stats.Archived).Error
	_ = db.Model(&models.RolePlayQuestArc{}).Count(&response.Stats.TotalArcs).Error
	_ = db.Model(&models.RolePlayQuestChapter{}).Count(&response.Stats.TotalChapters).Error

	var quests []models.RolePlayQuestTemplate
	if err := db.
		Preload("Arcs", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("position ASC").Order("id ASC")
		}).
		Preload("Arcs.Chapters", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("position ASC").Order("id ASC")
		}).
		Order("created_at DESC").
		Limit(200).
		Find(&quests).Error; err != nil {
		return response, err
	}

	response.Quests = make([]adminRolePlayQuestData, 0, len(quests))
	for _, quest := range quests {
		item := adminRolePlayQuestData{
			Id:        quest.Id,
			CreatedAt: quest.CreatedAt,
			UpdatedAt: quest.UpdatedAt,
			Slug:      quest.Slug,
			Title:     quest.Title,
			Summary:   quest.Summary,
			Prompt:    quest.Prompt,
			Theme:     quest.Theme,
			Level:     quest.Level,
			Xp:        quest.Xp,
			Coin:      quest.Coin,
			Source:    quest.Source,
			Status:    quest.Status,
			ArcCount:  len(quest.Arcs),
			Arcs:      make([]adminRolePlayArcData, 0, len(quest.Arcs)),
		}
		for _, arc := range quest.Arcs {
			arcData := adminRolePlayArcData{
				Id:           arc.Id,
				Position:     arc.Position,
				Title:        arc.Title,
				Summary:      arc.Summary,
				Objective:    arc.Objective,
				ChapterCount: len(arc.Chapters),
				Chapters:     make([]adminRolePlayChapterData, 0, len(arc.Chapters)),
			}
			item.ChapterCount += len(arc.Chapters)
			for _, chapter := range arc.Chapters {
				arcData.Chapters = append(arcData.Chapters, adminRolePlayChapterData{
					Id:        chapter.Id,
					Position:  chapter.Position,
					Title:     chapter.Title,
					Summary:   chapter.Summary,
					Objective: chapter.Objective,
					IsBoss:    chapter.IsBoss,
					Xp:        chapter.Xp,
					Coin:      chapter.Coin,
				})
			}
			item.Arcs = append(item.Arcs, arcData)
		}
		response.Quests = append(response.Quests, item)
	}
	return response, nil
}

func (s *Server) deleteRolePlayQuestTemplate(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.RolePlayQuestRun{}).
			Where("template_id = ?", id).
			Update("template_id", nil).Error; err != nil {
			return err
		}
		if err := tx.Where("template_id = ?", id).Delete(&models.RolePlayQuestChapter{}).Error; err != nil {
			return err
		}
		if err := tx.Where("template_id = ?", id).Delete(&models.RolePlayQuestArc{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.RolePlayQuestTemplate{}, id).Error
	})
}

func (s *Server) clearRolePlayQuestTemplates(ctx context.Context) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.RolePlayQuestRun{}).
			Where("template_id IS NOT NULL").
			Update("template_id", nil).Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.RolePlayQuestChapter{}).Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.RolePlayQuestArc{}).Error; err != nil {
			return err
		}
		return tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.RolePlayQuestTemplate{}).Error
	})
}

func (s *Server) nexusCoinData(ctx context.Context) (nexusCoinResponse, error) {
	if err := s.ensureDefaultNexusCoinPlans(ctx); err != nil {
		return nexusCoinResponse{}, err
	}
	var plans []models.NexusCoinPlan
	if err := s.db.WithContext(ctx).Order("position ASC, id ASC").Find(&plans).Error; err != nil {
		return nexusCoinResponse{}, err
	}
	stats := s.nexusCoinStats(ctx)
	return nexusCoinResponse{
		Stats:       stats,
		Estimations: nexusCoinEstimations(stats),
		Plans:       plans,
	}, nil
}

func (s *Server) ensureDefaultNexusCoinPlans(ctx context.Context) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.NexusCoinPlan{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	stats := s.nexusCoinStats(ctx)
	estimations := nexusCoinEstimations(stats)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for index, estimate := range estimations {
			plan := models.NexusCoinPlan{
				Slug:                   estimate.Slug,
				Position:               index + 1,
				Name:                   estimate.Name,
				Subtitle:               estimate.Subtitle,
				Description:            estimate.Description,
				Status:                 "active",
				TokenBudget:            estimate.TokenBudget,
				NexusCoins:             estimate.NexusCoins,
				BaseCostMicros:         estimate.BaseCostMicros,
				MarginPercent:          estimate.MarginPercent,
				PriceMicros:            estimate.PriceMicros,
				EstimatedCalls:         estimate.EstimatedCalls,
				EstimatedTokensPerCall: estimate.EstimatedTokensPerCall,
			}
			if err := tx.Create(&plan).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Server) saveNexusCoinPlan(ctx context.Context, id *uint, input nexusCoinPlanInput) (models.NexusCoinPlan, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Slug = strings.TrimSpace(input.Slug)
	input.Status = defaultValue(strings.TrimSpace(input.Status), "active")
	if input.Name == "" {
		return models.NexusCoinPlan{}, fmt.Errorf("le nom du plan est requis")
	}
	if input.Slug == "" {
		input.Slug = defaultSlug("", input.Name)
	}
	if input.TokenBudget <= 0 {
		return models.NexusCoinPlan{}, fmt.Errorf("le budget tokens doit etre positif")
	}
	if input.NexusCoins <= 0 {
		input.NexusCoins = input.TokenBudget / 1000
	}
	if input.MarginPercent < 0 {
		return models.NexusCoinPlan{}, fmt.Errorf("la marge ne peut pas etre negative")
	}
	if input.MarginPercent > 1000 {
		return models.NexusCoinPlan{}, fmt.Errorf("la marge est trop haute")
	}

	stats := s.nexusCoinStats(ctx)
	baseCostMicros := estimateNexusBaseCostMicros(input.TokenBudget, stats.AverageCostMicrosPerToken)
	priceMicros := applyMarginMicros(baseCostMicros, input.MarginPercent)
	estimatedCalls := int64(0)
	if stats.AverageTokensPerCall > 0 {
		estimatedCalls = input.TokenBudget / stats.AverageTokensPerCall
	}

	plan := models.NexusCoinPlan{}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if id != nil {
			if err := tx.Where("id = ?", *id).First(&plan).Error; err != nil {
				return err
			}
		}
		plan.Slug = input.Slug
		plan.Position = input.Position
		plan.Name = input.Name
		plan.Subtitle = strings.TrimSpace(input.Subtitle)
		plan.Description = strings.TrimSpace(input.Description)
		plan.Status = input.Status
		plan.TokenBudget = input.TokenBudget
		plan.NexusCoins = input.NexusCoins
		plan.BaseCostMicros = baseCostMicros
		plan.MarginPercent = input.MarginPercent
		plan.PriceMicros = priceMicros
		plan.EstimatedCalls = estimatedCalls
		plan.EstimatedTokensPerCall = stats.AverageTokensPerCall

		if id == nil {
			return tx.Create(&plan).Error
		}
		return tx.Save(&plan).Error
	})
	return plan, err
}

func (s *Server) nexusCoinStats(ctx context.Context) nexusCoinStats {
	var stats nexusCoinStats
	_ = s.db.WithContext(ctx).
		Model(&models.AIUsageRecord{}).
		Select(`
			COUNT(*) AS call_count,
			COALESCE(SUM(total_tokens), 0) AS total_tokens,
			COALESCE(SUM(estimated_cost_micros), 0) AS total_cost_micros
		`).
		Where("total_tokens > 0").
		Scan(&stats).Error

	if stats.CallCount > 0 {
		stats.AverageTokensPerCall = stats.TotalTokens / stats.CallCount
	}
	if stats.AverageTokensPerCall <= 0 {
		stats.AverageTokensPerCall = int64(envInt("NEXUS_COIN_FALLBACK_TOKENS_PER_CALL", 4000))
	}
	if stats.TotalTokens > 0 && stats.TotalCostMicros > 0 {
		stats.AverageCostMicrosPerToken = float64(stats.TotalCostMicros) / float64(stats.TotalTokens)
		stats.CostSource = "ai_usage_records"
	} else {
		fallbackUSDPer1M := envFloatDefault("NEXUS_COIN_FALLBACK_USD_PER_1M_TOKENS", 2.5)
		stats.AverageCostMicrosPerToken = fallbackUSDPer1M
		stats.CostSource = "fallback:NEXUS_COIN_FALLBACK_USD_PER_1M_TOKENS"
	}
	stats.MarginPercent = envInt("NEXUS_COIN_DEFAULT_MARGIN_PERCENT", 50)
	return stats
}

func nexusCoinEstimations(stats nexusCoinStats) []nexusCoinEstimate {
	templates := []struct {
		slug        string
		name        string
		subtitle    string
		description string
		tokens      int64
	}{
		{"starter", "Starter", "Premiers credits Nexus Coin", "Pack d'entree pour tester les fonctions IA du jeu.", 100_000},
		{"adventurer", "Aventurier", "Credit confortable pour jouer regulierement", "Pack equilibre pour plusieurs sessions Battle IA ou RolePlay IA.", 350_000},
		{"champion", "Champion", "Gros volume de credits IA", "Pack pense pour les joueurs actifs et les longues sessions.", 1_000_000},
		{"legend", "Legende", "Reserve maximale de Nexus Coin", "Pack haut volume pour les gros consommateurs de generation IA.", 3_000_000},
	}

	estimates := make([]nexusCoinEstimate, 0, len(templates))
	for _, item := range templates {
		baseCost := estimateNexusBaseCostMicros(item.tokens, stats.AverageCostMicrosPerToken)
		price := applyMarginMicros(baseCost, stats.MarginPercent)
		estimatedCalls := int64(0)
		if stats.AverageTokensPerCall > 0 {
			estimatedCalls = item.tokens / stats.AverageTokensPerCall
		}
		estimates = append(estimates, nexusCoinEstimate{
			Slug:                   item.slug,
			Name:                   item.name,
			Subtitle:               item.subtitle,
			Description:            item.description,
			TokenBudget:            item.tokens,
			NexusCoins:             item.tokens / 1000,
			BaseCostMicros:         baseCost,
			MarginPercent:          stats.MarginPercent,
			PriceMicros:            price,
			EstimatedCalls:         estimatedCalls,
			EstimatedTokensPerCall: stats.AverageTokensPerCall,
			CostSource:             stats.CostSource,
		})
	}
	return estimates
}

func estimateNexusBaseCostMicros(tokenBudget int64, averageCostMicrosPerToken float64) int64 {
	if tokenBudget <= 0 || averageCostMicrosPerToken <= 0 {
		return 0
	}
	return int64(math.Round(float64(tokenBudget) * averageCostMicrosPerToken))
}

func applyMarginMicros(baseCostMicros int64, marginPercent int) int64 {
	if baseCostMicros <= 0 {
		return 0
	}
	return int64(math.Round(float64(baseCostMicros) * (1 + float64(marginPercent)/100)))
}

func runtimeSnapshot() runtimeStatsData {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	return runtimeStatsData{
		StartedAt:      adminProcessStartedAt.Format(time.RFC3339),
		UptimeSeconds:  int64(time.Since(adminProcessStartedAt).Seconds()),
		GoVersion:      runtime.Version(),
		GOOS:           runtime.GOOS,
		GOARCH:         runtime.GOARCH,
		NumCPU:         runtime.NumCPU(),
		NumGoroutine:   runtime.NumGoroutine(),
		AllocBytes:     mem.Alloc,
		HeapAllocBytes: mem.HeapAlloc,
		SysBytes:       mem.Sys,
		NumGC:          mem.NumGC,
	}
}

func requestSnapshot() requestStatsData {
	total := atomic.LoadInt64(&adminRequestStats.totalRequests)
	totalLatency := atomic.LoadInt64(&adminRequestStats.totalLatencyNS)
	avg := 0.0
	if total > 0 {
		avg = float64(totalLatency) / float64(total) / float64(time.Millisecond)
	}
	return requestStatsData{
		TotalRequests:    total,
		ActiveRequests:   atomic.LoadInt64(&adminRequestStats.activeRequests),
		Status2xx:        atomic.LoadInt64(&adminRequestStats.status2xx),
		Status3xx:        atomic.LoadInt64(&adminRequestStats.status3xx),
		Status4xx:        atomic.LoadInt64(&adminRequestStats.status4xx),
		Status5xx:        atomic.LoadInt64(&adminRequestStats.status5xx),
		AverageLatencyMS: avg,
		MaxLatencyMS:     float64(atomic.LoadInt64(&adminRequestStats.maxLatencyNS)) / float64(time.Millisecond),
	}
}

func (s *Server) usageDashboardData(ctx context.Context) usageData {
	data := usageData{
		PricingHint: "Configure AI_PRICE_*_USD_PER_1M pour estimer les couts plateforme. A 0, seuls les tokens sont collectes.",
	}
	load := func(mode string) usageSummary {
		var summary usageSummary
		query := s.db.WithContext(ctx).Model(&models.AIUsageRecord{})
		if mode != "" {
			query = query.Where("session_mode = ?", mode)
		}
		_ = query.Select(`
			COUNT(*) AS call_count,
			COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) AS completion_tokens,
			COALESCE(SUM(total_tokens), 0) AS total_tokens,
			COALESCE(SUM(estimated_cost_micros), 0) AS estimated_cost_micros
		`).Scan(&summary).Error
		return summary
	}

	data.Total = load("")
	data.Battle = load(constants.ModeBattleIA)
	data.RolePlay = load(constants.ModeRolePlayIA)
	_ = s.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(12).
		Find(&data.Recent).Error
	return data
}

func (s *Server) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.ensureAdminOrRedirect(c) {
			c.Next()
		}
	}
}

func (s *Server) requireAdminAPI() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !s.hasAdminSession(c) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "admin authentication required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (s *Server) ensureAdminOrRedirect(c *gin.Context) bool {
	if !s.hasAdminSession(c) {
		c.Redirect(http.StatusSeeOther, "/admin/login")
		c.Abort()
		return false
	}
	return true
}

func (s *Server) hasAdminSession(c *gin.Context) bool {
	cookie, err := c.Cookie(adminCookieName)
	if err != nil {
		adminAuthLog("requireAdmin: missing/invalid cookie ip=%s path=%s err=%v", c.ClientIP(), c.Request.URL.Path, err)
		return false
	}

	ok, reason := verifyAdminSession(cookie)
	if !ok {
		adminAuthLog("requireAdmin: session rejected ip=%s path=%s reason=%s", c.ClientIP(), c.Request.URL.Path, reason)
		return false
	}
	adminAuthLog("requireAdmin: session accepted ip=%s path=%s", c.ClientIP(), c.Request.URL.Path)
	return true
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
Les IA sont seulement les participantes du debat: les sujets ne doivent pas tourner principalement autour de l'IA.
Genere surtout des questions generales, de culture generale, de vie quotidienne, sport, cuisine, famille, travail, ecole, cinema, musique, ville, voyage, morale ou societe.
Au moins 6 quetes doivent etre non technologiques.
Au moins 5 quetes doivent partir d'une situation de tous les jours ou de culture generale.
Maximum 1 quete peut parler d'IA, de robots ou de technologie numerique.
Au minimum 4 quetes doivent avoir un angle clairement humoristique, tout en restant debatables.
Evite les themes IA/robots/algorithmes sauf exception unique.
Format:
[{"title":"...","content":"question debat complete","level":"facile|moyen|difficile","theme":"...","point":10,"xp":25,"coin":5,"metadata":{"angle":"...","humour":true}}]`, count)
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
	quests := make([]generatedRolePlayQuest, 0, count)
	for len(quests) < count {
		batchCount := minInt(adminRolePlayGenerationBatchSize(), count-len(quests))
		batch, err := generateRolePlayQuestPayloadBatch(ctx, url, apiKey, model, batchCount)
		if err != nil {
			if len(quests) > 0 {
				return quests, nil
			}
			return nil, err
		}
		if len(batch) == 0 {
			if len(quests) > 0 {
				return quests, nil
			}
			return nil, fmt.Errorf("provider returned no roleplay quest")
		}
		quests = append(quests, batch...)
	}
	return quests, nil
}

func generateRolePlayQuestPayloadBatch(ctx context.Context, url string, apiKey string, model string, count int) ([]generatedRolePlayQuest, error) {
	prompt := fmt.Sprintf(`Genere exactement %d quetes de jeu de role.
Reponds uniquement en JSON valide, sans markdown.
Les quetes doivent avoir une longueur variable selon leur niveau.
Tu dois varier les niveaux dans le batch: environ 35%% facile, 40%% moyen, 25%% difficile quand le count le permet.
Regles de structure obligatoires:
- level "facile": 2 ou 3 arcs; chaque arc a 2 ou 3 chapitres; total 4 a 7 chapitres.
- level "moyen": 3 ou 4 arcs; chaque arc a 2 a 4 chapitres; total 7 a 12 chapitres.
- level "difficile": 4 a 6 arcs; chaque arc a 3 a 5 chapitres; total 12 a 24 chapitres.
Ne genere pas toujours la structure minimale. Dans un meme batch, alterne le nombre d'arcs et de chapitres.
Ajoute exactement un chapitre boss pour les quetes difficiles, et zero ou un chapitre boss pour les autres.
Les arcs et chapitres doivent etre clairement identifies et ordonnes dans le JSON.
Format:
[{"title":"...","summary":"resume court","prompt":"prompt global de la quete","theme":"fantasy|sf|horreur|steampunk|modern","level":"facile|moyen|difficile","xp":80,"coin":30,"metadata":{"ton":"..."},"arcs":[{"title":"Arc 1","summary":"...","objective":"...","prompt":"brief de l'arc","metadata":{"tone":"..."},"chapters":[{"title":"Chapitre 1","summary":"...","objective":"objectif jouable","introPrompt":"situation initiale du chapitre","successPrompt":"consequence en cas de reussite","failurePrompt":"consequence en cas d'echec","isBoss":false,"xp":20,"coin":8,"metadata":{"stakes":"..."}}]}]}]`, count)
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

func adminGeneratedRolePlayQuestInput(item generatedRolePlayQuest, slug string) service.RolePlayQuestInput {
	return service.RolePlayQuestInput{
		Slug:     slug,
		Title:    item.Title,
		Summary:  item.Summary,
		Prompt:   item.Prompt,
		Theme:    item.Theme,
		Level:    item.Level,
		Xp:       item.Xp,
		Coin:     item.Coin,
		Source:   "admin_ai",
		Status:   constants.QuestStatusPublished,
		Metadata: item.Meta,
		Arcs:     adminGeneratedRolePlayArcInputs(item.Arcs),
	}
}

func adminGeneratedRolePlayArcInputs(items []generatedRolePlayArc) []service.RolePlayQuestArcInput {
	arcs := make([]service.RolePlayQuestArcInput, 0, len(items))
	for index, item := range items {
		arcs = append(arcs, service.RolePlayQuestArcInput{
			Position:  index + 1,
			Title:     item.Title,
			Summary:   item.Summary,
			Objective: item.Objective,
			Prompt:    item.Prompt,
			Metadata:  item.Meta,
			Chapters:  adminGeneratedRolePlayChapterInputs(item.Chapters),
		})
	}
	return arcs
}

func adminGeneratedRolePlayChapterInputs(items []generatedRolePlayChapter) []service.RolePlayQuestChapterInput {
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

func hasAdminGeneratedRolePlayStructure(item generatedRolePlayQuest) bool {
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

func callAdminProvider(ctx context.Context, url string, apiKey string, model string, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, adminProviderTimeout())
	defer cancel()

	ai := provider.NewsProvider(apiKey, url, model)
	return ai.Chat(ctx, []provider.ProviderMessage{{Role: "system", Content: prompt}})
}

func adminProviderTimeout() time.Duration {
	seconds := envInt("ADMIN_AI_TIMEOUT_SECONDS", 480)
	return time.Duration(seconds) * time.Second
}

func adminRolePlayGenerationBatchSize() int {
	size := envInt("ADMIN_RP_GENERATION_BATCH_SIZE", 2)
	if size < 1 {
		return 1
	}
	if size > 5 {
		return 5
	}
	return size
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envFloatDefault(key string, fallback float64) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv(key)), 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func checkAdminPassword(password string) bool {
	if hash := os.Getenv("ADMIN_PASSWORD_BCRYPT"); hash != "" {
		adminAuthLog("checkAdminPassword: ADMIN_PASSWORD_BCRYPT detected (len=%d)", len(hash))
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
		if err == nil {
			adminAuthLog("checkAdminPassword: bcrypt validation success")
			return true
		}
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			adminAuthLog("checkAdminPassword: bcrypt hash present but password mismatch")
			return false
		}
		if errors.Is(err, bcrypt.ErrHashTooShort) {
			adminAuthLog("checkAdminPassword: bcrypt hash too short, trying ADMIN_PASSWORD fallback")
		} else {
			adminAuthLog("checkAdminPassword: bcrypt validation error (%v), trying ADMIN_PASSWORD fallback", err)
		}
	}
	if plain := os.Getenv("ADMIN_PASSWORD"); plain != "" {
		ok := subtleEqual(plain, password)
		if ok {
			adminAuthLog("checkAdminPassword: ADMIN_PASSWORD exact match success (plain_len=%d input_len=%d)", len(plain), len(password))
			return true
		}

		trimmedPlain := strings.TrimSpace(plain)
		trimmedInput := strings.TrimSpace(password)
		trimmedOK := subtleEqual(trimmedPlain, trimmedInput)
		adminAuthLog(
			"checkAdminPassword: ADMIN_PASSWORD mismatch exact=false trimmed_match=%t plain_len=%d input_len=%d trimmed_plain_len=%d trimmed_input_len=%d",
			trimmedOK,
			len(plain),
			len(password),
			len(trimmedPlain),
			len(trimmedInput),
		)
		return trimmedOK
	}
	if env("GIN_MODE", "debug") != "release" {
		ok := subtleEqual("admin", password)
		adminAuthLog("checkAdminPassword: debug fallback used result=%t", ok)
		return ok
	}
	adminAuthLog("checkAdminPassword: no validation method available")
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

func verifyAdminSession(token string) (bool, string) {
	parts := strings.Split(token, "|")
	if len(parts) != 4 {
		return false, fmt.Sprintf("invalid token format: expected 4 parts got %d", len(parts))
	}
	payload := strings.Join(parts[:3], "|")
	if !subtleEqual(sign(payload), parts[3]) {
		return false, "signature mismatch"
	}
	if parts[0] != adminUsername() {
		return false, fmt.Sprintf("username mismatch in token: token=%q expected=%q", parts[0], adminUsername())
	}
	expires, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return false, fmt.Sprintf("invalid expiry: %v", err)
	}
	if time.Now().Unix() >= expires {
		return false, fmt.Sprintf("session expired at unix=%d", expires)
	}
	return true, "ok"
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

func usdMicros(value int64) string {
	return fmt.Sprintf("$%.6f", float64(value)/1_000_000)
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

func adminAuthLog(format string, args ...any) {
	log.Printf("[admin-auth] "+format, args...)
}
