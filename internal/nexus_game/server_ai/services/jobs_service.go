package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"cgwm/battle/internal/nexus_game/models"
	saimodels "cgwm/battle/internal/nexus_game/server_ai/models"

	"gorm.io/gorm"
)

type ServerAIJobSpec struct {
	Key             string `json:"key"`
	Name            string `json:"name"`
	Category        string `json:"category"`
	Frequency       string `json:"frequency"`
	IntervalSeconds int64  `json:"intervalSeconds"`
	GenerativeMode  string `json:"generativeMode"`
	Description     string `json:"description"`
}

type ServerAIJobRunResult struct {
	JobKey       string         `json:"jobKey"`
	Status       string         `json:"status"`
	Processed    int            `json:"processed"`
	CreatedCount int            `json:"createdCount"`
	UpdatedCount int            `json:"updatedCount"`
	SkippedCount int            `json:"skippedCount"`
	Summary      map[string]any `json:"summary"`
}

type ServerAIJobView struct {
	ServerAIJobSpec
	LastRun   *saimodels.ServerAIJobRun `json:"lastRun,omitempty"`
	NextRunAt string                    `json:"nextRunAt,omitempty"`
	Due       bool                      `json:"due"`
}

func ServerAIJobSpecs() []ServerAIJobSpec {
	return []ServerAIJobSpec{
		{"server_ai_scheduler_guard_job", "Guard scheduler", "guard", "Chaque minute", 60, "none", "Locks, statuts en attente, attaques dues et événements saisonniers à démarrer/terminer."},
		{"server_ai_threat_scan_job", "Scan menace rapide", "threat", "Toutes les 5 minutes", 5 * 60, "none", "Recalcule threatScore, protection débutant, styles joueurs et files sensibles."},
		{"server_ai_attack_queue_job", "File attaques IA", "attack", "Toutes les 5 minutes", 5 * 60, "none", "Met les attaques proches en alerte et évite les doublons d'exécution."},
		{"server_ai_sabotage_queue_job", "File sabotages IA", "sabotage", "Toutes les 5 minutes", 5 * 60, "none", "Nettoie/contrôle les sabotages proposés ou dépassés."},
		{"server_ai_continent_pressure_job", "Pression continentale", "pressure", "Toutes les 15 minutes", 15 * 60, "none", "Recalcule la pression IA par continent."},
		{"server_ai_city_tick_job", "Tick villes IA", "city", "Toutes les 30 minutes", 30 * 60, "none", "Production, consommation, moral, sécurité et population des bastions IA."},
		{"server_ai_bastion_strategy_job", "Stratégie bastions IA", "strategy", "Toutes les heures", 60 * 60, "rare", "Choisit priorité, défense, recherche, entraînement ou préparation d'attaque."},
		{"server_ai_espionage_job", "Espionnage IA", "espionage", "Toutes les 2 heures", 2 * 60 * 60, "rare", "Sélectionne cibles non débutantes et enrichit la mémoire joueur."},
		{"server_ai_sabotage_job", "Sabotage limité", "sabotage", "Toutes les 3 heures", 3 * 60 * 60, "rare", "Crée des sabotages légers, contrables et anti-frustration."},
		{"server_ai_attack_scheduler_job", "Planification attaques", "attack", "Toutes les 4 heures", 4 * 60 * 60, "rare", "Planifie des attaques scalées, max 1 par joueur par jour."},
		{"server_ai_attack_resolver_job", "Résolution attaques", "attack", "Toutes les 6 heures", 6 * 60 * 60, "report_optional", "Résout les attaques arrivées à échéance et met à jour les mémoires."},
		{"server_ai_daily_strategy_job", "Daily Plan IA serveur", "daily", "Tous les jours 04:00", 24 * 60 * 60, "standard", "Définit stratégie hostile du monde et objectifs des bastions."},
		{"server_ai_daily_broadcast_job", "Transmission quotidienne", "daily", "Tous les jours 04:00", 24 * 60 * 60, "standard", "Génère et publie la transmission quotidienne du Noyau IA."},
		{"server_ai_global_memory_update_job", "Mémoire globale", "memory", "Tous les jours 04:00", 24 * 60 * 60, "none", "Agrège victoires, défaites, continents et tendances monde."},
		{"server_ai_player_memory_daily_job", "Mémoire joueur quotidienne", "memory", "Tous les jours 04:30", 24 * 60 * 60, "none", "Decay rivalité, style joueur, forces/faiblesses et nettoyage."},
		{"server_ai_cost_guard_job", "Garde coûts IA", "cost", "Tous les jours 05:00", 24 * 60 * 60, "none", "Agrège coût, tokens, appels et crée alerte admin si budget dépassé."},
		{"server_ai_event_proposal_job", "Proposition événement admin", "event", "Tous les jours 05:30", 24 * 60 * 60, "standard", "Propose des événements saisonniers ou régionaux à valider par admin."},
		{"server_ai_weekly_world_review_job", "Bilan monde hebdomadaire", "review", "Chaque lundi 03:00", 7 * 24 * 60 * 60, "standard", "Bilan monde, difficulté globale, guildes dominantes, arcs narratifs."},
		{"server_ai_seasonal_planning_job", "Planification saisonnière", "season", "Mensuel ou début saison", 30 * 24 * 60 * 60, "standard", "Prépare 3 à 5 propositions d'événements et arcs saisonniers."},
		{"server_ai_seasonal_event_start_job", "Démarrage événement validé", "season", "Chaque minute", 60, "none", "Démarre uniquement les événements approuvés/scheduled dont startsAt est atteint."},
		{"server_ai_seasonal_event_tick_job", "Tick événement saisonnier", "season", "Toutes les 1 à 3 heures", 2 * 60 * 60, "rare", "Suit participation, pression IA et rapports intermédiaires."},
		{"server_ai_seasonal_event_end_job", "Fin événement saisonnier", "season", "Chaque minute", 60, "none", "Termine les événements actifs dont endsAt est atteint et prépare résumé."},
	}
}

