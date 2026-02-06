.PHONY: build build-linux release run debug test test-race vet lint clean

APP_NAME := lazyfirewall
CMD_PATH := ./cmd/lazyfirewall
BIN_DIR ?= bin
DIST_DIR ?= dist

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X lazyfirewall/internal/version.Version=$(VERSION) \
	-X lazyfirewall/internal/version.Commit=$(COMMIT) \
	-X lazyfirewall/internal/version.Date=$(DATE)
GO_BUILD_FLAGS := -trimpath -ldflags "$(LDFLAGS)"

build: build-linux

build-linux:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(BIN_DIR)/$(APP_NAME) $(CMD_PATH)

release:
	bash ./scripts/build-release.sh

run: build-linux
	sudo ./$(BIN_DIR)/$(APP_NAME)

debug: build-linux
	LAZYFIREWALL_DEBUG=1 sudo ./$(BIN_DIR)/$(APP_NAME)

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

lint: vet
	@unformatted="$$(gofmt -l $$(git ls-files '*.go'))"; \
	if [ -n "$$unformatted" ]; then \
		echo "Unformatted files:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

clean:
	@if [ -d "$(BIN_DIR)" ]; then rm -rf "$(BIN_DIR)"; fi
	@if [ -d "$(DIST_DIR)" ]; then rm -rf "$(DIST_DIR)"; fi
	@find . -maxdepth 1 -type f -name '*.test' -delete
	@rm -f $(APP_NAME)
