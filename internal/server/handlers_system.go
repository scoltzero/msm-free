package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func (a *App) handleMonitorSystem(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.monitorSystemSnapshot()})
}

func (a *App) handleMonitorHardware(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.monitorHardwareSnapshot()})
}

func (a *App) handleMonitorResources(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.monitorResourceSnapshot()})
}

func (a *App) handleMonitorNetwork(w http.ResponseWriter, r *http.Request) {
	data := a.monitorNetworkSnapshot(time.Now())
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "interfaces": data["local_ips"], "data": data})
}

func (a *App) handleMonitorHistory(w http.ResponseWriter, r *http.Request) {
	mem := readMemInfo()
	now := time.Now()
	point := map[string]any{
		"time":           now.Format(time.RFC3339),
		"timestamp":      now.Unix(),
		"cpu_percent":    sampleCPUPercent(),
		"memory_percent": percent(mem["MemTotal"]-mem["MemAvailable"], mem["MemTotal"]),
		"network":        readNetworkCounters(),
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": []any{point}})
}

func (a *App) handleMonitorStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.monitorPayload()})
}

func (a *App) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.diagnosticsPayload())
}

func (a *App) handleDiagnosticsRun(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.diagnosticsPayload())
}

func (a *App) handleDiagnosticsDownload(w http.ResponseWriter, r *http.Request) {
	payload := a.diagnosticsPayload()
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "json_error", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=msm-free-diagnostics.json")
	_, _ = w.Write(b)
}

func (a *App) diagnosticsPayload() map[string]any {
	configDirOK := dirReadable(filepath.Join(a.DataDir, "configs"))
	configFiles := a.validateConfigFiles()
	deps := dependencyChecks()
	ports := diagnosticPortRows()
	disk := diskUsage(a.DataDir)
	diskOK := diskHealthy(disk)
	permissionsOK := dirWritable(a.DataDir) && dirWritable(filepath.Join(a.DataDir, "logs")) && dirWritable(filepath.Join(a.DataDir, "configs"))
	checks := []map[string]any{
		{"name": "配置目录", "key": "config_dir", "ok": configDirOK, "message": boolMessage(configDirOK, "配置目录存在且可访问", "配置目录缺失或不可访问"), "details": filepath.Join(a.DataDir, "configs")},
		{"name": "配置文件", "key": "config_files", "ok": configFiles["ok"], "message": configFiles["message"], "details": configFiles["details"]},
		{"name": "依赖项", "key": "dependencies", "ok": deps["ok"], "message": deps["message"], "details": deps["details"]},
		{"name": "端口占用", "key": "ports", "ok": true, "message": fmt.Sprintf("已检查 %d 个端口", len(ports)), "details": ports},
		{"name": "磁盘空间", "key": "disk", "ok": diskOK, "message": boolMessage(diskOK, "磁盘空间充足", "磁盘空间不足或无法读取"), "details": disk},
		{"name": "文件权限", "key": "permissions", "ok": permissionsOK, "message": boolMessage(permissionsOK, "具有必要的读写权限", "缺少必要的读写权限"), "details": map[string]any{"data_dir": a.DataDir}},
	}
	summary := diagnosticSummary(checks)
	uiChecks := diagnosticUIChecks(checks)
	overallStatus := diagnosticOverallStatus(uiChecks)
	systemInfo := map[string]any{"os": runtime.GOOS, "arch": runtime.GOARCH, "go_version": runtime.Version(), "cpu_cores": runtime.NumCPU(), "pid": os.Getpid(), "uid": os.Getuid(), "euid": os.Geteuid(), "is_root": fmt.Sprintf("%t", os.Geteuid() == 0), "version": a.Version, "data_dir": a.DataDir}
	data := map[string]any{
		"checks":         uiChecks,
		"raw_checks":     checks,
		"summary":        summary,
		"overall_status": overallStatus,
		"ports":          ports,
		"dependencies":   deps,
		"disk":           disk,
		"network":        readNetworkCounters(),
		"system":         systemInfo,
		"system_info":    systemInfo,
	}
	return map[string]any{"success": true, "checks": uiChecks, "raw_checks": checks, "summary": summary, "overall_status": overallStatus, "system_info": systemInfo, "ports": ports, "data": data}
}

func (a *App) handleNetworkInfo(w http.ResponseWriter, r *http.Request) {
	content, _ := a.readTextFile("configs/network/network.yaml")
	data := a.networkInfoPayload(content)
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    data,
		"config":  content,
		"nft":     data["nft"],
	})
}

