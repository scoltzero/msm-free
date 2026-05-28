package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func (a *App) handleMonitorSystem(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"hostname": hostname(), "os": runtime.GOOS, "arch": runtime.GOARCH, "local_ips": localIPs(), "data_dir": a.DataDir,
	}})
}

func (a *App) handleMonitorHardware(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"cpu":    map[string]any{"model": cpuModel(), "cores": runtime.NumCPU(), "supports_amd64v3": supportsAMD64v3()},
		"memory": readMemInfo(),
	}})
}

func (a *App) handleMonitorResources(w http.ResponseWriter, r *http.Request) {
	mem := readMemInfo()
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"cpu_percent":    0,
		"memory_total":   mem["MemTotal"],
		"memory_free":    mem["MemAvailable"],
		"memory_used":    mem["MemTotal"] - mem["MemAvailable"],
		"memory_percent": percent(mem["MemTotal"]-mem["MemAvailable"], mem["MemTotal"]),
		"goroutines":     runtime.NumGoroutine(),
	}})
}

func (a *App) handleMonitorNetwork(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "interfaces": localIPs()})
}

func (a *App) handleMonitorHistory(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": []any{}})
}

func (a *App) handleMonitorStats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"services": a.Services.List(), "resources": map[string]any{"goroutines": runtime.NumGoroutine()},
	}})
}

func (a *App) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	checks := []map[string]any{
		{"name": "root", "ok": os.Geteuid() == 0, "message": "root is required for port 53 and nftables"},
		{"name": "mihomo", "ok": a.Services.Status("mihomo").Installed},
		{"name": "mosdns", "ok": a.Services.Status("mosdns").Installed},
		{"name": "nftables_config", "ok": fileExists(a.DataDir + "/configs/network/network.nft")},
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "checks": checks})
}

func (a *App) handleNetworkInfo(w http.ResponseWriter, r *http.Request) {
	content, _ := a.readTextFile("configs/network/network.yaml")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "config": content, "nft": fileExists(a.DataDir + "/configs/network/network.nft")})
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
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		out, err := cmd.CombinedOutput()
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
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		out, _ := cmd.CombinedOutput()
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
	if out, err := exec.Command("nft", "list", "table", "inet", "msm_free").CombinedOutput(); err == nil {
		status["table_loaded"] = true
		status["nft"] = string(out)
	}
	if out, err := exec.Command("ip", "rule", "show").CombinedOutput(); err == nil {
		text := string(out)
		status["rule_loaded"] = strings.Contains(text, "fwmark 0x1") && strings.Contains(text, "lookup 100")
		status["ip_rules"] = text
	}
	return status
}

func (a *App) handleSettingsGet(w http.ResponseWriter, r *http.Request) {
	rows, _ := a.DB.Query(`select key,value from settings`)
	defer rows.Close()
	settings := map[string]string{}
	for rows.Next() {
		var k, v string
		_ = rows.Scan(&k, &v)
		settings[k] = v
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "settings": settings})
}

func (a *App) handleSettingsPut(w http.ResponseWriter, r *http.Request) {
	var settings map[string]string
	if err := decodeJSON(r, &settings); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	for k, v := range settings {
		_, _ = a.DB.Exec(`insert or replace into settings(key,value,updated_at) values(?,?,?)`, k, v, nowString())
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
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
