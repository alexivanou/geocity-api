.PHONY: help download-data migrate migrate-down seed test build run stats clean

help:
	@echo "Available targets:"
	@echo "  download-data - Download GeoNames data files"
	@echo "  migrate       - Run database migrations"
	@echo "  seed          - Load data into database"
	@echo "  test          - Run tests"
	@echo "  build         - Build application"
	@echo "  run           - Run application"
	@echo "  clean         - Clean build artifacts"

DATA_DIR := data

download-data:
	@echo "Downloading GeoNames data..."
	@mkdir -p $(DATA_DIR)
	@curl -L -o $(DATA_DIR)/cities1000.zip https://download.geonames.org/export/dump/cities1000.zip
	@curl -L -o $(DATA_DIR)/alternateNames.zip https://download.geonames.org/export/dump/alternateNames.zip
	@curl -L -o $(DATA_DIR)/countryInfo.txt https://download.geonames.org/export/dump/countryInfo.txt
	@echo "Data downloaded to $(DATA_DIR)/"

migrate:
	@echo "Running migrations..."
	@go run ./cmd/migrate -command=up

migrate-down:
	@echo "Rolling back migrations..."
	@go run ./cmd/migrate -command=down

migrate-version:
	@echo "Checking migration version..."
	@go run ./cmd/migrate -command=version

seed:
	@echo "Seeding database..."
	@go run ./cmd/seeder

test:
	@echo "Running tests..."
	@go test -v ./...

test-cover:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

build:
	@echo "Building application..."
	@go build -o bin/app ./cmd/app
	@go build -o bin/seeder ./cmd/seeder
	@go build -o bin/stats ./cmd/stats
	@go build -o bin/migrate ./cmd/migrate

run:
	@go run ./cmd/app

stats:
	@echo "Collecting statistics..."
	@go run ./cmd/stats

stats-json:
	@echo "Collecting statistics (JSON format)..."
	@OUTPUT_FORMAT=json go run ./cmd/stats

clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

collect:
	CGO_LDFLAGS="-lm" go run collect.go
	