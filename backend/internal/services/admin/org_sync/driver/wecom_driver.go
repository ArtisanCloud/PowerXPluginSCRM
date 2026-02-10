package driver

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerLibs/v3/object"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/work"
	workauth "github.com/ArtisanCloud/PowerWeChat/v3/src/work/auth"
	cachex "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/cache"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/sirupsen/logrus"
)

type cachedWeComApp struct {
	app         *work.Work
	fingerprint string
	createdAt   time.Time
}

var (
	weComAppCacheMu sync.RWMutex
	weComAppCache   = map[string]cachedWeComApp{}
	weComCacheMu    sync.RWMutex
	weComCache      kernel.CacheInterface
	weComConfigDir  string
)

// WeComDriver implements org sync via PowerWechat.
type WeComDriver struct{}

type WeComContactTestResult struct {
	MembersTotal int
	UnitsTotal   int
}

// ConfigureWeComCache sets redis cache for PowerWeChat if configured.
func ConfigureWeComCache(cfg *config.Config) {
	cache := cachex.NewPowerWeChatCache(cfg, logrus.WithField("component", "wecom_cache"))
	weComCacheMu.Lock()
	weComCache = cache
	if cfg != nil {
		weComConfigDir = strings.TrimSpace(cfg.ConfigDir)
	}
	weComCacheMu.Unlock()
	if cfg != nil && cfg.Cache != nil {
		logrus.WithFields(logrus.Fields{
			"driver":      cfg.Cache.Driver,
			"host":        cfg.Cache.Host,
			"port":        cfg.Cache.Port,
			"db":          cfg.Cache.DB,
			"redis_url":   cfg.Cache.RedisURL,
			"prefix":      cfg.Cache.Prefix,
			"default_ttl": cfg.Cache.DefaultTTL,
		}).Info("wecom cache config loaded")
	} else {
		logrus.Info("wecom cache config missing")
	}
}

func (d *WeComDriver) FetchUnits(ctx context.Context, account AccountContext) ([]SourceUnitDTO, error) {
	app, err := getWeComApp(account)
	if err != nil {
		return nil, err
	}
	resp, err := app.Department.List(ctx, 0)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("wecom department list response empty")
	}
	if resp.ErrCode != 0 {
		return nil, fmt.Errorf("wecom department list failed: %d %s", resp.ErrCode, resp.ErrMsg)
	}
	out := make([]SourceUnitDTO, 0, len(resp.Departments))
	for _, dept := range resp.Departments {
		if dept == nil {
			continue
		}
		externalID := strconv.Itoa(dept.ID)
		var parentID *string
		if dept.ParentID > 0 {
			val := strconv.Itoa(dept.ParentID)
			parentID = &val
		}
		out = append(out, SourceUnitDTO{
			ExternalUnitID:       externalID,
			ParentExternalUnitID: parentID,
			Name:                 strings.TrimSpace(dept.Name),
			Order:                dept.Order,
			Status:               "active",
		})
	}
	return out, nil
}

