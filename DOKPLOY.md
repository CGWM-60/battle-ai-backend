# Déploiement Dokploy

Ce projet est prêt pour Dokploy avec `compose.prod.yml`.

## 1) Créer le service Dokploy
- Type: **Docker Compose**
- Fichier compose: `compose.prod.yml`

## 2) Variables d'environnement
- Copier les variables de `dokploy.env.example`
- Coller dans **Dokploy > Environment Variables**
- Générer des valeurs fortes pour:
  - `JWT_SECRET`
  - `ADMIN_PASSWORD_BCRYPT`
  - `ADMIN_SESSION_SECRET`
  - `ADMIN_API_SECRET`
  - `DB_PASSWORD`
  - `MARIADB_ROOT_PASSWORD`

### Admin login: piège courant
- Si vous utilisez `ADMIN_PASSWORD` en clair, évitez les caractères comme `#` dans la valeur brute.
- Selon la manière dont Dokploy ou un parser dotenv injecte les variables, `#` peut être interprété comme un commentaire et tronquer la valeur.
- Pour un test simple, utilisez temporairement une valeur comme `Admin2026Secure`.
- En production, préférez `ADMIN_PASSWORD_BCRYPT` et laissez `ADMIN_PASSWORD` vide.

## 3) Réseau / domaine
- Exposer le service `app` sur le port `8080`
- Configurer le domaine Dokploy vers ce service

## 4) Persistance
- Le volume `mariadb-data` est défini dans `compose.prod.yml` pour conserver les données MariaDB.

## Notes
- `compose.prod.yml` inclut `app + mariadb` et un `healthcheck` DB.
- Si vous utilisez une DB externe, remplacez simplement:
  - `DB_HOST`
  - `DB_PORT`
  - `DB_NAME`
  - `DB_USER`
  - `DB_PASSWORD`
  et vous pouvez retirer le service `mariadb` du compose.
