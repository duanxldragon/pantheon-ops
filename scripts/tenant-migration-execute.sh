#!/bin/bash
# Tenant Migration Execution Script
# Date: 2026-10-09
# Purpose: Execute tenant migrations against the configured database.

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/.." && pwd)"
BACKEND_DIR="$REPO_ROOT/backend"
MIGRATE=(go run ./cmd/tenantmigration)

printf '%s\n' '==========================================' 'Pantheon Base - Tenant Migration Execution' "Date: $(date '+%Y-%m-%d %H:%M:%S')" '=========================================='

require_dsn() {
  if [[ -z "${PANTHEON_DSN:-}" ]]; then
    echo 'PANTHEON_DSN is required' >&2
    return 1
  fi
}

check_mysql() {
  echo '[1/6] Checking MySQL connection...'
  require_dsn
  (cd "$BACKEND_DIR" && PANTHEON_DSN="$PANTHEON_DSN" "${MIGRATE[@]}" status)
}

check_status() {
  echo '[2/6] Checking current migration status...'
  (cd "$BACKEND_DIR" && PANTHEON_DSN="$PANTHEON_DSN" "${MIGRATE[@]}" status)
}

backup_schema() {
  echo '[3/6] Verifying pre-migration backup...'
  if [[ -z "${PANTHEON_BACKUP_FILE:-}" || ! -s "$PANTHEON_BACKUP_FILE" ]]; then
    echo 'Set PANTHEON_BACKUP_FILE to a non-empty mysqldump created before running migrations.' >&2
    return 1
  fi
  echo "Backup verified: $PANTHEON_BACKUP_FILE"
}

execute_migrations() {
  echo '[4/6] Executing tenant migrations...'
  (cd "$BACKEND_DIR" && PANTHEON_DSN="$PANTHEON_DSN" "${MIGRATE[@]}" up)
}

verify_migrations() {
  echo '[5/6] Verifying migration results...'
  (cd "$BACKEND_DIR" && PANTHEON_DSN="$PANTHEON_DSN" "${MIGRATE[@]}" status)
}

smoke_test() {
  echo '[6/6] Running post-migration smoke tests...'
  (cd "$BACKEND_DIR" && go test -short ./...)
}

main() {
  require_dsn
  check_mysql
  check_status
  backup_schema
  execute_migrations
  verify_migrations
  smoke_test
  echo
  echo 'Tenant migration completed successfully.'
  echo 'Rollback: stop the application and restore PANTHEON_BACKUP_FILE with mysql; do not use a nonexistent migrate down command.'
}

main "$@"
