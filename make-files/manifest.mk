PLUGIN_FILE ?= plugin.yaml
MANIFEST_FILE ?= docs/lifecycle/examples/manifest.yaml
MANIFEST_SCHEMA ?= docs/lifecycle/contracts/manifest.schema.json

ABS_PLUGIN_FILE := $(abspath $(PLUGIN_FILE))
ABS_MANIFEST_FILE := $(abspath $(MANIFEST_FILE))
ABS_MANIFEST_SCHEMA := $(abspath $(MANIFEST_SCHEMA))
BACKEND_GOCACHE := $(abspath backend/.cache/go-build)

.PHONY: verify-manifest
verify-manifest:
	@echo "[manifest] Validating $(PLUGIN_FILE) against $(MANIFEST_FILE)"
	@mkdir -p $(BACKEND_GOCACHE)
	@cd backend && GOCACHE=$(BACKEND_GOCACHE) go run ./cmd/manifestcheck \
		--plugin "$(ABS_PLUGIN_FILE)" \
		--manifest "$(ABS_MANIFEST_FILE)" \
		--schema "$(ABS_MANIFEST_SCHEMA)"

.PHONY: check-capability
check-capability:
	@echo "[manifest] Running capability validation against $(PLUGIN_FILE)"
	@mkdir -p $(BACKEND_GOCACHE)
	@cd backend && GOCACHE=$(BACKEND_GOCACHE) go run ./cmd/manifestcheck \
		--plugin "$(ABS_PLUGIN_FILE)" \
		--manifest "$(ABS_MANIFEST_FILE)" \
		--schema "$(ABS_MANIFEST_SCHEMA)" \
		--capabilities-only
