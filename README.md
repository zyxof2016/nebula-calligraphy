# nebula-calligraphy

`nebula-calligraphy` is the independent C-side product line for AI-assisted Chinese calligraphy learning and copybook-based artwork composition.

It connects to the Nebula platform foundation for identity, storage, AI gateway, audit, and future billing, but it does not belong to the Signage scheduling/player/device lifecycle flow.

## Product Scope

MVP focuses on the core user loop:

1. Search a character.
2. Compare famous copybook glyphs.
3. Download a daily practice template.
4. Mark practice and review recent history.
5. Save favorite glyphs for repeated study.
6. Compose text into a calligraphy artwork draft.
7. Export SVG/PDF/PNG learning templates.

Deferred scope:

- AR tracing.
- Community feed.
- Teacher/classroom workflows.
- Full AI handwriting scoring.
- Personalized style training.

## Repository Layout

```text
nebula-calligraphy/
├── apps/mobile/              # Flutter C-side app
├── web/admin/                # React + Ant Design content/annotation admin
├── services/calligraphy/     # Go API service
├── pkg/layout/               # Layout algorithms
├── pkg/glyph/                # Glyph matching and metadata logic
├── pkg/render/               # PDF/PNG/SVG export helpers
├── docs/products/            # Product specs
├── docs/contracts/           # Machine-readable contracts
└── scripts/
```

## Foundation Dependencies

| Capability | Nebula dependency |
|------------|-------------------|
| User identity | Local MVP auth now; `nebula-platform` Identity for production |
| API gateway | `nebula-platform` Gateway |
| Object storage | S3/OSS/MinIO deployment profile |
| AI features | Nebula AI gateway/model adapter |
| Audit | `nebula-analytics-audit` event model |
| Open API | `nebula-open-platform` SDK and webhook pattern |

## MVP Runtime

The first runtime slice is the Go API service in `services/calligraphy`.

```bash
cd services/calligraphy
GOCACHE=/tmp/nebula-calligraphy-go-cache go test ./...
PORT=8090 go run ./cmd/calligraphy
```

For a restart-safe local trial, enable file-backed drafts and local SVG artifacts:

```bash
mkdir -p .local/artifacts
CALLIGRAPHY_AUTH_FILE=.local/auth.json \
CALLIGRAPHY_DATA_FILE=.local/drafts.json \
CALLIGRAPHY_LEARNING_FILE=.local/learning.json \
CALLIGRAPHY_AUDIT_FILE=.local/audit.jsonl \
CALLIGRAPHY_EXPORT_DIR=.local/artifacts \
CALLIGRAPHY_WEB_DIR=../../web/app \
PORT=8090 \
go run ./cmd/calligraphy
```

Then open `http://127.0.0.1:8090/` to use the trial workbench. The workbench covers local registration/login, preset common glyph groups, glyph lookup, glyph detail, practice template preview/download, favorite glyphs, practice records, learning profile, layout preview, draft save/list/load/delete, SVG export history, and SVG export/download.

Available MVP endpoints:

