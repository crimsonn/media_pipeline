PROTO_DIR := api/proto
PB_DIR    := pkg/pb

.PHONY: proto
proto:
	@mkdir -p $(PB_DIR)
	protoc -I $(PROTO_DIR) \
		--go_out=$(PB_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(PB_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/*.proto

.PHONY: proto-clean
proto-clean:
	rm -f $(PB_DIR)/*.pb.go
