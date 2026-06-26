#!/usr/bin/env bash
set -euo pipefail

backup_dir="${1:?usage: calligraphy-restore.sh <backup-dir>}"

restore_if_set() {
  local env_name="$1"
  local source_name="$2"
  local target_path="${!env_name:-}"
  local source_path="$backup_dir/$source_name"
  if [[ -n "$target_path" && -f "$source_path" ]]; then
    mkdir -p "$(dirname "$target_path")"
    cp "$source_path" "$target_path"
  fi
}

restore_if_set CALLIGRAPHY_AUTH_FILE auth.json
restore_if_set CALLIGRAPHY_DATA_FILE drafts.json
restore_if_set CALLIGRAPHY_LEARNING_FILE learning.json
restore_if_set CALLIGRAPHY_AUDIT_FILE audit.jsonl

if [[ -n "${CALLIGRAPHY_EXPORT_DIR:-}" && -d "$backup_dir/artifacts" ]]; then
  mkdir -p "$CALLIGRAPHY_EXPORT_DIR"
  cp -a "$backup_dir/artifacts/." "$CALLIGRAPHY_EXPORT_DIR/"
fi

printf 'backup restored from %s\n' "$backup_dir"
