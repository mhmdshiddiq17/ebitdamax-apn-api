#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root_dir"

[ -f .env ] || { echo ".env tidak ditemukan" >&2; exit 1; }

env_value() {
  awk -F= -v key="$1" '$1 == key { sub(/^[^=]*=/, ""); print; exit }' .env
}

rehearsal_db=${REHEARSAL_DB_NAME:-ebitdamax_apn_rehearsal}
if [ "$rehearsal_db" != "ebitdamax_apn_rehearsal" ]; then
  echo "REHEARSAL_DB_NAME harus ebitdamax_apn_rehearsal" >&2
  exit 1
fi

reset=false
if [ "${1:-}" = "--reset" ]; then
  reset=true
elif [ "$#" -ne 0 ]; then
  echo "penggunaan: scripts/s9-rehearsal.sh [--reset]" >&2
  exit 1
fi

postgres_user=${POSTGRES_USER:-$(env_value POSTGRES_USER)}
postgres_password=${POSTGRES_PASSWORD:-$(env_value POSTGRES_PASSWORD)}
postgres_db=${POSTGRES_DB:-$(env_value POSTGRES_DB)}
postgres_port=${POSTGRES_PORT:-$(env_value POSTGRES_PORT)}
postgres_user=${postgres_user:-ebitdamax}
postgres_db=${postgres_db:-ebitdamax_apn}
postgres_port=${postgres_port:-5433}

postgres() {
  PGPASSWORD="$postgres_password" psql -h 127.0.0.1 -p "$postgres_port" -U "$postgres_user" "$@"
}

docker compose up -d postgres redis minio

for _ in $(seq 1 30); do
  if postgres -d "$postgres_db" -Atqc 'SELECT 1' >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
postgres -d "$postgres_db" -Atqc 'SELECT 1' >/dev/null

if postgres -d postgres -Atqc "SELECT 1 FROM pg_database WHERE datname = '$rehearsal_db'" | rg -qx '1'; then
  if [ "$reset" != true ]; then
    echo "database rehearsal sudah ada; gunakan --reset untuk backup lalu membangunnya ulang" >&2
    exit 1
  fi

  backup_dir="tmp/s9-rehearsal/$(date +%Y%m%dT%H%M%S)"
  mkdir -p "$backup_dir"
  PGPASSWORD="$postgres_password" pg_dump -h 127.0.0.1 -p "$postgres_port" -U "$postgres_user" -Fc "$rehearsal_db" >"$backup_dir/$rehearsal_db.dump"
  PGPASSWORD="$postgres_password" dropdb -h 127.0.0.1 -p "$postgres_port" -U "$postgres_user" --if-exists "$rehearsal_db"
  echo "backup rehearsal: $backup_dir/$rehearsal_db.dump"
fi

PGPASSWORD="$postgres_password" createdb -h 127.0.0.1 -p "$postgres_port" -U "$postgres_user" "$rehearsal_db"
awk 'index($0, "-- +goose Down") { exit } { print }' migrations/00001_initial_schema.sql | postgres -d "$rehearsal_db" -q -v ON_ERROR_STOP=1 >/dev/null
awk 'index($0, "-- +goose Down") { exit } { print }' migrations/00002_customer_analyses.sql | postgres -d "$rehearsal_db" -q -v ON_ERROR_STOP=1 >/dev/null

DB_NAME="$rehearsal_db" SEED_MANAGER_ENABLED=false go run ./cmd/seed
DB_NAME="$rehearsal_db" go run ./cmd/migrate-legacy-kdkmp --apply --rehearsal --without-files

file_metadata=$(postgres -d "$rehearsal_db" -Atqc "SELECT count(*) FROM users WHERE manager_sk_document IS NOT NULL; SELECT count(*) FROM task_reports WHERE started_photo IS NOT NULL OR finished_photo IS NOT NULL OR started_documents IS NOT NULL OR finished_documents IS NOT NULL; SELECT count(*) FROM task_report_values trv JOIN task_additional_fields taf ON taf.id = trv.task_additional_field_id WHERE taf.input_type = 'file' AND NULLIF(BTRIM(trv.value), '') IS NOT NULL; SELECT count(*) FROM meeting_minute_attachments;")
if [ "$file_metadata" != $'0\n0\n0\n0' ]; then
  echo "metadata berkas rehearsal belum bersih: $file_metadata" >&2
  exit 1
fi

echo "rehearsal siap: DB_NAME=$rehearsal_db"
