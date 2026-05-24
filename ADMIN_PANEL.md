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
- voir les stats principales
- creer des quetes Battle manuellement
- creer des quetes RolePlay manuellement
- generer des quetes Battle avec un provider IA
- generer des quetes RolePlay avec un provider IA
- voir les dernieres battles
- voir les dernieres quetes
- voir les live sessions
- terminer une live session

Les cles API utilisees pour la generation IA sont envoyees uniquement via le formulaire et ne sont pas stockees en base.

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
