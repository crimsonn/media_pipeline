PROTO_DIR  := api/proto
PB_DIR     := pkg/pb
GO_MODULE  := github.com/crimsonn/media_pipeline
PROTO_FILES := $(shell find $(PROTO_DIR) -name '*.proto')
BIN_DIR  := bin
SERVICES := watcher transcoder api


.PHONY: proto
proto:
	@mkdir -p $(PB_DIR)
	protoc -I $(PROTO_DIR) \
		--go_out=. --go_opt=module=$(GO_MODULE) \
		--go-grpc_out=. --go-grpc_opt=module=$(GO_MODULE) \
		$(PROTO_FILES)

.PHONY: proto-clean
proto-clean:
	rm -rf $(PB_DIR)

.PHONY: run-watcher
run-watcher:
	go run cmd/watcher/main.go

.PHONY: run-transcoder
run-transcoder:
	go run cmd/transcoder/main.go

.PHONY: api
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