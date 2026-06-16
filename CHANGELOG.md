# Changelog

Toutes les modifications notables sont documentées ici.

Le format s'inspire de [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/)
et le projet suit le [versionnage sémantique](https://semver.org/lang/fr/).

> Les notes détaillées de chaque version publiée sont aussi disponibles dans les
> [GitHub Releases](https://github.com/statoon54/mailhive/releases) (générées par
> GoReleaser à partir des commits).

## [Non publié]

## [0.1.1] - 2026-06-16

### Corrigé

- **Seed** : la config SMTP Mailpit par défaut n'est seedée qu'en mode dev
  (`SMTP_MODE=mailpit`). En mode `real` (prod), seul le tenant admin est créé —
  fini la config `is_default` pointant vers un hôte `mailpit` inexistant.

### Documentation

- README : ajout d'un lien vers le schéma OpenAPI (`api/openapi.yaml`) et retrait
  de la note sur le slug du dépôt.
- Correction des exemples de tag d'image GHCR (semver **sans préfixe `v`** :
  `0.1.0`) dans le README, `docker-compose.prod.yml` et le Makefile.

## [0.1.0] - 2026-06-16

### Ajouté

- API multi-tenant d'envoi et de gestion d'e-mails : file d'attente asynchrone
  (Asynq/Redis, 3 priorités), templates embarqués, configurations SMTP (mots de
  passe chiffrés AES-GCM), branding, journal d'audit partitionné par mois,
  internationalisation FR/EN.
- Pièces jointes **dédupliquées et adressées par contenu** (backend PostgreSQL
  ou S3 compatible SeaweedFS/MinIO/S3/R2), avec téléchargement depuis l'UI admin
  et l'API (`GET /mails/{id}/attachments/{attachmentId}`).
- Rate limiting distribué par tenant (token bucket Redis, script Lua atomique)
  et circuit breaker par configuration SMTP.
- Sondage adaptatif temps réel du front (liste des mails, détail, tableau de
  bord), suspendu quand l'onglet est masqué.
- Versionnage du binaire (`mailhive version`, ldflags), intégration continue
  GitHub Actions, releases GoReleaser (binaires multi-plateformes) et image
  multi-arch publiée sur GHCR.
- Déploiement Docker : stack dev (Mailpit, SeaweedFS) et stack prod (image GHCR
  + S3) ; schéma OpenAPI servi via Swagger UI.
- Documentation bilingue FR/EN.

[Non publié]: https://github.com/statoon54/mailhive/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/statoon54/mailhive/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/statoon54/mailhive/releases/tag/v0.1.0
