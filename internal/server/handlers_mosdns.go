package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (a *App) registerMosDNSRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/mosdns/status", a.handleMosDNSStatus)
	mux.HandleFunc("GET /api/v1/mosdns/overview", a.handleMosDNSOverview)
	mux.HandleFunc("GET /api/v1/mosdns/stats", a.handleMosDNSStats)
	mux.HandleFunc("GET /api/v1/mosdns/metrics", a.handleMosDNSMetrics)
	mux.HandleFunc("GET /api/v1/mosdns/version", a.handleMosDNSVersion)
	mux.HandleFunc("GET /api/v1/mosdns/versions", a.handleMosDNSVersions)
	mux.HandleFunc("POST /api/v1/mosdns/version", a.handleMosDNSVersionSwitch)
	mux.HandleFunc("GET /api/v1/mosdns/logs", a.handleMosDNSLogs)
	mux.HandleFunc("POST /api/v1/mosdns/install", a.handleMosDNSInstall)
	mux.HandleFunc("POST /api/v1/mosdns/start", a.handleMosDNSStart)
	mux.HandleFunc("POST /api/v1/mosdns/stop", a.handleMosDNSStop)
	mux.HandleFunc("POST /api/v1/mosdns/restart", a.handleMosDNSRestart)
	mux.HandleFunc("POST /api/v1/mosdns/cache/clear", a.handleMosDNSCacheClear)
	mux.HandleFunc("POST /api/v1/mosdns/clear-cache", a.handleMosDNSCacheClear)

	mux.HandleFunc("GET /api/v1/mosdns/clients", a.handleMosDNSClients)
	mux.HandleFunc("POST /api/v1/mosdns/clients", a.handleMosDNSClientCreate)
	mux.HandleFunc("PATCH /api/v1/mosdns/clients/{id}", a.handleMosDNSClientPatch)
	mux.HandleFunc("DELETE /api/v1/mosdns/clients/{id}", a.handleMosDNSClientDelete)
	mux.HandleFunc("POST /api/v1/mosdns/clients/scan", a.handleMosDNSClientScan)
	mux.HandleFunc("POST /api/v1/mosdns/clients/scan/reset", a.handleMosDNSClientScanReset)
	mux.HandleFunc("GET /api/v1/mosdns/clients/scan/{id}", a.handleMosDNSClientScanTask)
	mux.HandleFunc("POST /api/v1/mosdns/scan", a.handleMosDNSClientScan)
	mux.HandleFunc("GET /api/v1/mosdns/client-ips", a.handleMosDNSClientIPs)
	mux.HandleFunc("POST /api/v1/mosdns/client-ips", a.handleMosDNSClientIPCreate)
	mux.HandleFunc("DELETE /api/v1/mosdns/client-ips/{id}", a.handleMosDNSClientIPDelete)
	mux.HandleFunc("GET /api/v1/mosdns/client-proxy-mode", a.handleMosDNSClientProxyMode)
	mux.HandleFunc("POST /api/v1/mosdns/client-proxy-mode", a.handleMosDNSClientProxyModePut)

	mux.HandleFunc("GET /api/v1/mosdns/rules", a.handleMosDNSRules)
	mux.HandleFunc("GET /api/v1/mosdns/rules/{path...}", a.handleMosDNSRuleGet)
	mux.HandleFunc("PUT /api/v1/mosdns/rules/{path...}", a.handleMosDNSRulePut)
	mux.HandleFunc("POST /api/v1/mosdns/rules/{path...}", a.handleMosDNSRulePut)
	mux.HandleFunc("DELETE /api/v1/mosdns/rules/{path...}", a.handleMosDNSRuleDelete)
	mux.HandleFunc("GET /api/v1/mosdns/switches", a.handleMosDNSSwitches)
	mux.HandleFunc("PUT /api/v1/mosdns/switches", a.handleMosDNSSwitchesPut)
	mux.HandleFunc("GET /api/v1/mosdns/feature-switches", a.handleMosDNSSwitches)
	mux.HandleFunc("PUT /api/v1/mosdns/feature-switches", a.handleMosDNSSwitchesPut)

	mux.HandleFunc("GET /api/v1/mosdns/query-log", a.handleMosDNSQueryLog)
	mux.HandleFunc("GET /api/v1/mosdns/query-logs", a.handleMosDNSQueryLog)
	mux.HandleFunc("GET /api/v1/mosdns/query-meta", a.handleMosDNSQueryMeta)
	mux.HandleFunc("GET /api/v1/mosdns/rule-sets", a.handleMosDNSRuleSets)
	mux.HandleFunc("GET /api/v1/mosdns/audit", a.handleMosDNSAudit)
	mux.HandleFunc("GET /api/v1/mosdns/audit/rank", a.handleMosDNSAuditRank)
	mux.HandleFunc("GET /api/v1/mosdns/audit/ranks", a.handleMosDNSAuditRank)
	mux.HandleFunc("GET /api/v1/mosdns/audit/stats", a.handleMosDNSAuditStats)
	mux.HandleFunc("GET /api/v1/mosdns/cache/detailed", a.handleMosDNSCacheDetailed)
	mux.HandleFunc("GET /api/v1/mosdns/upstream/stats", a.handleMosDNSUpstreamStats)
	mux.HandleFunc("GET /api/v1/mosdns/routing/task", a.handleMosDNSRoutingTask)
	mux.HandleFunc("GET /api/v1/mosdns/upstreams", a.handleMosDNSUpstreams)
	mux.HandleFunc("PUT /api/v1/mosdns/upstreams", a.handleMosDNSUpstreamsPut)
	mux.HandleFunc("GET /api/v1/mosdns/forward-settings", a.handleMosDNSUpstreams)
	mux.HandleFunc("PUT /api/v1/mosdns/forward-settings", a.handleMosDNSUpstreamsPut)
	mux.HandleFunc("GET /api/v1/mosdns/config/file", a.handleMosDNSConfigFile)
	mux.HandleFunc("PUT /api/v1/mosdns/config/file", a.handleMosDNSConfigFilePut)
	mux.HandleFunc("GET /api/v1/mosdns/config/files", a.handleMosDNSConfigFiles)

	mux.HandleFunc("GET /api/v1/mosdns/system/cache", a.handleMosDNSSystemCache)
	mux.HandleFunc("POST /api/v1/mosdns/system/cache/clear", a.handleMosDNSCacheClear)
	mux.HandleFunc("GET /api/v1/mosdns/system/client-ip-list", a.handleMosDNSClientIPListGet)
	mux.HandleFunc("POST /api/v1/mosdns/system/client-ip-list", a.handleMosDNSClientIPListPut)
	mux.HandleFunc("GET /api/v1/mosdns/system/domains/{name...}", a.handleMosDNSSystemDomains)
	mux.HandleFunc("GET /api/v1/mosdns/system/feature-switches", a.handleMosDNSSystemFeatureSwitches)
	mux.HandleFunc("POST /api/v1/mosdns/system/feature-switches", a.handleMosDNSSystemFeatureSwitchesPut)
	mux.HandleFunc("GET /api/v1/mosdns/system/forward-settings", a.handleMosDNSUpstreams)
	mux.HandleFunc("POST /api/v1/mosdns/system/forward-settings", a.handleMosDNSUpstreamsPut)
	mux.HandleFunc("GET /api/v1/mosdns/system/log-capacity", a.handleMosDNSLogCapacity)
	mux.HandleFunc("POST /api/v1/mosdns/system/log-capacity", a.handleMosDNSLogCapacityPut)
	mux.HandleFunc("GET /api/v1/mosdns/system/overrides", a.handleMosDNSOverrides)
	mux.HandleFunc("POST /api/v1/mosdns/system/overrides", a.handleMosDNSOverridesPut)
	mux.HandleFunc("GET /api/v1/mosdns/system/routing", a.handleMosDNSRoutingTask)
	mux.HandleFunc("GET /api/v1/mosdns/system/routing/status", a.handleMosDNSRoutingTask)
	mux.HandleFunc("POST /api/v1/mosdns/system/routing/start", a.handleMosDNSRoutingStart)
	mux.HandleFunc("POST /api/v1/mosdns/system/routing/save", a.handleMosDNSRoutingSave)
	mux.HandleFunc("POST /api/v1/mosdns/system/routing/clear", a.handleMosDNSRoutingClear)
	mux.HandleFunc("GET /api/v1/mosdns/system/routing/scheduler", a.handleMosDNSRoutingScheduler)
	mux.HandleFunc("POST /api/v1/mosdns/system/routing/scheduler", a.handleMosDNSRoutingSchedulerPut)
	mux.HandleFunc("GET /api/v1/mosdns/system/switches", a.handleMosDNSSystemSwitches)
	mux.HandleFunc("POST /api/v1/mosdns/system/switches", a.handleMosDNSSwitchesPutCompat)
	mux.HandleFunc("GET /api/v1/mosdns/system/upstream-overrides", a.handleMosDNSUpstreamOverrides)
	mux.HandleFunc("POST /api/v1/mosdns/system/upstream-overrides", a.handleMosDNSUpstreamOverridesPut)
}

