package server

import (
	"net/http"
	"strings"
)

func (a *App) handleServices(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.Services.List(), "services": a.Services.List()})
}

func (a *App) handleService(w http.ResponseWriter, r *http.Request) {
	name := normalizeServiceName(r.PathValue("name"))
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.Services.Status(name)})
}

func (a *App) handleServiceExists(w http.ResponseWriter, r *http.Request) {
	name := normalizeServiceName(r.PathValue("name"))
	st := a.Services.Status(name)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "exists": st.Installed, "data": st})
}

func (a *App) handleServiceStart(w http.ResponseWriter, r *http.Request) {
	name := normalizeServiceName(r.PathValue("name"))
	st, err := a.Services.Start(r.Context(), name)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": st})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": st})
}

func (a *App) handleServiceStop(w http.ResponseWriter, r *http.Request) {
	name := normalizeServiceName(r.PathValue("name"))
	st, err := a.Services.Stop(r.Context(), name)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": st})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": st})
}

func (a *App) handleServiceRestart(w http.ResponseWriter, r *http.Request) {
	name := normalizeServiceName(r.PathValue("name"))
	st, err := a.Services.Restart(r.Context(), name)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": st})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": st})
}

func (a *App) handleServiceLogs(w http.ResponseWriter, r *http.Request) {
	name := normalizeServiceName(r.PathValue("name"))
	lines := a.serviceLogLines(name, 300)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "lines": lines, "content": strings.Join(lines, "\n")})
}

func (a *App) handleServiceConfig(w http.ResponseWriter, r *http.Request) {
	name := normalizeServiceName(r.PathValue("name"))
	var req struct {
		Content string `json:"content"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	rel := "configs/" + name + "/config.yaml"
	if name == "proxy" {
		rel = "configs/mihomo/config.yaml"
	}
	if err := a.writeTextFile(rel, req.Content); err != nil {
		writeError(w, http.StatusBadRequest, "write_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleProxySummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.proxySnapshot()})
}

func normalizeServiceName(name string) string {
	if name == "proxy" || name == "clash" {
		return "mihomo"
	}
	return name
}

func (a *App) serviceLogLines(name string, n int) []string {
	spec, err := a.Services.spec(name)
	if err != nil {
		st := a.Services.Status(name)
		if st.LogPath == "" {
			return nil
		}
		lines, err := tailFile(st.LogPath, n)
		if err != nil {
			return []string{err.Error()}
		}
		return lines
	}
	var merged []string
	for _, path := range []string{spec.Stdout, spec.Stderr} {
		if path == "" {
			continue
		}
		lines, err := tailFile(path, n)
		if err != nil {
			continue
		}
		merged = append(merged, lines...)
	}
	if len(merged) > n && n > 0 {
		merged = merged[len(merged)-n:]
	}
	return merged
}