func (a *App) networkInfoPayload(content string) map[string]any {
	var cfg map[string]any
	_ = yaml.Unmarshal([]byte(content), &cfg)
	setup, _ := a.latestSetupConfig()
	iface := firstNonEmpty(stringMapValue(cfg, "interface"), setup.SelectedInterface, defaultSetupInterface())
	localIP := primaryIPForInterface(iface)
	if localIP == "" {
		ips := localIPs()
		if len(ips) > 0 {
			localIP = ips[0]
		}
	}
	nftPath := filepath.Join(a.DataDir, "configs/network/network.nft")
	interfaces := networkInterfaceSummaries()
	return map[string]any{
		"config":             content,
		"nft":                fileExists(nftPath),
		"nft_enabled":        fileExists(nftPath),
		"interface":          iface,
		"selected_interface": iface,
		"localIP":            localIP,
		"local_ip":           localIP,
		"localIPs":           localIPs(),
		"local_ips":          localIPs(),
		"interfaces":         interfaces,
		"ipip":               nil,
		"ipsb":               nil,
	}
}

func primaryIPForInterface(name string) string {
	if name == "" {
		return ""
	}
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return ""
	}
	addrs, _ := iface.Addrs()
	var values []string
	for _, addr := range addrs {
		values = append(values, addr.String())
	}
	return primaryInterfaceIP(values)
}

func networkInterfaceSummaries() []map[string]any {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	out := make([]map[string]any, 0, len(ifaces))
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		var values []string
		for _, addr := range addrs {
			values = append(values, addr.String())
		}
		ip := primaryInterfaceIP(values)
		out = append(out, map[string]any{
			"name":        iface.Name,
			"index":       iface.Index,
			"mac":         iface.HardwareAddr.String(),
			"flags":       iface.Flags.String(),
			"is_up":       iface.Flags&net.FlagUp != 0,
			"is_loopback": iface.Flags&net.FlagLoopback != 0,
			"addresses":   values,
			"ip":          ip,
			"primary_ip":  ip,
		})
	}
	return out
}

func (a *App) handleNFTInfo(w http.ResponseWriter, r *http.Request) {
	content, _ := a.readTextFile("configs/network/network.nft")
	status := a.nftStatus()
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "enabled": fileExists(a.DataDir + "/configs/network/network.nft"), "config": content, "status": status})
}

func (a *App) handleNFTStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.nftStatus()})
}

func (a *App) handleNFTApply(w http.ResponseWriter, r *http.Request) {
	output, err := a.applyNFT(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "output": output, "data": a.nftStatus()})
		return
	}
	a.setSetting(nftDesiredKey, "true")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "output": output, "data": a.nftStatus()})
}

func (a *App) applyNFT(ctx context.Context) (string, error) {
	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("nftables is only supported on Linux")
	}
	if os.Geteuid() != 0 {
		return "", fmt.Errorf("root permission is required to apply nftables and policy routing")
	}
	nftPath := filepath.Join(a.DataDir, "configs/network/network.nft")
	if _, err := os.Stat(nftPath); err != nil {
		return "", fmt.Errorf("nftables config is missing: %s", nftPath)
	}
	var output bytes.Buffer
	cmds := [][]string{
		{"nft", "-f", nftPath},
		{"ip", "rule", "add", "fwmark", "1", "table", "100"},
		{"ip", "route", "add", "local", "0.0.0.0/0", "dev", "lo", "table", "100"},
		{"ip", "-6", "rule", "add", "fwmark", "1", "table", "100"},
		{"ip", "-6", "route", "add", "local", "::/0", "dev", "lo", "table", "100"},
	}
	for _, args := range cmds {
		out, err := combinedOutputWithTimeout(ctx, 8*time.Second, args[0], args[1:]...)
		if len(out) > 0 {
			output.Write(out)
			if output.Len() > 0 && !bytes.HasSuffix(output.Bytes(), []byte("\n")) {
				output.WriteByte('\n')
			}
		}
		if err != nil && !strings.Contains(string(out), "File exists") {
			return output.String(), fmt.Errorf("%s: %w", strings.Join(args, " "), err)
		}
	}
	return output.String(), nil
}

func (a *App) handleNFTClear(w http.ResponseWriter, r *http.Request) {
	output, err := a.clearNFT(r.Context())
	a.setSetting(nftDesiredKey, "false")
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "output": output, "data": a.nftStatus()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "output": output, "data": a.nftStatus()})
}