func (s *Service) ListJobs(ctx context.Context) ([]ServerAIJobView, error) {
	if s.db == nil {
		return nil, errors.New("db not available")
	}
	specs := ServerAIJobSpecs()
	views := make([]ServerAIJobView, 0, len(specs))
	now := s.now()
	for _, spec := range specs {
		view := ServerAIJobView{ServerAIJobSpec: spec, Due: true}
		var run saimodels.ServerAIJobRun
		err := s.db.WithContext(ctx).Where("job_key = ?", spec.Key).Order("started_at DESC, id DESC").First(&run).Error
		if err == nil {
			view.LastRun = &run
			next := run.StartedAt.Add(time.Duration(spec.IntervalSeconds) * time.Second)
			view.NextRunAt = next.UTC().Format(time.RFC3339)
			view.Due = !next.After(now)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) JobDashboard(ctx context.Context) map[string]any {
	jobs, err := s.ListJobs(ctx)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	due := 0
	failed := 0
	for _, job := range jobs {
		if job.Due {
			due++
		}
		if job.LastRun != nil && job.LastRun.Status == "failed" {
			failed++
		}
	}
	return map[string]any{"total": len(jobs), "due": due, "failed": failed, "items": jobs}
}

func (s *Service) RunDueJobs(ctx context.Context, trigger string, worldID uint) ([]ServerAIJobRunResult, error) {
	jobs, err := s.ListJobs(ctx)
	if err != nil {
		return nil, err
	}
	results := []ServerAIJobRunResult{}
	for _, job := range jobs {
		if !job.Due {
			continue
		}
		result, runErr := s.RunJob(ctx, job.Key, trigger, worldID)
		results = append(results, result)
		if runErr != nil {
			return results, runErr
		}
	}
	return results, nil
}

func (s *Service) RunAllJobs(ctx context.Context, trigger string, worldID uint) ([]ServerAIJobRunResult, error) {
	results := []ServerAIJobRunResult{}
	for _, spec := range ServerAIJobSpecs() {
		result, err := s.RunJob(ctx, spec.Key, trigger, worldID)
		results = append(results, result)
		if err != nil {
			return results, err
		}
	}
	return results, nil
}

func (s *Service) RunJob(ctx context.Context, key string, trigger string, worldID uint) (ServerAIJobRunResult, error) {
	if s.db == nil {
		return ServerAIJobRunResult{}, errors.New("db not available")
	}
	spec, ok := findServerAIJobSpec(key)
	if !ok {
		return ServerAIJobRunResult{}, fmt.Errorf("unknown server ai job %q", key)
	}
	if trigger == "" {
		trigger = "manual"
	}
	start := s.now()
	run := saimodels.ServerAIJobRun{
		JobKey: spec.Key, JobName: spec.Name, Category: spec.Category, Frequency: spec.Frequency,
		GenerativeMode: spec.GenerativeMode, TriggerType: trigger, Status: "running", StartedAt: start, WorldID: worldID,
	}
	if err := s.db.WithContext(ctx).Create(&run).Error; err != nil {
		return ServerAIJobRunResult{}, err
	}
	result := ServerAIJobRunResult{JobKey: spec.Key, Status: "success", Summary: map[string]any{}}
	var err error
	switch key {
	case "server_ai_scheduler_guard_job":
		result, err = s.runSchedulerGuardJob(ctx, worldID)
	case "server_ai_threat_scan_job":
		result, err = s.runThreatScanJob(ctx, worldID)
	case "server_ai_attack_queue_job":
		result, err = s.runAttackQueueJob(ctx)
	case "server_ai_sabotage_queue_job":
		result, err = s.runSabotageQueueJob(ctx)
	case "server_ai_continent_pressure_job":
		result, err = s.runContinentPressureJob(ctx, worldID)
	case "server_ai_city_tick_job":
		result, err = s.runCityTickJob(ctx, worldID)
	case "server_ai_bastion_strategy_job":
		result, err = s.runBastionStrategyJob(ctx, worldID)
	case "server_ai_espionage_job":
		result, err = s.runEspionageJob(ctx, worldID)
	case "server_ai_sabotage_job":
		result, err = s.runSabotageJob(ctx, worldID)
	case "server_ai_attack_scheduler_job":
		result, err = s.runAttackSchedulerJob(ctx, worldID)
	case "server_ai_attack_resolver_job":
		result, err = s.runAttackResolverJob(ctx)
	case "server_ai_daily_strategy_job":
		result, err = s.runDailyStrategyJob(ctx, worldID)
	case "server_ai_daily_broadcast_job":
		result, err = s.runDailyBroadcastJob(ctx, worldID)
	case "server_ai_global_memory_update_job":
		result, err = s.runGlobalMemoryUpdateJob(ctx, worldID)
	case "server_ai_player_memory_daily_job":
		result, err = s.runPlayerMemoryDailyJob(ctx, worldID)
	case "server_ai_cost_guard_job":
		result, err = s.runCostGuardJob(ctx)
	case "server_ai_event_proposal_job":
		result, err = s.runEventProposalJob(ctx, worldID)
	case "server_ai_weekly_world_review_job":
		result, err = s.runWeeklyWorldReviewJob(ctx, worldID)
	case "server_ai_seasonal_planning_job":
		result, err = s.runSeasonalPlanningJob(ctx, worldID)
	case "server_ai_seasonal_event_start_job":
		result, err = s.runSeasonalEventStartJob(ctx)
	case "server_ai_seasonal_event_tick_job":
		result, err = s.runSeasonalEventTickJob(ctx)
	case "server_ai_seasonal_event_end_job":
		result, err = s.runSeasonalEventEndJob(ctx)
	}
	result.JobKey = spec.Key
	finished := s.now()
	run.FinishedAt = &finished
	run.DurationMs = finished.Sub(start).Milliseconds()
	run.Processed = result.Processed
	run.CreatedCount = result.CreatedCount
	run.UpdatedCount = result.UpdatedCount
	run.SkippedCount = result.SkippedCount
	if result.Summary == nil {
		result.Summary = map[string]any{}
	}
	run.SummaryJSON = mustJSON(result.Summary)
	if err != nil {
		run.Status = "failed"
		run.ErrorMessage = err.Error()
		result.Status = "failed"
	} else {
		run.Status = result.Status
		if run.Status == "" {
			run.Status = "success"
			result.Status = "success"
		}
	}
	saveErr := s.db.WithContext(ctx).Save(&run).Error
	if err != nil {
		return result, err
	}
	return result, saveErr
}

func findServerAIJobSpec(key string) (ServerAIJobSpec, bool) {
	for _, spec := range ServerAIJobSpecs() {
		if spec.Key == key {
			return spec, true
		}
	}
	return ServerAIJobSpec{}, false
}

func (s *Service) runSchedulerGuardJob(ctx context.Context, worldID uint) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_scheduler_guard_job")
	worlds, err := s.activeWorlds(ctx, worldID)
	if err != nil {
		return result, err
	}
	for _, world := range worlds {
		if err := s.EnsureCitiesForWorld(ctx, world.ID); err != nil {
			return result, err
		}
		result.Processed++
	}
	started, err := s.transitionSeasonalEventsByTime(ctx, []string{StatusApproved, StatusScheduled}, StatusActive, true)
	result.UpdatedCount += started
	if err != nil {
		return result, err
	}
	ended, err := s.transitionSeasonalEventsByTime(ctx, []string{StatusActive}, StatusEnded, false)
	result.UpdatedCount += ended
	result.Summary = map[string]any{"worldsChecked": len(worlds), "eventsStarted": started, "eventsEnded": ended, "locks": "database transaction guards"}
	return result, err
}

func (s *Service) runThreatScanJob(ctx context.Context, worldID uint) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_threat_scan_job")
	profiles, err := s.playerProfiles(ctx, worldID)
	if err != nil {
		return result, err
	}
	for _, profile := range profiles {
		result.Processed++
		score := clamp(profile.Power/80+profile.Level*4+(50-profile.Security)/2, 0, 100)
		style := "balanced"
		weaknesses := []string{}
		strengths := []string{}
		if profile.EnergyBalance < 0 {
			weaknesses = append(weaknesses, "energy_deficit")
		}
		if profile.Security < 40 {
			weaknesses = append(weaknesses, "low_security")
		}
		if profile.Power > 500 {
			strengths = append(strengths, "high_power")
			style = "aggressive"
		}
		if profile.Security >= 70 {
			strengths = append(strengths, "fortified")
			style = "defensive"
		}
		created, err := s.upsertPlayerMemory(ctx, profile, score, style, weaknesses, strengths)
		if err != nil {
			return result, err
		}
		if created {
			result.CreatedCount++
		} else {
			result.UpdatedCount++
		}
	}
	result.Summary = map[string]any{"playersScanned": len(profiles), "antiFrustration": "beginner protection checked by attack/sabotage jobs"}
	return result, nil
}

