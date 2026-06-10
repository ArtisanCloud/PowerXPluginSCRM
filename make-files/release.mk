PACKAGE_STAGE_ROOT ?= build/pxp
PACKAGE_VERSION_DIR ?= $(PACKAGE_STAGE_ROOT)/$(VERSION)
PACKAGE_META_DIR ?= $(PACKAGE_VERSION_DIR)/meta
PACKAGE_BACKEND_DIR ?= $(PACKAGE_VERSION_DIR)/backend
PACKAGE_FRONTEND_DIR ?= $(PACKAGE_VERSION_DIR)/web-admin
PACKAGE_HASH_FILE ?= $(PACKAGE_VERSION_DIR)/hashes.txt
PACKAGE_AUDIT_LOG ?= $(PACKAGE_VERSION_DIR)/audit.log
PACKAGE_SIGNATURE ?= $(PACKAGE_VERSION_DIR)/signature.json
ADVISORY_SOURCE_DIR ?= build/security/advisories
ADVISORY_DIST_ROOT ?= dist/security
ADVISORY_DIST_VERSION ?= $(ADVISORY_DIST_ROOT)/$(VERSION)

PX_PLUGIN_BIN ?= px-plugin
PACK_OUTPUT_DIR ?= $(DIST_ROOT)/artifacts
MARKETPLACE_PUBLIC_KEY ?= $(PUBLIC_KEY)
KEY_ID ?=

API_BASE ?=
TOKEN ?=
ENABLE ?= true
FORCE ?= false
LOCAL_INSTALL_SRC ?= $(abspath $(DIST_DIR))
LOCAL_INSTALL_TMP ?= $(DIST_ROOT)/.local-install
LOCAL_INSTALL_PXP_TMP ?= $(DIST_ROOT)/.pxp-unpack
PACKAGE ?=

.PHONY: package-pxp
package-pxp: verify-manifest build frontend-build
	@echo "[package] Staging artefacts in $(PACKAGE_VERSION_DIR)"
	@rm -rf $(PACKAGE_VERSION_DIR)
	@mkdir -p $(PACKAGE_META_DIR) $(PACKAGE_BACKEND_DIR) $(PACKAGE_FRONTEND_DIR)
	@cp $(PLUGIN_FILE) $(PACKAGE_META_DIR)/plugin.yaml
	@if [ -f "$(MANIFEST_FILE)" ]; then \
		cp "$(MANIFEST_FILE)" $(PACKAGE_META_DIR)/manifest.yaml; \
	else \
		echo "⚠️  manifest file $(MANIFEST_FILE) not found; produced bundle without manifest"; \
	fi
	@if [ -f "$(BUILD_DIR)/plugin" ]; then cp $(BUILD_DIR)/plugin $(PACKAGE_BACKEND_DIR)/; fi
	@if [ -f "$(BUILD_DIR)/migrate" ]; then cp $(BUILD_DIR)/migrate $(PACKAGE_BACKEND_DIR)/; fi
	@if [ -d "$(FRONTEND_OUTPUT)" ] && [ -n "$$\(ls -A $(FRONTEND_OUTPUT) 2>/dev/null)" ]; then \
		cp -R $(FRONTEND_OUTPUT)/. $(PACKAGE_FRONTEND_DIR)/; \
	else \
		echo "⚠️  No frontend output found at $(FRONTEND_OUTPUT)"; \
	fi
	@python scripts/hash_package.py "$(PACKAGE_VERSION_DIR)" "$(PACKAGE_HASH_FILE)"
	@{ \
		echo "created_at=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)"; \
		echo "plugin_version=$(VERSION)"; \
		echo "source_commit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)"; \
		echo "staged_dir=$(PACKAGE_VERSION_DIR)"; \
	} > $(PACKAGE_AUDIT_LOG)
	@printf '{\n  "status": "pending",\n  "signed_by": "",\n  "signed_at": "",\n  "note": "Upload package to signing service to finalize signatures"\n}\n' > $(PACKAGE_SIGNATURE)
	@echo "[package] Artefacts staged. Hashes recorded in $(PACKAGE_HASH_FILE)"
	@rm -rf $(ADVISORY_DIST_VERSION)
	@mkdir -p $(ADVISORY_DIST_VERSION)
	@if [ -d "$(ADVISORY_SOURCE_DIR)" ] && [ -n "$$(ls -A $(ADVISORY_SOURCE_DIR) 2>/dev/null)" ]; then \
		echo "[package] Bundling advisories from $(ADVISORY_SOURCE_DIR) into $(ADVISORY_DIST_VERSION)"; \
		cp -R $(ADVISORY_SOURCE_DIR)/. $(ADVISORY_DIST_VERSION)/; \
	else \
		echo "[package] No advisories found at $(ADVISORY_SOURCE_DIR); skipping bundle"; \
	fi

