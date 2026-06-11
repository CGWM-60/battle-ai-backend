# Nexus Game — Règles complètes du jeu

Ce document décrit les règles **réellement appliquées par le backend `nexus_game`** à ce jour.  
Il sert de référence canonique pour Flutter, le panneau admin et les futures évolutions.

---

## 1) Philosophie générale

- **Serveur autoritaire** : toute vérité de jeu vient du backend.
- **Le client affiche, le serveur décide** : Flutter ne fait qu’anticiper visuellement et re-synchroniser.
- **Évolution continue** : les villes, la production, les recherches et les événements évoluent au fil des ticks ou des syncs manuelles.
- **Règles data-driven** : les bâtiments utilisent leurs `EffectsJSON`, les recherches ajoutent des modificateurs, les ressources sont calculées à partir des bâtiments terminés.

---

## 2) Entités principales

### Profil joueur (`ProfileGamer`)

Le profil est la base de toute progression.

Il contient notamment :
- population
- capacité de population
- morale
- sécurité
- production d’énergie
- consommation d’énergie
- balance d’énergie
- énergie stockée
- monde / faction / avatar / compagnon IA

### Ville / statistiques de ville (`PlayerCityStats`)

Statistiques globales de la ville :
- capacité de stockage
- production / consommation / balance de nourriture
- timestamp de dernière sync de production

### Ressources joueur (`PlayerResource`)

Chaque ressource officielle a une ligne dédiée par joueur.

Champs importants :
- `amount`
- `capacity`
- `productionPerTick`
- `consumptionPerTick`
- `balancePerTick`

### Bâtiments joueur (`PlayerBuilding`)

Un bâtiment appartient à un profil, possède :
- `contentId`
- `level`
- état de construction ou non
- timestamps de début / fin de construction

### Recherches joueur (`PlayerResearch`)

Une recherche complétée est enregistrée par `profileGamerId + contentId`.

---

## 3) Ressources officielles

Les ressources officielles incluent notamment :
- `population`
- `credits`
- `food`
- `energy`
- `metal`
- `components`
- `data`
- `influence`
- `guild_marks`
- `tokens`
- `provider_budget`
- `inference_credit`
- `local_compute`
- `agent_focus`
- `quantum_core`
- `neural_fiber`
- `void_fragment`

### Allocation de départ

Le joueur reçoit une allocation de départ basée sur `InitialAmount` de la ressource.

Exemples actuels :
- `credits`: 450
- `food`: 500
- `energy`: 300
- `metal`: 800
- `components`: 120
- `data`: 100
- `influence`: 25
- `tokens`: 50
- `inference_credit`: 20
- `local_compute`: 10
- `agent_focus`: 5

Les ressources rares commencent à 0.

### Règles de stockage

- Les ressources marquées comme stockables ont une capacité par défaut.
- La capacité est recalculée par le moteur de sync et peut être augmentée par les recherches.
- Les montants ne peuvent pas devenir négatifs.
- Si une ressource est limitée par stockage, elle est clampée à sa capacité.

---

## 4) Construction des bâtiments

### Démarrage

Un bâtiment se construit via le flux de construction.

Règles :
- le niveau demandé doit respecter la séquence (`niveau courant + 1`)
- un bâtiment déjà en construction ne peut pas être redémarré sans terminer / annuler
- la durée dépend de la définition du bâtiment et du niveau cible
- le serveur calcule `startedAt`, `endsAt` et `remainingSec`

### Fin de construction

Quand le temps est écoulé :
- le bâtiment passe au niveau suivant
- l’état `isConstructing` devient `false`
- les effets du bâtiment sont pris en compte au prochain recalcul de production

### Règle de base

**Un bâtiment ne produit rien tant qu’il n’est pas terminé.**

---

## 5) Bâtiments et effets

Les bâtiments sont définis dans le catalogue de contenu.

### Types d’effets supportés

Le moteur supporte les effets suivants dans les `EffectsJSON` :
- `resourceProduction`
- `statBonus`
- `buildingBonus` via recherches
- `resourceBonus` sur certaines valeurs de stockage

### Règles de calcul

- `resourceProduction` augmente une production de ressource par niveau ou par valeur fixe.
- `statBonus` agit sur des stats de ville ou de profil selon le champ ciblé.
- `buildingBonus` vient des recherches terminées et applique un bonus à un bâtiment précis.

