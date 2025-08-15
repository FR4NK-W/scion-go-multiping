# #!/usr/bin/env bash
set -euo pipefail

DB_NAME="${POSTGRES_DB:-original_filtered}"
DB_USER="${POSTGRES_USER:-postgres}"
OUT_DIR="/exports"
QDIR="/docker-entrypoint-initdb.d/queries"

shopt -s nullglob
queries=("$QDIR"/*.sql)
if (( ${#queries[@]} == 0 )); then
  echo "No .sql files in $QDIR."
  exit 0
fi

echo "Exporting $DB_NAME query results to CSV at $OUT_DIR"

for f in "${queries[@]}"; do
  base="$(basename "$f" .sql)"
  out="$OUT_DIR/${base}.csv"

  cleaned_sql="$(
    sed -E 's/[[:space:]]+$//' "$f" \
    | sed -E 's/--.*$//' \
    | sed -E ':a;N;$!ba;s@/\*[^*]*\*+([^/*][^*]*\*+)*/@@g' \
    | tr '\n' ' ' \
    | sed -E 's/;[[:space:]]*$//'
    )"
    
  echo " -> $base.sql  →  $(basename "$out")"

  psql -v ON_ERROR_STOP=1 --username "$DB_USER" --dbname "$DB_NAME" \
    -c "\copy ( $cleaned_sql ) TO '$out' CSV HEADER;"
done

echo "All exports completed."