.PHONY: pack
pack: dist ## 使用 px-plugin pack 生成 .pxp 元数据包
	@command -v $(PX_PLUGIN_BIN) >/dev/null 2>&1 || { echo "❌ 未找到 $(PX_PLUGIN_BIN)，请先在 PowerXPlugin/tools/cli 构建 CLI 或将 px-plugin 加入 PATH"; exit 1; }
	@if [ -z "$(MARKETPLACE_PUBLIC_KEY)" ]; then \
		echo "❌ 需要提供 PUBLIC_KEY=./keys/marketplace.pem（Marketplace 公钥或证书链）"; \
		exit 1; \
	fi
	@if [ -z "$(KEY_ID)" ]; then \
		echo "❌ 需要提供 KEY_ID=marketplace-dev 用于记录签名来源"; \
		exit 1; \
	fi
	@mkdir -p $(PACK_OUTPUT_DIR)
	@echo "==> 调用 $(PX_PLUGIN_BIN) pack，输出目录：$(abspath $(PACK_OUTPUT_DIR))"
	@$(PX_PLUGIN_BIN) pack \
		--manifest $(abspath $(DIST_DIR))/plugin.yaml \
		--artefact $(abspath $(DIST_DIR)) \
		--output-dir $(abspath $(PACK_OUTPUT_DIR)) \
		--marketplace-public-key $(MARKETPLACE_PUBLIC_KEY) \
		--key-id $(KEY_ID)
	@echo "ℹ️  Run example: make pack KEY_ID=marketplace-dev PUBLIC_KEY=./keys/marketplace.pem"

.PHONY: local-install
local-install: dist local-install-run ## 调用 /admin/plugins/install/local 安装 dist 目录

.PHONY: local-reinstall
local-reinstall: dist ## 禁用当前插件 -> 强制安装 -> 切换并启用目标版本
	@if [ -z "$(API_BASE)" ]; then \
		echo "❌ 需要提供 API_BASE=https://dev-api.powerx.local/api/v1"; \
		exit 1; \
	fi
	@if [ -z "$(TOKEN)" ]; then \
		echo "❌ 需要提供 TOKEN=<admin bearer token>"; \
		exit 1; \
	fi
	@echo "==> [reinstall] disable current plugin: $(PLUGIN_ID)"
	@curl -sS -X POST "$(API_BASE)/admin/plugins/$(PLUGIN_ID)/disable" \
		-H "Authorization: Bearer $(TOKEN)" \
		-H "Content-Type: application/json" >/tmp/powerx-plugin-disable.json || true
	@if [ -s /tmp/powerx-plugin-disable.json ]; then \
		if command -v jq >/dev/null 2>&1; then jq . /tmp/powerx-plugin-disable.json; else cat /tmp/powerx-plugin-disable.json; fi; \
	fi
	@echo "==> [reinstall] force install version=$(VERSION) enable=false"
	@$(MAKE) --no-print-directory local-install-run \
		LOCAL_INSTALL_SRC="$(abspath $(DIST_DIR))" \
		API_BASE="$(API_BASE)" \
		TOKEN="$(TOKEN)" \
		ENABLE=false \
		FORCE=true
	@echo "==> [reinstall] switch_version $(PLUGIN_ID) -> $(VERSION) (enable=true)"
	@PAYLOAD=$$(printf '{"version":"%s","enable":true}' "$(VERSION)"); \
		RESPONSE=$$(curl -sS -X POST "$(API_BASE)/admin/plugins/$(PLUGIN_ID)/switch_version" \
			-H "Authorization: Bearer $(TOKEN)" \
			-H "Content-Type: application/json" \
			-d "$$PAYLOAD"); \
		if command -v jq >/dev/null 2>&1; then \
			CLEAN_RESPONSE=$$(printf '%s' "$$RESPONSE" | tr -d '\000-\010\013\014\016-\037'); \
			if printf '%s' "$$CLEAN_RESPONSE" | jq . >/dev/null 2>&1; then printf '%s' "$$CLEAN_RESPONSE" | jq; else printf '%s\n' "$$RESPONSE"; fi; \
		else \
			echo "$$RESPONSE"; \
		fi

