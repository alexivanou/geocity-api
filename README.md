# GeoCity API 🌍

![Go Version](https://img.shields.io/badge/go-1.23-blue)
![License](https://img.shields.io/badge/license-Custom-green)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen)

**GeoCity** is a high-performance, self-hosted microservice written in Go for querying global city data, 
finding nearest cities by coordinates, and handling localized geographical names.

It parses the massive [GeoNames](https://www.geonames.org/) dataset and serves it via a REST JSON API with sub-millisecond response 
times using in-memory SQLite (for development) or PostgreSQL (for production).

## � Features

- 🚀 **Blazing Fast**: Optimized for speed using efficient SQL queries and proper indexing.
- 🗺️ **Geospatial Search**: Find the nearest city to any GPS coordinate.
- 📣 **Multilingual**: Supports localized city names (e.g., search "Moskau" or "Moscow" -> returns ID 524901).
- 📦 **Zero Dependency Dev**: Runs out-of-the-box with embedded SQLite and auto-migrations.
- 🐳 **Production Ready**: Includes Docker composition and PostgreSQL support with fuzzy search extensions.
- 📊 **Statistics**: Built-in runtime and database statistics collector.

## 🚀 Quick Status (For Beginners)

You don't need to be a Go expert to run this. We have automated everything.

### Prerequisites
- Docker & Docker Compose **OR** Go 1.23+

### Option 1: Run with Docker (Recommended)

1. **Status the stack:**
   ```bash
   docker-compose up -d
   ```
   *The app will automatically download the necessary GeoNames data, seed the database, and start the server.*

2. **Test it:**
   ```bash
   curl "http://localhost:8080/api/v1/suggest?q=Berli"
   ```

### Option 2: Run Locally (Go)

1. **Download Data:**
   ```bash
   make download-data
   ```
2. **Run the App:**
   ```bash
   go run cmd/app/main.go
   ```
   *The app detects an empty database and automatically seeds it from the `data/` folder.*

## API Usage

### 1. Suggest Cities
Search for cities by name (supports partial matching and translations).

**Request:**
`GET /api/v1/suggest?q=Par&lang=en&limit=5`

**Response:**
```json
{
  "results": [
    {
      "id": 2988507,
      "name": "Paris",
      "country": "France",
      "country_code": "FR",
      "population": 2138551
    }
  ]
}
```

### 2. Find Nearest City
Get the closest city to a specific latitude/longitude.

**Request:**
`GET /api/v1/nearest?lat=40.71&lon=-74.00`

### 3. Get City Details
Get full data including timezone, elevation, and population.

**Request:**
`GET /api/v1/city/2988507`

## ⚙ Configuration

The application is configured via Environment Variables.

| Variable | Default | Description |
|----------|---------|-------------|
| `APP_PORT` | `8080` | Port to listen on |
| `DB_TYPE` | `memory` | `postgres` or `memory` (SQLite) |
| `DB_HOST` | `localhost` | Database host |
| `DB_PORT` | `5432` | Database port |
| `DB_USER` | `geocity` | Database user |
| `DB_PASSWORD` | `geocity_password` | Database password |
| `DB_NAME` | `geocity` | Database name |
| `SEEDER_BATCH_SIZE` | `10000` | Rows per SQL insert batch |
| `SEEDER_MIN_POPULATION` | `10000` | Import only cities larger than X |
| `SEEDER_ALLOWED_LANGUAGES`| *(Empty)*| Comma-separated (e.g. `en,ru,de`). Empty = all |

## 🗠 For Developers

### Project Structure
- `cmd/`: Entry points (API server, seeder CLI, migration tool).
- `internal/api/`: HTTP Handlers and Router.
- `internal/model/`: Domain structs.
- `internal/repository/`: Database access layer (Clean Architecture).
- `internal/service/`: Builness logic.
- `internal/seeder/`: High-performance parsing logic for GeoNames text files.

### Running Tests
```bash
make test
```

### Useful Commands
Check `Makefile` for all commands:
- `make build`: Compile binaries.
- `make clean`: Remove artifacts.
- `make stats`: Run the stats CLI tool.

## 📲 License
See [LICENSE](LICENSE) file.