.PHONY: build-frontend build-backend build clean run

BINARY_NAME=behringer-recorder
FRONTEND_DIR=frontend
STATIC_DIR=static

build: build-frontend build-backend

build-frontend:
	@echo "Building frontend..."
	cd $(FRONTEND_DIR) && npm install && npm run build
	mkdir -p $(STATIC_DIR)
	cp -r $(FRONTEND_DIR)/dist/* $(STATIC_DIR)/

build-backend:
	@echo "Building backend..."
	go build -o $(BINARY_NAME) main.go

clean:
	@echo "Cleaning..."
	rm -rf $(BINARY_NAME) $(STATIC_DIR)/* $(FRONTEND_DIR)/dist $(FRONTEND_DIR)/node_modules

run: build-backend
	./$(BINARY_NAME)
