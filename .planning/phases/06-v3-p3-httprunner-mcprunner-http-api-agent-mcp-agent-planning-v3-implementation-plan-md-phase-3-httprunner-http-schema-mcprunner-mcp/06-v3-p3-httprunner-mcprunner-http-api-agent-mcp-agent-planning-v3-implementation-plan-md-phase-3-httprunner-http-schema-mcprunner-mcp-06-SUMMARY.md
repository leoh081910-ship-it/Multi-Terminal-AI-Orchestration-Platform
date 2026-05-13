## Plan 06: Full regression and validation

**Status:** Complete

**Verification commands all exit 0:**
- `go test ./...` — all packages green
- `npm --prefix web run lint` — no errors
- `npm --prefix web run build` — production build succeeds

**No waivers applied. All tests pass without unrelated-failure exceptions.**
