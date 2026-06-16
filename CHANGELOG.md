# Changelog

Toutes les modifications notables sont documentées ici.

Le format s'inspire de [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/)
et le projet suit le [versionnage sémantique](https://semver.org/lang/fr/).

> Les notes des versions publiées (tags `vX.Y.Z`) sont générées automatiquement
> par GoReleaser à partir des commits et disponibles dans les
> [GitHub Releases](https://github.com/statoon54/mailhive/releases). Cette page
> récapitule les changements non encore publiés.

## [Non publié]

- Déduplication et externalisation des pièces jointes (backend PostgreSQL ou S3
  compatible SeaweedFS/MinIO), avec téléchargement depuis l'UI admin.
- Sondage adaptatif du front (liste des mails, détail, tableau de bord).
- Versionnage du binaire (`mailhive version`) et intégration continue GitHub.
