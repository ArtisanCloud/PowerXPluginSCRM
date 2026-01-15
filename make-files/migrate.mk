# migrate.mk 管理数据库迁移与数据初始化任务

.PHONY: migrate
migrate: ## 运行数据库迁移
	@echo "运行数据库迁移..."
	cd $(BACKEND_DIR) && go run ./cmd/database/main.go migrate

.PHONY: migrate-cmd
migrate-cmd: migrate ## 兼容旧命令，内部调用 migrate 目标
	@:

.PHONY: seed
seed: ## 运行数据种子脚本
	@echo "运行数据种子..."
	cd $(BACKEND_DIR) && go run ./cmd/database/main.go seed

.PHONY: setup-db
setup-db: ## 执行迁移并填充初始数据
	@echo "运行迁移并填充初始数据..."
	cd $(BACKEND_DIR) && go run ./cmd/database/main.go setup

.PHONY: reset-db
reset-db: ## 重置数据库（危险操作）
	@echo "警告: 这将删除所有数据！"
	@read -p "确定要继续吗？[y/N] " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		echo "重置数据库..."; \
		cd $(BACKEND_DIR) && go run ./cmd/database/main.go refresh; \
	else \
		echo "操作已取消"; \
	fi
