# Binary name produced by `go build`.
BINARY := mile-server

# Port for the server (override with: make run PORT=9000)
PORT ?= 8080

.PHONY: all build run fmt vet tidy test clean

# Default target.
all: build

# Compile the server.
build:
	go build -o $(BINARY) .

# Remove old binary, build fresh, and run the server.
run: clean build
	PORT=$(PORT) ./$(BINARY)

# Format all Go source files in place.
fmt:
	gofmt -w .

# Static analysis across the module.
vet:
	go vet ./...

# Prune/sync go.mod and go.sum.
tidy:
	go mod tidy

# Run all tests in the module.
test:
	go test ./...

# Delete the compiled binary.
clean:
	rm -f $(BINARY)