func (a *App) clearNFT(ctx context.Context) (string, error) {
	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("nftables is only supported on Linux")
	}
	if os.Geteuid() != 0 {
		return "", fmt.Errorf("root permission is required to clear nftables and policy routing")
	}
	var output bytes.Buffer
	cmds := [][]string{
		{"nft", "delete", "table", "inet", "msm_free"},
		{"ip", "rule", "del", "fwmark", "1", "table", "100"},
		{"ip", "route", "del", "local", "0.0.0.0/0", "dev", "lo", "table", "100"},
		{"ip", "-6", "rule", "del", "fwmark", "1", "table", "100"},
		{"ip", "-6", "route", "del", "local", "::/0", "dev", "lo", "table", "100"},
	}
	for _, args := range cmds {
		out, _ := combinedOutputWithTimeout(ctx, 5*time.Second, args[0], args[1:]...)
		if len(out) > 0 {
			output.Write(out)
			if output.Len() > 0 && !bytes.HasSuffix(output.Bytes(), []byte("\n")) {
				output.WriteByte('\n')
			}
		}
	}
	return output.String(), nil
}

func (a *App) nftStatus() map[string]any {
	status := map[string]any{"supported": runtime.GOOS == "linux", "is_root": os.Geteuid() == 0, "table_loaded": false, "rule_loaded": false}
	if runtime.GOOS != "linux" {
		return status
	}
	if out, err := combinedOutputWithTimeout(context.Background(), 3*time.Second, "nft", "list", "table", "inet", "msm_free"); err == nil {
		status["table_loaded"] = true
		status["nft"] = string(out)
	}
	if out, err := combinedOutputWithTimeout(context.Background(), 3*time.Second, "ip", "rule", "show"); err == nil {
		text := string(out)
		status["rule_loaded"] = strings.Contains(text, "fwmark 0x1") && strings.Contains(text, "lookup 100")
		status["ip_rules"] = text
	}
	return status
}

func combinedOutputWithTimeout(ctx context.Context, timeout time.Duration, name string, args ...string) ([]byte, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, name, args...)
	out, err := cmd.CombinedOutput()
	if cmdCtx.Err() == context.DeadlineExceeded {
		return out, fmt.Errorf("%s %s timed out after %s", name, strings.Join(args, " "), timeout)
	}
	return out, err
}

func (a *App) handleSettingsGet(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(`select key,value from settings`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	defer rows.Close()
	settings := map[string]string{}
	for rows.Next() {
		var k, v string
		_ = rows.Scan(&k, &v)
		settings[k] = v
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "settings": settings, "data": settings})
}

func (a *App) handleSettingsPut(w http.ResponseWriter, r *http.Request) {
	var raw map[string]any
	if err := decodeJSON(r, &raw); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	for k, value := range raw {
		v := fmtAny(value)
		_, _ = a.DB.Exec(`insert or replace into settings(key,value,updated_at) values(?,?,?)`, k, v, nowString())
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleSettingsProfileGet(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请提供认证令牌")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": u, "user": u})
}

func (a *App) handleSettingsProfilePut(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请提供认证令牌")
		return
	}
	var req struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	_, err := a.DB.Exec(`update users set email=?,display_name=?,updated_at=? where id=?`, req.Email, req.DisplayName, time.Now(), u.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "update_failed", err.Error())
		return
	}
	updated, _ := a.userByID(u.ID)
	a.audit(u, "settings.profile.update", "settings", "", true, "")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": updated, "user": updated})
}

func (a *App) handleSettingsAppearanceGet(w http.ResponseWriter, r *http.Request) {
	appearance := map[string]string{
		"theme":        a.setting("appearance.theme", a.setting("theme", "system")),
		"language":     a.setting("appearance.language", a.setting("language", "zh-CN")),
		"compact":      a.setting("appearance.compact", "false"),
		"menu_order":   a.setting("appearance.menu_order", ""),
		"accent_color": a.setting("appearance.accent_color", ""),
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": appearance, "appearance": appearance})
}

func (a *App) handleSettingsAppearancePut(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	for key, value := range req {
		if key == "" {
			continue
		}
		a.setSetting("appearance."+key, fmtAny(value))
	}
	a.audit(currentUser(r), "settings.appearance.update", "settings", "", true, "")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": req})
}

func (a *App) handleLicenseStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"edition": "free", "status": "unlocked", "is_pro": true, "features": "all", "message": "msm-free does not enforce paid licensing",
	}})
}

