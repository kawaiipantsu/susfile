# susfile — build automation. See PROJECT.md for the target contract.

BINARY  := susfile
MODULE  := github.com/kawaiipantsu/susfile
PKG     := $(MODULE)/internal/version

VERSION ?= $(shell sed -n 's/^## \[\([0-9][^]]*\)\].*/\1/p' CHANGELOG.md 2>/dev/null | head -1)
VERSION := $(if $(VERSION),$(VERSION),0.1.0-dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DIRTY   := $(shell test -n "$$(git status --porcelain 2>/dev/null)" && echo true || echo false)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X '$(PKG).Version=$(VERSION)' \
	-X '$(PKG).Commit=$(COMMIT)' \
	-X '$(PKG).Dirty=$(DIRTY)'
ifndef SOURCE_DATE_EPOCH
LDFLAGS += -X '$(PKG).Date=$(DATE)'
endif

GO    ?= go
DIST  := dist
COVER := coverage.out
CMD   := ./cmd/susfile

# Linux-only release matrix: intel + arm, 32 + 64 bit.
LINUX_ARCHES := amd64 386 arm64 arm

.DEFAULT_GOAL := help

## ---------------------------------------------------------------- help

.PHONY: help
help: ## Show available targets
	@echo "Development:"
	@echo "  run                Run susfile (ARGS=... to pass arguments)"
	@echo "  deps               Download and verify modules"
	@echo "  fmt / fmt-check    Format code / fail if unformatted"
	@echo "  vet / lint         go vet / golangci-lint when installed"
	@echo "  test / race        Run the test suite / with the race detector"
	@echo "  bench / coverage   Benchmarks / HTML coverage report"
	@echo ""
	@echo "Build:"
	@echo "  build              Build the host binary"
	@echo "  build-linux        Build all four Linux targets ($(LINUX_ARCHES))"
	@echo "  build-all          Alias for build-linux"
	@echo "  install            go install into GOPATH/bin"
	@echo "  clean              Remove generated files"
	@echo ""
	@echo "Release (see feature/build-system):"
	@echo "  dist deb snapshot release-check security"
	@echo ""
	@echo "Version: $(VERSION)  Commit: $(COMMIT)"

## ---------------------------------------------------------------- dev

.PHONY: deps
deps: ## Download and verify modules
	$(GO) mod download
	$(GO) mod verify

.PHONY: fmt
fmt: ## Format code
	$(GO) fmt ./...

.PHONY: fmt-check
fmt-check: ## Fail when code is not gofmt-clean
	@unformatted=$$(gofmt -l . 2>/dev/null || true); \
	if [ -n "$$unformatted" ]; then echo "not gofmt'd:"; echo "$$unformatted"; exit 1; fi

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: lint
lint: ## Run golangci-lint when installed, else go vet
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed; running go vet"; $(GO) vet ./...; \
	fi

.PHONY: test
test: ## Run the full test suite
	$(GO) test ./...

.PHONY: race
race: ## Run tests with the race detector
	$(GO) test -race ./...

.PHONY: bench
bench: ## Run benchmarks
	$(GO) test -run '^$$' -bench=. -benchmem ./...

.PHONY: coverage
coverage: ## Generate an HTML coverage report
	$(GO) test -coverprofile=$(COVER) -covermode=atomic ./...
	$(GO) tool cover -func=$(COVER) | tail -1
	$(GO) tool cover -html=$(COVER) -o coverage.html
	@echo "wrote coverage.html"

.PHONY: run
run: ## Run susfile (ARGS=... to pass arguments)
	$(GO) run -ldflags "$(LDFLAGS)" $(CMD) $(ARGS)

## ---------------------------------------------------------------- build

.PHONY: build
build: ## Build the host binary
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

# $(1) = GOARCH
define build_linux
	@echo "building linux/$(1)"
	@mkdir -p $(DIST)/$(BINARY)_$(VERSION)_linux_$(1)
	CGO_ENABLED=0 GOOS=linux GOARCH=$(1) $(if $(filter arm,$(1)),GOARM=7,) \
		$(GO) build -trimpath -ldflags "$(LDFLAGS)" \
		-o $(DIST)/$(BINARY)_$(VERSION)_linux_$(1)/$(BINARY) $(CMD)

endef

.PHONY: build-linux
build-linux: ## Build all four Linux targets
	$(foreach a,$(LINUX_ARCHES),$(call build_linux,$(a)))

.PHONY: build-all
build-all: build-linux ## Alias for build-linux

.PHONY: install
install: ## go install into GOPATH/bin
	CGO_ENABLED=0 $(GO) install -trimpath -ldflags "$(LDFLAGS)" $(CMD)

.PHONY: clean
clean: ## Remove generated files
	rm -rf $(DIST) $(BINARY) $(COVER) coverage.html
	$(GO) clean -cache -testcache >/dev/null 2>&1 || true

.PHONY: generate
generate: ## Run go generate
	$(GO) generate ./...

.PHONY: security
security: ## Run govulncheck when installed
	@if command -v govulncheck >/dev/null 2>&1; then govulncheck ./...; \
	else echo "govulncheck not installed: go install golang.org/x/vuln/cmd/govulncheck@latest"; fi