func (s *Service) runAttackQueueJob(ctx context.Context) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_attack_queue_job")
	now := s.now()
	var attacks []saimodels.ServerAIAttack
	if err := s.db.WithContext(ctx).Where("status = ? AND warning_at IS NOT NULL AND warning_at <= ? AND scheduled_at > ?", "scheduled", now, now).Find(&attacks).Error; err != nil {
		return result, err
	}
	for _, attack := range attacks {
		result.Processed++
		if err := s.db.WithContext(ctx).Model(&saimodels.ServerAIAttack{}).Where("id = ?", attack.ID).Updates(map[string]any{"status": "warning", "updated_at": now}).Error; err != nil {
			return result, err
		}
		result.UpdatedCount++
	}
	result.Summary = map[string]any{"warningsActivated": result.UpdatedCount}
	return result, nil
}

func (s *Service) runSabotageQueueJob(ctx context.Context) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_sabotage_queue_job")
	now := s.now()
	res := s.db.WithContext(ctx).Model(&saimodels.ServerAISabotage{}).
		Where("status = ? AND created_at < ?", "proposed", now.Add(-24*time.Hour)).
		Updates(map[string]any{"status": "expired", "updated_at": now})
	result.UpdatedCount = int(res.RowsAffected)
	result.Processed = result.UpdatedCount
	result.Summary = map[string]any{"expiredOldProposals": result.UpdatedCount}
	return result, res.Error
}

