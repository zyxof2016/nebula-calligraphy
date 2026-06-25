# nebula-calligraphy Split Readiness

## Current Status

`nebula-calligraphy` now contains the product and contract baseline plus a compilable MVP Go API runtime.

The runtime is intentionally limited to licensed/published glyph search, 120+ preset common learning glyphs per supported seed style (`ou`, `yan`), deterministic layout preview, artwork drafts, SVG export/download, and a static trial web workbench. Drafts can run in memory or persist to a local JSON file for trials; SVG exports can be returned inline or written to a local artifact directory. Identity enforcement, PostgreSQL persistence, object storage, and AI features remain explicit next integration steps.

## Required Checks

```bash
make -f Makefile.split split-check
make -f Makefile.split split-deploy-test
```

## MVP Readiness Gates

- Copybook and glyph licensing plan is approved before importing real assets.
- MVP data scope is fixed: two copybooks, two styles, 1000 high-frequency glyphs.
- Identity, object storage, AI gateway, and audit integration contracts are wired behind configuration.
- Trial deployments can set `CALLIGRAPHY_DATA_FILE` and `CALLIGRAPHY_EXPORT_DIR` to avoid losing drafts across restarts.
- Trial deployments can set `CALLIGRAPHY_WEB_DIR=../../web/app` when running from `services/calligraphy` to serve the browser workbench.
- Export contract supports SVG now; PNG/PDF output and object-storage checksum records are required before public beta.
- Layout engine supports zhongtang, tiaofu, doufang, couplet, and custom size before public beta.
- Admin publication flow prevents restricted or unreviewed glyphs from being served to C-side users.
