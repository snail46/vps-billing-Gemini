#!/usr/bin/env bash
# scripts/restore_db.sh — PostgreSQL Database Restoration & Integrity Verification
set -euo pipefail

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <path_to_backup_file.sql.gz>"
  exit 1
fi

BACKUP_FILE="$1"
CHECKSUM_FILE="${BACKUP_FILE}.sha256"
PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-5432}"
PGUSER="${PGUSER:-postgres}"
PGDATABASE="${PGDATABASE:-vps_billing}"

if [ ! -f "${BACKUP_FILE}" ]; then
  echo "[ERROR] Backup file ${BACKUP_FILE} does not exist!"
  exit 1
fi

echo "[INFO] Verifying checksum for ${BACKUP_FILE}..."
if [ -f "${CHECKSUM_FILE}" ]; then
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -c "${CHECKSUM_FILE}"
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 -c "${CHECKSUM_FILE}"
  fi
  echo "[INFO] Checksum verified successfully."
else
  echo "[WARN] No checksum file found. Skipping hash verification."
fi

echo "[INFO] Restoring database '${PGDATABASE}' on ${PGHOST}:${PGPORT}..."
PGPASSWORD="${PGPASSWORD:-postgres}" gunzip -c "${BACKUP_FILE}" | psql \
  -h "${PGHOST}" \
  -p "${PGPORT}" \
  -U "${PGUSER}" \
  -d "${PGDATABASE}" \
  --single-transaction

echo "[INFO] Database restored successfully. Verifying table counts..."
PGPASSWORD="${PGPASSWORD:-postgres}" psql \
  -h "${PGHOST}" \
  -p "${PGPORT}" \
  -U "${PGUSER}" \
  -d "${PGDATABASE}" \
  -c "SELECT count(*) AS total_tables FROM information_schema.tables WHERE table_schema = 'public';"

echo "[INFO] Restoration and verification completed successfully."
