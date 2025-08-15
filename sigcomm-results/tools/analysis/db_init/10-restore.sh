#!/usr/bin/env bash
set -euo pipefail

DB_NAME="${POSTGRES_DB:-original_filtered}"
DB_USER="${POSTGRES_USER:-postgres}"
DUMP_FILE="/dump/selected.dump"

echo "Initializing database '$DB_NAME'..."

psql -v ON_ERROR_STOP=1 --username "$DB_USER" --dbname "$DB_NAME" <<'SQL'
CREATE EXTENSION IF NOT EXISTS timescaledb;
SQL

if [[ -f "$DUMP_FILE" ]]; then
  echo "Restoring dump from $DUMP_FILE into '$DB_NAME'..."
  pg_restore \
    --verbose \
    --no-owner --no-privileges \
    --role="$DB_USER" \
    --dbname="$DB_NAME" \
    "$DUMP_FILE"
  echo "Restore complete."
else
  echo "No dump file found at $DUMP_FILE, skipping restore."
fi