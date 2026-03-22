.PHONY: build-frontend build-backend build clean run test test-backend test-frontend

BINARY_NAME=behringer-recorder
FRONTEND_DIR=frontend
STATIC_DIR=static

build: build-frontend build-backend

build-frontend:
	@echo "Building frontend..."
	cd $(FRONTEND_DIR) && npm install && npm run build
	mkdir -p $(STATIC_DIR)
	rm -rf $(STATIC_DIR)/*
	cp -r $(FRONTEND_DIR)/static/* $(STATIC_DIR)/

build-backend:
	@echo "Generating Swagger docs..."
	go run github.com/swaggo/swag/cmd/swag@latest init
	@echo "Building backend..."
	go build -o $(BINARY_NAME) main.go

clean:
	@echo "Cleaning..."
	rm -rf $(BINARY_NAME) $(STATIC_DIR)/* $(FRONTEND_DIR)/dist $(FRONTEND_DIR)/node_modules coverage.out $(FRONTEND_DIR)/coverage


run:
	./$(BINARY_NAME)

test: test-backend test-frontend

test-backend:
	@echo "Running backend tests..."
	go test ./...

test-frontend:
	@echo "Running frontend unit tests..."
	cd $(FRONTEND_DIR) && npm run test:unit
	@echo "Running frontend E2E tests..."
	cd $(FRONTEND_DIR) && npx playwright test --project=chromium
