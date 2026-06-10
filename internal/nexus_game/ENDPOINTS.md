# Nexus Game API Contract

Ce fichier de contrat ne couvre que `internal/nexus_game`.

## Principes communs

- Base gameplay/admin Nexus: `/api/nexus-game`
- Base Flutter publique contenu/construction: `/api/v1`
- Les erreurs standard renvoient `{ "error": "message" }`.
- Les blocages de prerequis renvoient `409` sur les actions et un objet `requirementsStatus`.
- `profileGamerId`, `profileId`, `profile_gamer_id` et `profile_id` sont acceptes selon les endpoints historiques; preferer `profileGamerId` pour Flutter v1.

## Structure JSON complete d'un batiment envoye a Flutter

Endpoint: `GET /api/v1/buildings/:key?profileGamerId=12`

```json
{
  "building": {
    "id": 1,
    "contentId": "building_ai_center",
    "domain": "building",
    "type": "ia",
    "nameKey": "Centre IA",
    "descriptionKey": "Une tour de calcul nerveuse qui orchestre agents, predictions et protocoles de crise du Nexus.",
    "flavorTextKey": "Le Nexus commence a penser plus vite que ses ennemis.",
    "levelDescriptionKeys": {
      "1": "Premier noyau decisionnel",
      "10": "Coordination multi-agents"
    },
    "assetId": "building_ai_center",
    "assetsByTier": {
      "tier1": "https://host/nexus-assets/content/buildings/building_ai_center_tier1.webp",
      "tier2": "https://host/nexus-assets/content/buildings/building_ai_center_tier2.webp"
    },
    "maxLevel": 30,
    "rarity": "rare",
    "nexusLevelRequired": 4,
    "requiredBuildings": "[{\"contentId\":\"building_research_lab\",\"level\":2}]",
    "requiredResearch": "[\"research_agent_basics\"]",
    "costBaseCredits": 1200,
    "costBaseMetal": 600,
    "costBaseData": 200,
    "durationBaseSeconds": 3600,
    "effects": "[{\"stat\":\"aiEfficiency\",\"operation\":\"add_percent\",\"value\":0.05}]",
    "workersMin": 1,
    "workersMax": 5,
    "aiAgentSlots": 2,
    "aiAgentTypes": ["strategist", "analyst"],
    "balanceVersion": "v2.0",
    "isPublished": true,
    "createdAt": "2026-06-10T10:00:00Z",
    "updatedAt": "2026-06-10T10:00:00Z"
  },
  "requirementsStatus": {
    "profileGamerId": 12,
    "domain": "building",
    "contentId": "building_ai_center",
    "allowed": false,
    "requirements": [
      { "type": "nexus_level", "requiredLevel": 4, "currentLevel": 3, "required": true, "satisfied": false, "message": "NEXUS_LEVEL_REQUIRED: Niveau Nexus 4 requis." },
      { "type": "building", "contentId": "building_research_lab", "requiredLevel": 2, "currentLevel": 1, "required": true, "satisfied": false, "message": "BUILDING_REQUIRED: building_research_lab niveau 2 requis." },
      { "type": "research", "contentId": "research_agent_basics", "requiredLevel": 1, "currentLevel": 0, "required": true, "satisfied": false, "message": "RESEARCH_REQUIRED: research_agent_basics requis." }
    ],
    "missing": [
      { "type": "nexus_level", "requiredLevel": 4, "currentLevel": 3, "required": true, "satisfied": false, "message": "NEXUS_LEVEL_REQUIRED: Niveau Nexus 4 requis." }
    ]
  }
}
```

Champs critiques pour Flutter:

- `durationBaseSeconds` est envoye dans le catalogue; les endpoints preview/action renvoient ou materialisent la duree calculee (`durationSeconds`, `constructionEndsAt`, `remainingSec`).
- `nexusLevelRequired`, `requiredBuildings`, `requiredResearch` sont envoyes pour que Flutter puisse afficher les verrous.
- `requirementsStatus.allowed` est la verite serveur pour savoir si le bouton construire/entrainer/rechercher est actif.