func (a *App) handleHardwareFingerprint(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "fingerprint": tokenHash(hostname() + runtime.GOOS + runtime.GOARCH)})
}

func (a *App) handleLicenseNoop(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "status": "unlocked"})
}

func readMemInfo() map[string]uint64 {
	out := map[string]uint64{"MemTotal": 0, "MemAvailable": 0}
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(b), "\n") {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		key := strings.TrimSuffix(parts[0], ":")
		if key == "MemTotal" || key == "MemAvailable" {
			v, _ := strconv.ParseUint(parts[1], 10, 64)
			out[key] = v * 1024
		}
	}
	return out
}

func percent(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return float64(used) * 100 / float64(total)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func dirReadable(path string) bool {
	entries, err := os.ReadDir(path)
	return err == nil && entries != nil
}

func dirWritable(path string) bool {
	if err := os.MkdirAll(path, 0755); err != nil {
		return false
	}
	tmp := filepath.Join(path, ".msm-free-write-test")
	if err := os.WriteFile(tmp, []byte("ok"), 0644); err != nil {
		return false
	}
	_ = os.Remove(tmp)
	return true
}

func (a *App) validateConfigFiles() map[string]any {
	root := filepath.Join(a.DataDir, "configs")
	total := 0
	errors := []map[string]string{}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			return nil
		}
		total++
		b, err := os.ReadFile(path)
		if err != nil {
			errors = append(errors, map[string]string{"path": path, "error": err.Error()})
			return nil
		}
		var decoded any
		if ext == ".json" {
			err = json.Unmarshal(b, &decoded)
		} else {
			err = yaml.Unmarshal(b, &decoded)
		}
		if err != nil {
			errors = append(errors, map[string]string{"path": path, "error": err.Error()})
		}
		return nil
	})
	ok := len(errors) == 0
	return map[string]any{"ok": ok, "message": boolMessage(ok, "配置文件有效", fmt.Sprintf("发现 %d 个配置错误", len(errors))), "details": map[string]any{"total": total, "errors": errors}}
}

func dependencyChecks() map[string]any {
	names := []string{"nft", "ip", "curl", "tar", "unzip", "gzip"}
	var details []map[string]any
	okCount := 0
	for _, name := range names {
		path, err := exec.LookPath(name)
		ok := err == nil
		if ok {
			okCount++
		}
		details = append(details, map[string]any{"name": name, "ok": ok, "path": path})
	}
	allOK := okCount == len(names)
	return map[string]any{"ok": allOK, "message": fmt.Sprintf("依赖检查通过 %d/%d", okCount, len(names)), "details": details}
}

func diagnosticSummary(checks []map[string]any) map[string]any {
	passed := 0
	failed := 0
	warnings := 0
	for _, check := range checks {
		if check["ok"] == true {
			passed++
		} else {
			failed++
		}
	}
	total := len(checks)
	return map[string]any{"total": total, "passed": passed, "failed": failed, "warnings": warnings, "pass_rate": percent(uint64(passed), uint64(total))}
}

func diagnosticUIChecks(checks []map[string]any) []map[string]any {
	rows := make([]map[string]any, 0, len(checks))
	for _, check := range checks {
		ok := check["ok"] == true
		status := "error"
		if ok {
			status = "success"
		}
		key := fmtAny(check["key"])
		if !ok && key == "dependencies" {
			status = "warning"
		}
		rows = append(rows, map[string]any{
			"name":        fmtAny(check["name"]),
			"key":         key,
			"ok":          ok,
			"status":      status,
			"message":     fmtAny(check["message"]),
			"details":     diagnosticDetailsText(check["details"]),
			"raw_details": check["details"],
		})
	}
	return rows
}

func diagnosticOverallStatus(checks []map[string]any) string {
	hasWarning := false
	for _, check := range checks {
		switch fmtAny(check["status"]) {
		case "error":
			return "critical"
		case "warning":
			hasWarning = true
		}
	}
	if hasWarning {
		return "warning"
	}
	return "healthy"
}

func diagnosticDetailsText(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []byte:
		return string(v)
	default:
		if b, err := json.MarshalIndent(v, "", "  "); err == nil {
			return string(b)
		}
		return fmtAny(v)
	}
}

func boolMessage(ok bool, good, bad string) string {
	if ok {
		return good
	}
	return bad
}

func diskHealthy(disk map[string]any) bool {
	value, ok := disk["percent"].(float64)
	return disk["ok"] == true && ok && value < 95
}
