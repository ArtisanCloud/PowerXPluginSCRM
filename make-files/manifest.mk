PLUGIN_FILE ?= plugin.yaml
ifneq ($(wildcard docs/lifecycle/examples/manifest.yaml),)
MANIFEST_FILE ?= docs/lifecycle/examples/manifest.yaml
else ifneq ($(wildcard ../docs/standards/powerx-plugin/lifecycle/examples/manifest.yaml),)
MANIFEST_FILE ?= ../docs/standards/powerx-plugin/lifecycle/examples/manifest.yaml
else
MANIFEST_FILE ?= docs/contracts/manifest.yaml
endif
ifneq ($(wildcard docs/lifecycle/contracts/manifest.schema.json),)
MANIFEST_SCHEMA ?= docs/lifecycle/contracts/manifest.schema.json
else ifneq ($(wildcard ../docs/standards/powerx-plugin/lifecycle/contracts/manifest.schema.json),)
MANIFEST_SCHEMA ?= ../docs/standards/powerx-plugin/lifecycle/contracts/manifest.schema.json
else
MANIFEST_SCHEMA ?= ../specs/001-powerxplugin-foundation/contracts/manifest.schema.json
endif
ABS_PLUGIN_FILE := $(abspath $(PLUGIN_FILE))
ABS_MANIFEST_FILE := $(abspath $(MANIFEST_FILE))
ABS_MANIFEST_SCHEMA := $(abspath $(MANIFEST_SCHEMA))
EVENT_FABRIC_FILE ?= config/event_fabric.yaml
ABS_EVENT_FABRIC_FILE := $(abspath $(EVENT_FABRIC_FILE))
MANIFESTCHECK_DIR := $(BACKEND_DIR)/cmd/manifestcheck
BACKEND_GOCACHE := $(abspath $(BACKEND_DIR)/.cache/go-build)

.PHONY: plugin-yaml-sync
plugin-yaml-sync:
	@echo "[manifest] syncing plugin catalogs from contracts/capabilities"
	@mkdir -p $(BACKEND_GOCACHE)
	@cd $(BACKEND_DIR) && GOCACHE=$(BACKEND_GOCACHE) go run ./cmd/manifestcheck --plugin "$(ABS_PLUGIN_FILE)" --sync-catalogs --sync-only

.PHONY: verify-manifest
verify-manifest:
	@echo "[manifest] Validating $(PLUGIN_FILE) against $(MANIFEST_FILE)"
	@mkdir -p $(BACKEND_GOCACHE)
	@cd $(BACKEND_DIR) && GOCACHE=$(BACKEND_GOCACHE) go run ./cmd/manifestcheck --plugin "$(ABS_PLUGIN_FILE)" --manifest "$(ABS_MANIFEST_FILE)" --schema "$(ABS_MANIFEST_SCHEMA)"

.PHONY: check-capability
check-capability:
	@echo "[manifest] Running capability validation against $(PLUGIN_FILE)"
	@mkdir -p $(BACKEND_GOCACHE)
	@cd $(BACKEND_DIR) && GOCACHE=$(BACKEND_GOCACHE) go run ./cmd/manifestcheck --plugin "$(ABS_PLUGIN_FILE)" --manifest "$(ABS_MANIFEST_FILE)" --schema "$(ABS_MANIFEST_SCHEMA)" --capabilities-only

.PHONY: plugin-yaml-check
plugin-yaml-check: plugin-yaml-sync plugin-id-check
	@echo "[manifest] running one-shot plugin.yaml checks (id + capabilities + events)"
	@mkdir -p $(BACKEND_GOCACHE)
	@cd $(BACKEND_DIR) && GOCACHE=$(BACKEND_GOCACHE) go run ./cmd/manifestcheck --plugin "$(ABS_PLUGIN_FILE)" --manifest "" --schema "$(ABS_MANIFEST_SCHEMA)" --capabilities-only --plugin-only --event-fabric "$(ABS_EVENT_FABRIC_FILE)"
	@echo "✅ plugin.yaml checks passed"

.PHONY: plugin-id-check
plugin-id-check:
	@echo "[manifest] checking plugin id naming convention"
	@PLUGIN_ID_VAL=$$(awk -F': *' '/^id:/ {print $$2; exit}' $(PLUGIN_FILE)); \
	case "$$PLUGIN_ID_VAL" in com.powerx.plugins.*) ;; *) echo "❌ plugin id 不符合规范: $$PLUGIN_ID_VAL（应以 com.powerx.plugins. 开头）"; exit 1;; esac
	@LEGACY_MATCH=$$(rg -n "com\\.powerx\\.plugin\\." web-admin backend make-files plugin.yaml -g '!make-files/manifest.mk' 2>/dev/null || true); \
	if [ -n "$$LEGACY_MATCH" ]; then echo "❌ 发现旧命名 com.powerx.plugin.* 残留:"; echo "$$LEGACY_MATCH"; exit 1; fi; \
	echo "✅ plugin id naming check passed"

.PHONY: manifest-align-fix
manifest-align-fix:
	@node .codex/skills/ci/manifest-align/scripts/manifest-align-check.mjs --fix

.PHONY: manifest-align-check
manifest-align-check:
	@node .codex/skills/ci/manifest-align/scripts/manifest-align-check.mjs
