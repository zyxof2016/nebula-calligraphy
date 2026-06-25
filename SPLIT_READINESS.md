# nebula-calligraphy Split Readiness

## Current Status

`nebula-calligraphy` now contains the product and contract baseline plus a compilable MVP Go API runtime.

The runtime is intentionally limited to licensed/published glyph search and deterministic layout preview. Persistent user assets, export rendering, identity enforcement, object storage, and AI features remain explicit next integration steps.

## Required Checks

```bash
make -f Makefile.split split-check
make -f Makefile.split split-deploy-test
```

## MVP Readiness Gates

- Copybook and glyph licensing plan is approved before importing real assets.
- MVP data scope is fixed: two copybooks, two styles, 1000 high-frequency glyphs.
- Identity, object storage, AI gateway, and audit integration contracts are wired behind configuration.
- Export contract supports PNG/PDF output and checksum storage.
- Layout engine supports zhongtang, tiaofu, doufang, couplet, and custom size before public beta.
- Admin publication flow prevents restricted or unreviewed glyphs from being served to C-side users.
