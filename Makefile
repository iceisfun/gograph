.PHONY: build frontend clean test vet dev

# Production build: compile frontend then Go
build: frontend
	go build ./...

# Build frontend assets
frontend:
	cd frontend && npm install --silent && npm run build

# Development: frontend watch + Go server from disk
dev:
	@echo "Starting frontend watch in background..."
	cd frontend && npm run watch &
	@echo "Starting Go server (dev mode)..."
	go run ./examples/basic -dev -addr :8080

# Clean build artifacts
clean:
	rm -rf frontend/dist/bundle.js frontend/dist/bundle.js.map frontend/node_modules

# Run tests
test:
	go test ./...

# Vet and type-check
vet:
	go vet ./...
	cd frontend && npx tsc --noEmit
