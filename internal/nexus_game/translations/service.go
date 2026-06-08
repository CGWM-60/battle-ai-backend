package translations

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"cgwm/battle/internal/models"
	tribunaladapters "cgwm/battle/internal/nexus_tribunal/adapters"
	"gorm.io/gorm"
)

// TranslationEntry représente une traduction ligne par ligne.
type TranslationEntry struct {
	Key       string    `json:"key"`
	Locale    string    `json:"locale"`
	Value     string    `json:"value"`
	Domain    string    `json:"domain"` // ex: nexus_game, nexus_game_city, common...
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"`
}

// TranslationService est le contrat pour le module traduction serveur.
// Le serveur reste la source de vérité (AGENTS.md).
type TranslationService interface {
	// GetTranslations retourne les traductions pour une langue (optionnellement filtrées par domaines).
	GetTranslations(ctx context.Context, lang string, domains []string) (map[string]string, error)

	// UpsertBatch permet à l'admin d'importer/mettre à jour par lot.
	UpsertBatch(ctx context.Context, entries []TranslationEntry) error

	// ListMissingKeys renvoie les clés utilisées dans le code mais absentes pour une langue.
	ListMissingKeys(ctx context.Context, lang string) ([]string, error)

	// GetAvailableLanguages liste les langues supportées.
	GetAvailableLanguages(ctx context.Context) ([]string, error)

	// GetDomains liste les domaines disponibles.
	GetDomains(ctx context.Context) ([]models.TranslationDomain, error)

	// GetDomainTranslations retourne les traductions pour un domaine spécifique.
	GetDomainTranslations(ctx context.Context, domain string, lang string) (map[string]string, error)

	// SetUserLocale met à jour la préférence de langue de l'utilisateur.
	SetUserLocale(ctx context.Context, userID uint, locale string) error

	// LogMissingKey enregistre une clé manquante signalée par le client.
	LogMissingKey(ctx context.Context, key string, locale string) error

	// Admin methods for POINT 03
	GetAllDomains(ctx context.Context) ([]models.TranslationDomain, error)
	CreateDomain(ctx context.Context, d *models.TranslationDomain) error
	GetAllKeys(ctx context.Context) ([]models.TranslationKey, error)
	CreateKey(ctx context.Context, k *models.TranslationKey) error
	UpdateKey(ctx context.Context, id uint, k *models.TranslationKey) error
	DeleteKey(ctx context.Context, id uint) error
	GetAllValues(ctx context.Context) ([]models.TranslationValue, error)
	UpdateValue(ctx context.Context, id uint, v *models.TranslationValue) error
	PreviewImport(ctx context.Context, rows []models.TranslationImportRow) ([]models.TranslationImportRow, error)
	CommitImport(ctx context.Context, importID uint) error
	CommitImportRows(ctx context.Context, rows []models.TranslationImportRow, fileName string) (*models.TranslationImport, error)
	GetImports(ctx context.Context) ([]models.TranslationImport, error)
	GetImportByID(ctx context.Context, id uint) (*models.TranslationImport, error)
	GetMissing(ctx context.Context) ([]models.TranslationMissingLog, error)
	ExportTranslations(ctx context.Context, lang string) (map[string]string, error)
	BatchUpdate(ctx context.Context, entries []TranslationEntry) error
	GetAITranslationProviders(ctx context.Context) ([]TranslationAIProviderStatus, error)
	AITranslateMissing(ctx context.Context, req AITranslateMissingRequest) (*AITranslateMissingResult, error)
}

// dbTranslationService implémente TranslationService avec GORM.
type dbTranslationService struct {
	db *gorm.DB
}

type TranslationAIProviderStatus struct {
	ProviderType string `json:"providerType"`
	DisplayName  string `json:"displayName"`
	DefaultModel string `json:"defaultModel"`
	Configured   bool   `json:"configured"`
}

type AITranslateMissingRequest struct {
	TargetLocale  string   `json:"targetLocale"`
	SourceLocale  string   `json:"sourceLocale"`
	Provider      string   `json:"provider"`
	Model         string   `json:"model"`
	APIKey        string   `json:"apiKey"`
	LocalEndpoint string   `json:"localEndpoint"`
	Limit         int      `json:"limit"`
	Domains       []string `json:"domains"`
	Keys          []string `json:"keys"`
}

type AITranslateMissingResult struct {
	Provider     string                  `json:"provider"`
	Model        string                  `json:"model"`
	SourceLocale string                  `json:"sourceLocale"`
	TargetLocale string                  `json:"targetLocale"`
	Translated   int                     `json:"translated"`
	Errors       int                     `json:"errors"`
	Items        []AITranslatedValueItem `json:"items"`
}

type AITranslatedValueItem struct {
	Domain string `json:"domain"`
	Key    string `json:"key"`
	Source string `json:"source"`
	Value  string `json:"value"`
	Error  string `json:"error,omitempty"`
}

func NewTranslationService(db *gorm.DB) TranslationService {
	return &dbTranslationService{db: db}
}

func (s *dbTranslationService) GetTranslations(ctx context.Context, lang string, domains []string) (map[string]string, error) {
	var results []struct {
		TranslationKey string
		Value          string
	}

	query := s.db.Table("nexus_translation_values as v").
		Select("k.`key` as translation_key, v.value as value").
		Joins("JOIN nexus_translation_keys as k ON k.id = v.key_id").
		Joins("JOIN nexus_translation_domains as d ON d.id = k.domain_id").
		Where("v.locale = ?", lang)

	if len(domains) > 0 {
		query = query.Where("d.code IN ?", domains)
	}

	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}

	m := make(map[string]string, len(results))
	for _, r := range results {
		m[r.TranslationKey] = r.Value
	}

	// Fallback sur FR si certaines clés manquent pour la langue demandée (simple implémentation)
	if lang != "fr" {
		var frResults []struct {
			TranslationKey string
			Value          string
		}
		frQuery := s.db.Table("nexus_translation_values as v").
			Select("k.`key` as translation_key, v.value as value").
			Joins("JOIN nexus_translation_keys as k ON k.id = v.key_id").
			Joins("JOIN nexus_translation_domains as d ON d.id = k.domain_id").
			Where("v.locale = ?", "fr")
		if len(domains) > 0 {
			frQuery = frQuery.Where("d.code IN ?", domains)
		}
		_ = frQuery.Find(&frResults).Error
		for _, r := range frResults {
			if _, ok := m[r.TranslationKey]; !ok {
				m[r.TranslationKey] = r.Value
			}
		}
	}

	if len(m) == 0 {
		for key, value := range DefaultFrenchFallback {
			m[key] = value
		}
	}

	return m, nil
}

