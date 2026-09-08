PROTO_DIR  := api/proto
PB_DIR     := pkg/pb
GO_MODULE  := github.com/crimsonn/media_pipeline
PROTO_FILES := $(shell find $(PROTO_DIR) -name '*.proto')

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
