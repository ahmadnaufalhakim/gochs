APP_NAME    := gochs
MODULE_PATH := github.com/ahmadnaufalhakim/gochs

GIT_COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_AUTHOR  := $(shell git log -1 --format='%an' 2>/dev/null || echo "unknown")
BUILD_DATE  := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
VERSION     := $(shell git describe --tags 2>/dev/null || echo "dev")

LDFLAGS := -ldflags "\
    -X main.version=$(VERSION) \
    -X main.commit=$(GIT_COMMIT) \
    -X main.date=$(BUILD_DATE) \
    -X 'main.author=$(GIT_AUTHOR)'"

.PHONY: dev download build run test tidy lint migrate migrate-new

dev:
	@command -v air >/dev/null 2>&1 || { echo "Missing: air - install with 'go install github.com/air-verse/air@latest'"; exit 1; }
	air

build:
	mkdir -p build
	go build $(LDFLAGS) -o build/$(APP_NAME) ./cmd/gochs

run:
	go run ./cmd/gochs

test:
	go test ./... -v

tidy:
	go mod tidy

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Missing: golangci-lint — install from https://golangci-lint.run/"; exit 1; }
	golangci-lint run ./...

migrate:
	./scripts/migrate.sh up

migrate-new:
	@read -p "Migration name: " name; ./scripts/migrate.sh new $$name