func (d *WeComDriver) FetchMembers(ctx context.Context, account AccountContext) ([]SourceMemberDTO, error) {
	app, err := getWeComApp(account)
	if err != nil {
		return nil, err
	}
	out, err := fetchMembersByDepartmentList(ctx, app)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func TestWeComContacts(ctx context.Context, account AccountContext) (*WeComContactTestResult, error) {
	app, err := newWeComContactApp(account.Credentials)
	if err != nil {
		return nil, err
	}
	out, err := fetchMembersByListID(ctx, app)
	if err != nil {
		return nil, err
	}
	return &WeComContactTestResult{MembersTotal: len(out)}, nil
}

func TestWeComContactsDetail(ctx context.Context, account AccountContext) (*WeComContactTestResult, error) {
	app, err := newWeComApp(account.Credentials)
	if err != nil {
		return nil, err
	}
	deptResp, err := app.Department.List(ctx, 0)
	if err != nil {
		return nil, err
	}
	if deptResp == nil {
		return nil, fmt.Errorf("wecom department list response empty")
	}
	if deptResp.ErrCode != 0 {
		return nil, fmt.Errorf("wecom department list failed: %d %s", deptResp.ErrCode, deptResp.ErrMsg)
	}
	members, err := fetchMembersByDepartmentList(ctx, app)
	if err != nil {
		return nil, err
	}
	return &WeComContactTestResult{
		MembersTotal: len(members),
		UnitsTotal:   len(deptResp.Departments),
	}, nil
}

func TestWeComDepartments(ctx context.Context, account AccountContext) (*WeComContactTestResult, error) {
	app, err := newWeComApp(account.Credentials)
	if err != nil {
		return nil, err
	}
	deptResp, err := app.Department.List(ctx, 0)
	if err != nil {
		return nil, err
	}
	if deptResp == nil {
		return nil, fmt.Errorf("wecom department list response empty")
	}
	if deptResp.ErrCode != 0 {
		return nil, fmt.Errorf("wecom department list failed: %d %s", deptResp.ErrCode, deptResp.ErrMsg)
	}
	return &WeComContactTestResult{
		MembersTotal: 0,
		UnitsTotal:   len(deptResp.Departments),
	}, nil
}

func newWeComContactApp(credentials map[string]string) (*work.Work, error) {
	corpID := strings.TrimSpace(credentials["corp_id"])
	secret := strings.TrimSpace(credentials["secret"])
	appSecret := strings.TrimSpace(credentials["app_secret"])
	if appSecret != "" {
		secret = appSecret
	}
	if secret == "" {
		secret = strings.TrimSpace(credentials["app_secret"])
	}
	if corpID == "" || secret == "" {
		return nil, fmt.Errorf("缺少企业 ID（CorpID）或应用 Secret")
	}
	callback := strings.TrimSpace(credentials["oauth_callback"])
	if callback == "" || (!strings.HasPrefix(callback, "http://") && !strings.HasPrefix(callback, "https://")) {
		callback = "http://localhost"
	}
	cache := buildWeComCache()
	logFile, logError := resolveWeComLogFiles()
	httpDebug := parseCredentialBool(credentials["http_debug"])
	logrus.WithFields(logrus.Fields{
		"corp_id": corpID,
		"secret":  secret,
	}).Info("wecom contact init credentials")
	logrus.WithFields(logrus.Fields{
		"corp_id":        corpID,
		"contact_secret": secret,
		"oauth_callback": callback,
		"http_debug":     httpDebug,
	}).Info("wecom contact credentials debug")
	app, err := work.NewWork(&work.UserConfig{
		CorpID:      corpID,
		Secret:      secret,
		CallbackURL: callback,
		Cache:       cache,
		Log: work.Log{
			Level:  "debug",
			File:   logFile,
			Error:  logError,
			Stdout: httpDebug || (logFile == "" && logError == ""),
		},
		HttpDebug: httpDebug,
		OAuth: work.OAuth{
			Callback: callback,
			Scopes:   nil,
		},
	})
	if err != nil {
		return nil, err
	}
	patchWeComAccessToken(app, corpID, secret)
	return app, nil
}

// TestWeComConnection validates credentials by fetching callback IPs.
func TestWeComConnection(ctx context.Context, account AccountContext) ([]string, error) {
	app, err := getWeComApp(account)
	if err != nil {
		return nil, err
	}
	resp, err := app.Base.GetCallbackIP(ctx)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("wecom callback ip response empty")
	}
	if resp.ErrCode != 0 {
		return nil, fmt.Errorf("wecom callback ip failed: %d %s", resp.ErrCode, resp.ErrMsg)
	}
	return resp.IPList, nil
}

func newWeComApp(credentials map[string]string) (*work.Work, error) {
	corpID := strings.TrimSpace(credentials["corp_id"])
	secret := strings.TrimSpace(credentials["secret"])
	appSecret := strings.TrimSpace(credentials["app_secret"])
	if appSecret != "" {
		secret = appSecret
	}
	if secret == "" {
		secret = strings.TrimSpace(credentials["app_secret"])
	}
	agentID := strings.TrimSpace(credentials["agent_id"])
	if corpID == "" || secret == "" {
		return nil, fmt.Errorf("缺少企业 ID（CorpID）或应用 Secret")
	}
	agentIDInt, err := strconv.Atoi(agentID)
	if err != nil {
		return nil, fmt.Errorf("应用 AgentID 无效")
	}
	callback := strings.TrimSpace(credentials["oauth_callback"])
	if callback == "" {
		return nil, fmt.Errorf("缺少回调地址")
	}
	if !strings.HasPrefix(callback, "http://") && !strings.HasPrefix(callback, "https://") {
		return nil, fmt.Errorf("回调地址无效：必须以 http/https 开头")
	}
	cache := buildWeComCache()
	logFile, logError := resolveWeComLogFiles()
	httpDebug := parseCredentialBool(credentials["http_debug"])
	logrus.WithFields(logrus.Fields{
		"corp_id":  corpID,
		"agent_id": agentID,
		"secret":   secret,
	}).Info("wecom init credentials")
	logrus.WithFields(logrus.Fields{
		"corp_id":        corpID,
		"agent_id":       agentID,
		"app_secret":     secret,
		"token":          strings.TrimSpace(credentials["token"]),
		"oauth_callback": callback,
		"http_debug":     httpDebug,
	}).Info("wecom app credentials debug")
	app, err := work.NewWork(&work.UserConfig{
		CorpID:  corpID,
		AgentID: agentIDInt,
		Secret:  secret,
		Token:   strings.TrimSpace(credentials["token"]),
		Cache:   cache,
		Log: work.Log{
			Level:  "debug",
			File:   logFile,
			Error:  logError,
			Stdout: httpDebug || (logFile == "" && logError == ""),
		},
		HttpDebug: httpDebug,
		OAuth: work.OAuth{
			Callback: callback,
			Scopes:   nil,
		},
	})
	if err != nil {
		return nil, err
	}
	patchWeComAccessToken(app, corpID, secret)
	return app, nil
}

func parseCredentialBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func getWeComApp(account AccountContext) (*work.Work, error) {
	cacheKey := buildWeComCacheKey(account)
	fingerprint := buildCredentialFingerprint(account.Credentials)
	weComAppCacheMu.RLock()
	if cached, ok := weComAppCache[cacheKey]; ok && cached.app != nil && cached.fingerprint == fingerprint {
		weComAppCacheMu.RUnlock()
		return cached.app, nil
	}
	weComAppCacheMu.RUnlock()

	app, err := newWeComApp(account.Credentials)
	if err != nil {
		return nil, err
	}
	weComAppCacheMu.Lock()
	weComAppCache[cacheKey] = cachedWeComApp{
		app:         app,
		fingerprint: fingerprint,
		createdAt:   time.Now().UTC(),
	}
	weComAppCacheMu.Unlock()
	return app, nil
}

func buildWeComCache() kernel.CacheInterface {
	weComCacheMu.RLock()
	cache := weComCache
	weComCacheMu.RUnlock()
	return cache
}

func fetchMembersByListID(ctx context.Context, app *work.Work) ([]SourceMemberDTO, error) {
	if app == nil {
		return nil, fmt.Errorf("wecom app is nil")
	}
	const limit = 10000
	cursor := ""
	seen := map[string]struct{}{}
	members := make([]SourceMemberDTO, 0, 1024)
	for {
		resp, err := app.User.ListID(ctx, cursor, limit)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, fmt.Errorf("wecom list_id response empty")
		}
		if resp.ErrCode != 0 {
			return nil, fmt.Errorf("wecom list_id failed: %d %s", resp.ErrCode, resp.ErrMsg)
		}
		for _, item := range resp.DeptUser {
			if item == nil {
				continue
			}
			userID := strings.TrimSpace(item.UserID)
			if userID == "" {
				continue
			}
			if _, ok := seen[userID]; ok {
				continue
			}
			seen[userID] = struct{}{}
			members = append(members, SourceMemberDTO{
				ExternalMemberID: userID,
				Name:             "",
				Phone:            "",
				Email:            "",
				ProfileStatus:    "limited",
				Status:           "active",
			})
		}
		if strings.TrimSpace(resp.NextCursor) == "" {
			break
		}
		cursor = resp.NextCursor
	}
	return members, nil
}

