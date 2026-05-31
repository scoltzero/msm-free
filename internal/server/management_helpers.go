package server

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func managedServiceNames() []string {
	return []string{"mosdns", "mihomo"}
}

func supportedServiceName(name string) bool {
	name = normalizeServiceName(name)
	return name == "mosdns" || name == "mihomo"
}

func serviceState(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running":
		return "RUNNING"
	case "error", "unknown":
		return "ERROR"
	default:
		return "STOPPED"
	}
}

func (a *App) enhancedServiceStatus(name string) map[string]any {
	name = normalizeServiceName(name)
	if !supportedServiceName(name) {
		return map[string]any{
			"name":              name,
			"display_name":      displayServiceName(name),
			"status":            "stopped",
			"state":             "STOPPED",
			"compatible_status": "STOPPED",
			"running":           false,
			"installed":         false,
			"supported":         false,
			"enabled":           false,
			"desired_enabled":   false,
			"auto_start":        false,
			"error":             "sing-box is not supported in this msm-free build",
			"health_ports":      []map[string]any{},
		}
	}
	st := a.Services.Status(name)
	desired := a.setting(serviceDesiredKey(name), "false") == "true"
	state := serviceState(st.Status)
	item := map[string]any{
		"name":              name,
		"display_name":      firstNonEmpty(st.DisplayName, displayServiceName(name)),
		"displayName":       firstNonEmpty(st.DisplayName, displayServiceName(name)),
		"status":            st.Status,
		"raw_status":        st.Status,
		"state":             state,
		"compatible_status": state,
		"running":           st.Running,
		"installed":         st.Installed,
		"supported":         true,
		"enabled":           desired,
		"desired_enabled":   desired,
		"auto_start":        desired,
		"pid":               st.PID,
		"cpu":               st.CPU,
		"cpu_percent":       st.CPU,
		"memory":            st.Memory,
		"memory_bytes":      st.Memory,
		"uptime":            st.Uptime,
		"uptime_seconds":    st.Uptime,
		"version":           a.serviceVersion(name, st),
		"binary_path":       st.BinaryPath,
		"config_path":       st.ConfigPath,
		"log_path":          st.LogPath,
		"log_paths":         a.logPathRows(name),
		"health_ports":      serviceHealthPorts(name),
		"error":             st.Error,
	}
	if spec, err := a.Services.spec(name); err == nil {
		item["stdout_path"] = spec.Stdout
		item["stderr_path"] = spec.Stderr
		item["pid_path"] = spec.PIDFile
		item["work_dir"] = spec.Dir
		item["args"] = spec.Args
	}
	return item
}

func (a *App) enhancedServiceList() []map[string]any {
	items := make([]map[string]any, 0, len(managedServiceNames()))
	for _, name := range managedServiceNames() {
		items = append(items, a.enhancedServiceStatus(name))
	}
	return items
}

func displayServiceName(name string) string {
	switch normalizeServiceName(name) {
	case "mosdns":
		return "MosDNS"
	case "mihomo":
		return "Mihomo"
	case "singbox", "sing-box":
		return "Sing-box"
	case "msm":
		return "msm-free"
	default:
		return name
	}
}

func serviceHealthPorts(name string) []map[string]any {
	defs := map[string][]struct {
		Port int
		Name string
	}{
		"mosdns": {
			{53, "DNS"}, {7777, "node-resolver"}, {8888, "internal-dns"}, {2222, "local-upstream"},
			{3333, "nocn-forward"}, {4444, "nocn-cache"}, {5656, "split-router"}, {9099, "webinfo"},
		},
		"mihomo": {
			{7890, "http"}, {7891, "socks"}, {7892, "mixed"}, {6666, "dns"},
			{7896, "tproxy"}, {7877, "redirect"}, {9090, "controller"}, {9099, "dashboard-metrics"},
		},
	}
	rows := make([]map[string]any, 0, len(defs[name]))
	for _, def := range defs[name] {
		rows = append(rows, map[string]any{"port": def.Port, "name": def.Name, "listening": tcpPortOpen(def.Port)})
	}
	return rows
}

