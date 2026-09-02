package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/db"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/google/uuid"
)

type commandResult struct {
	Success    bool   `json:"success"`
	DryRun     bool   `json:"dry_run"`
	TenantUUID string `json:"tenant_uuid,omitempty"`
	Scanned    int    `json:"scanned,omitempty"`
	Missing    int    `json:"missing,omitempty"`
	Created    int    `json:"created,omitempty"`
	ErrorCode  string `json:"error_code,omitempty"`
}

func main() {
	flags := flag.NewFlagSet("lead_audit_repair", flag.ContinueOnError)
	tenantUUID := flags.String("tenant-uuid", "", "")
	apply := flags.Bool("apply", false, "")
	flags.SetOutput(os.Stderr)
	if err := flags.Parse(os.Args[1:]); err != nil {
		exitWith("INVALID_ARGUMENTS", "")
	}
	cleanTenantUUID := strings.ToLower(strings.TrimSpace(*tenantUUID))
	parsedTenantUUID, err := uuid.Parse(cleanTenantUUID)
	if err != nil {
		exitWith("INVALID_TENANT_UUID", cleanTenantUUID)
	}
	cleanTenantUUID = parsedTenantUUID.String()
	cfg, err := config.LoadForMigration()
	if err != nil {
		exitWith("CONFIG_LOAD_FAILED", cleanTenantUUID)
	}
	models.InitSchemaFrom(cfg.Database.Schema)
	database, err := db.Connect(cfg.Database)
	if err != nil {
		exitWith("DATABASE_CONNECT_FAILED", cleanTenantUUID)
	}
	service := leadsvc.NewLeadService(leadrepo.NewLeadRepository(database))
	result, err := service.RepairHistoricalAttachmentAudits(context.Background(), cleanTenantUUID, *apply)
	if err != nil {
		exitWith("REPAIR_FAILED", cleanTenantUUID)
	}
	_ = json.NewEncoder(os.Stdout).Encode(commandResult{
		Success: true, DryRun: !*apply, TenantUUID: cleanTenantUUID,
		Scanned: result.Scanned, Missing: result.Missing, Created: result.Created,
	})
}

func exitWith(code, tenantUUID string) {
	_ = json.NewEncoder(os.Stderr).Encode(commandResult{Success: false, DryRun: true, TenantUUID: tenantUUID, ErrorCode: code})
	os.Exit(1)
}
