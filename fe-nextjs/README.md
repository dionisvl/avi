# Public Frontend

Public storefront for the **Avi** marketplace. Homepage implemented against the Go API in [`../api-go`](../api-go).

- Next.js 16
- React 19
- TypeScript
- Tailwind v4
- shadcn/ui
- TanStack Query v5
- Biome

## Run

```bash
npm install
npm run dev        # http://localhost:3000
```

Data comes from `NEXT_PUBLIC_API_URL` (default `http://api.avi.test`; start the backend with `make dev` at the repo root). Without it the page renders with loading skeletons. Copy `.env.example` → `.env.local` to override.

## Scripts

```bash
npm run build       # production build
npm run lint        # Biome
npm run typecheck   # tsc --noEmit
npm run gen:api     # regenerate API types from ../api-go/docs/swagger.json
```

## Demo mode

Public-demo behaviour is gated by `NEXT_PUBLIC_DEMO_MODE` (build-time). When `true`:

- Registration is not exposed; personal-data write forms short-circuit (`assertNotDemo()` in `lib/demo.ts`).
- The login page shows preset accounts with public credentials and a Mailpit link.

Preset accounts (seeded by `../api-go/internal/migrations/00002_seed_demo_users.sql`; content in `00003_*`):

| email | password |
| --- | --- |
| demo1@avi.test | demo1 |
| demo2@avi.test | demo2 |
| demo3@avi.test | demo3 |

To run as a normal (non-demo) app, set `NEXT_PUBLIC_DEMO_MODE=false` — the backend is unchanged, so registration and all endpoints work as usual. `NEXT_PUBLIC_*` are inlined at build time, so rebuild after changing them.

## Architecture

Feature-based + colocation. Routes stay thin; logic lives in `features/`.

```
src/
  app/            # routing only (page/layout/providers)
  features/
    listing/      # cards, sections, infinite scroll + queries/mutations
    search/       # hero, search bar
    catalog/      # header, chips, promo, bottom-nav + categories/cities
  components/ui/  # shadcn primitives
  lib/            # api client + generated types, format, utils
  i18n/           # en base + ru placeholder
```

- **Data flow**: `get-*.ts` (openapi-fetch) → `use-*.ts` (TanStack Query) → components. No fetching in components.
- **Types**: generated from the API's OpenAPI schema (`npm run gen:api`); domain aliases in `lib/api/types.ts`.
- **Boundaries**: features don't import each other (one exception: `search` uses `catalog`'s `useCities`). No barrel files; `@/` across layers, relative within a feature; kebab-case + `get-`/`use-` suffixes.
- **Tokens**: `src/app/globals.css` (`@theme`) — use semantic utilities (`bg-surface`, `text-h1`…), not raw hex.
- **i18n**: copy via `useT()`; English base, Russian scaffolded.
