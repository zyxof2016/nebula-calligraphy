#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
api_base_url="${CALLIGRAPHY_RELEASE_API_BASE_URL:-}"
output_dir="${CALLIGRAPHY_RELEASE_OUTPUT_DIR:-$repo_root/dist}"
archive_name="${CALLIGRAPHY_RELEASE_ARCHIVE_NAME:-nebula-calligraphy-linux-amd64.tar.gz}"
staging_root="$(mktemp -d "${TMPDIR:-/tmp}/nebula-calligraphy-release.XXXXXX")"
release_dir="$staging_root/calligraphy-ip-release"

cleanup() {
  rm -rf -- "$staging_root"
}
trap cleanup EXIT

mkdir -p "$release_dir/bin" "$release_dir/web" "$release_dir/assets/fonts" "$release_dir/assets/copybooks/jiuchenggong" "$release_dir/deploy"

(
  cd "$repo_root/apps/mobile"
  flutter build web --release --dart-define="CALLIGRAPHY_API_BASE_URL=$api_base_url"
)

(
  cd "$repo_root/services/calligraphy"
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "$release_dir/bin/calligraphy" ./cmd/calligraphy
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o "$release_dir/bin/calligraphy-glyph-manifest" ./cmd/calligraphy-glyph-manifest
)

rsync -a "$repo_root/apps/mobile/build/web/" "$release_dir/web/"
install -m 0644 "$repo_root/assets/fonts/MaShanZheng-Regular.ttf" "$release_dir/assets/fonts/"
install -m 0644 "$repo_root/assets/fonts/MaShanZheng-OFL.txt" "$release_dir/assets/fonts/"
install -m 0644 "$repo_root/assets/copybooks/jiuchenggong/manifest.sample.json" "$release_dir/assets/copybooks/jiuchenggong/"
install -m 0644 "$repo_root/assets/copybooks/jiuchenggong/README.md" "$release_dir/assets/copybooks/jiuchenggong/"
install -m 0755 "$repo_root/deploy/ip/update-on-server.sh" "$release_dir/deploy/"
install -m 0644 "$repo_root/deploy/ip/calligraphy-ip.nginx.conf" "$release_dir/deploy/"
install -m 0644 "$repo_root/deploy/ip/nebula-calligraphy.service" "$release_dir/deploy/"
install -m 0644 "$repo_root/deploy/ip/README.deploy.md" "$release_dir/README.deploy.md"

mkdir -p "$output_dir"
tar -C "$staging_root" -czf "$output_dir/$archive_name" calligraphy-ip-release
sha256sum "$output_dir/$archive_name" >"$output_dir/$archive_name.sha256"
printf 'release: %s\n' "$output_dir/$archive_name"
printf 'sha256: %s\n' "$output_dir/$archive_name.sha256"
