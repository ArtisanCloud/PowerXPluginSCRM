package migrate

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	domainmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models"
	domainAcquisitionModel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/acquisition"
	domainLeadCaptureModel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	adminconsoleModel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/admin_console"
	customerModel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/customer"
	iammodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	integrationModel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	leadCaptureModel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	marketplaceModel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	operationsModel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/operations"
	OrgSyncModel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	runtimeOpsModel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/runtime_ops"
	securityModel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/security"
	socialModel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	templateModel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/template"
	toolgrantModel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/tool_grant"
	"github.com/jackc/pgconn"
	"gorm.io/gorm"
)

var businessTables = []interface{}{
	&models.PluginCredential{},
	&models.PluginTenantExt{},
	&templateModel.Template{},
	&marketplaceModel.Listing{},
	&marketplaceModel.ListingAsset{},
	&marketplaceModel.ListingVersion{},
	&marketplaceModel.ChecklistRun{},
	&marketplaceModel.ChecklistItem{},
	&marketplaceModel.PricingPlan{},
	&marketplaceModel.PlanTier{},
	&marketplaceModel.License{},
	&marketplaceModel.LicenseEvent{},
	&marketplaceModel.TaxTransaction{},
	&customerModel.CustomerAccount{},
	&runtimeOpsModel.MCPSession{},
	&runtimeOpsModel.RuntimeAuditEvent{},
	&runtimeOpsModel.QuotaLedger{},
	&runtimeOpsModel.MarketplaceOverage{},
	&operationsModel.SupportChannel{},
	&operationsModel.SupportTicket{},
	&operationsModel.SupportTicketEvent{},
	&operationsModel.ReadinessChecklistItem{},
	&operationsModel.SLAProfile{},
	&operationsModel.SLAAdjustment{},
	&operationsModel.Incident{},
	&operationsModel.IncidentTimelineEntry{},
	&operationsModel.IncidentChecklistItem{},
	&integrationModel.GrantMatrixOverride{},
	&securityModel.BaselineChecklist{},
	&securityModel.AuditReport{},
	&toolgrantModel.Revocation{},
	&toolgrantModel.UsageEvent{},
	&adminconsoleModel.AuditEvent{},
	&adminconsoleModel.ConfigChange{},
	&adminconsoleModel.JobRun{},
	&socialModel.ChannelAccount{},
	&socialModel.AuditEvent{},
	&socialModel.ChannelPlatformSetting{},
	&socialModel.ChannelAuthBinding{},
	&socialModel.ChannelAuthEvent{},
	&socialModel.WeComOpenAuthBinding{},
	&socialModel.WeComOpenAuthEvent{},
	&socialModel.WeComOpenCallbackTask{},
	&socialModel.SyncBaselineJob{},
	&socialModel.SyncConflictRecord{},
	&socialModel.SyncFoundationBinding{},
	&socialModel.SyncJob{},
	&socialModel.SyncCheckpoint{},
	&socialModel.SyncConflict{},
	&socialModel.SyncWritebackPolicy{},
	&socialModel.SyncDeadLetterItem{},
	&OrgSyncModel.SourceAccount{},
	&OrgSyncModel.SourceUnit{},
	&OrgSyncModel.SourceMember{},
	&OrgSyncModel.SourceMemberUnit{},
	&OrgSyncModel.SourceMemberProfile{},
	&OrgSyncModel.UnitMapping{},
	&OrgSyncModel.MemberMapping{},
	&OrgSyncModel.SyncLog{},
	&leadCaptureModel.Lead{},
	&leadCaptureModel.LeadSource{},
	&leadCaptureModel.LeadActivity{},
	&leadCaptureModel.LeadAssignment{},
	&leadCaptureModel.LeadStatusHistory{},
	&leadCaptureModel.LeadSyncTask{},
	&leadCaptureModel.ConversationEvent{},
	&leadCaptureModel.LeadConversationBinding{},
	&leadCaptureModel.LeadConversationPending{},
	&leadCaptureModel.LeadRealtimeProjection{},
	&leadCaptureModel.LeadSourceCatalog{},
	&leadCaptureModel.ChannelRule{},
	&domainLeadCaptureModel.ChannelCode{},
	&domainLeadCaptureModel.CodeWelcomeConfig{},
	&domainLeadCaptureModel.CodeWelcomeSyncAttempt{},
	&domainLeadCaptureModel.ChannelCodeEvent{},
	&domainLeadCaptureModel.LeadAttributionRecord{},
	&domainLeadCaptureModel.CodeConfigChangeLog{},
	&domainAcquisitionModel.StaffLiveCode{},
	&domainAcquisitionModel.StaffWelcomeConfig{},
	&domainAcquisitionModel.StaffWelcomeSyncAttempt{},
	&domainAcquisitionModel.GroupLiveCode{},
}

