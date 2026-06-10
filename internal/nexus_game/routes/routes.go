package routes

import (
	"cgwm/battle/internal/nexus_game/cache"
	"cgwm/battle/internal/nexus_game/handlers"
	"cgwm/battle/internal/nexus_game/models"
	"cgwm/battle/internal/nexus_game/seeds"
	serverairoutes "cgwm/battle/internal/nexus_game/server_ai/routes"
	"cgwm/battle/internal/nexus_game/services"
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func nexusAssetsBaseDir() string {
	return getEnv("NEXUS_ASSETS_BASE_DIR", "/nexus_game/assets")
}

func nexusAssetsBaseURL() string {
	return getEnv("NEXUS_ASSETS_BASE_URL", "/nexus_game/assets")
}

func nexusContentAssetsDir() string {
	return filepath.Join(nexusAssetsBaseDir(), "content")
}

// RegisterAdminStatic registers Nexus asset files only.
// Do not mount /admin with StaticFS here: Gin's /admin/*filepath wildcard conflicts with
// explicit admin routes such as /admin/login. The Next.js admin UI is served by
// internal/admin via adminNoRoute + serveAdminUI, which avoids wildcard route conflicts.
func RegisterAdminStatic(router *gin.Engine) {
	// Backward-compatible public alias for all persistent Nexus assets.
	router.Static("/nexus-assets", nexusAssetsBaseDir())
}

func copyLegacyContentAssetsToVolume(dstRoot string) {
	legacyRoot := "./content/assets"
	if filepath.Clean(legacyRoot) == filepath.Clean(dstRoot) {
		return
	}
	info, err := os.Stat(legacyRoot)
	if err != nil || !info.IsDir() {
		return
	}
	_ = filepath.Walk(legacyRoot, func(src string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		rel, err := filepath.Rel(legacyRoot, src)
		if err != nil || rel == "." {
			return nil
		}
		dst := filepath.Join(dstRoot, rel)
		if info.IsDir() {
			_ = os.MkdirAll(dst, 0755)
			return nil
		}
		if _, err := os.Stat(dst); err == nil {
			return nil
		}
		in, err := os.Open(src)
		if err != nil {
			return nil
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return nil
		}
		out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return nil
		}
		defer out.Close()
		_, _ = io.Copy(out, in)
		return nil
	})
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
	resourceH := handlers.NewResourceHandler(database)

	// Auto migrate models (inside nexus_game only)
	if database != nil {
		database.AutoMigrate(&models.Avatar{}, &models.Faction{}, &models.IACompanion{}, &models.ProfileGamer{}, &models.World{}, &models.Continent{}, &models.Prompt{}, &models.AIOutput{}, &models.MmoIAAgent{}, &models.DailyPlan{},
			&models.BuildingDefinition{}, &models.PlayerBuilding{},
			&models.UnitDefinition{}, &models.PlayerUnit{},
			&models.ResearchDefinition{}, &models.PlayerResearch{},
			&models.ResourceCatalog{}, &models.PlayerResource{}, &models.PlayerCityStats{},
			&models.ResourceTransaction{}, &models.DailyGrantClaim{}, &models.DailyGrantConfig{},
			&models.InitialAllocationLog{})
		_ = serverairoutes.AutoMigrate(database)

		// Seed initial content for dev (full reference v2.0: buildings, units, research).
		// In prod: use admin CRUD + asset upload for the complete catalogs.
		contentSvc := services.NewContentService(database, nexusContentAssetsDir())
		_ = seeds.SeedInitialBuildings(database, contentSvc)
		_ = seeds.SeedInitialUnits(database, contentSvc)
		_ = seeds.SeedInitialResearch(database, contentSvc)
		_ = services.NewResourceService(database).SeedDefaults(context.Background())
		_ = serverairoutes.SeedDefaults(database)
		// Full catalogue from reference v2.0 seeded for buildings (20), units (15), research (11 branches x7). Use admin to add/update images and data.
	}

	// Ensure persistent asset directories exist on startup (prevents loss on recreate if volume is attached)
	assetsBaseDir := nexusAssetsBaseDir()
	os.MkdirAll(assetsBaseDir, 0755)
	os.MkdirAll(assetsBaseDir+"/avatar", 0755)
	os.MkdirAll(assetsBaseDir+"/faction", 0755)
	os.MkdirAll(assetsBaseDir+"/companion", 0755)
	os.MkdirAll(assetsBaseDir+"/content/buildings", 0755)
	os.MkdirAll(assetsBaseDir+"/content/units", 0755)
	os.MkdirAll(assetsBaseDir+"/content/research", 0755)
	copyLegacyContentAssetsToVolume(nexusContentAssetsDir())

	// Serve persistent assets - use BASE_DIR and BASE_URL so one volume mount covers avatar, faction, companion.
	// In Dokploy: configure Persistent Storage with Container Path = value of NEXUS_ASSETS_BASE_DIR (default /nexus_game/assets)
	assetsBaseURL := nexusAssetsBaseURL()
	if assetsBaseURL != "/nexus-assets" {
		router.Static(assetsBaseURL, assetsBaseDir)
	}

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

	// Resources and daily grants. profile_gamer_id query param identifies the player profile.
	group.GET("/resources", resourceH.Resources)
	group.GET("/resources/catalog", resourceH.Catalog)
	group.GET("/resources/transactions", resourceH.Transactions)
	group.GET("/city/stats", resourceH.CityStats)
	group.GET("/daily-grant/status", resourceH.DailyGrantStatus)
	group.POST("/daily-grant/claim", resourceH.DailyGrantClaim)
	group.GET("/daily-grant/history", resourceH.DailyGrantHistory)
	serverairoutes.Register(group, database)

	// Content system (Buildings first, extensible to Units/Research per NEXUS GAME CONTENT REFERENCE v2.0).
	// Admin CRUD + asset upload (images served by this server after upload).
	// Player construction endpoints (queues, completion).
	// Each major item (buildings/units/research) will have its table + CRUD here.
	contentH := handlers.NewContentHandler(services.NewContentService(database, nexusContentAssetsDir()))

	// Admin / catalog
	// Page routes must be mounted before /:contentId routes so "/page" is not
	// interpreted as a content id.
	group.GET("/admin/content/buildings/page", contentH.AdminBuildingsPage)
	group.GET("/admin/content/units/page", contentH.AdminUnitsPage)
	group.GET("/admin/content/research/page", contentH.AdminResearchPage)
	group.GET("/admin/content/catalog", contentH.Catalog)
	group.GET("/admin/content/assets/status", contentH.AssetStatus)
	group.GET("/admin/content/translations/status", contentH.TranslationStatus)
	group.GET("/admin/resources/catalog", resourceH.AdminCatalog)
	group.POST("/admin/resources/seed/preview", resourceH.AdminSeedPreview)
	group.POST("/admin/resources/seed/commit", resourceH.AdminSeedCommit)
	group.GET("/admin/resources/seed/status", resourceH.AdminSeedStatus)

	group.GET("/admin/content/buildings", contentH.ListBuildings)
	group.DELETE("/admin/content/buildings/by-id/:id", contentH.DeleteBuildingByID)
	group.GET("/admin/content/buildings/:contentId", contentH.GetBuilding)
	group.POST("/admin/content/buildings", contentH.CreateOrUpdateBuilding)
	group.PUT("/admin/content/buildings/:contentId", contentH.CreateOrUpdateBuilding)
	group.DELETE("/admin/content/buildings/:contentId", contentH.DeleteBuilding)
	group.POST("/admin/content/buildings/:contentId/delete", contentH.DeleteBuilding)
	group.POST("/admin/content/upload-asset", contentH.UploadAsset) // multipart: file + contentId + domain + tier

	// Player constructions (used by Flutter mmo construction flows)
	// Use :id to match existing /profile/:id/* routes and avoid Gin wildcard conflict
	group.GET("/profile/:id/buildings", contentH.ListPlayerBuildings)
	group.POST("/profile/:id/construction/start", contentH.StartConstruction)
	group.POST("/profile/:id/construction/complete-ready", contentH.CompleteReadyConstructions)

	// === /api/v1/buildings and construction endpoints for Flutter (public client) ===
	// Matches the requested contract. Implemented on top of existing service.
	v1 := router.Group("/api/v1")
	v1.GET("/prerequisites/validate", contentH.ValidatePrerequisitesV1)
	// Catalog
	v1.GET("/buildings/catalog", contentH.ListBuildingsV1)
	v1.GET("/buildings/catalog/version", contentH.CatalogVersionV1)
	v1.GET("/buildings/:key", contentH.GetBuildingV1)
	v1.GET("/buildings/:key/research-tree", contentH.GetBuildingResearchTreeV1)
	v1.GET("/units/catalog", contentH.ListUnitsV1)
	v1.GET("/units/:key", contentH.GetUnitV1)
	v1.GET("/research/catalog", contentH.ListResearchV1)
	v1.GET("/research/:key", contentH.GetResearchV1)
	// Assets
	v1.GET("/assets/buildings/manifest", contentH.BuildingsAssetsManifestV1)
	v1.GET("/assets/buildings/updates", contentH.BuildingsAssetsUpdatesV1)
	// Player buildings
	v1.GET("/buildings", contentH.ListPlayerBuildingsV1) // requires ?profileGamerId= or auth later
	// Legacy build/upgrade preview (can be no-op or call service calc)
	v1.POST("/buildings/build", contentH.LegacyBuildPreviewV1)
	v1.POST("/buildings/:id/upgrade", contentH.LegacyUpgradePreviewV1)
	// Construction
	v1.GET("/construction/queue", contentH.ConstructionQueueV1)
	v1.POST("/construction/start", contentH.StartConstructionV1)
	v1.POST("/construction/:id/upgrade", contentH.StartUpgradeV1)
	v1.POST("/construction/:id/speedup", contentH.SpeedupConstructionV1)
	v1.POST("/construction/:id/cancel", contentH.CancelConstructionV1)
	v1.POST("/construction/:id/complete", contentH.CompleteConstructionV1)

	// NOTE: Both /nexus-assets (game content images) and /admin (Next.js MMO admin UI) are now registered
	// EARLY by the central router (see RegisterAdminStatic + the call in internal/router/router.go).
	// This must happen before any Group("/api/...") to avoid Gin's "*filepath catch-all conflicts with existing 'api'" panic.
	// The late registration here was removed.

	// JSON CRUD for units and research (copy of buildings - extend service impl + seed from reference)
	group.GET("/admin/content/units", contentH.ListUnits)
	group.DELETE("/admin/content/units/by-id/:id", contentH.DeleteUnitByID)
	group.GET("/admin/content/units/:contentId", contentH.GetUnit)
	group.POST("/admin/content/units", contentH.CreateOrUpdateUnit)
	group.PUT("/admin/content/units/:contentId", contentH.CreateOrUpdateUnit)
	group.DELETE("/admin/content/units/:contentId", contentH.DeleteUnit)
	group.POST("/admin/content/units/:contentId/delete", contentH.DeleteUnit)
	group.GET("/admin/content/research", contentH.ListResearch)
	group.DELETE("/admin/content/research/by-id/:id", contentH.DeleteResearchByID)
	group.GET("/admin/content/research/:contentId", contentH.GetResearch)
	group.POST("/admin/content/research", contentH.CreateOrUpdateResearch)
	group.PUT("/admin/content/research/:contentId", contentH.CreateOrUpdateResearch)
	group.DELETE("/admin/content/research/:contentId", contentH.DeleteResearch)
	group.POST("/admin/content/research/:contentId/delete", contentH.DeleteResearch)

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
