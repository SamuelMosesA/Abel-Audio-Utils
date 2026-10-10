PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
SYSCONFDIR ?= /etc/abel
DATADIR ?= $(PREFIX)/share/abel
SYSTEMDDIR ?= /etc/systemd/system

.PHONY: all build build-frontend sync-static build-backend install install-systemd enable-systemd uninstall clean test dev

all: build

build: build-frontend sync-static build-backend

build-frontend:
	@echo "==> Building SvelteKit frontend..."
	cd src/frontend && npm install && npm run build

sync-static:
	@echo "==> Syncing static assets..."
	mkdir -p src/backend/static
	rm -rf src/backend/static/*
	cp -r src/frontend/build/* src/backend/static/

build-backend:
	@echo "==> Compiling Abel backend..."
	mkdir -p bin
	go build -o bin/abel src/backend/main.go

install: build
	@echo "==> Installing Abel to $(PREFIX)..."
	install -d $(DESTDIR)$(BINDIR)
	install -m 755 bin/abel $(DESTDIR)$(BINDIR)/abel
	install -m 755 scripts/abel-service.sh $(DESTDIR)$(BINDIR)/abel-service
	install -d $(DESTDIR)$(DATADIR)
	install -m 644 docker-compose.yaml $(DESTDIR)$(DATADIR)/docker-compose.yaml
	cp -r observability $(DESTDIR)$(DATADIR)/
	install -d $(DESTDIR)$(SYSCONFDIR)
	@if [ ! -f $(DESTDIR)$(SYSCONFDIR)/config.yaml ]; then \
		install -m 644 config/config-example.yaml $(DESTDIR)$(SYSCONFDIR)/config.yaml; \
		echo "Installed default config to $(DESTDIR)$(SYSCONFDIR)/config.yaml"; \
	else \
		echo "Preserving existing config at $(DESTDIR)$(SYSCONFDIR)/config.yaml"; \
	fi

install-systemd:
	@echo "==> Installing systemd service..."
	install -d $(DESTDIR)$(SYSTEMDDIR)
	install -m 644 scripts/abel.service $(DESTDIR)$(SYSTEMDDIR)/abel.service
	@if command -v systemctl >/dev/null 2>&1; then \
		systemctl daemon-reload; \
		echo "systemd service installed. Enable with: sudo systemctl enable --now abel"; \
	fi

enable-systemd: install-systemd
	@echo "==> Enabling and starting Abel systemd service..."
	systemctl enable --now abel.service

uninstall:
	@echo "==> Uninstalling Abel..."
	@if command -v systemctl >/dev/null 2>&1; then \
		systemctl stop abel.service 2>/dev/null || true; \
		systemctl disable abel.service 2>/dev/null || true; \
	fi
	rm -f $(DESTDIR)$(BINDIR)/abel
	rm -f $(DESTDIR)$(BINDIR)/abel-service
	rm -f $(DESTDIR)$(SYSTEMDDIR)/abel.service
	rm -rf $(DESTDIR)$(DATADIR)
	@echo "Note: Configuration at $(DESTDIR)$(SYSCONFDIR) was preserved."
	@if command -v systemctl >/dev/null 2>&1; then \
		systemctl daemon-reload; \
	fi

clean:
	@echo "==> Cleaning build artifacts..."
	rm -rf bin
	rm -rf src/backend/static/*
	rm -rf src/frontend/build

test:
	@echo "==> Running backend tests..."
	go test -race ./src/backend/...
	@echo "==> Running frontend tests..."
	npm --prefix src/frontend run test:unit

dev: build
	@echo "==> Starting Abel..."
	@mkdir -p $$HOME/.config/abel
	@if [ ! -f $$HOME/.config/abel/config.yaml ]; then \
		cp config/config-example.yaml $$HOME/.config/abel/config.yaml; \
	fi
	./bin/abel
