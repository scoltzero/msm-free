package server

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func (a *App) registerUpdateRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/update/status", a.handleUpdateStatus)
	mux.HandleFunc("POST /api/v1/update/check", a.handleUpdateCheck)
	mux.HandleFunc("GET /api/v1/update/config", a.handleUpdateConfig)
	mux.HandleFunc("PUT /api/v1/update/config", a.handleUpdateConfigPut)
	mux.HandleFunc("GET /api/v1/update/releases", a.handleUpdateReleases)
	mux.HandleFunc("POST /api/v1/update/download", a.handleUpdateNoop)
	mux.HandleFunc("POST /api/v1/update/install", a.handleUpdateNoop)
	mux.HandleFunc("POST /api/v1/update/cancel", a.handleUpdateNoop)

	mux.HandleFunc("GET /api/v1/component-updates", a.handleComponentUpdates)
	mux.HandleFunc("GET /api/v1/component-updates/{component}", a.handleComponentUpdateStatus)
	mux.HandleFunc("GET /api/v1/component-updates/{component}/status", a.handleComponentUpdateStatus)
	mux.HandleFunc("POST /api/v1/component-updates/{component}/check", a.handleComponentUpdateCheck)
	mux.HandleFunc("POST /api/v1/component-updates/{component}/update", a.handleComponentUpdateRun)
	mux.HandleFunc("GET /api/v1/component-updates/{component}/config", a.handleComponentUpdateConfig)
	mux.HandleFunc("PUT /api/v1/component-updates/{component}/config", a.handleComponentUpdateConfigPut)
}

func (a *App) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	item := map[string]any{
		"component":       "msm-free",
		"current_version": a.Version,
		"latest_version":  a.Version,
		"has_update":      false,
		"status":          "idle",
		"progress":        0,
		"release_notes":   "msm-free 自更新接口已预留，首版请使用发布包覆盖升级。",
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": item})
}

func (a *App) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"component":       "msm-free",
		"current_version": a.Version,
		"latest_version":  a.Version,
		"has_update":      false,
		"status":          "checked",
		"last_check_time": time.Now(),
	}})
}

func (a *App) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"auto_check":     true,
		"auto_update":    false,
		"check_interval": 86400,
	}})
}

func (a *App) handleUpdateConfigPut(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleUpdateReleases(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": []any{}})
}

func (a *App) handleUpdateNoop(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "message": "self update is not enabled in this build"})
}

func (a *App) handleComponentUpdates(w http.ResponseWriter, r *http.Request) {
	components := []string{"mosdns", "mihomo", "zashboard"}
	items := make([]map[string]any, 0, len(components))
	for _, c := range components {
		items = append(items, a.componentUpdateState(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items})
}

func (a *App) handleComponentUpdateStatus(w http.ResponseWriter, r *http.Request) {
	component := normalizeComponent(r.PathValue("component"))
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.componentUpdateState(component)})
}

func (a *App) handleComponentUpdateCheck(w http.ResponseWriter, r *http.Request) {
	component := normalizeComponent(r.PathValue("component"))
	state := a.componentUpdateState(component)
	state["last_check_time"] = time.Now()
	_, _ = a.DB.Exec(`insert into component_update_info(component,current_version,latest_version,has_update,download_url,status,progress,last_check_time,created_at,updated_at)
		values(?,?,?,?,?,?,?,?,?,?)
		on conflict(component) do update set current_version=excluded.current_version,latest_version=excluded.latest_version,has_update=excluded.has_update,download_url=excluded.download_url,status='checked',progress=0,last_check_time=excluded.last_check_time,updated_at=excluded.updated_at`,
		component, state["current_version"], state["latest_version"], state["has_update"], state["download_url"], "checked", 0, time.Now(), time.Now(), time.Now())
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": state})
}

