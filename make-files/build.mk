PLUGIN_ID ?= com.powerx.plugins.scrm
VERSION ?= $(shell awk -F': *' '/^version:/ {print $$2; exit}' plugin.yaml 2>/dev/null || echo "0.1.0")
BACKEND_DIR ?= .
BUILD_DIR ?= $(BACKEND_DIR)/bin
ABS_BACKEND_DIR := $(abspath $(BACKEND_DIR))
ABS_BUILD_DIR := $(abspath $(BUILD_DIR))
GO_BUILD_CACHE ?= $(abspath $(BACKEND_DIR)/.cache/go-build)
PLATFORM ?= host
TARGET_ARCH ?= amd64
FRONTEND_DIR ?= web-admin
FRONTEND_OUTPUT ?= $(FRONTEND_DIR)/.output
DIST_ROOT ?= dist
DIST_DIR ?= $(DIST_ROOT)/$(VERSION)
DIST_BACKEND_BIN ?= $(DIST_DIR)/backend/bin
DIST_WEBADMIN_DIR ?= $(DIST_DIR)/web-admin
DIST_WEBADMIN_OUTPUT ?= $(DIST_WEBADMIN_DIR)/.output
RELEASE_ROOT ?= target
RELEASE_DIR ?= $(RELEASE_ROOT)/$(VERSION)
RELEASE_BACKEND_BIN ?= $(RELEASE_DIR)/backend/bin
RELEASE_WEBADMIN_DIR ?= $(RELEASE_DIR)/web-admin
RELEASE_WEBADMIN_OUTPUT ?= $(RELEASE_WEBADMIN_DIR)/.output
POWERX_ADMIN_BASE ?= /_p/$(PLUGIN_ID)/admin/

.PHONY: build
build:
	@mkdir -p $(ABS_BUILD_DIR) $(GO_BUILD_CACHE)
	@if [ "$(PLATFORM)" = "linux" ]; then \
		echo "==> 构建后端二进制（Linux/$(TARGET_ARCH)）"; \
		GOOS=linux GOARCH=$(TARGET_ARCH) GOCACHE=$(GO_BUILD_CACHE) go build -C $(ABS_BACKEND_DIR) -o $(ABS_BUILD_DIR)/plugin ./cmd/plugin; \
		if [ -d "$(ABS_BACKEND_DIR)/cmd/database" ]; then GOOS=linux GOARCH=$(TARGET_ARCH) GOCACHE=$(GO_BUILD_CACHE) go build -C $(ABS_BACKEND_DIR) -o $(ABS_BUILD_DIR)/migrate ./cmd/database; fi; \
	else \
		echo "==> 构建后端二进制（本机平台）"; \
		GOCACHE=$(GO_BUILD_CACHE) go build -C $(ABS_BACKEND_DIR) -o $(ABS_BUILD_DIR)/plugin ./cmd/plugin; \
		if [ -d "$(ABS_BACKEND_DIR)/cmd/database" ]; then GOCACHE=$(GO_BUILD_CACHE) go build -C $(ABS_BACKEND_DIR) -o $(ABS_BUILD_DIR)/migrate ./cmd/database; fi; \
	fi

.PHONY: frontend-build
frontend-build:
	cd $(FRONTEND_DIR) && POWERX_PROXY=1 POWERX_PLUGIN_ID="$(PLUGIN_ID)" POWERX_PLUGIN_VERSION="$(VERSION)" NUXT_PUBLIC_POWERX_PLUGIN_ID="$(PLUGIN_ID)" NUXT_PUBLIC_POWERX_PLUGIN_VERSION="$(VERSION)" NUXT_PUBLIC_API_BASE= NUXT_PUBLIC_API_PREFIX= POWERX_ADMIN_BASE="$(POWERX_ADMIN_BASE)" NODE_ENV=production npm run build

.PHONY: frontend-build-standalone
frontend-build-standalone:
	cd $(FRONTEND_DIR) && POWERX_PROXY=0 POWERX_PLUGIN_ID="$(PLUGIN_ID)" POWERX_PLUGIN_VERSION="$(VERSION)" NUXT_PUBLIC_POWERX_PLUGIN_ID="$(PLUGIN_ID)" NUXT_PUBLIC_POWERX_PLUGIN_VERSION="$(VERSION)" NODE_ENV=production npm run build

