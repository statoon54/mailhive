# MailHive

[![CI](https://github.com/statoon54/mailhive/actions/workflows/ci.yml/badge.svg)](https://github.com/statoon54/mailhive/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/statoon54/mailhive?sort=semver)](https://github.com/statoon54/mailhive/releases)
[![Go Reference](https://img.shields.io/badge/Go-module-00ADD8?logo=go)](https://github.com/statoon54/mailhive)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Français** · [English](README.en.md)

Plateforme multi-tenant d'envoi et de gestion d'e-mails, avec file d'attente asynchrone, templates embarqués, chiffrement SMTP et tableau de bord temps réel.

Développé dans le cadre du projet MailHive

![Tableau de bord MailHive](img/mailhive-dashboard.png)

---

## Table des matières

- [MailHive](#mailhive)
  - [Table des matières](#table-des-matières)
  - [Fonctionnalités](#fonctionnalités)
    - [Gestion multi-tenant](#gestion-multi-tenant)
    - [Envoi d'e-mails](#envoi-de-mails)
    - [Templates d'e-mails](#templates-de-mails)
    - [Configurations SMTP](#configurations-smtp)
    - [Personnalisation (Branding)](#personnalisation-branding)
    - [File d'attente et workers](#file-dattente-et-workers)
    - [Rate limiting — Token Bucket Redis](#rate-limiting--token-bucket-redis)
    - [Monitoring temps réel](#monitoring-temps-réel)
    - [Sécurité](#sécurité)
    - [Journal d'audit](#journal-daudit)
  - [Architecture](#architecture)
    - [Flux d'envoi d'un mail](#flux-denvoi-dun-mail)
  - [Stack technique](#stack-technique)
  - [Infrastructure Docker](#infrastructure-docker)
    - [Sous-commandes du binaire](#sous-commandes-du-binaire)
    - [Stockage des pièces jointes (S3 / SeaweedFS)](#stockage-des-pièces-jointes-s3--seaweedfs)
  - [Installation](#installation)
    - [Prérequis](#prérequis)
    - [Démarrage rapide](#démarrage-rapide)
    - [Développement local (sans Docker)](#développement-local-sans-docker)
    - [Déploiement en production](#déploiement-en-production)
  - [Configuration](#configuration)
    - [Mode simulation SMTP](#mode-simulation-smtp)
    - [Mailpit — Serveur SMTP de test](#mailpit--serveur-smtp-de-test)
    - [Tests d'intégration SMTP](#tests-dintégration-smtp)
  - [Utilisation](#utilisation)
    - [Authentification](#authentification)
    - [Envoyer un mail](#envoyer-un-mail)
    - [Planifier un envoi différé](#planifier-un-envoi-différé)
    - [Interface web](#interface-web)
  - [API REST](#api-rest)
    - [Routes publiques](#routes-publiques)
    - [Routes protégées (JWT requis)](#routes-protégées-jwt-requis)
      - [Mails](#mails)
      - [Templates](#templates)
      - [Configurations SMTP (1)](#configurations-smtp-1)
      - [Administration (rôle admin)](#administration-rôle-admin)
  - [Base de données](#base-de-données)
  - [Frontend](#frontend)
  - [Internationalisation (i18n)](#internationalisation-i18n)
  - [Génération de contenu par IA](#génération-de-contenu-par-ia)
  - [Commandes Make](#commandes-make)
  - [Auteur](#auteur)
  - [Licence](#licence)

---

## Fonctionnalités

### Gestion multi-tenant

- Chaque tenant dispose de son propre espace isolé (mails, templates, configs SMTP)
- Clé API unique par tenant pour l'authentification
- Paramètres individuels : limite de débit, burst, nombre max de destinataires
- Activation/désactivation de tenants par l'administrateur

### Envoi d'e-mails

- Envoi asynchrone via file d'attente Redis/Asynq (3 niveaux de priorité : critical, default, low)
- Support des destinataires multiples (to, cc, bcc)
- Corps texte et HTML
- Pièces jointes
- Planification d'envoi différé (`scheduled_at`) avec formats flexibles (RFC3339, datetime, date)
- Métadonnées personnalisées par mail
- Suivi du statut : `pending` → `queued` → `sending` → `sent` | `failed` | `cancelled`
- Annulation de mails en attente et relance de mails en échec

### Templates d'e-mails

- Création de templates réutilisables avec variables (syntaxe Go `{{.Variable}}`)
- Éditeur HTML maison : édition du code source avec aperçu en direct (iframe) et bascule source / aperçu
- Sanitisation HTML côté serveur (bluemonday) avant stockage et envoi
- Génération de contenu assistée par IA (voir [Génération de contenu par IA](#génération-de-contenu-par-ia))
- Prévisualisation avec données de test
- Slug unique par tenant pour référencement facile

### Configurations SMTP

- Plusieurs configs SMTP par tenant avec config par défaut
- Méthodes d'authentification : PLAIN, LOGIN, CRAM-MD5, NONE
- Politiques TLS : mandatory, opportunistic, none
- Mots de passe chiffrés en AES-GCM dans la base de données
- Test de connexion SMTP intégré

### Personnalisation (Branding)

- Titre et sous-titre de l'application personnalisables
- Upload de logo (PNG, JPEG, SVG)
- Fuseau horaire configurable (pour les dates sans timezone)
- Sélecteur de langue de l'interface (français / anglais)
- Tooltips contextuels sur tous les champs de formulaire

### File d'attente et workers

- Workers Asynq avec concurrence configurable (défaut : 50 goroutines)
- 3 queues avec priorités pondérées (critical:6, default:3, low:1)
- Retry automatique avec backoff exponentiel (max 5 tentatives)
- Circuit breaker par config SMTP (5 échecs → open, 30s cooldown → half-open)
- Rate limiting distribué par tenant via **token bucket Redis** (script Lua atomique)

### Rate limiting — Token Bucket Redis

Le contrôle de débit utilise un algorithme de **seau à jetons** (token bucket) implémenté dans Redis via un **script Lua atomique**, appliqué côté worker lors de l'envoi effectif des mails :

```txt
Redis Hash  rate_limit:{tenantID}
┌───────────────────────────────┐
│  tokens: 18.5                 │  ← jetons disponibles
│  last:   1711234567890123     │  ← dernier timestamp (µs)
└───────────────────────────────┘
           │
     Script Lua atomique
           │
    ┌──────▼──────┐
    │ elapsed =   │
    │ (now-last)  │  → tokens += elapsed × rate
    │   / 1e6     │  → cap(tokens, burst)
    └──────┬──────┘
           │
     tokens >= 1 ?
     ├─ oui → tokens -= 1, return 1 (autorisé)
     └─ non → return 0 (refusé → Asynq replanifie)
```

**Principe** : chaque tenant dispose d'un hash Redis (`rate_limit:{tenantID}`) contenant le nombre de tokens et le timestamp du dernier accès. Un script Lua atomique calcule les tokens accumulés depuis le dernier appel, consomme un token si disponible, et définit un TTL de sécurité. Cette approche est **distribuée** : plusieurs instances du worker partagent le même état de rate limiting via Redis.

Avec la configuration par défaut (`rate_limit=10`, `rate_burst=20`) :

| Situation | Résultat |
| ----------- | ---------- |
| 10 mails en 1 seconde | Tous passent (débit normal) |
| 20 mails d'un coup | Tous passent (le burst absorbe le pic) |
| 25 mails d'un coup | 20 passent, 5 replanifiés par Asynq |
| Silence 5s puis 20 mails | Tous passent (seau rerempli à sa capacité max) |

**Avantages de l'implémentation Redis** par rapport à un limiter en mémoire (`sync.Map`) :

- **Distribué** : fonctionne avec plusieurs instances du worker
- **Atomique** : le script Lua garantit l'absence de race conditions
- **Persistant** : l'état survit au redémarrage des workers (TTL auto-nettoyant)
- **Isolation** : chaque tenant a sa propre clé Redis

### Monitoring temps réel

- Tableau de bord avec statistiques issues de PostgreSQL (statuts des mails)
- Monitoring des queues Asynq en temps réel (sondage adaptatif : 2s tant qu'un envoi est en cours, 15s au repos, suspendu quand l'onglet est masqué)
- Métriques par queue : tâches actives, en attente, planifiées, en retry, archivées, latence
- Statistiques par tenant (mails envoyés, en attente, échoués)
- Graphiques interactifs (barres, circulaire) via Recharts

### Sécurité

- Authentification JWT stateless avec rafraîchissement de tokens
- Middleware de contrôle d'accès admin
- Isolation des données par tenant (tenant_id dans toutes les requêtes)
- Chiffrement AES-GCM des mots de passe SMTP
- Sanitisation HTML des templates côté serveur (bluemonday)
- Protection contre les injections SQL (requêtes paramétrées pgx)
- ID de requête unique (X-Request-ID) pour la traçabilité

### Journal d'audit

- Traçabilité complète des actions (création, modification, suppression, test, annulation, relance)
- Filtrage par type de ressource et statut
- Détails d'audit enrichis (destinataires, sujet, corps tronqué)
- Partitionnement PostgreSQL par mois pour les performances

---

## Architecture

Le projet suit une **architecture hexagonale** (ports & adapters) avec une séparation stricte des responsabilités :

```txt
cmd/
└── mailhive/main.go         # Binaire unique (API + Worker + migrations)

internal/
├── domain/                  # Entités métier et DTOs
│   ├── tenant.go            # Modèle Tenant + settings
│   ├── mail.go              # Modèle Mail + statuts + destinataires
│   ├── template.go          # Modèle Template + variables
│   ├── smtp_config.go       # Modèle SMTPConfig + auth/TLS
│   ├── dto.go               # Requêtes de création/mise à jour
│   └── errors.go            # Erreurs métier (ErrNotFound, ErrConflict, ...)
│
├── i18n/                    # Internationalisation backend (FR/EN)
│   └── i18n.go              # Messages traduits, détection Accept-Language
│
├── port/                    # Interfaces (contrats)
│   ├── repository.go        # Interfaces des repositories
│   └── service.go           # Interfaces des services + MailSender + QueueClient
│
├── service/                 # Logique métier
│   ├── auth_service.go      # Génération/refresh JWT
│   ├── tenant_service.go    # CRUD tenants
│   ├── mail_service.go      # Création mails, mise en queue, stats
│   ├── template_service.go  # CRUD templates + rendu Go template
│   ├── smtp_config_service.go # CRUD SMTP + chiffrement AES-GCM
│   └── llm_service.go       # Génération de contenu IA (Ollama / OpenAI)
│
├── handler/                 # Handlers HTTP (Echo v5)
│   ├── handler.go           # Helpers communs (ok, handleError, pagination)
│   ├── auth_handler.go      # POST /auth/token, /auth/refresh
│   ├── health_handler.go    # GET /health
│   ├── tenant_handler.go    # CRUD /admin/tenants
│   ├── mail_handler.go      # CRUD /mails + stats
│   ├── template_handler.go  # CRUD /templates + preview
│   ├── smtp_config_handler.go # CRUD /smtp-configs + test
│   ├── queue_handler.go     # GET /admin/queues (monitoring Asynq)
│   └── llm_handler.go       # POST /ai/generate, GET /ai/status
│
├── middleware/              # Middlewares Echo
│   ├── jwt.go               # Validation JWT
│   ├── tenant_context.go    # Extraction tenant depuis JWT
│   ├── admin_only.go        # Restriction admin
│   └── request_id.go        # Génération X-Request-ID
│
├── adapter/                 # Implémentations concrètes
│   ├── postgres/            # Repositories PostgreSQL (pgx/v5)
│   ├── redis/               # Client Redis
│   └── mailer/              # Envoi SMTP (go-mail)
│
├── worker/                  # Système de jobs asynchrones
│   ├── tasks.go             # Définition des types de tâches
│   ├── mail_handler.go      # Traitement des envois (retry, rate limit)
│   ├── server.go            # Serveur Asynq
│   └── queue_client.go      # Client d'enqueue
│
├── frontend/                # Frontend React embarqué (go:embed)
│   ├── embed.go             # Handler SPA avec fallback index.html
│   └── dist/                # Build Vite (généré par npm run build)
│
├── config/                  # Chargement configuration (env vars)
└── templates/               # Templates e-mail embarqués
```

### Flux d'envoi d'un mail

```mermaid
sequenceDiagram
    autonumber
    participant Client as Client (Frontend / curl)
    participant API as API (Echo)
    participant MailSvc as MailService
    participant QC as QueueClient (Adapter)
    participant AsynqC as asynq.Client
    participant PG as PostgreSQL
    participant Blob as BlobStore (S3/Postgres)
    participant Redis as Redis
    participant AsynqS as asynq.Server
    participant MailH as MailHandler
    participant SMTP as Serveur SMTP

    rect rgb(230, 245, 255)
    Note over Client,Redis: Phase 1 — API : Persister & envoyer les IDs dans la queue
    Client->>API: POST /api/v1/mails (JWT + payload)
    API->>MailSvc: Create(tenantID, CreateMailRequest)
    opt Pièces jointes présentes
        MailSvc->>MailSvc: storeAttachments : décode base64 + SHA-256
        MailSvc->>PG: AttachmentRepository.UpsertMeta(tenant, sha256)<br/>déduplication : 1 ligne par contenu
        MailSvc->>Blob: Put(tenant, sha256, contenu)<br/>uniquement si le contenu est nouveau
    end
    MailSvc->>PG: INSERT mail (status=pending) via MailRepository
    MailSvc->>PG: INSERT recipients + mail_attachments (liens mail→PJ)
    Note over MailSvc,QC: Seuls les IDs sont transmis,<br/>pas les données complètes
    MailSvc->>QC: EnqueueMailSend(mailID, tenantID, priority, scheduledAt)
    QC->>QC: NewMailSendTask() → payload = {mail_id, tenant_id}
    QC->>AsynqC: Enqueue(task)
    AsynqC->>Redis: PUSH {mail_id, tenant_id}<br/>dans asynq:{queue}
    Redis-->>AsynqC: OK
    AsynqC-->>QC: taskID
    QC-->>MailSvc: taskID
    MailSvc->>PG: UPDATE status=queued, task_id
    API-->>Client: 201 Created
    end

    rect rgb(255, 245, 230)
    Note over Redis,SMTP: Phase 2 — Worker : Reconstruire le contexte depuis la BDD
    AsynqS->>Redis: BRPOPLPUSH (polling)
    Redis-->>AsynqS: task {mail_id, tenant_id}
    AsynqS->>MailH: HandleMailSend(ctx, task)
    MailH->>MailH: Unmarshal → {mail_id, tenant_id}

    Note over MailH,PG: Le worker utilise les mêmes<br/>ports/repos que l'API pour<br/>reconstruire toutes les données
    MailH->>PG: MailRepository.GetByID(tenant_id, mail_id)
    PG-->>MailH: mail (from, subject, template_id, template_data...)
    MailH->>PG: MailRepository.GetRecipients(mail_id)
    PG-->>MailH: []recipients

    opt Pièces jointes liées au mail
        MailH->>PG: MailRepository.GetAttachmentRefs(mail_id)
        PG-->>MailH: []refs (filename, sha256, content_type)
        loop pour chaque pièce jointe
            MailH->>Blob: AttachmentService.Load(tenant, sha256)
            Blob-->>MailH: contenu
        end
        MailH->>MailH: reconstitue mail.Attachments (base64) avant l'envoi
    end

    opt Rendu template différé (subject/body vides)
        MailH->>PG: TemplateRepository.GetByID(tenant_id, template_id)
        PG-->>MailH: template (subject_tmpl, text_body, html_body)
        MailH->>MailH: Compile + Render(template, template_data)<br/>→ subject, textBody, htmlBody
    end

    MailH->>MailH: Rate Limiter (tenant)
    MailH->>PG: TenantRepository.GetByID(tenant_id)
    PG-->>MailH: tenant (rate_limit, rate_burst)

    MailH->>PG: SMTPConfigRepository.GetByID(tenant_id, smtp_config_id)
    PG-->>MailH: smtp_config (host, port, auth, password chiffré)
    MailH->>MailH: SMTPConfigService.DecryptPassword()

    MailH->>PG: MailRepository.UpdateStatus(sending) + IncrementAttempts
    MailH->>SMTP: sender.Send(smtp_config, mail)

    alt Succès
        SMTP-->>MailH: OK
        MailH->>PG: MailRepository.UpdateStatus(sent) + SetSentAt
    else Erreur permanente
        MailH->>PG: MailRepository.UpdateStatus(failed)
        MailH-->>AsynqS: SkipRetry
    else Erreur temporaire + retries restants
        MailH->>PG: MailRepository.UpdateStatus(pending)
        MailH-->>AsynqS: error → retry (backoff exponentiel)
    end
    end

    rect rgb(240, 255, 240)
    Note over QC,PG: Repositories partagés entre API et Worker
    Note over QC,PG: MailRepository — lecture/écriture des mails<br/>SMTPConfigRepository — configs SMTP<br/>TemplateRepository — templates de mail<br/>TenantRepository — tenants et rate limits<br/>AttachmentRepository — métadonnées des PJ (dédup par sha256)<br/>BlobStore — contenu des PJ (S3 ou table attachment_blobs)<br/><br/>Mêmes interfaces (port.*),<br/>mêmes implémentations,<br/>instances séparées (pools de connexion distincts)
    end
```

---

## Stack technique

| Composant | Technologie | Version |
| ----------- | ------------- | --------- |
| **Backend** | Go | 1.26 |
| **Framework HTTP** | Echo | v5 |
| **Base de données** | PostgreSQL | 18 |
| **Driver DB** | pgx | v5 |
| **Cache & Queue** | Valkey (compatible Redis) | 8 |
| **File d'attente** | Asynq | v0.26 |
| **Object store (pièces jointes)** | S3 compatible (SeaweedFS/MinIO/S3/R2) via minio-go | v7 |
| **Frontend** | React + TypeScript | 19 / 6 |
| **Bundler** | Vite (Rolldown) | 8 |
| **CSS** | Tailwind CSS | v4 |
| **Éditeur HTML** | Composant maison (code source + aperçu iframe) | — |
| **Sanitisation HTML** | bluemonday (backend) | v1.0 |
| **i18n frontend** | react-i18next + i18next | 17 / 26 |
| **i18n backend** | Package interne `internal/i18n` | — |
| **Graphiques** | Recharts | 3 |
| **Routeur** | React Router | v7 |
| **HTTP Client** | Axios | 1.x |

---

## Infrastructure Docker

L'application se distribue comme un **binaire unique** embarquant le frontend React (via `go:embed`) et se déploie via `docker compose` avec 3 services :

```txt
┌──────────────────────────────────────────────────┐
│                  Réseau Docker                    │
│                                                   │
│  ┌────────────────┐                               │
│  │   MailHive     │                               │
│  │ (API + Worker  │                               │
│  │  + Frontend)   │                               │
│  │    :8080       │                               │
│  └───────┬────────┘                               │
│          │                                        │
│    ┌─────┴──────┐                                 │
│    ▼            ▼                                 │
│ ┌──────────┐ ┌──────────┐   ┌──────────┐         │
│ │PostgreSQL│ │  Redis   │   │ Mailpit  │ (dev)   │
│ │  :5432   │ │  :6379   │   │ :1025    │         │
│ └──────────┘ └──────────┘   │ :8025 UI │         │
│                              └──────────┘         │
└──────────────────────────────────────────────────┘
```

| Service | Image | Port | Rôle |
| --------- | ------- | ------ | ------ |
| **mailhive** | Build multi-stage (Node + Go) | 8080 | API REST + Worker + Frontend SPA |
| **postgres** | postgres:18-alpine | 5432 | Base de données principale |
| **redis** | redis:7-alpine | 6379 | Cache + backend de la file d'attente |
| **mailpit** | axllent/mailpit | 1025 / 8025 | Serveur SMTP de test (profil `dev` uniquement) |
| **seaweedfs** | chrislusf/seaweedfs | 8333 (S3) / 8888 (filer) | Object store S3 local pour tester `BLOB_BACKEND=s3` (profil `dev` uniquement) |

Le binaire MailHive embarque le frontend React et le sert directement via un middleware SPA avec fallback sur `index.html` pour le routage côté client.

### Sous-commandes du binaire

```bash
mailhive serve          # API + Worker (défaut)
mailhive api            # API seule
mailhive worker         # Worker seul
mailhive migrate        # Appliquer les migrations
mailhive migrate-down   # Rollback dernière migration
```

### Stockage des pièces jointes (S3 / SeaweedFS)

Le contenu des pièces jointes est **adressé par contenu** (clé = SHA-256) et
**dédupliqué par tenant** : un même fichier envoyé à N destinataires n'est stocké
qu'une fois. Le backend de stockage est choisi par `BLOB_BACKEND` :

- `postgres` (défaut) : contenu dans la table `attachment_blobs` ;
- `s3` : contenu dans un object store compatible S3 (SeaweedFS, MinIO, AWS S3, R2).

Pour tester le backend S3 en local, une instance **SeaweedFS** est fournie sous le
profil `dev` :

```bash
make docker-dev-s3   # stack dev + SeaweedFS, BLOB_BACKEND=s3
```

Les objets sont rangés dans le bucket `mailhive-attachments` sous la clé
`<tenant_id>/<sha256-hex>`, et le contenu est stocké **déchiffré/brut** (un objet
par contenu unique).

#### Inspecter le contenu stocké

**Via le filer HTTP (port 8888, sans authentification — pratique en dev) :**

```bash
# Lister les objets du bucket (JSON)
curl -s -H "Accept: application/json" \
  "http://localhost:8888/buckets/mailhive-attachments/" | jq '.Entries[].FullPath'

# Télécharger un objet
curl -s "http://localhost:8888/buckets/mailhive-attachments/<tenant>/<hash>" -o blob.bin
```

On peut aussi naviguer dans le bucket depuis un navigateur :
<http://localhost:8888/buckets/mailhive-attachments/>.

**Via l'API S3 (port 8333, avec les identifiants de `docker/seaweedfs/s3.json` —
parité avec une vraie cible S3),** par ex. avec le client MinIO `mc` lancé en
conteneur (aucune installation requise) :

```bash
docker run --rm --network atsi-mailforge_default \
  -e MC_HOST_sw=http://mailhive:mailhive_secret@seaweedfs:8333 \
  minio/mc ls --recursive sw/mailhive-attachments
```

Avec l'AWS CLI : `aws --endpoint-url http://localhost:8333 --region us-east-1
s3 ls s3://mailhive-attachments/ --recursive` (identifiants `mailhive` /
`mailhive_secret`).

> Le script de charge `scripts/send_mails_attachments.sh` génère une pièce jointe de
> zéros (faux PDF) : un objet récupéré fera la bonne taille mais ne s'ouvrira pas
> comme un vrai PDF — vérifier avec `wc -c` plutôt que par un aperçu.

---

## Installation

### Prérequis

- Docker et Docker Compose
- (Optionnel pour le dev) Go 1.26+, Node.js 24+, Make

### Démarrage rapide

```bash
# Cloner le projet
git clone <url-du-repo>
cd mailhive

# Copier et adapter la configuration
cp .env.example .env

# Lancer les 3 services (mailhive + postgres + redis)
docker compose up --build -d

# Avec Mailpit (serveur SMTP de test) pour le développement
docker compose --profile dev up --build -d

# Vérifier les logs
docker compose logs -f
```

L'application est accessible sur :

- **Frontend + API** : <http://localhost:8080>
- **API REST** : <http://localhost:8080/api/v1>
- **Swagger UI** : <http://localhost:8080/swagger/>
- **Mailpit** (profil dev) : <http://localhost:8025>

### Développement local (sans Docker)

```bash
# Backend
cp .env.example .env
# Adapter les variables (DB_HOST=localhost, REDIS_ADDR=localhost:6379)
make run          # API + Worker (binaire unifié)
make run-api      # API seule
make run-worker   # Worker seul

# Frontend
cd frontend
npm install
npm run dev       # Serveur de dev Vite sur :5173

# Build complet (frontend + binaire Go)
make build        # Compile frontend puis Go avec embed
make build-go     # Go uniquement (si frontend déjà buildé)
```

### Déploiement en production

`docker-compose.prod.yml` exécute l'**image publiée** (GHCR) au lieu de
construire depuis les sources, avec les pièces jointes sur **S3** (SeaweedFS
auto-hébergé par défaut). C'est un fichier autonome (ne pas combiner avec
`docker-compose.yml`).

```bash
# Fournir les secrets via un .env (ou l'environnement) — voir .env.example
MAILHIVE_TAG=0.1.0 docker compose -f docker-compose.prod.yml up -d
# ou : make docker-prod
```

> Le tag de l'image suit le semver **sans préfixe `v`** (`0.1.0`, `0.1`, `latest`),
> contrairement au tag git (`v0.1.0`).

Les secrets `JWT_SECRET`, `ENCRYPTION_KEY` (64 hex), `ADMIN_API_KEY` et
`DB_PASSWORD` sont **obligatoires** (le démarrage échoue sinon). Les migrations
s'appliquent automatiquement au lancement. Seul le port `8080` est exposé.

À adapter pour un vrai déploiement :

- faire tourner les identifiants S3 dans `docker/seaweedfs/s3.json` et aligner
  `BLOB_S3_ACCESS_KEY` / `BLOB_S3_SECRET_KEY` ;
- pour un S3 externe (AWS S3, R2, MinIO managé) : renseigner `BLOB_S3_ENDPOINT`
  + identifiants, puis retirer le service `seaweedfs` et sa dépendance ;
- `MAILHIVE_IMAGE` permet de surcharger le dépôt de l'image.

#### Sans Compose (services existants)

L'image ne contient **que l'application** (binaire unique : API + worker +
frontend embarqué) ; elle ne fournit ni base de données, ni file, ni object
store. Compose n'est qu'une façon pratique de tout câbler — l'image tourne avec
n'importe quels services joignables (managés, Kubernetes, `docker run`…).

Dépendances de l'image :

| Service | Requis | Rôle | Alternatives |
| --- | --- | --- | --- |
| PostgreSQL | **oui** | données + migrations | toute instance Postgres joignable |
| Redis / Valkey | **oui** | file Asynq + rate limiting | tout serveur compatible Redis |
| S3 | seulement si `BLOB_BACKEND=s3` | contenu des pièces jointes | AWS S3, R2, MinIO… (défaut `postgres` : pas de S3) |

Exemple sans Compose, vers des services existants (pièces jointes en base) :

```bash
docker run -d -p 8080:8080 \
  -e DB_HOST=postgres.interne -e DB_PASSWORD=… \
  -e REDIS_ADDR=redis.interne:6379 \
  -e JWT_SECRET=… -e ENCRYPTION_KEY=… -e ADMIN_API_KEY=… \
  -e BLOB_BACKEND=postgres \
  ghcr.io/statoon54/mailhive:0.1.0
```

Seuls **PostgreSQL et un serveur compatible Redis** sont indispensables ; un
object store S3 n'est nécessaire qu'avec `BLOB_BACKEND=s3`. Les migrations sont
appliquées au démarrage.

---

## Configuration

Toutes les variables sont chargées depuis l'environnement (fichier `.env`) :

| Variable | Défaut | Description |
| ---------- | -------- | ------------- |
| `API_HOST` | `0.0.0.0` | Adresse d'écoute |
| `API_PORT` | `8080` | Port du serveur API |
| `DB_HOST` | `localhost` | Hôte PostgreSQL |
| `DB_PORT` | `5432` | Port PostgreSQL |
| `DB_USER` | `mailhive` | Utilisateur DB |
| `DB_PASSWORD` | — | Mot de passe DB |
| `DB_NAME` | `mailhive` | Nom de la base |
| `DB_SSL_MODE` | `disable` | Mode SSL (disable/require) |
| `DB_MAX_CONNS` | `25` | Connexions max au pool |
| `REDIS_ADDR` | `localhost:6379` | Adresse Redis |
| `REDIS_PASSWORD` | — | Mot de passe Redis |
| `REDIS_DB` | `0` | Base Redis |
| `JWT_SECRET` | — | Clé de signature JWT (min 32 chars) |
| `JWT_EXPIRATION` | `24h` | Durée de validité des tokens |
| `ADMIN_API_KEY` | — | Clé API administrateur |
| `ENCRYPTION_KEY` | — | Clé hex 32 bytes pour chiffrement SMTP |
| `WORKER_CONCURRENCY` | `50` | Goroutines du worker |
| `WORKER_QUEUE_CRITICAL` | `6` | Poids queue critical |
| `WORKER_QUEUE_DEFAULT` | `3` | Poids queue default |
| `WORKER_QUEUE_LOW` | `1` | Poids queue low |
| `DEFAULT_RATE_LIMIT` | `100` | Mails/seconde par tenant |
| `DEFAULT_RATE_BURST` | `200` | Burst par tenant |
| `SMTP_MODE` | `real` | Mode d'envoi : `real` (SMTP réel) ou `simulation` (log uniquement) |

### Mode simulation SMTP

Pour tester l'API sans serveur SMTP réel, activez le mode simulation :

```bash
SMTP_MODE=simulation
```

En mode simulation :

- Les mails ne sont **pas envoyés** via SMTP
- Les détails de chaque mail (expéditeur, destinataires, sujet, aperçu du corps, pièces jointes) sont **affichés dans les logs** du worker
- Le mail passe en statut `sent` comme en mode réel
- Un léger délai (100ms) simule le temps d'envoi

Pour envoyer réellement via SMTP, utilisez `SMTP_MODE=real` (défaut dans `docker-compose.yml`).

### Mailpit — Serveur SMTP de test

[Mailpit](https://github.com/axllent/mailpit) est intégré à la stack Docker comme serveur SMTP de test. Il capture tous les mails envoyés et offre une interface web pour les visualiser, sans jamais les transmettre réellement.

**Ports exposés :**

| Port | Protocole | Usage |
| ---- | --------- | ----- |
| 1025 | SMTP | Réception des mails (pas d'auth requise) |
| 8025 | HTTP | Interface web + API REST |

**Utilisation en développement :**

1. Démarrer la stack avec Mailpit : `docker compose --profile dev up -d`
2. Créer une config SMTP pointant vers Mailpit via l'API :

   ```bash
   curl -X POST http://localhost:8080/api/v1/smtp-configs \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "name": "Mailpit (dev)",
       "host": "mailpit",
       "port": 1025,
       "auth_method": "NONE",
       "tls_policy": "none",
       "from_email": "test@mailhive.dev",
       "from_name": "MailHive Dev",
       "is_default": true
     }'
   ```

3. Envoyer un mail via l'API (le mode par défaut est `SMTP_MODE=real`)
4. Visualiser les mails reçus sur <http://localhost:8025>

> **Note** : au démarrage en mode `real`, MailHive crée automatiquement un tenant admin et une config SMTP Mailpit par défaut si aucune n'existe.

### Tests d'intégration SMTP

Des tests d'intégration vérifient l'envoi réel de mails via Mailpit et le contenu reçu via son API REST. Ils utilisent le build tag `integration` et ne s'exécutent pas avec `go test ./...`.

**Stack de test dédiée** (`docker-compose.test.yml`) : contient uniquement Mailpit, sans dépendance à PostgreSQL ou Redis.

```bash
# Démarrer Mailpit pour les tests
make docker-test-up

# Lancer les tests d'intégration SMTP
make test-smtp

# Arrêter Mailpit
make docker-test-down
```

**Tests couverts :**

| Test | Vérification |
| ---- | ------------ |
| `TestIntegration_SendBasicMail` | Subject, expéditeur, destinataire, corps texte et HTML |
| `TestIntegration_SendMailWithMultipleRecipients` | Destinataires TO multiples, CC, BCC |
| `TestIntegration_SendMailWithAttachment` | Pièce jointe (nom, type MIME, présence) |
| `TestIntegration_SendHTMLOnlyMail` | Mail HTML sans corps texte |
| `TestIntegration_SendMailWithUTF8Characters` | Caractères accentués, CJK, emojis |

**Variables d'environnement optionnelles :**

| Variable | Défaut | Description |
| -------- | ------ | ----------- |
| `MAILPIT_URL` | `http://localhost:8025` | URL de l'API Mailpit |
| `MAILPIT_SMTP_HOST` | `localhost` | Hôte SMTP Mailpit |

---

## Utilisation

### Authentification

```bash
# Obtenir un token JWT (admin)
curl -X POST http://localhost:8080/api/v1/auth/token \
  -H "Content-Type: application/json" \
  -d '{"api_key": "admin-dev-key"}'

# Utiliser le token
export TOKEN="eyJ..."
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/mails/stats
```

### Envoyer un mail

```bash
curl -X POST http://localhost:8080/api/v1/mails \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "to": [{"email": "dest@example.com", "name": "Destinataire"}],
    "subject": "Test MailHive",
    "html_body": "<h1>Bonjour !</h1><p>Mail envoyé via MailHive.</p>"
  }'
```

### Planifier un envoi différé

```bash
curl -X POST http://localhost:8080/api/v1/mails \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "to": [{"email": "dest@example.com", "name": "Destinataire"}],
    "template_id": "ID_DU_TEMPLATE",
    "template_data": {"name": "Marie", "activation_url": "https://example.com/activate"},
    "scheduled_at": "2026-03-10 09:00:00"
  }'
```

Formats de date acceptés pour `scheduled_at` :

| Format | Exemple |
| ------ | ------- |
| RFC3339 | `2026-03-10T09:00:00Z` |
| RFC3339 avec offset | `2026-03-10T09:00:00+02:00` |
| Datetime avec T | `2026-03-10T09:00:00` |
| Datetime avec espace | `2026-03-10 09:00:00` |
| Date seule | `2026-03-10` |

### Interface web

Connectez-vous sur <http://localhost:8080> avec la clé API d'un tenant. Le dashboard affiche :

- Statistiques globales (total, envoyés, échoués, tenants)
- Graphiques de répartition par statut et par tenant
- Monitoring temps réel des queues Asynq (rafraîchi toutes les 2s)
- Liste des derniers mails envoyés

---

## API REST

Spécification **OpenAPI** : [`api/openapi.yaml`](api/openapi.yaml). Elle est aussi
servie par l'application sur `/swagger/openapi.yaml`, avec l'UI Swagger sur
<http://localhost:8080/swagger/>.

### Routes publiques

| Méthode | Endpoint | Description |
| --------- | ---------- | ------------- |
| POST | `/api/v1/auth/token` | Générer un token JWT |
| POST | `/api/v1/auth/refresh` | Rafraîchir un token JWT |
| GET | `/api/v1/health` | Vérification de santé (DB + Redis) |

### Routes protégées (JWT requis)

#### Mails

| Méthode | Endpoint | Description |
| --------- | ---------- | ------------- |
| POST | `/mails` | Créer et mettre en queue un mail |
| GET | `/mails` | Lister les mails (pagination, filtre par statut) |
| GET | `/mails/stats` | Statistiques des mails du tenant |
| GET | `/mails/:id` | Détail d'un mail avec destinataires |
| POST | `/mails/:id/cancel` | Annuler un mail en attente |
| POST | `/mails/:id/retry` | Relancer un mail en échec |

#### Templates

| Méthode | Endpoint | Description |
| --------- | ---------- | ------------- |
| POST | `/templates` | Créer un template |
| GET | `/templates` | Lister les templates |
| GET | `/templates/:id` | Détail d'un template |
| PUT | `/templates/:id` | Modifier un template |
| DELETE | `/templates/:id` | Supprimer un template |
| POST | `/templates/:id/preview` | Prévisualiser avec des données |

#### Configurations SMTP (1)

| Méthode | Endpoint | Description |
| --------- | ---------- | ------------- |
| POST | `/smtp-configs` | Créer une config SMTP |
| GET | `/smtp-configs` | Lister les configs |
| GET | `/smtp-configs/:id` | Détail d'une config |
| PUT | `/smtp-configs/:id` | Modifier une config |
| DELETE | `/smtp-configs/:id` | Supprimer une config |
| POST | `/smtp-configs/:id/test` | Tester la connexion SMTP |

#### Administration (rôle admin)

| Méthode | Endpoint | Description |
| --------- | ---------- | ------------- |
| POST | `/admin/tenants` | Créer un tenant |
| GET | `/admin/tenants` | Lister les tenants |
| GET | `/admin/tenants/:id` | Détail d'un tenant |
| PUT | `/admin/tenants/:id` | Modifier un tenant |
| DELETE | `/admin/tenants/:id` | Supprimer un tenant |
| POST | `/admin/tenants/:id/regenerate-key` | Renouveler la clé API d'un tenant |
| GET | `/admin/stats/by-tenant` | Stats mails agrégées par tenant |
| GET | `/admin/queues` | Monitoring des queues Asynq |

---

## Base de données

Schéma PostgreSQL, migrations automatiques au démarrage :

| Table | Description |
| ------- | ------------- |
| `tenants` | Comptes multi-tenant (UUID, slug unique, clé API, settings JSON) |
| `smtp_configs` | Configurations SMTP par tenant (mot de passe chiffré AES-GCM) |
| `mail_templates` | Templates d'e-mail avec variables (JSON) |
| `mails` | Messages e-mail (statut, tentatives, planification) |
| `mail_recipients` | Destinataires par mail (to/cc/bcc) |
| `attachments` | Métadonnées des pièces jointes, adressées par contenu et dédupliquées par tenant (`tenant_id`, `sha256`) |
| `attachment_blobs` | Contenu des pièces jointes pour le backend `postgres` (vide en backend `s3`) |
| `mail_attachments` | Liens mail → pièce jointe (filename propre au mail) |
| `app_branding` | Personnalisation de l'application (titre, logo, fuseau, langue) |
| `audit_logs` | Journal d'audit, **partitionné par mois** |

Les tables `mails` et `mail_recipients` ont des variantes d'archivage
partitionnées (`*_archive`). Index notamment sur `tenant_id`, `status`,
`(tenant_id, status)`, `created_at DESC`.

---

## Frontend

SPA React servie par le binaire (embarquée via `go:embed`) sur <http://localhost:8080> ; en développement, le serveur Vite tourne sur `:5173` :

| Page | Route | Description |
| ------ | ------- | ------------- |
| Connexion | `/login` | Authentification par clé API |
| Tableau de bord | `/` | Dashboard avec stats, graphiques et monitoring queues |
| Mails | `/mails` | Liste paginée avec filtres par statut |
| Détail mail | `/mails/:id` | Détails, destinataires, pièces jointes |
| Templates | `/templates` | Gestion CRUD + prévisualisation |
| Configs SMTP | `/smtp-configs` | Gestion CRUD + test de connexion |
| Tenants | `/tenants` | Gestion des tenants (admin) |

---

## Internationalisation (i18n)

Le frontend est entièrement internationalisé via **react-i18next** avec détection automatique de la langue du navigateur :

- **Langues supportées** : Français (`fr`), Anglais (`en`)
- **Détection** : `localStorage` → `navigator.language` → fallback `fr`
- **Fichiers de traduction** : `frontend/src/i18n/fr.json`, `frontend/src/i18n/en.json`
- **Sélecteur de langue** dans la barre de navigation

Toutes les pages, formulaires, messages d'erreur et libellés de l'éditeur HTML sont traduits.

---

## Génération de contenu par IA

L'éditeur de templates intègre un assistant IA pour générer du contenu HTML d'e-mail à partir d'un prompt en langage naturel.

**Fournisseurs supportés :**

| Variable | Description | Défaut |
| --- | --- | --- |
| `LLM_PROVIDER` | Fournisseur LLM (`ollama` ou `openai`) | `ollama` |
| `LLM_BASE_URL` | URL de l'API du fournisseur | `http://localhost:11434` |
| `LLM_MODEL` | Modèle à utiliser | `llama3` |
| `LLM_API_KEY` | Clé API (requis pour OpenAI) | — |

**Fonctionnement :**

1. L'utilisateur décrit le contenu souhaité dans une modale
2. `POST /api/v1/ai/generate` envoie le prompt au LLM configuré
3. Le LLM génère uniquement le contenu HTML interne (pas de `<html>`, `<body>`)
4. Le résultat est inséré dans l'éditeur TipTap, modifiable avant sauvegarde

---

## Commandes Make

```bash
make build             # Compiler frontend + binaire Go (avec embed)
make build-go          # Compiler le binaire Go uniquement
make build-frontend    # Compiler le frontend uniquement
make run               # Lancer MailHive (API + Worker)
make run-api           # Lancer l'API seule
make run-worker        # Lancer le worker seul
make test              # Exécuter les tests
make test-unit         # Tests unitaires uniquement
make test-integration  # Tests d'intégration (BDD)
make test-smtp         # Tests d'intégration SMTP (Mailpit requis)
make test-all          # Tous les tests avec timeout étendu
make lint              # Linter (golangci-lint v2)
make tidy              # go mod tidy
make migrate-up        # Exécuter les migrations
make migrate-down      # Rollback dernière migration
make docker-up         # docker compose up --build -d
make docker-down       # docker compose down
make docker-logs       # docker compose logs -f
make docker-dev        # Stack dev (Mailpit) en profil dev
make docker-dev-s3     # Stack dev + SeaweedFS, pièces jointes sur S3 (BLOB_BACKEND=s3)
make docker-dev-down   # Arrêter la stack dev
make docker-dev-logs   # Logs de la stack dev
make docker-test-up    # Démarrer Mailpit (tests SMTP)
make docker-test-down  # Arrêter Mailpit
```

---

## Auteur

- **Franck Paszkowski**
- contact: <statoon54@gmail.com>

---

## Licence

Ce projet est distribué sous licence MIT. Voir le fichier [LICENSE](LICENSE) pour plus de détails.