var iamTables = []interface{}{
	&iammodel.Tenant{},
	&iammodel.User{},
	&iammodel.Member{},
	&iammodel.Role{},
	&iammodel.Permission{},
	&iammodel.Department{},
	&iammodel.MemberRole{},
	&iammodel.RolePermission{},
	&iammodel.RefreshToken{},
	&iammodel.AuditLog{},
}

// MigratePluginModels 只做 AutoMigrate（最小实现）
func MigratePluginModels(ctx context.Context, db *gorm.DB, includeIAM bool) error {
	if db == nil {
		return nil
	}
	tables := append([]interface{}{}, businessTables...)
	if isSQLite(db) {
		tables = filterSQLiteIncompatibleTables(tables)
	}
	if err := ensureLeadCaptureActivityTable(ctx, db); err != nil {
		return err
	}
	if includeIAM {
		tables = append(tables, iamTables...)
	}
	if len(tables) == 0 {
		return nil
	}
	if err := safeAutoMigrate(ctx, db, tables); err != nil {
		return err
	}
	if err := ensureSocialChannelAccountColumns(ctx, db); err != nil {
		return err
	}
	if err := ensureChannelCodeAcquisitionIndexes(ctx, db); err != nil {
		return err
	}
	if err := ensureOpenWorkFoundationIndexes(ctx, db); err != nil {
		return err
	}
	if includeIAM {
		if err := ensureIAMConstraints(ctx, db); err != nil {
			return err
		}
	}
	return nil
}

func safeAutoMigrate(ctx context.Context, db *gorm.DB, tables []interface{}) error {
	for _, tbl := range tables {
		if err := migrateWithTolerance(ctx, db, tbl); err != nil {
			return err
		}
	}
	return nil
}

