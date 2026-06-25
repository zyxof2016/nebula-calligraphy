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
| User assets | Favorites, artwork drafts, recent history | Community and classroom workflows |

## Current Runtime Slice

The repository now includes `services/calligraphy`, a Go MVP API service used to validate the C-side loop before mobile and admin UI work begins.

| API | Status | Notes |
|-----|--------|-------|
| `GET /health` | Implemented | Container and process health probe |
| `GET /api/v1/calligraphy/glyphs/search` | Implemented | Searches only licensed and published seed glyphs |
| `POST /api/v1/calligraphy/layouts/preview` | Implemented | Traditional `vertical_rtl` layout preview with margin, signature, and seal slots |

This service does not yet persist user drafts or render exports. Those features must be added with Identity, object storage, audit, and licensed copybook ingestion.

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
