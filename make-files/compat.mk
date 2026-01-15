COMPAT_CONFIG ?= contracts/compatibility.yaml
COMPAT_OUTPUT ?= build/compat
COMPAT_SCRIPT := $(abspath scripts/check-compatibility.mjs)
COMPAT_NODE ?= node

.PHONY: check-compat
check-compat:
	@echo "[compat] Generating compatibility report using $(COMPAT_CONFIG)"
	@mkdir -p $(COMPAT_OUTPUT)
	@$(COMPAT_NODE) $(COMPAT_SCRIPT) --config "$(abspath $(COMPAT_CONFIG))" --out "$(abspath $(COMPAT_OUTPUT))"
