# Test S3 upload

## Upload (type = avatar | item)

Everything in one command inside the api container — the token is obtained, a minimal JPEG is created in a temp file, and it's uploaded right away:

```bash
docker compose -f compose.yml -f compose.override.dev.yml exec api sh -c '
  TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"test@example.com\",\"password\":\"password123\"}" | jq -r .access_token)

  printf "\377\330\377\340" > /tmp/t.jpg
  dd if=/dev/zero bs=1 count=508 >> /tmp/t.jpg 2>/dev/null

  curl -s -X POST http://localhost:8080/api/v1/upload \
    -H "Authorization: Bearer $TOKEN" \
    -F "type=avatar" \
    -F "file=@/tmp/t.jpg;type=image/jpeg"
' | jq .
```

For a listing photo, replace `avatar` with `item` in `-F "type=..."`.

## Storage mode

The local dev environment supports two modes through the same set of `S3_*` variables:

- `S3_ENDPOINT=file:///app/.uploads` — local filesystem storage inside the API container. Files are available via `S3_PUBLIC_BASE_URL=http://api.avi.test/uploads`.
- `S3_ENDPOINT=https://s3.cloud` — a real cloud dev bucket. Requires real `S3_KEY_ID`, `S3_KEY_SECRET`, `S3_BUCKET=bucket-avi-dev`, and `S3_PUBLIC_BASE_URL=https://global.s3.cloud/avi-dev`.

After switching `.env`, restart the containers with:

```bash
make dev
```

The prod config should use a separate prod bucket, e.g. `S3_BUCKET=bucket-classifieds-prod`, and a public prod URL.

## Verify

### Avatar

```bash
# DB
docker compose -f compose.yml exec db psql -U avi -d avi \
  -c "SELECT id, object_key, size_bytes FROM user_avatars ORDER BY created_at DESC LIMIT 3;"
```

### Item photo

```bash
docker compose -f compose.yml exec db psql -U avi -d avi \
  -c "SELECT id, object_key, size_bytes FROM item_photos ORDER BY created_at DESC LIMIT 3;"
```

### Object storage

In dev, local filesystem storage `file:///app/.uploads` is used by default; files are served through `S3_PUBLIC_BASE_URL`, e.g. `http://api.avi.test/uploads/...`.

If you switched `.env` to cloud S3 and configured the rclone remote `cloudru`:

```bash
rclone ls cloudru:bucket-avi-dev
```

## Troubleshooting

- **`401`** — token wasn't obtained. Check that the api container is running and the user `test@example.com` exists (register and verify the email, or use another user).
- **`unsupported image type`** — the body wasn't recognized as `image/*`. Make sure `/tmp/t.jpg` isn't empty (`ls -l /tmp/t.jpg` inside the container).
- **`500` + `MissingContentLength` in api logs** — the SDK couldn't set `Content-Length`. Should already be fixed: the body is buffered and `ContentLength` is passed explicitly to `PutObject`. If it reproduces, rebuild api: `make dev-build`.
- **`500` + `NoSuchBucket` in api logs** — the API is pointing at a remote S3 bucket that doesn't exist or isn't accessible. For local development, switch back to the dev settings `S3_ENDPOINT=file:///app/.uploads` and `S3_PUBLIC_BASE_URL=http://api.avi.test/uploads`, or create/fix the remote bucket.
- **`403 AccessDenied`** — check `S3_KEY_ID` / `S3_KEY_SECRET` / `S3_BUCKET` in `.env`. Key format for cloud: `tenantID:keyID`. Cross-check the values against the secret source.
- **`500` right after startup with `S3_KEY_SECRET=YOUR_S3_KEY_SECRET`** — a placeholder from `.env.dev.example` is still in `.env`; substitute the real secret.
