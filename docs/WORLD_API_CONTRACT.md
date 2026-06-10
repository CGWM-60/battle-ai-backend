# World API Contract

## 1. Vue générale

Le backend Go est la source de vérité pour le module Monde IA. Flutter consomme les données et envoie uniquement des actions.

Format de réponse standard utilisé par les endpoints Monde/Armée/Bâtiments compatibles :

- succès:
  - `success: true`
  - `data: {...}`
  - `meta.serverTime`
  - `meta.nextUpdateAt`
- erreur:
  - `success: false`
  - `error.code`
  - `error.message`
  - `error.details`
  - `meta.serverTime`
  - `meta.nextUpdateAt`

## 2. Authentification

- JWT requis sur les endpoints privés.
- Scope actuel : `/api/v1/*`.
- Identité joueur dérivée du token côté backend.

## 3. Endpoints Monde

### GET `/api/v1/world/overview`
- Objectif: état global joueur/monde.
- Retour: save, bâtiments, événements, conflits, météo, messages.

### GET `/api/v1/world/events`
- Objectif: lister les événements Monde.

### GET `/api/v1/world/events/{id}`
- Objectif: détail d’un événement.

### GET `/api/v1/world/regions`
- Objectif: (à enrichir) mapping régions.
- Statut: partiellement couvert via continents/state actuels.

### GET `/api/v1/world/reports`
- Objectif: rapports monde consolidés.
- Statut: à enrichir via `GenerateWorldReports()`.

### GET `/api/v1/world/conflicts/{id}/simulation`
- Objectif: simulation d’intervention.
- Retour: chance, coûts, pertes, durée, impacts réputation/diplomatie/tension.

### POST `/api/v1/world/conflicts/{id}/intervene`
- Objectif: lancer une intervention.
- Entrée: stratégie + unités (à enrichir pour lock unités persistant).

## 4. Endpoints Quêtes

- `GET /api/v1/quests` (à exposer explicitement depuis services de quêtes)
- `POST /api/v1/quests/{id}/start`
- `GET /api/v1/quests/{id}/progress`
- `POST /api/v1/quests/{id}/claim`

Statut: base service existante, mapping REST Monde à compléter.

## 10) Améliorations "toutes phases d'un coup" (exécution immédiate)

- Routines réelles implémentées : UpdateEnemyAIWorldBehavior, UpdateCityPopulationAndStability (règles habitations/nourriture/satisfaction), CompleteWeatherPlans, GetRecommendedWeatherPlans (DTO riche).
- WeatherActionPlan : id/title/cost/durationMin/effect/riskPercent/finalImpact/whyUseful (pour maquette "pourquoi avant clic").
- Bâtiments lvl 30 + BUILDING_MAX_LEVEL_REACHED déjà dans CalculateBuildingUpgradeCost.
- Army training + consumption déjà fonctionnels (améliorés).
- Commerce/Météo : DTOs prêts pour solde + bonus/risque + plans stratégiques complets.
- Style maquette respecté côté Flutter (améliorations UI existantes prolongées).

Voir world_routines.go et world_game_service.go pour détails.

## 5. Endpoints Bâtiments

### GET `/api/v1/buildings`
- Objectif: état bâtiments joueur.

### GET `/api/v1/buildings/catalog`
- Objectif: catalogue complet des bâtiments constructibles par Flutter.
- Retour par bâtiment: `key`, `name`, `description`, `category`, `researchTreeKey`, `maxLevel`, `baseCostJson`, `levelCostFormulaJson`, `effectsJson`, `unlockRequirementsJson`, `assets`.
- Les assets existants restent séparés dans `assets`; une mise à jour de description ne doit pas écraser les images des piliers.

### POST `/api/v1/buildings/build`
- Entrée: `type`, `level`.
- Retour: action queue + coût calculé.

### POST `/api/v1/buildings/{id}/upgrade`
- Entrée: `type`, `currentLevel`.
- Retour: coût + durée niveau suivant.

