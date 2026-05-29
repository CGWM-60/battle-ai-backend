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

## 5. Endpoints Bâtiments

### GET `/api/v1/buildings`
- Objectif: état bâtiments joueur.

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

Base existante:
- `/api/v1/world/diplomacy/relations`
- `/api/v1/world/diplomacy/treaties`
- `/api/v1/world/diplomacy/reports`
- `/api/v1/world/diplomacy/negotiations/open`
- `/api/v1/world/diplomacy/emissaries/status`
- `/api/v1/world/diplomacy/emissaries/send`

À aligner avec les routes cibles `/api/v1/diplomacy/*`.

## 9. Endpoints Commerce

Base existante:
- `/api/v1/world/commerce/routes`
- `/api/v1/world/commerce/details`
- `/api/v1/world/commerce/agreements`
- `/api/v1/world/commerce/routes/optimize`
- `/api/v1/world/commerce/report`

À aligner avec `/api/v1/trade/*`.

## 10. Endpoints Météo & Risques

Base existante:
- `/api/v1/world/weather/forecast`
- `/api/v1/world/weather/zones`
- `/api/v1/world/weather/actions/{actionKey}`
- `/api/v1/world/weather/report`

À aligner avec `/api/v1/weather/*`.

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
- Formule coût upgrade niveau N:
  - `baseCost * pow(1.28, N-1)`
- Formule durée upgrade niveau N:
  - `baseDuration * pow(1.22, N-1)`
  - cap à 30 jours.
- Caserne (`barracks`) de niveau 1 à 30, paliers de déblocage unités implémentés.

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

## 16. TODO Flutter

- Consommer le wrapper JSON standard (`success/data/meta/error`).
- Migrer les appels vers endpoints `/api/v1/army/*`, `/api/v1/buildings/*`, `/api/v1/world/conflicts/{id}/simulation`.
- Afficher les blocages backend (niveau caserne, ressources, max level).
- Supprimer les données mock côté Monde IA au profit des payloads backend.