.PHONY: dist
dist: plugin-yaml-check build frontend-build
	@echo "==> 生成 dist 安装包目录：$(DIST_DIR)"
	@rm -rf $(DIST_DIR)
	@mkdir -p $(DIST_BACKEND_BIN) $(DIST_WEBADMIN_OUTPUT)
	@awk -v ver="$(VERSION)" 'BEGIN{patched=0} /^[[:space:]]*version:[[:space:]]*/ && !patched {print "version: " ver; patched=1; next} {print} END{if(!patched) print "version: " ver}' plugin.yaml > $(DIST_DIR)/plugin.yaml
	@if [ -d "plugin.d" ]; then mkdir -p $(DIST_DIR)/plugin.d; cp -R plugin.d/. $(DIST_DIR)/plugin.d/; fi
	@if [ -f "config/event_fabric.yaml" ]; then mkdir -p $(DIST_DIR)/config; cp config/event_fabric.yaml $(DIST_DIR)/config/event_fabric.yaml; fi
	@cp $(BUILD_DIR)/plugin $(DIST_BACKEND_BIN)/
	@if [ -f "$(BUILD_DIR)/migrate" ]; then cp $(BUILD_DIR)/migrate $(DIST_BACKEND_BIN)/; fi
	@chmod 0755 $(DIST_BACKEND_BIN)/plugin
	@if [ -f "$(DIST_BACKEND_BIN)/migrate" ]; then chmod 0755 $(DIST_BACKEND_BIN)/migrate; fi
	@if [ -d "backend/etc" ]; then mkdir -p $(DIST_DIR)/backend/etc; cp -R backend/etc/. $(DIST_DIR)/backend/etc/; fi
	@if [ -d "contracts" ]; then mkdir -p $(DIST_DIR)/contracts; cp -R contracts/. $(DIST_DIR)/contracts/; fi
	@if [ -d "contracts/schema" ]; then mkdir -p $(DIST_DIR)/schema; cp -R contracts/schema/. $(DIST_DIR)/schema/; fi
	@if [ -d "$(FRONTEND_OUTPUT)" ]; then cp -R $(FRONTEND_OUTPUT)/. $(DIST_WEBADMIN_OUTPUT)/; fi
	@if [ -d "$(FRONTEND_DIR)/i18n" ]; then mkdir -p $(DIST_WEBADMIN_DIR)/i18n; cp -R $(FRONTEND_DIR)/i18n/. $(DIST_WEBADMIN_DIR)/i18n/; fi
	@if [ -f README.md ]; then cp README.md $(DIST_DIR)/; fi
	@$(MAKE) --no-print-directory dist-verify DIST_DIR="$(DIST_DIR)" DIST_BACKEND_BIN="$(DIST_BACKEND_BIN)" DIST_WEBADMIN_OUTPUT="$(DIST_WEBADMIN_OUTPUT)"

.PHONY: dist-linux
dist-linux:
	@$(MAKE) dist PLATFORM=linux TARGET_ARCH="$(TARGET_ARCH)"

