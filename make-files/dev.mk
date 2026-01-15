# dev.mk 侧重于本地开发相关任务

.PHONY: run
run: ## 启动后端服务（开发模式）
	@echo "启动后端服务..."
	cd $(BACKEND_DIR) && \
		POWERX_BIND_ADDR=":8086" \
		POWERX_DB_SCHEMA="$(POWERX_DB_SCHEMA)" \
		POWERX_LOG_LEVEL="debug" \
		POWERX_DEV_MODE=1 \
		go run ./cmd/plugin

.PHONY: dev
dev: migrate run ## 迁移并启动开发服务

.PHONY: dev-setup
dev-setup: ## 初始化本地 Go 依赖与 lint 工具
	@echo "下载 Go 依赖..."
	cd $(BACKEND_DIR) && go mod download
	@echo "检测 golangci-lint..."
	@which golangci-lint > /dev/null || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin
