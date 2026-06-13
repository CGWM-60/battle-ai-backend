# Nexus Unit Rarity & Research Audit

Date: 2026-06-13

## Périmètre vérifié

Audit basé sur :
- backend Go `internal/nexus_game`
- Flutter Nexus `lib/nexus_games`
- docs contractuels du workspace
- service legacy de recherche/armée hors `internal/nexus_game` quand pertinent pour éviter les doublons

Règle de lecture : le backend Go reste la source de vérité gameplay.

---

## 1. Modèles GORM existants liés aux unités

### Dans `internal/nexus_game/models/unit_definition.go`
- `UnitDefinition`
  - `ContentID`, `Domain`, `Type`
  - traductions, assets, `MaxLevel`
  - `Rarity` en simple `string`
  - `NexusLevelRequired`
  - `RequiredBuildingsJSON`
  - `RequiredResearchJSON`
  - stats de base: `HealthBase`, `AttackBase`, `DefenseBase`, `SpeedBase`
  - économie: `TrainingTimeBaseSeconds`, `UpkeepBase`
  - `EffectsJSON`, `BalanceVersion`, `IsPublished`
- `PlayerUnit`
  - `ProfileGamerID`, `ContentID`, `Count`
  - quantités annexes déjà prévues: `ReserveQuantity`, `AssignedQuantity`, `WoundedQuantity`, `DamagedQuantity`, `InactiveQuantity`, `DestroyedQuantity`

### Dans `internal/nexus_game/models/army.go`
- `UnitTrainingQueue`
  - file d’entraînement persistée, liée à `ProfileGamerID`, `WorldID`, `UnitCode`
  - `Status`, `StartedAt`, `CompletedAt`, `ClaimedAt`, `CancelledAt`
  - `CostJSON`, `RefundJSON`, `ReservedCapacity`

### Observation
- Le catalogue unité existe déjà.
- La rareté existe déjà comme champ texte, mais pas comme enum canonique.
- Les unités ont déjà les champs nécessaires pour prérequis, coût et temps d’entraînement.

---

## 2. Modèles GORM existants liés au catalogue d’unités

### Catalogue côté Go
- `UnitDefinition` est le modèle catalogue officiel.
- Le service `ContentService` expose le catalogue via :
  - `ListUnits(publishedOnly bool)`
  - `GetUnit(contentID string)`
  - `CreateOrUpdateUnit(def *models.UnitDefinition)`
  - `DeleteUnit(contentID string)`

### Traitement de la rareté et du coût
- `internal/nexus_game/content/balance/formulas.go`
  - `ApplyRarityMultiplier(base int, rarity string, isDuration bool)`
  - rarités déjà reconnues: `common`, `uncommon`, `rare`, `epic`, `legendary`, `nexus`
- `ContentService.CalculateBuildingCostAtLevel` réutilise déjà cette logique pour les bâtiments.

### Point important
- Le catalogue d’unités ne possède pas encore une enum/const globale dédiée à la rareté.
- La logique de rareté existe au niveau formule, mais pas encore comme contrat centralisé partagé par toutes les couches.

---

## 3. Modèles GORM existants liés aux recherches

### Dans `internal/nexus_game/models/research_definition.go`
- `ResearchDefinition`
  - `ContentID`, `Domain`, `Branch`
  - `Tier`
  - `Rarity` en `string`
  - `CostBaseCredits`, `CostBaseData`, `DurationBaseSeconds`
  - `EffectsJSON`
  - `NexusLevelRequired`
  - `RequiredBuildingsJSON`
  - `PrerequisitesJSON`
  - `BalanceVersion`, `IsPublished`
- `PlayerResearch`
  - `ProfileGamerID`, `ContentID`, `CompletedAt`

### Observation
- Les recherches existent déjà comme catalogue persistant.
- Les dépendances sont déjà stockées en JSON.
- La branche de recherche est déjà un champ de modèle, pas une table dédiée.

---

## 4. Modèles GORM existants liés aux branches de recherche

### Dans le modèle Nexus actuel
- `ResearchDefinition.Branch` est une chaîne simple.
- Aucune enum GORM dédiée pour les branches n’existe actuellement.

### Branches seedées
Le seed de recherche actuel couvre notamment :
- `economy`
- `energy`
- `ville`
- `militaire`
- `drones`
- `ia`
- `diplomatie`
- `monde`
- `guilde`
- `lore`
- `tribunal`

### Legacy hors `internal/nexus_game`
- `internal/models/research_*` et `internal/service/research_service.go` existent aussi.
- Ce système legacy manipule `ResearchTreeDefinition`, `ResearchNodeDefinition` et `PlayerSave.ResearchJSON`.
- Il ne faut pas le dupliquer pour le nouveau besoin Nexus.

