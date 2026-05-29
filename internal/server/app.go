package server

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const apiPrefix = "/api/v1"

type Options struct {
	DataDir string
	Version string
}

type App struct {
	DataDir string
	Version string
	DB      *sql.DB
	Secret  []byte

	Services *ServiceManager
}

type APIError struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

func New(opts Options) (*App, error) {
	if opts.DataDir == "" {
		return nil, errors.New("data dir is required")
	}
	if opts.Version == "" {
		opts.Version = "dev"
	}
	if err := os.MkdirAll(opts.DataDir, 0755); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(opts.DataDir, "database", "msm-free.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	app := &App{DataDir: opts.DataDir, Version: opts.Version, DB: db}
	if err := app.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	if err := app.ensureSecret(); err != nil {
		db.Close()
		return nil, err
	}
	app.Services = NewServiceManager(app)
	return app, nil
}

func (a *App) Close() {
	if a.DB != nil {
		_ = a.DB.Close()
	}
}

func (a *App) EnsureBaseLayout() error {
	dirs := []string{
		"configs/mosdns/sub_config",
		"configs/mosdns/rules",
		"configs/mosdns/webinfo",
		"configs/logs",
		"configs/mihomo/rules",
		"configs/mihomo/proxy_providers",
		"configs/mihomo/ui",
		"configs/network",
		"configs/singbox",
		"data/binaries/mosdns",
		"data/binaries/mihomo",
		"data/binaries/zashboard",
		"logs/supervisor",
		"database",
		"backups",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(a.DataDir, d), 0755); err != nil {
			return err
		}
	}
	if err := a.ensureDefaultConfigs(); err != nil {
		return err
	}
	return nil
}

func (a *App) Router() http.Handler {
	mux := http.NewServeMux()
	a.registerRoutes(mux)
	return a.withCommonMiddleware(mux)
}

func (a *App) withCommonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if strings.HasPrefix(r.URL.Path, apiPrefix) && !a.publicAPI(r.URL.Path) {
			user, err := a.authenticateRequest(r)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "请提供认证令牌")
				return
			}
			if !a.authorizeRequest(user, r) {
				writeError(w, http.StatusForbidden, "forbidden", "当前角色没有执行该操作的权限")
				return
			}
			r = r.WithContext(context.WithValue(r.Context(), userContextKey{}, user))
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) publicAPI(path string) bool {
	public := []string{
		"/api/v1/version",
		"/api/v1/setup/check",
		"/api/v1/setup/system-info",
		"/api/v1/setup/network-interfaces",
		"/api/v1/setup/privilege",
		"/api/v1/setup/initialize",
		"/api/v1/setup/activate",
		"/api/v1/auth/login",
		"/api/v1/auth/refresh",
		"/api/v1/license-activation/status",
		"/api/v1/license-activation/hardware-fingerprint",
	}
	for _, p := range public {
		if path == p {
			return true
		}
	}
	return strings.HasPrefix(path, "/api/v1/setup/download/")
}