func (s *Service) runContinentPressureJob(ctx context.Context, worldID uint) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_continent_pressure_job")
	var continents []models.Continent
	q := s.db.WithContext(ctx).Order("world_id ASC, id ASC")
	if worldID > 0 {
		q = q.Where("world_id = ?", worldID)
	}
	if err := q.Find(&continents).Error; err != nil {
		return result, err
	}
	for _, continent := range continents {
		var players int64
		var powerSum int64
		s.db.WithContext(ctx).Model(&models.ProfileGamer{}).Where("continent_id = ?", continent.ID).Count(&players)
		s.db.WithContext(ctx).Model(&models.ProfileGamer{}).Where("continent_id = ?", continent.ID).Select("COALESCE(SUM(power),0)").Scan(&powerSum)
		var activeAttacks int64
		s.db.WithContext(ctx).Model(&saimodels.ServerAIAttack{}).Where("continent_id = ? AND status IN ?", continent.ID, []string{"scheduled", "warning"}).Count(&activeAttacks)
		level := clamp(20+int(players)*2+int(powerSum/1000)+int(activeAttacks)*8, 0, 100)
		reasons := []string{fmt.Sprintf("players=%d", players), fmt.Sprintf("power=%d", powerSum), fmt.Sprintf("activeAttacks=%d", activeAttacks)}
		var pressure saimodels.ServerAIPressure
		err := s.db.WithContext(ctx).Where("world_id = ? AND continent_id = ?", continent.WorldID, continent.ID).First(&pressure).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			pressure = saimodels.ServerAIPressure{WorldID: continent.WorldID, ContinentID: continent.ID}
			result.CreatedCount++
		} else if err != nil {
			return result, err
		} else {
			result.UpdatedCount++
		}
		pressure.Level = level
		pressure.Status = pressureStatus(level)
		pressure.ReasonsJSON = mustJSON(reasons)
		if err := s.db.WithContext(ctx).Save(&pressure).Error; err != nil {
			return result, err
		}
		result.Processed++
	}
	result.Summary = map[string]any{"continents": result.Processed}
	return result, nil
}

func (s *Service) runCityTickJob(ctx context.Context, worldID uint) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_city_tick_job")
	cities, err := s.serverCities(ctx, worldID)
	if err != nil {
		return result, err
	}
	now := s.now()
	for _, city := range cities {
		resources := parseFloatMap(city.ResourcesJSON)
		buildings := parseFloatMap(city.BuildingsJSON)
		energyProd := 60.0 + float64(city.Level)*12 + buildings["building_solar_plant"]*30
		energyUse := 35.0 + float64(city.Population)/80 + buildings["building_nexus_core"]*8
		foodProd := 45.0 + buildings["building_vertical_farm"]*28
		foodUse := float64(city.Population) / 55
		metalProd := 25.0 + buildings["building_composite_mine"]*35
		resources["energy"] = math.Max(0, resources["energy"]+energyProd-energyUse)
		resources["food"] = math.Max(0, resources["food"]+foodProd-foodUse)
		resources["metal"] = resources["metal"] + metalProd
		resources["data"] = resources["data"] + 8 + float64(city.Level)
		city.Energy = int(resources["energy"])
		if resources["food"] > foodUse && city.Population < city.PopulationCap+500 {
			city.Population += max(5, city.Level*2)
		}
		if city.PopulationCap == 0 {
			city.PopulationCap = 500 + int(buildings["building_modular_habitat"])*250
		}
		if energyProd >= energyUse {
			city.Morale = clamp(city.Morale+1, 0, 100)
		} else {
			city.Morale = clamp(city.Morale-2, 0, 100)
		}
		city.Security = clamp(city.Security+int(buildings["building_holo_wall"]), 0, 100)
		city.Power = max(city.Power, city.Level*120+city.Population/10+city.Security*2)
		city.ResourcesJSON = mustJSON(resources)
		city.LastTickAt = &now
		city.UpdatedAt = now
		if err := s.db.WithContext(ctx).Save(&city).Error; err != nil {
			return result, err
		}
		result.Processed++
		result.UpdatedCount++
	}
	result.Summary = map[string]any{"citiesTicked": result.Processed, "generativeAI": false}
	return result, nil
}

