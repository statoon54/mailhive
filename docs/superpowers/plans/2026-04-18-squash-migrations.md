# Migration Squash Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the 11 incremental development migrations with 3 logically-grouped baseline migrations for MailHive's v1 release, producing a schema that is byte-for-byte identical to running the full 11-migration history on a fresh database.

**Architecture:** A verification script (`scripts/verify-migration-squash.sh`) is written first. It runs the old migration set (from `main`) and the new migration set (from the working branch) against two throwaway Postgres 18 containers, `pg_dump --schema-only` each, and diffs them. "Success" means an empty diff. We then swap the 11 old SQL files for 3 new ones (`000001_core`, `000002_mails`, `000003_audit`) plus their `.down.sql` counterparts, iterating against the script until the diff is empty.

**Tech Stack:** golang-migrate v4 (iofs embed + CLI for verification), PostgreSQL 18 Alpine Docker image, `pg_dump`, bash.

**Reference:** [docs/superpowers/specs/2026-04-18-squash-migrations-design.md](../specs/2026-04-18-squash-migrations-design.md)

**Non-goals (do not do any of these):**
- Change the `tenants.settings` JSONB default (it stays `'{"rate_limit": 10, "rate_burst": 20, "max_destinataires": 1000}'`). The original `000010`/`000011` migrations only `UPDATE`d existing rows; they did not alter the default.
- Change `internal/adapter/postgres/migrate.go` or `migrations/embed.go`.
- Add or remove any test.
- Change column order in any table — preserve the cumulative order produced by the incremental `ALTER TABLE ADD COLUMN` statements (see per-task column lists).

---

## Task 1: Verification script

**Files:**
- Create: `scripts/verify-migration-squash.sh`

The script is the TDD "test." Running it with the same ref on both sides (e.g., `main main`) must produce an empty diff — that proves the mechanism is sound before we touch any migration file. Once we start swapping migrations on the branch, running it as `main HEAD` reveals any schema drift.

- [ ] **Step 1.1: Write the verification script**

Create `scripts/verify-migration-squash.sh` with this exact content:

```bash
#!/usr/bin/env bash
# Verifies that the migration set at NEW_REF produces a schema
# byte-for-byte identical to the set at OLD_REF.
#
# Usage: scripts/verify-migration-squash.sh [OLD_REF] [NEW_REF]
#   Defaults: OLD_REF=main, NEW_REF=HEAD
#
# Requires: docker, migrate CLI (golang-migrate v4), pg_dump, git.
set -euo pipefail

OLD_REF=${1:-main}
NEW_REF=${2:-HEAD}

OLD_CONTAINER=pg-squash-old-$$
NEW_CONTAINER=pg-squash-new-$$
OLD_PORT=54321
NEW_PORT=54322

OLD_DIR=$(mktemp -d)
NEW_DIR=$(mktemp -d)
OLD_DUMP=$(mktemp)
NEW_DUMP=$(mktemp)

cleanup() {
    docker rm -f "$OLD_CONTAINER" "$NEW_CONTAINER" >/dev/null 2>&1 || true
    rm -rf "$OLD_DIR" "$NEW_DIR" "$OLD_DUMP" "$NEW_DUMP"
}
trap cleanup EXIT

extract_migrations() {
    local ref=$1 dest=$2
    # Extract only the migrations/ subtree from the git ref.
    git archive "$ref" migrations/ | tar -x -C "$dest" --strip-components=1
    # migrate CLI chokes on non-migration files; drop .go and non-SQL.
    find "$dest" -maxdepth 1 -type f ! -name '*.sql' -delete
}

start_pg() {
    local name=$1 port=$2
    docker run --rm -d \
        --name "$name" \
        -e POSTGRES_PASSWORD=test \
        -e POSTGRES_USER=postgres \
        -e POSTGRES_DB=postgres \
        -p "${port}:5432" \
        postgres:18-alpine >/dev/null
    # Wait for readiness.
    for _ in $(seq 1 30); do
        if docker exec "$name" pg_isready -U postgres >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
    done
    echo "ERROR: $name did not become ready" >&2
    return 1
}

run_migrations() {
    local dir=$1 port=$2
    migrate \
        -source "file://$dir" \
        -database "postgres://postgres:test@localhost:${port}/postgres?sslmode=disable" \
        up
}

dump_schema() {
    local port=$1 out=$2
    PGPASSWORD=test pg_dump \
        --schema-only \
        --no-owner \
        --no-privileges \
        --exclude-table=schema_migrations \
        -h localhost -p "$port" -U postgres postgres > "$out"
}

echo ">>> Extracting migrations: $OLD_REF -> $OLD_DIR"
extract_migrations "$OLD_REF" "$OLD_DIR"
echo ">>> Extracting migrations: $NEW_REF -> $NEW_DIR"
extract_migrations "$NEW_REF" "$NEW_DIR"

echo ">>> Starting Postgres containers"
start_pg "$OLD_CONTAINER" "$OLD_PORT"
start_pg "$NEW_CONTAINER" "$NEW_PORT"

echo ">>> Running OLD migrations ($OLD_REF)"
run_migrations "$OLD_DIR" "$OLD_PORT"
echo ">>> Running NEW migrations ($NEW_REF)"
run_migrations "$NEW_DIR" "$NEW_PORT"

echo ">>> Dumping schemas"
dump_schema "$OLD_PORT" "$OLD_DUMP"
dump_schema "$NEW_PORT" "$NEW_DUMP"

echo ">>> Diff (empty means success):"
if diff -u "$OLD_DUMP" "$NEW_DUMP"; then
    echo ">>> OK: schemas match"
    exit 0
else
    echo ">>> FAIL: schemas differ (see diff above)"
    exit 1
fi
```

