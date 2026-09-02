# apps/

- `web/` — the Nuxt 4 frontend (owner: trading-frontend). Currently lives at the repo-root
  `1Trade Frontend/`; migrates here per `docs/plans/REPO_LAYOUT.md` §3.
- `cli/` — the `1trade` Go CLI (owner: platform-core, F04).

Both are thin clients over the public APIs in `docs/contracts/`. The frontend follows
`docs/plans/DESIGN_SYSTEM.md` (tokens only).
