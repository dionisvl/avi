# Docker Compose Best Practices

This document describes critical settings for a production-ready Docker Compose configuration.

## Main problems with compose files

Everything works locally, but problems appear in production:
- Postgres eats all the memory, and the OOM-killer kills neighboring services
- A container crashes at night and Docker doesn't restart it (restart policy = `no` by default)
- A month of logs takes up 40 GB, and the disk fills up
- Docker doesn't know the app is stuck (the process is alive, but the service doesn't respond)
- Volumes without backups — one `docker compose down -v` and the data is gone

All of these are solved with a few lines in the compose file.

---

## 1. Memory and CPU limits

### Problem
By default, a container can use **all the memory** and **all the host's cores**. On a server with several containers, any one of them can grab all the resources. It's usually Postgres — it's told "take what you need," and it does. When RAM runs out, the Linux OOM-killer kills the process with the highest consumption, and it's often not Postgres but your application.

### Solution
Add limits via `deploy.resources`:

```yaml
services:
  api:
    deploy:
      resources:
        limits:
          memory: 512M      # Ceiling
          cpus: "1.0"
        reservations:
          memory: 256M      # Guaranteed minimum
```

**What happens:**
- `limits` — a hard ceiling. If the container exceeds the memory limit, Docker kills it itself (cgroup OOM).
- `reservations` — a guaranteed minimum amount of resources.

### Checking for OOM
```bash
docker inspect myapp --format='{{.State.OOMKilled}}'
```
If `true` — the container was killed for exceeding its limit.

### Postgres: shared_buffers
When setting a memory limit, Postgres needs `shared_buffers` configured — usually **25% of available memory**.

Example: a 1G limit → `shared_buffers=256MB`

```yaml
services:
  db:
    command: postgres -c shared_buffers=256MB
    deploy:
      resources:
        limits:
          memory: 1G
```

If you leave the default 128MB with a 1G limit, Postgres will use RAM through the OS file cache — less predictable.

---

## 2. Restart Policy

### Problem
By default it's `restart: no`. The container crashes — and stays down. Nobody restarts it until you wake up.

### Solution
```yaml
services:
  api:
    restart: unless-stopped
```

**Options:**
- `no` (default) — don't restart
- `always` — always restart (even after `docker compose stop` — gets in the way during maintenance)
- `unless-stopped` ✅ — restart on any crash, except a manual stop
- `on-failure` — restart only on a non-zero exit code

### Exceptions
For one-off tasks (migrations, seeding data), use `restart: "no"` or `on-failure`:

```yaml
services:
  migrator:
    image: myapp:latest
    command: ["python", "manage.py", "migrate"]
    restart: "no"

  app:
    restart: unless-stopped
    depends_on:
      migrator:
        condition: service_completed_successfully
```

The application only starts after a successful migration. If the migration fails, the application won't start.

---

## 3. Log rotation

### Problem
Docker writes logs to a JSON file **with no size limit**. A service at 100 RPS will generate tens of gigabytes in a month.

Real example: `/var/lib/docker/containers/` grew to 80 GB, and the disk ran out in the middle of the night. Everything went down: the containers and Docker itself.

### Solution
Limit the log size for each container:

```yaml
services:
  api:
    logging:
      driver: json-file
      options:
        max-size: "10m"    # One file no larger than 10 MB
        max-file: "3"      # Maximum of 3 files
```

**Total:** up to 30 MB per container. Old files are deleted automatically.

For most services, 30 MB = a few hours of logs, which is enough for on-the-spot diagnostics. If you need a week of history:
```yaml
max-size: "50m"
max-file: "10"  # = 500 MB
```

### Global configuration
You can set this for all containers via `/etc/docker/daemon.json`:

```json
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "5"
  }
}
```

After changing it: `sudo systemctl restart docker`

### Checking current log size
```bash
du -sh /var/lib/docker/containers/*/*-json.log | sort -h
```

If you see files of 5–10 GB — rotation isn't configured.

---

## 4. Health Checks

### Problem
Docker doesn't know **whether your application is working**. It knows the process is running (the PID exists), but not whether the service responds.

The application might:
- Hang
- Run out of DB connections
- Enter an infinite loop
- Boot up but not respond to requests

Process alive → Docker considers the container running.

### Solution
Add a `healthcheck`:

```yaml
services:
  api:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s      # Check every 30 seconds
      timeout: 5s        # Check timeout
      retries: 3         # How many retries before unhealthy
      start_period: 10s  # Initialization time after startup
```