- [ ] **Step 1.2: Make it executable**

Run: `chmod +x scripts/verify-migration-squash.sh`

- [ ] **Step 1.3: Verify the script works with identical refs (should produce empty diff)**

Run: `./scripts/verify-migration-squash.sh main main`
Expected: script exits 0, final line is `>>> OK: schemas match`, no diff output.

If it fails, debug the script (Docker not running, migrate/pg_dump not installed, port already in use) before proceeding. Do not move to Task 2 until this step succeeds.

- [ ] **Step 1.4: Commit**

```bash
git add scripts/verify-migration-squash.sh
git commit -m "$(cat <<'EOF'
chore(db): ajouter le script de vérification du squash des migrations

Compare le schéma produit par deux refs Git en exécutant leurs
migrations dans deux conteneurs Postgres éphémères, puis diff
des pg_dump --schema-only. Une sortie vide signifie que les
deux ensembles de migrations sont équivalents.
EOF
)"
```

---

## Task 2: Write the 3 new up migrations and delete the 11 old ones

**Files:**
- Create: `migrations/000001_core.up.sql`
- Create: `migrations/000002_mails.up.sql`
- Create: `migrations/000003_audit.up.sql`
- Delete: `migrations/000001_create_tenants.up.sql` (+ `.down.sql`)
- Delete: `migrations/000002_create_smtp_configs.up.sql` (+ `.down.sql`)
- Delete: `migrations/000003_create_mail_templates.up.sql` (+ `.down.sql`)
- Delete: `migrations/000004_create_mails.up.sql` (+ `.down.sql`)
- Delete: `migrations/000005_create_mail_recipients.up.sql` (+ `.down.sql`)
- Delete: `migrations/000006_create_app_branding.up.sql` (+ `.down.sql`)
- Delete: `migrations/000007_create_audit_logs.up.sql` (+ `.down.sql`)
- Delete: `migrations/000008_create_mails_archive.up.sql` (+ `.down.sql`)
- Delete: `migrations/000009_add_spam_score_and_tags.up.sql` (+ `.down.sql`)
- Delete: `migrations/000010_add_tenant_language.up.sql` (+ `.down.sql`)
- Delete: `migrations/000011_add_store_body.up.sql` (+ `.down.sql`)

**Column order rationale:** Original migrations built `mails` and `mails_archive` incrementally via `ALTER TABLE ADD COLUMN`, which appends columns at the end. `pg_dump` preserves creation order, so the squashed `CREATE TABLE` must list columns in the exact cumulative order: the baseline columns first, then (for `mails`) `spam_score`, `tags`, `compressed_body` at the very end; for `mails_archive`, `archived_at` last from the original `000008`, then `spam_score`, `tags`, `compressed_body` appended. Deviating from this order will make the verification diff non-empty even though tables are "logically" the same.