func fetchMembersByDepartmentList(ctx context.Context, app *work.Work) ([]SourceMemberDTO, error) {
	if app == nil {
		return nil, fmt.Errorf("wecom app is nil")
	}
	deptResp, err := app.Department.List(ctx, 0)
	if err != nil {
		return nil, err
	}
	if deptResp == nil {
		return nil, fmt.Errorf("wecom department list response empty")
	}
	if deptResp.ErrCode != 0 {
		return nil, fmt.Errorf("wecom department list failed: %d %s", deptResp.ErrCode, deptResp.ErrMsg)
	}
	deptIDs := make([]int, 0, len(deptResp.Departments))
	for _, dept := range deptResp.Departments {
		if dept == nil || dept.ID <= 0 {
			continue
		}
		deptIDs = append(deptIDs, dept.ID)
	}
	if len(deptIDs) == 0 {
		return []SourceMemberDTO{}, nil
	}
	memberMap := map[string]*SourceMemberDTO{}
	for _, deptID := range deptIDs {
		userResp, err := app.User.GetDetailedDepartmentUsers(ctx, deptID, 0)
		if err != nil {
			return nil, err
		}
		if userResp == nil {
			return nil, fmt.Errorf("wecom user list response empty")
		}
		if userResp.ErrCode != 0 {
			return nil, fmt.Errorf("wecom user list failed: %d %s", userResp.ErrCode, userResp.ErrMsg)
		}
		for _, user := range userResp.UserList {
			if user == nil {
				continue
			}
			userID := strings.TrimSpace(user.UserID)
			if userID == "" {
				continue
			}
			dto := memberMap[userID]
			if dto == nil {
				dto = &SourceMemberDTO{
					ExternalMemberID: userID,
					DepartmentOrders: map[string]int{},
				}
				memberMap[userID] = dto
			}
			name := strings.TrimSpace(user.Name)
			phone := strings.TrimSpace(user.Mobile)
			email := strings.TrimSpace(user.Email)
			if dto.Name == "" && name != "" {
				dto.Name = name
			}
			if dto.Phone == "" && phone != "" {
				dto.Phone = phone
			}
			if dto.Email == "" && email != "" {
				dto.Email = email
			}
			dto.DepartmentIDs = mergeDepartmentIDs(dto.DepartmentIDs, user.Department, user.MainDepartment)
			for idx, depID := range user.Department {
				if depID <= 0 || idx >= len(user.Order) {
					continue
				}
				depKey := strconv.Itoa(depID)
				if depKey == "" {
					continue
				}
				dto.DepartmentOrders[depKey] = user.Order[idx]
			}
			dto.ProfileStatus = resolveProfileStatus(dto.Name, dto.Phone, dto.Email)
			status := resolveMemberStatus(user.Status)
			if dto.Status == "" || (dto.Status == "active" && status != "active") {
				dto.Status = status
			}
		}
	}
	userIDs := make([]string, 0, len(memberMap))
	for userID := range memberMap {
		userIDs = append(userIDs, userID)
	}
	reporter := progressFromContext(ctx)
	if reporter != nil {
		reporter(0, len(userIDs), "fetch_user_detail")
	}
	for idx, userID := range userIDs {
		detail, err := app.User.Get(ctx, userID)
		if err != nil {
			return nil, err
		}
		if detail == nil {
			return nil, fmt.Errorf("wecom user get response empty")
		}
		if detail.ErrCode != 0 {
			return nil, fmt.Errorf("wecom user get failed: %d %s", detail.ErrCode, detail.ErrMsg)
		}
		dto := memberMap[userID]
		if dto == nil || detail.UserDetail == nil {
			continue
		}
		if name := strings.TrimSpace(detail.Name); name != "" {
			dto.Name = name
		}
		if mobile := strings.TrimSpace(detail.Mobile); mobile != "" {
			dto.Phone = mobile
		}
		if email := strings.TrimSpace(detail.Email); email != "" {
			dto.Email = email
		}
		dto.BizMail = strings.TrimSpace(detail.BizMail)
		dto.Position = strings.TrimSpace(detail.Position)
		dto.Address = strings.TrimSpace(detail.Address)
		dto.AvatarURL = strings.TrimSpace(detail.Avatar)
		if detail.MainDepartment > 0 {
			dto.MainDepartmentID = strconv.Itoa(detail.MainDepartment)
		}
		dto.DepartmentIDs = mergeDepartmentIDs(dto.DepartmentIDs, detail.Department, detail.MainDepartment)
		for idx2, depID := range detail.Department {
			if depID <= 0 || idx2 >= len(detail.Order) {
				continue
			}
			depKey := strconv.Itoa(depID)
			if depKey == "" {
				continue
			}
			dto.DepartmentOrders[depKey] = detail.Order[idx2]
		}
		dto.ProfileStatus = resolveProfileStatus(dto.Name, dto.Phone, dto.Email)
		if reporter != nil {
			if (idx+1)%5 == 0 || idx+1 == len(userIDs) {
				reporter(idx+1, len(userIDs), "fetch_user_detail")
			}
		}
	}
	out := make([]SourceMemberDTO, 0, len(memberMap))
	for _, member := range memberMap {
		if member == nil || strings.TrimSpace(member.ExternalMemberID) == "" {
			continue
		}
		if member.ProfileStatus == "" {
			member.ProfileStatus = resolveProfileStatus(member.Name, member.Phone, member.Email)
		}
		if member.Status == "" {
			member.Status = "active"
		}
		if member.DepartmentOrders == nil {
			member.DepartmentOrders = map[string]int{}
		}
		out = append(out, *member)
	}
	return out, nil
}

func mergeDepartmentIDs(existing []string, list []int, mainDepartment int) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(existing))
	for _, item := range existing {
		clean := strings.TrimSpace(item)
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	for _, id := range list {
		if id <= 0 {
			continue
		}
		val := strconv.Itoa(id)
		if _, ok := seen[val]; ok {
			continue
		}
		seen[val] = struct{}{}
		out = append(out, val)
	}
	if mainDepartment > 0 {
		val := strconv.Itoa(mainDepartment)
		if _, ok := seen[val]; !ok {
			out = append(out, val)
		}
	}
	return out
}

