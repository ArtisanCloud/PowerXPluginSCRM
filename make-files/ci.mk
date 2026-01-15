# ci.mk 汇总 CI 相关快捷目标

.PHONY: ci
ci: ci-frontend ci-integration ## 运行完整 CI（frontend + integration）

.PHONY: ci-frontend
ci-frontend: ## 运行 web-admin 前端 CI（install/test/lint/build）
	$(MAKE) -C web-admin ci-frontend

.PHONY: ci-integration
ci-integration: ## 运行集成特性 CI 流程
	scripts/ci/integration.sh
