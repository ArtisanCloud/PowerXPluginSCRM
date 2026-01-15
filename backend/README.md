# Skeleton Backend

该目录提供最小可运行的 PowerX 插件后端示例：

1. 确保已在仓库根目录执行 `go work sync`。
2. 切换到本目录并运行：

   ```bash
   go run ./cmd/plugin
   ```

3. 验证接口：

   ```bash
   curl http://localhost:8078/api/v1/ping
   ```

4. 模板 CRUD 示例（Tenant 默认为 1，可通过 `X-Tenant-UUID` 覆盖）：

   ```bash
   # 创建
   curl -X POST http://localhost:8078/api/v1/templates \
     -H 'Content-Type: application/json' \
     -d '{"name":"Demo","description":"From skeleton","content":"Hello"}'

   # 查询列表
   curl http://localhost:8078/api/v1/templates
   ```

Skeleton 内部使用内存仓储模拟 constitution 约束（仓储内嵌 BaseRepository 语义、多租户隔离、`SET LOCAL app.tenant_uuid`），实际接入数据库时可直接替换 `internal/templates` 包中的实现。