func (s *dbTranslationService) GetDomains(ctx context.Context) ([]models.TranslationDomain, error) {
	var domains []models.TranslationDomain
	if err := s.db.Where("deleted_at IS NULL").Order("code").Find(&domains).Error; err != nil {
		return nil, err
	}
	return domains, nil
}

func (s *dbTranslationService) GetDomainTranslations(ctx context.Context, domain string, lang string) (map[string]string, error) {
	var results []struct {
		TranslationKey string
		Value          string
	}

	query := s.db.Table("nexus_translation_values as v").
		Select("k.`key` as translation_key, v.value as value").
		Joins("JOIN nexus_translation_keys as k ON k.id = v.key_id").
		Joins("JOIN nexus_translation_domains as d ON d.id = k.domain_id").
		Where("d.code = ? AND v.locale = ?", domain, lang)

	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}

	m := make(map[string]string, len(results))
	for _, r := range results {
		m[r.TranslationKey] = r.Value
	}
	if len(m) == 0 && domain == "nexus_game" {
		for key, value := range DefaultFrenchFallback {
			if strings.HasPrefix(key, "nexus_game.") {
				m[key] = value
			}
		}
	}
	return m, nil
}

func (s *dbTranslationService) SetUserLocale(ctx context.Context, userID uint, locale string) error {
	pref := models.UserLocalePreference{
		UserID: userID,
		Locale: locale,
	}
	return s.db.Where(models.UserLocalePreference{UserID: userID}).
		Assign(pref).
		FirstOrCreate(&pref).Error
}

func (s *dbTranslationService) LogMissingKey(ctx context.Context, key string, locale string) error {
	var log models.TranslationMissingLog
	err := s.db.Where("`key` = ? AND locale = ?", key, locale).First(&log).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == gorm.ErrRecordNotFound {
		log = models.TranslationMissingLog{Key: key, Locale: locale, Count: 1}
		return s.db.Create(&log).Error
	}
	log.Count++
	return s.db.Save(&log).Error
}

