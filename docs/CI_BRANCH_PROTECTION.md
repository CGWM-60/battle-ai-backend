# Protection de branche GitHub

Configurer manuellement dans **Settings → Branches → Branch protection rules** pour `main` (et `develop` si utilisé).

## Règles requises

| Règle | Valeur |
|-------|--------|
| Require a pull request before merging | Oui |
| Require status checks to pass | Oui |
| Required checks | `Go backend checks` (workflow `backend-ci.yml`) |
| Require branches to be up to date | Oui |
| Do not allow bypassing | Recommandé |
| Restrict force pushes | Oui |
| Restrict deletions | Oui |

## Vérification

1. Ouvrir une PR vers `main`.
2. Confirmer que **Backend CI** s’exécute sur `push` et `pull_request`.
3. Le merge doit être bloqué tant que la CI est rouge.