func (a *App) handleMosDNSStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.Services.Status("mosdns")})
}

func (a *App) handleMosDNSOverview(w http.ResponseWriter, r *http.Request) {
	st := a.Services.Status("mosdns")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"service":       st,
		"running":       st.Running,
		"clients":       a.countTable("mosdns_clients"),
		"client_ips":    a.countTable("mosdns_client_ips"),
		"switches":      a.mosDNSSwitchMap(),
		"api_endpoint":  "http://127.0.0.1:9099",
		"dns_listen":    ":53",
		"query_count":   0,
		"cache_entries": 0,
	}})
}

func (a *App) handleMosDNSStats(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{}
	if ok := proxyJSON("http://127.0.0.1:9099/metrics", &data); ok {
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"running":     a.Services.Status("mosdns").Running,
		"query_count": 0,
		"uptime":      0,
	}})
}

func (a *App) handleMosDNSMetrics(w http.ResponseWriter, r *http.Request) {
	if text, ok := proxyText("http://127.0.0.1:9099/metrics"); ok {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(text))
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("# mosdns metrics unavailable\n"))
}

func (a *App) handleMosDNSVersion(w http.ResponseWriter, r *http.Request) {
	version := "unknown"
	if st := a.Services.Status("mosdns"); st.Installed {
		if out, err := exec.Command(st.BinaryPath, "version").CombinedOutput(); err == nil {
			version = strings.TrimSpace(string(out))
		} else {
			version = "installed"
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "version": version, "data": map[string]any{"version": version}})
}

