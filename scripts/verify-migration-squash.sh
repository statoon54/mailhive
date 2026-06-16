#!/usr/bin/env bash
# Verifies that the migration set at NEW_REF produces a schema
# byte-for-byte identical to the set at OLD_REF.
#
# Usage: scripts/verify-migration-squash.sh [OLD_REF] [NEW_REF]
#   Defaults: OLD_REF=main, NEW_REF=HEAD
#
# Requires: docker, migrate CLI (golang-migrate v4), git.
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
    local container=$1 out=$2
    # pg_dump is run inside the container: host pg_dump v17 refuses a PG 18 server.
    # grep -v strips PG 18's per-dump random \restrict/\unrestrict nonces,
    # which would otherwise cause identical schemas to diff.
    # '|| true' guards against grep exiting 1 if every line were filtered
    # (degenerate case; pg_dump always emits header lines in practice).
    docker exec -e PGPASSWORD=test "$container" pg_dump \
        --schema-only \
        --no-owner \
        --no-privileges \
        --exclude-table=schema_migrations \
        -U postgres postgres \
    | { grep -v '^\\\(restrict\|unrestrict\)' || true; } > "$out"
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
dump_schema "$OLD_CONTAINER" "$OLD_DUMP"
dump_schema "$NEW_CONTAINER" "$NEW_DUMP"

echo ">>> Diff (empty means success):"
if diff -u "$OLD_DUMP" "$NEW_DUMP"; then
    echo ">>> OK: schemas match"
    exit 0
else
    echo ">>> FAIL: schemas differ (see diff above)"
    exit 1
fi