## Prerequis batiments, unites, recherches

Les trois piliers utilisent le meme schema:

```json
{
  "profileGamerId": 12,
  "domain": "building",
  "contentId": "building_ai_center",
  "allowed": false,
  "requirements": [
    { "type": "building", "contentId": "building_research_lab", "requiredLevel": 2, "currentLevel": 1, "required": true, "satisfied": false, "message": "BUILDING_REQUIRED: building_research_lab niveau 2 requis." }
  ],
  "missing": []
}
```

Domains acceptes: `building`, `unit`, `research`, avec alias `build`, `construction`, `buildings`, `units`, `tech`, `research_node`.

Formats de prerequis acceptes en base:

```json
[
  { "contentId": "building_barracks", "level": 2 },
  { "buildingKey": "building_research_lab", "minLevel": 3 },
  { "researchContentId": "research_swarm_control" }
]
```

```json
{
  "buildings": [{ "contentId": "building_ai_center", "requiredLevel": 4 }],
  "research": ["research_basic_tactics"]
}
```

## Flutter public `/api/v1`

| Method | Path | Attendu | Retour JSON |
| --- | --- | --- | --- |
| GET | `/api/v1/prerequisites/validate?profileGamerId=12&domain=building&contentId=building_ai_center` | Query `profileGamerId`, `domain`, `contentId` | `PrerequisiteValidation` |
| GET | `/api/v1/buildings/catalog?profileGamerId=12` | Query optionnelle `profileGamerId` | `{ "buildings": [BuildingDefinition], "count": 20, "requirementsStatus": { "contentId": PrerequisiteValidation } }` |
| GET | `/api/v1/buildings/catalog/version` | Aucun | `{ "version": "...", "generatedAt": "..." }` |
| GET | `/api/v1/buildings/:key?profileGamerId=12` | Path `key` | `{ "building": BuildingDefinition, "requirementsStatus": PrerequisiteValidation }` |
| GET | `/api/v1/buildings/:key/research-tree` | Path `key` | `{ "building": "...", "research": [...] }` |
| GET | `/api/v1/units/catalog?profileGamerId=12` | Query optionnelle `profileGamerId` | `{ "units": [UnitDefinition], "count": 15, "requirementsStatus": { "contentId": PrerequisiteValidation } }` |
| GET | `/api/v1/units/:key?profileGamerId=12` | Path `key` | `{ "unit": UnitDefinition, "requirementsStatus": PrerequisiteValidation }` |
| GET | `/api/v1/research/catalog?profileGamerId=12` | Query optionnelle `profileGamerId` | `{ "research": [ResearchDefinition], "count": 77, "requirementsStatus": { "contentId": PrerequisiteValidation } }` |
| GET | `/api/v1/research/:key?profileGamerId=12` | Path `key` | `{ "research": ResearchDefinition, "requirementsStatus": PrerequisiteValidation }` |
| GET | `/api/v1/assets/buildings/manifest` | Aucun | `{ "manifest": [{ "contentId": "...", "assetId": "..." }], "version": "v1" }` |
| GET | `/api/v1/assets/buildings/updates?sinceVersion=v1` | Query `sinceVersion` optionnel | `{ "updates": [...], "version": "v1", "since": "v1" }` |
| GET | `/api/v1/buildings?profileGamerId=12` | Query `profileGamerId` | `{ "buildings": [PlayerBuilding] }` |
| POST | `/api/v1/buildings/build` | `{ "profileId": 12, "contentId": "building_ai_center", "targetLevel": 1 }` | `{ "cost": {...}, "durationSeconds": 3600, "preview": true }` |
| POST | `/api/v1/buildings/:id/upgrade` | `{ "profileId": 12, "contentId": "building_ai_center", "targetLevel": 2 }` | `{ "cost": {...}, "durationSeconds": 4140, "preview": true }` |
| GET | `/api/v1/construction/queue?profileGamerId=12` | Query `profileGamerId` | `{ "queue": [{ "id": 1, "contentId": "...", "level": 0, "startedAt": "...", "endsAt": "...", "remainingSec": 120, "assignedWorkers": 0 }] }` |
| POST | `/api/v1/construction/start` | `{ "profileGamerId": 12, "contentId": "building_ai_center", "targetLevel": 1 }` | `{ "playerBuilding": PlayerBuilding }` |
| POST | `/api/v1/construction/:id/upgrade` | `{ "profileGamerId": 12, "targetLevel": 2 }` | `{ "playerBuilding": PlayerBuilding }` |
| POST | `/api/v1/construction/:id/speedup` | `{ "seconds": 300 }` | `{ "ok": true }` |
| POST | `/api/v1/construction/:id/cancel` | Aucun body requis | `{ "ok": true }` |
| POST | `/api/v1/construction/:id/complete` | Aucun body requis | `{ "completed": true }` |

