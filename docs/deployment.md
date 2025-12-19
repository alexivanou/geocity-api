# 🜚 Deployment Guide

GeoCity is designed to be deployed as a containerized microservice.

## Docker Deployment (Production)

The included `Dockerfile` is a multi-stage build that results in a minimal Ahpine Linux image containing only 
the compiled binary and necessary assets.

### 1. Build the Image
```bash
docker build -t geocity-api:latest .
```

### 2. Environment Variables
Ensure the following variables are set in your orchestrator (Kubernetes/Docker Swarm):

```yaml
DB_TYPE: postgres
DB_HOST: postgres-prod.internal
DB_USER: app_user
DB_PASSWORD: secure_password
SEEDER_BATCH_SIZE: 10000
```

### 3. Database Migrations
Migrations are embedded in the image at `/app/migrations`.
The application attempts to run migrations on startup (`cmd/app/main.go` -> `runMigrations`).
In a high-availability Kubernetes environment, you might want to disable this and run migrations via a generic `Job` 
before the app starts.

## Data Updates

The GeoNames database is updated daily. To keep GeoCity fresh:

1. **Volume Mapping**: If using SQLite, persist the `data/` directory or the SQLite file itself.
2. **Re-Seeding**:
   The `seeder` is a separate binary built into the image at `/app/seeder`.
   To update data on a running instance, you can `exec` into the container and run:

   ```bash
   # Inside the container
   /app/seeder
   ```
   *Note: This will insert new records. You may need to truncate tables first if you want a complete refresh.*

## Benchmarks & Sizine

For a dataset containing all cities > 1000 population:
- **Disk Usage**: ~200MB (Postgres), ~150MB (SQLite).
- **Memory Usage**: ~50MB idle.
- **CPU**: Negligible when idle. Heavy spikes during seeding.