- [ ] **Step 2.1: Create `migrations/000001_core.up.sql` with this exact content**

```sql
-- Baseline v1 : tables de configuration (tenants, SMTP, templates, branding).

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tenants
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    api_key VARCHAR(64) NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    settings JSONB NOT NULL DEFAULT '{"rate_limit": 10, "rate_burst": 20, "max_destinataires": 1000}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_api_key ON tenants(api_key);
CREATE INDEX idx_tenants_is_active ON tenants(is_active);

-- Configurations SMTP par tenant
CREATE TABLE smtp_configs (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    host VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL DEFAULT 587,
    username VARCHAR(255),
    password TEXT,
    auth_method VARCHAR(20) NOT NULL DEFAULT 'PLAIN',
    tls_policy VARCHAR(20) NOT NULL DEFAULT 'opportunistic',
    from_email VARCHAR(255) NOT NULL,
    from_name VARCHAR(255) NOT NULL DEFAULT '',
    is_default BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    charset VARCHAR(20) NOT NULL DEFAULT 'UTF-8',
    encoding VARCHAR(20) NOT NULL DEFAULT 'quoted-printable',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_smtp_configs_tenant_id ON smtp_configs(tenant_id);

-- Contrainte : un seul is_default=true par tenant
CREATE UNIQUE INDEX idx_smtp_configs_default_per_tenant
    ON smtp_configs(tenant_id) WHERE is_default = true;

-- Templates de mails
CREATE TABLE mail_templates (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    subject_tmpl TEXT NOT NULL DEFAULT '',
    text_body TEXT NOT NULL DEFAULT '',
    html_body TEXT NOT NULL DEFAULT '',
    variables JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(tenant_id, slug)
);

CREATE INDEX idx_mail_templates_tenant_id ON mail_templates(tenant_id);

-- Branding applicatif (ligne unique : id = 1)
CREATE TABLE IF NOT EXISTS app_branding (
    id          INTEGER PRIMARY KEY CHECK (id = 1),
    app_title   VARCHAR(255) NOT NULL DEFAULT 'MailHive',
    app_subtitle VARCHAR(255) NOT NULL DEFAULT 'Gestion des mails',
    logo_data   TEXT NOT NULL DEFAULT '',
    logo_content_type VARCHAR(100) NOT NULL DEFAULT '',
    timezone    VARCHAR(50) NOT NULL DEFAULT 'Europe/Paris',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO app_branding (id) VALUES (1) ON CONFLICT DO NOTHING;
```

- [ ] **Step 2.2: Create `migrations/000002_mails.up.sql` with this exact content**

Column order for `mails`: baseline (from original `000004`) first, then `spam_score`, `tags`, `compressed_body` at the end. `created_at` / `updated_at` come *before* the appended columns because they were in the original `CREATE TABLE`.

Column order for `mails_archive`: baseline (from original `000008`) including `archived_at` at the end, then `spam_score`, `tags`, `compressed_body` appended after.