Blocage action:

```json
{
  "error": "BUILDING_REQUIRED: building_research_lab niveau 2 requis.",
  "code": "REQUIREMENTS_NOT_MET",
  "requirementsStatus": { "allowed": false, "missing": [] }
}
```

## Base Nexus `/api/nexus-game`

| Method | Path | Attendu | Retour JSON |
| --- | --- | --- | --- |
| GET | `/health` | Aucun | `{ "status": "...", "database": "...", "redis": "..." }` |
| GET | `/debug/status` | Aucun | `{ "database": {...}, "redis": {...} }` |
| GET | `/bootstrap` | Aucun | Payload de bootstrap Nexus |

## Assets, factions, compagnons

| Method | Path | Attendu | Retour JSON |
| --- | --- | --- | --- |
| POST | `/assets/avatar` | Multipart `name`, `image` | `{ "avatar": Avatar }` |
| GET | `/assets/avatars` | Aucun | `{ "avatars": [Avatar] }` |
| PUT | `/assets/avatars/:id` | JSON champs avatar | `{ "avatar": Avatar }` ou `{ "ok": true }` |
| DELETE | `/assets/avatars/:id` | Aucun | `{ "ok": true }` |
| GET | `/factions` | Aucun | `{ "factions": [Faction] }` |
| POST | `/factions` | `Faction` JSON | `{ "faction": Faction }` |
| PUT | `/factions/:id` | JSON champs faction | `{ "faction": Faction }` |
| DELETE | `/factions/:id` | Aucun | `{ "ok": true }` |
| GET | `/ia-companions` | Aucun | `{ "companions": [IACompanion] }` |
| POST | `/ia-companions` | `IACompanion` JSON | `{ "companion": IACompanion }` |
| PUT | `/ia-companions/:id` | JSON champs compagnon | `{ "companion": IACompanion }` |
| DELETE | `/ia-companions/:id` | Aucun | `{ "ok": true }` |

## Profil joueur et plan IA quotidien

