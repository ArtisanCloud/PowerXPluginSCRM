# 本地开发启动指南（Local）

> 目标：从零开始把项目在本地跑起来。
>
> 配置项与联调说明请看：[config.md](./config.md)

## 0. 克隆代码

```bash
git clone https://github.com/ArtisanCloud/PowerXPluginSCRM.git com.powerx.plugin.scrm
cd com.powerx.plugin.scrm
```

默认分支就是 `main`，不需要切分支。

## 1. 安装依赖

### 1.1 后端

```bash
cd backend
go mod tidy
cd ..
```

### 1.2 前端

```bash
cd web-admin
npm install
cd ..
```

## 2. 初始化数据库

```bash
make migrate
```

## 3. 启动服务

### 3.1 启动后端

```bash
make dev
```

### 3.2 启动管理端（另一个终端）

```bash
cd web-admin
npm run dev
```

## 4. 访问页面

浏览器打开：

`http://127.0.0.1:3033`
