.PHONY: wasm clean serve run dev

# Default target
all: wasm

# Build WebAssembly binary
wasm:
	GOOS=js GOARCH=wasm go build -o main.wasm

# Build native binary
build:
	go build -o suika-shaker

# Run native version
run:
	go run .

# Start HTTP server for WASM version
serve: wasm
	@echo "Starting server at http://localhost:8080"
	@echo "Press Ctrl+C to stop"
	python3 -m http.server 8080

# Development mode: rebuild and serve
dev: clean wasm serve

# Clean build artifacts
clean:
	rm -f main.wasm suika-shaker
	@echo "Cleaned build artifacts"

# Help message
help:
	@echo "Available targets:"
	@echo "  make wasm    - Build WebAssembly binary (main.wasm)"
	@echo "  make build   - Build native binary (suika-shaker)"
	@echo "  make run     - Run native version"
	@echo "  make serve   - Build WASM and start HTTP server"
	@echo "  make dev     - Clean, build, and serve (development mode)"
	@echo "  make clean   - Remove build artifacts"
	@echo "  make help    - Show this help message"