| Method | Path | Attendu | Retour JSON |
| --- | --- | --- | --- |
| GET | `/profile?user_id=7` | Query `user_id` | `{ "exists": true, "profile": ProfileGamer, "avatar_url": "...", "world_name": "...", "faction_name": "...", "starter_allocation": {"credits":450,...}, "resources": {"credits":450,...}, "city_stats": {...}, "daily_plan_context": {...} }` |
| POST | `/profile` | `{ "user_id": 7, "avatar_id": 1, "faction_id": 2, "ia_companion_id": 3, "pseudo": "Neo", "city_name": "Neon Spire" }` | `{ "exists": true, "profile": ProfileGamer, "avatar_url": "...", "world_name": "...", "faction_name": "...", "starter_allocation": {"credits":450,...}, "resources": {"credits":450,...}, "city_stats": {...}, "daily_plan_context": {...} }` |
| POST | `/profile/ia-agents` | `{ "profile_gamer_id": 12, "name": "...", "role": "...", "personality": "...", "provider": "...", "model": "...", "avatar_id": 1, "is_companion": false }` | `{ "agent": MmoIAAgent, "message": "ia agent/companion saved" }` |
| GET | `/profile/:id/ia-agents` | Path profile id | `{ "agents": [MmoIAAgent] }` |
| GET | `/profile/:id/daily-plan/context` | Path profile id | `{ "context": DailyPlanContext, "note": "..." }` |
| GET | `/profile/:id/daily-plan` | Path profile id | `{ "plan": DailyPlan, "recommendations": [DailyPlanRecommendation], "today": "YYYY-MM-DD" }` |
| POST | `/profile/:id/daily-plan/recommendations` | `{ "recommendations": [...], "generated_by": "player_provider", "summary": "..." }` | `{ "ok": true, "plan_id": 1 }` |
| POST | `/profile/:id/daily-plan/apply` | `{ "indices": [0, 2] }` | `{ "ok": true, "applied": [...], "impacts": {...}, "profile_snapshot": {...} }` |

## Ressources

| Method | Path | Attendu | Retour JSON |
| --- | --- | --- | --- |
| GET | `/resources?profileGamerId=12` | Query profil | Snapshot ressources joueur |
| GET | `/resources/catalog` | Aucun | `{ "resources": [ResourceCatalog] }` |
| GET | `/resources/transactions?profileGamerId=12&limit=50` | Query profil, `limit` optionnel | `{ "transactions": [ResourceTransaction] }` |
| GET | `/city/stats?profileGamerId=12` | Query profil | `{ "cityStats": PlayerCityStats }` |
| GET | `/daily-grant/status?profileGamerId=12` | Query profil | Statut daily grant |
| POST | `/daily-grant/claim?profileGamerId=12` | Query profil | Statut daily grant apres claim |
| GET | `/daily-grant/history?profileGamerId=12&limit=30` | Query profil | `{ "claims": [DailyGrantClaim] }` |

Notes sync économie temps réel:

- Chaque appel `GET /resources` déclenche un recalcul serveur de la production/consommation à partir des bâtiments terminés (`EffectsJSON`).
- Les montants sont ensuite accrétés côté serveur depuis `cityStats.lastProductionSyncAt`.
- Les lignes `PlayerResource` renvoient `productionPerTick`, `consumptionPerTick`, `balancePerTick` pour l'affichage live côté client.
- `GET /city/stats` renvoie aussi `lastProductionSyncAt`, `foodProduction`, `foodConsumption`, `foodBalance`.

## Admin contenu et ressources

