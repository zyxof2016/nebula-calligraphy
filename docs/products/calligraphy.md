# Nebula Calligraphy Product Spec

## Positioning

Nebula Calligraphy is a C-side AI calligraphy learning and copybook-based artwork composition app. It gives learners a portable calligraphy tutor for character lookup, style comparison, composition planning, and exportable practice templates.

## MVP Scope

| Module | Included | Deferred |
|--------|----------|----------|
| Character dictionary | Pinyin/radical/stroke search, glyph comparison, source copybook, basic writing notes | Photo search, handwriting search, voice explanation |
| Copybook library | `Jiuchenggong` and `Duobaota`, 1000 high-frequency glyphs | Large-scale copybook expansion |
| Composition | Text input, style/copybook choice, paper format, automatic layout, glyph replacement | Radical synthesis, cross-calligrapher automatic mixing |
| Export | PNG/PDF, tracing template, artwork reference | AR tracing and video cards |
| User assets | Favorites, daily practice records, learning profile, artwork drafts, recent history | Community and classroom workflows |

## Current Runtime Slice

The repository now includes `services/calligraphy`, a Go MVP API service used to validate the C-side loop before mobile and admin UI work begins.

| API | Status | Notes |
|-----|--------|-------|
| `GET /health` | Implemented | Container and process health probe |
| `GET /api/v1/calligraphy/glyphs/search` | Implemented | Searches only licensed and published seed glyphs |
| `GET /api/v1/calligraphy/glyphs/presets` | Implemented | Returns 120+ preset common learning glyphs per style, grouped by practice purpose |
| `GET /api/v1/calligraphy/glyphs/{id}` | Implemented | Returns glyph detail, structure notes, brushwork notes, and practice templates |
| `POST /api/v1/calligraphy/layouts/preview` | Implemented | Traditional `vertical_rtl` layout preview with margin, signature, and seal slots |
| `POST /api/v1/calligraphy/artworks/drafts` | Implemented | Saves an artwork draft from a layout request; memory by default, JSON file when configured |
| `GET /api/v1/calligraphy/artworks/drafts` | Implemented | Lists drafts by `owner_user_id`; Identity middleware is not wired yet |
| `DELETE /api/v1/calligraphy/artworks/drafts/{id}` | Implemented | Deletes one trial draft |
| `POST /api/v1/calligraphy/artworks/drafts/{id}/exports` | Implemented | Produces an SVG reference export with SHA256; inline by default, local artifact file when configured |
| `GET /api/v1/calligraphy/users/{id}/learning` | Implemented | Returns favorite glyphs, recent practice records, and learning counters |
| `POST /api/v1/calligraphy/users/{id}/favorites` | Implemented | Saves a published glyph as a learner favorite |
| `DELETE /api/v1/calligraphy/users/{id}/favorites/{glyph_id}` | Implemented | Removes one learner favorite |
| `POST /api/v1/calligraphy/users/{id}/practice` | Implemented | Records one glyph practice action with template and grid type |
| `GET /artifacts/{storage_key}` | Implemented | Serves local SVG exports when `CALLIGRAPHY_EXPORT_DIR` is configured |
| Static trial workbench | Implemented | Served from `web/app` through `CALLIGRAPHY_WEB_DIR`; supports preset common glyphs, glyph lookup/detail, practice template preview/download, favorites, practice records, learning profile, composition preview, save, list, load, delete, export history, and SVG export/download |

This service can persist drafts and learning records to local JSON files for trials, but it does not yet provide production persistence or object storage. Those features must be added with Identity, PostgreSQL, object storage, audit, and licensed copybook ingestion.

## Foundation Integration

| Foundation capability | Usage |
|----------------------|-------|
| Identity | User account, membership, personal workspace |
| Object storage | Copybook images, glyph crops, exports, user artworks |
| AI gateway | OCR, similarity, critique, glyph generation, all behind feature flags |
| Audit | Export, AI generation, admin publication, licensing-sensitive operations |
| Open platform | Future glyph/layout/export APIs |

## Non-goals

- It is not part of Signage scheduling, Player playback, Device Hub OTA, or RemoteOps.
- It does not ship community features in MVP.
- It does not claim authoritative AI scoring before expert-validated evaluation is available.
