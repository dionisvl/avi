# Project context for Copilot code review

## Stack & versions (pinned — do NOT flag as "outdated")
- Go: **1.26.2** (toolchain in `go.mod`). Project actively uses Go 1.26+ semantics.
- Postgres 18.3-alpine, `pgx/v5`, `chi/v5`, `validator/v10`, `goose` migrations.
- Year: 2026.
- Read compose.yml and api-go/go.mod always to see actual versions

## Go review rules — what NOT to flag
- Do NOT warn about loop-variable capture in `for i := ...` / `for k, v := range`
  — fixed in Go 1.22. Also do NOT warn when `var x` is declared **inside**
  the loop body (`for rows.Next() { var x T; m[k] = &x }`) — each iteration
  gets its own variable; escape analysis handles `&x` correctly.
- Modern stdlib is preferred and not a smell:
  `slices.Contains`, `slices.Backward`, `maps.Copy`, `reflect.TypeFor[T]()`,
  `strings.SplitSeq`, `for range N`, `any` (not `interface{}`).
- `json.RawMessage` arrives without surrounding whitespace — no need to
  `bytes.TrimSpace` before comparing to `"null"`.
- `any` is correct; do not suggest replacing with `interface{}`.

## Architecture (CQRS-lite)
- Reads: `internal/query/*` — must NOT import `internal/service/*`.
- Writes: `internal/service/*`.
- JSON tags live ONLY in `internal/api/*` view models, not in query layer.
- Repo errors in service/query must be wrapped via `apierr.Wrap`, never returned raw.
- Use `errors.Is(err, pgx.ErrNoRows)`, never `err == pgx.ErrNoRows`.

## What TO flag
- Input validation: bodies/query via `validator/v10`; path params via
  `uuid.Parse` / `strconv` — NOT validator.
- New env vars must be added to all three: `.env`, `.env.dev.example`, `.env.prod.example`.
- New user-facing strings must support i18n (`ru` + `en` minimum).
- API contract changes without regenerated swagger (`make sw`).
- Raw SQL string concatenation; always use parameterized queries.

## Domain conventions

## Tests
- Use random data (UUIDs, random emails) — never hardcoded identifiers
  that can collide.