| Method | Path | Attendu | Retour JSON |
| --- | --- | --- | --- |
| GET | `/admin/content/catalog?published=true` | Query `published` optionnelle | `{ "buildings": [...], "units": [...], "research": [...] }` |
| GET | `/admin/content/assets/status` | Aucun | `{ "rows": [...], "count": 0, "missingCount": 0 }` |
| GET | `/admin/content/translations/status?locales=fr,en,de` | Query locales optionnelle | `{ "rows": [...], "count": 0, "missingCount": 0, "locales": ["fr","en","de"] }` |
| POST | `/admin/content/upload-asset` | Multipart `file`, `contentId`, `domain=building|unit|research`, `tier` optionnel | `{ "ok": true, "savedAs": "...", "url": "...", "urlHint": "...", "publicPath": "/nexus-assets/content/..." }` |
| GET | `/admin/content/buildings` | Aucun | `{ "buildings": [BuildingDefinition], "count": 20 }` |
| GET | `/admin/content/buildings/:contentId` | Path | `{ "building": BuildingDefinition }` |
| POST | `/admin/content/buildings` | `BuildingDefinition` JSON | `{ "ok": true, "contentId": "..." }` |
| PUT | `/admin/content/buildings/:contentId` | `BuildingDefinition` JSON | `{ "ok": true, "contentId": "..." }` |
| DELETE | `/admin/content/buildings/:contentId` | Aucun | `{ "ok": true }` |
| DELETE | `/admin/content/buildings/by-id/:id` | Aucun | `{ "ok": true }` |
| POST | `/admin/content/buildings/:contentId/delete` | Aucun | `{ "ok": true }` |
| GET | `/admin/content/units` | Aucun | `{ "units": [UnitDefinition], "count": 15 }` |
| GET | `/admin/content/units/:contentId` | Path | `{ "unit": UnitDefinition }` |
| POST | `/admin/content/units` | `UnitDefinition` JSON | `{ "ok": true, "contentId": "..." }` |
| PUT | `/admin/content/units/:contentId` | `UnitDefinition` JSON | `{ "ok": true, "contentId": "..." }` |
| DELETE | `/admin/content/units/:contentId` | Aucun | `{ "ok": true }` |
| DELETE | `/admin/content/units/by-id/:id` | Aucun | `{ "ok": true }` |
| POST | `/admin/content/units/:contentId/delete` | Aucun | `{ "ok": true }` |
| GET | `/admin/content/research` | Aucun | `{ "research": [ResearchDefinition], "count": 77 }` |
| GET | `/admin/content/research/:contentId` | Path | `{ "research": ResearchDefinition }` |
| POST | `/admin/content/research` | `ResearchDefinition` JSON | `{ "ok": true, "contentId": "..." }` |
| PUT | `/admin/content/research/:contentId` | `ResearchDefinition` JSON | `{ "ok": true, "contentId": "..." }` |
| DELETE | `/admin/content/research/:contentId` | Aucun | `{ "ok": true }` |
| DELETE | `/admin/content/research/by-id/:id` | Aucun | `{ "ok": true }` |
| POST | `/admin/content/research/:contentId/delete` | Aucun | `{ "ok": true }` |
| GET | `/admin/resources/catalog` | Aucun | `{ "resources": [ResourceCatalog] }` |
| POST | `/admin/resources/seed/preview` | Aucun | `{ "resources": [...], "count": 17 }` |
| POST | `/admin/resources/seed/commit` | Aucun | `{ "status": "seeded", "count": 17 }` |
| GET | `/admin/resources/seed/status` | Aucun | `{ "expected": 17, "current": 17, "complete": true }` |

Pages HTML dev existantes:

- `GET /admin/content/buildings/page`
- `GET /admin/content/units/page`
- `GET /admin/content/research/page`

## Constructions joueur historiques

| Method | Path | Attendu | Retour JSON |
| --- | --- | --- | --- |
| GET | `/profile/:id/buildings` | Path profile id | `{ "buildings": [PlayerBuilding] }` |
| POST | `/profile/:id/construction/start` | `{ "profileId": 12, "contentId": "...", "targetLevel": 1 }` | `{ "playerBuilding": PlayerBuilding, "note": "Construction started. Complete on tick or poll /complete." }` |
| POST | `/profile/:id/construction/complete-ready` | Path profile id | `{ "completed": [1,2], "remaining": 0 }` |

## Worlds, events, prompts et generation IA

