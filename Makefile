# Foundry build helpers. Everything here works without changing any source file.
MODULE   := github.com/SandSoftwareSolutions/Foundry
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE     ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w -X $(MODULE)/internal/cli.version=$(VERSION) -X $(MODULE)/internal/cli.commit=$(COMMIT) -X $(MODULE)/internal/cli.date=$(DATE)

.PHONY: all build test race cover vet fmt fmt-check check lint vuln smoke templates dist install clean

all: check build

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o foundry ./cmd/foundry

test:
	go test ./...

race:
	go test -race ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "These files need gofmt:"; gofmt -l .; exit 1)

# What CI runs (the quick part).
check: fmt-check vet test

# Static analysis and known-vulnerability scan; they download the tools on first use.
lint:
	go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./...

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...

smoke: build
	FOUNDRY_BIN=$(CURDIR)/foundry scripts/smoke-test.sh

# Creates a project from every built-in template and runs that language's own tests on it.
templates: build
	FOUNDRY_BIN=$(CURDIR)/foundry scripts/test-templates.sh

# Cross-compiled archives and checksums in dist/ (see scripts/dist.sh).
dist:
	VERSION=$(VERSION) COMMIT=$(COMMIT) DATE=$(DATE) scripts/dist.sh

install:
	go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/foundry

clean:
	rm -rf foundry foundry.exe coverage.out dist/*.tar.gz dist/*.zip dist/*.deb dist/SHA256SUMS