```sql
-- Baseline v1 : données mail (envois + archivage) et types ENUM partagés.

-- ENUMs (doivent exister avant les tables qui les référencent)
CREATE TYPE mail_status AS ENUM ('pending', 'queued', 'sending', 'sent', 'failed', 'cancelled');
CREATE TYPE mail_priority AS ENUM ('critical', 'default', 'low');
CREATE TYPE recipient_type AS ENUM ('to', 'cc', 'bcc');

-- Table principale des mails
CREATE TABLE mails (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    smtp_config_id UUID REFERENCES smtp_configs(id) ON DELETE SET NULL,
    template_id UUID REFERENCES mail_templates(id) ON DELETE SET NULL,
    from_email VARCHAR(255) NOT NULL,
    from_name VARCHAR(255) NOT NULL DEFAULT '',
    subject TEXT NOT NULL,
    text_body TEXT NOT NULL DEFAULT '',
    html_body TEXT NOT NULL DEFAULT '',
    template_data JSONB NOT NULL DEFAULT '{}',
    attachments JSONB NOT NULL DEFAULT '[]',
    status mail_status NOT NULL DEFAULT 'pending',
    status_message TEXT NOT NULL DEFAULT '',
    attempts INTEGER NOT NULL DEFAULT 0,
    priority mail_priority NOT NULL DEFAULT 'default',
    scheduled_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    task_id VARCHAR(255) NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    spam_score REAL,
    tags TEXT[] NOT NULL DEFAULT '{}',
    compressed_body BYTEA
);

CREATE INDEX idx_mails_tenant_status_created ON mails (tenant_id, status, created_at DESC);
CREATE INDEX idx_mails_created_at ON mails(created_at DESC);
CREATE INDEX idx_mails_tags ON mails USING GIN (tags);

-- Destinataires
CREATE TABLE mail_recipients (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    mail_id UUID NOT NULL REFERENCES mails(id) ON DELETE CASCADE,
    type recipient_type NOT NULL,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL DEFAULT ''
);

CREATE INDEX idx_mail_recipients_mail_id ON mail_recipients(mail_id);

-- Archive des mails terminés (partitionnée, sans FK pour la performance)
CREATE TABLE mails_archive (
    id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    smtp_config_id UUID,
    template_id UUID,
    from_email VARCHAR(255) NOT NULL,
    from_name VARCHAR(255) NOT NULL DEFAULT '',
    subject TEXT NOT NULL,
    text_body TEXT NOT NULL DEFAULT '',
    html_body TEXT NOT NULL DEFAULT '',
    template_data JSONB NOT NULL DEFAULT '{}',
    attachments JSONB NOT NULL DEFAULT '[]',
    status mail_status NOT NULL,
    status_message TEXT NOT NULL DEFAULT '',
    attempts INTEGER NOT NULL DEFAULT 0,
    priority mail_priority NOT NULL DEFAULT 'default',
    scheduled_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    task_id VARCHAR(255) NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    spam_score REAL,
    tags TEXT[] NOT NULL DEFAULT '{}',
    compressed_body BYTEA,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Partition par défaut (attrape-tout)
CREATE TABLE mails_archive_default PARTITION OF mails_archive DEFAULT;

-- Archive des destinataires
CREATE TABLE mail_recipients_archive (
    id UUID NOT NULL,
    mail_id UUID NOT NULL,
    type recipient_type NOT NULL,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL DEFAULT ''
);

CREATE INDEX idx_mails_archive_tenant_created ON mails_archive (tenant_id, created_at DESC);
CREATE INDEX idx_mail_recipients_archive_mail_id ON mail_recipients_archive (mail_id);
```

- [ ] **Step 2.3: Create `migrations/000003_audit.up.sql` with this exact content**

```sql
-- Baseline v1 : journaux d'audit partitionnés.

CREATE TABLE audit_logs (
    id UUID NOT NULL DEFAULT uuidv7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    action VARCHAR(20) NOT NULL,
    resource_type VARCHAR(30) NOT NULL,
    resource_id VARCHAR(255),
    status VARCHAR(10) NOT NULL,
    status_code INTEGER NOT NULL,
    error_message TEXT,
    method VARCHAR(10) NOT NULL,
    path TEXT NOT NULL,
    details TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Partition par défaut (attrape-tout de sécurité)
CREATE TABLE audit_logs_default PARTITION OF audit_logs DEFAULT;

CREATE INDEX idx_audit_logs_tenant_created ON audit_logs (tenant_id, created_at DESC);
CREATE INDEX idx_audit_logs_status ON audit_logs (status);
```

- [ ] **Step 2.4: Delete the 11 old up/down files**

```bash
git rm migrations/000001_create_tenants.up.sql migrations/000001_create_tenants.down.sql
git rm migrations/000002_create_smtp_configs.up.sql migrations/000002_create_smtp_configs.down.sql
git rm migrations/000003_create_mail_templates.up.sql migrations/000003_create_mail_templates.down.sql
git rm migrations/000004_create_mails.up.sql migrations/000004_create_mails.down.sql
git rm migrations/000005_create_mail_recipients.up.sql migrations/000005_create_mail_recipients.down.sql
git rm migrations/000006_create_app_branding.up.sql migrations/000006_create_app_branding.down.sql
git rm migrations/000007_create_audit_logs.up.sql migrations/000007_create_audit_logs.down.sql
git rm migrations/000008_create_mails_archive.up.sql migrations/000008_create_mails_archive.down.sql
git rm migrations/000009_add_spam_score_and_tags.up.sql migrations/000009_add_spam_score_and_tags.down.sql
git rm migrations/000010_add_tenant_language.up.sql migrations/000010_add_tenant_language.down.sql
git rm migrations/000011_add_store_body.up.sql migrations/000011_add_store_body.down.sql
```

