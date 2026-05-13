.PHONY: build build-cli build-wasm build-binary test test-integration proto clean

build:
	go build -o bin/leanimt-plus-server ./cmd/leanimt-plus-server

build-cli:
	go build -o bin/leanimt-plus-build ./cmd/leanimt-plus-build

build-wasm:
	GOOS=js GOARCH=wasm go build -o wasm/leanimt-plus.wasm ./wasm/
	@WASM_JS="$$(go env GOROOT)/lib/wasm/wasm_exec.js"; \
	if [ ! -f "$$WASM_JS" ]; then WASM_JS="$$(go env GOROOT)/misc/wasm/wasm_exec.js"; fi; \
	cp "$$WASM_JS" wasm/

build-binary: build-cli
	./bin/leanimt-plus-build --binary

benchmark: build-cli build-wasm
	@echo "Converting G2 JSON snapshot to binary..."
	./bin/leanimt-plus-build --convert-binary data/g2/tree-snapshot.json.gz
	cp data/g2/tree-snapshot.bin.gz wasm/
	@echo ""
	@echo "=== Ready ==="
	@echo "Serving at http://localhost:8080/benchmark.html"
	@echo "Press Ctrl+C to stop"
	@cd wasm && python3 -m http.server 8080

test:
	go test ./... -v

test-integration:
	go test -tags integration ./internal/api/ -run TestIntegration -v -timeout 10m
	go test -tags integration ./internal/crl/... -v -timeout 30m

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		pkg/proto/revocation/revocation.proto

clean:
	rm -rf bin/

run: build
	./bin/leanimt-plus-server