func resolveProfileStatus(name, phone, email string) string {
	if strings.TrimSpace(name) == "" && strings.TrimSpace(phone) == "" && strings.TrimSpace(email) == "" {
		return "limited"
	}
	return "full"
}

func resolveMemberStatus(status int) string {
	switch status {
	case 1:
		return "active"
	case 2:
		return "disabled"
	case 4:
		return "inactive"
	case 5:
		return "quit"
	default:
		return "active"
	}
}

func resolveWeComLogFiles() (string, string) {
	baseDir := resolveWeComBaseDir()
	if baseDir == "" {
		return "", ""
	}
	logDir := filepath.Join(baseDir, "logs", "wechat")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		logrus.WithError(err).WithField("path", logDir).Warn("wecom log dir create failed")
		return "", ""
	}
	infoPath := filepath.Join(logDir, "info.log")
	errorPath := filepath.Join(logDir, "error.log")
	logrus.WithFields(logrus.Fields{
		"info_log":  infoPath,
		"error_log": errorPath,
	}).Info("wecom log path resolved")
	return infoPath, errorPath
}

func resolveWeComBaseDir() string {
	cfgPath := resolveConfigDir()
	if cfgPath != "" {
		base := cfgPath
		switch strings.ToLower(filepath.Base(base)) {
		case "etc", "config":
			base = filepath.Dir(base)
		}
		return base
	}
	return ""
}

func resolveConfigDir() string {
	weComCacheMu.RLock()
	cfgDir := weComConfigDir
	weComCacheMu.RUnlock()
	if cfgDir != "" {
		return cfgDir
	}
	if v := os.Getenv("CONFIG_PATH"); v != "" {
		resolved := strings.TrimSpace(v)
		if resolved != "" {
			info, err := os.Stat(resolved)
			if err == nil {
				if info.IsDir() {
					return resolved
				}
				return filepath.Dir(resolved)
			}
		}
	}
	return ""
}

func buildWeComCacheKey(account AccountContext) string {
	return buildWeComCacheKeyFromFields(account.ChannelCode, account.AppType, account.ChannelAccountUUID, account.Credentials)
}

// InvalidateCache clears cached wecom app for a channel account.
func InvalidateCache(channel, appType, accountUUID string) {
	cacheKey := buildWeComCacheKeyFromFields(channel, appType, accountUUID, nil)
	if cacheKey == "" {
		return
	}
	weComAppCacheMu.Lock()
	delete(weComAppCache, cacheKey)
	weComAppCacheMu.Unlock()
}

func buildWeComCacheKeyFromFields(channel, appType, accountUUID string, credentials map[string]string) string {
	channel = strings.ToLower(strings.TrimSpace(channel))
	appType = strings.ToLower(strings.TrimSpace(appType))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if channel == "" || appType == "" {
		return ""
	}
	if accountUUID == "" {
		return fmt.Sprintf("%s/%s/%s", channel, appType, buildCredentialFingerprint(credentials))
	}
	return fmt.Sprintf("%s/%s/%s", channel, appType, accountUUID)
}

func buildCredentialFingerprint(credentials map[string]string) string {
	if len(credentials) == 0 {
		return ""
	}
	keys := make([]string, 0, len(credentials))
	for key := range credentials {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, key := range keys {
		sb.WriteString(key)
		sb.WriteString("=")
		sb.WriteString(strings.TrimSpace(credentials[key]))
		sb.WriteString(";")
	}
	return sb.String()
}

func patchWeComAccessToken(app *work.Work, corpID, secret string) {
	if app == nil || corpID == "" || secret == "" {
		return
	}
	component := app.GetComponent("AccessToken")
	token, ok := component.(*workauth.AccessToken)
	if !ok || token == nil {
		return
	}
	cacheKey := buildWeComAccessTokenCacheKey(token.CachePrefix, corpID, secret)
	if cacheKey != "" {
		token.SetCacheKey(cacheKey)
	}
	token.GetCredentials = func() *object.StringMap {
		return &object.StringMap{
			"corpid":     corpID,
			"corpsecret": secret,
		}
	}
}

func buildWeComAccessTokenCacheKey(prefix, corpID, secret string) string {
	if corpID == "" || secret == "" {
		return ""
	}
	data := fmt.Sprintf("%s%s", corpID, secret)
	sum := md5.Sum([]byte(data))
	return prefix + hex.EncodeToString(sum[:])
}
