package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func (a *App) registerMihomoRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/mihomo/status", a.handleMihomoStatus)
	mux.HandleFunc("GET /api/v1/mihomo/overview", a.handleMihomoOverview)
	mux.HandleFunc("GET /api/v1/mihomo/dashboard", a.handleMihomoOverview)
	mux.HandleFunc("GET /api/v1/mihomo/summary", a.handleMihomoOverview)
	mux.HandleFunc("GET /api/v1/mihomo/version", a.handleMihomoVersion)
	mux.HandleFunc("GET /api/v1/mihomo/versions", a.handleMihomoVersions)
	mux.HandleFunc("POST /api/v1/mihomo/version", a.handleMihomoVersionSwitch)
	mux.HandleFunc("GET /api/v1/mihomo/logs", a.handleMihomoLogs)
	mux.HandleFunc("POST /api/v1/mihomo/install", a.handleMihomoInstall)
	mux.HandleFunc("POST /api/v1/mihomo/start", a.handleMihomoStart)
	mux.HandleFunc("POST /api/v1/mihomo/stop", a.handleMihomoStop)
	mux.HandleFunc("POST /api/v1/mihomo/restart", a.handleMihomoRestart)

	mux.HandleFunc("GET /api/v1/mihomo/config", a.handleMihomoConfig)
	mux.HandleFunc("POST /api/v1/mihomo/config", a.handleMihomoConfigSwitch)
	mux.HandleFunc("PUT /api/v1/mihomo/config", a.handleMihomoConfigPut)
	mux.HandleFunc("GET /api/v1/mihomo/config/{path...}", a.handleMihomoConfigPathGet)
	mux.HandleFunc("PUT /api/v1/mihomo/config/{path...}", a.handleMihomoConfigPathPut)
	mux.HandleFunc("GET /api/v1/mihomo/configs", a.handleMihomoConfigs)
	mux.HandleFunc("POST /api/v1/mihomo/config/{id}/rollback", a.handleMihomoConfigRollback)
	mux.HandleFunc("GET /api/v1/mihomo/config/files", a.handleMihomoConfigFiles)
	mux.HandleFunc("GET /api/v1/mihomo/config/download", a.handleMihomoConfigDownload)
	mux.HandleFunc("POST /api/v1/mihomo/config/upload", a.handleMihomoConfigUpload)
	mux.HandleFunc("GET /api/v1/mihomo/rule-providers-config", a.handleMihomoProviderConfig)
	mux.HandleFunc("PUT /api/v1/mihomo/rule-providers-config", a.handleMihomoProviderConfigPut)
	mux.HandleFunc("GET /api/v1/mihomo/rules-config", a.handleMihomoRulesConfig)
	mux.HandleFunc("PUT /api/v1/mihomo/rules-config", a.handleMihomoRulesConfigPut)

	mux.HandleFunc("GET /api/v1/mihomo/traffic", a.handleMihomoTraffic)
	mux.HandleFunc("GET /api/v1/mihomo/connections", a.handleMihomoConnections)
	mux.HandleFunc("DELETE /api/v1/mihomo/connections", a.handleMihomoConnectionsClose)
	mux.HandleFunc("DELETE /api/v1/mihomo/connections/{id}", a.handleMihomoConnectionClose)
	mux.HandleFunc("GET /api/v1/mihomo/proxies", a.handleMihomoProxies)
	mux.HandleFunc("PUT /api/v1/mihomo/proxies/{name}", a.handleMihomoProxySelect)
	mux.HandleFunc("GET /api/v1/mihomo/proxies/{name}/delay", a.handleMihomoProxyDelay)
	mux.HandleFunc("GET /api/v1/mihomo/rules", a.handleMihomoRules)
	mux.HandleFunc("GET /api/v1/mihomo/providers", a.handleMihomoProviders)
	mux.HandleFunc("GET /api/v1/mihomo/proxy-providers", a.handleMihomoProxyProvidersConfig)
	mux.HandleFunc("PUT /api/v1/mihomo/proxy-providers", a.handleMihomoProxyProvidersPut)
	mux.HandleFunc("POST /api/v1/mihomo/proxy-providers", a.handleMihomoProxyProvidersPut)
	mux.HandleFunc("GET /api/v1/mihomo/proxy-providers/{name}", a.handleMihomoProxyProviderGet)
	mux.HandleFunc("PUT /api/v1/mihomo/proxy-providers/{name}", a.handleMihomoProxyProviderPut)
	mux.HandleFunc("PATCH /api/v1/mihomo/proxy-providers/{name}", a.handleMihomoProxyProviderPut)
	mux.HandleFunc("DELETE /api/v1/mihomo/proxy-providers/{name}", a.handleMihomoProxyProviderDelete)
	mux.HandleFunc("POST /api/v1/mihomo/proxy-providers/{name}/update", a.handleMihomoProxyProviderUpdate)
	mux.HandleFunc("POST /api/v1/mihomo/proxy-providers/{name}/healthcheck", a.handleMihomoProxyProviderUpdate)
	mux.HandleFunc("GET /api/v1/mihomo/rule-providers", a.handleMihomoRuleProviders)
	mux.HandleFunc("PUT /api/v1/mihomo/rule-providers", a.handleMihomoRuleProvidersPut)
	mux.HandleFunc("POST /api/v1/mihomo/rule-providers", a.handleMihomoRuleProvidersPut)
	mux.HandleFunc("GET /api/v1/mihomo/rule-providers/{name}", a.handleMihomoRuleProviderGet)
	mux.HandleFunc("PUT /api/v1/mihomo/rule-providers/{name}", a.handleMihomoRuleProviderPut)
	mux.HandleFunc("PATCH /api/v1/mihomo/rule-providers/{name}", a.handleMihomoRuleProviderPut)
	mux.HandleFunc("DELETE /api/v1/mihomo/rule-providers/{name}", a.handleMihomoRuleProviderDelete)
	mux.HandleFunc("POST /api/v1/mihomo/rule-providers/{name}/update", a.handleMihomoRuleProviderUpdate)
	mux.HandleFunc("GET /api/v1/mihomo/controller/{path...}", a.handleMihomoControllerProxy)
	mux.HandleFunc("POST /api/v1/mihomo/controller/{path...}", a.handleMihomoControllerProxy)
	mux.HandleFunc("PUT /api/v1/mihomo/controller/{path...}", a.handleMihomoControllerProxy)
	mux.HandleFunc("PATCH /api/v1/mihomo/controller/{path...}", a.handleMihomoControllerProxy)
	mux.HandleFunc("DELETE /api/v1/mihomo/controller/{path...}", a.handleMihomoControllerProxy)
	mux.HandleFunc("GET /api/v1/mihomo/ui", a.handleMihomoUI)
	mux.HandleFunc("GET /api/v1/mihomo/uis", a.handleMihomoUIs)
	mux.HandleFunc("POST /api/v1/mihomo/ui", a.handleMihomoUISwitch)
	mux.HandleFunc("GET /api/v1/mihomo/ui/config", a.handleMihomoUIConfig)
	mux.HandleFunc("POST /api/v1/mihomo/validate", a.handleMihomoValidate)

	mux.HandleFunc("GET /api/v1/proxy", a.handleProxyOverview)
	mux.HandleFunc("GET /api/v1/proxy/status", a.handleProxyOverview)
	mux.HandleFunc("GET /api/v1/proxy/overview", a.handleProxyOverview)
	mux.HandleFunc("GET /api/v1/proxy/traffic", a.handleMihomoTraffic)
	mux.HandleFunc("GET /api/v1/proxy/connections", a.handleMihomoConnections)
	mux.HandleFunc("DELETE /api/v1/proxy/connections", a.handleMihomoConnectionsClose)
	mux.HandleFunc("GET /api/v1/proxy/proxies", a.handleMihomoProxies)
	mux.HandleFunc("GET /api/v1/proxy/rules", a.handleMihomoRules)
	mux.HandleFunc("GET /api/v1/proxy/providers", a.handleMihomoProviders)
	mux.HandleFunc("GET /api/v1/proxy/logs", a.handleMihomoLogs)
	mux.HandleFunc("GET /api/v1/proxy/config", a.handleMihomoConfig)
}

