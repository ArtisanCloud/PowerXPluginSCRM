# com.powerx.plugins.scrm Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-01-15

## Active Technologies
- Go 1.24（后端），Node 20 + TypeScript 4 + Nuxt 4（前端） + Gin + GORM（后端），Nuxt UI 3.3.x（前端） (001-lead-management)
- PostgreSQL（插件 schema） (001-lead-management)
- Go 1.24（backend）, Node 20 + TypeScript 4 + Nuxt 4（web-admin） + Gin, GORM, Nuxt UI 3.3.x (001-org-sync)
- Go 1.24（backend）, TypeScript 4 + Nuxt 4（web-admin） + Gin, GORM, PowerWechat SDK, Nuxt UI 3.3.x, Pinia (004-lead-managment)
- PostgreSQL（主存储，租户隔离/RLS），Redis（可选缓存/队列） (004-lead-managment)
- PostgreSQL（plugin schema + RLS），Redis（可选，用于异步任务/重试队列） (005-channel-code-acquisition)

- Go 1.24 (backend), Node 20 + TypeScript 4 + Nuxt 4 (web-admin) + Gin + GORM (backend), Nuxt UI 3.3.x (frontend) (001-social-channel-governance)

## Project Structure

```text
backend/
frontend/
tests/
```

## Commands

npm test && npm run lint

## Code Style

Go 1.24 (backend), Node 20 + TypeScript 4 + Nuxt 4 (web-admin): Follow standard conventions

## Recent Changes
- 005-channel-code-acquisition: Added Go 1.24（backend）, TypeScript 4 + Nuxt 4（web-admin） + Gin, GORM, PowerWechat SDK, Nuxt UI 3.3.x, Pinia
- 004-lead-managment: Added Go 1.24（backend）, TypeScript 4 + Nuxt 4（web-admin） + Gin, GORM, PowerWechat SDK, Nuxt UI 3.3.x, Pinia
- 001-org-sync: Added Go 1.24（backend）, Node 20 + TypeScript 4 + Nuxt 4（web-admin） + Gin, GORM, Nuxt UI 3.3.x


<!-- MANUAL ADDITIONS START -->
Always respond in Chinese-simplified
<!-- MANUAL ADDITIONS END -->