**Parameters:**
- `test` — the check command
- `interval` — how often to check
- `timeout` — maximum time for the command to run
- `retries` — how many consecutive failures = unhealthy
- `start_period` — grace period after startup (failures don't count)

### start_period: why it's needed
The application needs time to initialize:
- Loading configuration
- Warming up the cache
- Connecting to the database

Without `start_period`, Docker might decide the container is unhealthy while it's still starting up.

### Combining with depends_on
The most useful part — guaranteeing startup order:

```yaml
services:
  api:
    depends_on:
      db:
        condition: service_healthy
      redis:
        condition: service_healthy
```

The application only starts **once Postgres and Redis have passed their healthchecks**.

Without this, the application might start before the database and crash on the first request.

### Examples for different services

**Postgres:**
```yaml
db:
  healthcheck:
    test: ["CMD-SHELL", "pg_isready -U postgres"]
    interval: 10s
    timeout: 5s
    retries: 5
```

**HTTP endpoint (if there's no curl in the image):**
```yaml
api:
  healthcheck:
    test: ["CMD-SHELL", "wget -q --spider http://localhost:8080/health || exit 1"]
```

**Next.js:**
```yaml
fe-nextjs:
  healthcheck:
    test: ["CMD", "node", "-e", "fetch('http://127.0.0.1:3000/api/health').then(r=>process.exit(r.ok?0:1))"]
    start_period: 30s  # Next.js compiles routes on first access
```

---

## 5. Volume backups

### Problem
Named volumes store data across restarts. Physically, that's a folder on the host under `/var/lib/docker/volumes/`.

**Risks:**
- The host dies → data lost
- Someone accidentally runs `docker compose down -v` → data lost
- The disk fails → data lost

### Solution: automatic pg_dump

A minimal backup — run `pg_dump` every 24 hours:

```yaml
services:
  postgres-backup:
    image: postgres:16-alpine
    depends_on:
      db:
        condition: service_healthy
    volumes:
      - ./backups:/backups
    entrypoint: >
      sh -c "while true; do
        PGPASSWORD=$$POSTGRES_PASSWORD pg_dump -h db -U postgres mydb |
        gzip > /backups/backup_$$(date +%Y%m%d_%H%M%S).sql.gz;
        find /backups -mtime +7 -delete;
        sleep 86400;
      done"
    environment:
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    restart: unless-stopped
```

**What it does:**
- Every 24 hours: dump → gzip compression → delete dumps older than 7 days
- The `./backups` folder needs to be synced to S3 or another server

### Backup on the same host
This protects against `docker compose down -v`, but **does not protect against hardware failure**.

Make sure to set up syncing backups to remote storage:
- S3 (AWS, MinIO, Yandex Object Storage)
- Another server via rsync
- Cloud backup (Backblaze B2, Google Cloud Storage)

### Redis persistence
If Redis is used as a **cache** — no backup is needed.

If it's used as **primary storage** — enable persistence:
```yaml
redis:
  command: redis-server --save 60 1000 --appendonly yes
  volumes:
    - redisdata:/data
```

---

## Production checklist

- [ ] **Memory and CPU limits** for all services
- [ ] **Postgres shared_buffers** configured (25% of the memory limit)
- [ ] **Restart policy: unless-stopped** for all long-running services
- [ ] **Log rotation** (max-size + max-file)
- [ ] **Healthcheck** for all critical services (api, db, frontend)
- [ ] **depends_on with condition: service_healthy** wherever startup order matters
- [ ] **Automatic backup** of volumes with important data
- [ ] **Syncing backups** to remote storage

---

## Useful commands

### Checking for OOM kills
```bash
docker inspect <container> --format='{{.State.OOMKilled}}'
```

### Log size across all containers
```bash
du -sh /var/lib/docker/containers/*/*-json.log | sort -h
```

### Healthcheck status
```bash
docker ps --format "table {{.Names}}\t{{.Status}}"
```

### Resource usage
```bash
docker stats --no-stream
```

### Restoring from a backup
```bash
gunzip -c backup_20260101_120000.sql.gz | \
  docker compose exec -T db psql -U postgres -d mydb
```

---

## Additional practices

### 1. Don't use :latest in production
```yaml
# ❌ Bad
image: postgres:latest

# ✅ Good
image: postgres:19.3-alpine
```

### 2. Use read-only where possible
```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock:ro
  - ./nginx.conf:/etc/nginx/conf.d/default.conf:ro
```

### 3. Limit the number of restarts
```yaml
restart: unless-stopped
deploy:
  restart_policy:
    condition: on-failure
    delay: 5s
    max_attempts: 3
    window: 120s
```

### 4. Monitoring and alerts
Set up alerts for:
- Container unhealthy for more than 5 minutes
- OOMKilled = true
- Restarts more than 3 times per hour
- Memory usage > 90% of the limit

### 5. Labels for organization
```yaml
labels:
  com.example.project: "avi"
  com.example.environment: "production"
  com.example.version: "${APP_VERSION}"
```

---

## References

- [Docker Compose Deploy specification](https://docs.docker.com/compose/compose-file/deploy/)
- [Docker logging drivers](https://docs.docker.com/config/containers/logging/configure/)
- [Container healthchecks](https://docs.docker.com/engine/reference/builder/#healthcheck)
- [Habr article](https://habr.com/ru/companies/otus/articles/1034390/)
