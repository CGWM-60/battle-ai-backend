# Guide Backend

Ce document te donne un cadre fiable pour construire le backend de `go-battle-ia` du debut a la fin sans repartir en arriere.

Le principe est simple :

1. tu figes d'abord les contrats de donnees
2. tu construis la persistence
3. tu ajoutes la logique applicative par blocs isoles
4. tu exposes les routes HTTP uniquement quand le bloc metier est stable
5. tu ajoutes le live en dernier

Ne saute pas d'etape.

## 1. Objectif du projet

Le backend doit couvrir ces modes :

- `Battle IA` solo avec sauvegarde et reprise
- `Battle Arena` publique qu'un autre joueur peut rejoindre
- `RolePlay IA` avec sauvegarde et reprise
- `Quetes systeme` pour battle et roleplay
- `Mode coop`
- `Live` pour battle et roleplay

## 2. Regle d'or

Pour chaque feature, respecte toujours cet ordre :

1. `model`
2. `migration`
3. `repository`
4. `service`
5. `handler HTTP`
6. `tests`

Si tu fais l'inverse, tu vas casser des choses ou dupliquer de la logique.

## 3. Etat actuel du repo

Fichiers deja presents :

- [main.go](/Users/canguilieme/Documents/Developments/Dev peso/Golang dev/go-battle-ia/main.go:1)
- [internal/db/db.go](/Users/canguilieme/Documents/Developments/Dev peso/Golang dev/go-battle-ia/internal/db/db.go:1)
- [internal/models/models.go](/Users/canguilieme/Documents/Developments/Dev peso/Golang dev/go-battle-ia/internal/models/models.go:1)
- [internal/router/router.go](/Users/canguilieme/Documents/Developments/Dev peso/Golang dev/go-battle-ia/internal/router/router.go:1)
- [internal/scenarios/battle.go](/Users/canguilieme/Documents/Developments/Dev peso/Golang dev/go-battle-ia/internal/scenarios/battle.go:1)
- [internal/scenarios/rolePlayer.go](/Users/canguilieme/Documents/Developments/Dev peso/Golang dev/go-battle-ia/internal/scenarios/rolePlayer.go:1)
- [internal/provider/providerIa.go](/Users/canguilieme/Documents/Developments/Dev peso/Golang dev/go-battle-ia/internal/provider/providerIa.go:1)

## 4. Structure cible recommandee

Avant d'ecrire la vraie logique, cree cette structure :

```text
internal/
  app/
    battle/
    arena/
    roleplay/
    coop/
    live/
    quest/
    auth/
  repository/
  service/
  transport/
    http/
  dto/
  validator/
  middleware/
```

Regle :

- `models` = schema GORM uniquement
- `repository` = acces base
- `service` = logique metier
- `transport/http` = handlers Gin
- `scenarios` = moteur IA pur

Le moteur IA ne doit pas connaitre Gin ni GORM directement.

## 5. Ordre fiable de construction

Voici l'ordre que je te conseille. Ne passe a l'etape suivante que si la precedente est stable.

### Etape 0. Figer les conventions

Objectif :

- fixer les noms de status
- fixer les noms de modes
- fixer les regles de relation entre tables

A faire :

- definir une liste stable de `status`
- definir une liste stable de `mode`
- definir les permissions de base par type de session

Convention conseillee :

- battle status : `draft`, `live`, `paused`, `finished`, `abandoned`
- arena status : `waiting`, `running`, `paused`, `finished`, `closed`
- roleplay status : `draft`, `live`, `paused`, `finished`, `failed`, `abandoned`
- live status : `waiting`, `streaming`, `paused`, `ended`
- mode : `battle_ia`, `roleplay_ia`, `coop`

Definition of done :

- toutes les chaines sont listees dans un seul fichier de constantes
- aucun status libre en dur dans les handlers

### Etape 1. Stabiliser la base

Objectif :

- rendre les models exploitables sans revenir dessus tous les 2 jours

A faire :

- relire tous les models GORM
- verifier les foreign keys
- verifier les champs JSON
- verifier les index
- verifier les champs dates utiles a la reprise et au live

Tables deja prevues :

- `users`
- `quest_ia_battles`
- `battle_saves`
- `battle_save_turns`
- `battle_arenas`
- `battle_arena_members`
- `role_play_quest_templates`
- `role_play_quest_runs`
- `role_play_sessions`
- `role_play_session_turns`
- `coop_parties`
- `coop_party_members`
- `live_sessions`
- `live_events`

Definition of done :

- `go build ./...`
- `AutoMigrate` passe
- tu peux creer la base a vide sans erreur

### Etape 2. Introduire les repositories

Objectif :

- ne plus appeler GORM partout dans le code

A creer :

