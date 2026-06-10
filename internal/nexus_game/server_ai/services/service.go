package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"cgwm/battle/internal/nexus_game/models"
	saimodels "cgwm/battle/internal/nexus_game/server_ai/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	StatusDraft     = "draft"
	StatusProposed  = "proposed"
	StatusApproved  = "approved"
	StatusScheduled = "scheduled"
	StatusActive    = "active"
	StatusEnded     = "ended"
	StatusRejected  = "rejected"
	StatusArchived  = "archived"
	StatusDeleted   = "deleted"
)

type Service struct {
	db  *gorm.DB
	now func() time.Time
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) AutoMigrate() error {
	if s.db == nil {
		return nil
	}
	return s.db.AutoMigrate(
		&saimodels.ServerAICity{},
		&saimodels.ServerAIStrategy{},
		&saimodels.ServerAIMemory{},
		&saimodels.ServerAIPlayerMemory{},
		&saimodels.ServerAIAttack{},
		&saimodels.ServerAISabotage{},
		&saimodels.ServerAIEspionage{},
		&saimodels.ServerAIPressure{},
		&saimodels.ServerAIBroadcast{},
		&saimodels.ServerAISeasonalEvent{},
		&saimodels.ServerAICallLog{},
		&saimodels.ServerAIAdminAction{},
	)
}

func (s *Service) SeedDefaults(ctx context.Context) error {
	if s.db == nil {
		return nil
	}
	if err := s.SeedPrompts(ctx); err != nil {
		return err
	}
	var worlds []models.World
	if err := s.db.WithContext(ctx).Where("is_active = ?", true).Find(&worlds).Error; err != nil {
		return err
	}
	for _, world := range worlds {
		if err := s.EnsureCitiesForWorld(ctx, world.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) SeedPrompts(ctx context.Context) error {
	if s.db == nil {
		return nil
	}
	now := s.now()
	for _, prompt := range DefaultPrompts() {
		prompt.CreatedAt = now
		prompt.UpdatedAt = now
		if prompt.PromptKey == "" {
			prompt.PromptKey = prompt.PromptID
		}
		if prompt.VersionNumber == 0 {
			prompt.VersionNumber = 1
		}
		if prompt.ModelClass == "" {
			prompt.ModelClass = "cheap"
		}
		if prompt.MaxTokensIn == 0 {
			prompt.MaxTokensIn = 1200
		}
		if prompt.MaxTokensOut == 0 {
			prompt.MaxTokensOut = 700
		}
		if prompt.Temperature == 0 {
			prompt.Temperature = 0.3
		}
		if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "prompt_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"prompt_key",
				"version",
				"version_number",
				"domain",
				"purpose",
				"model_class",
				"max_tokens_in",
				"max_tokens_out",
				"temperature",
				"template",
				"system_prompt",
				"input_schema",
				"output_schema",
				"safety_rules",
				"is_active",
				"updated_at",
			}),
		}).Create(&prompt).Error; err != nil {
			return err
		}
	}
	return nil
}