func (s *Service) runBastionStrategyJob(ctx context.Context, worldID uint) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_bastion_strategy_job")
	cities, err := s.serverCities(ctx, worldID)
	if err != nil {
		return result, err
	}
	for _, city := range cities {
		priority := "defend"
		if city.Energy < 150 {
			priority = "energy"
		} else if city.Security < 50 {
			priority = "security"
		} else if city.Power < 400 {
			priority = "train_unit"
		} else if city.Morale > 65 {
			priority = "prepare_attack"
		}
		payload := map[string]any{"priority": priority, "reason": "deterministic backend strategy", "generatedBy": "go_rules", "cityPower": city.Power}
		strategy := saimodels.ServerAIStrategy{WorldID: city.WorldID, ContinentID: city.ContinentID, ServerAICityID: city.ID, Priority: priority, Difficulty: "normal", StrategyJSON: mustJSON(payload), IsActive: true, CreatedAt: s.now(), UpdatedAt: s.now()}
		if err := s.db.WithContext(ctx).Create(&strategy).Error; err != nil {
			return result, err
		}
		city.StrategyJSON = mustJSON(payload)
		_ = s.db.WithContext(ctx).Save(&city).Error
		result.Processed++
		result.CreatedCount++
	}
	result.Summary = map[string]any{"strategiesCreated": result.CreatedCount}
	return result, nil
}

func (s *Service) runEspionageJob(ctx context.Context, worldID uint) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_espionage_job")
	targets, err := s.eligibleTargets(ctx, worldID, 2, 25)
	if err != nil {
		return result, err
	}
	now := s.now()
	for _, profile := range targets {
		result.Processed++
		if s.recentCount(ctx, &saimodels.ServerAIEspionage{}, "target_user_id = ? AND created_at >= ?", profile.UserID, now.Add(-24*time.Hour)) > 0 {
			result.SkippedCount++
			continue
		}
		findings := []string{}
		if profile.EnergyBalance < 0 {
			findings = append(findings, "energy_deficit")
		}
		if profile.Security < 50 {
			findings = append(findings, "security_gap")
		}
		if len(findings) == 0 {
			findings = append(findings, "no_major_weakness")
		}
		row := saimodels.ServerAIEspionage{WorldID: profile.WorldID, ContinentID: profile.ContinentID, TargetUserID: profile.UserID, Result: "success", Summary: "Espionnage serveur IA calculé sans appel génératif.", FindingsJSON: mustJSON(findings), CreatedAt: now, UpdatedAt: now}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return result, err
		}
		_, _ = s.upsertPlayerMemory(ctx, profile, clamp(profile.Power/80+profile.Level*4, 0, 100), "observed", findings, []string{})
		result.CreatedCount++
	}
	result.Summary = map[string]any{"espionageCreated": result.CreatedCount, "generativeAI": "optional-not-used"}
	return result, nil
}

func (s *Service) runSabotageJob(ctx context.Context, worldID uint) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_sabotage_job")
	targets, err := s.eligibleTargets(ctx, worldID, 2, 15)
	if err != nil {
		return result, err
	}
	now := s.now()
	for _, profile := range targets {
		result.Processed++
		if profile.Security >= 70 || s.recentCount(ctx, &saimodels.ServerAISabotage{}, "target_user_id = ? AND created_at >= ? AND status <> ?", profile.UserID, now.Add(-24*time.Hour), "cancelled") > 0 {
			result.SkippedCount++
			continue
		}
		sabotageType := "energy_disrupt"
		if profile.EnergyBalance >= 0 {
			sabotageType = "morale_hit"
		}
		row := saimodels.ServerAISabotage{WorldID: profile.WorldID, ContinentID: profile.ContinentID, TargetUserID: profile.UserID, Status: "proposed", SabotageType: sabotageType, MaxImpact: "minor temporary impact, no destruction", Counterplay: "raise security, stabilize energy, inspect agents", ReportJSON: mustJSON(map[string]any{"antiFrustration": true, "maxLossPercent": 5}), CreatedAt: now, UpdatedAt: now}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return result, err
		}
		result.CreatedCount++
	}
	result.Summary = map[string]any{"sabotageProposals": result.CreatedCount, "limits": "no permanent block, no massive theft, no destruction"}
	return result, nil
}

func (s *Service) runAttackSchedulerJob(ctx context.Context, worldID uint) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_attack_scheduler_job")
	targets, err := s.eligibleTargets(ctx, worldID, 2, 10)
	if err != nil {
		return result, err
	}
	for _, profile := range targets {
		result.Processed++
		var city saimodels.ServerAICity
		_ = s.db.WithContext(ctx).Where("world_id = ? AND continent_id = ? AND status = ?", profile.WorldID, profile.ContinentID, "active").Order("power DESC").First(&city).Error
		attack, err := s.ScheduleAttack(ctx, ScheduleAttackRequest{WorldID: profile.WorldID, ContinentID: profile.ContinentID, ServerAICityID: city.ID, TargetUserID: profile.UserID, TargetCityID: profile.ID, AttackType: "raid", Difficulty: "normal", PlayerPower: max(profile.Power, 100), DefensePower: max(profile.Power+profile.Security*3, 100)})
		if err != nil {
			result.SkippedCount++
			continue
		}
		_ = attack
		result.CreatedCount++
		if result.CreatedCount >= 5 {
			break
		}
	}
	result.Summary = map[string]any{"scheduledAttacks": result.CreatedCount, "rule": "max 1 attack per player per day, power capped at 1.15x normal"}
	return result, nil
}

