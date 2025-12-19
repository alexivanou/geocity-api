# Contributing to GeoCity

First off, thanks for taking the time to contribute! 🎉

Whether you are a seasoned Go developer or just starting out, we welcome your contributions. 
This document will guide you through setting up your development environment and submitting a pull request.

## 🛳 Development Setup

### Prerequisites
1. **Go 1.23+**: Ensure you have a recent version of Go installed.
2. **Make**: Required to run automation scripts.
3. **SQLite**: Required for running tests locally (usually pre-installed on most OS).

### Steps
1. **Fork and Clone** the repository.
2. **Download Test Data**:
   ```bash
   make download-data
   ```
3. **Run Tests** to ensure everything is working:
   ```bash
   make test
   ```

## 📐 Coding Standards

### Style
- We use standard `gofmt`.
- We use `golangci-lint` for linting.
- Comments should explain *why* something is done, not *what* is done (unless complex).

### Architecture
This project follows a **Layered Architecture**:
1. **Handler (`internal/api`)**: Parses HTTP requests, calls Service. Returns JSON.
2. **Service (`internal/service`)**: Business logic. No SQL here.
3. **Repository (`internal/repository`)**: Database interactions. No HTTP logic here.

Please respect these boundaries when adding new features.

### Database
- This project supports both **PostgreSQL** (Production) and **SQLite** (Dev/Test).
- If you write a raw SQL query, ensure it is compatible with both, 
or implement separate methods in `pg_impl.go` and `sqlite_impl.go`.

## 🗪 Testing

- **Unit Tests**: Place them next to the code (e.g., `handler_test.go`). Use `testify/assert` and mocks.
- **Integration Tests**: See `internal/api/integration_test.go`. 
These spin up an in-memory SQLite DB to test the full flow.

Run specific tests:
```bash
go test -v ./internal/service/...
```

## 📩 Submitting a Pull Request

1. Create a new branch: `git checkout -b feature/my-new-feature`.
2. Commit your changes: `git commit -m 'feat: add support for Mars coordinates'`.
   - Please follow [Conventional Commits](https://www.conventionalcommits.org/).
3. Push to the branch: `git push origin feature/my-new-feature`.
4. Submit a Pull Request!

## 🐛 Reporting Bugs

Please include:
1. The version of the app.
2. Steps to reproduce.
3. Expected vs Actual behavior.