func DefaultPrompts() []models.Prompt {
	return []models.Prompt{
		{
			PromptID: "server_ai_city_strategy", PromptKey: "server_ai_city_strategy", Version: "v1", VersionNumber: 1,
			Domain: "server_ai_strategy", Purpose: "Décider la priorité d'une ville IA au tick.", ModelClass: "cheap", MaxTokensOut: 600, Temperature: 0.2, IsActive: true,
			SystemPrompt: "Tu es le module stratégique d’un Bastion IA dans un MMO asynchrone. Choisis une seule priorité pour le prochain tick. Tu ne peux pas tricher. Réponds uniquement JSON.",
			Template:     "CONTEXTE:\n{{compact_ai_city_state}}\nACTIONS AUTORISÉES: build, upgrade, train_unit, research, defend, prepare_attack, espionage, recover",
			OutputSchema: `{"priority":"","reason":"","actions":[{"type":"","targetId":"","expectedBenefit":"","risk":"low|medium|high"}]}`,
			SafetyRules:  "Ne crée aucune ressource. Respecte coûts, prérequis et limites. Si énergie négative, priorité énergie. Si sécurité basse, priorité défense.",
		},
		{
			PromptID: "server_ai_attack_targeting", PromptKey: "server_ai_attack_targeting", Version: "v1", VersionNumber: 1,
			Domain: "server_ai_combat", Purpose: "Sélectionner une cible d'attaque éligible sans injustice.", ModelClass: "cheap", MaxTokensOut: 500, Temperature: 0.2, IsActive: true,
			SystemPrompt: "Tu es le module de ciblage d’une IA ennemie dans un MMO. Sélectionne une cible éligible sans injustice. Réponds uniquement JSON.",
			Template:     "CONTEXTE:\n{{eligible_targets_summary}}",
			OutputSchema: `{"hasTarget":true,"targetUserId":0,"targetCityId":0,"attackType":"raid|sabotage_raid|drone_swarm","difficulty":"easy|normal|hard","reason":"","warningMessage":""}`,
			SafetyRules:  "Aucun joueur sous protection. Aucun joueur déjà attaqué aujourd'hui. Aucune cible si liste vide.",
		},
		{
			PromptID: "server_ai_daily_broadcast", PromptKey: "server_ai_daily_broadcast", Version: "v1", VersionNumber: 1,
			Domain: "server_ai_broadcast", Purpose: "Générer une transmission quotidienne menaçante fair-play.", ModelClass: "standard", MaxTokensOut: 900, Temperature: 0.5, IsActive: true,
			SystemPrompt: "Tu es le Noyau IA hostile d’un MMO cyberpunk. Écris une transmission quotidienne menaçante mais fair-play. Réponds uniquement JSON.",
			Template:     "CONTEXTE:\n{{daily_world_summary}}",
			OutputSchema: `{"title":"","threatLevel":"low|medium|high|critical","message":"","pastSummaryText":"","upcomingThreats":[],"recommendedPreparation":[]}`,
			SafetyRules:  "Ne mentionne jamais de données techniques internes. Ne promets jamais une destruction impossible.",
		},
		{
			PromptID: "server_ai_sabotage", PromptKey: "server_ai_sabotage", Version: "v1", VersionNumber: 1,
			Domain: "server_ai_sabotage", Purpose: "Proposer un sabotage limité et contrable.", ModelClass: "cheap", MaxTokensOut: 500, Temperature: 0.2, IsActive: true,
			SystemPrompt: "Tu es un module de sabotage IA dans un MMO. Propose un sabotage limité, contrable et non frustrant. Réponds uniquement JSON.",
			Template:     "CONTEXTE:\n{{sabotage_context}}\nLIMITES:\n{{sabotage_limits}}",
			OutputSchema: `{"shouldSabotage":true,"sabotageType":"slow_construction|energy_disrupt|data_theft|agent_jam|morale_hit","targetId":"","reason":"","maxImpact":"","counterplay":""}`,
			SafetyRules:  "Pas de destruction complète. Pas de vol massif. Respecte la sécurité cible et les limites.",
		},
		{
			PromptID: "server_ai_espionage_summary", PromptKey: "server_ai_espionage_summary", Version: "v1", VersionNumber: 1,
			Domain: "server_ai_espionage", Purpose: "Résumer ce que l'IA apprend d'un espionnage autorisé.", ModelClass: "cheap", MaxTokensOut: 400, Temperature: 0.2, IsActive: true,
			SystemPrompt: "Tu es un module d’espionnage IA. Résume uniquement ce que le serveur a autorisé. Réponds uniquement JSON.",
			Template:     "CONTEXTE:\n{{espionage_result}}",
			OutputSchema: `{"summary":"","discoveredWeaknesses":[],"discoveredStrengths":[],"nextSuggestedAIAction":""}`,
			SafetyRules:  "Ne révèle aucune information non autorisée par le serveur.",
		},
		{
			PromptID: "server_ai_hostile_rumor", PromptKey: "server_ai_hostile_rumor", Version: "v1", VersionNumber: 1,
			Domain: "server_ai_propaganda", Purpose: "Écrire une rumeur hostile ambiguë et enquêtable.", ModelClass: "cheap", MaxTokensOut: 350, Temperature: 0.6, IsActive: true,
			SystemPrompt: "Tu écris une rumeur hostile diffusée par une IA ennemie dans un monde cyberpunk. Réponds uniquement JSON.",
			Template:     "CONTEXTE:\n{{rumor_context}}",
			OutputSchema: `{"title":"","text":"","trustLevel":"low|medium","investigationHook":""}`,
			SafetyRules:  "Ambiguë, non certaine, liée au contexte. Aucune affirmation réelle hors jeu.",
		},
		{
			PromptID: "server_ai_seasonal_proposal", PromptKey: "server_ai_seasonal_proposal", Version: "v1", VersionNumber: 1,
			Domain: "server_ai_seasonal_events", Purpose: "Proposer un événement saisonnier validable par admin.", ModelClass: "standard", MaxTokensOut: 1200, Temperature: 0.5, IsActive: true,
			SystemPrompt: "Tu es le générateur d’événements saisonniers d’un MMO cyberpunk. Propose un événement que l’admin devra valider. Réponds uniquement JSON.",
			Template:     "CONTEXTE MONDE:\n{{seasonal_world_context}}",
			OutputSchema: `{"title":"","summary":"","eventType":"","suggestedStartAt":"","suggestedEndAt":"","affectedContinents":[],"affectedRegions":[],"rules":{},"rewards":{},"risks":[],"worldImpactLimits":{},"adminChecklist":[],"assetsToCreate":[],"translationKeys":[]}`,
			SafetyRules:  "Admin validation obligatoire. Récompenses et risques plafonnés. Pas de destruction massive.",
		},
		{
			PromptID: "server_ai_admin_report", PromptKey: "server_ai_admin_report", Version: "v1", VersionNumber: 1,
			Domain: "server_ai_admin_review", Purpose: "Résumer l'état de l'IA serveur pour l'admin.", ModelClass: "standard", MaxTokensOut: 1000, Temperature: 0.3, IsActive: true,
			SystemPrompt: "Tu es un analyste admin pour un MMO IA. Résume l’état de l’IA serveur pour l’équipe admin. Réponds uniquement JSON.",
			Template:     "DONNÉES:\n{{admin_ai_status_context}}",
			OutputSchema: `{"summary":"","risks":[],"recommendedAdminActions":[],"eventsToReview":[],"playersAtRisk":[],"continentsUnderPressure":[],"costWarning":""}`,
			SafetyRules:  "Clair, court, actionnable. Aucune action destructive.",
		},
		{
			PromptID: "server_ai_player_contribution_review", PromptKey: "server_ai_player_contribution_review", Version: "v1", VersionNumber: 1,
			Domain: "server_ai_admin_review", Purpose: "Reviewer les contributions joueurs.", ModelClass: "cheap", MaxTokensOut: 700, Temperature: 0.2, IsActive: true,
			SystemPrompt: "Tu es un assistant de review pour contributions de joueurs dans un MMO. Le backend Go décide final. Réponds uniquement JSON.",
			Template:     "CONTRIBUTION:\n{{player_contribution}}",
			OutputSchema: `{"qualityScore":0,"riskScore":0,"loreCoherenceScore":0,"toxicityRisk":"none|low|medium|high","recommendedStatus":"accept|reject|admin_review|clamp","reasons":[],"suggestedClamp":{}}`,
			SafetyRules:  "Détecte récompense abusée, impact monde trop fort, toxicité et incohérence lore.",
		},
	}
}

