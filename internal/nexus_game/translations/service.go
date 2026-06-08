package translations

import (
	"context"
	"time"
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
}

// DefaultFrenchFallback contient un minimum vital pour que l'app ne soit jamais vide.
// Le vrai contenu vient de la DB / admin.
var DefaultFrenchFallback = map[string]string{
	"common.app.title":           "NEXUS GAMES",
	"common.button.confirm":      "CONFIRMER",
	"common.button.cancel":       "ANNULER",
	"nexus_game.city.population": "Population",
	"nexus_game.city.moral":      "Moral",
	"nexus_game.ai.plan":         "Plan IA",
	"nexus_game.ai.ask":          "Demander à mon IA",
}

// TODO (backend):
// - Implémenter le service avec DB (probablement Postgres via les modèles existants)
// - Ajouter Redis cache (comme demandé dans AGENTS 5.5)
// - Endpoints dans le router du package nexus_game
// - Admin Next.js pour édition / import / export / logs
// - Détection de clés manquantes (via logs ou scan statique)
