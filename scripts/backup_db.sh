#!/usr/bin/env bash
# scripts/backup_db.sh — PostgreSQL Automated Backup & Verification Script
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-./backups}"
PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-5432}"
PGUSER="${PGUSER:-postgres}"
PGDATABASE="${PGDATABASE:-vps_billing}"
TIMESTAMP=$(date -u +"%Y%m%d_%H%M%SZ")
BACKUP_FILE="${BACKUP_DIR}/vps_billing_backup_${TIMESTAMP}.sql.gz"
CHECKSUM_FILE="${BACKUP_FILE}.sha256"

mkdir -p "${BACKUP_DIR}"

echo "[INFO] Starting database backup for '${PGDATABASE}' on ${PGHOST}:${PGPORT} at ${TIMESTAMP}..."

# Execute pg_dump with custom compressed plain SQL format
PGPASSWORD="${PGPASSWORD:-postgres}" pg_dump \
  -h "${PGHOST}" \
  -p "${PGPORT}" \
  -U "${PGUSER}" \
  -d "${PGDATABASE}" \
  --clean --if-exists --no-owner --no-privileges | gzip -9 > "${BACKUP_FILE}"

echo "[INFO] Backup written to ${BACKUP_FILE}"

# Generate SHA256 checksum for integrity verification
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "${BACKUP_FILE}" > "${CHECKSUM_FILE}"
elif command -v shasum >/dev/null 2>&1; then
  shasum -a 256 "${BACKUP_FILE}" > "${CHECKSUM_FILE}"
fi

echo "[INFO] Checksum generated at ${CHECKSUM_FILE}"
echo "[INFO] Backup completed successfully."
