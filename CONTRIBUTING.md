# Contribuer à MailHive

Merci de votre intérêt ! Ce document décrit comment construire, tester et
proposer des modifications.

## Prérequis

- Go (voir la directive `go` dans `go.mod`)
- Node.js 24+ (frontend React)
- Docker + Docker Compose (stack locale, tests d'intégration)
- `golangci-lint` v2 (linter Go)

## Mise en route

```bash
# Backend + frontend en local via Docker (Mailpit comme SMTP de test)
make docker-dev

# ou exécuter le binaire directement (API + worker)
make run
```

Voir le `README.md` pour la configuration (`.env.example`) et les sous-commandes.

## Boucle de développement

```bash
make build            # compile le frontend puis le binaire Go (avec version)
make lint             # golangci-lint
make test-unit        # tests unitaires (sans Docker)
make test-integration # tests d'intégration (testcontainers, Docker requis)
```

Côté frontend :

```bash
cd frontend
npm install
npm run build         # tsc + vite
```

**Avant toute Pull Request, assurez-vous que `build`, `lint` et les tests
passent au vert.**

## Style

- Backend en **Go** (architecture hexagonale : `internal/{domain,port,service,adapter,handler}`).
- Frontend en **React + TypeScript + Tailwind**.
- Commentaires et messages de commit en **français** ; clés/champs techniques en anglais.
- Interface et messages utilisateur bilingues **FR/EN** (i18n) — toute chaîne
  visible doit avoir sa clé dans `frontend/src/i18n/{fr,en}.json`.

## Commits

Format **Conventional Commits** :

```
feat(scope): …    # nouvelle fonctionnalité
fix(scope): …     # correction de bug
chore/docs/test(scope): …
```

Le changelog des releases est généré automatiquement à partir de ces préfixes.

## Pull Requests

1. Créez une branche dédiée (`feat/…`, `fix/…`, `chore/…`).
2. Gardez la PR ciblée sur un seul sujet.
3. Vérifiez que la CI (build, lint, tests) est verte.
4. Décrivez le quoi et le pourquoi du changement.

## Signaler un bug

Ouvrez une *issue* avec : version (`mailhive version`), étapes de reproduction,
comportement attendu vs observé, et logs pertinents.
