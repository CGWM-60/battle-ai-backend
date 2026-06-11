# Nexus Game — Guide complet Flutter (temps réel + re-sync)

Ce document décrit le flux recommandé pour synchroniser l'économie/bâtiments entre Flutter et le backend, avec serveur autoritaire.

## 1) Principes d'architecture

- **Serveur autoritaire** : les montants finaux de ressources proviennent toujours du backend.
- **Client prédictif** : Flutter peut animer localement les compteurs via `balancePerTick`, puis corriger à chaque refresh serveur.
- **Re-sync fréquent mais léger** : cadence plus élevée sur la construction, plus basse sur le reste.

## 2) Endpoints à utiliser

### Hydratation initiale (après login / ouverture ville)

1. `GET /api/nexus-game/profile?user_id={id}`
   - Récupère `profile`, `starter_allocation`, `resources`, `city_stats`.
2. `GET /api/v1/construction/queue?profileId={profileId}`
   - Récupère progression serveur de la file de construction (`remainingSec`, `startedAt`, `endsAt`).
3. (optionnel) `GET /api/nexus-game/resources?profileGamerId={profileId}`
   - Force un snapshot économie complet si besoin de revalidation immédiate.

### Boucle live (écran ville ouvert)

- `GET /api/v1/construction/queue?profileId={profileId}` toutes les **3-5s**.
- `GET /api/nexus-game/resources?profileGamerId={profileId}` toutes les **8-15s**.
- `GET /api/nexus-game/city/stats?profileGamerId={profileId}` toutes les **15-30s** (ou inclus dans profile/resources selon écran).

### Actions joueur

- Démarrer construction: `POST /api/v1/construction/start`
- Upgrade: `POST /api/v1/construction/:id/upgrade`
- Speedup/cancel/complete: endpoints v1 dédiés
- Après chaque action réussie:
  - refresh immédiat de `construction/queue`
  - refresh immédiat de `resources`

## 3) Contrat utile côté Flutter

## `PlayerResource`

Utiliser ces champs pour l'UI live:

- `amount`: valeur serveur à l'instant du snapshot
- `productionPerTick`
- `consumptionPerTick`
- `balancePerTick = productionPerTick - consumptionPerTick`

## `PlayerCityStats`

- `foodProduction`
- `foodConsumption`
- `foodBalance`
- `populationCapacity`
- `populationFree`
- `populationGrowthPerHour`
- `populationRemainder`
- `lastPopulationSyncAt` (**pivot temporel serveur population**)
- `lastProductionSyncAt` (**pivot temporel serveur**)

## `ProfileGamer`

- `population`: population actuelle.
- `populationCapacity`: limite serveur calculee depuis les habitats termines et les bonus de recherche.
- `energyProduction`: production d'energie par heure, pour affichage.
- `energyConsumption`: consommation d'energie par heure, pour affichage.
- `energyBalance`: `energyProduction - energyConsumption`, par heure.

Important: `ProfileGamer.energy*` est horaire, alors que `PlayerResource.*PerTick` est par seconde. Flutter anime les stocks avec `balancePerTick`, mais affiche les limites population/energie depuis `ProfileGamer`.

Important population: Flutter ne renvoie jamais une population predite au backend. Le backend est source de verite: `GET /resources` synchronise production + croissance population, puis renvoie `profile` et `city` pour recaler `mmoEntryProvider`.

Formules serveur utiles a afficher:

- capacite habitat niveau N: `500 + 250*(N-1)` par `building_modular_habitat` termine avec la configuration par defaut, avant bonus de recherche.
- nourriture consommee: `population * 0.08 / heure`.
- energie consommee: base `5/h + population/20`, plus surcharge des batiments termines.

## 4) Algorithme UI live recommandé

Soit:

- $A_0$ = `amount` reçu du serveur
- $b$ = `balancePerTick`
- $t_0$ = timestamp local à la réception du snapshot
- $t$ = timestamp courant

Affichage client:

$$
A_{ui}(t) = A_0 + b \cdot (t - t_0)
$$

Règles:

- clamp à $\ge 0$
- si capacité connue ($capacity > 0$), clamp à $\le capacity$
- ne **jamais** persister cette valeur en local comme vérité; c'est une valeur d'affichage.

## 5) Re-sync (app resume, reconnect, foreground)

Déclencheurs de re-sync complet:

- retour en foreground
- reconnect réseau
- changement d'écran vers city/base
- après erreur API 409/422/5xx sur construction

Procédure:

1. stop timer UI local
2. `GET /profile`
3. `GET /construction/queue`
4. `GET /resources`
5. redémarrer timer UI avec les nouvelles bases

## 6) Gestion des conflits et erreurs

- **Construction non prête** (`complete` trop tôt): conserver le timer actuel, re-poller queue.
- **Prérequis manquants** (`start/upgrade`): afficher message backend tel quel, puis refresh `profile + resources`.
- **Déconnexion**:
  - garder l'animation locale max 30-60s,
  - au-delà, geler l'affichage et afficher état “resync pending”.

## 7) Machine d'état Flutter conseillée

États globaux:

- `hydrating`
- `live`
- `resyncing`
- `degraded_offline`

Transitions:

- `hydrating -> live` après succès profile+queue
- `live -> resyncing` sur trigger (resume/reconnect/action importante)
- `resyncing -> live` après refresh complet
- `live -> degraded_offline` après timeout réseau répétés

## 8) Stratégie de polling recommandée

- Queue construction: 3-5s (priorité UX)
- Ressources: 8-15s
- City stats: 15-30s
- En background: stopper les pollings, re-sync au resume.

## 9) Bonnes pratiques d'implémentation Flutter

- Une source de vérité centralisée (`Riverpod`/`Bloc`/`Cubit`) pour profile/resources/queue.
- Mettre `serverNow` (si disponible) et `receivedAt` pour corriger les dérives de temps.
- Debounce des refreshs après actions multiples (ex: speedup + complete).
- Toujours fusionner par clé ressource (`resourceCode`) et non par index de liste.

## 10) Checklist QA mobile

- [ ] Un bâtiment terminé modifie bien `productionPerTick` après refresh.
- [ ] Les compteurs UI évoluent même entre deux polls.
- [ ] Après fermeture/réouverture app, les montants se recalent sans saut incohérent.
- [ ] Après perte réseau, reprise correcte via re-sync complet.
- [ ] `construction/queue` et `resources` restent cohérents après `complete`.

---

## Résumé

- Le backend calcule désormais la production à partir des bâtiments terminés.
- `GET /resources` sert de point de vérité pour l'économie live.
- Flutter peut afficher en continu via `balancePerTick`, puis corriger régulièrement via re-sync serveur.
