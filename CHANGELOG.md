# Changelog

Toutes les modifications notables sont documentées ici.

Le format s'inspire de [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/)
et le projet suit le [versionnage sémantique](https://semver.org/lang/fr/).

> Les notes détaillées de chaque version publiée sont aussi disponibles dans les
> [GitHub Releases](https://github.com/statoon54/mailhive/releases) (générées par
> GoReleaser à partir des commits).

## [Non publié]

### Modifié — rupture du contrat d'API

Le moteur JSON passe à `encoding/json/v2` (Go 1.27). Trois changements sont
visibles par les clients de l'API :

- Les **tableaux et objets vides** sont sérialisés `[]` et `{}` au lieu de
  `null` (champs `rules`, `issues`, `clients`, `links`, `variables`, `data`,
  `items`). C'est ce que décrivait déjà `openapi.yaml`, dans lequel aucun champ
  tableau n'est `nullable`.
- Les caractères **HTML ne sont plus échappés** en `\uXXXX` dans les chaînes :
  `<b>` est écrit tel quel au lieu de `\u003cb\u003e`. Le JSON reste valide et
  tout décodeur conforme est indifférent au changement.
- Une **clé JSON dupliquée** dans un corps de requête est désormais rejetée
  (400) au lieu d'être silencieusement résolue par la dernière occurrence.

La **tolérance à la casse** des noms de champs est conservée : `{"SUBJECT": …}`
continue d'alimenter `subject`. Cette tolérance est transitoire et sera retirée
lors d'une version majeure, avec préavis.

### Ajouté

- Prise en charge de **Go 1.27** (le projet ne compile plus avec une version
  antérieure).

### Corrigé

- Les erreurs de type dans un corps de requête indiquaient un type attendu vide
  pour les tableaux et les objets (« Attendu , reçu … »).
- Le type reçu n'était pas traduit en français (« reçu number » → « reçu
  nombre »).
- Le nom du champ fautif est désormais remonté pour les **champs imbriqués**
  (`to[0].email` était auparavant rapporté sans nom).
- Sur les endpoints de modification partielle, une **mise à vide explicite**
  d'un champ texte (`"name": ""`) est correctement transmise ; elle aurait été
  silencieusement supprimée par la sémantique `omitempty` de json/v2.

### Documentation

- Démarrage rapide « Make » + exemples d'envoi enrichis (CC/BCC, pièce jointe,
  envoi via template, envoi différé).
- README anglais (`README.en.md`) mis à parité sur les exemples.
- Diagramme du flux d'envoi remplacé par un flowchart (plus lisible sur GitHub).

### Interne

- Alignement mémoire de structs (fieldalignment) — sans impact fonctionnel.
- Modernisation du code aux idiomes Go 1.27 (`errors.AsType`, `sync.WaitGroup.Go`,
  `for range n`), vérifiée en CI par `go fix -diff`.
- CI : actions GitHub remises à niveau, `golangci-lint` via action officielle,
  `govulncheck`, exécution des tests d'intégration, et `dependabot.yml` pour
  éviter que le retard se reforme.

## [0.1.3] - 2026-06-16

### Ajouté

- Validation du **format des adresses `cc`/`bcc`** à la création d'un mail
  (auparavant seul `to` était validé).

## [0.1.2] - 2026-06-16

### Corrigé

- Outillage de release : le changelog GoReleaser exclut désormais aussi les
  commits **scopés** (`docs(...)`, `chore(...)`, `test(...)`).

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

[Non publié]: https://github.com/statoon54/mailhive/compare/v0.1.3...HEAD
[0.1.3]: https://github.com/statoon54/mailhive/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/statoon54/mailhive/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/statoon54/mailhive/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/statoon54/mailhive/releases/tag/v0.1.0
