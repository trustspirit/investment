.PHONY: dev dev-server dev-client build build-server build-client clean

# Development
dev:
	@make -j2 dev-server dev-client

dev-server:
	cd server && go run ./cmd/api

dev-client:
	cd client && npm run dev

# Build
build: build-server build-client

build-server:
	cd server && go build -o ../bin/api ./cmd/api

build-client:
	cd client && npm run build

# Clean
clean:
	rm -rf bin/ client/dist/

# Install dependencies
install:
	cd server && go mod tidy
	cd client && npm install
