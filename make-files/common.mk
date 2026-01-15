# common.mk 用于定义 Makefile 公共变量与帮助信息

# 基础变量（自动从 manifest / go.mod 推导）
PLUGIN_MANIFEST ?= plugin.yaml
DEFAULT_VERSION := 0.4.0
_MANIFEST_VERSION := $(strip $(shell awk -F': *' '/^version:/ {print $$2; exit}' $(PLUGIN_MANIFEST) 2>/dev/null))
ifeq ($(_MANIFEST_VERSION),)
VERSION ?= $(DEFAULT_VERSION)
else
VERSION ?= $(_MANIFEST_VERSION)
endif

_MANIFEST_ID := $(strip $(shell awk -F': *' '/^id:/ {print $$2; exit}' $(PLUGIN_MANIFEST) 2>/dev/null))
ifeq ($(_MANIFEST_ID),)
PLUGIN_ID ?= com.powerx.plugin.sample
else
PLUGIN_ID ?= $(_MANIFEST_ID)
endif

PLUGIN_SLUG := $(shell echo $(PLUGIN_ID) | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g')
APP_NAME ?= $(PLUGIN_SLUG)

BACKEND_DIR := backend
BACKEND_MODULE_FILE := $(BACKEND_DIR)/go.mod
_GO_MODULE := $(strip $(shell awk '/^module[[:space:]]+/ {print $$2; exit}' $(BACKEND_MODULE_FILE) 2>/dev/null))
ifeq ($(_GO_MODULE),)
GO_MODULE ?= github.com/ArtisanCloud/PowerXPlugin/skeleton/backend
else
GO_MODULE ?= $(_GO_MODULE)
endif

BUILD_DIR := $(BACKEND_DIR)/bin
MAIN_FILE := $(BACKEND_DIR)/cmd/plugin/main.go

DOCKER_IMAGE := $(APP_NAME):$(VERSION)
DOCKER_REGISTRY ?=

DIST_ROOT := dist
DIST_DIR := $(DIST_ROOT)/$(VERSION)
DIST_BACKEND_BIN := $(DIST_DIR)/backend/bin
DIST_WEBADMIN_DIR := $(DIST_DIR)/web-admin
DIST_WEBADMIN_OUTPUT := $(DIST_WEBADMIN_DIR)/.output

FRONTEND_DIR := web-admin
FRONTEND_OUTPUT := $(FRONTEND_DIR)/.output
FRONTEND_BUILD_CMD ?= npm --prefix $(FRONTEND_DIR) run build

_RAW_SCHEMA := $(shell echo $(PLUGIN_ID) | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/_/g')
POWERX_DB_SCHEMA ?= px_$(_RAW_SCHEMA)

RELEASE_ROOT := target
RELEASE_DIR := $(RELEASE_ROOT)/$(VERSION)
RELEASE_BACKEND_BIN := $(RELEASE_DIR)/backend/bin
RELEASE_WEBADMIN_DIR := $(RELEASE_DIR)/web-admin
RELEASE_WEBADMIN_OUTPUT := $(RELEASE_WEBADMIN_DIR)/.output

.DEFAULT_GOAL := help

.PHONY: help
help: ## 显示可用命令列表
	@echo "$(APP_NAME) Makefile"
	@echo ""
	@echo "可用的命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		sed 's|^[^:]*:||' | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf " %-18s %s\n", $$1, $$2}'