func (a *App) handleMosDNSVersions(w http.ResponseWriter, r *http.Request) {
	version := "not-installed"
	if st := a.Services.Status("mosdns"); st.Installed {
		version = "installed"
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": []string{version}, "current_version": version})
}

func (a *App) handleMosDNSVersionSwitch(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.Services.Status("mosdns")})
}

func (a *App) handleMosDNSLogs(w http.ResponseWriter, r *http.Request) {
	lines := a.serviceLogLines("mosdns", queryInt(r, "lines", 500))
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "lines": lines, "content": strings.Join(lines, "\n")})
}

func (a *App) handleMosDNSInstall(w http.ResponseWriter, r *http.Request) {
	a.runInstallJSON(w, "mosdns")
}

func (a *App) handleMosDNSStart(w http.ResponseWriter, r *http.Request) {
	st, err := a.Services.Start(r.Context(), "mosdns")
	a.writeServiceResult(w, st, err)
}

func (a *App) handleMosDNSStop(w http.ResponseWriter, r *http.Request) {
	st, err := a.Services.Stop(r.Context(), "mosdns")
	a.writeServiceResult(w, st, err)
}

func (a *App) handleMosDNSRestart(w http.ResponseWriter, r *http.Request) {
	st, err := a.Services.Restart(r.Context(), "mosdns")
	a.writeServiceResult(w, st, err)
}

func (a *App) handleMosDNSCacheClear(w http.ResponseWriter, r *http.Request) {
	_ = httpPostNoBody("http://127.0.0.1:9099/cache/flush")
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleMosDNSClients(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(`select id,coalesce(mac,''),ip,coalesce(hostname,''),coalesce(vendor,''),coalesce(custom_name,''),coalesce(custom_desc,''),coalesce(source,''),coalesce(type,''),query_count,first_seen_at,last_seen_at,last_scan_at,coalesce(interface,''),is_online,created_at,updated_at
		from mosdns_clients order by is_online desc,last_seen_at desc,id desc limit 1000`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	defer rows.Close()
	var items []map[string]any
	for rows.Next() {
		var id, count int64
		var mac, ip, hostname, vendor, customName, customDesc, source, typ, iface string
		var first, last, scan, created, updated sql.NullTime
		var online bool
		_ = rows.Scan(&id, &mac, &ip, &hostname, &vendor, &customName, &customDesc, &source, &typ, &count, &first, &last, &scan, &iface, &online, &created, &updated)
		items = append(items, map[string]any{
			"id": id, "mac": mac, "ip": ip, "hostname": hostname, "vendor": vendor, "custom_name": customName, "custom_desc": customDesc,
			"source": source, "type": typ, "query_count": count, "interface": iface, "is_online": online,
			"first_seen_at": nullableTimeString(first), "last_seen_at": nullableTimeString(last), "last_scan_at": nullableTimeString(scan),
			"created_at": nullableTimeString(created), "updated_at": nullableTimeString(updated),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items})
}

func (a *App) handleMosDNSClientCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MAC        string `json:"mac"`
		IP         string `json:"ip"`
		Hostname   string `json:"hostname"`
		CustomName string `json:"custom_name"`
		CustomDesc string `json:"custom_desc"`
		Type       string `json:"type"`
		Interface  string `json:"interface"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if net.ParseIP(req.IP) == nil {
		writeError(w, http.StatusBadRequest, "bad_ip", "invalid ip")
		return
	}
	now := time.Now()
	_, err := a.DB.Exec(`insert into mosdns_clients(mac,ip,hostname,custom_name,custom_desc,source,type,first_seen_at,last_seen_at,last_scan_at,interface,is_online,created_at,updated_at)
		values(?,?,?,?,?,'manual',?,?,?,?,?,?,?,?)
		on conflict(mac,ip) do update set hostname=excluded.hostname,custom_name=excluded.custom_name,custom_desc=excluded.custom_desc,last_seen_at=excluded.last_seen_at,last_scan_at=excluded.last_scan_at,interface=excluded.interface,is_online=excluded.is_online,updated_at=excluded.updated_at`,
		req.MAC, req.IP, req.Hostname, req.CustomName, req.CustomDesc, req.Type, now, now, now, req.Interface, true, now, now)
	if err != nil {
		writeError(w, http.StatusBadRequest, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleMosDNSClientDelete(w http.ResponseWriter, r *http.Request) {
	_, err := a.DB.Exec(`delete from mosdns_clients where id=?`, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "delete_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleMosDNSClientPatch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Hostname   string `json:"hostname"`
		CustomName string `json:"custom_name"`
		CustomDesc string `json:"custom_desc"`
		Type       string `json:"type"`
		Interface  string `json:"interface"`
		IsOnline   *bool  `json:"is_online"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	online := true
	if req.IsOnline != nil {
		online = *req.IsOnline
	}
	res, err := a.DB.Exec(`update mosdns_clients set hostname=coalesce(nullif(?,''),hostname),custom_name=coalesce(nullif(?,''),custom_name),custom_desc=coalesce(nullif(?,''),custom_desc),type=coalesce(nullif(?,''),type),interface=coalesce(nullif(?,''),interface),is_online=?,updated_at=? where id=? or ip=? or mac=?`,
		req.Hostname, req.CustomName, req.CustomDesc, req.Type, req.Interface, online, time.Now(), id, id, id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "update_failed", err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeError(w, http.StatusNotFound, "not_found", "client not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleMosDNSClientScan(w http.ResponseWriter, r *http.Request) {
	found := scanNeighbors()
	now := time.Now()
	for _, item := range found {
		_, _ = a.DB.Exec(`insert into mosdns_clients(mac,ip,hostname,source,type,first_seen_at,last_seen_at,last_scan_at,interface,is_online,created_at,updated_at)
			values(?,?,?,?,?,?,?,?,?,?,?,?)
			on conflict(mac,ip) do update set hostname=excluded.hostname,last_seen_at=excluded.last_seen_at,last_scan_at=excluded.last_scan_at,interface=excluded.interface,is_online=true,updated_at=excluded.updated_at`,
			item["mac"], item["ip"], item["hostname"], "scan", "lan", now, now, now, item["interface"], true, now, now)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "count": len(found), "data": found})
}

func (a *App) handleMosDNSClientScanReset(w http.ResponseWriter, r *http.Request) {
	_, _ = a.DB.Exec(`delete from mosdns_clients`)
	a.handleMosDNSClientScan(w, r)
}

func (a *App) handleMosDNSClientScanTask(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"id": r.PathValue("id"), "status": "completed", "progress": 100, "found": a.countTable("mosdns_clients"),
	}})
}

