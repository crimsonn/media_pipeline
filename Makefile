BIN_DIR  := bin
SERVICES := watcher transcoder api

.PHONY: run-watcher
run-watcher:
	go run cmd/watcher/main.go

.PHONY: run-transcoder
run-transcoder:
	go run cmd/transcoder/main.go

.PHONY: run-api
run-api:
	go run cmd/api/main.go

.PHONY: build
build:
	@mkdir -p $(BIN_DIR)
	@for s in $(SERVICES); do go build -o $(BIN_DIR)/$$s ./cmd/$$s || exit 1; done

.PHONY: run-all
run-all: build
	goreman -f Procfile -set-ports=false start

.PHONY: sqlc
sqlc:
	sqlc generate

.PHONY: tools
tools:
	go install github.com/mattn/goreman@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