.PHONY: dist-verify
dist-verify:
	@test -s "$(DIST_BACKEND_BIN)/plugin" || { echo "❌ dist 验证失败：缺少后端二进制 $(DIST_BACKEND_BIN)/plugin"; exit 1; }
	@test -x "$(DIST_BACKEND_BIN)/plugin" || { echo "❌ dist 验证失败：$(DIST_BACKEND_BIN)/plugin 不可执行"; exit 1; }
	@test -s "$(DIST_BACKEND_BIN)/migrate" || { echo "❌ dist 验证失败：缺少迁移二进制 $(DIST_BACKEND_BIN)/migrate"; exit 1; }
	@test -x "$(DIST_BACKEND_BIN)/migrate" || { echo "❌ dist 验证失败：$(DIST_BACKEND_BIN)/migrate 不可执行"; exit 1; }
	@test -f "$(DIST_DIR)/plugin.yaml" || { echo "❌ dist 验证失败：缺少 $(DIST_DIR)/plugin.yaml"; exit 1; }
	@test -f "$(DIST_DIR)/plugin.d/rbac.yaml" || { echo "❌ dist 验证失败：缺少 $(DIST_DIR)/plugin.d/rbac.yaml"; exit 1; }
	@test -f "$(DIST_DIR)/plugin.d/capabilities.yaml" || { echo "❌ dist 验证失败：缺少 $(DIST_DIR)/plugin.d/capabilities.yaml"; exit 1; }
	@test -f "$(DIST_DIR)/plugin.d/exposure.yaml" || { echo "❌ dist 验证失败：缺少 $(DIST_DIR)/plugin.d/exposure.yaml"; exit 1; }
	@test -f "$(DIST_DIR)/config/event_fabric.yaml" || { echo "❌ dist 验证失败：缺少 $(DIST_DIR)/config/event_fabric.yaml"; exit 1; }
		@test -f "$(DIST_DIR)/backend/etc/config.yaml" || { echo "❌ dist 验证失败：缺少 $(DIST_DIR)/backend/etc/config.yaml"; exit 1; }
		@test -d "$(DIST_DIR)/contracts/capabilities" || { echo "❌ dist 验证失败：缺少 $(DIST_DIR)/contracts/capabilities"; exit 1; }
		@test -f "$(DIST_WEBADMIN_OUTPUT)/server/index.mjs" || { echo "❌ dist 验证失败：缺少 Nuxt server 产物"; exit 1; }
		@ICON_PATH=$$(awk '/^[[:space:]]*metadata:[[:space:]]*$$/{in_metadata=1; next} in_metadata && /^[^[:space:]]/ {in_metadata=0} in_metadata && /^[[:space:]]*icon:[[:space:]]*/ {sub(/^[[:space:]]*icon:[[:space:]]*/, ""); gsub(/"/, ""); print; exit}' "$(DIST_DIR)/plugin.yaml"); \
		if [ -n "$$ICON_PATH" ]; then \
			test -f "$(DIST_WEBADMIN_OUTPUT)/public/$$ICON_PATH" || { echo "❌ dist 验证失败：metadata.icon 指向的文件不存在：$(DIST_WEBADMIN_OUTPUT)/public/$$ICON_PATH"; exit 1; }; \
		fi
		@awk '/^[[:space:]]*runtime:[[:space:]]*$$/{in_runtime=1; next} in_runtime && /^[^[:space:]]/ {in_runtime=0} in_runtime && /^[[:space:]]*entry:[[:space:]]*backend\/bin\/plugin[[:space:]]*$$/ {found=1} END{exit found?0:1}' "$(DIST_DIR)/plugin.yaml" || { echo "❌ plugin.yaml runtime.entry 必须是 backend/bin/plugin"; exit 1; }
	@rg -q 'POWERX_BIND_ADDR:[[:space:]]*":__POWERX_DYNAMIC_PORT__"' "$(DIST_DIR)/plugin.yaml" || { echo "❌ plugin.yaml runtime.env.POWERX_BIND_ADDR 必须是 :__POWERX_DYNAMIC_PORT__"; exit 1; }
	@rg -q 'POWERX_PLUGIN_REGISTRATION_MODE:[[:space:]]*"installed"' "$(DIST_DIR)/plugin.yaml" || { echo "❌ plugin.yaml runtime.env.POWERX_PLUGIN_REGISTRATION_MODE 必须是 installed"; exit 1; }
	@awk '/^[[:space:]]*backend:[[:space:]]*$$/{in_backend=1; next} in_backend && /^[^[:space:]]/ {in_backend=0} in_backend && /^[[:space:]]*port:[[:space:]]*0[[:space:]]*$$/ {found=1} END{exit found?0:1}' "$(DIST_DIR)/plugin.yaml" || { echo "❌ plugin.yaml backend.port 必须是 0"; exit 1; }
	@! rg -q '(^|[^0-9])(8078|8086)([^0-9]|$$)' "$(DIST_DIR)/plugin.yaml" || { echo "❌ plugin.yaml 不得固化旧 backend port：8078/8086"; exit 1; }
	@awk '/^[[:space:]]*migrations:[[:space:]]*$$/{in_migrations=1; next} in_migrations && /^[^[:space:]]/ {in_migrations=0} in_migrations && /^[[:space:]]*entry:[[:space:]]*backend\/bin\/migrate[[:space:]]*$$/ {found=1} END{exit found?0:1}' "$(DIST_DIR)/plugin.yaml" || { echo "❌ plugin.yaml migrations.entry 必须是 backend/bin/migrate"; exit 1; }
	@rg -q "resource: scrm.opportunity" "$(DIST_DIR)/plugin.d/rbac.yaml" || { echo "❌ dist 验证失败：rbac 缺少 scrm.opportunity"; exit 1; }
	@rg -q "resource: scrm.leads" "$(DIST_DIR)/plugin.d/rbac.yaml" || { echo "❌ dist 验证失败：rbac 缺少 scrm.leads"; exit 1; }
	@rg -q "resource: scrm.social_channel_accounts" "$(DIST_DIR)/plugin.d/rbac.yaml" || { echo "❌ dist 验证失败：rbac 缺少 scrm.social_channel_accounts"; exit 1; }
	@rg -q "/admin/opportunity/records" "$(DIST_DIR)/plugin.d/exposure.yaml" || { echo "❌ dist 验证失败：exposure 缺少商机接口"; exit 1; }
	@rg -q "/admin/leads" "$(DIST_DIR)/plugin.d/exposure.yaml" || { echo "❌ dist 验证失败：exposure 缺少线索接口"; exit 1; }
	@! rg -q "^[[:space:]]*-[[:space:]]+capability:" "$(DIST_DIR)/plugin.d/exposure.yaml" || { echo "❌ dist 验证失败：exposure channel 缺少 auth，请确保每个 channel 都先声明 auth"; exit 1; }
	@echo "✅ dist 验证通过：$(DIST_DIR)"