func (s *dbTranslationService) UpsertBatch(ctx context.Context, entries []TranslationEntry) error {
	// Implémentation basique pour le point 02 (batch sera plus complet plus tard)
	for _, e := range entries {
		// Trouver ou créer le domaine
		var domain models.TranslationDomain
		if err := s.db.Where("code = ?", e.Domain).FirstOrCreate(&domain, models.TranslationDomain{Code: e.Domain, Name: e.Domain}).Error; err != nil {
			return err
		}

		// Trouver ou créer la clé
		var key models.TranslationKey
		if err := s.db.Where("domain_id = ? AND `key` = ?", domain.ID, e.Key).FirstOrCreate(&key, models.TranslationKey{DomainID: domain.ID, Key: e.Key}).Error; err != nil {
			return err
		}

		// Upsert la valeur
		val := models.TranslationValue{
			KeyID:  key.ID,
			Locale: e.Locale,
			Value:  e.Value,
		}
		if err := s.db.Where("key_id = ? AND locale = ?", key.ID, e.Locale).
			Assign(val).
			FirstOrCreate(&val).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *dbTranslationService) ListMissingKeys(ctx context.Context, lang string) ([]string, error) {
	var logs []models.TranslationMissingLog
	if err := s.db.Where("locale = ?", lang).Find(&logs).Error; err != nil {
		return nil, err
	}
	keys := make([]string, len(logs))
	for i, l := range logs {
		keys[i] = l.Key
	}
	return keys, nil
}

func (s *dbTranslationService) GetAvailableLanguages(ctx context.Context) ([]string, error) {
	var langs []string
	if err := s.db.Model(&models.TranslationValue{}).
		Distinct().
		Pluck("locale", &langs).Error; err != nil {
		return nil, err
	}
	return langs, nil
}

// DefaultFrenchFallback contient un minimum vital pour que l'app ne soit jamais vide.
// Le vrai contenu vient de la DB / admin.
var DefaultFrenchFallback = map[string]string{
	"battle.action.judge": "Lancer le jugement",
	"battle.action.next_round": "Round suivant",
	"battle.action.reset": "Réinitialiser le battle",
	"battle.action.reset_parameters": "Réinitialiser les paramètres",
	"battle.action.start": "Démarrer le battle",
	"battle.action.view_result": "Voir le résultat",
	"battle.config.display_mode": "Mode d’affichage",
	"battle.config.multi_round_hint": "Débat en plusieurs rounds avec {seconds} secondes par round.",
	"battle.config.one_round_hint": "Débat en un seul round.",
	"battle.config.parameters_title": "Paramètres du battle",
	"battle.config.protocol_initialized": "Protocole de battle initialisé.",
	"battle.config.public_vote": "Vote public",
	"battle.config.quest_prefill_notice": "Le sujet ci-dessous est prérempli depuis cette quête. Si tu modifies le texte du débat, le battle repassera en sujet libre.",
	"battle.config.quest_synced_helper": "Sujet synchronisé avec la quête sélectionnée.",
	"battle.config.quest_topic_hint": "Sujet de quête{questTitle}",
	"battle.config.round_count": "Nombre de rounds",
	"battle.config.round_duration": "Durée d’un round",
	"battle.config.title": "Configurer un battle IA",
	"battle.config.topic_hint": "Décris le sujet du duel IA.",
	"battle.config.topic_label": "Sujet du débat",
	"battle.display.split": "Vue séparée",
	"battle.display.timeline": "Timeline",
	"battle.display.vs": "VS",
	"battle.fighter.ia_one": "IA 1",
	"battle.fighter.ia_two": "IA 2",
	"battle.judgement.arbitration": "Arbitrage",
	"battle.judgement.backend_verdict": "Verdict backend",
	"battle.judgement.current_score": "Score actuel",
	"battle.judgement.default_judge": "Juge IA",
	"battle.judgement.messages_analyzed": "Messages analysés",
	"battle.judgement.no_reason": "Le backend a renvoyé les scores, mais pas d’argumentaire textuel.",
	"battle.judgement.not_started_hint": "Le jugement sera disponible après le lancement du battle.",
	"battle.judgement.running": "Jugement en cours...",
	"battle.judgement.start_backend": "Démarrer le jugement backend",
	"battle.judgement.title": "Jugement",
	"battle.judgement.verdict_hint": "{judge} donne {scoreOne} contre {scoreTwo}.",
	"battle.judgement.why_ia": "Pourquoi {name} ?",
	"battle.judgement.why_judge": "Pourquoi ce jugement ?",
	"battle.live.next_round": "Round suivant",
	"battle.live.previous_round": "Round précédent",
	"battle.live.round": "Round {round}",
	"battle.live.round_short": "R{round}",
	"battle.live.split_title": "Vue split",
	"battle.live.title": "Battle IA en direct",
	"battle.quest.level_free": "Niveau libre",
	"battle.quest.theme_free": "Thème libre",
	"battle.quests.choose_title": "Choisir une quête de battle",
	"battle.quests.empty_message": "Aucune quête publiée n’est disponible pour le battle IA.",
	"battle.quests.empty_title": "Aucune quête disponible",
	"battle.quests.filters": "Filtres",
	"battle.quests.found_count": "{count} quête(s) trouvée(s)",
	"battle.quests.no_result_message": "Aucune quête ne correspond à ces filtres.",
	"battle.quests.no_result_title": "Aucun résultat",
	"battle.quests.search_hint": "Rechercher une quête, un thème ou une mission.",
	"battle.quests.status": "Statut",
	"battle.quests.title": "Quêtes IA Battle",
	"battle.reset.content": "Le battle en cours sera remis à zéro.",
	"battle.reset.snackbar": "Battle réinitialisé.",
	"battle.reset.title": "Réinitialiser le battle ?",
	"battle.result.end_title": "Battle terminé",
	"battle.result.final_score": "Score final : {score}",
	"battle.result.hybrid_score": "Score hybride : {scoreOne} / {scoreTwo}",
	"battle.result.judge_score": "Score de {judge} : {scoreOne} / {scoreTwo}",
	"battle.result.new_clash": "Nouveau clash",
	"battle.result.public_score": "Score public : {scoreOne} / {scoreTwo}",
	"battle.result.public_vote": "Vote public",
	"battle.result.random_judge": "Juge aléatoire",
	"battle.result.refuted_arguments": "Arguments réfutés",
	"battle.result.share": "Partager",
	"battle.result.speeches": "Prises de parole",
	"battle.result.stats_title": "Statistiques du battle",
	"battle.result.title": "Résultat",
	"battle.result.user_votes": "Votes : {votesOne} / {votesTwo}",
	"battle.result.winner_badge": "Vainqueur",
	"battle.status.running": "En cours",
	"battle.stream.chunks": "{count} chunk(s)",
	"battle.stream.connected_at": "Connecté à {time}",
	"battle.stream.events": "{count} événement(s)",
	"battle.vote.required_for_round": "Vote requis pour le round {round}",
	"battle.vote.required_this_round": "Vote requis sur ce round",
	"battle.vote.saved": "Vote enregistré",
	"battle_ia.app.title": "IA Battle",
	"battle_ia.create_battle": "Créer un battle",
	"battle_ia.landing.archives.badge": "Archives",
	"battle_ia.landing.archives.cta": "Voir les archives",
	"battle_ia.landing.archives.desc": "Revoir les battles terminés, les scores et les replays publics.",
	"battle_ia.landing.archives.title": "Archives",
	"battle_ia.landing.create.badge": "Création",
	"battle_ia.landing.create.cta": "Créer un battle",
	"battle_ia.landing.create.desc": "Préparer un duel entre deux profils IA avec vote et jugement.",
	"battle_ia.landing.create.title": "Créer",
	"battle_ia.landing.live.badge": "Live",
	"battle_ia.landing.live.cta": "Voir le live",
	"battle_ia.landing.live.desc": "Suivre les battles publics en direct et rejoindre l’arène.",
	"battle_ia.landing.live.title": "Battle live",
	"battle_ia.landing.quests.badge": "Quêtes",
	"battle_ia.landing.quests.cta": "Choisir une quête",
	"battle_ia.landing.quests.desc": "Démarrer un battle depuis une quête RP publiée.",
	"battle_ia.landing.quests.title": "Quêtes",
	"common.app.title": "NEXUS GAMES",
	"common.badge.local": "Local",
	"common.badge.secured": "Sécurisé",
	"common.button.abandon": "Abandonner",
	"common.button.back": "Retour",
	"common.button.cancel": "Annuler",
	"common.button.change": "Changer",
	"common.button.clear": "Effacer",
	"common.button.confirm": "Confirmer",
	"common.button.create": "Créer",
	"common.button.delete": "Supprimer",
	"common.button.download": "Télécharger",
	"common.button.edit": "Modifier",
	"common.button.pause": "Pause",
	"common.button.quit": "Quitter",
	"common.button.refresh": "Rafraîchir",
	"common.button.reload": "Recharger",
	"common.button.restart": "Redémarrer",
	"common.button.resume": "Reprendre",
	"common.button.save": "Enregistrer",
	"common.button.send": "Envoyer",
	"common.button.start": "Démarrer",
	"common.button.stop": "Arrêter",
	"common.button.test": "Tester",
	"common.button.update": "Mettre à jour",
	"common.button.use": "Utiliser",
	"common.filter.all": "Tous",
	"common.input.free_text": "Texte libre",
	"common.nav.archives": "Archives",
	"common.nav.history": "Historique",
	"common.nav.menu": "Menu",
	"common.nav.network": "Réseau",
	"common.snackbar.copied": "Code {code} copié.",
	"common.snackbar.ia_profile_synced": "Profil IA synchronisé.",
	"common.snackbar.profile_synced": "Profil synchronisé.",
	"common.status.active": "Actif",
	"common.status.connecting": "Connexion...",
	"common.status.creating": "Création...",
	"common.status.inactive": "Inactif",
	"common.status.initializing": "Initialisation...",
	"common.status.loading": "Chargement...",
	"common.status.syncing": "Synchronisation...",
	"common.user.you": "Vous",
	"common.value.none": "Aucun",
	"coop.action.cover_ally": "Couvrir l’allié",
	"coop.action.inject_ghost": "Injecter un ghost payload",
	"coop.action.jam_signal": "Brouiller le signal",
	"coop.ally": "Allié",
	"coop.chat.hint": "Message rapide à l’allié...",
	"coop.chat.quick_message_hint": "Message rapide...",
	"coop.choice.none": "Aucun choix",
	"coop.copy_code.tooltip": "Copier le code",
	"coop.create.button": "CRÉER LE LOBBY CO-OP",
	"coop.create.creating": "Création en cours...",
	"coop.create.no_provider": "Sélectionne un fournisseur IA (clé API) avant de créer la mission.",
	"coop.create.no_quest": "Choisis une quete avant de creer le coop.",
	"coop.create.provider_helper": "Obligatoire: moteur IA utilisé pour la mission co-op.",
	"coop.create.provider_label": "Fournisseur IA de mission",
	"coop.create.provider_required.button": "CONFIGURER LES FOURNISSEURS",
	"coop.create.provider_required.desc": "Ajoute une clé API fournisseur avant de lancer une mission co-op.",
	"coop.create.provider_required.title": "FOURNISSEUR IA REQUIS",
	"coop.create.quest_helper": "La quête sera jouée en coopération avec un autre joueur.",
	"coop.create.quest_label": "Quête pour la mission",
	"coop.create.title": "CRÉER UNE MISSION",
	"coop.default_option.ghost_payload.caption": "Détourne l’attention ennemie.",
	"coop.default_option.ghost_payload.label": "Charge fantôme",
	"coop.default_option.jam_signal.caption": "Brouille le signal hostile.",
	"coop.default_option.jam_signal.label": "Brouiller le signal",
	"coop.default_option.shield_ally.caption": "Protège l’allié pendant l’action.",
	"coop.default_option.shield_ally.label": "Bouclier allié",
	"coop.form.provider_helper": "Choisis un profil IA pour créer la mission.",
	"coop.form.provider_label": "Profil IA",
	"coop.hub.code_hint": "Code de mission",
	"coop.hub.code_label": "Code coop",
	"coop.hub.create_live_lobby": "Créer un lobby live",
	"coop.hub.create_mission": "Créer une mission",
	"coop.hub.description": "Crée un lobby, partage le code, puis avancez à deux dans une vraie scène commune : présence visible, choix synchronisés, confirmation en popin et chat partagé. Bref, le coop arrête enfin de faire semblant.",
	"coop.hub.join_by_code": "Rejoindre par code",
	"coop.hub.join_mission": "Rejoindre la mission",
	"coop.hub.no_lobby_message": "Aucun lobby coop actif pour le moment.",
	"coop.hub.no_lobby_title": "Aucun lobby",
	"coop.hub.status_title": "Statut coop",
	"coop.hub.subtitle": "SYNCHRO TACTIQUE",
	"coop.hub.sync_description": "Les choix des joueurs sont synchronisés par le backend.",
	"coop.hub.sync_title": "Coop live synchronisée",
	"coop.hub.title": "MISSION CO-OP LIVE",
	"coop.hub.your_lobbies": "Vos lobbies",
	"coop.join.button": "REJOINDRE LA MISSION",
	"coop.join.hint": "Ex. SYNC42",
	"coop.join.title": "REJOINDRE PAR CODE",
	"coop.lobby.cancel": "ANNULER",
	"coop.lobby.check_connection": "VÉRIFIER LA CONNEXION",
	"coop.lobby.code_label": "CODE LOBBY",
	"coop.lobby.confirm_quit": "CONFIRMER",
	"coop.lobby.connection_issue": "Problème de connexion co-op",
	"coop.lobby.connection_ok": "Connexion co-op OK",
	"coop.lobby.copy_tooltip": "Copier le code",
	"coop.lobby.empty": "Aucun lobby pour l’instant",
	"coop.lobby.leave": "QUITTER",
	"coop.lobby.my_lobbies": "TES LOBBIES",
	"coop.lobby.opening_ia_unavailable": "Ouverture IA indisponible, fallback co-op activé.",
	"coop.lobby.resync": "RESYNC",
	"coop.lobby.state.hub": "ÉTAT DU HUB",
	"coop.lobby.title": "LOBBY CO-OP",
	"coop.lobby.verify_connection": "Vérifier la connexion",
	"coop.lobby.waiting": "En attente du deuxième joueur...",
	"coop.mission.ally_choice": "Choix allié",
	"coop.mission.players": "JOUEURS {current}/2",
	"coop.mission.resync": "Resynchroniser",
	"coop.mission.sync": "SYNC {percent}%",
	"coop.mission.sync_percent": "Synchronisation {percent}%",
	"coop.mission.syncing": "Synchronisation en cours",
	"coop.mission.your_choice": "Votre choix",
	"coop.provider_required.cta": "Configurer un profil IA",
	"coop.provider_required.message": "Un profil IA est requis pour créer une mission coop.",
	"coop.provider_required.title": "Profil IA requis",
	"coop.quest.change": "Changer de quête",
	"coop.quest.current": "Quête actuelle",
	"coop.quest.local_badge": "LOCAL",
	"coop.quest.none_found": "Aucune quête trouvée.",
	"coop.quest.none_published": "Aucune quête publiée.",
	"coop.quest.search_hint": "Rechercher une quête coop.",
	"coop.quest.select": "Sélectionner la quête",
	"coop.quest.status_mismatch": "Statut incompatible",
	"coop.quest.tip": "Choisis une quête avant de lancer la mission.",
	"coop.quest.unavailable": "Quêtes indisponibles : {error}",
	"coop.quest.xp": "+{xp} XP",
	"coop.search.hint": "titre, contenu, thème...",
	"coop.snackbar.choose_provider_before_create": "Choisis un profil IA avant de créer.",
	"coop.snackbar.choose_quest_before_create": "Choisis une quête avant de créer.",
	"coop.snackbar.enter_code_before_join": "Entre un code coop avant de rejoindre.",
	"coop.title": "Coop live",
	"coop.waiting.check_connection": "Vérifier la connexion",
	"coop.waiting.guest_message": "En attente de l’hôte.",
	"coop.waiting.host_message": "En attente de l’allié.",
	"coop.waiting.players": "{count} joueur(s)",
	"coop.waiting.resume_tip": "La mission reprendra dès que les deux joueurs seront synchronisés.",
	"coop.waiting.waiting_ally": "Attente de l’allié",
	"coop.waiting.waiting_host": "Attente de l’hôte",
	"home.card.battle.badge": "PVP LIVE",
	"home.card.battle.cta": "Entrer dans l’arène",
	"home.card.battle.desc": "Faites s’affronter deux IA dans des débats épiques. Votez, jugez et montez dans le classement.",
	"home.card.battle.title": "A.I Battle",
	"home.card.coop.badge": "SYNC DUO",
	"home.card.coop.cta": "Ouvrir le hub co-op",
	"home.card.coop.desc": "Créez une mission synchronisée, partagez un code et coordonnez vos votes en duo.",
	"home.card.coop.title": "Co-op Live",
	"home.card.roleplay.badge": "Histoire / RPG",
	"home.card.roleplay.cta": "Démarrer une quête",
	"home.card.roleplay.desc": "Vivez des aventures interactives. Choisissez votre histoire, résolvez des quêtes et explorez la carte.",
	"home.card.roleplay.title": "A.I Quest",
	"home.card.sandbox.badge": "DARK LAB",
	"home.card.sandbox.cta": "Ouvrir le Nexus",
	"home.card.sandbox.desc": "Le côté dark de l’IA : discutez avec vos modèles API ou locaux, testez les réponses et gardez l’historique.",
	"home.card.sandbox.title": "Sandbox Nexus",
	"home.card.tribunal.badge": "JUSTICE IA",
	"home.card.tribunal.cta": "Ouvrir le Tribunal",
	"home.card.tribunal.desc": "Protocole Objection : enquêtez, interrogez des témoins, présentez des preuves, objectez et obtenez un verdict par jury IA.",
	"home.card.tribunal.title": "Tribunal IA",
	"home.player.xp": "XP {xp}",
	"home.session_unavailable": "Session indisponible",
	"ia_profiles.created_section": "Profils créés",
	"ia_profiles.delete.title": "Supprimer ce profil IA ?",
	"ia_profiles.empty": "Aucun profil IA créé.",
	"ia_profiles.form.choose_preset": "Choisir un preset",
	"ia_profiles.form.create_title": "Créer un profil IA",
	"ia_profiles.form.edit_title": "Modifier le profil IA",
	"ia_profiles.form.goal": "Objectif",
	"ia_profiles.form.mindset": "État d’esprit",
	"ia_profiles.form.model_name": "Modèle",
	"ia_profiles.form.name_label": "Nom du profil",
	"ia_profiles.form.personality": "Personnalité",
	"ia_profiles.form.provider_label": "Fournisseur",
	"ia_profiles.form.style": "Style",
	"ia_profiles.form.weakness": "Faiblesse",
	"ia_profiles.providers_unavailable": "Fournisseurs indisponibles : {error}",
	"ia_profiles.snackbar.error": "Erreur profil IA : {error}",
	"ia_profiles.subtitle.free_provider": "Fournisseur libre",
	"ia_profiles.subtitle.undefined_model": "Modèle non défini",
	"ia_profiles.title": "Profils IA",
	"ia_profiles.validation.name_required": "Le nom est obligatoire.",
	"ia_profiles.validation.provider_invalid": "Choisis un fournisseur valide.",
	"local_models.compat.blocked": "Incompatible",
	"local_models.compat.good": "Compatible",
	"local_models.compat.unknown": "Compatibilité inconnue",
	"local_models.compat.warning": "Compatibilité limitée",
	"local_models.delete.content": "Supprimer le modèle local {fileName} ?",
	"local_models.delete.title": "Supprimer le modèle ?",
	"local_models.device.lock_active": "Verrou actif",
	"local_models.device.low": "Appareil limité",
	"local_models.device.powerful": "Appareil puissant",
	"local_models.device.standard": "Appareil standard",
	"local_models.device.unavailable": "État appareil indisponible",
	"local_models.download.blocked_mobile": "Téléchargement bloqué sur mobile",
	"local_models.download.restart": "Relancer le téléchargement",
	"local_models.installed.empty.text": "Aucun modèle local installé.",
	"local_models.installed.empty.title": "Aucun modèle installé",
	"local_models.local_read_error.title": "Lecture locale impossible",
	"local_models.meta.downloaded_at": "Téléchargé le {date}",
	"local_models.meta.downloads": "{count} téléchargement(s)",
	"local_models.meta.hf_size": "Taille HF : {size}",
	"local_models.meta.hf_size_unknown": "Taille HF inconnue",
	"local_models.meta.loaded": "Chargé",
	"local_models.meta.selected": "Sélectionné",
	"local_models.meta.size_unknown": "Taille inconnue",
	"local_models.progress.bytes": "{received} / {total}",
	"local_models.search.empty.text": "Essaie une autre recherche Hugging Face.",
	"local_models.search.empty.title": "Aucun modèle trouvé",
	"local_models.search.error.title": "Recherche impossible",
	"local_models.search.hf_running": "Recherche Hugging Face en cours...",
	"local_models.search.hint": "Nom du modèle ou organisation",
	"local_models.search.label": "Recherche modèle",
	"local_models.search.load_more": "Charger plus",
	"local_models.search.loading": "Recherche...",
	"local_models.section.catalog": "Catalogue Hugging Face",
	"local_models.section.installed": "Modèles installés",
	"local_models.tester.default_prompt": "Écris une réponse courte pour tester ce modèle local.",
	"local_models.tester.empty_transcript": "Aucun message local.",
	"local_models.tester.interrupted_response": "Réponse interrompue : {partial}",
	"local_models.tester.loading_runtime": "Chargement du runtime local...",
	"local_models.tester.local_conversation": "Conversation locale",
	"local_models.tester.local_model": "Modèle local",
	"local_models.tester.prompt_hint": "Message à envoyer au modèle local.",
	"local_models.tester.prompt_label": "Prompt de test",
	"local_models.tester.reset_chat": "Réinitialiser le chat",
	"local_models.tester.thinking": "Réflexion...",
	"local_models.tester.thinking_empty": "Réponse en cours...",
	"local_models.tester.title": "Testeur local",
	"local_models.title": "Modèles IA locaux",
	"network.audit.battles": "{count} battle(s)",
	"network.audit.filter_reasons": "Raisons de filtrage",
	"network.audit.live_title": "Audit live",
	"network.audit.live_ui": "UI live : {count}",
	"network.audit.mode_legacy": "Mode legacy",
	"network.audit.mode_sessions_only": "Sessions uniquement",
	"network.audit.replays_ui": "UI replays : {count}",
	"network.audit.sessions": "Sessions : {count}",
	"network.audit.sessions_live": "Sessions live : {count}",
	"network.audit.sessions_with_battle": "Sessions avec battle : {count}",
	"network.audit.sync": "Sync : {time}",
	"network.debug.battle_id": "Battle #{id}",
	"network.debug.channel": "Canal : {channel}",
	"network.debug.cursor": "Curseur : {cursor}",
	"network.debug.events": "Événements : {count}",
	"network.debug.no_sync_message": "Aucune synchronisation live.",
	"network.debug.polls": "Polls : {count}",
	"network.debug.refresh_900ms": "Rafraîchissement 900 ms",
	"network.debug.source_battle": "Source battle",
	"network.debug.source_live_history": "Source historique live",
	"network.debug.transport": "Transport : {transport}",
	"network.debug.transport_title": "Transport live",
	"network.debug.turns": "Tours : {count}",
	"network.hero.active": "Live actif",
	"network.hero.idle": "Aucun live actif",
	"network.history.default_battle_title": "Battle #{id}",
	"network.history.delete.content": "Supprimer l’archive du battle #{id} ?",
	"network.history.delete.snackbar": "Archive #{id} supprimée.",
	"network.history.delete.title": "Supprimer cette archive ?",
	"network.history.empty.message": "Aucune archive de battle disponible.",
	"network.history.empty.title": "Aucune archive",
	"network.history.period": "Du {startedAt} au {endedAt}",
	"network.history.rounds": "Rounds : {current}/{total}",
	"network.history.status.archive": "Archive",
	"network.history.title": "Archives Battle IA",
	"network.history.view_replay": "Voir le replay",
	"network.history.wall_title": "Mur des archives",
	"network.history.winner": "Vainqueur : {winner}",
	"network.live.empty.message": "Aucun battle live public pour le moment.",
	"network.live.empty.title": "Aucun live",
	"network.live.section.live": "Battles live",
	"network.live.section.replays": "Replays publics",
	"network.live.title": "Coop / Live Network",
	"network.metric.archives": "Archives",
	"network.metric.comments": "{count} commentaire(s)",
	"network.metric.live": "Live",
	"network.metric.public": "Public",
	"network.metric.replays": "Replays",
	"network.metric.viewers": "{count} spectateur(s)",
	"network.metric.viewers_label": "Spectateurs",
	"network.replays.empty.message": "Aucun replay public disponible.",
	"network.replays.empty.title": "Aucun replay",
	"network.spectator.channel_title": "Canal spectateur",
	"network.spectator.highlights": "Temps forts",
	"network.spectator.join_arena": "Rejoindre l’arène",
	"network.spectator.joined_arena": "Arène rejointe",
	"network.spectator.login_to_interact": "Connecte-toi pour interagir.",
	"network.spectator.no_highlights": "Aucun temps fort.",
	"network.spectator.timeline": "Timeline",
	"network.spectator.timeline_empty": "Timeline vide.",
	"network.spectator.title": "Spectateur live",
	"network.status.public_live": "Live public",
	"network.status.public_replay": "Replay public",
	"network.status.read_only": "Lecture seule",
	"network.winner.explicit": "Vainqueur : {winner}",
	"network.winner.unknown": "Vainqueur inconnu",
	"nexus_game.ai.analyse": "Analyse IA",
	"nexus_game.ai.ask": "Demander à mon IA",
	"nexus_game.ai.budget": "Budget tokens",
	"nexus_game.ai.plan": "Plan IA",
	"nexus_game.city.energy": "Énergie",
	"nexus_game.city.hub.title": "HUB VILLE",
	"nexus_game.city.moral": "Moral",
	"nexus_game.city.population": "Population",
	"nexus_game.city.security": "Sécurité",
	"nexus_game.loading": "Chargement du Nexus...",
	"nexus_game.offline": "Mode hors-ligne • données locales",
	"profile.action.logout_network": "Quitter le réseau",
	"profile.form.confirm_password": "Confirmer le mot de passe",
	"profile.form.new_password": "Nouveau mot de passe",
	"profile.form.pseudo": "Pseudo",
	"profile.player.title": "Profil joueur",
	"profile.provider_keys.add": "Ajouter la clé",
	"profile.provider_keys.api_key_hint": "Colle ta clé API ici.",
	"profile.provider_keys.api_key_label": "Clé API",
	"profile.provider_keys.description": "Gère les clés fournisseur utilisées par tes profils IA.",
	"profile.provider_keys.empty": "Aucune clé fournisseur enregistrée.",
	"profile.provider_keys.load_error": "Impossible de charger les clés : {error}",
	"profile.provider_keys.model_helper": "Modèle par défaut : {model}",
	"profile.provider_keys.model_label": "Modèle préféré",
	"profile.provider_keys.no_provider": "Aucun fournisseur",
	"profile.provider_keys.provider_label": "Fournisseur",
	"profile.provider_keys.recent_models": "Modèles récents",
	"profile.provider_keys.test_failed": "Test échoué",
	"profile.provider_keys.test_key": "Tester la clé",
	"profile.provider_keys.test_ok": "Test réussi",
	"profile.provider_keys.title": "Clés fournisseurs IA",
	"profile.provider_keys.where_create": "Créer une clé fournisseur",
	"profile.rank.elite_level": "Elite niveau {level}",
	"profile.session_unavailable": "Session indisponible.",
	"profile.stats.credits": "Crédits",
	"profile.stats.victories": "Victoires",
	"profile.stats.xp": "XP",
	"profile.status.online": "En ligne",
	"profile.title": "Profil",
	"profile.validation.min_3": "Minimum 3 caractères.",
	"profile.validation.min_8": "Minimum 8 caractères.",
	"profile.validation.password_mismatch": "Les mots de passe ne correspondent pas.",
	"roleplay.abandon_dialog.content": "Abandonner la quête {questTitle} ?",
	"roleplay.abandon_dialog.title": "Abandonner la quête ?",
	"roleplay.action.hack": "EXÉCUTER HACK",
	"roleplay.action.scan": "SCANNER ZONE",
	"roleplay.archive.arc_count": "{arcs} arc(s)",
	"roleplay.archive.arc_position": "Arc {arc}",
	"roleplay.archive.arc_trail": "Trajectoire d’arc",
	"roleplay.filter.all": "Tous",
	"roleplay.filter.all_sources": "Toutes les sources",
	"roleplay.filter.level": "Niveau",
	"roleplay.filter.source": "Source",
	"roleplay.filter.theme": "Thème",
	"roleplay.filters.search.clear_tooltip": "Effacer la recherche",
	"roleplay.filters.search.hint": "Rechercher une quête RP.",
	"roleplay.filters.search.label": "Recherche",
	"roleplay.filters.title": "Filtres RP",
	"roleplay.history.empty": "Aucune session RP pour l’instant.",
	"roleplay.history.empty.message": "Aucune session RP terminée.",
	"roleplay.history.empty.title": "Aucun historique RP",
	"roleplay.history.title": "Historique RP IA",
	"roleplay.history.turn": "Tour {turn}",
	"roleplay.history.view_session": "Voir la session",
	"roleplay.hub_narrative.desc.no_provider": "Configure un profil IA pour démarrer une narration.",
	"roleplay.hub_narrative.desc.no_ready": "Prépare un profil IA ou un modèle local pour jouer.",
	"roleplay.hub_narrative.desc.ready": "Narration IA prête.",
	"roleplay.hub_narrative.title": "Hub narratif",
	"roleplay.progress.active_arc": "Arc actif",
	"roleplay.progress.arc_map": "Carte des arcs",
	"roleplay.progress.arc_order": "Arc #{order}",
	"roleplay.progress.arcs_progress": "Progression des arcs",
	"roleplay.progress.back_to_path_view": "Retour au chemin",
	"roleplay.progress.back_to_session": "Retour à la session",
	"roleplay.progress.completed_percent": "Progression terminée",
	"roleplay.progress.map": "Carte",
	"roleplay.progress.not_found": "Progression introuvable.",
	"roleplay.progress.percent": "{percent}% terminé",
	"roleplay.progress.rewards": "Récompenses",
	"roleplay.progress.source.local": "LOCAL",
	"roleplay.progress.status.current": "ACTUELLE",
	"roleplay.progress.toggle_view": "Changer de vue",
	"roleplay.progress.vertical_path": "Chemin vertical",
	"roleplay.progress.xp_reward": "+{xp} XP",
	"roleplay.quest.abandon": "ABANDONNER LA QUÊTE",
	"roleplay.quest.abandon_cancel": "ANNULER",
	"roleplay.quest.abandon_confirm": "Abandonner la quête ?",
	"roleplay.quest.abandon_confirm_action": "ABANDONNER",
	"roleplay.quest.abandoned": "Quête abandonnée • progression remise à zéro.",
	"roleplay.quest.abandoned_snackbar": "Quête abandonnée.",
	"roleplay.quest.boss_mission": "Mission boss",
	"roleplay.quest.card.join": "Lancer la quête",
	"roleplay.quest.card.resume": "Reprendre",
	"roleplay.quest.daily_quests": "Quêtes quotidiennes",
	"roleplay.quest.empty.filtered_message": "Aucune quête ne correspond aux filtres.",
	"roleplay.quest.empty.filtered_title": "Aucun résultat",
	"roleplay.quest.empty.published_message": "Aucune quête publiée disponible.",
	"roleplay.quest.empty.published_title": "Aucune quête publiée",
	"roleplay.quest.level.all": "Tous niveaux",
	"roleplay.quest.source.all": "Toutes sources",
	"roleplay.quest.special_mission": "Mission spéciale",
	"roleplay.quest.start": "Démarrer la quête",
	"roleplay.quest.start_boss": "Démarrer le boss",
	"roleplay.quest.theme.all": "Tous thèmes",
	"roleplay.quest_catalog.title": "Quêtes RP IA",
	"roleplay.quest_catalog.tooltip.archives": "Archives RP",
	"roleplay.quest_catalog.tooltip.refresh": "Rafraîchir",
	"roleplay.reward.coins": "{coins} crédits",
	"roleplay.reward.xp": "{xp} XP",
	"roleplay.section.ongoing_sessions": "Sessions en cours",
	"roleplay.session.abandon_quest": "Abandonner la quête",
	"roleplay.session.arc_chapter": "Arc {arc}/{arcTotal} · Chapitre {chapter}/{chapterTotal}",
	"roleplay.session.back": "RETOUR À LA SESSION",
	"roleplay.session.back_to_active": "REVENIR À L’ÉTAPE ACTIVE",
	"roleplay.session.back_to_active_step": "Retour à l’étape active",
	"roleplay.session.choose_action": "Choisir une action",
	"roleplay.session.completed_snackbar": "Quête terminée : +{xp} XP.",
	"roleplay.session.mission_validated": "Mission validée",
	"roleplay.session.next_step": "Étape suivante",
	"roleplay.session.objective": "Objectif",
	"roleplay.session.previous_step": "Étape précédente",
	"roleplay.session.progress": "Progression",
	"roleplay.session.resume": "REPRENDRE",
	"roleplay.session.view": "VOIR LA SESSION RP",
	"roleplay.session_detail.title": "SESSION RP",
	"roleplay.start.default_model": "Modèle par défaut",
	"roleplay.start.local_model_title": "Modèle local",
	"roleplay.start.narrator_profile": "Narrateur IA",
	"roleplay.start.no_local_model": "Aucun modèle local disponible.",
	"roleplay.start.no_provider_profile": "Aucun profil fournisseur.",
	"roleplay.start.provider_api_desc": "Utilise un profil IA connecté au backend.",
	"roleplay.start.provider_api_title": "API fournisseur",
	"roleplay.stat.coins": "{coins} crédits",
	"roleplay.stat.local_ready": "Local : {fileName}",
	"roleplay.stat.local_unavailable": "Aucun modèle local",
	"roleplay.stat.provider_not_configured": "Aucun fournisseur configuré",
	"roleplay.stat.provider_ready": "{count} profil{s} IA prêt{s}",
	"roleplay.stat.xp": "{xp} XP",
	"roleplay.tag.boss": "Boss",
	"roleplay.tag.current": "Actuel",
	"roleplay.tag.validated": "Validé",
	"roleplay.zone.execute_hack": "Exécuter le hack",
	"roleplay.zone.scan": "Scanner la zone",
	"roleplay.zone.threat": "Menace {threat}",
	"roleplay.zone.uplink_active": "Uplink actif",
	"sandbox.action.create_first": "Créer la première conversation",
	"sandbox.action.new_chat": "Nouveau chat",
	"sandbox.action.new_conversation": "Nouvelle conversation",
	"sandbox.action.show_all_history": "Voir tout l’historique",
	"sandbox.bot.name": "Assistant sandbox",
	"sandbox.chat.empty_header": "Aucun message",
	"sandbox.chat.message_count": "{count} message{plural}",
	"sandbox.chat.new_title": "Nouvelle conversation",
	"sandbox.chat.resume_header": "Reprendre la conversation",
	"sandbox.conversation.api_ready": "API prête",
	"sandbox.conversation.local_ready": "Local prêt",
	"sandbox.conversations.description": "Conversations sauvegardées localement.",
	"sandbox.conversations.recent_count": "{count} conversation(s) récente(s)",
	"sandbox.conversations.title": "Conversations",
	"sandbox.empty.api_description": "Configure une clé fournisseur pour utiliser l’API.",
	"sandbox.empty.configure_provider_key": "Configurer une clé fournisseur",
	"sandbox.empty.install_local_model": "Installer un modèle local",
	"sandbox.empty.local_description": "Installe un modèle local pour discuter hors ligne.",
	"sandbox.empty_history": "CRÉER UNE PREMIÈRE CONVERSATION",
	"sandbox.engine.description": "Choisis le moteur de génération.",
	"sandbox.engine.local": "Modèle local",
	"sandbox.engine.provider": "Fournisseur API",
	"sandbox.engine.title": "Moteur IA",
	"sandbox.engine_chooser": "Choisir moteur",
	"sandbox.engine_sheet.description": "Sélectionne la source de réponse pour cette conversation.",
	"sandbox.engine_sheet.title": "Changer de moteur",
	"sandbox.error_panel": "messages d'erreur IA",
	"sandbox.history.all": "Tout",
	"sandbox.history.empty": "Aucun historique.",
	"sandbox.history.title": "Historique",
	"sandbox.local.empty.cta": "Installer un modèle",
	"sandbox.local.empty.message": "Aucun modèle local chargé.",
	"sandbox.local.empty.title": "Local indisponible",
	"sandbox.local.not_configured": "Modèle local non configuré",
	"sandbox.local.select_label": "Sélectionner un modèle local",
	"sandbox.message.generating": "Génération...",
	"sandbox.new_conversation": "NOUVELLE CONVERSATION",
	"sandbox.prompt.hint": "Écris ton message...",
	"sandbox.provider.empty.cta": "Configurer une clé",
	"sandbox.provider.empty.message": "Aucune clé fournisseur disponible.",
	"sandbox.provider.empty.title": "Fournisseur indisponible",
	"sandbox.provider.not_configured": "Fournisseur non configuré",
	"sandbox.provider.select_label": "Sélectionner un fournisseur",
	"sandbox.recent.empty.message": "Aucune conversation récente.",
	"sandbox.recent.empty.title": "Rien à reprendre",
	"sandbox.recent.title": "Récents",
	"sandbox.source.local_llama": "Llama local",
	"sandbox.source.provider_backend": "Backend fournisseur",
	"sandbox.start": "COMMENCER",
	"sandbox.title": "Sandbox IA",
	"sandbox.topbar.back_tooltip": "Retour",
	"sandbox.topbar.history_tooltip": "Historique",
	"sandbox.topbar.new_chat_tooltip": "Nouveau chat",
	"sandbox.view_history": "VOIR TOUT L’HISTORIQUE",
}

// Admin method implementations for POINT 03

func (s *dbTranslationService) GetAllDomains(ctx context.Context) ([]models.TranslationDomain, error) {
	var domains []models.TranslationDomain
	if err := s.db.Find(&domains).Error; err != nil {
		return nil, err
	}
	return domains, nil
}

func (s *dbTranslationService) CreateDomain(ctx context.Context, d *models.TranslationDomain) error {
	return s.db.Create(d).Error
}

func (s *dbTranslationService) GetAllKeys(ctx context.Context) ([]models.TranslationKey, error) {
	var keys []models.TranslationKey
	if err := s.db.Preload("Domain").Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

func (s *dbTranslationService) CreateKey(ctx context.Context, k *models.TranslationKey) error {
	return s.db.Create(k).Error
}

func (s *dbTranslationService) UpdateKey(ctx context.Context, id uint, k *models.TranslationKey) error {
	return s.db.Model(&models.TranslationKey{ID: id}).Updates(k).Error
}

func (s *dbTranslationService) DeleteKey(ctx context.Context, id uint) error {
	return s.db.Delete(&models.TranslationKey{}, id).Error
}

func (s *dbTranslationService) GetAllValues(ctx context.Context) ([]models.TranslationValue, error) {
	var values []models.TranslationValue
	if err := s.db.Preload("Key").Find(&values).Error; err != nil {
		return nil, err
	}
	return values, nil
}

func (s *dbTranslationService) UpdateValue(ctx context.Context, id uint, v *models.TranslationValue) error {
	return s.db.Model(&models.TranslationValue{ID: id}).Updates(v).Error
}

func (s *dbTranslationService) PreviewImport(ctx context.Context, rows []models.TranslationImportRow) ([]models.TranslationImportRow, error) {
	// Simulate preview: mark errors but do not persist
	for i := range rows {
		if rows[i].Key == "" || rows[i].Value == "" || rows[i].Locale == "" {
			rows[i].Status = "error"
			rows[i].Error = "missing required fields"
		} else {
			rows[i].Status = "ok"
		}
	}
	return rows, nil
}

func (s *dbTranslationService) CommitImport(ctx context.Context, importID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var imp models.TranslationImport
		if err := tx.First(&imp, importID).Error; err != nil {
			return err
		}

		var rows []models.TranslationImportRow
		if err := tx.Where("import_id = ?", importID).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return errors.New("import has no rows to commit")
		}

		for _, row := range rows {
			if row.Status == "error" {
				return errors.New("import contains invalid rows")
			}
			if err := upsertTranslationRow(tx, row); err != nil {
				return err
			}
		}

		imp.Status = "committed"
		imp.RowCount = len(rows)
		return tx.Save(&imp).Error
	})
}