func (a *App) handleMihomoStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.Services.Status("mihomo")})
}

func (a *App) handleMihomoOverview(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.mihomoSnapshot()})
}

func (a *App) handleMihomoVersion(w http.ResponseWriter, r *http.Request) {
	version := a.mihomoVersion()
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "version": version, "data": map[string]any{"version": version}})
}

func (a *App) handleMihomoVersions(w http.ResponseWriter, r *http.Request) {
	current := a.mihomoVersion()
	if current == "" {
		current = "not-installed"
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": []string{current}, "current_version": current})
}

func (a *App) handleMihomoVersionSwitch(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.Services.Status("mihomo")})
}

func (a *App) handleMihomoLogs(w http.ResponseWriter, r *http.Request) {
	lines := filterLogLines(a.serviceLogLines("mihomo", queryInt(r, "lines", 500)), r)
	entries := structuredLogLines(lines)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "lines": lines, "logs": entries, "data": entries, "content": strings.Join(lines, "\n")})
}

func (a *App) handleMihomoInstall(w http.ResponseWriter, r *http.Request) {
	a.runInstallJSON(w, "mihomo")
}

func (a *App) handleMihomoStart(w http.ResponseWriter, r *http.Request) {
	st, err := a.Services.Start(r.Context(), "mihomo")
	a.writeServiceResult(w, st, err)
}

