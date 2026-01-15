# Admin Console Repositories

Repository interfaces and implementations for the Dev Console live here. They should:

- wrap tenant-scoped queries for audit events, configuration changes, and job runs
- reuse `repository.BaseRepository` helpers where possible
- expose pagination and filtering contracts needed by admin services and transports

Avoid leaking `*gorm.DB` outside these packages—constructors should accept the shared tenant-aware
DB handle and return typed repositories.
