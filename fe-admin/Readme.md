# FE admin

Admin panel frontend for a demo classifieds marketplace inspired by popular C2C
listing platforms.

This frontend is a neutral portfolio demo and is not affiliated with any
classifieds platform. All trademarks belong to their respective owners.

- Alpine.js 3.15
- Tailwind CSS 4.2
- FilePond 4.32
- Tests: Playwright 1.59.1

## Build

Vite builds everything into `dist/`. Nginx mounts `dist/` as document root.

```bash
cd fe-admin
npm run build
```

### env.js

After build, restart the container so the entrypoint regenerates `dist/env.js`
from `env.js.template` with `API_URL` substitution:

```bash
# from repository root
docker compose -f compose.yml -f compose.override.dev.yml restart fe-admin
```

Or use:

```bash
make dev-build
```

## Tests

Playwright specs mock the API routes that are not completed yet in the backend
rework. This keeps the test frontend aligned with the checked checklist items:
`items`, `categories`, `seller`, `favorites`, `chat`, `upload`, `category_ids`
catalog filtering, browser routes, and `promote_listing` payments.

```bash
cd fe-admin
npm test
```

### Covered areas

| File | Cases |
|------|-------|
| `auth.spec.js` | Login, logout, neutral registration payload, authenticated layout |
| `items.spec.js` | Catalog, ru/en labels, filters, detail, create form, promote listing payment |
| `favorites.spec.js` | Empty state, add/remove item favorites |
| `profile.spec.js` | Profile update, item-based preferences, password change states, delete validation |
| `chat.spec.js` | Conversations, messages, send message |
| `upload.spec.js` | Item photo upload control initialization |
| `routes.spec.js` | Direct catalog/detail routes, query-synced filters, auth return route, browser back navigation |
