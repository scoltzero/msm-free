package server

import (
	"net/http"
	"strings"
)

func (a *App) handleServices(w http.ResponseWriter, r *http.Request) {
	items := a.enhancedServiceList()
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items, "services": items})
}

func (a *App) handleService(w http.ResponseWriter, r *http.Request) {
	name := normalizeServiceName(r.PathValue("name"))
	item := a.enhancedServiceStatus(name)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": item, "service": item})
}

func (a *App) handleServiceExists(w http.ResponseWriter, r *http.Request) {
	name := normalizeServiceName(r.PathValue("name"))
	item := a.enhancedServiceStatus(name)
	exists, _ := item["installed"].(bool)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "exists": exists, "data": item})
}

func (a *App) handleServiceStart(w http.ResponseWriter, r *http.Request) {
	name := normalizeServiceName(r.PathValue("name"))
	st, err := a.Services.Start(r.Context(), name)
	wait, timeout := requestWait(r)
	if wait && err == nil {
		st = a.waitForServiceState(r.Context(), name, "running", timeout)
	}
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": a.enhancedServiceStatus(name), "raw": st})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.enhancedServiceStatus(name), "raw": st})
}

func (a *App) handleServiceStop(w http.ResponseWriter, r *http.Request) {
	name := normalizeServiceName(r.PathValue("name"))
	st, err := a.Services.Stop(r.Context(), name)
	wait, timeout := requestWait(r)
	if wait && err == nil {
		st = a.waitForServiceState(r.Context(), name, "stopped", timeout)
	}
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": a.enhancedServiceStatus(name), "raw": st})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.enhancedServiceStatus(name), "raw": st})
}

func (a *App) handleServiceRestart(w http.ResponseWriter, r *http.Request) {
	name := normalizeServiceName(r.PathValue("name"))
	st, err := a.Services.Restart(r.Context(), name)
	wait, timeout := requestWait(r)
	if wait && err == nil {
		st = a.waitForServiceState(r.Context(), name, "running", timeout)
	}
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": a.enhancedServiceStatus(name), "raw": st})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.enhancedServiceStatus(name), "raw": st})
}

func (a *App) handleServiceLogs(w http.ResponseWriter, r *http.Request) {
	name := normalizeServiceName(r.PathValue("name"))
	lines := filterLogLines(a.serviceLogLines(name, queryInt(r, "lines", 300)), r)
	logs := structuredLogLines(lines)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "service": name, "lines": lines, "logs": logs, "items": logs, "content": strings.Join(lines, "\n"), "paths": a.logPathRows(name)})
}

func (a *App) handleServiceConfig(w http.ResponseWriter, r *http.Request) {
	name := normalizeServiceName(r.PathValue("name"))
	var req struct {
		Content string `json:"content"`
		Config  string `json:"config"`
		Comment string `json:"comment"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Content == "" {
		req.Content = req.Config
	}
	rel := "configs/" + name + "/config.yaml"
	if name == "proxy" {
		rel = "configs/mihomo/config.yaml"
	}
	historyID, err := a.saveConfigFileWithHistory(rel, req.Content, firstNonEmpty(req.Comment, "service config update"), currentUsername(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "write_failed", err.Error())
		return
	}
	result := configWriteResult(rel, historyID)
	result["success"] = true
	writeJSON(w, http.StatusOK, result)
}

func (a *App) handleProxySummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.proxySnapshot()})
}

func (a *App) handleServicesStartAll(w http.ResponseWriter, r *http.Request) {
	a.handleServicesBatch(w, r, "start")
}

func (a *App) handleServicesStopAll(w http.ResponseWriter, r *http.Request) {
	a.handleServicesBatch(w, r, "stop")
}

func (a *App) handleServicesRestartAll(w http.ResponseWriter, r *http.Request) {
	a.handleServicesBatch(w, r, "restart")
}

func (a *App) handleServicesBatch(w http.ResponseWriter, r *http.Request, action string) {
	wait, timeout := requestWait(r)
	results := make([]map[string]any, 0, len(managedServiceNames()))
	success := true
	for _, name := range managedServiceNames() {
		var err error
		switch action {
		case "start":
			_, err = a.Services.Start(r.Context(), name)
			if wait && err == nil {
				_ = a.waitForServiceState(r.Context(), name, "running", timeout)
			}
		case "stop":
			_, err = a.Services.Stop(r.Context(), name)
			if wait && err == nil {
				_ = a.waitForServiceState(r.Context(), name, "stopped", timeout)
			}
		case "restart":
			_, err = a.Services.Restart(r.Context(), name)
			if wait && err == nil {
				_ = a.waitForServiceState(r.Context(), name, "running", timeout)
			}
		}
		item := a.enhancedServiceStatus(name)
		if err != nil {
			success = false
			item["success"] = false
			item["error"] = err.Error()
		} else {
			item["success"] = true
		}
		results = append(results, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": success, "action": action, "data": results, "services": results})
}

func normalizeServiceName(name string) string {
	if name == "proxy" || name == "clash" {
		return "mihomo"
	}
	return name
}

func (a *App) serviceLogLines(name string, n int) []string {
	var merged []string
	for _, row := range a.serviceLogLinesWithSources(name, n) {
		merged = append(merged, row["line"])
	}
	if len(merged) > n && n > 0 {
		merged = merged[len(merged)-n:]
	}
	return merged
}