.PHONY: skeleton-reinstall
skeleton-reinstall: local-reinstall ## 兼容旧命令名，独立插件请使用 local-reinstall

.PHONY: local-install-run
local-install-run:
	@if [ -z "$(API_BASE)" ]; then \
		echo "❌ 需要提供 API_BASE（当前值为空）"; \
		echo "   示例: make skeleton-reinstall API_BASE=http://127.0.0.1:8077/api/v1 TOKEN=\$$ADMIN_BEARER_TOKEN"; \
		exit 1; \
	fi
	@if [ -z "$(TOKEN)" ]; then \
		echo "❌ 需要提供 TOKEN=<admin bearer token>"; \
		exit 1; \
	fi
	@if [ ! -d "$(LOCAL_INSTALL_SRC)" ]; then \
		echo "❌ 未找到安装目录：$(LOCAL_INSTALL_SRC)"; \
		echo "   请先执行 make dist 或传入 LOCAL_INSTALL_SRC=/path/to/dist"; \
		exit 1; \
	fi
	@if [ ! -f "$(LOCAL_INSTALL_SRC)/plugin.yaml" ]; then \
		echo "❌ 安装目录缺少 plugin.yaml：$(LOCAL_INSTALL_SRC)"; \
		exit 1; \
	fi
	@if [ ! -x "$(LOCAL_INSTALL_SRC)/backend/bin/plugin" ]; then \
		echo "❌ 安装目录缺少可执行后端入口：$(LOCAL_INSTALL_SRC)/backend/bin/plugin"; \
		echo "   请重新执行 make dist，或修复权限：chmod 755 $(LOCAL_INSTALL_SRC)/backend/bin/plugin"; \
		exit 1; \
	fi
	@if [ ! -x "$(LOCAL_INSTALL_SRC)/backend/bin/migrate" ]; then \
		echo "❌ 安装目录缺少可执行迁移入口：$(LOCAL_INSTALL_SRC)/backend/bin/migrate"; \
		echo "   请重新执行 make dist，或修复权限：chmod 755 $(LOCAL_INSTALL_SRC)/backend/bin/migrate"; \
		exit 1; \
	fi
	@echo "==> 调用 $(API_BASE)/admin/plugins/install/local"
	@echo "    src_dir=$(LOCAL_INSTALL_SRC)"
	@echo "    enable=$(ENABLE) force=$(FORCE)"
	@PAYLOAD=$$(printf '{"src_dir":"%s","enable":%s,"force":%s}' "$(LOCAL_INSTALL_SRC)" "$(ENABLE)" "$(FORCE)"); \
		RESPONSE=$$(curl -sS -X POST "$(API_BASE)/admin/plugins/install/local" \
			-H "Authorization: Bearer $(TOKEN)" \
			-H "Content-Type: application/json" \
			-d "$$PAYLOAD"); \
		if command -v jq >/dev/null 2>&1; then \
			CLEAN_RESPONSE=$$(printf '%s' "$$RESPONSE" | tr -d '\000-\010\013\014\016-\037'); \
			if printf '%s' "$$CLEAN_RESPONSE" | jq . >/dev/null 2>&1; then \
				printf '%s' "$$CLEAN_RESPONSE" | jq; \
				CODE=$$(printf '%s' "$$CLEAN_RESPONSE" | jq -r '.code // 0'); \
			else \
				echo "⚠️ install/local 返回内容不是合法 JSON，输出原始响应："; \
				printf '%s\n' "$$RESPONSE"; \
				CODE=$$(printf '%s' "$$CLEAN_RESPONSE" | sed -n 's/.*"code":[[:space:]]*\([0-9][0-9]*\).*/\1/p' | head -n1); \
			fi; \
			if [ -z "$$CODE" ]; then CODE=0; fi; \
			if [ "$$CODE" != "0" ] && [ "$$CODE" -ge 400 ]; then \
				echo "❌ local-install failed with code=$$CODE"; \
				exit 1; \
			fi; \
		else \
			echo "$$RESPONSE"; \
			if echo "$$RESPONSE" | grep -q '"code":[[:space:]]*[45][0-9][0-9]'; then \
				echo "❌ local-install failed (non-2xx application code)"; \
				exit 1; \
			fi; \
		fi

