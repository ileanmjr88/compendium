VERSION ?= $(shell cat VERSION 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD)

export CGO_ENABLED := 0

LDFLAGS := -ldflags "\
	-X github.com/ileanmjr88/compendium/internal/buildinfo.Version=$(VERSION) \
	-X github.com/ileanmjr88/compendium/internal/buildinfo.Commit=$(COMMIT)"

.PHONY: build test coverage run fmt vet lint clean release release-check help

# Build the Go application
build:
	go build $(LDFLAGS) -o bin/compendium ./cmd/compendium

# Run tests
test:
	go test -v ./...

# Test coverage report. The cmd/compendium tests exec a subprocess binary built
# with `-cover`, which writes covdata to a temp dir and is converted by TestMain
# into cli-coverage.out. Merge it into coverage.out so subprocess-exercised code
# (internal/cli, cmd/compendium) shows up in the report.
coverage:
	go test -coverprofile=coverage.out ./...
	@if [ -f cli-coverage.out ]; then \
	    head -1 coverage.out > coverage.merged.out; \
	    tail -n +2 coverage.out     >> coverage.merged.out; \
	    tail -n +2 cli-coverage.out >> coverage.merged.out; \
	    mv coverage.merged.out coverage.out; \
	    rm cli-coverage.out; \
	fi
	go tool cover -func=coverage.out

# Run the application
run:
	go run ./cmd/compendium

# Format
fmt:
	gofmt -w .

# Vet
vet:
	go vet ./...

# Lint
lint:
	golangci-lint run

# Local goreleaser dry run. Cross-compiles darwin+linux × amd64+arm64 into
# dist/, produces tarballs and checksums.txt, but does not publish. Use this
# to verify a release will succeed before tagging. The real release runs in
# CI on tag push (see .github/workflows/release.yml).
release:
	goreleaser release --snapshot --clean

# Validate .goreleaser.yml without building anything.
release-check:
	goreleaser check

# Clean build artifacts
clean:
	rm -rf bin/ dist/
	go clean

# Display help
help:
	@echo "Available targets:"
	@echo "  make build   - Build the Go application"
	@echo "  make test    - Run tests"
	@echo "  make run     - Run the application"
	@echo "  make fmt     - Format the Go code"
	@echo "  make vet     - Vet the Go code"
	@echo "  make lint    - Lint the Go code"
	@echo "  make coverage      - Test coverage report"
	@echo "  make release       - Local goreleaser dry run (cross-compile, no publish)"
	@echo "  make release-check - Validate .goreleaser.yml"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make help          - Display this help message"