func (a *App) handleComponentUpdateRun(w http.ResponseWriter, r *http.Request) {
	component := normalizeComponent(r.PathValue("component"))
	if component == "" {
		writeError(w, http.StatusBadRequest, "bad_component", "unknown component")
		return
	}
	now := time.Now()
	_, _ = a.DB.Exec(`insert into component_update_info(component,current_version,latest_version,has_update,download_url,status,progress,created_at,updated_at)
		values(?,?,?,?,?,?,?,?,?)
		on conflict(component) do update set status='running',progress=5,error_message='',updated_at=excluded.updated_at`,
		component, a.componentCurrentVersion(component), "latest", true, componentDownloadURL(component), "running", 5, now, now)
	last := DownloadEvent{Status: "running", Progress: 5, Message: "starting"}
	err := a.installComponent(component, func(ev DownloadEvent) {
		last = ev
		_, _ = a.DB.Exec(`update component_update_info set status=?, progress=?, error_message='', updated_at=? where component=?`, ev.Status, ev.Progress, nowString(), component)
	})
	if err != nil {
		_, _ = a.DB.Exec(`update component_update_info set status='failed', progress=?, error_message=?, updated_at=? where component=?`, last.Progress, err.Error(), nowString(), component)
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": a.componentUpdateState(component)})
		return
	}
	_, _ = a.DB.Exec(`update component_update_info set current_version='installed', latest_version='latest', has_update=false, status='completed', progress=100, error_message='', updated_at=? where component=?`, nowString(), component)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.componentUpdateState(component)})
}

func (a *App) handleComponentUpdateConfig(w http.ResponseWriter, r *http.Request) {
	component := normalizeComponent(r.PathValue("component"))
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.componentUpdateConfig(component)})
}

func (a *App) handleComponentUpdateConfigPut(w http.ResponseWriter, r *http.Request) {
	component := normalizeComponent(r.PathValue("component"))
	var req struct {
		AutoCheck     bool `json:"auto_check"`
		CheckInterval int  `json:"check_interval"`
		AutoUpdate    bool `json:"auto_update"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.CheckInterval <= 0 {
		req.CheckInterval = 86400
	}
	_, err := a.DB.Exec(`insert into component_update_config(component,auto_check,check_interval,auto_update,created_at,updated_at)
		values(?,?,?,?,?,?)
		on conflict(component) do update set auto_check=excluded.auto_check,check_interval=excluded.check_interval,auto_update=excluded.auto_update,updated_at=excluded.updated_at`,
		component, req.AutoCheck, req.CheckInterval, req.AutoUpdate, time.Now(), time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.componentUpdateConfig(component)})
}

func (a *App) componentUpdateState(component string) map[string]any {
	component = normalizeComponent(component)
	current := a.componentCurrentVersion(component)
	installed := current != "not-installed"
	state := map[string]any{
		"component":       component,
		"current_version": current,
		"latest_version":  "latest",
		"has_update":      !installed,
		"download_url":    componentDownloadURL(component),
		"status":          "idle",
		"progress":        0,
		"error_message":   "",
	}
	row := a.DB.QueryRow(`select current_version,latest_version,has_update,coalesce(download_url,''),status,progress,coalesce(error_message,''),last_check_time from component_update_info where component=?`, component)
	var last sql.NullTime
	var currentVersion, latestVersion, downloadURL, status, errText string
	var hasUpdate bool
	var progress int
	if err := row.Scan(&currentVersion, &latestVersion, &hasUpdate, &downloadURL, &status, &progress, &errText, &last); err == nil {
		state["current_version"] = currentVersion
		state["latest_version"] = latestVersion
		state["has_update"] = hasUpdate
		state["download_url"] = downloadURL
		state["status"] = status
		state["progress"] = progress
		state["error_message"] = errText
		if last.Valid {
			state["last_check_time"] = last.Time
		}
	}
	return state
}

func (a *App) componentUpdateConfig(component string) map[string]any {
	out := map[string]any{"component": component, "auto_check": true, "check_interval": 86400, "auto_update": false}
	var autoCheck, autoUpdate bool
	var interval int
	err := a.DB.QueryRow(`select auto_check,check_interval,auto_update from component_update_config where component=?`, component).Scan(&autoCheck, &interval, &autoUpdate)
	if err == nil {
		out["auto_check"] = autoCheck
		out["check_interval"] = interval
		out["auto_update"] = autoUpdate
	}
	return out
}

func (a *App) componentCurrentVersion(component string) string {
	component = normalizeComponent(component)
	target := a.componentTarget(component)
	if target == "" {
		return "unknown"
	}
	if _, err := os.Stat(target); err == nil {
		if info, err := os.Stat(target); err == nil {
			return "installed-" + info.ModTime().Format("200601021504")
		}
		return "installed"
	}
	if component == "mihomo" {
		if _, err := os.Stat(filepath.Join(a.DataDir, "data/binaries/mihomo/latest/mihomo")); err == nil {
			return "installed"
		}
	}
	return "not-installed"
}

func normalizeComponent(component string) string {
	switch component {
	case "ui", "zashboard", "dashboard":
		return "zashboard"
	case "clash", "mihomo", "proxy":
		return "mihomo"
	case "mosdns":
		return "mosdns"
	default:
		return component
	}
}
