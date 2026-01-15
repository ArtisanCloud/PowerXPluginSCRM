# test.mk 汇总测试与代码质量相关目标

.PHONY: test
test: ## 执行 Go 单元测试
	@echo "运行测试..."
	cd $(BACKEND_DIR) && go test ./...

.PHONY: test-admin
test-admin: ## 运行 web-admin 测试
	@echo "运行 Web Admin 测试..."
	cd $(FRONTEND_DIR) && npm run test

.PHONY: lint-admin
lint-admin: ## 运行 web-admin Lint
	@echo "运行 Web Admin Lint..."
	cd $(FRONTEND_DIR) && npm run lint -- --max-warnings=0

.PHONY: build-admin
build-admin: ## 构建 web-admin 产物
	@echo "构建 Web Admin..."
	cd $(FRONTEND_DIR) && npm run build

.PHONY: test-coverage
test-coverage: ## 执行测试并生成覆盖率报告
	@echo "运行测试并生成覆盖率报告..."
	cd $(BACKEND_DIR) && go test -coverprofile=coverage.out ./...
	cd $(BACKEND_DIR) && go tool cover -html=coverage.out -o coverage.html

.PHONY: lint
lint: ## 运行 golangci-lint
	@echo "运行代码检查..."
	cd $(BACKEND_DIR) && golangci-lint run

.PHONY: fmt
fmt: ## 使用 go fmt 格式化代码
	@echo "格式化代码..."
	cd $(BACKEND_DIR) && go fmt ./...

.PHONY: mod-tidy
mod-tidy: ## 整理 Go 模块依赖
	@echo "整理 Go 模块依赖..."
	cd $(BACKEND_DIR) && go mod tidy

.PHONY: test-all
test-all: fmt lint lint-admin test test-admin build-admin ## 运行后端/前端全量验证
	@echo "所有测试与构建完成"

.PHONY: integration-smoke
integration-smoke: ## 运行集成回归演练（Webhook Replay + Nuxt 构建）
	@echo "运行 Webhook Replay Drill..."
	cd $(BACKEND_DIR) && go test ./internal/services/integration -run TestWebhookService_ReplayAttemptFlow -count=1
	$(MAKE) frontend-build

.PHONY: check
check: lint lint-admin test test-admin security-audit ## 运行 lint + test + security audit
