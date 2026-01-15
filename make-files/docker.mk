# docker.mk 存放与 Docker 镜像相关的任务

.PHONY: docker-build
docker-build: ## 构建 Docker 镜像
	@echo "构建 Docker 镜像..."
	docker build -t $(DOCKER_IMAGE) .

.PHONY: docker-run
docker-run: ## 使用 Docker 运行插件
	@echo "运行 Docker 容器..."
	docker run --rm -p 8086:8086 \
		-e POWERX_BIND_ADDR=":8086" \
		-e POWERX_DB_SCHEMA="$(POWERX_DB_SCHEMA)" \
		-e POWERX_LOG_LEVEL="debug" \
		$(DOCKER_IMAGE)

.PHONY: docker-push
docker-push: ## 推送 Docker 镜像到仓库
	@echo "推送 Docker 镜像..."
	@if [ -z "$(DOCKER_REGISTRY)" ]; then \
			echo "错误: DOCKER_REGISTRY 未设置"; \
			exit 1; \
		fi
	docker tag $(DOCKER_IMAGE) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE)