.PHONY: local-install-pxp
local-install-pxp: ## 先解包 PACKAGE 再调用 local install（当前 .pxp 仍需包含 dist 内容）
	@if [ -z "$(PACKAGE)" ]; then \
		echo "❌ 需要提供 PACKAGE=./dist/artifacts/<plugin>.pxp"; \
		exit 1; \
	fi
	@if [ ! -f "$(PACKAGE)" ]; then \
		echo "❌ 未找到 PACKAGE 文件：$(PACKAGE)"; \
		exit 1; \
	fi
	@rm -rf $(LOCAL_INSTALL_PXP_TMP)
	@mkdir -p $(LOCAL_INSTALL_PXP_TMP)
	@echo "==> 解包 $(PACKAGE) -> $(LOCAL_INSTALL_PXP_TMP)"
	@if echo "$(PACKAGE)" | grep -qiE '\.(zip|pxp)$$'; then \
		unzip -oq "$(PACKAGE)" -d $(LOCAL_INSTALL_PXP_TMP) 2>/dev/null || true; \
	elif echo "$(PACKAGE)" | grep -qiE '\.(tar\.gz|tgz)$$'; then \
		tar -xzf "$(PACKAGE)" -C $(LOCAL_INSTALL_PXP_TMP); \
	elif echo "$(PACKAGE)" | grep -qi '\.tar$$'; then \
		tar -xf "$(PACKAGE)" -C $(LOCAL_INSTALL_PXP_TMP); \
	else \
		cp "$(PACKAGE)" $(LOCAL_INSTALL_PXP_TMP)/package.pxp; \
	fi
	@echo "⚠️  当前 .pxp 仍需配套 dist 产物；若包内只有元数据，请先按文档手工准备 dist。"
	@TARGET_DIR=$$(cd $(LOCAL_INSTALL_PXP_TMP) && \
		if [ -f plugin.yaml ]; then pwd; \
		else \
			first=$$(find . -maxdepth 3 -type f -name plugin.yaml -print -quit); \
			if [ -n "$$first" ]; then dirname "$$first"; fi; \
		fi); \
		if [ -z "$$TARGET_DIR" ]; then \
			echo "❌ 未在 $(LOCAL_INSTALL_PXP_TMP) 中找到 plugin.yaml，无法继续"; \
			exit 1; \
		fi; \
		$(MAKE) --no-print-directory local-install-run \
			LOCAL_INSTALL_SRC=$$TARGET_DIR \
			API_BASE="$(API_BASE)" \
			TOKEN="$(TOKEN)" \
			ENABLE="$(ENABLE)" \
			FORCE="$(FORCE)"