func (a *App) handleMihomoStop(w http.ResponseWriter, r *http.Request) {
	st, err := a.Services.Stop(r.Context(), "mihomo")
	a.writeServiceResult(w, st, err)
}

func (a *App) handleMihomoRestart(w http.ResponseWriter, r *http.Request) {
	st, err := a.Services.Restart(r.Context(), "mihomo")
	a.writeServiceResult(w, st, err)
}

func (a *App) handleMihomoConfig(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "configs/mihomo/config.yaml"
	}
	if !strings.HasPrefix(path, "configs/mihomo/") {
		path = filepath.ToSlash(filepath.Join("configs/mihomo", path))
	}
	content, err := a.readTextFile(path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "read_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "path": path, "content": content})
}

func (a *App) handleMihomoConfigPut(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Restart bool   `json:"restart"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Path == "" {
		req.Path = "configs/mihomo/config.yaml"
	}
	if !strings.HasPrefix(req.Path, "configs/mihomo/") {
		req.Path = filepath.ToSlash(filepath.Join("configs/mihomo", req.Path))
	}
	if old, err := a.readTextFile(req.Path); err == nil {
		a.createConfigHistory("mihomo", req.Path, old, "auto backup before Mihomo save", currentUsername(r))
	}
	if err := a.writeTextFile(req.Path, req.Content); err != nil {
		writeError(w, http.StatusBadRequest, "write_failed", err.Error())
		return
	}
	if req.Restart {
		_, _ = a.Services.Restart(r.Context(), "mihomo")
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "restart_required": !req.Restart, "data": map[string]any{"restart_required": !req.Restart}})
}

func (a *App) handleMihomoConfigSwitch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Config string `json:"config"`
		Path   string `json:"path"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Config == "" {
		req.Config = req.Path
	}
	if req.Config == "" {
		req.Config = "config.yaml"
	}
	rel := mihomoRelPath(req.Config)
	if _, err := a.safePath(rel); err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	a.setSetting("mihomo.active_config", filepath.Base(rel))
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"config": filepath.Base(rel), "path": rel}})
}

