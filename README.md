# Avi - Classifieds Marketplace

[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](api-go/go.mod)
[![License](https://img.shields.io/badge/license-PolyForm%20Noncommercial%201.0.0-blue)](LICENSE)

Demo classifieds marketplace inspired by popular C2C listing platforms. Multi-service infrastructure with Traefik, a Go API, and a frontend.

---

> This project is not affiliated with Avito, OLX, or any other classifieds platform. All trademarks belong to their respective owners.

---

App under active development. There is no sensitive data, so feel free to change old migrations and drop the database at any time.

## Table of Contents

- [Tech Stack](#tech-stack)
- [Project Structure](#project-structure)
- [Architecture](#architecture)
- [DDD: Ubiquitous Language](#ddd-ubiquitous-language)
- [Quick Start](#quick-start)
- [Services](#services)
- [Database](#database)
- [Development](#development)
- [Production](#production)
- [License](#license)

## Tech Stack

**Backend**
- Go 1.26, [chi](https://github.com/go-chi/chi) router, CQRS-lite architecture (read/write separation)
- PostgreSQL via [pgx](https://github.com/jackc/pgx), [goose](https://github.com/pressly/goose) migrations
- JWT auth ([golang-jwt](https://github.com/golang-jwt/jwt)), request validation via [validator](https://github.com/go-playground/validator)
- WebSocket chat ([coder/websocket](https://github.com/coder/websocket))
- S3-compatible object storage via [AWS SDK v2](https://github.com/aws/aws-sdk-go-v2)
- Transactional email via [Resend](https://resend.com/), SMTP fallback
- YooKassa payments integration
- Swagger/OpenAPI docs via [swaggo](https://github.com/swaggo/swag)
- i18n: `ru` + `en` throughout the API and email templates

**Frontend (admin panel)**
- [Alpine.js](https://alpinejs.dev/), [Tailwind CSS](https://tailwindcss.com/), [Vite](https://vitejs.dev/)
- [FilePond](https://pqina.nl/filepond/) for uploads, [Tabulator](https://tabulator.info/) for data tables
- E2E tests with [Playwright](https://playwright.dev/)

**Infrastructure**
- Traefik v3 reverse proxy, Docker Compose (dev/prod overrides)
- Dozzle (logs), Netdata (metrics), Mailpit (dev SMTP)
- GitHub Actions CI: format, lint, build, test, Swagger drift check

## Project Structure

```
/
├── compose.yml                    # Base infrastructure (Traefik, Frontend, DB, etc)
├── compose.override.dev.yml       # Dev overrides (local domains, hot-reload)
├── compose.override.prod.yml      # Prod overrides (HTTPS, Let's Encrypt)
├── .env                           # Dev environment (auto-loaded)
├── Makefile                       # Commands
├── fe-admin/                      # Admin panel frontend
└── api-go/                        # Go API application
```

## Architecture

```mermaid
flowchart TB
    client["Internet / Local Browser"]
    traefik["Traefik - reverse proxy"]

    client --> traefik

    traefik --> feadmin["fe-admin - Nginx SPA"]
    traefik --> api["api-go - Go HTTP API"]
    traefik --> ops["Ops tools - Dozzle / Netdata"]

    feadmin -- "HTTP /api/v1" --> api

    api --> handlers["API handlers - auth, items, chat, upload, payments"]
    handlers --> app["Application layer - services and queries"]

    app --> pg["PostgreSQL - migrations, users, items, chat, payments"]
    app --> storage["Object storage - item photos, avatars, chat attachments"]
    app --> email["Email delivery - Mailpit in dev, Resend in prod"]
```

## DDD: Ubiquitous Language

One term - one concept. The team uses the same names in speech, documentation, and code.

- Entities are named after their domain meaning, not their UI, retrieval method, or technical implementation.
- Giving different names to the same concept is forbidden.
- Related but distinct concepts are separated explicitly: for example, promotion payment, withdrawal, and balance are different terms.

> A new term is recorded in the glossary first, then used in the code.

## Quick Start

```bash
# setup /etc/hosts
# setup Dozzle creds
# setup HTTP_BASIC_AUTH_HASH in .env (htpasswd -nbB admin 'pass')

# Start dev
make dev
```

### Access
* http://admin.avi.test    # Admin panel (fe-admin)
* http://api.avi.test      # API
* http://api.avi.test/swagger # API docs (Swagger UI)
* http://logs.avi.test     # Dozzle (logs)
* http://metrics.avi.test  # Netdata (metrics, basic auth)

## Services

- Traefik v3.6 - Reverse proxy
- API (Go) - Backend API
- Frontend (Nginx) - SPA
- PostgreSQL - Database (check actual port in .env)
- Dozzle - Log viewer
- Netdata - Metrics monitoring
- Mailpit - Test SMTP

## Database

- Dev: `postgres://avi:avi@127.0.0.1:54444/avi` (port published via `compose.override.dev.yml`)
- Prod: not published externally — connect via SSH tunnel or from within the docker network

## Development

```bash
make dev          # Start (dev mode)
make down         # Stop
make logs         # View logs
make sh           # Shell into API
```

## Production

```bash
make prod         # Switch to prod mode
```

Requires:
- `.env` with prod real values
- DNS configured for api.avi.app and avi.app

See Readme.md's in individual services for details.

---

## License

Software source code and software components are licensed for non-commercial
use only under the [PolyForm Noncommercial License 1.0.0](LICENSE). Commercial
use requires prior written permission from the copyright holder.

Any use, copying, distribution, modification, or incorporation of this project
or its components must preserve the required attribution notices in [NOTICE](NOTICE).

Documentation and non-code assets are licensed for non-commercial use with
attribution under Creative Commons Attribution-NonCommercial 4.0 International
(CC BY-NC 4.0), unless a file states otherwise.

Reference copy of the PolyForm license:
https://raw.githubusercontent.com/polyformproject/polyform-licenses/1.0.0/PolyForm-Noncommercial-1.0.0.md
