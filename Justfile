# Justfile for GoGlowControl project

# Default recipe
default:
    @echo "GoGlowControl - Control Yeelight devices"
    @echo "Available recipes:"
    @echo "  - just build: Build the project"
    @echo "  - just test: Run all tests"
    @echo "  - just run: Run the application"
    @echo "  - just install: Install dependencies"
    @echo "  - just clean: Remove build artifacts"
    @echo "  - just build-run: Build and run the application"
    @echo "  - just test-coverage: Run tests with coverage report"
    @echo "  - just fmt: Format the code"
    @echo "  - just vet: Vet the code for common issues"
    @echo "  - just lint: Lint the code"
    @echo "  - just lint-fix: Lint the code and apply auto-fixes"

# Build the project
build:
    @echo "Building GoGlowControl..."
    go build -o light ./src

# Run tests
test:
    @echo "Running tests..."
    GOCACHE="$(pwd)/.gocache" go test -v ./...

# Run the application
run:
    @echo "Running GoGlowControl..."
    go run ./src

# Install dependencies
install:
    @echo "Installing dependencies..."
    go mod tidy

# Clean build artifacts
clean:
    @echo "Cleaning build artifacts..."
    rm -rf light .gocache coverage.out coverage.html
    go clean ./...

# Build and run in one command
build-run:
    @echo "Building and running GoGlowControl..."
    just build
    ./light

# Run tests with coverage
test-coverage:
    @echo "Running tests with coverage..."
    GOCACHE="$(pwd)/.gocache" go test -v -coverprofile=coverage.out ./...
    GOCACHE="$(pwd)/.gocache" go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
    @echo "Formatting code..."
    go fmt ./...

# Vet code
vet:
    @echo "Vetting code..."
    go vet ./...

# Lint code
lint:
    @echo "Linting code..."
    @golangci-lint run --no-config -E errcheck -E govet -E ineffassign -E staticcheck -E unused ./...

# Lint with auto-fix
lint-fix:
    @echo "Linting code with auto-fix..."
    @golangci-lint run --no-config --fix -E errcheck -E govet -E ineffassign -E staticcheck -E unused ./...