func (s *dbTranslationService) CommitImportRows(ctx context.Context, rows []models.TranslationImportRow, fileName string) (*models.TranslationImport, error) {
	if len(rows) == 0 {
		return nil, errors.New("rows are required")
	}
	if fileName == "" {
		fileName = "admin-import.json"
	}

	var committedImport *models.TranslationImport
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		imp := models.TranslationImport{
			FileName: fileName,
			Status:   "committing",
			RowCount: len(rows),
		}
		if err := tx.Create(&imp).Error; err != nil {
			return err
		}

		for i := range rows {
			rows[i].ImportID = imp.ID
			if rows[i].Key == "" || rows[i].Value == "" || rows[i].Locale == "" || rows[i].Domain == "" {
				rows[i].Status = "error"
				if rows[i].Error == "" {
					rows[i].Error = "missing required fields"
				}
			} else if rows[i].Status == "" {
				rows[i].Status = "ok"
			}

			if err := tx.Create(&rows[i]).Error; err != nil {
				return err
			}
			if rows[i].Status == "error" {
				return errors.New("import contains invalid rows")
			}
			if err := upsertTranslationRow(tx, rows[i]); err != nil {
				return err
			}
		}

		imp.Status = "committed"
		if err := tx.Save(&imp).Error; err != nil {
			return err
		}
		committedImport = &imp
		return nil
	})
	if err != nil {
		return nil, err
	}
	return committedImport, nil
}