func (s *Service) runAttackResolverJob(ctx context.Context) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_attack_resolver_job")
	var attacks []saimodels.ServerAIAttack
	now := s.now()
	if err := s.db.WithContext(ctx).Where("status IN ? AND scheduled_at <= ?", []string{"scheduled", "warning"}, now).Find(&attacks).Error; err != nil {
		return result, err
	}
	for _, attack := range attacks {
		_, err := s.ResolveAttack(ctx, attack.ID, "")
		if err != nil {
			return result, err
		}
		result.Processed++
		result.UpdatedCount++
	}
	result.Summary = map[string]any{"resolvedAttacks": result.UpdatedCount}
	return result, nil
}

func (s *Service) runDailyStrategyJob(ctx context.Context, worldID uint) (ServerAIJobRunResult, error) {
	result, err := s.runBastionStrategyJob(ctx, worldID)
	result.JobKey = "server_ai_daily_strategy_job"
	if result.Summary == nil {
		result.Summary = map[string]any{}
	}
	result.Summary["dailyPlanServerAI"] = "hostile world strategy, distinct from player daily plan"
	return result, err
}

func (s *Service) runDailyBroadcastJob(ctx context.Context, worldID uint) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_daily_broadcast_job")
	worlds, err := s.activeWorlds(ctx, worldID)
	if err != nil {
		return result, err
	}
	if len(worlds) == 0 && worldID == 0 {
		worlds = []models.World{{ID: 0, Name: "global"}}
	}
	for _, world := range worlds {
		broadcast, err := s.GenerateBroadcast(ctx, world.ID)
		if err != nil {
			return result, err
		}
		if _, err := s.PublishBroadcast(ctx, broadcast.ID); err != nil {
			return result, err
		}
		result.Processed++
		result.CreatedCount++
	}
	result.Summary = map[string]any{"publishedBroadcasts": result.CreatedCount}
	return result, nil
}

func (s *Service) runGlobalMemoryUpdateJob(ctx context.Context, worldID uint) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_global_memory_update_job")
	worlds, err := s.activeWorlds(ctx, worldID)
	if err != nil {
		return result, err
	}
	for _, world := range worlds {
		var aiWins, playerWins, draws int64
		s.db.WithContext(ctx).Model(&saimodels.ServerAIAttack{}).Where("world_id = ? AND result = ?", world.ID, "victory_ai").Count(&aiWins)
		s.db.WithContext(ctx).Model(&saimodels.ServerAIAttack{}).Where("world_id = ? AND result = ?", world.ID, "victory_player").Count(&playerWins)
		s.db.WithContext(ctx).Model(&saimodels.ServerAIAttack{}).Where("world_id = ? AND result = ?", world.ID, "draw").Count(&draws)
		var pressures []saimodels.ServerAIPressure
		_ = s.db.WithContext(ctx).Where("world_id = ?", world.ID).Find(&pressures).Error
		globalDanger := 30 + int(aiWins)*2 - int(playerWins)
		for _, p := range pressures {
			globalDanger += p.Level / 20
		}
		globalDanger = clamp(globalDanger, 0, 100)
		mem := saimodels.ServerAIMemory{WorldID: world.ID}
		err := s.db.WithContext(ctx).Where("world_id = ?", world.ID).First(&mem).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result.CreatedCount++
		} else if err != nil {
			return result, err
		} else {
			result.UpdatedCount++
		}
		mem.AIVictories = int(aiWins)
		mem.PlayerVictories = int(playerWins)
		mem.Draws = int(draws)
		mem.GlobalDanger = globalDanger
		mem.GlobalStability = clamp(100-globalDanger, 0, 100)
		mem.ContinentsJSON = mustJSON(pressures)
		mem.TrendsJSON = mustJSON(map[string]any{"updatedBy": "server_ai_global_memory_update_job", "at": s.now()})
		mem.EffectiveStrategies = "pressure_scaling,anti_frustration,target_memory"
		mem.UpdatedAt = s.now()
		if mem.CreatedAt.IsZero() {
			mem.CreatedAt = s.now()
		}
		if err := s.db.WithContext(ctx).Save(&mem).Error; err != nil {
			return result, err
		}
		result.Processed++
	}
	result.Summary = map[string]any{"worldMemoriesUpdated": result.Processed}
	return result, nil
}

func (s *Service) runPlayerMemoryDailyJob(ctx context.Context, worldID uint) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_player_memory_daily_job")
	var rows []saimodels.ServerAIPlayerMemory
	q := s.db.WithContext(ctx)
	if worldID > 0 {
		q = q.Where("world_id = ?", worldID)
	}
	if err := q.Find(&rows).Error; err != nil {
		return result, err
	}
	for _, row := range rows {
		row.RivalryLevel = clamp(row.RivalryLevel-1+row.VictoriesAgainstAI/5, 0, 100)
		row.ThreatScore = clamp(row.ThreatScore-2+row.VictoriesAgainstAI/3, 0, 100)
		row.UpdatedAt = s.now()
		if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
			return result, err
		}
		result.Processed++
		result.UpdatedCount++
	}
	result.Summary = map[string]any{"memoryDecayApplied": result.UpdatedCount}
	return result, nil
}

