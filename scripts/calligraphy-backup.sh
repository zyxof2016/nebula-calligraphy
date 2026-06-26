#!/usr/bin/env bash
set -euo pipefail

backup_dir="${1:?usage: calligraphy-backup.sh <backup-dir>}"

mkdir -p "$backup_dir"

copy_if_set() {
  local env_name="$1"
  local target_name="$2"
  local source_path="${!env_name:-}"
  if [[ -n "$source_path" && -f "$source_path" ]]; then
    cp "$source_path" "$backup_dir/$target_name"
  fi
}

copy_if_set CALLIGRAPHY_AUTH_FILE auth.json
copy_if_set CALLIGRAPHY_DATA_FILE drafts.json
copy_if_set CALLIGRAPHY_LEARNING_FILE learning.json
copy_if_set CALLIGRAPHY_AUDIT_FILE audit.jsonl

if [[ -n "${CALLIGRAPHY_EXPORT_DIR:-}" && -d "$CALLIGRAPHY_EXPORT_DIR" ]]; then
  mkdir -p "$backup_dir/artifacts"
  cp -a "$CALLIGRAPHY_EXPORT_DIR/." "$backup_dir/artifacts/"
fi

printf 'backup written to %s\n' "$backup_dir"
