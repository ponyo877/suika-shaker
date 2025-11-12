.PHONY: wasm wasm-opt clean serve serve-https run dev optimize-assets

# Default target
all: wasm

# Build WebAssembly binary
wasm:
	GOOS=js GOARCH=wasm go build -o main.wasm

# Build optimized WebAssembly binary with TinyGo
wasm-opt:
	@echo "Building optimized WASM with TinyGo..."
	GOOS=js GOARCH=wasm tinygo build -o main.wasm -target wasm -opt=z -no-debug
	@if command -v wasm-opt >/dev/null 2>&1; then \
		echo "Optimizing WASM with wasm-opt..."; \
		wasm-opt -Oz main.wasm -o main.wasm.tmp && mv main.wasm.tmp main.wasm; \
	fi
	@ls -lh main.wasm
	@echo "Optimized WASM build complete!"

# Optimize assets (fonts, images, audio)
optimize-assets:
	@echo "Optimizing assets..."
	@echo "Converting fonts to subset TTF..."
	@if command -v pyftsubset >/dev/null 2>&1; then \
		pyftsubset internal/ui/fonts/Poppins-Bold.ttf \
			--output-file=internal/ui/fonts/Poppins-Bold-subset.ttf \
			--text="0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz:. " \
			--layout-features='*' --no-hinting; \
		pyftsubset internal/ui/fonts/Poppins-Regular.ttf \
			--output-file=internal/ui/fonts/Poppins-Regular-subset.ttf \
			--text="0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz:. " \
			--layout-features='*' --no-hinting; \
		echo "Font subsetting complete"; \
	else \
		echo "Warning: pyftsubset not found, skipping font optimization"; \
	fi
	@echo "Converting images to WebP..."
	@if command -v cwebp >/dev/null 2>&1; then \
		for file in assets/image/*.png; do \
			base=$$(basename "$$file" .png); \
			cwebp -q 90 -alpha_q 100 "$$file" -o "assets/image/$${base}.webp" 2>/dev/null; \
		done; \
		echo "Image conversion complete"; \
	else \
		echo "Warning: cwebp not found, skipping image optimization"; \
	fi
	@echo "Converting audio to OGG..."
	@if command -v ffmpeg >/dev/null 2>&1; then \
		ffmpeg -i assets/sound/background.wav -c:a libvorbis -q:a 5 assets/sound/background.ogg -y 2>/dev/null; \
		ffmpeg -i assets/sound/gameover.wav -c:a libvorbis -q:a 4 assets/sound/gameover.ogg -y 2>/dev/null; \
		ffmpeg -i assets/sound/join.wav -c:a libvorbis -q:a 4 assets/sound/join.ogg -y 2>/dev/null; \
		ffmpeg -i assets/sound/suikajoin.wav -c:a libvorbis -q:a 4 assets/sound/suikajoin.ogg -y 2>/dev/null; \
		echo "Audio conversion complete"; \
	else \
		echo "Warning: ffmpeg not found, skipping audio optimization"; \
	fi
	@echo "Asset optimization complete!"

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

# Start HTTPS server for WASM version (required for iOS)
serve-https: wasm
	@echo "Building HTTPS server..."
	@go build -o https-server ./cmd/server
	@echo "Starting HTTPS server..."
	@./https-server

# Development mode: rebuild and serve
dev: clean wasm serve

# Clean build artifacts
clean:
	rm -f main.wasm suika-shaker https-server server.crt server.key
	@echo "Cleaned build artifacts"

# Help message
help:
	@echo "Available targets:"
	@echo "  make wasm            - Build WebAssembly binary (main.wasm)"
	@echo "  make wasm-opt        - Build optimized WASM with TinyGo and wasm-opt"
	@echo "  make optimize-assets - Convert assets (fonts→subset, images→WebP, audio→OGG)"
	@echo "  make build           - Build native binary (suika-shaker)"
	@echo "  make run             - Run native version"
	@echo "  make serve           - Build WASM and start HTTP server"
	@echo "  make serve-https     - Build WASM and start HTTPS server (for iOS)"
	@echo "  make dev             - Clean, build, and serve (development mode)"
	@echo "  make clean           - Remove build artifacts"
	@echo "  make help            - Show this help message"