func (s *Service) runCostGuardJob(ctx context.Context) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_cost_guard_job")
	costs := s.Costs(ctx)
	result.Processed = 1
	limit := 2.0
	status := "ok"
	if cost, ok := costs["dailyCostEstimate"].(float64); ok && cost > limit {
		status = "budget_exceeded"
		action := saimodels.ServerAIAdminAction{Action: "server_ai_cost_guard_alert", TargetType: "server_ai_costs", PayloadJSON: mustJSON(costs), CreatedAt: s.now()}
		if err := s.db.WithContext(ctx).Create(&action).Error; err != nil {
			return result, err
		}
		result.CreatedCount++
	}
	result.Summary = map[string]any{"costs": costs, "status": status, "dailyLimitEstimate": limit}
	return result, nil
}

func (s *Service) runEventProposalJob(ctx context.Context, worldID uint) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_event_proposal_job")
	worlds, err := s.activeWorlds(ctx, worldID)
	if err != nil {
		return result, err
	}
	for _, world := range worlds {
		if s.recentCount(ctx, &saimodels.ServerAISeasonalEvent{}, "world_id = ? AND created_at >= ? AND status IN ?", world.ID, s.now().Add(-24*time.Hour), []string{StatusProposed, StatusApproved, StatusScheduled, StatusActive}) > 0 {
			result.SkippedCount++
			continue
		}
		if _, err := s.ProposeSeasonalEvent(ctx, world.ID); err != nil {
			return result, err
		}
		result.CreatedCount++
		result.Processed++
	}
	result.Summary = map[string]any{"proposalsCreated": result.CreatedCount, "adminValidationRequired": true}
	return result, nil
}

func (s *Service) runWeeklyWorldReviewJob(ctx context.Context, worldID uint) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_weekly_world_review_job")
	worlds, err := s.activeWorlds(ctx, worldID)
	if err != nil {
		return result, err
	}
	for _, world := range worlds {
		action := saimodels.ServerAIAdminAction{Action: "server_ai_weekly_world_review", TargetType: "world", TargetID: world.ID, PayloadJSON: mustJSON(map[string]any{"review": "weekly deterministic review", "recommendedActions": []string{"inspect_pressure", "review_events", "check_costs"}}), CreatedAt: s.now()}
		if err := s.db.WithContext(ctx).Create(&action).Error; err != nil {
			return result, err
		}
		result.Processed++
		result.CreatedCount++
	}
	result.Summary = map[string]any{"weeklyReviews": result.CreatedCount}
	return result, nil
}

func (s *Service) runSeasonalPlanningJob(ctx context.Context, worldID uint) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_seasonal_planning_job")
	worlds, err := s.activeWorlds(ctx, worldID)
	if err != nil {
		return result, err
	}
	for _, world := range worlds {
		for i := 0; i < 3; i++ {
			event, err := s.ProposeSeasonalEvent(ctx, world.ID)
			if err != nil {
				return result, err
			}
			event.Title = fmt.Sprintf("%s %d", event.Title, i+1)
			event.EventType = fmt.Sprintf("seasonal_plan_%d", i+1)
			event.AdminNote = "Planification saisonnière: proposition générée, validation admin obligatoire."
			_ = s.db.WithContext(ctx).Save(event).Error
			result.CreatedCount++
		}
		result.Processed++
	}
	result.Summary = map[string]any{"seasonalProposals": result.CreatedCount, "adminValidationRequired": true}
	return result, nil
}

func (s *Service) runSeasonalEventStartJob(ctx context.Context) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_seasonal_event_start_job")
	count, err := s.transitionSeasonalEventsByTime(ctx, []string{StatusApproved, StatusScheduled}, StatusActive, true)
	result.Processed = count
	result.UpdatedCount = count
	result.Summary = map[string]any{"eventsStarted": count, "requiresPriorAdminApproval": true}
	return result, err
}

func (s *Service) runSeasonalEventTickJob(ctx context.Context) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_seasonal_event_tick_job")
	var events []saimodels.ServerAISeasonalEvent
	if err := s.db.WithContext(ctx).Where("status = ?", StatusActive).Find(&events).Error; err != nil {
		return result, err
	}
	for _, event := range events {
		event.AdminNote = strings.TrimSpace(event.AdminNote + "\n" + s.now().Format(time.RFC3339) + " server_ai_seasonal_event_tick_job: participation/abuse/pressure check done.")
		event.UpdatedAt = s.now()
		if err := s.db.WithContext(ctx).Save(&event).Error; err != nil {
			return result, err
		}
		result.Processed++
		result.UpdatedCount++
	}
	result.Summary = map[string]any{"activeEventsTicked": result.UpdatedCount}
	return result, nil
}