Then stage the three new up files:

```bash
git add migrations/000001_core.up.sql migrations/000002_mails.up.sql migrations/000003_audit.up.sql
```

- [ ] **Step 2.5: Run `go build ./...` to confirm `embed.FS` still works**

Run: `go build ./...`
Expected: build succeeds with no output. (The `//go:embed *.sql` directive has at least one match — the new files — so it compiles.)

If it fails with `no matching files found`, one of the three new files is missing or misnamed; fix and rerun.

- [ ] **Step 2.6: Commit the staged changes as a scratch commit for verification**

The verification script reads migrations via `git archive`, which only sees committed content. So we commit first with a placeholder message, verify, then amend the message (and add down files in Task 3 via another amend).

```bash
git commit -m "WIP: migration squash (to be amended)"
```

- [ ] **Step 2.7: Run the verification script — diff MUST be empty**

Run: `./scripts/verify-migration-squash.sh main HEAD`
Expected: `>>> OK: schemas match` and exit 0.

If the diff is non-empty, the output identifies exactly what differs (column order, missing index, default value, missing type, etc.). Edit the offending `.up.sql` file, run `git commit --amend --no-edit -a`, and re-run the verification script. Iterate until the diff is empty.

**Do not move on to Task 3 until this script returns exit 0.**

---

## Task 3: Write the 3 new down migrations

**Files:**
- Create: `migrations/000001_core.down.sql`
- Create: `migrations/000002_mails.down.sql`
- Create: `migrations/000003_audit.down.sql`

Down migrations are a dev-only convenience (not supported for production rollback). Each drops objects in reverse dependency order with `IF EXISTS` for idempotency.

- [ ] **Step 3.1: Create `migrations/000003_audit.down.sql`**

```sql
-- Dev-only rollback. N'utilisez pas en production.
DROP TABLE IF EXISTS audit_logs_default;
DROP TABLE IF EXISTS audit_logs;
```

- [ ] **Step 3.2: Create `migrations/000002_mails.down.sql`**

```sql
-- Dev-only rollback. N'utilisez pas en production.
DROP TABLE IF EXISTS mail_recipients_archive;
DROP TABLE IF EXISTS mails_archive_default;
DROP TABLE IF EXISTS mails_archive;
DROP TABLE IF EXISTS mail_recipients;
DROP TABLE IF EXISTS mails;
DROP TYPE IF EXISTS recipient_type;
DROP TYPE IF EXISTS mail_priority;
DROP TYPE IF EXISTS mail_status;
```

- [ ] **Step 3.3: Create `migrations/000001_core.down.sql`**

```sql
-- Dev-only rollback. N'utilisez pas en production.
DROP TABLE IF EXISTS app_branding;
DROP TABLE IF EXISTS mail_templates;
DROP TABLE IF EXISTS smtp_configs;
DROP TABLE IF EXISTS tenants;
-- L'extension uuid-ossp est laissée en place (partagée).
```

- [ ] **Step 3.4: Test the full up→down→up cycle locally**

Spin up a throwaway Postgres and run the cycle:

```bash
docker run --rm -d --name pg-down-test -e POSTGRES_PASSWORD=test -p 54323:5432 postgres:18-alpine
# Wait for readiness
until docker exec pg-down-test pg_isready -U postgres >/dev/null 2>&1; do sleep 1; done

DSN="postgres://postgres:test@localhost:54323/postgres?sslmode=disable"

migrate -source file://migrations -database "$DSN" up
migrate -source file://migrations -database "$DSN" down -all
migrate -source file://migrations -database "$DSN" up

docker rm -f pg-down-test
```

Expected: all three `migrate` commands exit 0 with no error. The final `up` re-creates every table cleanly on an empty DB.

