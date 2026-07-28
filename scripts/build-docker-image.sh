#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
api_base_url="${CALLIGRAPHY_RELEASE_API_BASE_URL:-}"
image="${CALLIGRAPHY_DOCKER_IMAGE:-nebula-calligraphy:local}"

(
  cd "$repo_root/apps/mobile"
  flutter build web --release --dart-define="CALLIGRAPHY_API_BASE_URL=$api_base_url"
)

docker build -f "$repo_root/deploy/docker/Dockerfile" -t "$image" "$repo_root"
