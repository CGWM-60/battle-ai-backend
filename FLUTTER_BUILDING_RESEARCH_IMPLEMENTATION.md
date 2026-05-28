# Plan Flutter - Gestion batiment et arbre de recherche

## Objectif

Ajouter une page Flutter de gestion de batiment avec le flux suivant :

1. Afficher la liste/carte des batiments du joueur.
2. Cliquer sur un batiment.
3. Ouvrir la fiche d'information du batiment.
4. Charger l'arbre de recherche lie a ce batiment.
5. Permettre de lancer une recherche et de rafraichir l'etat de progression.

Les donnees P2W du document source ne sont pas exposees par l'API actuelle. Le client consomme uniquement les durees F2P via `levelProgressionJson`.

## Endpoints backend a brancher

Tous les endpoints ci-dessous sont authentifies comme les routes monde actuelles. Le prefixe HTTP backend est `/api/v1`.

- `GET /buildings/catalog`
  - Retourne `{ "buildings": BuildingDefinition[] }`.
  - Chaque batiment contient `researchTreeKey`.

- `GET /buildings/{buildingKey}`
  - Retourne la definition complete du batiment.

- `GET /buildings/{buildingKey}/research-tree`
  - Retourne `{ resources, trees, progress }`.
  - `trees` contient les noeuds actifs de l'arbre lie au batiment.
  - `progress.nodes[nodeKey].level` donne le niveau joueur.
  - `progress.active` donne la recherche en cours.

- `GET /research/catalog?buildingKey={buildingKey}`
  - Variante filtrable pour charger l'arbre d'un batiment.

- `GET /research/state`
  - Retourne uniquement l'etat joueur des recherches.

- `POST /research/{nodeKey}/start`
  - Lance le niveau suivant du noeud.
  - Retourne `{ node, progress }`.

- `POST /research/{nodeKey}/complete`
  - Termine une recherche si `finishAt <= now`.
  - Retourne `{ node, progress }`.

## Modeles Dart a creer

- `BuildingDefinition`
  - `id`, `key`, `name`, `description`, `category`, `researchTreeKey`, `maxLevel`
  - `baseCostJson`, `levelCostFormulaJson`, `effectsJson`, `unlockRequirementsJson`
  - `isActive`, `sortOrder`, `assets`

- `ResearchCatalogResponse`
  - `List<ResourceDefinition> resources`
  - `List<ResearchTreeDefinition> trees`
  - `PlayerResearchProgress progress`

- `ResearchTreeDefinition`
  - `id`, `key`, `name`, `description`, `domain`, `buildingKey`, `isActive`, `sortOrder`
  - `List<ResearchNodeDefinition> nodes`

- `ResearchNodeDefinition`
  - `id`, `researchTreeDefinitionId`, `key`, `name`, `description`, `domain`, `branch`
  - `resourcesJson: List<String>`
  - `parentKeysJson: List<String>`
  - `requirementsJson: Map<String, dynamic>`
  - `effectsJson: Map<String, dynamic>`
  - `levelProgressionJson: List<ResearchLevelProgression>`
  - `maxLevel`, `positionX`, `positionY`, `isActive`, `sortOrder`

- `ResearchLevelProgression`
  - `level`, `durationMinutes`, `cumulativeMinutes`

- `PlayerResearchProgress`
  - `Map<String, ResearchNodeProgress> nodes`
  - `ActiveResearchProgress? active`

- `ResearchNodeProgress`
  - `level`, `status`, `startedAt`, `completedAt`

- `ActiveResearchProgress`
  - `nodeKey`, `targetLevel`, `startedAt`, `finishAt`

## Architecture Flutter conseillee

1. `BuildingRepository`
   - `Future<List<BuildingDefinition>> fetchCatalog()`
   - `Future<BuildingDefinition> fetchBuilding(String buildingKey)`
   - `Future<ResearchCatalogResponse> fetchResearchTree(String buildingKey)`
   - `Future<PlayerResearchProgress> fetchResearchState()`
   - `Future<ResearchActionResult> startResearch(String nodeKey)`
   - `Future<ResearchActionResult> completeResearch(String nodeKey)`

2. `BuildingManagementController`
   - Etat : `loading`, `error`, `buildings`, `selectedBuilding`, `researchCatalog`
   - Actions : `loadBuildings`, `selectBuilding`, `startResearch`, `completeResearch`, `refreshResearch`

3. Ecrans/widgets
   - `BuildingManagementPage`
     - Grille ou liste des batiments possedes.
     - Selection d'un batiment par `buildingKey`.
   - `BuildingDetailSheet` ou `BuildingDetailPage`
     - Infos batiment : nom, niveau, description, effets, couts, requirements.
     - Bouton/onglet `Recherche`.
   - `BuildingResearchTreeView`
     - Regroupe les noeuds par `tree.domain`.
     - Positionne les noeuds avec `positionX/positionY` si une vue canvas est utilisee.
     - Affiche niveau courant, niveau max, ressources, duree du prochain niveau.
   - `ResearchNodeCard`
     - Etats : verrouille, disponible, en cours, termine au niveau max.
     - CTA : `Rechercher`, `Terminer`, ou compteur `finishAt`.

## Flux utilisateur detaille

1. A l'entree de page, appeler `GET /buildings/catalog`.
2. Afficher les batiments par categorie.
3. Au clic sur un batiment :
   - charger `GET /buildings/{buildingKey}` si le detail local est incomplet ;
   - charger `GET /buildings/{buildingKey}/research-tree` ;
   - ouvrir la vue detail.
4. Dans l'onglet recherche :
   - lire `progress.nodes[node.key]?.level ?? 0` ;
   - calculer `nextLevel = currentLevel + 1` ;
   - trouver la duree dans `node.levelProgressionJson.firstWhere(level == nextLevel)`.
5. Au clic `Rechercher`, appeler `POST /research/{nodeKey}/start`.
6. Si `progress.active.nodeKey == node.key`, afficher le compte a rebours jusqu'a `finishAt`.
7. Quand le compteur arrive a zero, appeler `POST /research/{nodeKey}/complete`.
8. Apres chaque action, remplacer l'etat local par le `progress` retourne.

## Gestion erreurs et etats limites

- Si `trees` est vide pour un batiment, afficher un etat vide sobre : aucune recherche disponible.
- Si l'API retourne `another research is already active`, rediriger visuellement vers le noeud actif.
- Si `research node already maxed`, desactiver le CTA et afficher `Niveau max`.
- Si `research is not finished`, recalculer le temps restant depuis `finishAt`.
- Toujours comparer les dates en UTC ou ISO parsees par `DateTime.parse`.

## Definition of done Flutter

- La page affiche le catalogue batiment.
- Un clic batiment ouvre sa fiche detail.
- L'onglet recherche charge l'arbre lie au `buildingKey`.
- Les niveaux existants et l'eventuelle recherche active sont visibles.
- `startResearch` et `completeResearch` mettent a jour l'UI sans redemarrage.
- Le client ne depend d'aucune donnee P2W.