func (a *App) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/version", a.handleVersion)

	mux.HandleFunc("GET /api/v1/setup/check", a.handleSetupCheck)
	mux.HandleFunc("GET /api/v1/setup/system-info", a.handleSetupSystemInfo)
	mux.HandleFunc("GET /api/v1/setup/network-interfaces", a.handleSetupNetworkInterfaces)
	mux.HandleFunc("GET /api/v1/setup/privilege", a.handleSetupPrivilege)
	mux.HandleFunc("GET /api/v1/setup/config", a.handleSetupGetConfig)
	mux.HandleFunc("PUT /api/v1/setup/config", a.handleSetupPutConfig)
	mux.HandleFunc("POST /api/v1/setup/initialize", a.handleSetupInitialize)
	mux.HandleFunc("POST /api/v1/setup/activate", a.handleSetupActivate)
	mux.HandleFunc("POST /api/v1/setup/reset", a.handleSetupReset)
	mux.HandleFunc("GET /api/v1/setup/download/{component}", a.handleSetupDownload)

	mux.HandleFunc("POST /api/v1/auth/login", a.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", a.handleLogout)
	mux.HandleFunc("POST /api/v1/auth/refresh", a.handleRefresh)
	mux.HandleFunc("GET /api/v1/auth/me", a.handleMe)
	mux.HandleFunc("GET /api/v1/profile", a.handleProfile)
	mux.HandleFunc("PUT /api/v1/profile", a.handleProfileUpdate)
	mux.HandleFunc("POST /api/v1/profile/password", a.handleChangePassword)

	mux.HandleFunc("GET /api/v1/users", a.handleUsers)
	mux.HandleFunc("POST /api/v1/users", a.handleCreateUser)
	mux.HandleFunc("GET /api/v1/users/{id}", a.handleGetUser)
	mux.HandleFunc("PUT /api/v1/users/{id}", a.handleUpdateUser)
	mux.HandleFunc("DELETE /api/v1/users/{id}", a.handleDeleteUser)
	mux.HandleFunc("POST /api/v1/users/{id}/reset-password", a.handleResetUserPassword)
	mux.HandleFunc("POST /api/v1/users/{id}/toggle-active", a.handleToggleUser)
	mux.HandleFunc("GET /api/v1/users/stats", a.handleUserStats)
	mux.HandleFunc("GET /api/v1/api-tokens", a.handleAPITokens)
	mux.HandleFunc("POST /api/v1/api-tokens", a.handleCreateAPIToken)
	mux.HandleFunc("DELETE /api/v1/api-tokens/{id}", a.handleRevokeAPIToken)

	mux.HandleFunc("GET /api/v1/services", a.handleServices)
	mux.HandleFunc("GET /api/v1/services/{name}", a.handleService)
	mux.HandleFunc("GET /api/v1/services/{name}/exists", a.handleServiceExists)
	mux.HandleFunc("POST /api/v1/services/{name}/start", a.handleServiceStart)
	mux.HandleFunc("POST /api/v1/services/{name}/stop", a.handleServiceStop)
	mux.HandleFunc("POST /api/v1/services/{name}/restart", a.handleServiceRestart)
	mux.HandleFunc("GET /api/v1/services/{name}/logs", a.handleServiceLogs)
	mux.HandleFunc("PUT /api/v1/services/{name}/config", a.handleServiceConfig)
	mux.HandleFunc("GET /api/v1/services/proxy", a.handleProxySummary)

	mux.HandleFunc("GET /api/v1/monitor/system", a.handleMonitorSystem)
	mux.HandleFunc("GET /api/v1/monitor/hardware", a.handleMonitorHardware)
	mux.HandleFunc("GET /api/v1/monitor/resources", a.handleMonitorResources)
	mux.HandleFunc("GET /api/v1/monitor/network", a.handleMonitorNetwork)
	mux.HandleFunc("GET /api/v1/monitor/history", a.handleMonitorHistory)
	mux.HandleFunc("GET /api/v1/monitor/stats", a.handleMonitorStats)
	mux.HandleFunc("GET /api/v1/system/diagnostics", a.handleDiagnostics)
	mux.HandleFunc("GET /api/v1/network/info", a.handleNetworkInfo)
	mux.HandleFunc("POST /api/v1/network/apply", a.handleNFTApply)
	mux.HandleFunc("POST /api/v1/network/stop", a.handleNFTClear)
	mux.HandleFunc("GET /api/v1/netlink/nftables", a.handleNFTInfo)
	mux.HandleFunc("POST /api/v1/netlink/nftables/apply", a.handleNFTApply)
	mux.HandleFunc("POST /api/v1/netlink/nftables/clear", a.handleNFTClear)
	mux.HandleFunc("GET /api/v1/netlink/nftables/status", a.handleNFTStatus)

	mux.HandleFunc("GET /api/v1/config/tree", a.handleConfigTree)
	mux.HandleFunc("GET /api/v1/config/file", a.handleConfigFile)
	mux.HandleFunc("PUT /api/v1/config/file", a.handleConfigFilePut)
	mux.HandleFunc("POST /api/v1/config/file", a.handleConfigFileCreate)
	mux.HandleFunc("DELETE /api/v1/config/file", a.handleConfigFileDelete)
	mux.HandleFunc("POST /api/v1/config/directory", a.handleConfigDirectory)
	mux.HandleFunc("POST /api/v1/config/copy", a.handleConfigCopy)
	mux.HandleFunc("POST /api/v1/config/rename", a.handleConfigRename)
	mux.HandleFunc("POST /api/v1/config/validate", a.handleConfigValidate)
	mux.HandleFunc("GET /api/v1/config/download", a.handleConfigDownload)
	mux.HandleFunc("POST /api/v1/config/upload", a.handleConfigUpload)
	mux.HandleFunc("GET /api/v1/config/backups", a.handleConfigBackups)
	mux.HandleFunc("POST /api/v1/config/backup", a.handleConfigBackup)
	mux.HandleFunc("GET /api/v1/config/backup/download", a.handleConfigBackupDownload)
	mux.HandleFunc("POST /api/v1/config/restore", a.handleConfigRestore)

	mux.HandleFunc("GET /api/v1/history", a.handleHistory)
	mux.HandleFunc("POST /api/v1/history", a.handleHistoryCreate)
	mux.HandleFunc("GET /api/v1/history/{id}", a.handleHistoryGet)
	mux.HandleFunc("POST /api/v1/history/{id}/rollback", a.handleHistoryRollback)
	mux.HandleFunc("POST /api/v1/history/{id}/star", a.handleHistoryStar)
	mux.HandleFunc("DELETE /api/v1/history/{id}", a.handleHistoryDelete)
	mux.HandleFunc("GET /api/v1/history/compare", a.handleHistoryCompare)

	mux.HandleFunc("GET /api/v1/logs/{service}", a.handleLogs)
	mux.HandleFunc("DELETE /api/v1/logs/{service}", a.handleLogsClear)
	mux.HandleFunc("GET /api/v1/logs/{service}/download", a.handleLogsDownload)
	mux.HandleFunc("GET /api/v1/logs/{service}/stats", a.handleLogsStats)

	mux.HandleFunc("GET /api/v1/settings", a.handleSettingsGet)
	mux.HandleFunc("PUT /api/v1/settings", a.handleSettingsPut)

	mux.HandleFunc("GET /api/v1/license-activation/status", a.handleLicenseStatus)
	mux.HandleFunc("GET /api/v1/license-activation/hardware-fingerprint", a.handleHardwareFingerprint)
	mux.HandleFunc("POST /api/v1/license-activation/activate", a.handleLicenseNoop)
	mux.HandleFunc("POST /api/v1/license-activation/deactivate", a.handleLicenseNoop)
	mux.HandleFunc("POST /api/v1/license-activation/refresh", a.handleLicenseNoop)

	a.registerUpdateRoutes(mux)
	a.registerMosDNSRoutes(mux)
	a.registerMihomoRoutes(mux)
	a.registerSingBoxRoutes(mux)
	a.registerEvents(mux)
	a.registerStatic(mux)
}

func (a *App) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"version":    a.Version,
			"go_version": runtime.Version(),
			"platform":   runtime.GOOS + "/" + runtime.GOARCH,
			"build_time": "",
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, APIError{Error: code, Message: message})
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 16<<20)).Decode(dst)
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func (a *App) ensureSecret() error {
	var value string
	err := a.DB.QueryRow(`select value from settings where key='jwt_secret'`).Scan(&value)
	if err == nil && value != "" {
		a.Secret = []byte(value)
		return nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	value = randomHex(48)
	_, err = a.DB.Exec(`insert or replace into settings(key,value,updated_at) values('jwt_secret',?,?)`, value, time.Now())
	if err != nil {
		return err
	}
	a.Secret = []byte(value)
	return nil
}

func localIPs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []string
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.To4() != nil {
				out = append(out, ipNet.IP.String())
			}
		}
	}
	return out
}

func nowString() string {
	return time.Now().Format(time.RFC3339)
}

func stringValue(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func boolValue(m map[string]any, key string, def bool) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return def
}

func intValue(m map[string]any, key string, def int) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return def
}
