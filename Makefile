.PHONY: dev build frontend import-web package unraid test clean

APP_NAME := msm-free
DIST := dist
WEB_EXPORT ?= msm_html_export.tar.gz
VERSION ?= 0.1.0-dev
UNRAID_VERSION ?= $(VERSION)
GITHUB_REPO ?= luochuhan/msm-free
RELEASE_TAG ?= v$(VERSION)
GOOS ?= linux
GOARCH ?= amd64
BIN := $(DIST)/$(APP_NAME)-$(GOOS)-$(GOARCH)
PACKAGE_DIR := $(DIST)/$(APP_NAME)-$(VERSION)-$(GOOS)-$(GOARCH)

frontend:
	@test -f internal/server/web/dist/index.html || (echo "missing exported web assets; run: make import-web WEB_EXPORT=$(WEB_EXPORT)" && exit 1)
	@echo "using exported MSM web assets from internal/server/web/dist"

import-web:
	@tmp=$$(mktemp -d); \
	tar -xzf "$(WEB_EXPORT)" -C "$$tmp"; \
	src=$$(find "$$tmp" -type f -name index.raw.html -print -quit | xargs dirname); \
	test -n "$$src"; \
	rm -rf internal/server/web/dist; \
	mkdir -p internal/server/web/dist; \
	cp "$$src/index.raw.html" internal/server/web/dist/index.html; \
	for name in assets logo pages offline_pages dashboard_preview.png manifest.json; do \
		if [ -e "$$src/$$name" ]; then cp -R "$$src/$$name" internal/server/web/dist/; fi; \
	done; \
	if [ -f internal/server/web/dist/manifest.json ]; then mv internal/server/web/dist/manifest.json internal/server/web/dist/export-manifest.json; fi; \
	rm -rf "$$tmp"; \
	echo "imported exported MSM web assets from $(WEB_EXPORT)"

build: frontend package

package: frontend
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(BIN) ./cmd/msm-free
	rm -rf $(PACKAGE_DIR)
	mkdir -p $(PACKAGE_DIR)/systemd
	cp $(BIN) $(PACKAGE_DIR)/$(APP_NAME)
	cp packaging/install.sh packaging/uninstall.sh $(PACKAGE_DIR)/
	cp packaging/systemd/$(APP_NAME).service $(PACKAGE_DIR)/systemd/
	cp packaging/README-linux-amd64.md $(PACKAGE_DIR)/README.md
	chmod 0755 $(PACKAGE_DIR)/$(APP_NAME) $(PACKAGE_DIR)/install.sh $(PACKAGE_DIR)/uninstall.sh
	cd $(PACKAGE_DIR) && if command -v sha256sum >/dev/null 2>&1; then find . -type f ! -name SHA256SUMS -print | LC_ALL=C sort | xargs sha256sum > SHA256SUMS; else find . -type f ! -name SHA256SUMS -print | LC_ALL=C sort | xargs shasum -a 256 > SHA256SUMS; fi
	cd $(DIST) && tar -czf $(APP_NAME)-$(GOOS)-$(GOARCH).tar.gz $(notdir $(PACKAGE_DIR))

unraid: package
	APP_NAME=$(APP_NAME) VERSION=$(VERSION) UNRAID_VERSION=$(UNRAID_VERSION) GITHUB_REPO=$(GITHUB_REPO) RELEASE_TAG=$(RELEASE_TAG) DIST=$(DIST) packaging/unraid/build-unraid.sh

dev:
	go run ./cmd/msm-free serve -c ./data -p 7777

test:
	go test ./...

clean:
	rm -rf $(DIST)
