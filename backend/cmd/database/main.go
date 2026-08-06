// cmd/database/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/cmd/database/migrate"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/cmd/database/seed"
	pluginbootstrap "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/bootstrap"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/db"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s [migrate|seed|refresh]", os.Args[0])
	}
	cmd := os.Args[1]
	ensureMigrationJWTSecret(cmd)
	flag.Parse()

	// 加载迁移配置：安装阶段尚未启动插件运行态，不要求 gateway STS/API Key 凭证。
	cfg, err := config.LoadForMigration()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if err := pluginbootstrap.EnsureLocalIAMSecret(cfg); err != nil {
		log.Fatalf("初始化本地 IAM Secret 失败: %v", err)
	}
	models.InitSchemaFrom(cfg.Database.Schema) // 必须在所有 DB 操作之前

	providerResolver, err := pluginbootstrap.NewProviderResolver(cfg)
	if err != nil {
		log.Fatalf("解析 provider mode 失败: %v", err)
	}
	includeIAM := providerResolver.Mode() == iamservice.ProviderModeLocal
	log.Printf("[provider] mode=%s source=%s includeIAM=%v", providerResolver.Mode(), providerResolver.Source(), includeIAM)
	seedOptions := seed.PluginSeedOptions{
		ProviderMode: string(providerResolver.Mode()),
		DevMode:      isDevMode(cfg),
	}

	ctx := context.Background()
	// 连接数据库
	db, err := db.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	switch cmd {
	case "migrate":
		if err := migrate.MigratePluginModels(ctx, db, includeIAM); err != nil {
			log.Fatal("migrate failed:", err)
		}
		fmt.Println("migrate ok")

	case "seed":
		if includeIAM {
			if err := iamservice.SeedLocalAdmin(ctx, db, cfg, providerResolver.Mode()); err != nil {
				log.Fatal("iam seed failed:", err)
			}
		}
		if err := seed.SeedPluginData(ctx, db, seedOptions); err != nil {
			log.Fatal("seed failed:", err)
		}
		fmt.Println("seed ok")

	case "setup":
		if err := migrate.MigratePluginModels(ctx, db, includeIAM); err != nil {
			log.Fatal("migrate failed:", err)
		}
		fmt.Println("migrate ok")

		if includeIAM {
			if err := iamservice.SeedLocalAdmin(ctx, db, cfg, providerResolver.Mode()); err != nil {
				log.Fatal("iam seed failed:", err)
			}
		}
		if err := seed.SeedPluginData(ctx, db, seedOptions); err != nil {
			log.Fatal("seed failed:", err)
		}
		fmt.Println("seed ok")

	case "refresh":
		// 先 drop database（或 drop all tables）
		if err := migrate.ResetDatabase(ctx, db, cfg.Database); err != nil {
			log.Fatal("reset failed:", err)
		}
		fmt.Println("reset ok")

		// 再 migrate
		if err := migrate.MigratePluginModels(ctx, db, includeIAM); err != nil {
			log.Fatal("migrate failed:", err)
		}
		fmt.Println("migrate ok")

		if includeIAM {
			if err := iamservice.SeedLocalAdmin(ctx, db, cfg, providerResolver.Mode()); err != nil {
				log.Fatal("iam seed failed:", err)
			}
		}
		// 最后 seed
		if err := seed.SeedPluginData(ctx, db, seedOptions); err != nil {
			log.Fatal("seed failed:", err)
		}
		fmt.Println("seed ok")

	default:
		log.Fatalf("Unknown command: %s", cmd)
	}
}

func isDevMode(cfg *config.Config) bool {
	if cfg == nil {
		return false
	}
	if cfg.DevMode {
		return true
	}
	return cfg.Server != nil && cfg.Server.DevMode
}

func ensureMigrationJWTSecret(cmd string) {
	switch cmd {
	case "migrate", "setup", "refresh":
	default:
		return
	}
	_ = os.Setenv("POWERX_ALLOW_EMPTY_CUSTOMER_AUTH_JWT", "1")

	if os.Getenv("POWERX_AUTH_JWTSECRET") != "" {
		return
	}
	if v := os.Getenv("POWERX_SECURITY_JWT_SECRET"); v != "" {
		_ = os.Setenv("POWERX_AUTH_JWTSECRET", v)
		return
	}
	// Migration runs before plugin runtime boots; provide a process-local fallback
	// so config validation does not block schema setup in host install flow.
	const fallback = "migration-temporary-secret"
	_ = os.Setenv("POWERX_AUTH_JWTSECRET", fallback)
	_ = os.Setenv("POWERX_SECURITY_JWT_SECRET", fallback)
}