| Method | Path | Attendu | Retour JSON |
| --- | --- | --- | --- |
| GET | `/world-players?world_id=1&continent_id=2&search=neo&limit=50&offset=0&assignment=all` | Query optionnelle | Liste joueurs monde paginee |
| GET | `/worlds` | Aucun | `{ "worlds": [...] }` |
| POST | `/worlds` | Aucun | `{ "world": World }` |
| POST | `/worlds/repair-player-assignments` | Aucun | `{ "repaired": 0 }` |
| GET | `/worlds/:id` | Path world id | `{ "world": {...} }` |
| GET | `/worlds/:id/players?continent_id=2&search=neo&limit=50&offset=0` | Query optionnelle | Liste joueurs monde paginee |
| GET | `/continents` | Aucun | `{ "message": "use ListWorlds for full view with Redis capacities" }` |
| POST | `/worlds/:id/generate-event` | Path world id | `{ "world_id": 1, "proposed_event": {...}, "note": "..." }` |
| POST | `/worlds/:id/trigger-tick` | Path world id | `{ "message": "tick executed", "world_id": 1 }` |
| GET | `/prompts?domain=world` | Query `domain` optionnelle | `{ "prompts": [Prompt] }` |
| POST | `/prompts` | `Prompt` JSON | `{ "prompt": Prompt }` |
| PUT | `/prompts/:id` | JSON champs prompt | `{ "message": "prompt updated" }` |
| GET | `/ai-outputs` | Aucun | `{ "outputs": [...] }` |
| POST | `/ai/generate` | `{ "world_id": 1, "feature": "quest_seed", "prompt_id": "...", "prompt_version": "...", "extra": {} }` | `{ "success": true, "feature": "...", "world_id": 1, "output": {...}, "used_prompt": Prompt|null, "note": "..." }` |

## IA serveur publique

Tous ces endpoints sont sous `/api/nexus-game`.

| Method | Path | Attendu | Retour JSON |
| --- | --- | --- | --- |
| GET | `/ai-server/cities` | Aucun | `{ "cities": [ServerAICity] }` |
| GET | `/ai-server/cities/:id` | Path city id | `{ "city": ServerAICity }` |
| GET | `/ai-server/threat-level?profile_gamer_id=12` | Query profil | Niveau de menace joueur |
| GET | `/ai-server/attacks?user_id=7` | Query user optionnelle | `{ "attacks": [ServerAIAttack] }` |
| GET | `/ai-server/attacks/:id` | Path attack id | `{ "attack": ServerAIAttack }` |
| GET | `/ai-server/daily-broadcast` | Aucun | `{ "broadcast": ServerAIBroadcast }` |
| GET | `/seasonal-events/active` | Aucun | `{ "events": [ServerAISeasonalEvent] }` |
| GET | `/seasonal-events/upcoming` | Aucun | `{ "events": [ServerAISeasonalEvent] }` |
| GET | `/seasonal-events/:id` | Path event id | `{ "event": ServerAISeasonalEvent }` |

## Admin IA serveur

Tous ces endpoints sont sous `/api/nexus-game/admin`.