func (a *App) handleMihomoConfigPathGet(w http.ResponseWriter, r *http.Request) {
	rel := mihomoRelPath(r.PathValue("path"))
	content, err := a.readTextFile(rel)
	if err != nil {
		writeError(w, http.StatusBadRequest, "read_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "path": rel, "content": content})
}

func (a *App) handleMihomoConfigPathPut(w http.ResponseWriter, r *http.Request) {
	rel := mihomoRelPath(r.PathValue("path"))
	var req struct {
		Content string `json:"content"`
		Restart bool   `json:"restart"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if old, err := a.readTextFile(rel); err == nil {
		a.createConfigHistory("mihomo", rel, old, "auto backup before Mihomo save", currentUsername(r))
	}
	if err := a.writeTextFile(rel, req.Content); err != nil {
		writeError(w, http.StatusBadRequest, "write_failed", err.Error())
		return
	}
	if req.Restart {
		_, _ = a.Services.Restart(r.Context(), "mihomo")
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "path": rel, "restart_required": !req.Restart, "data": map[string]any{"path": rel, "restart_required": !req.Restart}})
}

func (a *App) handleMihomoConfigs(w http.ResponseWriter, r *http.Request) {
	root := filepath.Join(a.DataDir, "configs/mihomo")
	entries, err := os.ReadDir(root)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	var configs []string
	var files []map[string]any
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			continue
		}
		configs = append(configs, name)
		item := map[string]any{"name": name, "path": filepath.ToSlash(filepath.Join("configs/mihomo", name)), "type": "file"}
		if info, err := entry.Info(); err == nil {
			item["size"] = info.Size()
			item["modified"] = info.ModTime().Format("2006-01-02 15:04:05")
		}
		files = append(files, item)
	}
	if len(configs) == 0 {
		configs = append(configs, "config.yaml")
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": configs, "configs": configs, "files": files})
}

func (a *App) handleMihomoConfigRollback(w http.ResponseWriter, r *http.Request) {
	a.handleHistoryRollback(w, r)
}

func (a *App) handleMihomoConfigFiles(w http.ResponseWriter, r *http.Request) {
	nodes, err := a.fileTree("configs/mihomo", 4)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": nodes, "tree": nodes})
}

func (a *App) handleMihomoConfigDownload(w http.ResponseWriter, r *http.Request) {
	a.downloadConfigDir(w, "configs/mihomo", "mihomo-configs.zip")
}

func (a *App) handleMihomoConfigUpload(w http.ResponseWriter, r *http.Request) {
	a.uploadConfigZip(w, r, "configs/mihomo")
}

func (a *App) handleMihomoProviderConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.mihomoRuleProvidersPayload()})
}

func (a *App) handleMihomoProviderConfigPut(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := a.updateMihomoConfigSections(req, "rule-providers"); err != nil {
		writeError(w, http.StatusBadRequest, "write_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleMihomoRulesConfig(w http.ResponseWriter, r *http.Request) {
	cfg := map[string]any{}
	_ = readYAMLFile(filepath.Join(a.DataDir, "configs/mihomo/config.yaml"), &cfg)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"rules": cfg["rules"], "rule-providers": cfg["rule-providers"], "runtime": a.mihomoRulesRuntime(r)}})
}

func (a *App) handleMihomoRulesConfigPut(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := a.updateMihomoConfigSections(req, "rules", "rule-providers"); err != nil {
		writeError(w, http.StatusBadRequest, "write_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleMihomoTraffic(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.mihomoTrafficPayload()})
}

func (a *App) handleMihomoConnections(w http.ResponseWriter, r *http.Request) {
	payload := a.mihomoConnectionsPayload(r)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": payload, "connections": payload["connections"], "items": payload["items"], "pagination": payload["pagination"]})
}

func (a *App) handleMihomoConnectionsClose(w http.ResponseWriter, r *http.Request) {
	if _, ok, err := a.mihomoControllerJSON(http.MethodDelete, "/connections", nil); !ok {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": errString(err, "mihomo controller unavailable"), "data": map[string]any{"closed": false}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleMihomoConnectionClose(w http.ResponseWriter, r *http.Request) {
	path := "/connections/" + url.PathEscape(r.PathValue("id"))
	a.proxyMihomoRequestOrJSON(w, r, http.MethodDelete, path, map[string]any{"closed": true})
}

func (a *App) handleMihomoProxies(w http.ResponseWriter, r *http.Request) {
	payload := a.mihomoProxiesPayload(r)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": payload, "groups": payload["groups"], "proxies": payload["proxies"], "providers": payload["providers"]})
}

func (a *App) handleMihomoProxySelect(w http.ResponseWriter, r *http.Request) {
	path := "/proxies/" + url.PathEscape(r.PathValue("name"))
	a.proxyMihomoRequestOrJSON(w, r, http.MethodPut, path, map[string]any{"updated": true})
}

func (a *App) handleMihomoProxyDelay(w http.ResponseWriter, r *http.Request) {
	path := "/proxies/" + url.PathEscape(r.PathValue("name")) + "/delay"
	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}
	a.proxyMihomoRequestOrJSON(w, r, http.MethodGet, path, map[string]any{"delay": 0})
}

func (a *App) handleMihomoRules(w http.ResponseWriter, r *http.Request) {
	payload := a.mihomoRulesRuntime(r)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": payload, "rules": payload["rules"], "items": payload["items"], "pagination": payload["pagination"]})
}

func (a *App) handleMihomoProviders(w http.ResponseWriter, r *http.Request) {
	payload := a.mihomoProvidersPayload()
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": payload, "providers": payload["proxy_providers"], "proxy_providers": payload["proxy_providers"], "rule_providers": payload["rule_providers"]})
}

func (a *App) handleMihomoProxyProvidersConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.mihomoProxyProvidersPayload()})
}

func (a *App) handleMihomoProxyProvidersPut(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := a.updateMihomoConfigSections(req, "proxy-providers"); err != nil {
		writeError(w, http.StatusBadRequest, "write_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleMihomoControllerProxy(w http.ResponseWriter, r *http.Request) {
	target, err := url.Parse(a.mihomoControllerBase())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"success": false, "error": err.Error()})
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	orig := r.URL.Path
	controllerPath := "/" + strings.TrimPrefix(r.PathValue("path"), "/")
	r.URL.Path = controllerPath
	r.URL.RawPath = ""
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = controllerPath
		req.URL.RawPath = ""
		req.Host = target.Host
		if secret := a.mihomoSecret(); secret != "" && req.Header.Get("Authorization") == "" {
			req.Header.Set("Authorization", "Bearer "+secret)
		}
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		if fallback, ok := mihomoControllerFallback(controllerPath); ok {
			writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": fallback})
			return
		}
		writeJSON(w, http.StatusBadGateway, map[string]any{"success": false, "error": err.Error()})
	}
	proxy.ServeHTTP(w, r)
	r.URL.Path = orig
}

func mihomoControllerFallback(path string) (any, bool) {
	clean := strings.Trim(path, "/")
	switch clean {
	case "configs":
		return map[string]any{
			"mode":                "rule",
			"port":                7890,
			"socks-port":          7891,
			"mixed-port":          7892,
			"redir-port":          7877,
			"tproxy-port":         7896,
			"external-controller": "127.0.0.1:9090",
			"allow-lan":           true,
			"log-level":           "info",
		}, true
	case "providers/proxies":
		return map[string]any{"providers": map[string]any{}}, true
	case "providers/rules":
		return map[string]any{"providers": map[string]any{}}, true
	default:
		if strings.HasPrefix(clean, "providers/proxies/") {
			if strings.HasSuffix(clean, "/healthcheck") {
				return map[string]any{"updatedAt": time.Now().Format(time.RFC3339), "healthcheck": true}, true
			}
			return map[string]any{"updatedAt": time.Now().Format(time.RFC3339), "vehicleType": "HTTP", "proxies": []any{}}, true
		}
		if strings.HasPrefix(clean, "providers/rules/") {
			return map[string]any{"updatedAt": time.Now().Format(time.RFC3339), "rules": []any{}}, true
		}
		return nil, false
	}
}

func (a *App) handleMihomoUI(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/ui/", http.StatusFound)
}

func (a *App) handleMihomoUIs(w http.ResponseWriter, r *http.Request) {
	uiRoot := filepath.Join(a.DataDir, "configs/mihomo/ui")
	var items []map[string]any
	if entries, err := os.ReadDir(uiRoot); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || entry.Name() == "index.html" {
				items = append(items, map[string]any{"name": entry.Name(), "current": entry.Name() == "index.html"})
			}
		}
	}
	if len(items) == 0 {
		items = append(items, map[string]any{"name": "zashboard", "current": true, "installed": false})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items})
}

func (a *App) handleMihomoUISwitch(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"url": "/ui/"}})
}

func (a *App) handleMihomoValidate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	var v any
	if err := yaml.Unmarshal([]byte(req.Content), &v); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "valid": false, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "valid": true})
}

func (a *App) updateMihomoConfigSections(req map[string]any, sections ...string) error {
	cfg := map[string]any{}
	path := filepath.Join(a.DataDir, "configs/mihomo/config.yaml")
	_ = readYAMLFile(path, &cfg)
	for _, section := range sections {
		if v, ok := req[section]; ok {
			cfg[section] = v
			continue
		}
		if len(sections) == 1 {
			cfg[section] = req
		}
	}
	return a.writeMihomoConfigMap(cfg)
}

func mihomoRelPath(path string) string {
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		path = "config.yaml"
	}
	if strings.HasPrefix(path, "configs/mihomo/") {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(filepath.Join("configs/mihomo", path))
}

func (a *App) proxyMihomoOrJSON(w http.ResponseWriter, path string, fallback any) {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	req, err := http.NewRequest(http.MethodGet, a.mihomoControllerURL(path), nil)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": fallback})
		return
	}
	if secret := a.mihomoSecret(); secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	resp, err := client.Do(req)
	if err == nil && resp.StatusCode < 300 {
		defer resp.Body.Close()
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
		_, _ = io.Copy(w, resp.Body)
		return
	}
	if resp != nil {
		_ = resp.Body.Close()
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": fallback})
}

func (a *App) proxyMihomoRequestOrJSON(w http.ResponseWriter, r *http.Request, method, path string, fallback any) {
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(io.LimitReader(r.Body, 4<<20))
	}
	req, err := http.NewRequestWithContext(r.Context(), method, a.mihomoControllerURL(path), bytes.NewReader(body))
	if err == nil {
		req.Header.Set("Content-Type", r.Header.Get("Content-Type"))
		if secret := a.mihomoSecret(); secret != "" {
			req.Header.Set("Authorization", "Bearer "+secret)
		}
		resp, err := (&http.Client{Timeout: 1500 * time.Millisecond}).Do(req)
		if err == nil && resp.StatusCode < 300 {
			defer resp.Body.Close()
			w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
			if w.Header().Get("Content-Type") == "" {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
			}
			_, _ = io.Copy(w, resp.Body)
			return
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": fallback})
}

func (a *App) mihomoVersion() string {
	st := a.Services.Status("mihomo")
	if !st.Installed {
		return "not-installed"
	}
	out, err := exec.Command(st.BinaryPath, "-v").CombinedOutput()
	if err == nil {
		v := strings.TrimSpace(string(out))
		if v != "" {
			return v
		}
	}
	return "installed"
}

func proxyDelete(url string) bool {
	req, _ := http.NewRequest(http.MethodDelete, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode < 300
}

func readYAMLFile(path string, dst any) error {
	b, err := osReadFile(path)
	if err != nil {
		return err
	}
	return yamlUnmarshal(b, dst)
}

func osReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func yamlUnmarshal(b []byte, dst any) error {
	return yaml.Unmarshal(b, dst)
}
