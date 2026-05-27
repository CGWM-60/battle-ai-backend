package router

import (
	"cgwm/battle/internal/admin"
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/repository"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type signupRequest struct {
	Email        string `form:"email" json:"email"`
	Password     string `form:"password" json:"password"`
	Pseudo       string `form:"pseudo" json:"pseudo"`
	BirthdayDate string `form:"birthdayDate" json:"birthdayDate"`
	Avatar       string `form:"avatar" json:"avatar"`
}

type loginRequest struct {
	User     string `form:"user" json:"user"`
	Email    string `form:"email" json:"email"`
	Password string `form:"password" json:"password"`
}

func RouterApp(database *gorm.DB) {
	router := gin.Default()
	router.Use(securityHeaders())
	router.Use(requestBodyLimit(maxBodyBytes()))
	router.Use(admin.RequestMetricsMiddleware())
	_ = os.MkdirAll(getEnv("BUILDING_ASSET_PUBLIC_DIR", "storage/assets/buildings"), 0o755)
	router.Static("/assets/buildings", getEnv("BUILDING_ASSET_PUBLIC_DIR", "storage/assets/buildings"))

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
	admin.Register(router, database)

	api := router.Group("/api/v1")
	api.POST("/auth/login", login(database))
	api.POST("/auth/signup", signup(database))
	api.POST("/subscrit", signup(database))
	api.GET("/ai/providers", listAIProviders())

	queue := newRequestQueue(maxConcurrentRequests(), queueTimeout())
	publicFeed := api.Group("/public")
	publicFeed.Use(queue.Middleware())
	publicFeed.GET("/battles", listPublicBattles(database))
	publicFeed.GET("/battles/:id", getPublicBattle(database))
	publicFeed.GET("/battles/:id/turns", getPublicBattleTurns(database))
	publicFeed.GET("/arenas", listPublicArenas(database))
	publicFeed.GET("/arenas/:code", getPublicArena(database))
	publicFeed.GET("/live/sessions", listPublicLiveSessions(database))
	publicFeed.GET("/live/sessions/:id", getPublicLiveSession(database))
	publicFeed.GET("/live/sessions/:id/events", getPublicLiveSessionEvents(database))
	publicFeed.GET("/live/:channel/history", getPublicLiveHistory(database))
	publicFeed.GET("/live/:channel/stream", streamPublicLiveChannel(database))

	authenticated := api.Group("")
	authenticated.Use(jwtAuth())
	authenticated.GET("/live/:channel/stream", streamLiveChannel(database))

	private := authenticated.Group("")
	private.Use(queue.Middleware())

	private.GET("/me", me(database))
	private.PUT("/me", updateMe(database))
	private.PATCH("/me", updateMe(database))
	private.POST("/ai/providers/test", testAIProvider())
	private.POST("/ai/providers/generate", generateAIProviderText())

	adminAPI := private.Group("")
	adminAPI.Use(adminAPIAuth())
	adminAPI.PATCH("/users/:id/progression", updateUserProgression(database))
	registerAdminWorldGameRoutes(adminAPI, database)

	strictAdminAPI := router.Group("/api/admin")
	strictAdminAPI.Use(jwtAuth(), adminAPIAuth(), queue.Middleware())
	registerStrictAdminWorldGameRoutes(strictAdminAPI, database)

	private.POST("/battle", startBattle(database))
	private.POST("/battles", startBattle(database))
	private.GET("/battles", listBattles(database))
	private.GET("/battles/:id", getBattle(database))
	private.GET("/battles/:id/turns", getBattleTurns(database))
	private.GET("/battles/:id/usage", getBattleUsage(database))
	private.POST("/battles/:id/next-round", nextBattleRound(database))
	private.POST("/battles/:id/judge", judgeBattle(database))
	private.POST("/battles/:id/resume", resumeBattle(database))
	private.POST("/battles/:id/cancel", cancelBattle(database))

	private.GET("/battle-quests", listBattleQuests(database))
	private.GET("/battle-quests/random", randomBattleQuest(database))
	private.GET("/battle-quests/:id", getBattleQuest(database))
	private.POST("/battle-quests", createBattleQuest(database))
	private.PUT("/battle-quests/:id", updateBattleQuest(database))
	private.DELETE("/battle-quests/:id", deleteBattleQuest(database))
	private.POST("/battle-quests/:id/publish", publishBattleQuest(database))
	private.POST("/battle-quests/:id/archive", archiveBattleQuest(database))

	private.GET("/ia-profiles", listIAProfiles(database))
	private.POST("/ia-profiles", createIAProfile(database))
	private.GET("/ia-profiles/:id", getIAProfile(database))
	private.PUT("/ia-profiles/:id", updateIAProfile(database))
	private.DELETE("/ia-profiles/:id", deleteIAProfile(database))

	private.GET("/roleplay/quests", listRolePlayQuests(database))
	private.GET("/roleplay/quests/:id", getRolePlayQuest(database))
	private.POST("/roleplay/quests", createRolePlayQuest(database))
	private.PUT("/roleplay/quests/:id", updateRolePlayQuest(database))
	private.DELETE("/roleplay/quests/:id", deleteRolePlayQuest(database))
	private.POST("/roleplay/quests/:id/start", startRolePlayQuest(database))
	private.POST("/roleplay/sessions", createRolePlaySession(database))
	private.GET("/roleplay/sessions", listRolePlaySessions(database))
	private.GET("/roleplay/sessions/:id", getRolePlaySession(database))
	private.GET("/roleplay/sessions/:id/turns", getRolePlaySessionTurns(database))
	private.GET("/roleplay/sessions/:id/usage", getRolePlayUsage(database))
	private.POST("/roleplay/sessions/:id/resume", resumeRolePlaySession(database))
	private.POST("/roleplay/sessions/:id/action", appendRolePlayAction(database))
	private.POST("/roleplay/sessions/:id/end", endRolePlaySession(database))

	private.GET("/arenas", listArenas(database))
	private.POST("/arenas", createArena(database))
	private.GET("/arenas/:code", getArena(database))
	private.POST("/arenas/:code/join", joinArena(database))
	private.POST("/arenas/:code/leave", leaveArena(database))
	private.GET("/arenas/:code/members", listArenaMembers(database))
	private.POST("/coop/parties", createCoopParty(database))
	private.GET("/coop/parties", listCoopParties(database))
	private.GET("/coop/parties/:code", getCoopParty(database))
	private.POST("/coop/parties/:code/join", joinCoopParty(database))
	private.POST("/coop/parties/:code/leave", leaveCoopParty(database))
	private.POST("/coop/parties/:code/ready", readyCoopParty(database))
	private.GET("/coop/parties/:code/members", listCoopMembers(database))
	private.PUT("/coop/parties/:code/state", updateCoopState(database))
	private.GET("/live/sessions", listLiveSessions(database))
	private.POST("/live/sessions", createLiveSession(database))
	private.GET("/live/sessions/:id", getLiveSession(database))
	private.GET("/live/sessions/:id/events", getLiveSessionEvents(database))
	private.GET("/live/:channel/history", getLiveHistory(database))
	private.POST("/live/sessions/:id/end", endLiveSession(database))
	registerWorldGameRoutes(private, database)

	host := getEnv("APP_HOST", "0.0.0.0")
	port := getEnv("APP_PORT", "8080")

	router.Run(host + ":" + port)
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}

