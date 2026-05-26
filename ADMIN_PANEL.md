# Interface Admin

L'interface de gestion est disponible sur :

```text
http://localhost:8080/admin
```

## Securite

Routes admin :

```text
GET  /admin/login
POST /admin/login
GET  /admin
GET  /admin/accounts
GET  /admin/system
GET  /admin/usage
GET  /admin/nexus-coin
GET  /admin/quests
GET  /admin/live
GET  /admin/api/dashboard
GET  /admin/api/accounts
GET  /admin/api/system
GET  /admin/api/usage
GET  /admin/api/nexus-coin
POST /admin/api/nexus-coin/plans
PUT  /admin/api/nexus-coin/plans/:id
PATCH /admin/api/nexus-coin/plans/:id
DELETE /admin/api/nexus-coin/plans/:id
GET  /admin/api/quests
GET  /admin/api/live
GET  /api/v1/nexus-coin/plans
POST /admin/logout
POST /admin/quests/battle
POST /admin/quests/rp
POST /admin/generate/battle
POST /admin/generate/rp
POST /admin/live/:id/end
```

L'admin utilise un login dedie et un cookie `HttpOnly` signe.

En dev Docker, les valeurs par defaut sont :

```text
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin
ADMIN_SESSION_SECRET=dev-admin-session-secret
ADMIN_API_SECRET=dev-admin-api-secret
```

En production, `compose.prod.yml` exige :

```text
ADMIN_USERNAME=admin
ADMIN_PASSWORD_BCRYPT=<hash bcrypt>
ADMIN_SESSION_SECRET=<secret long aleatoire>
ADMIN_API_SECRET=<secret long aleatoire pour les routes admin API>
```

Pour creer un hash bcrypt rapidement :

```bash
htpasswd -bnBC 12 "" "ton-password" | tr -d ':\n'
```

## Fonctions Disponibles

Le dashboard admin permet de :

- verifier la sante DB
- verifier la configuration de charge backend
- voir les stats principales sur des pages Next.js separees
- consulter les comptes actuels avec progression et activite
- suivre les connexions DB, les requetes HTTP, la charge runtime Go et les stats reseau applicatives
- gerer les packs Nexus Coin pour les credits IA avec estimation token + marge
- creer des quetes Battle manuellement
- creer des quetes RolePlay manuellement
- generer des quetes Battle avec un provider IA
- generer des quetes RolePlay avec un provider IA
- voir les dernieres battles
- voir les dernieres quetes
- voir les live sessions
- terminer une live session

Les cles API utilisees pour la generation IA sont envoyees uniquement via le formulaire et ne sont pas stockees en base.

## Frontend Next.js

L'interface `/admin` est une application Next.js statique dans `admin/`, construite avec Next.js `16.2.6`, React `19.2.6` et `basePath=/admin`.

En local hors Docker :

```bash
cd admin
npm install
npm run build
cd ..
go run .
```

Le backend sert ensuite `admin/out` directement. Si `admin/out` n'existe pas, l'ancien template Go reste disponible comme fallback minimal.

## Nexus Coin

La page `/admin/nexus-coin` affiche :

- 4 estimations automatiques calculees depuis `AIUsageRecord`
- une marge par defaut de `50%` configurable via `NEXUS_COIN_DEFAULT_MARGIN_PERCENT`
- 4 plans persistants en base (`nexus_coin_plans`) crees au premier chargement
- un CRUD admin pour modifier le texte, la marge, les credits, le statut et les tokens

Les plans actifs sont exposables au client Flutter via :

```text
GET /api/v1/nexus-coin/plans
```

Si aucun cout IA n'est encore disponible en base, l'estimation utilise `NEXUS_COIN_FALLBACK_USD_PER_1M_TOKENS` avec `2.5` par defaut.

## Providers IA

Providers acceptes :

```text
mistral
openai
openrouter
xia
```

Le modele est libre selon le provider choisi, par exemple :

```text
mistral-large-latest
gpt-4o-mini
openai/gpt-4o-mini
grok-3-mini
```

## Limites Actuelles

L'interface admin pilote les fonctions deja implementees dans le backend. Elle ne remplace pas encore :

- un vrai systeme de roles admin multi-utilisateurs
- un audit log des actions admin
- un editeur detaille pour modifier/supprimer les quetes existantes
- un terminal ou des commandes systeme distantes

Ces points pourront etre ajoutes ensuite si tu veux une console d'exploitation plus avancee.