- `user_repository.go`
- `battle_repository.go`
- `arena_repository.go`
- `roleplay_repository.go`
- `coop_repository.go`
- `live_repository.go`
- `quest_repository.go`

Chaque repository doit faire seulement :

- `Create`
- `GetByID`
- `GetByCode`
- `List`
- `Update`
- `Delete soft`
- `AppendTurn`
- `AppendEvent`

Ne mets pas de regles metier dedans.

Definition of done :

- les handlers n'utilisent plus GORM directement
- les services peuvent etre testes avec mocks ou interfaces

### Etape 3. Construire le bloc Battle IA

Objectif :

- lancer une battle
- enregistrer sa progression
- la reprendre

Flux minimal :

1. creer un `BattleSave`
2. stocker `IASnapshot`
3. stocker le `Context`
4. a chaque message IA, creer un `BattleSaveTurn`
5. mettre a jour `CurrentRound` et `LastActivityAt`
6. a la fin, passer la session en `finished`

Endpoints a faire ensuite :

- `POST /api/v1/battles`
- `GET /api/v1/battles/:id`
- `GET /api/v1/battles/:id/turns`
- `POST /api/v1/battles/:id/resume`
- `POST /api/v1/battles/:id/cancel`

Attention :

- ne reconstruis jamais une battle depuis le front
- la source de verite doit rester `BattleSave` + `BattleSaveTurn`

Definition of done :

- une battle peut etre reprise sans perdre le contexte
- les tours deja produits restent coherents

### Etape 4. Ajouter les quetes Battle IA

Objectif :

- separer le catalogue systeme des executions joueur

Regle :

- `QuestIaBattle` = template
- `BattleSave` = execution concrete

A faire :

- CRUD admin des quetes
- publication / archivage
- selection aleatoire ou par theme
- rattacher la quete a `BattleSave.QuestID`

Definition of done :

- une battle peut venir d'une quete systeme
- l'historique de la session ne depend pas du template apres creation

### Etape 5. Construire l'Arena

Objectif :

- permettre a un autre joueur de rejoindre une battle publique en cours

Flux minimal :

1. le host cree une `BattleSave`
2. le host ouvre une `BattleArena`
3. les joueurs rejoignent via `BattleArenaMember`
4. la battle continue de tourner sur la session parente
5. l'arene ne fait qu'encadrer acces, presence et role

Endpoints conseilles :

- `POST /api/v1/arenas`
- `GET /api/v1/arenas/:code`
- `POST /api/v1/arenas/:code/join`
- `POST /api/v1/arenas/:code/leave`
- `GET /api/v1/arenas/:code/members`

Important :

- l'arene ne doit pas dupliquer les messages de battle
- l'arene pointe vers `BattleSaveID`

Definition of done :

- un joueur rejoint une arene en cours
- les membres voient la meme session parente

### Etape 6. Construire RolePlay IA

Objectif :

- lancer une session roleplay persistante avec historique et reprise

Flux minimal :

1. creer `RolePlaySession`
2. stocker le prompt de scenario
3. a chaque action ou reponse, creer `RolePlaySessionTurn`
4. mettre a jour le `Snapshot`
5. terminer ou mettre en pause

Endpoints conseilles :

- `POST /api/v1/roleplay/sessions`
- `GET /api/v1/roleplay/sessions/:id`
- `GET /api/v1/roleplay/sessions/:id/turns`
- `POST /api/v1/roleplay/sessions/:id/resume`

Definition of done :

- la session se recharge depuis la base sans recalcul fragile

### Etape 7. Ajouter les quetes RolePlay

Objectif :

- gerer un catalogue systeme et la progression joueur

Regle :

- `RolePlayQuestTemplate` = modele systeme
- `RolePlayQuestRun` = progression du joueur
- `RolePlaySession` = execution narrative

Flux minimal :

1. choisir un template
2. creer un `RolePlayQuestRun`
3. creer ou lier une `RolePlaySession`
4. mettre a jour `CurrentStep`, `State`, `Journal`
5. terminer la quete

Definition of done :

- une quete roleplay peut etre reprise
- le journal et l'etat persistent proprement

### Etape 8. Construire le mode coop

Objectif :

- permettre a plusieurs joueurs de partager une meme partie

Regle de modelisation :

- `CoopParty` = salon
- `CoopPartyMember` = membres
- la partie coop pointe soit vers `BattleSave`, soit vers `RolePlaySession`

Ne fais pas deux systemes coop differents au debut.

Flux minimal :

1. creer le salon
2. associer la source de jeu
3. faire rejoindre les membres
4. marquer les membres `ready`
5. synchroniser le state partage

Definition of done :

- un salon coop sait qui participe
- le state partage est unique

### Etape 9. Construire le live

Objectif :

- diffuser ce qui se passe pendant battle et roleplay

Regle :

- `LiveSession` = canal
- `LiveEvent` = flux d'evenements