If any step errors, the `.down.sql` files are dropping objects in the wrong order — fix the order and retry.

- [ ] **Step 3.5: Re-run full verification**

Run: `./scripts/verify-migration-squash.sh main HEAD`
Expected: `>>> OK: schemas match`, exit 0. (Down files don't affect the up-only schema diff, but run it again to be safe.)

- [ ] **Step 3.6: Amend the scratch commit to include down files and rewrite its message**

Add the three down files to the existing scratch commit (the one currently titled `WIP: migration squash (to be amended)`) and replace the message with the final wording. This yields one clean commit covering the entire swap (up + down + deletions).

```bash
git add migrations/000001_core.down.sql migrations/000002_mails.down.sql migrations/000003_audit.down.sql
git commit --amend -m "$(cat <<'EOF'
feat(db): remplacer 11 migrations par 3 fichiers de base v1

Squash de l'historique de développement en trois fichiers thématiques
pour la release v1 :
  - 000001_core   : tenants, smtp_configs, mail_templates, app_branding
  - 000002_mails  : ENUMs + mails, recipients, archives
  - 000003_audit  : audit_logs (partitionnée)

Les fichiers .down.sql sont fournis pour la commodité en
développement uniquement (non supportés pour la production).

Schéma vérifié byte-for-byte équivalent via
scripts/verify-migration-squash.sh.

BREAKING CHANGE (dev) : chaque développeur doit supprimer et
recréer sa base locale après ce commit.
EOF
)"
```

---

## Task 4: Run the full Go test suite

**Files:** none (validation only)

The existing test suite runs migrations against its own test database; if any test depends on a specific migration's shape (e.g., column presence), we'll see it here.

- [ ] **Step 4.1: Run unit + integration tests**

Run: `make test`
Expected: all tests pass. Exit code 0.

If any test fails, read the failure message carefully:
- If it's about a missing column / table → the squash dropped something accidentally; compare your `.up.sql` against the 11 originals and fix.
- If it's about a wrong default / index → re-run verification script; the diff will show what's off.
- If it's unrelated (e.g., flaky network test) → retry once; if still failing, flag it but don't hide it.

Do **not** modify tests to make them pass — the schema is wrong if tests fail, not the other way around.

- [ ] **Step 4.2: Run the linter**

Run: `make lint`
Expected: no lint errors. (No Go files changed in this plan, so this should be clean.)

- [ ] **Step 4.3: If Task 4 required any fix to the `.up.sql` files, amend Task 2's commit**

Only if you actually had to change a `.up.sql` to make tests pass:

```bash
git add migrations/*.up.sql
git commit --amend --no-edit
./scripts/verify-migration-squash.sh main HEAD   # must still be OK
make test                                          # must still be OK
```

If no fix was needed, skip this step.

---

## Task 5: Final state check and PR handoff

**Files:** none (verification only)

- [ ] **Step 5.1: Confirm the final tree state**

Run: `ls migrations/`
Expected, exactly these files (in any order):
```
000001_core.down.sql
000001_core.up.sql
000002_mails.down.sql
000002_mails.up.sql
000003_audit.down.sql
000003_audit.up.sql
embed.go
```

- [ ] **Step 5.2: Confirm the branch contains exactly two commits over `main`**

Run: `git log --oneline main..HEAD`
Expected, exactly two lines (most recent first):
1. `feat(db): remplacer 11 migrations par 3 fichiers de base v1`
2. `chore(db): ajouter le script de vérification du squash des migrations`

The verification-script commit (Task 1) is the older of the two; the migration-swap commit (Task 2 + Task 3 amend) sits on top. If you see three or more commits, an amend was missed — squash them manually with `git rebase -i main`.

- [ ] **Step 5.3: Run final full verification end-to-end**

```bash
./scripts/verify-migration-squash.sh main HEAD && make test && make lint
```

Expected: all three succeed, exit 0.

- [ ] **Step 5.4: Report back to the user**

Summarize: branch name, number of commits, verification command to re-run, and the explicit dev-team instruction to drop local databases after merge.

**Do NOT open a PR unless the user explicitly asks.** The plan ends at a clean, verified branch ready for the user's review.
