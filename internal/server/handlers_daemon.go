package server

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const daemonSystemdUnit = "msm-free.service"

func (a *App) handleDaemonStatus(w http.ResponseWriter, r *http.Request) {
	payload := a.daemonStatusPayload()
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    payload,
		"status":  payload["status"],
		"running": payload["running"],
	})
}

func (a *App) handleDaemonRestart(w http.ResponseWriter, r *http.Request) {
	method := "exit-fallback"
	if daemonSystemctlUsable() {
		method = "systemd"
		go daemonRunAfterResponse(func() {
			_ = exec.Command("systemctl", "restart", strings.TrimSuffix(daemonSystemdUnit, ".service")).Run()
		})
	} else {
		go daemonRunAfterResponse(func() {
			_ = a.Services.StopAll(context.Background())
			os.Exit(2)
		})
	}
	payload := a.daemonStatusPayload()
	payload["scheduled"] = true
	payload["method"] = method
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"action":  "restart",
		"data":    payload,
	})
}

func (a *App) handleDaemonStop(w http.ResponseWriter, r *http.Request) {
	method := "exit"
	if daemonSystemctlUsable() {
		method = "systemd"
		go daemonRunAfterResponse(func() {
			_ = exec.Command("systemctl", "stop", strings.TrimSuffix(daemonSystemdUnit, ".service")).Run()
		})
	} else {
		go daemonRunAfterResponse(func() {
			_ = a.Services.StopAll(context.Background())
			os.Exit(0)
		})
	}
	payload := a.daemonStatusPayload()
	payload["scheduled"] = true
	payload["method"] = method
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"action":  "stop",
		"data":    payload,
	})
}

func (a *App) daemonStatusPayload() map[string]any {
	return map[string]any{
		"name":      "msm-free",
		"status":    "running",
		"running":   true,
		"pid":       os.Getpid(),
		"version":   a.Version,
		"data_dir":  a.DataDir,
		"platform":  runtime.GOOS + "/" + runtime.GOARCH,
		"unit":      daemonSystemdUnit,
		"systemd":   daemonSystemdStatus(),
		"services":  a.enhancedServiceList(),
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

func daemonRunAfterResponse(fn func()) {
	time.Sleep(300 * time.Millisecond)
	fn()
}

func daemonSystemdStatus() map[string]any {
	status := map[string]any{
		"unit":      daemonSystemdUnit,
		"available": daemonSystemctlAvailable(),
		"installed": daemonSystemdUnitExists(),
		"active":    "unavailable",
		"enabled":   "unavailable",
	}
	if !daemonSystemctlAvailable() {
		return status
	}
	if out, err := combinedOutputWithTimeout(context.Background(), 1500*time.Millisecond, "systemctl", "is-active", strings.TrimSuffix(daemonSystemdUnit, ".service")); err == nil || len(out) > 0 {
		status["active"] = strings.TrimSpace(string(out))
	}
	if out, err := combinedOutputWithTimeout(context.Background(), 1500*time.Millisecond, "systemctl", "is-enabled", strings.TrimSuffix(daemonSystemdUnit, ".service")); err == nil || len(out) > 0 {
		status["enabled"] = strings.TrimSpace(string(out))
	}
	return status
}

func daemonSystemctlUsable() bool {
	return daemonSystemctlAvailable() && daemonSystemdUnitExists()
}

func daemonSystemctlAvailable() bool {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}
	return fileExists("/run/systemd/system")
}

func daemonSystemdUnitExists() bool {
	for _, dir := range []string{"/etc/systemd/system", "/lib/systemd/system", "/usr/lib/systemd/system"} {
		if fileExists(filepath.Join(dir, daemonSystemdUnit)) {
			return true
		}
	}
	return false
}