### Conclusion
- La branche existe déjà, mais seulement comme valeur textuelle.
- Si une enum est ajoutée, elle doit rester compatible avec les valeurs seedées.

---

## 5. Services existants liés à l’entraînement d’unités

### `internal/nexus_game/services/army_service.go`
Service principal actuel pour les unités Nexus.

Fonctions importantes :
- `Catalog(ctx, publishedOnly)`
- `PlayerUnits(ctx, profileID)`
- `TrainingQueue(ctx, profileID, includeDone)`
- `Capacity(ctx, profileID)`
- `TrainUnit(ctx, req)`
- `RefreshTrainingQueue(ctx, profileID)`
- `ClaimTraining(ctx, profileID, queueID)`
- `CancelTraining(ctx, profileID, queueID)`
- `AssignUnit(ctx, profileID, formationID, slotID, req)`
- `RemoveUnit(ctx, profileID, formationID, slotID, req)`
- `RecalculateFormation(ctx, formationID)`

### Ce que fait déjà `TrainUnit`
- valide `profileGamerId`
- valide `unitCode`
- valide la quantité
- charge la définition d’unité
- bloque les unités non publiées
- appelle `ValidatePrerequisites(profileID, "unit", unitCode)`
- vérifie la capacité militaire
- vérifie les slots d’entraînement
- débite les ressources
- crée une `UnitTrainingQueue`
- journalise l’opération

### Formules déjà présentes
- coût d’entraînement: `trainingCostForUnit`
- durée: `trainingDurationForUnit`
- capacité consommée: `capacityCostForUnit`
- puissance: `unitPower`

### Limite actuelle
- la rareté n’est pas encore une règle métier centrale dans `TrainUnit`.
- l’accès à l’unité dépend surtout des prérequis JSON et du seed.
- la complexité/cout/durée selon rareté n’est pas encore un mécanisme complet et uniforme pour toutes les unités.

---

## 6. Services existants liés à la validation des prérequis

### `internal/nexus_game/services/content_service.go`
Fonction centrale : `ValidatePrerequisites(profileID uint, domain string, contentID string)`

Elle valide déjà :
- `building`
- `unit`
- `research`

Elle sait parser des prérequis JSON sous plusieurs formes :
- tableaux d’objets
- tableaux de strings
- objet imbriqué `buildings` / `research`
- anciens alias (`buildingKey`, `researchKey`, `minLevel`, etc.)

Elle s’appuie sur :
- `normalizeDomain`
- `nexusLevelRequirement`
- `parseBuildingRequirements`
- `parseResearchRequirements`
- `playerRequirementState`
- `applyBuildingOrUnlock`

### Ce qui est déjà important pour ce ticket
- la validation serveur existe déjà
- les prérequis bâtiment / Nexus / recherche sont déjà supportés
- le backend peut déjà refuser une unité si les conditions ne sont pas remplies

### Limite actuelle
- il n’existe pas encore de règle générique disant :
  - `common` => bâtiment + niveau suffisent
  - `uncommon/rare/epic/legendary/nexus` => recherche militaire obligatoire
- cette règle doit donc être ajoutée dans la logique serveur, pas dans Flutter.

---

## 7. Routes backend existantes

### Routes publiques Flutter
Dans `internal/nexus_game/routes/routes.go` et `internal/nexus_game/handlers/content_handler.go` :
- `GET /api/v1/prerequisites/validate`
- `GET /api/v1/buildings/catalog`
- `GET /api/v1/buildings/catalog/version`
- `GET /api/v1/buildings/:key`
- `GET /api/v1/buildings/:key/research-tree`
- `GET /api/v1/units/catalog`
- `GET /api/v1/units/:key`
- `GET /api/v1/research/catalog`
- `GET /api/v1/research/:key`
- `GET /api/v1/assets/buildings/manifest`
- `GET /api/v1/assets/buildings/updates`
- `GET /api/v1/buildings`
- `POST /api/v1/buildings/destroy`
- `POST /api/v1/buildings/build`
- `POST /api/v1/buildings/:id/upgrade`
- `GET /api/v1/construction/queue`
- `POST /api/v1/construction/start`
- `POST /api/v1/construction/:id/upgrade`
- `POST /api/v1/construction/:id/speedup`
- `POST /api/v1/construction/:id/cancel`
- `POST /api/v1/construction/:id/complete`