### GET `/api/v1/buildings/{id}`
- Retour minimal: id + `maxLevel: 30`.

Règles métier:
- max global bâtiments = **30**.
- upgrade > 30 ->
  - `error.code = BUILDING_MAX_LEVEL_REACHED`
  - `error.message = Ce bâtiment a déjà atteint le niveau maximum.`

## 6. Endpoints Armée / Caserne

### GET `/api/v1/army/overview`
- Retour: total unités, répartition, puissance.

### GET `/api/v1/army/catalog`
- Objectif: catalogue unités pour Flutter avant entraînement.
- Retour par unité: `unitType`, `name`, `description`, `role`, `requiredBarracksLevel`, `trainingDurationSeconds`, `trainingDurationMinutes`, `trainingCost`, `upkeepPerHour`, `stats`, `constraints`.
- Contraintes globales: `buildingKey = barracks`, `currentBarracksLevel`, `maxQuantityPerRequest = 500`, `durationScalesWithQuantity = true`.

### GET `/api/v1/army/units`
- Retour: unités du joueur.

### POST `/api/v1/army/train`
- Entrée:
  - `unitType`
  - `quantity`
- Vérifications:
  - caserne existante
  - niveau caserne requis selon type unité
  - ressources suffisantes
- Retour:
  - `trainingId`
  - `unitType`
  - `quantity`
  - `startedAt`
  - `finishesAt`
  - `durationSeconds`
  - `durationMinutes`
  - `cost`
  - `status`

### POST `/api/v1/army/assign-conflict`
- Entrée: `conflictId`, `soldierIds`, `strategy`.

### POST `/api/v1/army/recall`
- Entrée: `soldierIds`.

### GET `/api/v1/army/reports`
- Retour: rapport armée synthétique.

## 7. Endpoints Conflits

- `GET /api/v1/world/conflicts`
- `GET /api/v1/world/conflicts/{id}`
- `GET /api/v1/world/conflicts/{id}/simulation`
- `POST /api/v1/world/conflicts/{id}/intervene`
- `GET /api/v1/world/conflicts/reports` (à unifier alias)

## 8. Endpoints Diplomatie

Routes exposées:
- `GET /api/v1/diplomacy/overview`
- `GET /api/v1/diplomacy/relations`
- `GET /api/v1/diplomacy/treaties`
- `POST /api/v1/diplomacy/treaties/{id}/accept`
- `POST /api/v1/diplomacy/treaties/{id}/reject`
- `POST /api/v1/diplomacy/treaties/{id}/break`
- `GET /api/v1/diplomacy/negotiations`
- `POST /api/v1/diplomacy/negotiations/start`
- `POST /api/v1/diplomacy/negotiations/{id}/action`
- `GET /api/v1/diplomacy/emissaries`
- `POST /api/v1/diplomacy/emissaries/send`
- `GET /api/v1/diplomacy/reports`

Compatibilité legacy conservée via `/api/v1/world/diplomacy/*`.

## 9. Endpoints Commerce

Routes exposées:
- `GET /api/v1/trade/overview`
- `GET /api/v1/trade/routes`
- `POST /api/v1/trade/routes/create`
- `POST /api/v1/trade/routes/{id}/optimize`
- `POST /api/v1/trade/routes/{id}/pause`
- `POST /api/v1/trade/routes/{id}/protect`
- `GET /api/v1/trade/agreements`
- `POST /api/v1/trade/agreements/{id}/accept`
- `POST /api/v1/trade/agreements/{id}/reject`
- `POST /api/v1/trade/agreements/{id}/negotiate`
- `GET /api/v1/trade/reports`

Compatibilité legacy conservée via `/api/v1/world/commerce/*`.

## 10. Endpoints Météo & Risques

Routes exposées:
- `GET /api/v1/weather/overview`
- `GET /api/v1/weather/events`
- `GET /api/v1/weather/risks`
- `GET /api/v1/weather/plans`
- `POST /api/v1/weather/plans/{id}/start`
- `GET /api/v1/weather/reports`