| Endpoint | Purpose |
|----------|---------|
| `GET /health` | Service health probe |
| `GET /ready` | Readiness probe; production profile validates persistence configuration |
| `GET /metrics` | Prometheus text metrics for process uptime and request count |
| `POST /api/v1/calligraphy/auth/register` | Create a local MVP learner account and session |
| `POST /api/v1/calligraphy/auth/login` | Login and return a local session |
| `POST /api/v1/calligraphy/auth/logout` | Revoke the current local session token |
| `GET /api/v1/calligraphy/auth/me` | Resolve current user from `Authorization: Bearer <token>` |
| `GET /api/v1/calligraphy/glyphs/search?character=山&style=ou` | Search licensed and published glyph samples |
| `GET /api/v1/calligraphy/glyphs/presets` | List preset common learning glyph groups |
| `GET /api/v1/calligraphy/glyphs/{glyph_id}` | Read glyph detail, notes, and practice templates |
| `POST /api/v1/calligraphy/layouts/preview` | Generate traditional vertical right-to-left composition preview |
| `POST /api/v1/calligraphy/artworks/drafts` | Save a composition draft from a layout request |
| `GET /api/v1/calligraphy/artworks/drafts?owner_user_id=user-1` | List drafts for a user |
| `GET /api/v1/calligraphy/artworks/drafts/{artwork_id}` | Read one draft |
| `DELETE /api/v1/calligraphy/artworks/drafts/{artwork_id}` | Delete one draft |
| `POST /api/v1/calligraphy/artworks/drafts/{artwork_id}/exports` | Export an MVP inline SVG reference artifact |
| `GET /api/v1/calligraphy/users/{owner_user_id}/learning` | Read favorite glyphs, recent practice, and learning counters |
| `POST /api/v1/calligraphy/users/{owner_user_id}/favorites` | Save a glyph to the learner's favorites |
| `DELETE /api/v1/calligraphy/users/{owner_user_id}/favorites/{glyph_id}` | Remove a saved favorite glyph |
| `POST /api/v1/calligraphy/users/{owner_user_id}/practice` | Record one daily practice action for a glyph and template |
| `GET /artifacts/{storage_key}` | Download locally stored trial export artifacts when `CALLIGRAPHY_EXPORT_DIR` is configured |

User-owned draft, favorite, practice, and learning-profile endpoints require `Authorization: Bearer <token>`. The server derives the effective owner from the session and rejects mismatched `owner_user_id` values with `403`.

The MVP service intentionally uses an in-memory seed catalog with 120+ preset common learning glyphs for both `ou` and `yan` styles. Auth, drafts, and learning records default to memory, but `CALLIGRAPHY_AUTH_FILE`, `CALLIGRAPHY_DATA_FILE`, and `CALLIGRAPHY_LEARNING_FILE` enable local JSON persistence for trial deployments, while `CALLIGRAPHY_AUDIT_FILE` writes JSONL audit events and `CALLIGRAPHY_EXPORT_DIR` writes SVG exports to disk. `CALLIGRAPHY_WEB_DIR` serves the static trial workbench from the same Go service. Production identity should move to Nebula Identity with PostgreSQL-backed users, while production content and exports must come from licensed copybook ingestion, expert-reviewed publication workflows, PostgreSQL persistence, and object storage exports before public launch.