### Routes Nexus gameplay
- `GET /api/nexus-game/units/catalog`
- `GET /api/nexus-game/units`
- `GET /api/nexus-game/units/training-queue`
- `POST /api/nexus-game/units/train`
- `POST /api/nexus-game/units/training/:id/cancel`
- `POST /api/nexus-game/units/training/:id/claim`
- `GET /api/nexus-game/army/progression`
- `GET /api/nexus-game/army/formations`
- `GET /api/nexus-game/army/formations/:id`
- `POST /api/nexus-game/army/formations/:id/recalculate`
- `POST /api/nexus-game/army/formations/:id/slots/:slotId/assign`
- `POST /api/nexus-game/army/formations/:id/slots/:slotId/remove`
- `POST /api/nexus-game/army/formations/:id/commander-ai/suggest`

### Admin content
- CRUD admin pour unités/recherches/bâtiments sous `/api/nexus-game/admin/content/...`
- pages admin existantes pour tables/unités/recherches

### Observation
- les routes de catalogue et validation existent déjà
- les routes de training unit existent déjà
- il manque surtout un pont métier clair entre rareté et recherche militaire

---

## 8. Seeds existants pour unités

### `internal/nexus_game/seeds/content_seed.go`
Fonction : `SeedInitialUnits(db, svc)`

Elle seed 15 unités.

Raretés déjà seedées :
- `common`
- `uncommon`
- `rare`
- `epic`
- `nexus`

### Exemples de prérequis déjà seedés
- `unit_milicien_nexus` → Nexus level minimum
- `unit_drone_sentinelle` → bâtiment `building_drone_factory` + recherche `research_drone_assembly`
- `unit_fantassin_augmente` → bâtiment `building_barracks` + recherche `research_augmented_infantry`
- `unit_drone_assaut` → `building_drone_factory` + `research_swarm_control`
- `unit_drone_bouclier` → `building_drone_factory` + `research_shield_drone_matrix`
- `unit_hacker_de_combat` → `building_ai_center` + `research_multi_agent_routing`
- `unit_artillerie_railgun` → `building_barracks` + `research_railgun_engineering`
- `unit_mecha_leger` → plusieurs bâtiments + `research_battlefield_logistics`
- `unit_titan_nexus` → rareté `nexus`

### Point important
- les seeds ont déjà une logique de montée en difficulté
- mais la règle “toutes les unités non common nécessitent une recherche militaire correspondante” n’est pas encore centralisée et garantie par une règle unique

---

## 9. Seeds existants pour recherches

### `internal/nexus_game/seeds/content_seed.go`
Fonction : `SeedInitialResearch(db, svc)`

Le seed couvre :
- 11 branches
- 7 tiers par branche
- `common`, `uncommon`, `rare`, `epic`, `legendary`, `nexus` selon les nœuds

### Branche militaire déjà seedée
- `research_basic_tactics`
- `research_augmented_infantry`
- `research_squad_coordination`
- `research_railgun_engineering`
- `research_battlefield_logistics`
- `research_elite_doctrine`
- `research_nexus_warfare`

### Observation
- la branche militaire existe déjà et peut servir de base à la règle de débloquage par rareté
- les coûts et durées progressent déjà par tier dans le seed
- la recherche “réseau” pour les unités non common est donc faisable sans créer de nouveau système parallèle

---

## 10. Pages Flutter existantes liées aux unités / recherches

### Pages Flutter Nexus repérées
Dans `lib/nexus_games/mmo/presentation/screens/` :
- `mmo_unites_screen.dart`
- `mmo_recherche_technologique_screen.dart`
- `mmo_all_screens_navigator.dart`
- `mmo_army_composer_screen.dart`
- `mmo_building_detail_screen.dart`
- `mmo_construction_batiments_screen.dart`
- `nexus_city_dashboard_screen.dart`

### Constat sur l’existant
- `mmo_unites_screen.dart` est actuellement un écran maquette avec données codées en dur.
- `mmo_recherche_technologique_screen.dart` est également une maquette statique.
- `mmo_all_screens_navigator.dart` route bien vers ces deux écrans.
- `mmo_construction_batiments_screen.dart` montre déjà le bon pattern d’intégration API + cache + refresh serveur.

### Ce qu’il faut retenir
- les écrans existent déjà
- ils ne sont pas encore branchés au vrai catalogue unités / recherche
- ils doivent être adaptés, pas remplacés

---

## 11. Providers Flutter existants

### Providers repérés
Dans `lib/nexus_games/mmo/providers/` :
- `mmoUnitesProvider.dart`
- `mmoRechercheTechnologiqueProvider.dart`
- `nexus_api_service_provider.dart`
- `mmoConstructionBatimentsProvider.dart`
- `mmoEntryProvider.dart` (utile pour le `profileGamerId`)