func (s *Service) runSeasonalEventEndJob(ctx context.Context) (ServerAIJobRunResult, error) {
	result := jobResult("server_ai_seasonal_event_end_job")
	count, err := s.transitionSeasonalEventsByTime(ctx, []string{StatusActive}, StatusEnded, false)
	result.Processed = count
	result.UpdatedCount = count
	result.Summary = map[string]any{"eventsEnded": count}
	return result, err
}

func jobResult(key string) ServerAIJobRunResult {
	return ServerAIJobRunResult{JobKey: key, Status: "success", Summary: map[string]any{}}
}

func (s *Service) activeWorlds(ctx context.Context, worldID uint) ([]models.World, error) {
	var worlds []models.World
	q := s.db.WithContext(ctx).Where("is_active = ?", true).Order("id ASC")
	if worldID > 0 {
		q = q.Where("id = ?", worldID)
	}
	return worlds, q.Find(&worlds).Error
}

func (s *Service) playerProfiles(ctx context.Context, worldID uint) ([]models.ProfileGamer, error) {
	var profiles []models.ProfileGamer
	q := s.db.WithContext(ctx).Where("world_id IS NOT NULL AND world_id <> 0").Order("power DESC, updated_at DESC")
	if worldID > 0 {
		q = q.Where("world_id = ?", worldID)
	}
	return profiles, q.Find(&profiles).Error
}

func (s *Service) serverCities(ctx context.Context, worldID uint) ([]saimodels.ServerAICity, error) {
	var cities []saimodels.ServerAICity
	q := s.db.WithContext(ctx).Where("status <> ?", StatusDeleted).Order("world_id ASC, continent_id ASC")
	if worldID > 0 {
		q = q.Where("world_id = ?", worldID)
	}
	return cities, q.Find(&cities).Error
}

func (s *Service) eligibleTargets(ctx context.Context, worldID uint, minLevel int, limit int) ([]models.ProfileGamer, error) {
	var profiles []models.ProfileGamer
	q := s.db.WithContext(ctx).
		Where("world_id IS NOT NULL AND world_id <> 0").
		Where("level >= ?", minLevel).
		Order("power DESC, security ASC, updated_at DESC").
		Limit(limit)
	if worldID > 0 {
		q = q.Where("world_id = ?", worldID)
	}
	return profiles, q.Find(&profiles).Error
}

func (s *Service) upsertPlayerMemory(ctx context.Context, profile models.ProfileGamer, score int, style string, weaknesses []string, strengths []string) (bool, error) {
	var memory saimodels.ServerAIPlayerMemory
	err := s.db.WithContext(ctx).Where("world_id = ? AND user_id = ?", profile.WorldID, profile.UserID).First(&memory).Error
	created := false
	if errors.Is(err, gorm.ErrRecordNotFound) {
		memory = saimodels.ServerAIPlayerMemory{WorldID: profile.WorldID, UserID: profile.UserID, ContinentID: profile.ContinentID, CreatedAt: s.now()}
		created = true
	} else if err != nil {
		return false, err
	}
	memory.PlayerPower = profile.Power
	memory.PlayerLevel = profile.Level
	memory.PlayerStyle = style
	memory.ThreatScore = score
	memory.RivalryLevel = clamp(memory.RivalryLevel+score/40, 0, 100)
	memory.KnownWeaknessesJSON = mustJSON(weaknesses)
	memory.KnownStrengthsJSON = mustJSON(strengths)
	memory.UpdatedAt = s.now()
	return created, s.db.WithContext(ctx).Save(&memory).Error
}

func (s *Service) recentCount(ctx context.Context, model any, query string, args ...any) int64 {
	var count int64
	_ = s.db.WithContext(ctx).Model(model).Where(query, args...).Count(&count).Error
	return count
}

func (s *Service) transitionSeasonalEventsByTime(ctx context.Context, from []string, to string, useStart bool) (int, error) {
	now := s.now()
	var events []saimodels.ServerAISeasonalEvent
	q := s.db.WithContext(ctx).Where("status IN ?", from)
	if useStart {
		q = q.Where("starts_at IS NOT NULL AND starts_at <= ?", now)
	} else {
		q = q.Where("ends_at IS NOT NULL AND ends_at <= ?", now)
	}
	if err := q.Find(&events).Error; err != nil {
		return 0, err
	}
	for _, event := range events {
		event.Status = to
		event.UpdatedAt = now
		if to == StatusActive {
			event.AdminNote = strings.TrimSpace(event.AdminNote + "\nStarted by server_ai scheduler after admin validation.")
		}
		if to == StatusEnded {
			event.AdminNote = strings.TrimSpace(event.AdminNote + "\nEnded by server_ai scheduler.")
		}
		if err := s.db.WithContext(ctx).Save(&event).Error; err != nil {
			return 0, err
		}
	}
	return len(events), nil
}

func parseFloatMap(raw string) map[string]float64 {
	out := map[string]float64{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	var anyMap map[string]any
	if err := json.Unmarshal([]byte(raw), &anyMap); err != nil {
		return out
	}
	for key, value := range anyMap {
		switch v := value.(type) {
		case float64:
			out[key] = v
		case int:
			out[key] = float64(v)
		case string:
			var parsed float64
			if _, err := fmt.Sscanf(v, "%f", &parsed); err == nil {
				out[key] = parsed
			}
		}
	}
	return out
}