Le live doit etre derive des evenements persistants, pas l'inverse.

Ordre conseille :

1. persister `LiveSession`
2. persister `LiveEvent`
3. exposer un endpoint de lecture historique
4. exposer ensuite SSE ou WebSocket

Endpoints conseilles :

- `GET /api/v1/live/:channel/history`
- `GET /api/v1/live/:channel/stream`

Evenements minimums :

- `join`
- `leave`
- `message`
- `chunk`
- `status`
- `score`
- `error`

Definition of done :

- si le stream coupe, tu peux rejouer l'historique depuis `LiveEvent`

### Etape 10. Auth et permissions

Objectif :

- proteger les parties et l'acces multi-joueurs

A faire seulement quand battle, roleplay et arena sont stables.

Minimum :

- auth utilisateur
- middleware d'identite
- controle d'acces sur owner / host / member / spectator

Regles minimales :

- seul le owner peut reprendre ou annuler sa session
- seul le host gere son arena ou son salon coop
- un spectator ne modifie rien

### Etape 11. Nettoyage architecture

Objectif :

- sortir les gros blocs melanges avant qu'ils deviennent ingables

Priorites :

- enlever la logique HTTP de `router.go`
- sortir la creation des providers dans un service dedie
- sortir le code de scenario des handlers
- injecter `db`, repositories et services au lieu d'appels globaux

Definition of done :

- `main.go` ne fait que wiring
- `router.go` ne fait que brancher les routes

## 6. Ordre des fichiers a coder

Ordre concret recommande :

1. `internal/app/constants/status.go`
2. `internal/repository/*.go`
3. `internal/service/battle_service.go`
4. `internal/service/quest_service.go`
5. `internal/service/arena_service.go`
6. `internal/service/roleplay_service.go`
7. `internal/service/coop_service.go`
8. `internal/service/live_service.go`
9. `internal/transport/http/battle_handler.go`
10. `internal/transport/http/arena_handler.go`
11. `internal/transport/http/roleplay_handler.go`
12. `internal/transport/http/coop_handler.go`
13. `internal/transport/http/live_handler.go`

## 7. Contrats minimums par domaine

### Battle Service

Doit savoir :

- creer une session
- lancer une execution
- append un tour
- sauvegarder un context
- reprendre une session
- terminer une session

### Arena Service

Doit savoir :

- ouvrir une arene
- rejoindre
- quitter
- lister les membres
- fermer

### RolePlay Service

Doit savoir :

- creer une session
- append un tour
- mettre a jour le snapshot
- reprendre
- terminer

### Coop Service

Doit savoir :

- creer un salon
- inviter / rejoindre
- changer l'etat d'un membre
- lire le state partage

### Live Service

Doit savoir :

- ouvrir un canal
- append un event
- lire l'historique
- fermer le canal

## 8. Ce que tu ne dois pas faire trop tot

Evite ca au debut :

- WebSocket avant d'avoir `LiveEvent`
- permissions complexes avant d'avoir les services
- microservices
- CQRS
- event bus externe
- worker queue
- admin front

Tu peux les ajouter plus tard si le socle tient deja.

## 9. Strategie de tests fiable

Pour chaque etape, fais ces tests dans cet ordre :

1. test repository
2. test service
3. test handler
4. test flux complet

Je te conseille au minimum :

- creation de session battle
- reprise d'une battle
- join arena
- creation session roleplay
- reprise roleplay
- creation salon coop
- replay d'un historique live

## 10. Checklist avant de passer a l'etape suivante

Pose-toi ces questions :

- est-ce que l'etat source est en base ?
- est-ce que je peux reprendre sans recalcul fragile ?
- est-ce que je peux reconstituer un historique ?
- est-ce que HTTP ne fait que transporter ?
- est-ce que la logique metier vit dans un service ?
- est-ce que le moteur IA est decouple de GORM et Gin ?

Si une reponse est `non`, ne continue pas.

## 11. Premier plan de travail concret

Si tu veux avancer sans te perdre, fais exactement ca :

1. extraire une instance `*gorm.DB` partagee proprement
2. creer les repositories
3. faire `BattleService`
4. brancher creation + reprise de battle
5. brancher les quetes battle
6. faire `ArenaService`
7. faire `RolePlayService`
8. brancher les quetes roleplay
9. faire `CoopService`
10. faire `LiveService`
11. nettoyer `router.go`
12. ajouter auth et middleware

## 12. Regle finale

Quand tu ajoutes une nouvelle feature, verifie toujours dans cet ordre :

1. ou est l'etat source ?
2. quelle table le porte ?
3. qui a le droit de le modifier ?
4. comment je le rejoue ?
5. comment je le reprends apres crash ?

Si tu construis toujours avec ces 5 questions, ton backend restera fiable.