The Go service also sets conservative HTTP timeouts and security response headers by default: `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, and `Content-Security-Policy`.

## Production Trial Profile

Use `CALLIGRAPHY_RUNTIME_PROFILE=production` for any externally reachable deployment. In this profile the service refuses to start unless these settings are present:

- `CALLIGRAPHY_AUTH_FILE`
- `CALLIGRAPHY_DATA_FILE`
- `CALLIGRAPHY_LEARNING_FILE`
- `CALLIGRAPHY_AUDIT_FILE`
- `CALLIGRAPHY_EXPORT_DIR`
- `CALLIGRAPHY_WEB_DIR`

Terminate HTTPS at a reverse proxy or load balancer and forward only HTTP to the Go process on a private port. The public proxy should set HSTS and request-size limits, and should expose `/health` and `/ready` to monitoring.

## Managed Foundation Profile

Use `CALLIGRAPHY_RUNTIME_PROFILE=managed` only when the deployment is wired to external platform foundations. The service validates the required foundation settings and reports `foundation_mode=managed` from `/ready`.

Required settings:

- `CALLIGRAPHY_DATABASE_URL`
- `CALLIGRAPHY_IDENTITY_ISSUER`
- `CALLIGRAPHY_IDENTITY_BASE_URL`
- `CALLIGRAPHY_IDENTITY_JWKS_URL` or `CALLIGRAPHY_IDENTITY_HS256_SECRET`
- `CALLIGRAPHY_OBJECT_STORAGE_ENDPOINT`
- `CALLIGRAPHY_OBJECT_STORAGE_BUCKET`
- `CALLIGRAPHY_OBJECT_STORAGE_REGION`
- `CALLIGRAPHY_OBJECT_STORAGE_ACCESS_KEY`
- `CALLIGRAPHY_OBJECT_STORAGE_SECRET_KEY`
- `CALLIGRAPHY_AUDIT_SINK`
- `CALLIGRAPHY_WEB_DIR`

Managed mode uses PostgreSQL stores for users, sessions, drafts, favorites, and practice records; S3-compatible object storage for exports; JWKS/RS256 or Nebula HS256 Bearer-token verification for Identity tokens; and an HTTP/HTTPS audit sink that receives JSON audit events. Set `CALLIGRAPHY_AUDIT_TOKEN` when the audit sink requires a bearer token.

Browser login is driven by `GET /api/v1/calligraphy/runtime-config`. The endpoint returns only public settings, never verifier secrets, object-storage credentials, database URLs, or audit tokens.

Supported managed web auth modes:

- `CALLIGRAPHY_AUTH_MODE=oidc-pkce`: standard browser SSO integration. Set `CALLIGRAPHY_IDENTITY_CLIENT_ID`; authorization and token endpoints default to `${CALLIGRAPHY_IDENTITY_BASE_URL}/api/v1/auth/authorize` and `${CALLIGRAPHY_IDENTITY_BASE_URL}/api/v1/auth/token`, or override with `CALLIGRAPHY_IDENTITY_AUTHORIZATION_ENDPOINT` and `CALLIGRAPHY_IDENTITY_TOKEN_ENDPOINT`.
- `CALLIGRAPHY_AUTH_MODE=nebula-direct`: compatibility fallback. The Web app posts username/password to `CALLIGRAPHY_IDENTITY_LOGIN_ENDPOINT`, defaulting to `${CALLIGRAPHY_IDENTITY_BASE_URL}/api/v1/auth/login`, then uses the returned access token against Calligraphy APIs.

Prefer `oidc-pkce` for managed production. When `CALLIGRAPHY_AUTH_MODE` is unset, managed mode uses `oidc-pkce` automatically if `CALLIGRAPHY_IDENTITY_CLIENT_ID` is present; otherwise it falls back to `nebula-direct` for controlled internal deployments. The service CSP automatically allows browser connections only to the configured Identity origin plus the Calligraphy origin.

For production, prefer routing Calligraphy and Identity behind the same HTTPS gateway origin. If they must be separate origins, configure Identity CORS to allow only the Calligraphy origin for the login or token endpoints used by the selected auth mode.

Identity must register Calligraphy as an exact-match OIDC public client before `oidc-pkce` is enabled:

```bash
OIDC_ISSUER=https://identity.example.com
OIDC_PUBLIC_CLIENTS=nebula-calligraphy-web=https://calligraphy.example.com/
CORS_ORIGINS=https://calligraphy.example.com
```

Then configure Calligraphy with the matching client and Identity public base URL:

```bash
CALLIGRAPHY_AUTH_MODE=oidc-pkce
CALLIGRAPHY_IDENTITY_BASE_URL=https://identity.example.com
CALLIGRAPHY_IDENTITY_CLIENT_ID=nebula-calligraphy-web
```

Before starting managed mode, run the PostgreSQL migration:

```bash
cd services/calligraphy
CALLIGRAPHY_DATABASE_URL=postgres://calligraphy:password@postgres:5432/calligraphy \
go run ./cmd/calligraphy-migrate
```

Then start the service with the managed settings above. `/ready` reports `foundation_mode=managed`.

Backup local production-trial state with:

```bash
CALLIGRAPHY_AUTH_FILE=/srv/calligraphy/auth.json \
CALLIGRAPHY_DATA_FILE=/srv/calligraphy/drafts.json \
CALLIGRAPHY_LEARNING_FILE=/srv/calligraphy/learning.json \
CALLIGRAPHY_AUDIT_FILE=/srv/calligraphy/audit.jsonl \
CALLIGRAPHY_EXPORT_DIR=/srv/calligraphy/artifacts \
scripts/calligraphy-backup.sh /srv/backups/calligraphy-$(date +%Y%m%d%H%M%S)
```

Restore the same variables and run:

```bash
scripts/calligraphy-restore.sh /srv/backups/<backup-dir>
```

## Local Checks

```bash
make -f Makefile.split split-check
make -f Makefile.split split-deploy-test
```
