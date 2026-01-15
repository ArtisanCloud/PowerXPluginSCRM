# 构建与打包指引

本文针对 `make-files/build.mk` 中的目标进行简要说明，帮助团队快速完成插件的构建、打包与发布。默认假设命令在项目根目录执行。

## 环境准备
- Go 1.21+（根据 `go.mod` 要求调整）。
- Node.js 18+/npm 9+（用于 Nuxt 4 前端构建）。
- `golangci-lint`（可通过 `make dev-setup` 自动安装到 `GOPATH/bin`）。

常用初始化命令：
```bash
make dev-setup
```

## 核心变量
所有变量均可在执行 `make` 时通过命令行覆盖，例如：
```bash
BUILD_DIR=backend/out VERSION=0.1.1 make build
```
主要变量说明：

| 变量名 | 默认值 | 作用 |
| --- | --- | --- |
| `VERSION` | `0.1.0` | 当前插件版本号，影响产物目录与包名 |
| `BUILD_DIR` | `backend/bin` | Go 二进制输出目录 |
| `FRONTEND_BUILD_CMD` | `npm --prefix web-admin run build` | 前端构建命令，可结合 CI 需求调整 |
| `DIST_ROOT` | `dist` | 提供给 PowerX `install/local` 的目录根 |
| `RELEASE_ROOT` | `target` | 发布产物根目录（用于交付或归档） |
| `DOCKER_IMAGE` | `powerx-plugin-base:$(VERSION)` | Docker 镜像名称 |

## 常用命令

| 命令 | 说明 |
| --- | --- |
| `make build` | 构建本机平台的后端二进制到 `$(BUILD_DIR)` |
| `make build-linux` | 交叉编译 Linux/amd64 二进制 |
| `make frontend-build` | 构建 `web-admin/.output` 前端产物 |
| `make dist` | 生成 `$(DIST_ROOT)/$(VERSION)`，用于本地目录模式安装 |
| `make release` | 生成 `$(RELEASE_ROOT)/$(VERSION)`，包含前后端完整发布产物 |
| `make package` | 打包 `dist/<version>` 为 zip（适合远程安装接口） |
| `make package-release` | 打包 `target/<version>` 为 zip（对外发布文件） |
| `make docker-build` | 构建 Docker 镜像 |

查看所有目标：
```bash
make help
```

## 本地安装目录模式
1. 构建后端 & 前端：
   ```bash
   make build frontend-build
   ```
2. 生成目录：
   ```bash
   make dist
   ```
3. 调用 PowerX 的安装接口时，`src_dir` 指向 `$(pwd)/dist/$(VERSION)` 即可，目录内至少包含：
   - `plugin.yaml`
   - `backend/bin/plugin`
   - （可选）`web-admin/.output` 前端静态资源

## Release 产物
`make release` 会在项目根目录下创建 `target/<version>/`，目录结构示例：
```
target/
  0.1.0/
    plugin.yaml
    backend/bin/plugin
    web-admin/.output/
    README.md (若存在)
```

该命令默认依赖 `make build` 与 `make frontend-build`，确保前后端均为最新编译结果。若需要压缩包，可继续执行：
```bash
make package-release
```
压缩包默认命名为 `powerx-plugin-base-<version>-release.zip`。

## 自定义构建输出
- 修改后端输出目录：
  ```bash
  BUILD_DIR=backend/out make build
  ```
- 选择不同二进制平台：
  ```bash
  GOOS=darwin GOARCH=arm64 BUILD_DIR=backend/bin/darwin make build
  ```
- 更换发布目录根位置：
  ```bash
  RELEASE_ROOT=/tmp/px-release make release
  ```

## 注意事项
- `make frontend-build` 仅执行 `npm run build`，需提前安装依赖（`npm --prefix web-admin install`）。
- 如果不打算包含前端产物，可以跳过 `make frontend-build`；此时 `make dist` 会提示 `.output` 不存在。
- 发布前建议执行 `make check`（lint + test）确保质量，并在必要时运行 `make test-coverage`。

如需进一步自动化（例如 CI/CD），可参考以上变量，结合环境注入版本号或发布路径。EOF
