# Squash Migrations for v1 Release — Design

- **Date:** 2026-04-18
- **Status:** Approved (pending spec review)
- **Scope:** Collapse the current 11 incremental migrations into 3 logically-grouped baseline migrations for the v1 release of MailHive.

## 1. Goal

MailHive is preparing its v1 release. No production database has been deployed, so the migration history has no value to preserve. The 11 migrations that accrued during development (mix of `CREATE TABLE` and incremental `ALTER`s) should be replaced with a small, reviewable set of baseline migrations that reflect the final schema exactly.

## 2. Non-goals

- **No schema changes.** The post-squash schema must be byte-for-byte equivalent (per `pg_dump --schema-only`) to the current post-migration schema. Any divergence is a bug.
- **No changes to runtime code.** `internal/adapter/postgres/migrate.go` (`RunMigrations`, `RollbackMigrations`) and `migrations/embed.go` (`//go:embed *.sql`) stay untouched.
- **No test changes.** The existing Go test suite is the behavioral check.
- **No migration-tooling migration.** Still `golang-migrate/v4` via `iofs`.

## 3. Current state

`migrations/` holds:

```
000001_create_tenants.up.sql          / .down.sql
000002_create_smtp_configs.up.sql     / .down.sql
000003_create_mail_templates.up.sql   / .down.sql
000004_create_mails.up.sql            / .down.sql
000005_create_mail_recipients.up.sql  / .down.sql
000006_create_app_branding.up.sql     / .down.sql
000007_create_audit_logs.up.sql       / .down.sql
000008_create_mails_archive.up.sql    / .down.sql
000009_add_spam_score_and_tags.up.sql / .down.sql
000010_add_tenant_language.up.sql     / .down.sql
000011_add_store_body.up.sql          / .down.sql
embed.go
```

Run via `iofs.New(migrations, ".")` in `internal/adapter/postgres/migrate.go`.

## 4. Target file layout

```
migrations/
├── embed.go                   (unchanged: //go:embed *.sql)
├── 000001_core.up.sql         tenants, smtp_configs, mail_templates, app_branding
├── 000001_core.down.sql
├── 000002_mails.up.sql        ENUMs + mails + recipients + archive tables
├── 000002_mails.down.sql
├── 000003_audit.up.sql        audit_logs (partitioned) + audit_logs_default
└── 000003_audit.down.sql
```

The old 11 `.up.sql` / `.down.sql` files are deleted. Git history preserves them.

## 5. Content consolidation

### 5.1 `000001_core.up.sql`

- `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";` (carried from original `000001`).
- `tenants` table — identical to original `000001`, including the `settings` JSONB default:

  ```json
  {"rate_limit": 10, "rate_burst": 20, "max_destinataires": 1000}
  ```

  **Do not** bake `language` or `store_body` into this default. The original `000010` / `000011` migrations only `UPDATE`d existing rows; they did not change the column default. On a fresh DB (no prior rows) those `UPDATE`s are no-ops, so the post-squash schema must leave the default alone to pass the `pg_dump` diff check. The `UPDATE tenants SET settings = ...` statements from `000010` / `000011` are simply dropped from the squash (they had no effect on a fresh DB). Application code already handles missing `language` / `store_body` keys via Go zero-values.
- `idx_tenants_slug`, `idx_tenants_api_key`, `idx_tenants_is_active` (verbatim).
- `smtp_configs` table — final shape, including `charset` and `encoding` columns. `idx_smtp_configs_tenant_id` and partial unique index `idx_smtp_configs_default_per_tenant`.
- `mail_templates` table — verbatim, with its `idx_mail_templates_tenant_id` index and `UNIQUE(tenant_id, slug)` constraint.
- `app_branding` table — includes `timezone VARCHAR(50) NOT NULL DEFAULT 'Europe/Paris'`. Seed row `INSERT INTO app_branding (id) VALUES (1) ON CONFLICT DO NOTHING;` retained.

### 5.2 `000002_mails.up.sql`

- ENUM types first (must exist before tables reference them):
  - `mail_status` (`pending`, `queued`, `sending`, `sent`, `failed`, `cancelled`)
  - `mail_priority` (`critical`, `default`, `low`)
  - `recipient_type` (`to`, `cc`, `bcc`)