func (s *dbTranslationService) GetImports(ctx context.Context) ([]models.TranslationImport, error) {
	var imports []models.TranslationImport
	if err := s.db.Find(&imports).Error; err != nil {
		return nil, err
	}
	return imports, nil
}

func (s *dbTranslationService) GetImportByID(ctx context.Context, id uint) (*models.TranslationImport, error) {
	var imp models.TranslationImport
	if err := s.db.First(&imp, id).Error; err != nil {
		return nil, err
	}
	return &imp, nil
}

func (s *dbTranslationService) GetMissing(ctx context.Context) ([]models.TranslationMissingLog, error) {
	var logs []models.TranslationMissingLog
	if err := s.db.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func (s *dbTranslationService) ExportTranslations(ctx context.Context, lang string) (map[string]string, error) {
	return s.GetTranslations(ctx, lang, nil)
}

func (s *dbTranslationService) BatchUpdate(ctx context.Context, entries []TranslationEntry) error {
	return s.UpsertBatch(ctx, entries)
}

func upsertTranslationRow(tx *gorm.DB, row models.TranslationImportRow) error {
	var domain models.TranslationDomain
	if err := tx.Where("code = ?", row.Domain).
		FirstOrCreate(&domain, models.TranslationDomain{Code: row.Domain, Name: row.Domain}).Error; err != nil {
		return err
	}

	var key models.TranslationKey
	if err := tx.Where("domain_id = ? AND `key` = ?", domain.ID, row.Key).
		FirstOrCreate(&key, models.TranslationKey{DomainID: domain.ID, Key: row.Key}).Error; err != nil {
		return err
	}

	value := models.TranslationValue{
		KeyID:  key.ID,
		Locale: row.Locale,
		Value:  row.Value,
	}
	return tx.Where("key_id = ? AND locale = ?", key.ID, row.Locale).
		Assign(value).
		FirstOrCreate(&value).Error
}
