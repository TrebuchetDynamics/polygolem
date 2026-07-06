# Polygolem Docs Site

Astro/Starlight presentation site for Polygolem documentation.

Canonical source-of-truth docs live one level up in [`docs/`](..). Keep this
site focused on navigation, landing pages, and web presentation; avoid creating
a second independent protocol reference here.

## Commands

```bash
npm install
npm run dev
npm run build
npm run deploy
```

`npm run deploy` builds the Astro/Starlight docsite and runs:

```bash
wrangler pages deploy dist --project-name polygolem
```

Requires Cloudflare/Wrangler auth for the `polygolem` Pages project.

## GitHub Actions

`.github/workflows/ci.yml` builds the docsite with `npm --prefix docs/docs-site ci`
and `npm --prefix docs/docs-site run build` on pull requests and main pushes.

`.github/workflows/docs-deploy.yml` publishes to Cloudflare Pages automatically
on main pushes that touch `docs/docs-site/**` (and via manual `workflow_dispatch`).
The deploy job is gated on the `CLOUDFLARE_API_TOKEN` repository secret: if the
secret is absent the job succeeds without deploying, so forks and un-provisioned
repos are never blocked. `npm run deploy` remains available for local publishes.

Automated and local deploys both require Cloudflare/Wrangler auth for the
`polygolem` Pages project:

- `CLOUDFLARE_API_TOKEN` — add as a repository secret to enable the workflow
- `CLOUDFLARE_ACCOUNT_ID` — add as a repository secret (or Wrangler env locally)

## Route map

- Landing page: `src/pages/index.astro` → `/`
- Starlight docs: `src/content/docs/docs/index.mdx` → `/docs/`
- Information architecture: `src/content/docs/docs/getting-started/docs-map.mdx` → `/docs/getting-started/docs-map`
- Guides/concepts/reference: `src/content/docs/docs/**` → `/docs/**`
- Cloudflare Pages config: `wrangler.toml` (`pages_build_output_dir = "./dist"`)

When linking between doc pages, use absolute `/docs/...` URLs. Links like
`/guides/...` or `/reference/...` point at the site root and will miss the
Starlight `/docs` base path.

## Authoring rules

1. Keep protocol facts in the canonical Markdown docs one level up, then link to
   them or mirror only the short navigation-focused summary here.
2. Keep every live-trading page explicit about the safe path: read-only by
   default, deposit-wallet/POLY_1271 only, credentials opt-in.
3. Run `npm run build` before deploy; this catches MDX/frontmatter errors and
   produces the exact `dist/` directory Wrangler uploads.
4. Do not commit `node_modules/`, `dist/`, `.astro/`, or `.wrangler/`; they are
   ignored in this directory.
