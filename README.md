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
| User identity | `nebula-platform` Identity |
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
CALLIGRAPHY_DATA_FILE=.local/drafts.json \
CALLIGRAPHY_LEARNING_FILE=.local/learning.json \
CALLIGRAPHY_EXPORT_DIR=.local/artifacts \
CALLIGRAPHY_WEB_DIR=../../web/app \
PORT=8090 \
go run ./cmd/calligraphy
```

Then open `http://127.0.0.1:8090/` to use the trial workbench. The workbench covers preset common glyph groups, glyph lookup, glyph detail, practice template preview/download, favorite glyphs, practice records, learning profile, layout preview, draft save/list/load/delete, SVG export history, and SVG export/download.

Available MVP endpoints:

| Endpoint | Purpose |
|----------|---------|
| `GET /health` | Service health probe |
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

The MVP service intentionally uses an in-memory seed catalog with 120+ preset common learning glyphs for both `ou` and `yan` styles. Drafts and learning records default to memory, but `CALLIGRAPHY_DATA_FILE` and `CALLIGRAPHY_LEARNING_FILE` enable local JSON persistence for trial deployments, while `CALLIGRAPHY_EXPORT_DIR` writes SVG exports to disk. `CALLIGRAPHY_WEB_DIR` serves the static trial workbench from the same Go service. Production data must come from licensed copybook ingestion, expert-reviewed publication workflows, PostgreSQL persistence, and object storage exports before public launch.

## Local Checks

```bash
make -f Makefile.split split-check
make -f Makefile.split split-deploy-test
```
