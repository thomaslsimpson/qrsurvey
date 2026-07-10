#!/usr/bin/env bash
# Snapshot the SQLite database. Meant to run from cron, e.g. daily:
#   0 3 * * * /opt/qrsurvey/deploy/backup.sh
set -euo pipefail

DB_PATH="${DB_PATH:-/opt/qrsurvey/data/qrsurvey.db}"
BACKUP_DIR="${BACKUP_DIR:-/opt/qrsurvey/backups}"

mkdir -p "$BACKUP_DIR"
dest="$BACKUP_DIR/qrsurvey-$(date -u +%Y-%m-%dT%H-%M-%SZ).db"
sqlite3 "$DB_PATH" ".backup '$dest'"

# keep the last 30 backups
ls -1t "$BACKUP_DIR"/qrsurvey-*.db 2>/dev/null | tail -n +31 | xargs -r rm --