.PHONY: release
release: build frontend-build
	@rm -rf $(RELEASE_DIR)
	@mkdir -p $(RELEASE_BACKEND_BIN) $(RELEASE_WEBADMIN_OUTPUT)
	@awk -v ver="$(VERSION)" 'BEGIN{patched=0} /^[[:space:]]*version:[[:space:]]*/ && !patched {print "version: " ver; patched=1; next} {print} END{if(!patched) print "version: " ver}' plugin.yaml > $(RELEASE_DIR)/plugin.yaml
	@if [ -d "plugin.d" ]; then mkdir -p $(RELEASE_DIR)/plugin.d; cp -R plugin.d/. $(RELEASE_DIR)/plugin.d/; fi
	@if [ -f "config/event_fabric.yaml" ]; then mkdir -p $(RELEASE_DIR)/config; cp config/event_fabric.yaml $(RELEASE_DIR)/config/event_fabric.yaml; fi
	@cp $(BUILD_DIR)/plugin $(RELEASE_BACKEND_BIN)/
	@if [ -f "$(BUILD_DIR)/migrate" ]; then cp $(BUILD_DIR)/migrate $(RELEASE_BACKEND_BIN)/; fi
	@chmod 0755 $(RELEASE_BACKEND_BIN)/plugin
	@if [ -f "$(RELEASE_BACKEND_BIN)/migrate" ]; then chmod 0755 $(RELEASE_BACKEND_BIN)/migrate; fi
	@if [ -d "backend/etc" ]; then mkdir -p $(RELEASE_DIR)/backend/etc; cp -R backend/etc/. $(RELEASE_DIR)/backend/etc/; fi
	@if [ -d "contracts" ]; then mkdir -p $(RELEASE_DIR)/contracts; cp -R contracts/. $(RELEASE_DIR)/contracts/; fi
	@if [ -d "contracts/schema" ]; then mkdir -p $(RELEASE_DIR)/schema; cp -R contracts/schema/. $(RELEASE_DIR)/schema/; fi
	@if [ -d "$(FRONTEND_OUTPUT)" ]; then cp -R $(FRONTEND_OUTPUT)/. $(RELEASE_WEBADMIN_OUTPUT)/; fi
	@if [ -d "$(FRONTEND_DIR)/i18n" ]; then mkdir -p $(RELEASE_WEBADMIN_DIR)/i18n; cp -R $(FRONTEND_DIR)/i18n/. $(RELEASE_WEBADMIN_DIR)/i18n/; fi
	@if [ -f README.md ]; then cp README.md $(RELEASE_DIR)/; fi
