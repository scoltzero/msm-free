package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (a *App) handleLogs(w http.ResponseWriter, r *http.Request) {
	service := normalizeServiceName(r.PathValue("service"))
	lines := a.serviceLogLines(service, queryInt(r, "lines", 1000))
	logs := parseLogLines(lines)
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"service": service,
		"lines":   lines,
		"logs":    logs,
		"data":    logs,
		"content": strings.Join(lines, "\n"),
	})
}

func (a *App) handleLogsClear(w http.ResponseWriter, r *http.Request) {
	service := normalizeServiceName(r.PathValue("service"))
	st := a.Services.Status(service)
	if st.LogPath != "" {
		_ = os.WriteFile(st.LogPath, nil, 0644)
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleLogsDownload(w http.ResponseWriter, r *http.Request) {
	service := normalizeServiceName(r.PathValue("service"))
	st := a.Services.Status(service)
	if st.LogPath == "" {
		writeError(w, http.StatusNotFound, "not_found", "log not found")
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(st.LogPath))
	serveLocalFile(w, r, st.LogPath)
}

func (a *App) handleLogsStats(w http.ResponseWriter, r *http.Request) {
	service := normalizeServiceName(r.PathValue("service"))
	st := a.Services.Status(service)
	info, err := os.Stat(st.LogPath)
	size := int64(0)
	if err == nil {
		size = info.Size()
	}
	stats := logStats(parseLogLines(a.serviceLogLines(service, 5000)))
	stats["service"] = service
	stats["size"] = size
	stats["file_size"] = size
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": stats, "service": service, "size": size})
}

func parseLogLines(lines []string) []map[string]any {
	out := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		level := "INFO"
		upper := strings.ToUpper(line)
		switch {
		case strings.Contains(upper, "FATAL") || strings.Contains(upper, "ERROR") || strings.Contains(upper, "ERRO"):
			level = "ERROR"
		case strings.Contains(upper, "WARN") || strings.Contains(upper, "WRN"):
			level = "WARN"
		case strings.Contains(upper, "DEBUG") || strings.Contains(upper, "TRACE"):
			level = "DEBUG"
		}
		out = append(out, map[string]any{"time": "", "level": level, "message": line})
	}
	return out
}

func logStats(logs []map[string]any) map[string]any {
	stats := map[string]any{"total": len(logs), "info": 0, "warn": 0, "error": 0, "debug": 0}
	for _, entry := range logs {
		level, _ := entry["level"].(string)
		switch strings.ToUpper(level) {
		case "ERROR":
			stats["error"] = stats["error"].(int) + 1
		case "WARN", "WARNING":
			stats["warn"] = stats["warn"].(int) + 1
		case "DEBUG":
			stats["debug"] = stats["debug"].(int) + 1
		default:
			stats["info"] = stats["info"].(int) + 1
		}
	}
	return stats
}
