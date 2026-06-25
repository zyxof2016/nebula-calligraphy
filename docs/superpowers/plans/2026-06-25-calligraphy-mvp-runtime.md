# Nebula Calligraphy MVP Runtime Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a compilable MVP backend runtime for the independent `nebula-calligraphy` C-side product line.

**Architecture:** The first runtime slice is a small Go HTTP service under `services/calligraphy`. It exposes character glyph search and composition layout preview APIs, keeps data in an in-memory seed catalog for local validation, and leaves identity/storage/AI integration behind documented Nebula foundation boundaries.

**Tech Stack:** Go 1.23, `net/http`, `go-chi/chi/v5`, table-driven Go tests, JSON contracts under `docs/contracts`.

---

### Task 1: Domain Model

**Files:**
- Create: `services/calligraphy/internal/model/model.go`
- Test through service and handler tests.

- [ ] Define JSON-facing types for glyph metadata, layout request, paper spec, layout result, and positioned slots.
- [ ] Keep field names aligned with `docs/contracts/glyph-v1.json` and `docs/contracts/layout-v1.json`.

### Task 2: Layout Engine

**Files:**
- Create: `services/calligraphy/internal/service/layout.go`
- Create: `services/calligraphy/internal/service/layout_test.go`

- [ ] Write tests for punctuation filtering, vertical right-to-left slot ordering, margin validation, and empty text rejection.
- [ ] Run `go test ./internal/service -run Layout` from `services/calligraphy` and confirm tests fail before implementation.
- [ ] Implement the minimum deterministic layout algorithm needed for MVP preview.
- [ ] Re-run service tests and keep them green.

### Task 3: Glyph Catalog

**Files:**
- Create: `services/calligraphy/internal/service/glyph_catalog.go`
- Create: `services/calligraphy/internal/service/glyph_catalog_test.go`

- [ ] Write tests for character/style/copybook filtering and restricted-license exclusion.
- [ ] Run `go test ./internal/service -run Glyph` from `services/calligraphy` and confirm tests fail before implementation.
- [ ] Implement an in-memory seed catalog and search filters.
- [ ] Re-run service tests.

### Task 4: HTTP API

**Files:**
- Create: `services/calligraphy/internal/handler/handler.go`
- Create: `services/calligraphy/internal/handler/handler_test.go`
- Create: `services/calligraphy/cmd/calligraphy/main.go`
- Create: `services/calligraphy/go.mod`

- [ ] Write handler tests for `/health`, `/api/v1/calligraphy/glyphs/search`, and `/api/v1/calligraphy/layouts/preview`.
- [ ] Run `go test ./internal/handler` and confirm tests fail before implementation.
- [ ] Implement JSON helpers, routes, startup, and graceful shutdown.
- [ ] Re-run `go test ./...`.

### Task 5: Split Checks And Docs

**Files:**
- Modify: `Makefile.split`
- Modify: `README.md`
- Modify: `SPLIT_READINESS.md`
- Modify: `docs/products/calligraphy.md`

- [ ] Make `split-check` validate JSON contracts and run Go tests.
- [ ] Make `split-deploy-test` run the runtime test suite.
- [ ] Document local run commands and MVP API boundaries.
- [ ] Run `make -f Makefile.split split-check` from `nebula-calligraphy`.
