.PHONY: test postgres_start postgres_stop postgres_load clean build install

# PostgreSQL settings
PG_PORT := 9875
PG_USER := postgres
PG_PASSWORD := postgres
PG_DB := dbinfo_test
PG_CONTAINER := dbinfo-postgres-test
DSN := "postgres://$(PG_USER):$(PG_PASSWORD)@localhost:$(PG_PORT)/$(PG_DB)?sslmode=disable"

# Build settings
BINARY_NAME := dbinfo
BUILD_DIR := build

# Test command
test: postgres_start postgres_load
	@echo "Running tests..."
	@TEST_POSTGRES_DSN=$(DSN) go test -v ./...
	@make postgres_stop

# Start PostgreSQL in a Docker container
postgres_start:
	@echo "Starting PostgreSQL container..."
	@docker rm -f $(PG_CONTAINER) 2>/dev/null || true
	@docker run --name $(PG_CONTAINER) \
		-e POSTGRES_USER=$(PG_USER) \
		-e POSTGRES_PASSWORD=$(PG_PASSWORD) \
		-e POSTGRES_DB=$(PG_DB) \
		-p $(PG_PORT):5432 \
		-d postgres:14-alpine
	@echo "Waiting for PostgreSQL to start..."
	@sleep 3

# Stop and remove the PostgreSQL container
postgres_stop:
	@echo "Stopping PostgreSQL container..."
	@docker stop $(PG_CONTAINER) || true
	@docker rm $(PG_CONTAINER) || true

# Load test fixtures into PostgreSQL
postgres_load:
	@echo "Loading test fixtures..."
	@docker exec -i $(PG_CONTAINER) psql -U $(PG_USER) -d $(PG_DB) < fixture.sql
	@echo "Test fixtures loaded successfully."

# Build the dbinfo command
build:
	@echo "Building dbinfo command..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/dbinfo
	@echo "Binary created: $(BUILD_DIR)/$(BINARY_NAME)"

# Install the dbinfo command to $GOPATH/bin
install:
	@echo "Installing dbinfo command..."
	@go install ./cmd/dbinfo
	@echo "Binary installed to $$GOPATH/bin/$(BINARY_NAME)"

# Clean up
clean:
	@echo "Cleaning up..."
	@make postgres_stop
	@rm -rf $(BUILD_DIR)
	@echo "Build directory removed."