func (a *App) handleMosDNSClientIPs(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(`select id,ip,coalesce(comment,''),created_at,updated_at from mosdns_client_ips order by id desc`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	defer rows.Close()
	var items []map[string]any
	for rows.Next() {
		var id int64
		var ip, comment string
		var created, updated sql.NullTime
		_ = rows.Scan(&id, &ip, &comment, &created, &updated)
		items = append(items, map[string]any{"id": id, "ip": ip, "comment": comment, "created_at": nullableTimeString(created), "updated_at": nullableTimeString(updated)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items})
}

func (a *App) handleMosDNSClientIPCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IP      string `json:"ip"`
		Comment string `json:"comment"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if net.ParseIP(req.IP) == nil {
		writeError(w, http.StatusBadRequest, "bad_ip", "invalid ip")
		return
	}
	_, err := a.DB.Exec(`insert into mosdns_client_ips(ip,comment,created_at,updated_at) values(?,?,?,?) on conflict(ip) do update set comment=excluded.comment,updated_at=excluded.updated_at`, req.IP, req.Comment, time.Now(), time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "db_error", err.Error())
		return
	}
	_ = a.rewriteMosDNSClientIPFile()
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleMosDNSClientIPDelete(w http.ResponseWriter, r *http.Request) {
	_, err := a.DB.Exec(`delete from mosdns_client_ips where id=?`, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "delete_failed", err.Error())
		return
	}
	_ = a.rewriteMosDNSClientIPFile()
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleMosDNSClientProxyMode(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"mode": a.setting("mosdns_client_proxy_mode", "direct_default")}})
}

func (a *App) handleMosDNSClientProxyModePut(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode string `json:"mode"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Mode == "" {
		req.Mode = "direct_default"
	}
	a.setSetting("mosdns_client_proxy_mode", req.Mode)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"mode": req.Mode}})
}

func (a *App) handleMosDNSRules(w http.ResponseWriter, r *http.Request) {
	nodes, err := a.fileTree("configs/mosdns/rules", 3)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": nodes, "rules": nodes})
}

func (a *App) handleMosDNSRuleGet(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.PathValue("path"), "/"), "/")
	if path == "categories" {
		a.handleMosDNSRuleCategories(w, r)
		return
	}
	parts := strings.Split(path, "/")
	category := parts[0]
	if len(parts) > 1 && parts[1] == "export" {
		content, _ := a.readTextFile(mosDNSRuleCategoryFile(category))
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(content))
		return
	}
	items := a.readMosDNSRuleItems(category)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items, "rules": items})
}

func (a *App) handleMosDNSRulePut(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.PathValue("path"), "/"), "/")
	parts := strings.Split(path, "/")
	category := parts[0]
	var req struct {
		Pattern    string   `json:"pattern"`
		Patterns   []string `json:"patterns"`
		OldPattern string   `json:"old_pattern"`
		NewPattern string   `json:"new_pattern"`
		Content    string   `json:"content"`
	}
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
	} else {
		b, _ := io.ReadAll(io.LimitReader(r.Body, 16<<20))
		req.Content = string(b)
	}
	patterns := a.readMosDNSRulePatterns(category)
	if len(parts) > 1 && parts[1] == "import" {
		patterns = splitNonEmptyLines(req.Content)
	} else if len(parts) > 1 && parts[1] == "batch" {
		patterns = append(patterns, req.Patterns...)
	} else if req.OldPattern != "" {
		replaced := false
		for i, pattern := range patterns {
			if pattern == req.OldPattern {
				patterns[i] = req.NewPattern
				replaced = true
			}
		}
		if !replaced && req.NewPattern != "" {
			patterns = append(patterns, req.NewPattern)
		}
	} else if req.Content != "" && req.Pattern == "" {
		patterns = splitNonEmptyLines(req.Content)
	} else if req.Pattern != "" {
		patterns = append(patterns, req.Pattern)
	}
	if err := a.writeMosDNSRulePatterns(category, patterns); err != nil {
		writeError(w, http.StatusBadRequest, "write_failed", err.Error())
		return
	}
	items := a.readMosDNSRuleItems(category)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items, "rules": items})
}

func (a *App) handleMosDNSRuleDelete(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.PathValue("path"), "/"), "/")
	parts := strings.Split(path, "/")
	category := parts[0]
	if len(parts) > 1 && parts[1] == "all" {
		if err := a.writeMosDNSRulePatterns(category, nil); err != nil {
			writeError(w, http.StatusBadRequest, "delete_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": []any{}})
		return
	}
	var req struct {
		Pattern string `json:"pattern"`
	}
	_ = decodeJSON(r, &req)
	if req.Pattern != "" {
		var next []string
		for _, pattern := range a.readMosDNSRulePatterns(category) {
			if pattern != req.Pattern {
				next = append(next, pattern)
			}
		}
		if err := a.writeMosDNSRulePatterns(category, next); err != nil {
			writeError(w, http.StatusBadRequest, "delete_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.readMosDNSRuleItems(category)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.readMosDNSRuleItems(category)})
}

func (a *App) handleMosDNSSwitches(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.mosDNSSwitchMap()})
}

func (a *App) handleMosDNSSwitchesPut(w http.ResponseWriter, r *http.Request) {
	var req map[string]bool
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	now := time.Now()
	for k, enabled := range req {
		_, _ = a.DB.Exec(`insert into mosdns_switch_states(switch_key,enabled,created_at,updated_at) values(?,?,?,?) on conflict(switch_key) do update set enabled=excluded.enabled,updated_at=excluded.updated_at`, k, enabled, now, now)
	}
	_ = a.rewriteMosDNSSwitchFile()
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.mosDNSSwitchMap()})
}

func (a *App) handleMosDNSQueryLog(w http.ResponseWriter, r *http.Request) {
	lines := a.serviceLogLines("mosdns", queryInt(r, "lines", 300))
	entries := mosDNSQueryEntries(lines)
	page := queryInt(r, "page", 1)
	limit := queryInt(r, "limit", len(entries))
	if limit <= 0 {
		limit = 100
	}
	total := len(entries)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	payload := map[string]any{
		"logs":        entries[start:end],
		"items":       entries[start:end],
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": (total + limit - 1) / limit,
		"pagination": map[string]any{
			"page": page, "limit": limit, "page_size": limit, "total": total, "total_pages": (total + limit - 1) / limit,
		},
		"lines": lines,
	}
	if r.URL.Query().Get("stream") == "true" {
		a.sseLoop(w, r, 2*time.Second, func() any { return payload })
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": payload, "lines": lines, "logs": entries[start:end]})
}

func (a *App) handleMosDNSQueryMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"domains": []any{}, "clients": []any{}, "types": []string{"A", "AAAA", "HTTPS"}, "query_types": []string{"A", "AAAA", "HTTPS"},
		"rules": []any{}, "responses": []string{"NOERROR", "NXDOMAIN", "SERVFAIL"}, "response_codes": []string{"NOERROR", "NXDOMAIN", "SERVFAIL"},
	}})
}

func (a *App) handleMosDNSRuleSets(w http.ResponseWriter, r *http.Request) {
	nodes, _ := a.fileTree("configs/mosdns/rules", 2)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "rule_sets": nodes, "data": map[string]any{"rule_sets": nodes}})
}

func (a *App) handleMosDNSAudit(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": []any{}})
}

func (a *App) handleMosDNSAuditRank(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": []any{}, "ranks": []any{}})
}

func (a *App) handleMosDNSAuditStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"total_queries": 0, "blocked_queries": 0, "top_clients": []any{}, "top_domains": []any{},
	}})
}

func (a *App) handleMosDNSCacheDetailed(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"summary": map[string]any{"entries": 0, "hit_rate": 0}, "caches": []any{},
	}})
}

func (a *App) handleMosDNSUpstreamStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"upstreams": []any{}, "total": 0}})
}

func (a *App) handleMosDNSRoutingTask(w http.ResponseWriter, r *http.Request) {
	scheduler := a.mosDNSRoutingScheduler()
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"running": false, "enabled": false, "status": "idle", "progress": 0, "last_run_at": "", "rules": []any{}, "scheduler": scheduler,
		"execution_settings": map[string]any{
			"date_range_days":      7,
			"queries_per_second":   5,
			"resolver_address":     "127.0.0.1:53",
			"url_call_delay_ms":    100,
			"concurrency":          1,
			"include_empty_answer": false,
		},
	}})
}