Compatibilité legacy conservée via `/api/v1/world/weather/*`.

## 11. Modèles JSON attendus par Flutter

### Réponse succès
```json
{
  "success": true,
  "data": {},
  "meta": {
    "serverTime": "2026-05-29T14:00:00Z",
    "nextUpdateAt": "2026-05-29T14:05:00Z"
  }
}
```

### Réponse erreur
```json
{
  "success": false,
  "error": {
    "code": "NOT_ENOUGH_RESOURCES",
    "message": "Ressources insuffisantes pour lancer cette action.",
    "details": {}
  },
  "meta": {
    "serverTime": "2026-05-29T14:00:00Z",
    "nextUpdateAt": "2026-05-29T14:05:00Z"
  }
}
```

## 12. Actions Flutter disponibles

- Train soldiers
- Simulate conflict intervention
- Intervene conflict
- Upgrade/build buildings
- Trigger weather actions
- Open/send diplomacy actions
- Consume reports

## 13. Règles métier

- Tous les bâtiments: niveau max `30`.
- Jobs construction: `durationSeconds` et `durationMinutes` sont renvoyés avec `startedAt` / `completedAt` pour éviter tout calcul ambigu côté Flutter.
- Formule coût upgrade niveau N:
  - `baseCost * pow(1.28, N-1)`
- Formule durée upgrade niveau N:
  - `baseDuration * pow(1.22, N-1)`
  - cap à 30 jours.
- Caserne (`barracks`) de niveau 1 à 30, paliers de déblocage unités implémentés.
- Recherches: `levelProgressionJson` contient `durationMinutes` et `cumulativeMinutes`; une recherche active renvoie aussi `durationSeconds` et `durationMinutes`.
- Validations backend strictes appliquées sur actions critiques:
  - ownership des unités et routes
  - statut compatible (`active`, `available`, etc.)
  - cooldown sur négociations, émissaires et plans météo
  - contrôle ressources avant action
  - stratégie autorisée pour interventions conflits

## 14. Routines serveur

Implémentées/branchées :
- `CompleteArmyTraining()`
- `UpdateArmyConsumption()`
- `UpdateWorldEvents()`
- `UpdateWorldConflicts()`
- `RunWorldMaintenanceTick()` (appelée en cron monde)

Stubs présents pour extension:
- `CompleteBuildingConstruction()`
- `CompleteBuildingUpgrade()`
- `UpdateQuestProgress()`
- `ResolveConflictInterventions()`
- `UpdateWeatherEvents()`
- `CompleteWeatherPlans()`
- `UpdateDiplomaticNegotiations()`
- `UpdateTradeRoutes()`
- `GenerateWorldReports()`

## 15. Erreurs possibles

- `BUILDING_MAX_LEVEL_REACHED`
- `NOT_ENOUGH_RESOURCES`
- `NOT_FOUND`
- `BAD_REQUEST`
- `INTERNAL_SERVER_ERROR`
- `INVALID_PAYLOAD`
- `INVALID_STRATEGY`
- `SOLDIER_OWNERSHIP_MISMATCH`
- `SOLDIER_NOT_AVAILABLE`
- `CONFLICT_NOT_ACTIVE`
- `BARRACKS_LEVEL_REQUIRED`
- `COOLDOWN_ACTIVE`
- `EMISSARY_NOT_AVAILABLE`
- `ROUTE_NOT_FOUND`
- `UNKNOWN_PLAN`

## 16. TODO Flutter

- Consommer le wrapper JSON standard (`success/data/meta/error`).
- Migrer les appels vers endpoints `/api/v1/army/*`, `/api/v1/buildings/*`, `/api/v1/world/conflicts/{id}/simulation`.
- Afficher les blocages backend (niveau caserne, ressources, max level).
- Supprimer les données mock côté Monde IA au profit des payloads backend.