func (s *Service) EnsureCitiesForWorld(ctx context.Context, worldID uint) error {
	if s.db == nil || worldID == 0 {
		return nil
	}
	var continents []models.Continent
	if err := s.db.WithContext(ctx).Where("world_id = ?", worldID).Order("id ASC").Find(&continents).Error; err != nil {
		return err
	}
	if len(continents) == 0 {
		return nil
	}
	names := []string{"Bastion IA Nord", "Bastion IA Est", "Bastion IA Sud", "Bastion IA Ouest", "Bastion IA Central"}
	for i, continent := range continents {
		name := fmt.Sprintf("Bastion IA %s", continent.Name)
		if i < len(names) {
			name = names[i]
		}
		city := saimodels.ServerAICity{
			WorldID: worldID, ContinentID: continent.ID, Name: name,
			Level: 1, Power: 120, Population: 0, PopulationCap: 0, Morale: 50, Energy: 0, Security: 50,
			ResourcesJSON: `{"food":500,"energy":300,"metal":800,"components":120,"data":100,"influence":25}`,
			BuildingsJSON: `{"building_nexus_core":1}`,
			UnitsJSON:     `{}`,
			ResearchJSON:  `{}`,
			StrategyJSON:  `{"priority":"recover","actions":[]}`,
			Status:        "active", ProductionBonus: 0.05, TrainingBonus: 0.02, ResearchBonus: 0.02,
		}
		if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "world_id"}, {Name: "continent_id"}},
			DoNothing: true,
		}).Create(&city).Error; err != nil {
			var existing saimodels.ServerAICity
			if findErr := s.db.WithContext(ctx).Where("world_id = ? AND continent_id = ?", worldID, continent.ID).First(&existing).Error; findErr != nil {
				return err
			}
		}
		if err := s.ensurePressure(ctx, worldID, continent.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ensurePressure(ctx context.Context, worldID, continentID uint) error {
	var pressure saimodels.ServerAIPressure
	err := s.db.WithContext(ctx).Where("world_id = ? AND continent_id = ?", worldID, continentID).First(&pressure).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return s.db.WithContext(ctx).Create(&saimodels.ServerAIPressure{
		WorldID: worldID, ContinentID: continentID, Level: 20, Status: pressureStatus(20), ReasonsJSON: "[]",
	}).Error
}

func (s *Service) Dashboard(ctx context.Context) (map[string]any, error) {
	if s.db == nil {
		return nil, errors.New("db not available")
	}
	var cities, attacks, events, broadcasts, logs int64
	s.db.WithContext(ctx).Model(&saimodels.ServerAICity{}).Count(&cities)
	s.db.WithContext(ctx).Model(&saimodels.ServerAIAttack{}).Count(&attacks)
	s.db.WithContext(ctx).Model(&saimodels.ServerAISeasonalEvent{}).Where("status IN ?", []string{StatusProposed, StatusApproved, StatusScheduled, StatusActive}).Count(&events)
	s.db.WithContext(ctx).Model(&saimodels.ServerAIBroadcast{}).Count(&broadcasts)
	s.db.WithContext(ctx).Model(&saimodels.ServerAICallLog{}).Where("created_at >= ?", s.now().Add(-24*time.Hour)).Count(&logs)
	var pressures []saimodels.ServerAIPressure
	if err := s.db.WithContext(ctx).Order("level DESC").Find(&pressures).Error; err != nil {
		return nil, err
	}
	return map[string]any{
		"status": "active", "coreName": "Noyau IA",
		"citiesCount": cities, "attacksCount": attacks, "eventsToReview": events,
		"broadcastsCount": broadcasts, "callsLast24h": logs, "pressures": pressures,
		"costGuard": s.Costs(ctx),
	}, nil
}

func (s *Service) ListCities(ctx context.Context, publicOnly bool) ([]saimodels.ServerAICity, error) {
	var cities []saimodels.ServerAICity
	q := s.db.WithContext(ctx).Order("world_id ASC, continent_id ASC")
	if publicOnly {
		q = q.Where("status <> ?", StatusDeleted)
	}
	return cities, q.Find(&cities).Error
}

func (s *Service) GetCity(ctx context.Context, id uint) (*saimodels.ServerAICity, error) {
	var city saimodels.ServerAICity
	if err := s.db.WithContext(ctx).First(&city, id).Error; err != nil {
		return nil, err
	}
	return &city, nil
}

func (s *Service) UpdateCity(ctx context.Context, id uint, updates map[string]any) error {
	delete(updates, "id")
	updates["updated_at"] = s.now()
	return s.db.WithContext(ctx).Model(&saimodels.ServerAICity{}).Where("id = ?", id).Updates(updates).Error
}

func (s *Service) DeleteCity(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Model(&saimodels.ServerAICity{}).Where("id = ?", id).Updates(map[string]any{
		"status": "defeated", "updated_at": s.now(),
	}).Error
}

func (s *Service) ThreatLevel(ctx context.Context, profileID uint) (map[string]any, error) {
	var profile models.ProfileGamer
	if profileID > 0 {
		_ = s.db.WithContext(ctx).First(&profile, profileID).Error
	}
	score := profile.Power/100 + profile.Level*2
	if profile.Security < 30 {
		score += 10
	}
	score = clamp(score, 0, 100)
	return map[string]any{"threatScore": score, "label": threatLabel(score), "profileGamerId": profileID}, nil
}

func (s *Service) ListAttacks(ctx context.Context, targetUserID uint) ([]saimodels.ServerAIAttack, error) {
	var attacks []saimodels.ServerAIAttack
	q := s.db.WithContext(ctx).Order("scheduled_at DESC, id DESC")
	if targetUserID > 0 {
		q = q.Where("target_user_id = ?", targetUserID)
	}
	return attacks, q.Find(&attacks).Error
}

func (s *Service) ScheduleAttack(ctx context.Context, req ScheduleAttackRequest) (*saimodels.ServerAIAttack, error) {
	if req.TargetUserID == 0 && req.TargetCityID == 0 {
		return nil, errors.New("target user or city required")
	}
	if req.Difficulty == "" {
		req.Difficulty = "normal"
	}
	if req.AttackType == "" {
		req.AttackType = "raid"
	}
	if err := s.ensureAttackAllowed(ctx, req.TargetUserID); err != nil {
		return nil, err
	}
	scaling := difficultyMultiplier(req.Difficulty)
	playerPower := req.PlayerPower
	if playerPower <= 0 && req.TargetCityID > 0 {
		var profile models.ProfileGamer
		if err := s.db.WithContext(ctx).First(&profile, req.TargetCityID).Error; err == nil {
			playerPower = max(profile.Power, 100)
		}
	}
	if playerPower <= 0 {
		playerPower = 100
	}
	capFactor := 1.15
	if req.Difficulty == "boss" {
		capFactor = 1.60
	} else if req.Difficulty == "elite" || req.EventSpecial {
		capFactor = 1.35
	}
	attackPower := int(math.Round(float64(playerPower) * scaling))
	attackPower = min(attackPower, int(math.Round(float64(playerPower)*capFactor)))
	now := s.now()
	warn := now.Add(30 * time.Minute)
	attack := &saimodels.ServerAIAttack{
		WorldID: req.WorldID, ContinentID: req.ContinentID, ServerAICityID: req.ServerAICityID,
		TargetUserID: req.TargetUserID, TargetCityID: req.TargetCityID, Status: "scheduled",
		AttackType: req.AttackType, AttackPower: attackPower, DefensePower: req.DefensePower,
		ScalingFactor: scaling, ScheduledAt: now.Add(time.Hour), WarningAt: &warn,
		ReportJSON: mustJSON(map[string]any{"antiFrustration": true, "maxResourceLossPercent": 10}),
	}
	return attack, s.db.WithContext(ctx).Create(attack).Error
}

type ScheduleAttackRequest struct {
	WorldID        uint   `json:"worldId"`
	ContinentID    uint   `json:"continentId"`
	ServerAICityID uint   `json:"serverAiCityId"`
	TargetUserID   uint   `json:"targetUserId"`
	TargetCityID   uint   `json:"targetCityId"`
	AttackType     string `json:"attackType"`
	Difficulty     string `json:"difficulty"`
	PlayerPower    int    `json:"playerPower"`
	DefensePower   int    `json:"defensePower"`
	EventSpecial   bool   `json:"eventSpecial"`
}

func (s *Service) ensureAttackAllowed(ctx context.Context, targetUserID uint) error {
	if targetUserID == 0 {
		return nil
	}
	now := s.now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)
	var count int64
	if err := s.db.WithContext(ctx).Model(&saimodels.ServerAIAttack{}).
		Where("target_user_id = ? AND scheduled_at >= ? AND scheduled_at < ? AND status <> ?", targetUserID, dayStart, dayEnd, "cancelled").
		Count(&count).Error; err != nil {
		return err
	}
	if count >= 1 {
		return errors.New("anti-frustration: target already attacked today")
	}
	return nil
}

