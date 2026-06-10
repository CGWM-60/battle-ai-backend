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

### Source de la population

La population est gérée dans `ProfileGamer`.

### Croissance

La population évolue lors du tick monde selon :
- la capacité de logement
- la nourriture
- la morale
- la sécurité
- l’énergie
- l’état général du monde

### Forme générale

La croissance dépend d’un score de base multiplié par des facteurs de contexte.

### Règles concrètes

- Si la capacité de population est à 0, la population ne peut pas croître correctement.
- Si la capacité est pleine, la croissance est plafonnée.
- Si morale, sécurité ou énergie sont faibles, la croissance baisse ou devient négative.
- Si l’énergie est trop basse, la population peut décliner.

### Source de la capacité

La capacité de population vient principalement du `building_modular_habitat` et de ses bonus de recherche.

### Paliers population (mode MMO difficile)

Le système population doit utiliser des paliers de pression urbaine basés sur :

- `L_hab` = niveau total des `building_modular_habitat`
- `L_farm` = niveau total des `building_vertical_farm`
- `L_sec` = niveau total des bâtiments défensifs/sécurité
- `R_pop` = ratio de charge population = `population / populationCapacity`

#### Palier 1 — Colonie fragile (L_hab 1-5)

- croissance de base faible
- vulnérable au moindre déficit énergétique
- si `R_pop > 0.85` : malus direct de croissance

Règle recommandée :
- bonus croissance : `+0%`
- malus surpopulation : `-35%`

#### Palier 2 — Ville naissante (L_hab 6-12)

- croissance stabilisée si nourriture et énergie sont positives
- pénalité réduite en cas de tension légère

Règle recommandée :
- bonus croissance : `+12%`
- malus surpopulation (`R_pop > 0.9`) : `-25%`

#### Palier 3 — Métropole sous tension (L_hab 13-20)

- forte capacité, mais exige sécurité + alimentation solides
- si sécurité basse, risque de stagnation massive

Règle recommandée :
- bonus croissance : `+25%`
- si sécurité < 45 : `-30%` croissance

#### Palier 4 — Mégapole Nexus (L_hab 21+)

- très gros potentiel démographique
- système plus exigeant : la moindre crise énergétique ou alimentaire coûte cher

Règle recommandée :
- bonus croissance : `+40%`
- si énergie négative pendant 2 ticks consécutifs : `-2%` population immédiate
- si nourriture négative pendant 2 ticks consécutifs : `-3%` population immédiate

### Synergies population-bâtiments

- `building_vertical_farm` : réduit les malus de croissance liés à la nourriture
- `building_solar_plant` : réduit les malus de panique urbaine
- bâtiments sécurité/défense : absorbent les pénalités sur morale/sécurité aux paliers élevés

Règle hardcore recommandée :

- si `L_farm < (L_hab / 2)` alors `-15%` croissance population
- si `L_sec < (L_hab / 3)` alors `-10%` croissance + `-5` sécurité par tick

---

## 9) Morale, sécurité et énergie

### Morale

- monte si la ville est stable
- baisse si l’énergie est négative, si la sécurité est faible ou si la ville est surchargée

### Sécurité

- monte si la morale et l’énergie sont bonnes
- baisse si la ville est trop dense ou instable

### Énergie

- produite par les bâtiments dédiés
- consommée par l’activité de la ville
- si le bilan est négatif, la réserve peut être drainée
- si la réserve ne suffit plus, la balance d’énergie devient problématique pour le reste de la ville

### Paliers énergie (mode MMO difficile)

Le système énergie doit être piloté par :

- `L_solar` = niveau total des `building_solar_plant`
- `L_industry` = niveau total des bâtiments de production lourde
- `L_data` = niveau total des bâtiments IA / data
- `E_ratio` = `energyProduction / max(1, energyConsumption)`

#### Palier E0 — Survie énergétique (`E_ratio < 0.75`)

- état critique
- consommation prioritaire des services vitaux
- fortes pénalités globales

Pénalités recommandées :
- `-30%` croissance population
- `-20` morale
- `-15` sécurité
- `-25%` production non-énergie

#### Palier E1 — Tension (`0.75 <= E_ratio < 1.00`)

- déficit gérable à court terme
- la réserve compense mais s’érode vite

Pénalités recommandées :
- `-12%` croissance population
- `-8` morale
- `-5%` production globale

#### Palier E2 — Équilibre instable (`1.00 <= E_ratio < 1.20`)

- état nominal de progression lente
- aucun bonus majeur

Effets recommandés :
- croissance normale
- pas de bonus/malus énergétique majeur

#### Palier E3 — Excédent contrôlé (`1.20 <= E_ratio < 1.50`)

- ville optimisée
- bonus modérés sur économie et stabilité

Bonus recommandés :
- `+10%` production ressources
- `+5` morale
- `+5%` vitesse de recherche

#### Palier E4 — Suprématie énergétique (`E_ratio >= 1.50`)

- apex techno
- très puissant mais coûteux à maintenir

Bonus/Mécaniques recommandés :
- `+20%` production ressources
- `+12%` vitesse de recherche
- surcharge : si ce palier dure > 6 ticks sans upgrade réseau, risque d’événement panne (`5%`/tick)

### Coût exponentiel de montée énergétique

Pour garder un MMO difficile, le coût d’entretien doit croître avec les niveaux :

- consommation de base par ville : croissante avec population
- surcharge par bâtiment haut niveau :
  - bâtiment lvl 1-10 : surcharge faible
  - lvl 11-20 : surcharge moyenne
  - lvl 21+ : surcharge forte

Règle recommandée :

- coût énergie additionnel par bâtiment = `floor(level^1.25)`
- au-delà du lvl 20, multiplicateur crise = `x1.2`

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