func (a *App) handleMosDNSUpstreams(w http.ResponseWriter, r *http.Request) {
	local, _ := a.readTextFile("configs/mosdns/sub_config/forward_local.yaml")
	remote, _ := a.readTextFile("configs/mosdns/sub_config/forward_remote.yaml")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]string{"forward_local": local, "forward_remote": remote}})
}

func (a *App) handleMosDNSUpstreamsPut(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	for name, content := range req {
		switch name {
		case "forward_local", "local":
			_ = a.writeTextFile("configs/mosdns/sub_config/forward_local.yaml", content)
		case "forward_remote", "remote":
			_ = a.writeTextFile("configs/mosdns/sub_config/forward_remote.yaml", content)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleMosDNSConfigFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "configs/mosdns/config.yaml"
	}
	if !strings.HasPrefix(path, "configs/mosdns/") {
		path = filepath.ToSlash(filepath.Join("configs/mosdns", path))
	}
	content, err := a.readTextFile(path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "read_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "path": path, "content": content})
}

func (a *App) handleMosDNSConfigFilePut(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Path == "" {
		req.Path = "configs/mosdns/config.yaml"
	}
	if !strings.HasPrefix(req.Path, "configs/mosdns/") {
		req.Path = filepath.ToSlash(filepath.Join("configs/mosdns", req.Path))
	}
	if err := a.writeTextFile(req.Path, req.Content); err != nil {
		writeError(w, http.StatusBadRequest, "write_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleMosDNSConfigFiles(w http.ResponseWriter, r *http.Request) {
	nodes, err := a.fileTree("configs/mosdns", 4)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": nodes})
}

func (a *App) handleMosDNSSystemCache(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"entries": 0, "memory": 0, "hit_rate": 0, "caches": []any{},
	}})
}

func (a *App) handleMosDNSClientIPListGet(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(`select ip from mosdns_client_ips order by ip`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	defer rows.Close()
	var ips []string
	for rows.Next() {
		var ip string
		_ = rows.Scan(&ip)
		ips = append(ips, ip)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": ips, "ips": ips})
}

func (a *App) handleMosDNSClientIPListPut(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IPs []string `json:"ips"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	_, _ = a.DB.Exec(`delete from mosdns_client_ips`)
	now := time.Now()
	for _, ip := range req.IPs {
		if net.ParseIP(ip) != nil {
			_, _ = a.DB.Exec(`insert into mosdns_client_ips(ip,created_at,updated_at) values(?,?,?) on conflict(ip) do nothing`, ip, now, now)
		}
	}
	_ = a.rewriteMosDNSClientIPFile()
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleMosDNSSystemDomains(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.PathValue("name"), "/")
	content, _ := a.readTextFile(filepath.ToSlash(filepath.Join("configs/mosdns/rules", name)))
	lines := splitNonEmptyLines(content)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": lines, "domains": lines})
}

func (a *App) handleMosDNSSystemFeatureSwitches(w http.ResponseWriter, r *http.Request) {
	switches := a.mosDNSSwitchMap()
	keys := strings.Split(r.URL.Query().Get("keys"), ",")
	items := switchItems(switches, keys, "enable")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items, "switches": items})
}

func (a *App) handleMosDNSSystemFeatureSwitchesPut(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key    string `json:"key"`
		Enable bool   `json:"enable"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Key != "" {
		now := time.Now()
		_, _ = a.DB.Exec(`insert into mosdns_switch_states(switch_key,enabled,created_at,updated_at) values(?,?,?,?) on conflict(switch_key) do update set enabled=excluded.enabled,updated_at=excluded.updated_at`, req.Key, req.Enable, now, now)
		_ = a.rewriteMosDNSSwitchFile()
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.mosDNSSwitchMap()})
}

func (a *App) handleMosDNSLogCapacity(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"capacity": a.setting("mosdns_log_capacity", "5000")}})
}

func (a *App) handleMosDNSLogCapacityPut(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Capacity any `json:"capacity"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	value := "5000"
	if req.Capacity != nil {
		value = strings.TrimSpace(fmtAny(req.Capacity))
	}
	a.setSetting("mosdns_log_capacity", value)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"capacity": value}})
}

func (a *App) handleMosDNSOverrides(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.jsonSetting("mosdns_overrides", map[string]any{})})
}

func (a *App) handleMosDNSOverridesPut(w http.ResponseWriter, r *http.Request) {
	a.storeJSONBodySetting(w, r, "mosdns_overrides")
}

func (a *App) handleMosDNSRoutingStart(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"status": "started", "running": false}})
}

func (a *App) handleMosDNSRoutingSave(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleMosDNSRoutingClear(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleMosDNSRoutingScheduler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.mosDNSRoutingScheduler()})
}

func (a *App) handleMosDNSRoutingSchedulerPut(w http.ResponseWriter, r *http.Request) {
	a.storeJSONBodySetting(w, r, "mosdns_routing_scheduler")
}

func (a *App) handleMosDNSSystemSwitches(w http.ResponseWriter, r *http.Request) {
	keys := strings.Split(r.URL.Query().Get("keys"), ",")
	items := switchItems(a.mosDNSSwitchMap(), keys, "value")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items, "switches": items})
}

func (a *App) handleMosDNSSwitchesPutCompat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key    string `json:"key"`
		Value  bool   `json:"value"`
		Enable bool   `json:"enable"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Key != "" {
		now := time.Now()
		_, _ = a.DB.Exec(`insert into mosdns_switch_states(switch_key,enabled,created_at,updated_at) values(?,?,?,?) on conflict(switch_key) do update set enabled=excluded.enabled,updated_at=excluded.updated_at`, req.Key, req.Value || req.Enable, now, now)
	}
	_ = a.rewriteMosDNSSwitchFile()
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.mosDNSSwitchMap()})
}

