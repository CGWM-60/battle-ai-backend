package router

import (
	"cgwm/battle/internal/admin"
	cgwm "cgwm/battle/internal/cgwm"
	"cgwm/battle/internal/features"
	"cgwm/battle/internal/models"
	nexuscache "cgwm/battle/internal/nexus_game/cache"
	nexusdev "cgwm/battle/internal/nexus_game/dev"
	nexusroutes "cgwm/battle/internal/nexus_game/routes"
	translations "cgwm/battle/internal/nexus_game/translations"
	nexustribunal "cgwm/battle/internal/nexus_tribunal"
	"cgwm/battle/internal/repository"
	"cgwm/battle/internal/service"
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
	heroImageDir := service.HeroImagePublicDir()
	_ = os.MkdirAll(heroImageDir, 0o755)
	router.Static("/assets/heroes", heroImageDir)
	rolePlaySceneDir := service.RolePlayScenePublicDir()
	_ = os.MkdirAll(rolePlaySceneDir, 0o755)
	router.Static("/uploads/roleplay/quests", rolePlaySceneDir)

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	if features.NexusGameEnabled() {
		// Register Nexus game asset files early. The Next.js admin UI itself is served later by
		// admin.Register through NoRoute, not by a Gin /admin/*filepath wildcard, so /admin/login
		// and other explicit admin routes do not conflict at startup.
		nexusroutes.RegisterAdminStatic(router)
	}

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
	publicFeed.GET("/roleplay/hero-images/:id/image", getPublicRolePlayHeroImage(database))

	authenticated := api.Group("")
	authenticated.Use(jwtAuth())
	authenticated.GET("/live/:channel/stream", streamLiveChannel(database))

	private := authenticated.Group("")
	private.Use(queue.Middleware())

	private.GET("/me", me(database))
	private.PUT("/me", updateMe(database))
	private.PATCH("/me", updateMe(database))
	private.POST("/ai/providers/test", testAIProvider())
	private.POST("/ai/providers/generate", generateAIProviderText(database))
	registerBillingRoutes(private, database)
	nexustribunal.RegisterRoutes(router, database, jwtAuth(), adminAPIAuth())
	if features.NexusGameEnabled() {
		translations.RegisterRoutes(router, database)
		nexusdev.RegisterRoutes(router)
		nexuscache.RegisterRoutes(router)
		nexusroutes.RegisterRoutes(router, database)
	} else {
		registerDeprecatedNexusGameRoutes(router)
	}

	// Register full CGWM (ANIMA Cloud Game World Memory - Park + Social + realtime + schedulers)
	cgwm.RegisterCGWMRoutes(router, database)

	adminAPI := private.Group("")
	adminAPI.Use(adminAPIAuth())
	adminAPI.PATCH("/users/:id/progression", updateUserProgression(database))
	// Monde IA desactive: routes admin API legacy non enregistrees.
	// registerAdminWorldGameRoutes(adminAPI, database)

	strictAdminAPI := router.Group("/api/admin")
	strictAdminAPI.Use(jwtAuth(), adminAPIAuth(), queue.Middleware())
	translations.RegisterAdminRoutes(strictAdminAPI, database)
	// Monde IA desactive: routes strictes /api/admin/game non enregistrees.
	// registerStrictAdminWorldGameRoutes(strictAdminAPI, database)

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
	private.GET("/roleplay/characters", listRolePlayCharacters(database))
	private.POST("/roleplay/characters", createRolePlayCharacter(database))
	private.POST("/roleplay/characters/validate", validateRolePlayCharacter(database))
	private.POST("/roleplay/characters/generate", generateRolePlayCharacter(database))
	private.POST("/roleplay/characters/local-prompt", rolePlayCharacterLocalPrompt(database))
	private.GET("/roleplay/characters/:id", getRolePlayCharacter(database))
	private.PUT("/roleplay/characters/:id", updateRolePlayCharacter(database))
	private.PATCH("/roleplay/characters/:id", updateRolePlayCharacter(database))
	private.DELETE("/roleplay/characters/:id", deleteRolePlayCharacter(database))
	private.GET("/roleplay/hero-images", listRolePlayHeroImages(database))
	private.POST("/roleplay/sessions", createRolePlaySession(database))
	private.GET("/roleplay/sessions", listRolePlaySessions(database))
	private.GET("/roleplay/sessions/:id", getRolePlaySession(database))
	private.GET("/roleplay/sessions/:id/turns", getRolePlaySessionTurns(database))
	private.GET("/roleplay/sessions/:id/usage", getRolePlayUsage(database))
	private.POST("/roleplay/sessions/:id/resume", resumeRolePlaySession(database))
	private.POST("/roleplay/sessions/:id/action", appendRolePlayAction(database))
	private.POST("/roleplay/sessions/:id/end", endRolePlaySession(database))

	private.GET("/rpg/session/:id/map", getRPGMap(database))
	private.POST("/rpg/session/:id/map/generate", generateRPGMap(database))
	private.POST("/rpg/session/:id/map/enter", enterRPGMap(database))
	private.POST("/rpg/session/:id/map/move", moveRPGMap(database))
	private.POST("/rpg/session/:id/map/action", actionRPGMap(database))
	private.POST("/rpg/session/:id/map/roll", rollRPGMap(database))
	private.POST("/rpg/session/:id/map/resolve-action", resolveRPGMapAction(database))
	private.POST("/rpg/session/:id/map/interact-npc", interactRPGMapNPC(database))
	private.POST("/rpg/session/:id/map/loot", lootRPGMap(database))
	private.POST("/rpg/session/:id/map/trap/disarm", disarmRPGMapTrap(database))
	private.POST("/rpg/session/:id/map/combat/action", combatActionRPGMap(database))
	private.POST("/rpg/session/:id/map/combat/flee", fleeRPGMapCombat(database))

	private.GET("/coop/parties/:code/map", getCoopRPGMap(database))
	private.POST("/coop/parties/:code/map/enter", enterCoopRPGMap(database))
	private.POST("/coop/parties/:code/map/move", coopRPGMapMove(database))
	private.POST("/coop/parties/:code/map/vote", coopRPGMapVote(database))
	private.POST("/coop/parties/:code/map/separate", coopRPGMapSeparate(database))
	private.POST("/coop/parties/:code/map/regroup", coopRPGMapRegroup(database))
	private.POST("/coop/parties/:code/map/lever", coopRPGMapLever(database))
	private.POST("/coop/parties/:code/map/action", coopRPGMapAction(database))
	private.POST("/coop/parties/:code/map/roll", coopRPGMapRoll(database))
	private.POST("/coop/parties/:code/map/resolve-action", coopRPGMapResolveAction(database))
	private.POST("/coop/parties/:code/map/combat/action", coopRPGMapCombatAction(database))

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
	// Monde IA desactive: routes joueur /api/v1/world, city, guild, research, etc. non enregistrees.
	// registerWorldGameRoutes(private, database)

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

func registerDeprecatedNexusGameRoutes(router *gin.Engine) {
	handler := func(c *gin.Context) {
		c.JSON(http.StatusGone, features.NexusGameDisabledPayload())
	}
	adminHandler := func(c *gin.Context) {
		c.Data(http.StatusGone, "text/html; charset=utf-8", []byte(`<!doctype html><html lang="fr"><head><meta charset="utf-8"><title>Nexus Games desactive</title><style>body{font-family:Inter,Arial,sans-serif;background:#0f172a;color:#e2e8f0;margin:0;display:grid;min-height:100vh;place-items:center}main{max-width:680px;padding:32px}a{color:#67e8f9}</style></head><body><main><h1>Nexus Games / MMO desactive</h1><p>Ce module est deprecie et masque dans l'administration. Les modules conserves sont Battle IA, Quetes RP, Coop et Live.</p><p><a href="/admin/">Retour admin</a></p></main></body></html>`))
	}

	router.Any("/api/nexus-game", handler)
	router.Any("/api/nexus-game/*path", handler)
	router.Any("/api/v1/prerequisites/validate", handler)
	router.Any("/api/v1/buildings", handler)
	router.Any("/api/v1/buildings/*path", handler)
	router.Any("/api/v1/construction", handler)
	router.Any("/api/v1/construction/*path", handler)
	router.Any("/api/v1/units/catalog", handler)
	router.Any("/api/v1/units/:key", handler)
	router.Any("/api/v1/research/catalog", handler)
	router.Any("/api/v1/research/:key", handler)
	router.Any("/api/v1/assets/buildings/manifest", handler)
	router.Any("/api/v1/assets/buildings/updates", handler)

	router.GET("/admin/nexus", adminHandler)
	router.GET("/admin/nexus-coin", adminHandler)
	router.GET("/admin/nexus-coin/*path", adminHandler)
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