func signup(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req signupRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signup payload"})
			return
		}

		email := req.Email
		password := req.Password
		pseudo := req.Pseudo
		birthdayDate := req.BirthdayDate
		avatar := req.Avatar

		if email == "" || password == "" || pseudo == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email, password and pseudo are required"})
			return
		}

		users := repository.NewUserRepository(database)
		existingUser, err := users.GetByEmail(c.Request.Context(), email)
		if err == nil && existingUser != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
			return
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot check user"})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot hash password"})
			return
		}

		newUser := &models.Users{
			Email:        email,
			Password:     string(hashedPassword),
			Pseudo:       pseudo,
			BirthdayDate: birthdayDate,
			Avatar:       avatar,
			Xp:           0,
			Coin:         0,
		}

		err = users.Create(c.Request.Context(), newUser)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create user"})
			return
		}

		token, err := makeJWT(newUser)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create token"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"token":      token,
			"expires_in": int64((24 * time.Hour).Seconds()),
			"user":       userResponse(newUser),
		})
	}
}

func login(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := bindPayload(c, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid login payload"})
			return
		}

		email := req.User
		if email == "" {
			email = req.Email
		}
		password := req.Password

		users := repository.NewUserRepository(database)
		userVerif, err := users.GetByEmail(c.Request.Context(), email)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
			return
		}

		verifPassword := bcrypt.CompareHashAndPassword([]byte(userVerif.Password), []byte(password))
		if verifPassword != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}

		token, err := makeJWT(userVerif)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot create token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":      token,
			"expires_in": int64((24 * time.Hour).Seconds()),
			"user":       userResponse(userVerif),
		})
	}
}

func makeJWT(user *models.Users) (string, error) {
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	claims := jwt.MapClaims{
		"sub":    strconv.FormatUint(uint64(user.Id), 10),
		"email":  user.Email,
		"pseudo": user.Pseudo,
		"iat":    now.Unix(),
		"exp":    expiresAt.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(getEnv("JWT_SECRET", "dev-secret-change-me")))
}

func userResponse(user *models.Users) gin.H {
	return gin.H{
		"id":     user.Id,
		"email":  user.Email,
		"pseudo": user.Pseudo,
		"avatar": user.Avatar,
		"xp":     user.Xp,
		"coin":   user.Coin,
	}
}
