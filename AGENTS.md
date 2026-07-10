# WoW Markets clients — agent guide

Public monorepo for the WoW Markets addon and companion. The addon captures
Auctionator scans to SavedVariables; the Go/Wails companion archives and
uploads them. The API and web implementation are private and must never be
required by this checkout.

## Verify

- Everything: `make check`
- Addon: `make addon-check`
- Companion: `make companion-check`

The Go module root is `companion/`. Its `frontend/dist/` directory is
hand-written vanilla JavaScript, HTML, and CSS, not generated output.

## Invariants

- Never add credentials, private URLs, service implementation, deployment
  configuration, real user data, or material from local `refs/` or `data/`.
- Treat SavedVariables and canonical archive bytes as public contracts. Update
  fixtures and coordinate service compatibility before changing them.
- Never write to WoW SavedVariables or externally managed addon folders.
- Development must be upload-disabled or use a local mock by default. Only
  official release builds may select production implicitly.
- Release workflows and secrets must never be available to pull-request jobs.