func migrateWithTolerance(ctx context.Context, db *gorm.DB, table interface{}) error {
	const maxRetries = 5
	for attempts := 0; attempts < maxRetries; attempts++ {
		if err := db.WithContext(ctx).AutoMigrate(table); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == duplicateObjectCode {
				log.Printf("[migrate] duplicate constraint for %T, skipping: %s", table, pgErr.ConstraintName)
				return nil
			}
			// 部分包装错误解析不到 pgErr，但文本包含 already exists/constraint，直接跳过以保证幂等
			if strings.Contains(err.Error(), "already exists") && strings.Contains(err.Error(), "constraint") {
				log.Printf("[migrate] duplicate constraint (fallback) for %T, skipping: %v", table, err)
				return nil
			}
			handled, handleErr := tryHandleAutoMigrateError(ctx, db, table, err)
			if !handled {
				// 兜底：即便未被处理但仍是 42710，也不让迁移失败
				if errors.As(err, &pgErr) && pgErr.Code == duplicateObjectCode {
					log.Printf("[migrate] duplicate constraint (post-handle) for %T, skipping: %v", table, err)
					return nil
				}
				return fmt.Errorf("auto migrate %T failed: %w", table, err)
			}
			if handleErr != nil {
				return handleErr
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("auto migrate retries exceeded for %T", table)
}

func tryHandleAutoMigrateError(ctx context.Context, db *gorm.DB, table interface{}, migrateErr error) (bool, error) {
	if !strings.EqualFold(db.Dialector.Name(), "postgres") {
		return false, nil
	}
	var pgErr *pgconn.PgError
	if !errors.As(migrateErr, &pgErr) {
		return false, nil
	}
	if pgErr.Code != duplicateObjectCode {
		return false, nil
	}
	if strings.TrimSpace(pgErr.ConstraintName) == "" {
		log.Printf("[migrate] duplicate object reported without constraint name, cannot auto fix: %v", migrateErr)
		return false, nil
	}
	tableName, err := resolveTableName(db, table)
	if err != nil {
		return true, err
	}
	if err := dropConstraintIfExists(ctx, db, tableName, pgErr.ConstraintName); err != nil {
		return true, err
	}
	log.Printf("[migrate] dropped duplicate constraint %q on %s, retrying migrate", pgErr.ConstraintName, tableName)
	if retryErr := db.WithContext(ctx).AutoMigrate(table); retryErr != nil {
		return true, retryErr
	}
	return true, nil
}

const duplicateObjectCode = "42710"

func dropConstraintIfExists(ctx context.Context, db *gorm.DB, tableName, constraintName string) error {
	clean := sanitizeConstraintName(constraintName)
	query := fmt.Sprintf(`ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s`, tableName, quoteIdentifier(clean))
	return db.WithContext(ctx).Exec(query).Error
}

func resolveTableName(db *gorm.DB, table interface{}) (string, error) {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(table); err != nil {
		return "", err
	}
	if stmt.Schema == nil || stmt.Schema.Table == "" {
		return "", fmt.Errorf("failed to resolve table name for %T", table)
	}
	return stmt.Schema.Table, nil
}

func quoteIdentifier(name string) string {
	escaped := strings.ReplaceAll(name, "\"", "\"\"")
	return fmt.Sprintf(`"%s"`, escaped)
}

// sanitizeConstraintName strips surrounding quotes and schema prefixes that may appear in pg error messages.
func sanitizeConstraintName(name string) string {
	trimmed := strings.Trim(name, `"`)
	// drop all quotes that may be embedded (fk_"public"_foo)
	trimmed = strings.ReplaceAll(trimmed, `"`, "")
	// remove schema prefix like public_ or public__
	trimmed = strings.TrimPrefix(trimmed, `public_`)
	trimmed = strings.ReplaceAll(trimmed, `public__`, ``)
	return trimmed
}

func isSQLite(db *gorm.DB) bool {
	if db == nil || db.Dialector == nil {
		return false
	}
	return strings.EqualFold(db.Dialector.Name(), "sqlite")
}

func filterSQLiteIncompatibleTables(tables []interface{}) []interface{} {
	filtered := make([]interface{}, 0, len(tables))
	skipped := 0
	for _, tbl := range tables {
		if !isSQLiteSafeTable(tbl) {
			skipped++
			continue
		}
		filtered = append(filtered, tbl)
	}
	if skipped > 0 {
		log.Printf("[migrate] sqlite 环境仅迁移 IAM + 插件核心表，跳过 %d 张业务表", skipped)
	}
	return filtered
}

func isSQLiteSafeTable(tbl interface{}) bool {
	switch tbl.(type) {
	case *models.PluginCredential,
		*models.PluginTenantExt,
		*templateModel.Template,
		*runtimeOpsModel.MCPSession:
		return true
	default:
		return false
	}
}

func ensureSocialChannelAccountColumns(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	tableName := models.S(models.TableSocialChannelAccounts)
	indexName := "uq_social_channel_accounts_identity"
	dropStmt := fmt.Sprintf(`DROP INDEX IF EXISTS %s`, indexName)
	if err := db.WithContext(ctx).Exec(dropStmt).Error; err != nil {
		return err
	}
	createStmt := fmt.Sprintf(
		`CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (tenant_uuid, channel_code, app_type, account_id) WHERE deleted_at IS NULL`,
		indexName, tableName,
	)
	return db.WithContext(ctx).Exec(createStmt).Error
}

func ensureChannelCodeAcquisitionIndexes(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	tableName := domainmodels.S(domainmodels.TableLeadCaptureLeadAttributionRecords)
	indexName := "uq_lead_capture_attribution_primary"
	createStmt := fmt.Sprintf(
		`CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (tenant_uuid, lead_uuid) WHERE is_primary = TRUE`,
		indexName, tableName,
	)
	return db.WithContext(ctx).Exec(createStmt).Error
}

func ensureOpenWorkFoundationIndexes(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	bindingTable := models.S(models.TableSocialWeComAuthBindings)
	identityIndex := "uq_social_wecom_auth_binding_identity"
	dropIdentityStmt := fmt.Sprintf(`DROP INDEX IF EXISTS %s`, identityIndex)
	if err := db.WithContext(ctx).Exec(dropIdentityStmt).Error; err != nil {
		return err
	}
	identityStmt := fmt.Sprintf(
		`CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (tenant_uuid, corp_id, agent_id)`,
		identityIndex, bindingTable,
	)
	if err := db.WithContext(ctx).Exec(identityStmt).Error; err != nil {
		return err
	}
	defaultIndex := "uq_social_wecom_default_binding_per_tenant"
	defaultStmt := fmt.Sprintf(
		`CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (tenant_uuid, channel_code, app_type) WHERE is_default = TRUE AND deleted_at IS NULL`,
		defaultIndex, bindingTable,
	)
	if err := db.WithContext(ctx).Exec(defaultStmt).Error; err != nil {
		return err
	}
	jobTable := models.S(models.TableSocialSyncBaselineJobs)
	idempotencyIndex := "idx_social_sync_jobs_tenant_idempotency"
	idempotencyStmt := fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON %s (tenant_uuid, idempotency_key, created_at DESC)`,
		idempotencyIndex, jobTable,
	)
	if err := db.WithContext(ctx).Exec(idempotencyStmt).Error; err != nil {
		return err
	}
	callbackTaskTable := models.S(models.TableSocialWeComCallbackTasks)
	claimIndex := "idx_social_wecom_callback_tasks_claim"
	claimStmt := fmt.Sprintf(
		`CREATE INDEX IF NOT EXISTS %s ON %s (status, next_retry_at, created_at)`,
		claimIndex, callbackTaskTable,
	)
	if err := db.WithContext(ctx).Exec(claimStmt).Error; err != nil {
		return err
	}
	return nil
}

func ensureLeadCaptureActivityTable(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	oldTable := "lead_capture_events"
	newTable := models.TableLeadCaptureActivities
	oldName := models.S(oldTable)
	newName := models.S(newTable)
	if !db.Migrator().HasTable(newName) && db.Migrator().HasTable(oldName) {
		stmt := fmt.Sprintf(`ALTER TABLE %s RENAME TO %s`, oldName, newTable)
		if err := db.WithContext(ctx).Exec(stmt).Error; err != nil {
			return err
		}
	}
	if db.Migrator().HasTable(newName) {
		if db.Migrator().HasColumn(newName, "event_uuid") {
			stmt := fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN event_uuid TO activity_uuid`, newName)
			if err := db.WithContext(ctx).Exec(stmt).Error; err != nil {
				return err
			}
		}
		if db.Migrator().HasColumn(newName, "event_type") {
			stmt := fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN event_type TO activity_type`, newName)
			if err := db.WithContext(ctx).Exec(stmt).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func ensureIAMConstraints(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	departments := models.S(models.TableIAMDepartments)
	users := models.S(models.TableIAMMembers)
	audits := models.S(models.TableIAMAuditLogs)
	roles := models.S(models.TableIAMRoles)
	rolePerms := models.S(models.TableIAMRolePermissions)
	memberRoles := models.S(models.TableIAMMemberRoles)

	isSqlite := isSQLite(db)
	// SQLite 对表达式索引（lower(col)）在部分环境/驱动下会报 `near "(": syntax error`。
	// 这里用 COLLATE NOCASE 兼容本地开发，达到大小写不敏感的唯一性约束效果。
	deptCodeExpr := "lower(code)"
	memberUsernameExpr := "lower(username)"
	roleCodeExpr := "lower(code)"
	if isSqlite {
		deptCodeExpr = "code COLLATE NOCASE"
		memberUsernameExpr = "username COLLATE NOCASE"
		roleCodeExpr = "code COLLATE NOCASE"
	}

	statements := []string{
		fmt.Sprintf(`CREATE UNIQUE INDEX IF NOT EXISTS idx_iam_departments_tenant_code ON %s (tenant_uuid, %s)`, departments, deptCodeExpr),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_iam_departments_tenant_parent ON %s (tenant_uuid, parent_id)`, departments),
		fmt.Sprintf(`CREATE UNIQUE INDEX IF NOT EXISTS idx_iam_users_tenant_account ON %s (tenant_uuid, user_id)`, users),
		fmt.Sprintf(`CREATE UNIQUE INDEX IF NOT EXISTS idx_iam_users_tenant_username ON %s (tenant_uuid, %s)`, users, memberUsernameExpr),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_iam_users_tenant_status ON %s (tenant_uuid, status)`, users),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_iam_audit_logs_tenant_created ON %s (tenant_uuid, created_at DESC)`, audits),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_iam_audit_logs_actor_created ON %s (actor_member_id, created_at DESC)`, audits),
		fmt.Sprintf(`CREATE UNIQUE INDEX IF NOT EXISTS idx_iam_roles_tenant_code ON %s (tenant_uuid, %s)`, roles, roleCodeExpr),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_iam_roles_scope ON %s (tenant_uuid, scope_type)`, roles),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_iam_role_permissions_version ON %s (role_id, policy_version)`, rolePerms),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_iam_role_permissions_tenant_uuid ON %s (tenant_uuid)`, rolePerms),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_iam_member_roles_role ON %s (role_id)`, memberRoles),
	}

	for _, stmt := range statements {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if err := execIgnoreExists(ctx, db, stmt); err != nil {
			return err
		}
	}
	return backfillRolePermissionTenant(ctx, db)
}

func execIgnoreExists(ctx context.Context, db *gorm.DB, stmt string) error {
	if err := db.WithContext(ctx).Exec(stmt).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "already exists") {
			log.Printf("[migrate] index exists, skip: %s", stmt)
			return nil
		}
		return fmt.Errorf("exec stmt failed: %s: %w", stmt, err)
	}
	return nil
}

func backfillRolePermissionTenant(ctx context.Context, db *gorm.DB) error {
	rolePerms := models.S(models.TableIAMRolePermissions)
	roles := models.S(models.TableIAMRoles)
	var query string
	if strings.EqualFold(db.Dialector.Name(), "sqlite") {
		query = fmt.Sprintf(
			`UPDATE %[1]s SET tenant_uuid = (SELECT tenant_uuid FROM %[2]s WHERE %[2]s.id = %[1]s.role_id) WHERE tenant_uuid IS NULL OR trim(tenant_uuid) = ''`,
			rolePerms, roles,
		)
	} else {
		query = fmt.Sprintf(
			`UPDATE %s rp SET tenant_uuid = r.tenant_uuid FROM %s r WHERE rp.role_id = r.id AND (rp.tenant_uuid IS NULL OR rp.tenant_uuid::text = '')`,
			rolePerms, roles,
		)
	}
	if err := db.WithContext(ctx).Exec(query).Error; err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "tenant_uuid") && strings.Contains(lower, "column") {
			log.Printf("[migrate] tenant_uuid column missing on %s, skip backfill: %v", rolePerms, err)
			return nil
		}
		return err
	}
	return nil
}

func ResetDatabase(ctx context.Context, db *gorm.DB, cfg *config.DatabaseConfig) error {
	if strings.EqualFold(db.Dialector.Name(), "sqlite") || strings.TrimSpace(cfg.Schema) == "" {
		tables := append([]interface{}{}, businessTables...)
		tables = append(tables, iamTables...)
		return db.WithContext(ctx).Migrator().DropTable(tables...)
	}

	// 如果你用 GORM，可以直接 drop 所有表
	// 或者先获取表名，再循环 drop
	// 这里举例简单版本：
	err := db.Exec("DROP SCHEMA " + cfg.Schema + " CASCADE; CREATE SCHEMA " + cfg.Schema + ";").Error
	if err != nil {
		return err
	}
	return nil
}
