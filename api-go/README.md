# avi Marketplace API

Demo avi marketplace backend API built with Go (C2C item listings).

## Stack

- Date: 2026 April
- Language: Go 1.26.2
- OS: Docker Alpine 3.23
- Routing: Chi v5 (HTTP router)
- Database: PostgreSQL 18.3 with uuidv7()
  - DB Driver: pgx/v5
  - Migrations: Goose v3
  - Squirrel - SQL Builder (soon)
- Auth: JWT (Access + Refresh tokens)
- Password: bcrypt
- Validation: go-playground/validator/v10
- Logging: slog (structured logging)
- API Docs: Swagger/OpenAPI with swaggo
- Dev: Air (hot reload)
- Code Style: gofumpt, gci, golangci-lint, staticcheck and govet

## Project Structure

```
api-go/
├── cmd/api/               # Entry point
├── internal/
│   ├── api/              # HTTP handlers & middleware
│   ├── service/         # Business logic (write path)
│   ├── query/           # Read path (CQRS-lite)
│   ├── repository/      # Data access layer
│   ├── model/          # Domain models
│   ├── config/         # Configuration
│   ├── errors/         # Error types (RFC 9457)
│   ├── migrations/     # SQL migrations
│   └── app/           # DI container
├── tests/              # Integration tests
└── docs/               # Swagger docs

```

## API Endpoints

### Auth
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login (returns access + refresh tokens)
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/reset-password/request` - Request password reset
- `POST /api/v1/auth/reset-password/confirm` - Confirm reset code
- `POST /api/v1/auth/reset-password/set` - Set new password

### Health
- `GET /health` - API health check

## Error Handling

API returns [RFC 9457 Problem Details](https://datatracker.ietf.org/doc/html/rfc9457) JSON:

```json
{
  "status": 400,
  "title": "Validation Failed",
  "detail": "Request validation failed"
}
```

- Generic client messages (no internal error details exposed)
- Internal errors logged server-side for debugging
- Sentinel errors in business logic → HTTP mapping at handler layer

## Features

- ✅ Password hashing with bcrypt
- ✅ Rate limiting on auth endpoints (5 req/10sec)
- ✅ CORS configured for dev environment
- ✅ Database migrations (embedded FS, auto-run on startup)
- ✅ RFC 9457 Problem Details error responses
- ✅ Test user seeded in dev environment

## Development

View Swagger UI:
- http://api.avi.test/swagger/
- https://api.avi.app/swagger/


## Configuration

Environment variables (see `internal/config/config.go`):
- `APP_ENV` - Environment (dev/prod)
- `APP_PORT` - Server port (default :8080)
- `DB_DSN` - PostgreSQL connection string
- `JWT_ACCESS_SECRET` - Access token secret
- `JWT_REFRESH_SECRET` - Refresh token secret
- `SMTP_HOST` - Email server (default: mailpit)
- `SMTP_PORT` - Email server port (default: 1025)

## Read Path / CQRS-lite Architecture

The API implements a CQRS-lite pattern for better separation of concerns:

### Layers
- **Handlers** (`internal/api/*`) accept requests and return responses. For read operations, they depend on query services.
- **Query Services** (`internal/query/*`) handle read operations with view-specific projections.
- **Write Services** (`internal/service/*`) handle write operations (POST, PATCH, DELETE) only.
- **Repositories** (`internal/repository/*`) are the single source of truth for data access.

### Key Rules
1. Query layer **never imports** write service packages.
2. View models in `internal/query/*` have **no JSON tags** (separation from transport).
3. Response DTOs with JSON tags live in `internal/api/*response.go` files.
4. Handlers map query projections → response DTOs at the handler layer.
5. Write operations use write-services; read operations use query-services.

### Example: Item Listing
- **Handler** `internal/api/item/handler.go` calls `itemquery.Service.List()`.
- **Query Service** `internal/query/item/service.go` projects items to the `Item` read view with is_favorited.
- **Response DTO** `internal/api/item/response.go` maps the view to `ItemResponse` with JSON tags.

## Notes

- Migrations run automatically on startup
- Request bodies and query params validated with validator/v10; path params checked via uuid.Parse/strconv
- Response DTOs preserve JSON field names for API contracts; query models are internal projections
- Email sending is stubbed (no-op) for development