func (a *App) serviceVersion(name string, st ServiceStatus) string {
	if !st.Installed {
		return "not-installed"
	}
	switch name {
	case "mihomo":
		return a.mihomoVersion()
	case "mosdns":
		out, err := exec.Command(st.BinaryPath, "version").CombinedOutput()
		if err == nil && strings.TrimSpace(string(out)) != "" {
			return strings.TrimSpace(string(out))
		}
		return "installed"
	default:
		return firstNonEmpty(st.Version, "installed")
	}
}

func (a *App) waitForServiceState(ctx context.Context, name, want string, timeout time.Duration) ServiceStatus {
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	deadline := time.Now().Add(timeout)
	want = strings.ToLower(want)
	var st ServiceStatus
	for {
		st = a.Services.Status(name)
		if want == "" || strings.ToLower(st.Status) == want || (want == "stopped" && !st.Running) || (want == "running" && st.Running) {
			return st
		}
		if time.Now().After(deadline) {
			return st
		}
		select {
		case <-ctx.Done():
			return st
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func requestWait(r *http.Request) (bool, time.Duration) {
	wait := queryBool(r, "wait", false)
	timeoutMS := queryInt(r, "timeout_ms", 8000)
	if timeoutMS < 100 {
		timeoutMS = 100
	}
	return wait, time.Duration(timeoutMS) * time.Millisecond
}

func queryBool(r *http.Request, key string, def bool) bool {
	raw := strings.ToLower(strings.TrimSpace(r.URL.Query().Get(key)))
	if raw == "" {
		return def
	}
	return raw == "1" || raw == "true" || raw == "yes" || raw == "on"
}

func pageParams(r *http.Request, defaultSize int) (int, int) {
	page := queryInt(r, "page", 1)
	size := queryInt(r, "page_size", defaultSize)
	if size == defaultSize {
		size = queryInt(r, "limit", defaultSize)
	}
	if size <= 0 {
		size = defaultSize
	}
	if size > 500 {
		size = 500
	}
	return page, size
}

func pagination(page, size, total int) map[string]any {
	if size <= 0 {
		size = 20
	}
	pages := 0
	if total > 0 {
		pages = (total + size - 1) / size
	}
	return map[string]any{"page": page, "page_size": size, "limit": size, "total": total, "total_pages": pages}
}

func slicePage[T any](items []T, page, size int) []T {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	start := (page - 1) * size
	if start >= len(items) {
		return []T{}
	}
	end := start + size
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

func (a *App) saveConfigFileWithHistory(rel, content, comment, username string) (int64, error) {
	rel = normalizeConfigRel(rel)
	path, err := a.safePath(rel)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return 0, err
	}
	var historyID int64
	if old, err := os.ReadFile(path); err == nil {
		if username == "" {
			username = "system"
		}
		if comment == "" {
			comment = "auto backup before save"
		}
		now := nowString()
		res, err := a.DB.Exec(`insert into config_histories(service,file_path,content,comment,is_stable,created_by,created_at,updated_at) values(?,?,?,?,?,?,?,?)`,
			serviceFromPath(rel), rel, string(old), comment, false, username, now, now)
		if err != nil {
			return 0, err
		}
		historyID, _ = res.LastInsertId()
	}
	return historyID, os.WriteFile(path, []byte(content), 0644)
}

func configWriteResult(rel string, historyID int64) map[string]any {
	service := serviceFromPath(rel)
	restart := service == "mosdns" || service == "mihomo" || service == "network"
	return map[string]any{
		"history_id":               historyID,
		"affected_service":         service,
		"restart_required":         restart,
		"network_reapply_required": service == "network",
	}
}

func (a *App) logPaths(service string) []string {
	service = normalizeServiceName(service)
	if service == "proxy" {
		service = "mihomo"
	}
	if service == "msm" || service == "msm-free" || service == "app" {
		names := []string{"msm.log", "msm-free.log", "app.log"}
		var paths []string
		for _, name := range names {
			path := filepath.Join(a.DataDir, "logs", name)
			if _, err := os.Stat(path); err == nil {
				paths = append(paths, path)
			}
		}
		return paths
	}
	if spec, err := a.Services.spec(service); err == nil {
		return []string{spec.Stdout, spec.Stderr}
	}
	st := a.Services.Status(service)
	if st.LogPath != "" {
		return []string{st.LogPath}
	}
	return nil
}

func (a *App) logPathRows(service string) []map[string]any {
	paths := a.logPaths(service)
	rows := make([]map[string]any, 0, len(paths))
	for _, path := range paths {
		info, err := os.Stat(path)
		size := int64(0)
		exists := err == nil
		if err == nil {
			size = info.Size()
		}
		rows = append(rows, map[string]any{"path": path, "name": filepath.Base(path), "exists": exists, "size": size})
	}
	return rows
}

func (a *App) serviceLogLinesWithSources(service string, n int) []map[string]string {
	paths := a.logPaths(service)
	var rows []map[string]string
	for _, path := range paths {
		lines, err := tailFile(path, n)
		if err != nil {
			continue
		}
		source := strings.TrimSuffix(filepath.Base(path), ".log")
		for _, line := range lines {
			rows = append(rows, map[string]string{"line": line, "source": source, "path": path})
		}
	}
	if len(rows) == 0 {
		rows = append(rows, journalLogRows(service, n)...)
	}
	if len(rows) > n && n > 0 {
		rows = rows[len(rows)-n:]
	}
	return rows
}

func journalLogRows(service string, n int) []map[string]string {
	unit := journalUnitForService(service)
	if unit == "" {
		return nil
	}
	if n <= 0 {
		n = 200
	}
	out, err := combinedOutputWithTimeout(context.Background(), 3*time.Second, "journalctl", "-u", unit, "-n", fmtAny(n), "--no-pager", "--output=short-iso")
	if err != nil || len(out) == 0 {
		return nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	rows := make([]map[string]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "-- No entries --") {
			continue
		}
		rows = append(rows, map[string]string{"line": line, "source": unit, "path": "journalctl:" + unit})
	}
	return rows
}

func journalUnitForService(service string) string {
	switch normalizeServiceName(service) {
	case "msm", "msm-free", "app", "":
		return "msm-free.service"
	default:
		return ""
	}
}

func filterStructuredLogs(logs []map[string]any, r *http.Request) []map[string]any {
	level := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("level")))
	search := strings.ToLower(strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("q"), r.URL.Query().Get("keyword"), r.URL.Query().Get("search"))))
	sinceRaw := strings.TrimSpace(r.URL.Query().Get("since"))
	var since time.Time
	if sinceRaw != "" {
		since, _ = time.Parse(time.RFC3339, sinceRaw)
	}
	out := make([]map[string]any, 0, len(logs))
	for _, item := range logs {
		itemLevel := strings.ToLower(fmtAny(item["level"]))
		if level != "" && level != "all" && itemLevel != level {
			continue
		}
		line := strings.ToLower(firstNonEmpty(fmtAny(item["raw"]), fmtAny(item["message"])))
		if search != "" && !strings.Contains(line, search) {
			continue
		}
		if !since.IsZero() {
			if ts, ok := parseAnyLogTime(fmtAny(item["time"])); ok && ts.Before(since) {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

func parseAnyLogTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(strings.Trim(value, "[]"))
	if value == "" {
		return time.Time{}, false
	}
	layouts := []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func zipFiles(files map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, path := range files {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		w, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		f, err := os.Open(path)
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		_, copyErr := io.Copy(w, f)
		_ = f.Close()
		if copyErr != nil {
			_ = zw.Close()
			return nil, copyErr
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func userRowCount(db *sql.DB, where string, args ...any) int {
	query := `select count(*) from users where deleted_at is null`
	if where != "" {
		query += " and " + where
	}
	var n int
	_ = db.QueryRow(query, args...).Scan(&n)
	return n
}