- `mails` table — final shape with `spam_score REAL`, `tags TEXT[] NOT NULL DEFAULT '{}'`, and `compressed_body BYTEA` included from the start (no follow-up `ALTER`).
- Indexes: `idx_mails_tenant_status_created`, `idx_mails_created_at`, `idx_mails_tags` (GIN).
- `mail_recipients` table + `idx_mail_recipients_mail_id`.
- `mails_archive` partitioned table — includes `spam_score`, `tags`, `compressed_body` from the start. Plus `mails_archive_default` catch-all partition.
- `idx_mails_archive_tenant_created`.
- `mail_recipients_archive` table + `idx_mail_recipients_archive_mail_id`.

### 5.3 `000003_audit.up.sql`

- `audit_logs` partitioned table, verbatim from original `000007`.
- `audit_logs_default` catch-all partition.
- `idx_audit_logs_tenant_created`, `idx_audit_logs_status`.

### 5.4 Down migrations

Each down file drops the objects created by its corresponding up, in reverse dependency order, using `IF EXISTS` for idempotency.

- `000003_audit.down.sql`: `DROP TABLE IF EXISTS audit_logs_default;` then `DROP TABLE IF EXISTS audit_logs;`.
- `000002_mails.down.sql`: drop archive recipient + archive mail (+ default partition) tables, then `mail_recipients`, then `mails`, then the three ENUM types.
- `000001_core.down.sql`: drop `app_branding`, `mail_templates`, `smtp_configs`, `tenants` (cascades handle any FK remnants).

Down migrations are a **dev-only convenience**. They are *not* supported for production rollback. This stance is documented in the file header of each down migration.

## 6. Verification

Before the squash PR merges, the following must pass:

1. **Schema-diff check (`scripts/verify-migration-squash.sh`)**
   - Spin up two throwaway Postgres 18 containers: `pg_old`, `pg_new`.
   - Against `pg_old`: check out `main`, run `RunMigrations` (the 11 legacy files).
   - Against `pg_new`: check out the squash branch, run `RunMigrations` (the 3 new files).
   - `pg_dump --schema-only --no-owner --no-privileges` each database.
   - `diff` the two dumps. **Expected: empty diff. Script exits non-zero on any difference.**
   - Commit this script so reviewers can re-run it.
2. **Existing Go test suite (`make test`)** runs clean against the new schema. Any failure means something in the consolidation is off.

## 7. Rollout

- **Branch:** one branch, one PR titled `chore(db): squash migrations to 3 v1 baseline files` (or equivalent FR wording to match project conventions).
- **Commits on the branch:**
  1. Delete the old 11 up/down files; add the 3 new up/down files; update any doc references to specific old migration numbers.
  2. Add `scripts/verify-migration-squash.sh` + a short `README` note on how to run it.
- **Dev-team instructions (in PR description):** after merging, every developer must drop their local DB and let the new migrations recreate it. No automated upgrade path — pre-v1, nothing of value exists in local DBs.

## 8. Risks & mitigations

| Risk | Mitigation |
|------|------------|
| Subtle schema drift (column order, default, index name, partition clause) | `pg_dump` diff in verification script catches mechanically; reviewers re-run it. |
| Developer runs old migrations against new DB (or vice versa) | After merge, every dev drops their local DB. Covered in PR description + follow-up team message. |
| Someone later re-adds `000004` assuming no gaps | Next-free number is `000004`; the three baseline files are clearly named `_core`, `_mails`, `_audit` so there's no confusion about continuing from 4. |
| `RollbackMigrations` gets run against real data post-v1 | Down files explicitly marked "dev-only, not for production" in their header comment. Consider follow-up work (out of scope) to remove or guard the function once a prod DB exists. |

## 9. Open questions

None at time of writing. All decisions resolved during brainstorming:
- Fresh v1, no prod DB to preserve.
- 3 logical groups (core / mails / audit).
- Delete old files, drop/recreate dev DBs.
- Keep down migrations as dev-only convenience.
- Verify via `pg_dump` schema diff.
