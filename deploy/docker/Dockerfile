FROM golang:1.25.12-alpine@sha256:56961d79ea8129efddcc0b8643fd8a5416b4e6228cfd477e3fd61deb2672c587 AS backend

WORKDIR /src
COPY services/calligraphy/go.mod services/calligraphy/go.sum ./
RUN go mod download
COPY services/calligraphy/ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/calligraphy ./cmd/calligraphy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/calligraphy-migrate ./cmd/calligraphy-migrate

FROM alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 calligraphy \
    && adduser -S -D -H -u 10001 -G calligraphy calligraphy \
    && install -d -o calligraphy -g calligraphy /app/web /app/assets/fonts /data/artifacts /data/render-cache /data/glyphs

COPY --from=backend /out/calligraphy /app/calligraphy
COPY --from=backend /out/calligraphy-migrate /app/calligraphy-migrate
COPY apps/mobile/build/web/ /app/web/
COPY assets/fonts/MaShanZheng-Regular.ttf /app/assets/fonts/

ENV PORT=8090 \
    CALLIGRAPHY_RUNTIME_PROFILE=production \
    CALLIGRAPHY_AUTH_FILE=/data/auth.json \
    CALLIGRAPHY_DATA_FILE=/data/drafts.json \
    CALLIGRAPHY_LEARNING_FILE=/data/learning.json \
    CALLIGRAPHY_AUDIT_FILE=/data/audit.jsonl \
    CALLIGRAPHY_EXPORT_DIR=/data/artifacts \
    CALLIGRAPHY_WEB_DIR=/app/web \
    CALLIGRAPHY_GLYPH_MANIFEST_FILE=/data/glyphs/manifest.json \
    CALLIGRAPHY_RENDER_FONT_FILE=/app/assets/fonts/MaShanZheng-Regular.ttf \
    CALLIGRAPHY_RENDER_CACHE_DIR=/data/render-cache \
    CALLIGRAPHY_LEARNING_TIMEZONE=Asia/Shanghai \
    CALLIGRAPHY_SESSION_TTL=24h

USER calligraphy
EXPOSE 8090
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD wget -q -O /dev/null http://127.0.0.1:8090/ready || exit 1
ENTRYPOINT ["/app/calligraphy"]
