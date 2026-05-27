# TODO Monde IA Multijoueur

## Fait

- [x] Completer le CRUD admin generique: creation, edition JSON, suppression/action et rafraichissement depuis les pages `/admin/game`.
- [x] Ajouter l'upload reel des assets de batiments: multipart, stockage local, hash SHA-256, taille, version, URL publique et page admin dediee.
- [x] Appliquer reellement les recompenses d'evenements dans `PlayerSave` avec journal anti double-claim complet.
- [x] Durcir l'anti-triche de sync: deltas par temps, sauvegardes trop anciennes, historique actions importantes.
- [x] Completer les contraintes evenements: maximum 4 par jour, anti-chevauchement robuste, validation requirements.
- [x] Completer la resolution des conflits: consequences, limites par joueur/continent, protection debutant avancee.
- [x] Completer les regles guildes: permissions owner/officer, transfert owner, promotion/retrogradation stricte, contributions.
- [x] Ajouter WebSocket/SSE pour le chat si le client en a besoin; conserver le polling pagine comme fallback.

## A faire
- [ ] Enrichir la simulation NEXUS: cycles 15 min/1h/daily separes, profils continentaux plus differencies, replay/dry-run plus complet.
- [ ] Ajouter les graphiques admin demandes: activite chat, croissance joueurs, conflits par intensite, meteo par severite, ressources/recompenses.
- [ ] Ajouter la matrice de tests prioritaire complete: assignation DB, guildes, chat scopes, events, claims, conflits, manifest.
- [ ] Ajouter des alias stricts `/api/admin/...` si un client externe ne peut pas utiliser `/admin/api/game/...` ou `/api/v1/admin/...`.