### Bâtiments starter actuellement reliés

- `building_modular_habitat`
  - augmente la capacité de population
  - améliore aussi légèrement la morale
- `building_solar_plant`
  - produit de l’énergie
- `building_vertical_farm`
  - produit de la nourriture
- `building_composite_mine`
  - produit du métal
- `building_data_bank`
  - produit des données
- `building_nexus_market`
  - produit des crédits
- `building_nexus_core`
  - bonus global / cap de ville selon le seed

### Principe important

Les bâtiments ne sont **jamais** évalués côté client comme source de vérité.  
Le backend recalcule toujours la production à partir des bâtiments terminés.

---

## 6) Recherches

### Enregistrement

Une recherche terminée est stockée dans `PlayerResearch`.

### Effets des recherches

Les recherches peuvent :
- augmenter la capacité de stockage
- améliorer la production de certains bâtiments
- débloquer des bonus de ville
- ajouter des bonus globaux selon le type d’effet

### Exemple de règle appliquée

- `research_modular_housing` : bonus sur `building_modular_habitat`
- `research_solar_stabilization` : bonus sur `building_solar_plant`
- `research_efficient_storage` : augmente la capacité de stockage

### Règle importante

Une recherche complétée est considérée comme acquise pour tout calcul futur.  
Le moteur peut relire les recherches à chaque sync pour recalculer les bonus.

---

## 7) Production économique

### Recalcul serveur

À chaque sync de production, le backend :
1. lit les bâtiments terminés
2. lit les recherches terminées
3. calcule les effets
4. met à jour les `PlayerResource`
5. met à jour les statistiques de ville
6. met à jour les stats de profil utiles

### Balances

Pour chaque ressource :
- `productionPerTick`
- `consumptionPerTick`
- `balancePerTick = productionPerTick - consumptionPerTick`

### Accrual serveur

Si la sync est déclenchée avec accrual :
- le backend applique la production cumulée depuis la dernière sync
- le montant est clampé à 0 minimum
- le montant est clampé à la capacité maximum si nécessaire

### Dernière sync

`PlayerCityStats.lastProductionSyncAt` sert de référence pour l’accrual temporel.

---

## 8) Population

La population courante est stockee dans `ProfileGamer.population`.

La limite est stockee dans `ProfileGamer.populationCapacity` et recalculee par `SyncBuildingProduction`.

Regles appliquees:

- `building_modular_habitat` ajoute `500 + 250 * (niveau - 1)` via la configuration active `GameBalanceConfig`.
- Les recherches de type `buildingBonus/statBonus` peuvent modifier cette capacite.
- Si la capacite calculee est inferieure a la population actuelle, le backend garde au minimum la population actuelle pour eviter un clamp destructeur brutal.
- La nourriture consommee est `population * 0.08 par heure`, stockee cote ressource en `consumptionPerTick` apres division par `3600`.
- Flutter doit afficher la limite avec `profile.populationCapacity`, pas la recalculer localement.

Lecture par niveau:

| Niveau habitat | Capacite approx. d'un habitat | Lecture gameplay |
| --- | ---: | --- |
| 1 | 500 | premiers colons stables |
| 10 | 2 750 | quartier residentiel solide |
| 20 | 5 250 | bloc urbain majeur |
| 30 | 7 750 | habitat Nexus complet |

La capacite totale est la somme des habitats termines, apres bonus de recherche.

---

## 9) Morale, sécurité et énergie

### Morale

- monte si la ville est stable
- baisse si l’énergie est négative, si la sécurité est faible ou si la ville est surchargée

### Sécurité

- monte si la morale et l’énergie sont bonnes
- baisse si la ville est trop dense ou instable

### Énergie

- `ProfileGamer.energyProduction` et `energyConsumption` sont des valeurs par heure, lisibles par l'UI.
- `PlayerResource.productionPerTick`, `consumptionPerTick` et `balancePerTick` restent des valeurs par seconde pour l'animation et l'accrual serveur.
- `building_solar_plant` produit `80/h` au niveau 1, puis suit `productionGrowth`.
- La consommation de base est `5/h + population/20`.
- Chaque batiment termine ajoute une surcharge `floor(level^1.25)`; au-dessus du niveau 20, cette surcharge est multipliee par `1.2`.
- Les batiments industrie/data ajoutent une pression supplementaire, reduite par le total des niveaux solaires.
- Si les habitats depassent trop les fermes (`L_farm * 2 < L_hab`), une surcharge d'energie est ajoutee.

