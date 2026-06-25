# nebula-calligraphy

`nebula-calligraphy` is the independent C-side product line for AI-assisted Chinese calligraphy learning and copybook-based artwork composition.

It connects to the Nebula platform foundation for identity, storage, AI gateway, audit, and future billing, but it does not belong to the Signage scheduling/player/device lifecycle flow.

## Product Scope

MVP focuses on the core user loop:

1. Search a character.
2. Compare famous copybook glyphs.
3. Compose text into a calligraphy artwork draft.
4. Adjust layout and glyph candidates.
5. Export PNG/PDF templates.
6. Save favorites and artwork drafts.

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

Available MVP endpoints:

| Endpoint | Purpose |
|----------|---------|
| `GET /health` | Service health probe |
| `GET /api/v1/calligraphy/glyphs/search?character=山&style=ou` | Search licensed and published glyph samples |
| `POST /api/v1/calligraphy/layouts/preview` | Generate traditional vertical right-to-left composition preview |

The MVP service intentionally uses an in-memory seed catalog. Production data must come from licensed copybook ingestion and expert-reviewed publication workflows before public launch.

## Local Checks

```bash
make -f Makefile.split split-check
make -f Makefile.split split-deploy-test
```