| Method | Path | Attendu | Retour JSON |
| --- | --- | --- | --- |
| GET | `/ai-server/dashboard` | Aucun | Dashboard serveur IA |
| POST | `/ai-server/worlds/:worldId/ensure-cities` | Path world id | `{ "ok": true }` |
| GET | `/ai-server/cities` | Aucun | `{ "cities": [ServerAICity] }` |
| GET | `/ai-server/cities/:id` | Path city id | `{ "city": ServerAICity }` |
| PUT | `/ai-server/cities/:id` | JSON champs ville IA | `{ "ok": true }` |
| DELETE | `/ai-server/cities/:id` | Aucun | `{ "ok": true }` |
| GET | `/ai-server/memory` | Aucun | `{ "memory": [ServerAIMemory] }` |
| GET | `/ai-server/player-memory` | Aucun | `{ "playerMemory": [ServerAIPlayerMemory] }` |
| DELETE | `/ai-server/player-memory/:id` | Aucun | `{ "ok": true }` |
| GET | `/ai-server/attacks?target_user_id=7` | Query optionnelle | `{ "attacks": [ServerAIAttack] }` |
| POST | `/ai-server/attacks/schedule` | `ScheduleAttackRequest` JSON | `{ "attack": ServerAIAttack }` |
| POST | `/ai-server/attacks/:id/cancel` | Aucun | `{ "ok": true }` |
| POST | `/ai-server/attacks/:id/resolve` | `{ "result": "success|failed|..." }` | `{ "attack": ServerAIAttack }` |
| DELETE | `/ai-server/attacks/:id` | Aucun | `{ "ok": true }` |
| GET | `/ai-server/sabotages` | Aucun | `{ "sabotages": [ServerAISabotage] }` |
| POST | `/ai-server/sabotages/:id/cancel` | Aucun | `{ "ok": true }` |
| GET | `/ai-server/espionage` | Aucun | `{ "espionage": [ServerAIEspionage] }` |
| DELETE | `/ai-server/espionage/:id` | Aucun | `{ "ok": true }` |
| GET | `/ai-server/broadcasts` | Aucun | `{ "broadcasts": [ServerAIBroadcast] }` |
| POST | `/ai-server/broadcasts/generate?world_id=1` | Query world optionnelle | `{ "broadcast": ServerAIBroadcast }` |
| POST | `/ai-server/daily-broadcast/generate?world_id=1` | Query world optionnelle | `{ "broadcast": ServerAIBroadcast }` |
| PUT | `/ai-server/broadcasts/:id` | JSON champs broadcast | `{ "ok": true }` |
| POST | `/ai-server/broadcasts/:id/publish` | Aucun | `{ "broadcast": ServerAIBroadcast }` |
| POST | `/ai-server/daily-broadcast/publish` | Utilise id selon service | `{ "broadcast": ServerAIBroadcast }` |
| DELETE | `/ai-server/broadcasts/:id` | Aucun | `{ "ok": true }` |
| GET | `/ai-server/prompts` | Aucun | `{ "prompts": [Prompt] }` |
| POST | `/ai-server/prompts` | `Prompt` JSON | `{ "prompt": Prompt }` |
| PUT | `/ai-server/prompts/:id` | JSON champs prompt | `{ "ok": true }` |
| DELETE | `/ai-server/prompts/:id` | Aucun | `{ "ok": true }` |
| POST | `/ai-server/prompts/:id/test` | Aucun | `{ "result": {...} }` |
| POST | `/ai-server/prompts/seed` | Aucun | `{ "ok": true }` |
| GET | `/ai-server/call-logs?limit=100` | Query limit optionnelle | `{ "logs": [ServerAICallLog] }` |
| GET | `/ai-server/costs` | Aucun | Cout IA agrege |
| GET | `/seasonal-events/proposals` | Aucun | `{ "events": [...] }` |
| POST | `/seasonal-events/propose-by-ai?world_id=1` | Query world | `{ "event": ServerAISeasonalEvent }` |
| GET | `/seasonal-events?status=draft,proposed` | Query status optionnelle | `{ "events": [...] }` |
| GET | `/seasonal-events/:id` | Path event id | `{ "event": ServerAISeasonalEvent }` |
| PUT | `/seasonal-events/:id` | JSON champs evenement | `{ "ok": true }` |
| DELETE | `/seasonal-events/:id` | Aucun | `{ "ok": true }` |
| POST | `/seasonal-events/:id/approve` | `{ "reason": "..." }` optionnel | `{ "event": ServerAISeasonalEvent }` |
| POST | `/seasonal-events/:id/reject` | `{ "reason": "..." }` optionnel | `{ "event": ServerAISeasonalEvent }` |
| POST | `/seasonal-events/:id/schedule` | `{ "reason": "..." }` optionnel | `{ "event": ServerAISeasonalEvent }` |
| POST | `/seasonal-events/:id/start` | `{ "reason": "..." }` optionnel | `{ "event": ServerAISeasonalEvent }` |
| POST | `/seasonal-events/:id/end` | `{ "reason": "..." }` optionnel | `{ "event": ServerAISeasonalEvent }` |
| POST | `/seasonal-events/:id/archive` | `{ "reason": "..." }` optionnel | `{ "event": ServerAISeasonalEvent }` |