### État réel
- `mmoUnitesProvider.dart`
  - état entièrement mocké
  - aucune donnée backend
- `mmoRechercheTechnologiqueProvider.dart`
  - état mocké
  - `launchResearch()` est encore un TODO
- `nexus_api_service_provider.dart`
  - expose le service réseau
- `NexusApiService`
  - contient déjà les endpoints unitaires et constructions
  - ne contient pas encore de vrai client recherche Nexus dédié

### Pattern utile
- `mmoConstructionBatimentsProvider.dart` montre comment consommer le backend, les prérequis, le cache et les refreshs
- c’est le meilleur modèle à réutiliser pour unités et recherches

---

## 12. Ce qui est déjà en place

- Le backend contient déjà les modèles catalogue pour unités et recherches.
- Les rarités officielles existent déjà dans le code et les seeds.
- `ApplyRarityMultiplier` connaît déjà les 6 rarités officielles.
- `ValidatePrerequisites` sait déjà valider bâtiment, unité et recherche.
- Les seeds fournissent déjà une progression de rareté et de tiers.
- Les routes publiques `/api/v1` exposent déjà les catalogues et la validation.
- Les routes de training unit existent déjà côté gameplay.
- Flutter a déjà les écrans et providers à adapter.

---

## 13. Ce qui manque

- une enum/const canonique des rarités partagée et validée partout
- une règle serveur unique reliant rareté d’unité ↔ recherche militaire obligatoire
- une source de vérité claire pour associer une rareté à une branche / recherche de déblocage
- une complexité de recherche plus explicite pour les unités de rareté élevée
- un calcul de disponibilité unifié exposant clairement : bâtiment + niveau + recherche + rareté
- un vrai provider Flutter pour unités relié au backend
- un vrai provider Flutter pour recherche technologique relié au backend
- un écran Flutter recherche branché sur le catalogue réel
- un écran Flutter unités branché sur le catalogue réel
- des tests ciblés sur la règle de rareté et ses prérequis

---

## 14. Ce qui doit être adapté

### Backend Go
- `models.UnitDefinition`
- `models.ResearchDefinition`
- `services.ContentService.ValidatePrerequisites`
- `services.ArmyService.TrainUnit`
- `services.ArmyService.Catalog`
- `seeds.content_seed.go`
- éventuellement les helpers de formule côté `content/balance`

### Flutter
- `mmoUnitesProvider.dart`
- `mmoRechercheTechnologiqueProvider.dart`
- `mmo_unites_screen.dart`
- `mmo_recherche_technologique_screen.dart`
- `NexusApiService`
- `NexusApiServiceProvider`

### Architecture à garder
- backend Go = source de vérité gameplay
- Flutter = affichage + actions utilisateur
- admin Next.js = pilotage via API Go uniquement

---

## 15. Ce qui ne doit surtout pas être dupliqué

- ne pas recréer un second catalogue d’unités
- ne pas recréer un second système de rareté en parallèle du modèle existant
- ne pas dupliquer la logique de validation des prérequis dans Flutter
- ne pas refaire un legacy `WorldGameService` pour le gameplay Nexus moderne
- ne pas utiliser de SQL brut
- ne pas créer de logique gameplay dans Flutter
- ne pas introduire une nouvelle branche de recherche isolée sans réutiliser les seeds et le catalogue existants

---

## 16. Plan d’implémentation point par point

1. Introduire une enum/const backend pour les rarités officielles.
2. Ajouter un mapping serveur clair entre rareté d’unité et recherche militaire requise.
3. Centraliser une fonction de disponibilité d’unité qui combine :
   - bâtiment requis
   - niveau du bâtiment
   - rareté
   - recherche débloquée
   - Nexus level
4. Rendre la progression plus explicite dans les seeds et/ou helpers de calcul : coût, durée, difficulté.
5. Adapter `TrainUnit` pour bloquer proprement les unités non common tant que la recherche correspondante n’est pas débloquée.
6. Exposer ces règles dans les réponses catalogue / validation pour Flutter.
7. Remplacer les providers mock Flutter par des providers API-backed.
8. Relier les écrans unités et recherche technologique aux endpoints réels.
9. Ajouter des tests Go sur la disponibilité des unités selon rareté et recherche.
10. Vérifier qu’aucune logique gameplay ne fuit côté Flutter.

---

## Note finale

L’existant est déjà très proche de la cible : le catalogue, les prérequis et les routes sont là. Le vrai travail consiste surtout à **centraliser la règle de rareté/recherche militaire** et à **brancher les écrans Flutter sur le backend existant** sans recréer de système parallèle.
