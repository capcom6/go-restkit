.PHONY: all lint test benchmark deps

# Default target
all: fmt lint test benchmark

fmt:
	golangci-lint fmt

# Lint the code using golangci-lint
lint:
	golangci-lint run --timeout=5m

# Run tests with coverage
test:
	go test -race -shuffle=on -count=1 -covermode=atomic -coverpkg=./... -coverprofile=coverage.out ./...

# Run benchmarks
benchmark:
	go test -run=^$$ -bench=. -benchmem ./... | tee benchmark.txt

# Download dependencies
deps:
	go mod download