func (a *App) handleMosDNSUpstreamOverrides(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.jsonSetting("mosdns_upstream_overrides", map[string]any{})})
}

func (a *App) handleMosDNSUpstreamOverridesPut(w http.ResponseWriter, r *http.Request) {
	a.storeJSONBodySetting(w, r, "mosdns_upstream_overrides")
}

func (a *App) handleMosDNSRuleCategories(w http.ResponseWriter, r *http.Request) {
	defs := []struct {
		ID   string
		Name string
	}{
		{"whitelist", "直连"},
		{"greylist", "代理"},
		{"blocklist", "拦截"},
		{"direct_ip", "直连 IP"},
		{"rewrite", "重写"},
		{"ddnslist", "DDNS"},
	}
	items := make([]map[string]any, 0, len(defs))
	for _, def := range defs {
		items = append(items, map[string]any{
			"id":    def.ID,
			"name":  def.Name,
			"count": len(a.readMosDNSRulePatterns(def.ID)),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items})
}

func mosDNSRuleCategoryFile(category string) string {
	category = strings.TrimSpace(category)
	switch category {
	case "blacklist":
		category = "blocklist"
	case "":
		category = "whitelist"
	}
	category = filepath.Base(category)
	return filepath.ToSlash(filepath.Join("configs/mosdns/rules", category+".txt"))
}

func (a *App) readMosDNSRulePatterns(category string) []string {
	content, _ := a.readTextFile(mosDNSRuleCategoryFile(category))
	return splitNonEmptyLines(content)
}

func (a *App) writeMosDNSRulePatterns(category string, patterns []string) error {
	seen := map[string]bool{}
	out := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" || seen[pattern] {
			continue
		}
		seen[pattern] = true
		out = append(out, pattern)
	}
	content := strings.Join(out, "\n")
	if content != "" {
		content += "\n"
	}
	return a.writeTextFile(mosDNSRuleCategoryFile(category), content)
}

func (a *App) readMosDNSRuleItems(category string) []map[string]any {
	patterns := a.readMosDNSRulePatterns(category)
	items := make([]map[string]any, 0, len(patterns))
	for i, pattern := range patterns {
		mode := ""
		value := pattern
		for _, prefix := range []string{"domain:", "full:", "keyword:", "regexp:"} {
			if strings.HasPrefix(pattern, prefix) {
				mode = strings.TrimSuffix(prefix, ":")
				value = strings.TrimPrefix(pattern, prefix)
				break
			}
		}
		items = append(items, map[string]any{
			"id":         fmt.Sprintf("%s-%d", category, i+1),
			"name":       value,
			"pattern":    pattern,
			"match_mode": mode,
			"category":   category,
			"enabled":    true,
			"order":      i + 1,
		})
	}
	return items
}

func mosDNSQueryEntries(lines []string) []map[string]any {
	entries := make([]map[string]any, 0, len(lines))
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		queryName := extractDomainLike(line)
		if queryName == "" {
			queryName = line
		}
		if len(queryName) > 180 {
			queryName = queryName[:180]
		}
		entries = append(entries, map[string]any{
			"trace_id":      fmt.Sprintf("log-%d", i+1),
			"query_time":    time.Now().Add(time.Duration(i-len(lines)) * time.Second).Format(time.RFC3339),
			"query_name":    queryName,
			"client_ip":     "127.0.0.1",
			"query_type":    "A",
			"domain_set":    "unmatched_rule",
			"response_code": "NOERROR",
			"duration_ms":   0.0,
			"answers":       []any{},
			"raw":           line,
		})
	}
	return entries
}

func extractDomainLike(line string) string {
	fields := strings.FieldsFunc(line, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '"' || r == '\'' || r == '[' || r == ']' || r == '(' || r == ')' || r == ',' || r == ';'
	})
	for _, field := range fields {
		field = strings.Trim(field, ".")
		if strings.Contains(field, ".") && !strings.Contains(field, "/") && net.ParseIP(field) == nil {
			return field
		}
	}
	return ""
}

