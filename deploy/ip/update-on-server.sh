#!/usr/bin/env bash
set -euo pipefail
ROOT=/data/nebula-calligraphy
PRODUCTION_MANIFEST="$ROOT/assets/copybooks/production/manifest.json"
mkdir -p "$ROOT/bin" "$ROOT/web" "$ROOT/assets" "$ROOT/state/artifacts" "$ROOT/state/backups" "$ROOT/state/render-cache"
install -m 0755 bin/calligraphy "$ROOT/bin/calligraphy"
install -m 0755 bin/calligraphy-glyph-manifest "$ROOT/bin/calligraphy-glyph-manifest"
rsync -a --delete web/ "$ROOT/web/"
rsync -a assets/ "$ROOT/assets/"
if [[ ! -f "$PRODUCTION_MANIFEST" ]]; then
  printf '缺少生产字形清单: %s\n' "$PRODUCTION_MANIFEST" >&2
  printf '请先把已授权、已审校且包含 published 字形的 manifest.json 放入该路径。\n' >&2
  exit 1
fi
"$ROOT/bin/calligraphy-glyph-manifest" validate "$PRODUCTION_MANIFEST"
sudo install -d -m 0755 /etc/systemd/system/nebula-calligraphy.service.d
sudo install -m 0644 deploy/nebula-calligraphy.service /etc/systemd/system/nebula-calligraphy.service
sudo tee /etc/systemd/system/nebula-calligraphy.service.d/20-render.conf >/dev/null <<EOF
[Service]
Environment=CALLIGRAPHY_RENDER_FONT_FILE=$ROOT/assets/fonts/MaShanZheng-Regular.ttf
Environment=CALLIGRAPHY_RENDER_CACHE_DIR=$ROOT/state/render-cache
Environment=CALLIGRAPHY_LEARNING_TIMEZONE=Asia/Shanghai
EOF
sudo install -m 0644 deploy/calligraphy-ip.nginx.conf /etc/nginx/conf.d/calligraphy-ip.conf
sudo nginx -t
sudo systemctl daemon-reload
sudo systemctl enable nebula-calligraphy
sudo systemctl restart nebula-calligraphy
sudo systemctl restart nginx
curl -fsS http://127.0.0.1:8090/ready
curl -fsS http://127.0.0.1:8090/api/v1/calligraphy/glyphs/search | grep -q '"glyph_id"'
