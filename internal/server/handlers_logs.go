package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (a *App) handleLogs(w http.ResponseWriter, r *http.Request) {
	service := normalizeServiceName(r.PathValue("service"))
	linesLimit := queryInt(r, "lines", 1000)
	sourceRows := a.serviceLogLinesWithSources(service, linesLimit)
	rawLines := make([]string, 0, len(sourceRows))
	logs := make([]map[string]any, 0, len(sourceRows))
	for _, row := range sourceRows {
		rawLines = append(rawLines, row["line"])
		entry := structuredLogLines([]string{row["line"]})[0]
		entry["source"] = row["source"]
		entry["path"] = row["path"]
		logs = append(logs, entry)
	}
	logs = filterStructuredLogs(logs, r)
	page, pageSize := pageParams(r, len(logs))
	if r.URL.Query().Get("page") == "" && r.URL.Query().Get("page_size") == "" && r.URL.Query().Get("limit") == "" {
		page = 1
		pageSize = len(logs)
		if pageSize == 0 {
			pageSize = 1
		}
	}
	paged := slicePage(logs, page, pageSize)
	lines := make([]string, 0, len(paged))
	for _, item := range paged {
		lines = append(lines, firstNonEmpty(fmtAny(item["raw"]), fmtAny(item["message"])))
	}
	stats := logStats(logs)
	writeJSON(w, http.StatusOK, map[string]any{
		"success":    true,
		"service":    service,
		"lines":      lines,
		"raw_lines":  rawLines,
		"logs":       paged,
		"items":      paged,
		"data":       paged,
		"content":    strings.Join(lines, "\n"),
		"pagination": pagination(page, pageSize, len(logs)),
		"stats":      stats,
		"paths":      a.logPathRows(service),
	})
}

func (a *App) handleLogsClear(w http.ResponseWriter, r *http.Request) {
	service := normalizeServiceName(r.PathValue("service"))
	cleared := 0
	for _, path := range a.logPaths(service) {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err == nil {
			_ = os.WriteFile(path, nil, 0644)
			cleared++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "service": service, "cleared": cleared})
}

func (a *App) handleLogsDownload(w http.ResponseWriter, r *http.Request) {
	service := normalizeServiceName(r.PathValue("service"))
	paths := a.logPaths(service)
	files := map[string]string{}
	for _, path := range paths {
		files[filepath.Base(path)] = path
	}
	if len(files) == 0 {
		files["README.txt"] = ""
	}
	if len(files) == 1 && r.URL.Query().Get("format") != "zip" {
		for _, path := range files {
			if path == "" {
				break
			}
			w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(path))
			serveLocalFile(w, r, path)
			return
		}
	}
	b, err := zipFiles(files)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "zip_failed", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-logs.zip", service))
	_, _ = w.Write(b)
}

func (a *App) handleLogsStats(w http.ResponseWriter, r *http.Request) {
	service := normalizeServiceName(r.PathValue("service"))
	size := int64(0)
	for _, path := range a.logPaths(service) {
		if info, err := os.Stat(path); err == nil {
			size += info.Size()
		}
	}
	stats := logStats(parseLogLines(a.serviceLogLines(service, 5000)))
	stats["service"] = service
	stats["size"] = size
	stats["file_size"] = size
	stats["paths"] = a.logPathRows(service)
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
