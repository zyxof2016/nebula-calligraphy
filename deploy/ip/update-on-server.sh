#!/usr/bin/env bash
set -euo pipefail
ROOT=/data/nebula-calligraphy
mkdir -p "$ROOT/bin" "$ROOT/web" "$ROOT/assets" "$ROOT/state/artifacts" "$ROOT/state/backups"
install -m 0755 bin/calligraphy "$ROOT/bin/calligraphy"
install -m 0755 bin/calligraphy-glyph-manifest "$ROOT/bin/calligraphy-glyph-manifest"
rsync -a --delete web/ "$ROOT/web/"
rsync -a assets/ "$ROOT/assets/"
"$ROOT/bin/calligraphy-glyph-manifest" validate "$ROOT/assets/copybooks/jiuchenggong/manifest.sample.json"
sudo install -m 0644 deploy/calligraphy-ip.nginx.conf /etc/nginx/conf.d/calligraphy-ip.conf
sudo nginx -t
sudo systemctl restart nebula-calligraphy
sudo systemctl restart nginx
curl -fsS http://127.0.0.1:8090/ready
curl -fsS http://127.0.0.1:8090/api/v1/calligraphy/glyphs/search?character=山\&style=ou
