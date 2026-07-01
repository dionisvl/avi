# Integration Tests

These tests require external services to be running and are skipped by default.

## Email Integration Tests

Tests SMTP email sending via Mailpit.

**Requirements:**
- Mailpit running (available in `compose.override.dev.yml`)
- Environment variables:
  - `RUN_INTEGRATION_TESTS=true`
  - `SMTP_HOST=localhost` (default)
  - `SMTP_PORT=1025` (default)
  - Mailpit API available on port `8025`

**Run from container:**
```bash
make test-integration
```

**Run locally (if Mailpit is running):**
```bash
RUN_INTEGRATION_TESTS=true go test ./tests/integration/... -v
```

**With custom Mailpit host:**
```bash
SMTP_HOST=mailpit RUN_INTEGRATION_TESTS=true go test ./tests/integration/... -v
```
