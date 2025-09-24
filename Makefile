.PHONY: build install-tools check-tools proto lint test clean

build:
	go build -o bin/controller cmd/controller/main.go
	go build -o bin/cli cmd/cli/main.go

install-tools:
	@echo "Installing protoc-gen-go..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@echo "Installing protoc-gen-go-grpc..."
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed!"

check-tools:
	@which protoc > /dev/null || (echo "protoc not found. install protobuf compiler" && exit 1)
	@which protoc-gen-go > /dev/null || (echo "protoc-gen-go not found. Run 'make install-tools'" && exit 1)
	@which protoc-gen-go-grpc > /dev/null || (echo "protoc-gen-go-grpc not found. Run 'make install-tools'" && exit 1)
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Run 'make install-tools'" && exit 1)
	@echo "All tools found!"

proto:
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
	        --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	        api/proto/controlplane.proto
	@echo "Done!"

lint:
	@echo "Running golangci-lint..."
	@golangci-lint run --timeout=2m ./...
	@echo "Lint passed!"

test:
	@echo "Running tests..."
	@go test -v -race -cover ./...
	@echo "Tests completed!"

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@echo "Clean completed!"

build-fast:
	@echo "Fast build (skipping lint)..."
	@go build -o bin/controller cmd/controller/main.go
	@go build -o bin/cli cmd/cli/main.go
	@echo "Fast build completed!"