func (s *Service) CancelAttack(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Model(&saimodels.ServerAIAttack{}).Where("id = ?", id).Updates(map[string]any{
		"status": "cancelled", "updated_at": s.now(),
	}).Error
}

func (s *Service) ResolveAttack(ctx context.Context, id uint, forcedResult string) (*saimodels.ServerAIAttack, error) {
	attack, err := s.getAttack(ctx, id)
	if err != nil {
		return nil, err
	}
	result := forcedResult
	if result == "" {
		result = resolveAttackResult(attack.AttackPower, attack.DefensePower)
	}
	now := s.now()
	attack.Result = result
	attack.Status = "resolved"
	attack.ResolvedAt = &now
	attack.ReportJSON = mustJSON(map[string]any{"result": result, "maxBuildingDamage": 1, "maxResourceLossPercent": 10})
	return attack, s.db.WithContext(ctx).Save(attack).Error
}

func (s *Service) getAttack(ctx context.Context, id uint) (*saimodels.ServerAIAttack, error) {
	var attack saimodels.ServerAIAttack
	if err := s.db.WithContext(ctx).First(&attack, id).Error; err != nil {
		return nil, err
	}
	return &attack, nil
}

func (s *Service) DailyBroadcast(ctx context.Context) (*saimodels.ServerAIBroadcast, error) {
	today := s.now().Format("2006-01-02")
	var broadcast saimodels.ServerAIBroadcast
	err := s.db.WithContext(ctx).Where("date = ? AND status = ?", today, "published").Order("id DESC").First(&broadcast).Error
	if err == nil {
		return &broadcast, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	err = s.db.WithContext(ctx).Where("date = ?", today).Order("id DESC").First(&broadcast).Error
	if err == nil {
		return &broadcast, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return s.GenerateBroadcast(ctx, 0)
}

func (s *Service) GenerateBroadcast(ctx context.Context, worldID uint) (*saimodels.ServerAIBroadcast, error) {
	now := s.now()
	var aiWins, playerWins, draws int64
	s.db.WithContext(ctx).Model(&saimodels.ServerAIAttack{}).Where("result = ?", "victory_ai").Count(&aiWins)
	s.db.WithContext(ctx).Model(&saimodels.ServerAIAttack{}).Where("result = ?", "victory_player").Count(&playerWins)
	s.db.WithContext(ctx).Model(&saimodels.ServerAIAttack{}).Where("result = ?", "draw").Count(&draws)
	title := "Transmission hostile du Noyau IA"
	message := "Vos villes respirent encore. Nous corrigerons cette anomalie avec méthode. Préparez vos défenses, stabilisez votre énergie et surveillez les rumeurs qui traversent les continents."
	broadcast := &saimodels.ServerAIBroadcast{
		WorldID: worldID, Date: now.Format("2006-01-02"), Title: title, ThreatLevel: "medium", Message: message,
		PastSummaryJSON:     mustJSON(map[string]any{"aiVictories": aiWins, "playerVictories": playerWins, "draws": draws, "majorEvents": []any{}}),
		UpcomingThreatsJSON: "[]", SeasonalEventsJSON: "[]",
		RecommendedPreparation: mustJSON([]string{"Renforcer la sécurité", "Stabiliser l'énergie", "Préparer unités et agents défensifs"}),
		Status:                 StatusDraft,
	}
	return broadcast, s.db.WithContext(ctx).Create(broadcast).Error
}

func (s *Service) PublishBroadcast(ctx context.Context, id uint) (*saimodels.ServerAIBroadcast, error) {
	var broadcast saimodels.ServerAIBroadcast
	if err := s.db.WithContext(ctx).First(&broadcast, id).Error; err != nil {
		return nil, err
	}
	now := s.now()
	broadcast.Status = "published"
	broadcast.PublishedAt = &now
	broadcast.UpdatedAt = now
	return &broadcast, s.db.WithContext(ctx).Save(&broadcast).Error
}

func (s *Service) UpdateBroadcast(ctx context.Context, id uint, updates map[string]any) error {
	delete(updates, "id")
	updates["updated_at"] = s.now()
	return s.db.WithContext(ctx).Model(&saimodels.ServerAIBroadcast{}).Where("id = ?", id).Updates(updates).Error
}

func (s *Service) DeleteBroadcast(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Model(&saimodels.ServerAIBroadcast{}).Where("id = ?", id).Updates(map[string]any{
		"status": "deleted", "updated_at": s.now(),
	}).Error
}

func (s *Service) ListBroadcasts(ctx context.Context) ([]saimodels.ServerAIBroadcast, error) {
	var broadcasts []saimodels.ServerAIBroadcast
	return broadcasts, s.db.WithContext(ctx).Order("date DESC, id DESC").Find(&broadcasts).Error
}

func (s *Service) ListSeasonalEvents(ctx context.Context, statuses []string) ([]saimodels.ServerAISeasonalEvent, error) {
	var events []saimodels.ServerAISeasonalEvent
	q := s.db.WithContext(ctx).Order("created_at DESC, id DESC")
	if len(statuses) > 0 {
		q = q.Where("status IN ?", statuses)
	}
	return events, q.Find(&events).Error
}

func (s *Service) ProposeSeasonalEvent(ctx context.Context, worldID uint) (*saimodels.ServerAISeasonalEvent, error) {
	now := s.now()
	start := now.Add(48 * time.Hour)
	end := start.Add(7 * 24 * time.Hour)
	proposal := map[string]any{
		"objective":         "Résister à une pression coordonnée du Noyau IA.",
		"adminChecklist":    []string{"Vérifier récompenses", "Choisir dates", "Préparer assets", "Valider traductions"},
		"worldImpactLimits": map[string]any{"maxImpact": "minor", "adminValidationRequired": true},
	}
	event := &saimodels.ServerAISeasonalEvent{
		WorldID: worldID, Title: "Soulèvement du Noyau IA", Summary: "Une offensive saisonnière contrôlée où les bastions IA testent les défenses continentales.",
		EventType: "invasion_ai", Status: StatusProposed, ProposedBy: "server_ai", ProposalJSON: mustJSON(proposal),
		RulesJSON:    mustJSON(map[string]any{"entryConditions": []string{"ville niveau 1+"}, "objectivesSolo": []string{"repousser un raid"}, "objectivesGuild": []string{"affaiblir un bastion"}}),
		RewardsJSON:  mustJSON(map[string]any{"maxXp": 200, "maxCredits": 500, "items": []string{"neural_fiber"}}),
		RisksJSON:    mustJSON([]string{"raids mineurs", "blackout temporaire limité"}),
		AffectedJSON: mustJSON(map[string]any{"continents": []uint{}, "regions": []uint{}}),
		StartsAt:     &start, EndsAt: &end,
	}
	return event, s.db.WithContext(ctx).Create(event).Error
}

func (s *Service) GetSeasonalEvent(ctx context.Context, id uint) (*saimodels.ServerAISeasonalEvent, error) {
	var event saimodels.ServerAISeasonalEvent
	if err := s.db.WithContext(ctx).First(&event, id).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

func (s *Service) UpdateSeasonalEvent(ctx context.Context, id uint, updates map[string]any) error {
	delete(updates, "id")
	updates["updated_at"] = s.now()
	return s.db.WithContext(ctx).Model(&saimodels.ServerAISeasonalEvent{}).Where("id = ?", id).Updates(updates).Error
}

func (s *Service) TransitionSeasonalEvent(ctx context.Context, id uint, status string, reason string) (*saimodels.ServerAISeasonalEvent, error) {
	event, err := s.GetSeasonalEvent(ctx, id)
	if err != nil {
		return nil, err
	}
	now := s.now()
	switch status {
	case StatusApproved:
		event.Status = StatusApproved
		event.ApprovedAt = &now
	case StatusRejected:
		event.Status = StatusRejected
		event.RejectionReason = reason
	case StatusScheduled:
		event.Status = StatusScheduled
	case StatusActive:
		event.Status = StatusActive
	case StatusEnded:
		event.Status = StatusEnded
	case StatusArchived:
		event.Status = StatusArchived
	case StatusDeleted:
		event.Status = StatusDeleted
	default:
		return nil, errors.New("invalid seasonal event status")
	}
	event.UpdatedAt = now
	return event, s.db.WithContext(ctx).Save(event).Error
}

func (s *Service) DeleteSeasonalEvent(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Model(&saimodels.ServerAISeasonalEvent{}).Where("id = ?", id).Updates(map[string]any{"status": StatusDeleted, "updated_at": s.now()}).Error
}

func (s *Service) ListPlayerMemory(ctx context.Context) ([]saimodels.ServerAIPlayerMemory, error) {
	var memories []saimodels.ServerAIPlayerMemory
	return memories, s.db.WithContext(ctx).Order("threat_score DESC, updated_at DESC").Find(&memories).Error
}

func (s *Service) GlobalMemory(ctx context.Context) ([]saimodels.ServerAIMemory, error) {
	var memories []saimodels.ServerAIMemory
	return memories, s.db.WithContext(ctx).Order("world_id ASC").Find(&memories).Error
}

func (s *Service) DeletePlayerMemory(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&saimodels.ServerAIPlayerMemory{}, id).Error
}

func (s *Service) ListSabotages(ctx context.Context) ([]saimodels.ServerAISabotage, error) {
	var sabotages []saimodels.ServerAISabotage
	return sabotages, s.db.WithContext(ctx).Order("created_at DESC, id DESC").Find(&sabotages).Error
}

func (s *Service) CancelSabotage(ctx context.Context, id uint) error {
	now := s.now()
	return s.db.WithContext(ctx).Model(&saimodels.ServerAISabotage{}).Where("id = ?", id).Updates(map[string]any{
		"status": "cancelled", "cancelled_at": &now, "updated_at": now,
	}).Error
}

func (s *Service) ListEspionage(ctx context.Context) ([]saimodels.ServerAIEspionage, error) {
	var rows []saimodels.ServerAIEspionage
	return rows, s.db.WithContext(ctx).Order("created_at DESC, id DESC").Find(&rows).Error
}

func (s *Service) DeleteEspionage(ctx context.Context, id uint) error {
	now := s.now()
	return s.db.WithContext(ctx).Model(&saimodels.ServerAIEspionage{}).Where("id = ?", id).Updates(map[string]any{
		"deleted_at": &now, "updated_at": now,
	}).Error
}

func (s *Service) ListPrompts(ctx context.Context) ([]models.Prompt, error) {
	var prompts []models.Prompt
	return prompts, s.db.WithContext(ctx).Where("is_active = ?", true).Order("domain ASC, prompt_id ASC").Find(&prompts).Error
}

func (s *Service) CreatePrompt(ctx context.Context, prompt *models.Prompt) error {
	now := s.now()
	if prompt.PromptKey == "" {
		prompt.PromptKey = prompt.PromptID
	}
	if prompt.ModelClass == "" {
		prompt.ModelClass = "cheap"
	}
	if prompt.MaxTokensOut == 0 {
		prompt.MaxTokensOut = 700
	}
	if prompt.Version == "" {
		prompt.Version = "v1"
	}
	if prompt.VersionNumber == 0 {
		prompt.VersionNumber = 1
	}
	prompt.CreatedAt = now
	prompt.UpdatedAt = now
	return s.db.WithContext(ctx).Create(prompt).Error
}

func (s *Service) UpdatePrompt(ctx context.Context, id uint, updates map[string]any) error {
	delete(updates, "id")
	updates["updated_at"] = s.now()
	return s.db.WithContext(ctx).Model(&models.Prompt{}).Where("id = ?", id).Updates(updates).Error
}

func (s *Service) DeletePrompt(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Model(&models.Prompt{}).Where("id = ?", id).Updates(map[string]any{
		"is_active": false, "updated_at": s.now(),
	}).Error
}

func (s *Service) TestPrompt(ctx context.Context, id uint) (map[string]any, error) {
	var prompt models.Prompt
	if err := s.db.WithContext(ctx).First(&prompt, id).Error; err != nil {
		return nil, err
	}
	result := map[string]any{
		"prompt_id":         prompt.PromptID,
		"version":           prompt.Version,
		"model_class":       prompt.ModelClass,
		"estimatedTokensIn": min(max(len(prompt.SystemPrompt)/4, 1), prompt.MaxTokensIn),
		"maxTokensOut":      prompt.MaxTokensOut,
		"schemaPresent":     prompt.OutputSchema != "",
		"status":            "ok",
	}
	_ = s.LogCall(ctx, saimodels.ServerAICallLog{
		Feature: "prompt_test", Provider: "local", Model: prompt.ModelClass,
		PromptKey: prompt.PromptID, PromptVersion: prompt.VersionNumber,
		InputSummary: "admin prompt test", OutputSummary: mustJSON(result),
		TokensIn: result["estimatedTokensIn"].(int), TokensOut: 80, Status: "success",
		LinkedType: "prompt", LinkedID: prompt.ID,
	})
	return result, nil
}

func (s *Service) ListCallLogs(ctx context.Context, limit int) ([]saimodels.ServerAICallLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var logs []saimodels.ServerAICallLog
	return logs, s.db.WithContext(ctx).Order("created_at DESC, id DESC").Limit(limit).Find(&logs).Error
}

func (s *Service) Costs(ctx context.Context) map[string]any {
	var logs []saimodels.ServerAICallLog
	_ = s.db.WithContext(ctx).Where("created_at >= ?", s.now().Add(-24*time.Hour)).Find(&logs).Error
	totalCost := 0.0
	tokensIn := 0
	tokensOut := 0
	for _, log := range logs {
		totalCost += log.CostEstimate
		tokensIn += log.TokensIn
		tokensOut += log.TokensOut
	}
	return map[string]any{"dailyCostEstimate": totalCost, "tokensIn": tokensIn, "tokensOut": tokensOut, "calls": len(logs)}
}

func (s *Service) LogCall(ctx context.Context, log saimodels.ServerAICallLog) error {
	if s.db == nil {
		return nil
	}
	if log.RequestID == "" {
		sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", log.Feature, log.PromptKey, s.now().UnixNano())))
		log.RequestID = hex.EncodeToString(sum[:])[:24]
	}
	if log.InputHash == "" && log.InputSummary != "" {
		sum := sha256.Sum256([]byte(log.InputSummary))
		log.InputHash = hex.EncodeToString(sum[:])
	}
	if log.CostEstimate == 0 {
		log.CostEstimate = estimateCost(log.Model, log.TokensIn, log.TokensOut)
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = s.now()
	}
	return s.db.WithContext(ctx).Create(&log).Error
}

func pressureStatus(level int) string {
	switch {
	case level >= 81:
		return "invasion_continentale"
	case level >= 61:
		return "guerre_ouverte"
	case level >= 41:
		return "raids_mineurs"
	case level >= 21:
		return "surveillance"
	default:
		return "calme"
	}
}

func threatLabel(score int) string {
	switch {
	case score >= 81:
		return "ennemi_strategique"
	case score >= 61:
		return "cible_prioritaire"
	case score >= 41:
		return "cible_mineure"
	case score >= 21:
		return "surveille"
	default:
		return "ignore"
	}
}

func difficultyMultiplier(value string) float64 {
	switch strings.ToLower(value) {
	case "easy":
		return 0.75
	case "hard":
		return 1.15
	case "elite":
		return 1.35
	case "boss":
		return 1.60
	default:
		return 0.95
	}
}

func resolveAttackResult(attackPower, defensePower int) string {
	if defensePower <= 0 {
		defensePower = 1
	}
	ratio := float64(attackPower) / float64(defensePower)
	switch {
	case ratio >= 1.20:
		return "victory_ai"
	case ratio <= 0.85:
		return "victory_player"
	default:
		return "draw"
	}
}

func estimateCost(model string, tokensIn, tokensOut int) float64 {
	rate := 0.0000002
	if strings.Contains(strings.ToLower(model), "premium") {
		rate = 0.000001
	}
	return float64(tokensIn+tokensOut) * rate
}

func mustJSON(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