Le ratio serveur est:

```text
E_ratio = energyProduction / max(1, energyConsumption)
```

Effets appliques aux productions non-energie:

| Ratio energie | Multiplicateur ressources |
| --- | ---: |
| `< 0.75` | `0.75` |
| `0.75 - 0.99` | `0.95` |
| `1.00 - 1.19` | `1.00` |
| `1.20 - 1.49` | `1.10` |
| `>= 1.50` | `1.20` |

Flutter doit donc lire:

- limite population: `profile.populationCapacity`
- energie produite/consommee: `profile.energyProduction`, `profile.energyConsumption`, `profile.energyBalance`
- animation des stocks: `resources[].balancePerTick`

---

## 10) Tick monde

Le tick monde est le cycle de simulation principal.

### Il sert à :
- recalculer la production
- faire évoluer la population
- mettre à jour morale / sécurité / énergie
- générer ou proposer des événements IA
- enregistrer l’état du monde

### Règle actuelle

Le tick monde peut être déclenché :
- manuellement par l’admin
- via l’endpoint de trigger
- par un scheduler futur si on le branche plus tard

### Sans cron

Pour l’instant, il n’y a **pas de cron obligatoire** :
- `Tick Monde`
- `Sync Production`
- `Générer Événement IA`

peuvent être lancés manuellement via le panel admin.

---

## 11) IA serveur

### Principe

L’IA serveur est utilisée pour :
- générer des événements
- résumer les ticks
- écrire le living lore
- préparer des cas tribunal
- produire des sorties admin structurées

### Règle de fonctionnement

1. le backend tente d’abord un provider réel si une clé est configurée
2. sinon il utilise un fallback local déterministe
3. toutes les sorties sont loggées
4. les réponses restent sous contrôle du serveur

### Providers supportés

- `mistral`
- `openai`

### Règle de sécurité

Le provider ne décide jamais seul d’un effet final non validé.  
Le serveur garde la main sur l’application réelle des règles.

---

## 12) Événements IA

### Types

Le système peut proposer :
- événement monde
- lore vivant
- cas tribunal
- suggestion de quête
- résumé de tick

### Limites

- impact limité
- récompenses plafonnées
- pas plus de quelques événements majeurs par jour
- validation admin si nécessaire

---

## 13) Construction et recherche côté Flutter

### Vue client

Flutter doit afficher :
- la file de construction
- le temps restant
- la production par tick
- la capacité de stockage
- les prérequis des bâtiments et recherches

### Règle de re-sync

Le client doit recharger les données serveur :
- au lancement
- au retour en foreground
- au reconnect réseau
- après une action importante
- après une erreur de validation

### Stratégie recommandée

- polling construction : court
- polling économie : moyen
- polling monde : plus long
- re-sync complet : sur reprise ou conflit

---

## 14) États de gameplay

### Ville

- active
- en construction
- en surcapacité
- en déficit énergétique
- en stabilité haute / basse

### Construction

- en attente
- en cours
- terminée
- annulée
- accélérée

### Recherche

- non débloquée
- débloquée
- en cours
- terminée

### Monde

- stable
- sous tension
- événement actif
- tick en cours

---

## 15) Règles admin / contrôle manuel

### Sans scheduler automatique

Les actions importantes peuvent être déclenchées manuellement depuis le cockpit admin :
- tick monde
- sync production
- génération d’événement IA
- réparation d’assignations

### But

Permettre d’exploiter et tester le système proprement même sans cron en production.

---

## 16) Résumé ultra-court

- Les bâtiments terminés produisent des ressources.
- Les recherches terminées ajoutent des bonus.
- La population évolue au tick monde selon logement, morale, sécurité, nourriture et énergie.
- Le serveur est la seule source de vérité.
- Les actions IA peuvent être déclenchées manuellement en attendant un scheduler.

---

## 17) Statut de ce document

Ce document décrit l’état **actuellement implémenté** du module `nexus_game`.  
Il doit être mis à jour à chaque changement majeur de gameplay, d’économie ou d’IA.