func (a *App) mosDNSSwitchMap() map[string]bool {
	out := map[string]bool{}
	rows, err := a.DB.Query(`select switch_key,enabled from mosdns_switch_states order by switch_key`)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var enabled bool
		_ = rows.Scan(&key, &enabled)
		out[key] = enabled
	}
	return out
}

func (a *App) rewriteMosDNSSwitchFile() error {
	var b strings.Builder
	b.WriteString("plugins: []\n# switch states are managed by msm-free and mirrored in sqlite.\n")
	for k, v := range a.mosDNSSwitchMap() {
		b.WriteString("# ")
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(strconv.FormatBool(v))
		b.WriteString("\n")
	}
	return a.writeTextFile("configs/mosdns/sub_config/switch.yaml", b.String())
}

func (a *App) rewriteMosDNSClientIPFile() error {
	rows, err := a.DB.Query(`select ip from mosdns_client_ips order by ip`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var ip string
		_ = rows.Scan(&ip)
		lines = append(lines, ip)
	}
	return a.writeTextFile("configs/mosdns/client_ip.txt", strings.Join(lines, "\n")+"\n")
}

func scanNeighbors() []map[string]string {
	out, err := exec.Command("ip", "neigh").CombinedOutput()
	if err != nil {
		out, err = exec.Command("arp", "-an").CombinedOutput()
		if err != nil {
			return nil
		}
	}
	var items []map[string]string
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		item := map[string]string{"ip": strings.Trim(fields[0], "()"), "mac": "", "hostname": "", "interface": ""}
		for i, f := range fields {
			switch f {
			case "dev":
				if i+1 < len(fields) {
					item["interface"] = fields[i+1]
				}
			case "lladdr", "at":
				if i+1 < len(fields) {
					item["mac"] = fields[i+1]
				}
			}
		}
		if net.ParseIP(item["ip"]) != nil {
			items = append(items, item)
		}
	}
	return items
}

func nullableTimeString(v sql.NullTime) string {
	if !v.Valid {
		return ""
	}
	return v.Time.Format(time.RFC3339)
}

func queryInt(r *http.Request, key string, def int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || v <= 0 {
		return def
	}
	return v
}

func (a *App) countTable(table string) int64 {
	var n int64
	switch table {
	case "mosdns_clients":
		_ = a.DB.QueryRow(`select count(*) from mosdns_clients`).Scan(&n)
	case "mosdns_client_ips":
		_ = a.DB.QueryRow(`select count(*) from mosdns_client_ips`).Scan(&n)
	}
	return n
}

func (a *App) runInstallJSON(w http.ResponseWriter, component string) {
	var last DownloadEvent
	err := a.installComponent(component, func(ev DownloadEvent) { last = ev })
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "event": last})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "event": last})
}

func (a *App) writeServiceResult(w http.ResponseWriter, st ServiceStatus, err error) {
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": st})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": st})
}

func proxyJSON(url string, dst any) bool {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return false
	}
	return json.NewDecoder(resp.Body).Decode(dst) == nil
}

func proxyText(url string) (string, bool) {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Get(url)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", false
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", false
	}
	return string(b), true
}

func httpPostNoBody(url string) error {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Post(url, "application/json", nil)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}

func splitNonEmptyLines(content string) []string {
	var out []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			out = append(out, line)
		}
	}
	return out
}

func switchItems(values map[string]bool, keys []string, boolField string) []map[string]any {
	var items []map[string]any
	if len(keys) == 0 || (len(keys) == 1 && strings.TrimSpace(keys[0]) == "") {
		for key, value := range values {
			items = append(items, map[string]any{"key": key, "name": key, boolField: value, "value": value, "enable": value})
		}
		return items
	}
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		value := values[key]
		items = append(items, map[string]any{"key": key, "name": key, boolField: value, "value": value, "enable": value})
	}
	return items
}

func fmtAny(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatInt(int64(x), 10)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	default:
		return fmt.Sprint(v)
	}
}

func (a *App) setting(key, fallback string) string {
	var value string
	if err := a.DB.QueryRow(`select value from settings where key=?`, key).Scan(&value); err == nil {
		return value
	}
	return fallback
}

func (a *App) setSetting(key, value string) {
	_, _ = a.DB.Exec(`insert or replace into settings(key,value,updated_at) values(?,?,?)`, key, value, time.Now())
}

func (a *App) jsonSetting(key string, fallback any) any {
	value := a.setting(key, "")
	if value == "" {
		return fallback
	}
	var out any
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		return fallback
	}
	return out
}

func (a *App) mosDNSRoutingScheduler() map[string]any {
	fallback := map[string]any{"enabled": false, "interval": 86400, "interval_minutes": 1440, "start_datetime": time.Now().Format(time.RFC3339)}
	raw := a.jsonSetting("mosdns_routing_scheduler", fallback)
	scheduler, ok := raw.(map[string]any)
	if !ok {
		return fallback
	}
	for key, value := range fallback {
		if _, ok := scheduler[key]; !ok {
			scheduler[key] = value
		}
	}
	return scheduler
}

func (a *App) storeJSONBodySetting(w http.ResponseWriter, r *http.Request, key string) {
	var req any
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	b, err := json.Marshal(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	a.setSetting(key, string(b))
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": req})
}
