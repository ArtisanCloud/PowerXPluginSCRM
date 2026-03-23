PLUGIN_ID ?= com.powerx.plugins.scrm
VERSION ?= $(shell awk -F': *' '/^version:/ {print $$2; exit}' plugin.yaml 2>/dev/null || echo "0.1.0")
BACKEND_DIR ?= .
BUILD_DIR ?= $(BACKEND_DIR)/bin
ABS_BACKEND_DIR := $(abspath $(BACKEND_DIR))
ABS_BUILD_DIR := $(abspath $(BUILD_DIR))
GO_BUILD_CACHE ?= $(abspath $(BACKEND_DIR)/.cache/go-build)
FRONTEND_DIR ?= web-admin
FRONTEND_OUTPUT ?= $(FRONTEND_DIR)/.output
DIST_ROOT ?= dist
DIST_DIR ?= $(DIST_ROOT)/$(VERSION)
DIST_BACKEND_BIN ?= $(DIST_DIR)/bin
DIST_WEBADMIN_DIR ?= $(DIST_DIR)/web-admin
DIST_WEBADMIN_OUTPUT ?= $(DIST_WEBADMIN_DIR)/.output
RELEASE_ROOT ?= target
RELEASE_DIR ?= $(RELEASE_ROOT)/$(VERSION)
RELEASE_BACKEND_BIN ?= $(RELEASE_DIR)/bin
RELEASE_WEBADMIN_DIR ?= $(RELEASE_DIR)/web-admin
RELEASE_WEBADMIN_OUTPUT ?= $(RELEASE_WEBADMIN_DIR)/.output
POWERX_ADMIN_BASE ?= /_p/$(PLUGIN_ID)/admin/

.PHONY: build
build:
	@mkdir -p $(ABS_BUILD_DIR) $(GO_BUILD_CACHE)
	GOCACHE=$(GO_BUILD_CACHE) go build -C $(ABS_BACKEND_DIR) -o $(ABS_BUILD_DIR)/plugin ./cmd/plugin
	@if [ -d "$(ABS_BACKEND_DIR)/cmd/database" ]; then GOCACHE=$(GO_BUILD_CACHE) go build -C $(ABS_BACKEND_DIR) -o $(ABS_BUILD_DIR)/migrate ./cmd/database; fi

.PHONY: frontend-build
frontend-build:
	cd $(FRONTEND_DIR) && POWERX_PROXY=1 POWERX_PLUGIN_ID="$(PLUGIN_ID)" POWERX_PLUGIN_VERSION="$(VERSION)" NUXT_PUBLIC_POWERX_PLUGIN_ID="$(PLUGIN_ID)" NUXT_PUBLIC_POWERX_PLUGIN_VERSION="$(VERSION)" NUXT_PUBLIC_API_BASE= NUXT_PUBLIC_API_PREFIX= POWERX_ADMIN_BASE="$(POWERX_ADMIN_BASE)" NODE_ENV=production npm run build

.PHONY: frontend-build-standalone
frontend-build-standalone:
	cd $(FRONTEND_DIR) && POWERX_PROXY=0 POWERX_PLUGIN_ID="$(PLUGIN_ID)" POWERX_PLUGIN_VERSION="$(VERSION)" NUXT_PUBLIC_POWERX_PLUGIN_ID="$(PLUGIN_ID)" NUXT_PUBLIC_POWERX_PLUGIN_VERSION="$(VERSION)" NODE_ENV=production npm run build

.PHONY: dist
dist: plugin-yaml-check build frontend-build
	@rm -rf $(DIST_DIR)
	@mkdir -p $(DIST_BACKEND_BIN) $(DIST_WEBADMIN_OUTPUT)
	@awk -v ver="$(VERSION)" 'BEGIN{patched=0} /^[[:space:]]*version:[[:space:]]*/ && !patched {print "version: " ver; patched=1; next} {print} END{if(!patched) print "version: " ver}' plugin.yaml > $(DIST_DIR)/plugin.yaml
	@if [ -d "plugin.d" ]; then mkdir -p $(DIST_DIR)/plugin.d; cp -R plugin.d/. $(DIST_DIR)/plugin.d/; fi
	@if [ -f "config/event_fabric.yaml" ]; then mkdir -p $(DIST_DIR)/config; cp config/event_fabric.yaml $(DIST_DIR)/config/event_fabric.yaml; fi
	@cp $(BUILD_DIR)/plugin $(DIST_BACKEND_BIN)/
	@if [ -f "$(BUILD_DIR)/migrate" ]; then cp $(BUILD_DIR)/migrate $(DIST_BACKEND_BIN)/; fi
	@if [ -d "$(FRONTEND_OUTPUT)" ]; then cp -R $(FRONTEND_OUTPUT)/. $(DIST_WEBADMIN_OUTPUT)/; fi
	@if [ -d "$(FRONTEND_DIR)/i18n" ]; then mkdir -p $(DIST_WEBADMIN_DIR)/i18n; cp -R $(FRONTEND_DIR)/i18n/. $(DIST_WEBADMIN_DIR)/i18n/; fi
	@if [ -f README.md ]; then cp README.md $(DIST_DIR)/; fi

.PHONY: release
release: build frontend-build
	@rm -rf $(RELEASE_DIR)
	@mkdir -p $(RELEASE_BACKEND_BIN) $(RELEASE_WEBADMIN_OUTPUT)
	@awk -v ver="$(VERSION)" 'BEGIN{patched=0} /^[[:space:]]*version:[[:space:]]*/ && !patched {print "version: " ver; patched=1; next} {print} END{if(!patched) print "version: " ver}' plugin.yaml > $(RELEASE_DIR)/plugin.yaml
	@if [ -d "plugin.d" ]; then mkdir -p $(RELEASE_DIR)/plugin.d; cp -R plugin.d/. $(RELEASE_DIR)/plugin.d/; fi
	@if [ -f "config/event_fabric.yaml" ]; then mkdir -p $(RELEASE_DIR)/config; cp config/event_fabric.yaml $(RELEASE_DIR)/config/event_fabric.yaml; fi
	@cp $(BUILD_DIR)/plugin $(RELEASE_BACKEND_BIN)/
	@if [ -f "$(BUILD_DIR)/migrate" ]; then cp $(BUILD_DIR)/migrate $(RELEASE_BACKEND_BIN)/; fi
	@if [ -d "$(FRONTEND_OUTPUT)" ]; then cp -R $(FRONTEND_OUTPUT)/. $(RELEASE_WEBADMIN_OUTPUT)/; fi
	@if [ -d "$(FRONTEND_DIR)/i18n" ]; then mkdir -p $(RELEASE_WEBADMIN_DIR)/i18n; cp -R $(FRONTEND_DIR)/i18n/. $(RELEASE_WEBADMIN_DIR)/i18n/; fi
	@if [ -f README.md ]; then cp README.md $(RELEASE_DIR)/; fi
