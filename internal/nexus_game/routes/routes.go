package routes

import (
	"cgwm/battle/internal/nexus_game/cache"
	"cgwm/battle/internal/nexus_game/handlers"
	"cgwm/battle/internal/nexus_game/models"
	"cgwm/battle/internal/nexus_game/seeds"
	"cgwm/battle/internal/nexus_game/services"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func RegisterRoutes(router *gin.Engine, database *gorm.DB) {
	redis := cache.NewRedisServiceFromEnv()
	health := handlers.NewHealthHandler(database, redis)
	bootstrap := handlers.NewBootstrapHandler(database)
	avatar := handlers.NewAvatarHandler(database)
	factionH := handlers.NewFactionHandler(database, redis)
	companionH := handlers.NewIACompanionHandler(database)
	profileH := handlers.NewProfileHandler(database, redis)
	worldH := handlers.NewWorldHandler(database, redis)

	// Auto migrate models (inside nexus_game only)
	if database != nil {
		database.AutoMigrate(&models.Avatar{}, &models.Faction{}, &models.IACompanion{}, &models.ProfileGamer{}, &models.World{}, &models.Continent{}, &models.Prompt{}, &models.AIOutput{}, &models.MmoIAAgent{}, &models.DailyPlan{},
			&models.BuildingDefinition{}, &models.PlayerBuilding{},
			&models.UnitDefinition{}, &models.PlayerUnit{},
			&models.ResearchDefinition{}, &models.PlayerResearch{})

		// Seed initial content for dev (full reference v2.0: buildings, units, research).
		// In prod: use admin CRUD + asset upload for the complete catalogs.
		contentSvc := services.NewContentService(database, "./content/assets")
		_ = seeds.SeedInitialBuildings(database, contentSvc)
		_ = seeds.SeedInitialUnits(database, contentSvc)
		_ = seeds.SeedInitialResearch(database, contentSvc)
	}

	// Ensure persistent asset directories exist on startup (prevents loss on recreate if volume is attached)
	assetsBaseDir := getEnv("NEXUS_ASSETS_BASE_DIR", "/nexus_game/assets")
	os.MkdirAll(assetsBaseDir, 0755)
	os.MkdirAll(assetsBaseDir+"/avatar", 0755)
	os.MkdirAll(assetsBaseDir+"/faction", 0755)
	os.MkdirAll(assetsBaseDir+"/companion", 0755)

	// Serve persistent assets - use BASE_DIR and BASE_URL so one volume mount covers avatar, faction, companion.
	// In Dokploy: configure Persistent Storage with Container Path = value of NEXUS_ASSETS_BASE_DIR (default /nexus_game/assets)
	assetsBaseURL := getEnv("NEXUS_ASSETS_BASE_URL", "/nexus_game/assets")
	router.Static(assetsBaseURL, assetsBaseDir)

	group := router.Group("/api/nexus-game")
	group.GET("/health", health.Health)
	group.GET("/debug/status", health.DebugStatus)

	// Etape 1 - Bootstrap endpoint (chargement assets, quêtes, infos joueur, etc.)
	// Pour l'instant : simulation avec délai de 10s + TODO pour l'implémentation réelle.
	group.GET("/bootstrap", bootstrap.Load)

	// Etape 2 - Avatar management
	// POST multipart: name + image (converted to webp on server)
	group.POST("/assets/avatar", avatar.Upload)
	group.GET("/assets/avatars", avatar.List)
	group.PUT("/assets/avatars/:id", avatar.Update)
	group.DELETE("/assets/avatars/:id", avatar.Delete)

	// Factions (same principle as avatars)
	group.GET("/factions", factionH.List)
	group.POST("/factions", factionH.Create)
	group.PUT("/factions/:id", factionH.Update)
	group.DELETE("/factions/:id", factionH.Delete)

	// IA Companions (player's AI agents/companions)
	group.GET("/ia-companions", companionH.List)
	group.POST("/ia-companions", companionH.Create)
	group.PUT("/ia-companions/:id", companionH.Update)
	group.DELETE("/ia-companions/:id", companionH.Delete)

	// ProfileGamer save routes (prepare all save endpoints).
	// GET /profile?user_id=XX  -> {exists, profile}  (profile may be empty/zero IDs)
	// POST /profile body {user_id, avatar_id, faction_id, ia_companion_id, pseudo, city_name}
	// Server truth. Flutter calls after creation or on load check.
	group.GET("/profile", profileH.GetProfile)
	group.POST("/profile", profileH.SaveProfile)

	// IA Agents & Companions for Nexus (MmoCreationAgentIAScreen)
	// Multiple agents per profile, only 1 companion.
	// Avatar support for agents. Companion linked to profile creation choice.
	group.POST("/profile/ia-agents", profileH.SaveIAAgent)
	group.GET("/profile/:id/ia-agents", profileH.ListIAAgents)

	// Daily Plan (Player AI Daily Plan - full flow).
	// 1. GET /context -> safe context (city stats, rules, available actions) sent to player's AI (provider / local / governor agent).
	// 2. Client sends AI output (recommendations) via POST /recommendations (or save-generated).
	// 3. Player reviews in the city dashboard card.
	// 4. POST /apply -> server applies selected items (effects on ProfileGamer + queues). Uses the validate/resolve philosophy internally.
	// The card and system are designed to stay open: new recommendation types and impacts can be added over time.
	group.GET("/profile/:id/daily-plan/context", profileH.GetDailyPlanContext)
	group.GET("/profile/:id/daily-plan", profileH.GetDailyPlan)
	group.POST("/profile/:id/daily-plan/recommendations", profileH.SaveDailyPlanRecommendations)
	group.POST("/profile/:id/daily-plan/apply", profileH.ApplyDailyPlan)

	// Content system (Buildings first, extensible to Units/Research per NEXUS GAME CONTENT REFERENCE v2.0).
	// Admin CRUD + asset upload (images served by this server after upload).
	// Player construction endpoints (queues, completion).
	// Each major item (buildings/units/research) will have its table + CRUD here.
	contentH := handlers.NewContentHandler(services.NewContentService(database, "./content/assets"))

	// Admin / catalog
	group.GET("/admin/content/buildings", contentH.ListBuildings)
	group.GET("/admin/content/buildings/:contentId", contentH.GetBuilding)
	group.POST("/admin/content/buildings", contentH.CreateOrUpdateBuilding)
	group.PUT("/admin/content/buildings/:contentId", contentH.CreateOrUpdateBuilding)
	group.DELETE("/admin/content/buildings/:contentId", contentH.DeleteBuilding)
	group.POST("/admin/content/upload-asset", contentH.UploadAsset) // multipart: file + contentId + domain + tier

	// Player constructions (used by Flutter mmo construction flows)
	// Use :id to match existing /profile/:id/* routes and avoid Gin wildcard conflict
	group.GET("/profile/:id/buildings", contentH.ListPlayerBuildings)
	group.POST("/profile/:id/construction/start", contentH.StartConstruction)
	group.POST("/profile/:id/construction/complete-ready", contentH.CompleteReadyConstructions)

	// Serve uploaded content assets (images for buildings/units/research) from server disk.
	// After upload via /admin/content/upload-asset, images are at /nexus-assets/content/buildings/xxx_tier1.jpg etc.
	router.Static("/nexus-assets", "./content/assets")

	// Simple admin "pages" (HTML tables + basic CRUD forms) for each major item in backend.
	// Accessible in browser for dev/admin: /admin/content/buildings/page etc.
	// Full table + create/edit/delete + asset upload.
	group.GET("/admin/content/buildings/page", contentH.AdminBuildingsPage)
	group.GET("/admin/content/units/page", contentH.AdminUnitsPage)
	group.GET("/admin/content/research/page", contentH.AdminResearchPage)

	// JSON CRUD for units and research (copy of buildings - extend service impl + seed from reference)
	group.GET("/admin/content/units", contentH.ListUnits)
	group.POST("/admin/content/units", contentH.CreateOrUpdateUnit)
	group.PUT("/admin/content/units/:contentId", contentH.CreateOrUpdateUnit)
	group.DELETE("/admin/content/units/:contentId", contentH.DeleteUnit) // add DeleteUnit to handler if needed
	group.GET("/admin/content/research", contentH.ListResearch)
	group.POST("/admin/content/research", contentH.CreateOrUpdateResearch)
	group.PUT("/admin/content/research/:contentId", contentH.CreateOrUpdateResearch)
	group.DELETE("/admin/content/research/:contentId", contentH.DeleteResearch)

	// World management (gestion des worlds) - card entry via admin or API.
	// GET /worlds -> list worlds with continents and capacities (from Redis)
	// POST /worlds -> create new world (auto 5 continents)
	// GET /worlds/:id -> detail
	// POST /factions/assign-continent (internal on faction create)
	// POST /profiles/assign-continent (internal on profile create)
	// Uses Redis heavily for capacities, locks, player counts, faction assignments.
	group.GET("/world-players", worldH.ListWorldPlayers)
	group.GET("/worlds", worldH.ListWorlds)
	group.POST("/worlds", worldH.CreateWorld)
	group.POST("/worlds/repair-player-assignments", worldH.RepairPlayerAssignments)
	group.GET("/worlds/:id", worldH.GetWorld)
	group.GET("/worlds/:id/players", worldH.ListPlayersByWorld)
	group.GET("/continents", worldH.ListContinents)
	group.POST("/worlds/:id/generate-event", worldH.GenerateWorldEvent) // IA serveur trigger for gestion des world
	group.POST("/worlds/:id/trigger-tick", worldH.TriggerWorldTick)     // Integration point for World Tick with IA

	// Prompt CRUD for IA serveur (modifiable in world management "gestion des world").
	// GET /prompts?domain=... -> list versioned optimized prompts
	// POST /prompts -> create new version
	// PUT /prompts/:id -> update (evolve prompt)
	group.GET("/prompts", worldH.ListPrompts)
	group.POST("/prompts", worldH.CreatePrompt)
	group.PUT("/prompts/:id", worldH.UpdatePrompt)

	// AI outputs history (persisted in Redis for cross-sessions)
	group.GET("/ai-outputs", worldH.ListAIOutputs)

	// Flexible server AI generation for admin (quests, events, lore, tribunal proposals... using manually managed prompts).
	// POST /ai/generate {world_id, feature, prompt_id?, prompt_version?, extra?}
	// Powers the rich generation console in admin with visible progress + success/error.
	group.POST("/ai/generate", worldH.RunAIGeneration)
}
