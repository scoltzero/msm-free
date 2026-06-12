package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	updateConfigAutoCheckKey         = "update.auto_check"
	updateConfigAutoUpdateKey        = "update.auto_update"
	updateConfigCheckIntervalKey     = "update.check_interval"
	updateConfigNotifyKey            = "update.notify"
	updateConfigMosDNSUpgradeModeKey = "update.mosdns_upgrade_mode"
	updateConfigMihomoUpgradeModeKey = "update.mihomo_upgrade_mode"
	defaultUpdateCheckInterval       = 12 * 60 * 60
)

func (a *App) registerUpdateRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/update/status", a.handleUpdateStatus)
	mux.HandleFunc("POST /api/v1/update/check", a.handleUpdateCheck)
	mux.HandleFunc("GET /api/v1/update/config", a.handleUpdateConfig)
	mux.HandleFunc("PUT /api/v1/update/config", a.handleUpdateConfigPut)
	mux.HandleFunc("GET /api/v1/update/releases", a.handleUpdateReleases)
	mux.HandleFunc("POST /api/v1/update/download", a.handleUpdateDownload)
	mux.HandleFunc("POST /api/v1/update/install", a.handleUpdateInstall)
	mux.HandleFunc("POST /api/v1/update/cancel", a.handleUpdateCancel)

	mux.HandleFunc("GET /api/v1/component-updates", a.handleComponentUpdates)
	mux.HandleFunc("GET /api/v1/component-updates/{component}", a.handleComponentUpdateStatus)
	mux.HandleFunc("GET /api/v1/component-updates/{component}/status", a.handleComponentUpdateStatus)
	mux.HandleFunc("POST /api/v1/component-updates/{component}/check", a.handleComponentUpdateCheck)
	mux.HandleFunc("POST /api/v1/component-updates/{component}/update", a.handleComponentUpdateRun)
	mux.HandleFunc("POST /api/v1/component-updates/{component}/upload", a.handleComponentUpdateUpload)
	mux.HandleFunc("GET /api/v1/component-updates/{component}/config", a.handleComponentUpdateConfig)
	mux.HandleFunc("PUT /api/v1/component-updates/{component}/config", a.handleComponentUpdateConfigPut)
}

func (a *App) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	item := a.selfUpdateState()
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": item})
}

func (a *App) selfUpdateState() map[string]any {
	item := map[string]any{
		"component":       "msf",
		"current_version": a.Version,
		"latest_version":  a.Version,
		"has_update":      false,
		"status":          "idle",
		"progress":        0,
		"supported":       true,
	}
	if IsDockerRuntime() {
		item["supported"] = false
		item["disabled_reason"] = DockerUpdateDisabledReason()
		return item
	}
	row := a.DB.QueryRow(`select current_version,latest_version,has_update,status,progress,coalesce(error_message,''),coalesce(download_url,''),coalesce(release_notes,''),last_check_time from update_info where component='msf' order by id desc limit 1`)
	var current, latest, status, errText, downloadURL, notes string
	var hasUpdate bool
	var progress int
	var last sql.NullTime
	if err := row.Scan(&current, &latest, &hasUpdate, &status, &progress, &errText, &downloadURL, &notes, &last); err == nil {
		item["current_version"] = current
		item["latest_version"] = latest
		item["has_update"] = hasUpdate
		item["status"] = status
		item["progress"] = progress
		item["error_message"] = errText
		item["download_url"] = downloadURL
		item["release_notes"] = notes
		if last.Valid {
			item["last_check_time"] = last.Time
		}
		if downloadURL != "" {
			item["effective_download_url"] = a.rewriteDownloadURL(downloadURL)
			if fileExists(filepath.Join(a.DataDir, "data", "updates", filepath.Base(downloadURL))) {
				item["can_install"] = true
			}
		}
		if latest != "" && !versionDifferent(a.Version, latest) && oneOf(status, "installing", "downloaded", "checked") {
			item["current_version"] = a.Version
			item["has_update"] = false
			item["status"] = "completed"
			item["progress"] = 100
			item["can_install"] = false
		}
	}
	return item
}

func (a *App) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if IsDockerRuntime() {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": DockerUpdateDisabledReason(), "data": a.selfUpdateState()})
		return
	}
	release, err := a.fetchLatestRelease("scoltzero", "msf")
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": a.selfUpdateState()})
		return
	}
	downloadURL := releaseAssetURL(release, selfUpdateAssetContainsFor(runtime.GOOS, runtime.GOARCH), ".tar.gz")
	hasUpdate := versionDifferent(a.Version, release.TagName)
	now := time.Now()
	_, _ = a.DB.Exec(`insert into update_info(component,current_version,latest_version,has_update,status,progress,error_message,download_url,release_notes,last_check_time,created_at,updated_at)
		values('msf',?,?,?,?,?,?,?,?,?,?,?)
		on conflict(id) do nothing`,
		a.Version, release.TagName, hasUpdate, "checked", 0, "", downloadURL, release.Body, now, now, now)
	_, _ = a.DB.Exec(`update update_info set current_version=?,latest_version=?,has_update=?,status='checked',progress=0,error_message='',download_url=?,release_notes=?,last_check_time=?,updated_at=? where component='msf'`,
		a.Version, release.TagName, hasUpdate, downloadURL, release.Body, now, now)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"component":       "msf",
		"current_version": a.Version,
		"latest_version":  release.TagName,
		"has_update":      hasUpdate,
		"download_url":    downloadURL,
		"release_notes":   release.Body,
		"status":          "checked",
		"last_check_time": now,
	}})
}

func (a *App) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.updateConfig()})
}

func (a *App) handleUpdateConfigPut(w http.ResponseWriter, r *http.Request) {
	var raw map[string]any
	if err := decodeJSON(r, &raw); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	cfg := a.updateConfig()
	if value, ok := raw["auto_check"]; ok {
		parsed, err := structuredBoolValue(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid auto_check")
			return
		}
		cfg["auto_check"] = parsed
	}
	if value, ok := raw["auto_update"]; ok {
		parsed, err := structuredBoolValue(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid auto_update")
			return
		}
		cfg["auto_update"] = parsed
	}
	if value, ok := raw["notify"]; ok {
		parsed, err := structuredBoolValue(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid notify")
			return
		}
		cfg["notify"] = parsed
	}
	if value, ok := raw["check_interval"]; ok {
		parsed, err := structuredPositiveInt(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid check_interval")
			return
		}
		cfg["check_interval"] = parsed
	}
	if value, ok := raw["mosdns_upgrade_mode"]; ok {
		mode := strings.ToLower(strings.TrimSpace(fmtAny(value)))
		if !oneOf(mode, "full", "incremental", "reset") {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid mosdns_upgrade_mode")
			return
		}
		cfg["mosdns_upgrade_mode"] = mode
	}
	if value, ok := raw["mihomo_upgrade_mode"]; ok {
		mode := strings.ToLower(strings.TrimSpace(fmtAny(value)))
		if !oneOf(mode, "skip", "full") {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid mihomo_upgrade_mode")
			return
		}
		cfg["mihomo_upgrade_mode"] = mode
	}
	a.saveUpdateConfig(cfg)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.updateConfig()})
}

func (a *App) handleUpdateReleases(w http.ResponseWriter, r *http.Request) {
	releases, err := a.fetchReleases("scoltzero", "msf")
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": []any{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": releases})
}

func (a *App) handleUpdateDownload(w http.ResponseWriter, r *http.Request) {
	if IsDockerRuntime() {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": DockerUpdateDisabledReason(), "data": a.selfUpdateState()})
		return
	}
	state := a.selfUpdateState()
	rawURL := strings.TrimSpace(fmt.Sprint(state["download_url"]))
	if rawURL == "" {
		release, err := a.fetchLatestRelease("scoltzero", "msf")
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": state})
			return
		}
		rawURL = releaseAssetURL(release, selfUpdateAssetContainsFor(runtime.GOOS, runtime.GOARCH), ".tar.gz")
	}
	if rawURL == "" {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": "no " + selfUpdateAssetContainsFor(runtime.GOOS, runtime.GOARCH) + " release asset found", "data": state})
		return
	}
	dest := filepath.Join(a.DataDir, "data", "updates", filepath.Base(rawURL))
	effectiveURL := a.rewriteDownloadURL(rawURL)
	last := DownloadEvent{Status: "running", Progress: 5, Message: "starting"}
	err := a.downloadFile(rawURL, dest, func(ev DownloadEvent) {
		last = ev
		_, _ = a.DB.Exec(`update update_info set status='downloading',progress=?,error_message='',updated_at=? where component='msf'`, ev.Progress, nowString())
	})
	if err != nil {
		_, _ = a.DB.Exec(`update update_info set status='failed',progress=?,error_message=?,updated_at=? where component='msf'`, last.Progress, err.Error(), nowString())
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": a.selfUpdateState()})
		return
	}
	_, _ = a.DB.Exec(`update update_info set status='downloaded',progress=100,error_message='',download_url=?,updated_at=? where component='msf'`, rawURL, nowString())
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"path": dest, "download_url": rawURL, "effective_download_url": effectiveURL, "event": last}})
}

func (a *App) handleUpdateInstall(w http.ResponseWriter, r *http.Request) {
	if IsDockerRuntime() {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": DockerUpdateDisabledReason(), "data": a.selfUpdateState()})
		return
	}
	if os.Geteuid() != 0 {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": "需要 root 权限才能安装并重启 msf", "data": a.selfUpdateState()})
		return
	}
	if serverIsUnraidRuntime() {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": "Unraid 环境请通过插件管理页面更新 msf", "data": a.selfUpdateState()})
		return
	}
	state := a.selfUpdateState()
	rawURL := strings.TrimSpace(fmt.Sprint(state["download_url"]))
	if rawURL == "" {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": "请先检查并下载更新包", "data": state})
		return
	}
	archivePath := filepath.Join(a.DataDir, "data", "updates", filepath.Base(rawURL))
	if _, err := os.Stat(archivePath); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": "更新包不存在，请先下载更新", "data": state})
		return
	}
	_, _ = a.DB.Exec(`update update_info set status='installing',progress=95,error_message='',updated_at=? where component='msf'`, nowString())
	if err := a.startSelfUpdateInstaller(archivePath); err != nil {
		_, _ = a.DB.Exec(`update update_info set status='failed',progress=95,error_message=?,updated_at=? where component='msf'`, err.Error(), nowString())
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": a.selfUpdateState()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "message": "更新安装已开始，服务将自动重启。", "data": a.selfUpdateState()})
}

func (a *App) startSelfUpdateInstaller(archivePath string) error {
	workDir := filepath.Join(a.DataDir, "data", "updates", "install-"+time.Now().Format("20060102150405"))
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return err
	}
	if err := untarGz(archivePath, workDir); err != nil {
		return err
	}
	installScript := filepath.Join(workDir, "install.sh")
	if _, err := os.Stat(installScript); err != nil {
		return fmt.Errorf("update archive missing install.sh")
	}
	if err := os.Chmod(installScript, 0755); err != nil {
		return err
	}
	args := []string{
		"--prefix", selfUpdateInstallPrefix(),
		"--data-dir", a.DataDir,
		"--service-name", "msf",
		"--port", strconv.Itoa(a.selfUpdateWebPort()),
	}
	if serverSystemdAvailable() {
		unit := "msf-self-update-" + time.Now().Format("20060102150405")
		runArgs := append([]string{"--unit", unit, "--collect", "--property=Type=oneshot", "/bin/sh", installScript}, args...)
		out, err := exec.Command("systemd-run", runArgs...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("start update installer: %w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	}
	go func() {
		cmdArgs := append([]string{installScript}, args...)
		cmd := exec.Command("sh", cmdArgs...)
		cmd.Dir = workDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			_, _ = a.DB.Exec(`update update_info set status='failed',progress=95,error_message=?,updated_at=? where component='msf'`, fmt.Sprintf("%v: %s", err, strings.TrimSpace(string(out))), nowString())
			return
		}
		_, _ = a.DB.Exec(`update update_info set current_version=?,has_update=false,status='completed',progress=100,error_message='',updated_at=? where component='msf'`, a.Version, nowString())
	}()
	return nil
}

func selfUpdateInstallPrefix() string {
	exe, err := os.Executable()
	if err != nil {
		return "/usr/local"
	}
	dir := filepath.Dir(exe)
	if filepath.Base(dir) == "bin" {
		return filepath.Dir(dir)
	}
	return "/usr/local"
}

func (a *App) selfUpdateWebPort() int {
	var port int
	if err := a.DB.QueryRow(`select web_port from system_setups order by id desc limit 1`).Scan(&port); err != nil || port <= 0 {
		return 7777
	}
	return port
}

func serverSystemdAvailable() bool {
	if _, err := exec.LookPath("systemd-run"); err != nil {
		return false
	}
	if st, err := os.Stat("/run/systemd/system"); err == nil && st.IsDir() {
		return true
	}
	return false
}

func serverIsUnraidRuntime() bool {
	if fileExists("/etc/unraid-version") || fileExists("/usr/local/sbin/emhttp") || fileExists("/boot/config/plugins") {
		return true
	}
	return strings.Contains(strings.ToLower(os.Getenv("UNRAID_VERSION")), "unraid")
}

func (a *App) handleUpdateCancel(w http.ResponseWriter, r *http.Request) {
	_, _ = a.DB.Exec(`update update_info set status='idle',progress=0,updated_at=? where component='msf'`, nowString())
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.selfUpdateState()})
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
	remote, err := a.componentRemoteInfo(component)
	if err != nil {
		state["status"] = "failed"
		state["error_message"] = err.Error()
		state["last_check_time"] = time.Now()
		_, _ = a.DB.Exec(`insert into component_update_info(component,current_version,latest_version,has_update,download_url,download_digest,verified_digest,verified,verification_source,release_body,status,progress,error_message,last_check_time,created_at,updated_at)
			values(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
			on conflict(component) do update set current_version=excluded.current_version,status='failed',progress=0,error_message=excluded.error_message,last_check_time=excluded.last_check_time,updated_at=excluded.updated_at,verified=false`,
			component, state["current_version"], state["latest_version"], state["has_update"], state["download_url"], "", "", false, "", "", "failed", 0, err.Error(), state["last_check_time"], time.Now(), time.Now())
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": state})
		return
	}
	latestVersion := a.componentRemoteVersion(component, remote)
	downloadAsset, err := a.componentDownloadAssetFromRelease(component, remote)
	if err != nil {
		now := time.Now()
		state["status"] = "failed"
		state["error_message"] = err.Error()
		state["last_check_time"] = now
		state["latest_version"] = latestVersion
		state["download_url"] = a.componentDownloadURL(component)
		state["download_digest"] = ""
		state["verified_digest"] = ""
		state["verified"] = false
		state["verification_source"] = ""
		state["release_body"] = remote.Body
		_, _ = a.DB.Exec(`insert into component_update_info(component,current_version,latest_version,has_update,download_url,download_digest,verified_digest,verified,verification_source,release_body,status,progress,error_message,last_check_time,created_at,updated_at)
			values(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
			on conflict(component) do update set current_version=excluded.current_version,latest_version=excluded.latest_version,download_url=excluded.download_url,download_digest='',verified_digest='',verified=false,verification_source='',release_body=excluded.release_body,status='failed',progress=0,error_message=excluded.error_message,last_check_time=excluded.last_check_time,updated_at=excluded.updated_at`,
			component, state["current_version"], latestVersion, false, a.componentDownloadURL(component), "", "", false, "", remote.Body, "failed", 0, err.Error(), now, now, now)
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": state})
		return
	}
	downloadURL := downloadAsset.URL
	downloadDigest := downloadAsset.Digest
	now := time.Now()
	currentVersion := firstNonEmpty(componentStateString(state, "current_version_detail"), componentStateString(state, "current_version"))
	hasUpdate := componentHasUpdate(currentVersion, latestVersion)
	verifiedDigest, verified := preservedComponentVerification(state, downloadDigest, hasUpdate)
	displayVersion, detailVersion := componentDisplayCurrentVersion(component, currentVersion, latestVersion)
	state["current_version"] = displayVersion
	if detailVersion != "" {
		state["current_version_detail"] = detailVersion
	} else {
		delete(state, "current_version_detail")
	}
	state["latest_version"] = latestVersion
	state["download_url"] = downloadURL
	state["download_digest"] = downloadDigest
	state["verified_digest"] = verifiedDigest
	state["verified"] = verified
	state["verification_source"] = downloadAsset.VerificationSource
	state["release_body"] = remote.Body
	state["has_update"] = hasUpdate
	state["can_update"] = componentCanUpdate(currentVersion, latestVersion, hasUpdate, downloadURL)
	state["status"] = "checked"
	state["progress"] = 0
	state["error_message"] = ""
	state["last_check_time"] = now
	_, _ = a.DB.Exec(`insert into component_update_info(component,current_version,latest_version,has_update,download_url,download_digest,verified_digest,verified,verification_source,release_body,status,progress,error_message,last_check_time,created_at,updated_at)
		values(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		on conflict(component) do update set current_version=excluded.current_version,latest_version=excluded.latest_version,has_update=excluded.has_update,download_url=excluded.download_url,download_digest=excluded.download_digest,verified_digest=excluded.verified_digest,verified=excluded.verified,verification_source=excluded.verification_source,release_body=excluded.release_body,status='checked',progress=0,error_message='',last_check_time=excluded.last_check_time,updated_at=excluded.updated_at`,
		component, currentVersion, latestVersion, hasUpdate, downloadURL, downloadDigest, verifiedDigest, verified, downloadAsset.VerificationSource, remote.Body, "checked", 0, "", now, now, now)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": state})
}

func (a *App) handleComponentUpdateRun(w http.ResponseWriter, r *http.Request) {
	component := normalizeComponent(r.PathValue("component"))
	if component == "" {
		writeError(w, http.StatusBadRequest, "bad_component", "unknown component")
		return
	}
	now := time.Now()
	state := a.componentUpdateState(component)
	latestVersion := componentStateString(state, "latest_version")
	downloadURL := componentStateString(state, "download_url")
	downloadDigest := componentStateString(state, "download_digest")
	verifiedDigest := ""
	verificationSource := componentStateString(state, "verification_source")
	verified := false
	releaseBody := ""
	if remote, err := a.componentRemoteInfo(component); err == nil {
		latestVersion = a.componentRemoteVersion(component, remote)
		if asset, err := a.componentDownloadAssetFromRelease(component, remote); err == nil {
			downloadURL = asset.URL
			downloadDigest = asset.Digest
			verificationSource = asset.VerificationSource
		} else {
			downloadURL = a.componentDownloadURL(component)
			downloadDigest = ""
			verificationSource = ""
		}
		releaseBody = remote.Body
	}
	if isPlaceholderComponentVersion(latestVersion) {
		latestVersion = "-"
	}
	if downloadURL == "" {
		downloadURL = a.componentDownloadURL(component)
	}
	wasRunning := component != "zashboard" && component != "ui" && a.Services.Status(component).Running
	_, _ = a.DB.Exec(`insert into component_update_info(component,current_version,latest_version,has_update,download_url,download_digest,verified_digest,verified,verification_source,release_body,status,progress,error_message,created_at,updated_at)
		values(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		on conflict(component) do update set current_version=excluded.current_version,latest_version=excluded.latest_version,download_url=excluded.download_url,download_digest=excluded.download_digest,verified_digest='',verified=false,verification_source=excluded.verification_source,release_body=excluded.release_body,status='running',progress=5,error_message='',updated_at=excluded.updated_at`,
		component, a.componentCurrentVersion(component), latestVersion, true, downloadURL, downloadDigest, "", false, verificationSource, releaseBody, "running", 5, "", now, now)
	last := DownloadEvent{Status: "running", Progress: 5, Message: "starting"}
	err := a.installComponent(component, func(ev DownloadEvent) {
		last = ev
		if ev.DownloadDigest != "" {
			downloadDigest = ev.DownloadDigest
		}
		if ev.VerifiedDigest != "" {
			verifiedDigest = ev.VerifiedDigest
		}
		if ev.VerificationSource != "" {
			verificationSource = ev.VerificationSource
		}
		if ev.Verified {
			verified = true
		}
		_, _ = a.DB.Exec(`update component_update_info set status=?, progress=?, download_digest=?, verified_digest=?, verified=?, verification_source=?, error_message='', updated_at=? where component=?`,
			ev.Status, ev.Progress, downloadDigest, verifiedDigest, verified, verificationSource, nowString(), component)
	})
	if err != nil {
		_, _ = a.DB.Exec(`update component_update_info set status='failed', progress=?, error_message=?, updated_at=? where component=?`, last.Progress, err.Error(), nowString(), component)
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": a.componentUpdateState(component)})
		return
	}
	restarted := false
	if wasRunning {
		if _, err := a.Services.Restart(r.Context(), component); err != nil {
			_, _ = a.DB.Exec(`update component_update_info set status='failed', progress=100, error_message=?, updated_at=? where component=?`, "installed but restart failed: "+err.Error(), nowString(), component)
			writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": "installed but restart failed: " + err.Error(), "data": a.componentUpdateState(component)})
			return
		}
		restarted = true
	}
	currentVersion := a.componentCurrentVersion(component)
	if component == "zashboard" && !isPlaceholderComponentVersion(latestVersion) {
		currentVersion = latestVersion
	}
	_, _ = a.DB.Exec(`update component_update_info set current_version=?, latest_version=?, has_update=false, download_digest=?, verified_digest=?, verified=?, verification_source=?, status='completed', progress=100, error_message='', last_check_time=?, updated_at=? where component=?`, currentVersion, latestVersion, downloadDigest, verifiedDigest, verified, verificationSource, now, nowString(), component)
	next := a.componentUpdateState(component)
	next["restarted"] = restarted
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": next})
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
		req.CheckInterval = defaultUpdateCheckInterval
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
		"component":           component,
		"current_version":     current,
		"latest_version":      "-",
		"has_update":          !installed,
		"can_update":          a.componentDownloadURL(component) != "",
		"download_url":        a.componentDownloadURL(component),
		"download_digest":     "",
		"verified_digest":     "",
		"verified":            false,
		"verification_source": "",
		"status":              "idle",
		"progress":            0,
		"error_message":       "",
	}
	if display, detail := componentDisplayCurrentVersion(component, current, "-"); display != current {
		state["current_version"] = display
		state["current_version_detail"] = detail
	}
	row := a.DB.QueryRow(`select current_version,latest_version,has_update,coalesce(download_url,''),coalesce(download_digest,''),coalesce(verified_digest,''),coalesce(verified,false),coalesce(verification_source,''),status,progress,coalesce(error_message,''),last_check_time from component_update_info where component=?`, component)
	var last sql.NullTime
	var currentVersion, latestVersion, downloadURL, downloadDigest, verifiedDigest, verificationSource, status, errText string
	var hasUpdate, verified bool
	var progress int
	if err := row.Scan(&currentVersion, &latestVersion, &hasUpdate, &downloadURL, &downloadDigest, &verifiedDigest, &verified, &verificationSource, &status, &progress, &errText, &last); err == nil {
		if shouldUseLiveComponentVersion(component, current, currentVersion) {
			currentVersion = current
		}
		if isPlaceholderComponentVersion(latestVersion) {
			latestVersion = "-"
		}
		if currentVersion == "not-installed" {
			hasUpdate = true
		} else if isPlaceholderComponentVersion(latestVersion) {
			hasUpdate = false
		} else if !isPlaceholderComponentVersion(latestVersion) {
			hasUpdate = componentHasUpdate(currentVersion, latestVersion)
		}
		displayVersion, detailVersion := componentDisplayCurrentVersion(component, currentVersion, latestVersion)
		state["current_version"] = displayVersion
		if detailVersion != "" {
			state["current_version_detail"] = detailVersion
		} else {
			delete(state, "current_version_detail")
		}
		state["latest_version"] = latestVersion
		state["has_update"] = hasUpdate
		state["can_update"] = componentCanUpdate(currentVersion, latestVersion, hasUpdate, firstNonEmpty(downloadURL, a.componentDownloadURL(component)))
		state["download_url"] = downloadURL
		state["download_digest"] = downloadDigest
		state["verified_digest"] = verifiedDigest
		state["verified"] = verified
		state["verification_source"] = verificationSource
		state["status"] = status
		state["progress"] = progress
		state["error_message"] = errText
		if last.Valid {
			state["last_check_time"] = last.Time
		}
	}
	return state
}

func (a *App) componentRemoteInfo(component string) (githubRelease, error) {
	switch normalizeComponent(component) {
	case "mosdns":
		return a.fetchReleaseByTag("baozaodetudou", "mssb", "mosdns")
	case "mihomo":
		return a.fetchReleaseByTag("baozaodetudou", "mssb", "mihomo")
	case "zashboard":
		return a.fetchLatestRelease("Zephyruso", "zashboard")
	default:
		return githubRelease{}, fmt.Errorf("unknown component %s", component)
	}
}

var componentVersionTokenRE = regexp.MustCompile(`(?i)(ph-yyds-[0-9a-z._-]+|v?\d+(?:\.\d+){1,3}(?:[-+][0-9a-z._-]+)?)`)
var componentCommitTokenRE = regexp.MustCompile(`(?i)\b[0-9a-f]{7,40}\b`)

func (a *App) componentRemoteVersion(component string, release githubRelease) string {
	component = normalizeComponent(component)
	mihomoCoreType, _ := a.componentDownloadOptions()
	switch component {
	case "mosdns":
		if v := releaseBodyFieldVersion(release.Body, "版本号"); v != "" {
			return v
		}
		if v := releaseBodyFieldVersion(release.Body, "version"); v != "" {
			return v
		}
	case "mihomo":
		if v := mihomoReleaseBodyVersion(release.Body, mihomoCoreType); v != "" {
			return v
		}
	case "zashboard":
		if v := strings.TrimSpace(release.TagName); v != "" {
			return v
		}
	}
	if v := firstVersionToken(release.Body); v != "" {
		return v
	}
	return firstNonEmpty(strings.TrimSpace(release.TagName), strings.TrimSpace(release.Name), "-")
}

func releaseBodyFieldVersion(body, field string) string {
	field = strings.ToLower(strings.TrimSpace(field))
	for _, line := range strings.Split(body, "\n") {
		lower := strings.ToLower(strings.TrimSpace(line))
		if !strings.Contains(lower, field) {
			continue
		}
		for _, sep := range []string{":", "："} {
			if idx := strings.Index(line, sep); idx >= 0 {
				if v := cleanReleaseVersionCandidate(line[idx+len(sep):]); v != "" {
					return v
				}
			}
		}
		if v := firstVersionToken(line); v != "" {
			return v
		}
	}
	return ""
}

func mihomoReleaseBodyVersion(body, coreType string) string {
	coreType = normalizeMihomoCoreType(coreType)
	var fallback string
	for _, line := range strings.Split(body, "\n") {
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "mihomo") {
			continue
		}
		version := firstVersionToken(line)
		if version == "" {
			continue
		}
		if strings.Contains(lower, coreType) {
			return version
		}
		if fallback == "" {
			fallback = version
		}
	}
	return fallback
}

func cleanReleaseVersionCandidate(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, " \t\r\n,，。;；:：`'\"[]()（）")
	if value == "" {
		return ""
	}
	return firstVersionToken(value)
}

func firstVersionToken(value string) string {
	match := componentVersionTokenRE.FindString(strings.TrimSpace(value))
	if match == "" {
		return ""
	}
	return strings.Trim(match, " \t\r\n,，。;；:：`'\"[]()（）")
}

func compactVersionOutput(value string) string {
	lines := strings.FieldsFunc(value, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, " ")
}

func componentHasUpdate(current, latest string) bool {
	current = strings.TrimSpace(current)
	latest = strings.TrimSpace(latest)
	if current == "" || isInvalidComponentVersion(current) || isPlaceholderComponentVersion(latest) {
		return false
	}
	if current == "not-installed" {
		return true
	}
	if current == "unknown" || current == "installed" || isSyntheticInstalledVersion(current) {
		return false
	}
	return !componentVersionsEquivalent(current, latest)
}

func componentCanUpdate(current, latest string, hasUpdate bool, downloadURL string) bool {
	if strings.TrimSpace(downloadURL) == "" {
		return false
	}
	if hasUpdate {
		return true
	}
	current = strings.TrimSpace(strings.ToLower(current))
	if current == "not-installed" {
		return true
	}
	return componentVersionUncertain(current, latest)
}

func componentVersionUncertain(current, latest string) bool {
	current = strings.TrimSpace(strings.ToLower(current))
	if current == "" || isInvalidComponentVersion(current) || current == "unknown" || current == "installed" || isSyntheticInstalledVersion(current) {
		return true
	}
	return isPlaceholderComponentVersion(latest)
}

func componentStateString(state map[string]any, key string) string {
	value, ok := state[key]
	if !ok || value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		if isInvalidComponentVersion(s) {
			return ""
		}
		return strings.TrimSpace(s)
	}
	out := strings.TrimSpace(fmt.Sprint(value))
	if isInvalidComponentVersion(out) {
		return ""
	}
	return out
}

func componentStateBool(state map[string]any, key string) bool {
	value, ok := state[key]
	if !ok || value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	case string:
		v = strings.TrimSpace(strings.ToLower(v))
		return v == "true" || v == "1" || v == "yes"
	default:
		return false
	}
}

func preservedComponentVerification(state map[string]any, downloadDigest string, hasUpdate bool) (string, bool) {
	downloadDigest = strings.TrimSpace(downloadDigest)
	if hasUpdate || downloadDigest == "" || !componentStateBool(state, "verified") {
		return "", false
	}
	if componentStateString(state, "verification_source") != componentVerificationSourceGitHubAssetDigest {
		return "", false
	}
	if componentStateString(state, "download_digest") != downloadDigest {
		return "", false
	}
	verifiedDigest := componentStateString(state, "verified_digest")
	if verifiedDigest == "" {
		verifiedDigest = downloadDigest
	}
	if verifiedDigest != downloadDigest {
		return "", false
	}
	return verifiedDigest, true
}

func componentDisplayCurrentVersion(component, current, latest string) (string, string) {
	current = strings.TrimSpace(current)
	if current == "" || isInvalidComponentVersion(current) {
		return "-", ""
	}
	switch normalizeComponent(component) {
	case "mosdns":
		if !isPlaceholderComponentVersion(latest) && componentVersionsEquivalent(current, latest) && current != latest {
			return latest, current
		}
	case "mihomo":
		if version := firstVersionToken(current); version != "" && version != current {
			return version, current
		}
	}
	return current, ""
}

func componentVersionsEquivalent(current, latest string) bool {
	currentNorm := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(current)), "v")
	latestNorm := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(latest)), "v")
	if currentNorm == "" || latestNorm == "" {
		return false
	}
	if currentNorm == latestNorm || strings.Contains(currentNorm, latestNorm) || strings.Contains(latestNorm, currentNorm) {
		return true
	}
	currentCommit := lastComponentCommitToken(currentNorm)
	latestCommit := lastComponentCommitToken(latestNorm)
	return currentCommit != "" && latestCommit != "" && currentCommit == latestCommit
}

func lastComponentCommitToken(value string) string {
	matches := componentCommitTokenRE.FindAllString(value, -1)
	if len(matches) == 0 {
		return ""
	}
	return strings.ToLower(matches[len(matches)-1])
}

func shouldUseLiveComponentVersion(component, live, stored string) bool {
	live = strings.TrimSpace(live)
	stored = strings.TrimSpace(stored)
	if live == "" || live == "unknown" || live == "not-installed" {
		return stored == "" || isInvalidComponentVersion(stored) || isSyntheticInstalledVersion(stored)
	}
	if isInvalidComponentVersion(stored) || isSyntheticInstalledVersion(stored) || stored == "" || stored == "latest" {
		return true
	}
	if component == "mosdns" || component == "mihomo" {
		return live != "installed"
	}
	return false
}

func isSyntheticInstalledVersion(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "installed" || strings.HasPrefix(value, "installed-") {
		return true
	}
	return false
}

func isPlaceholderComponentVersion(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return value == "" || value == "-" || value == "latest"
}

func isInvalidComponentVersion(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return value == "<nil>" || value == "nil" || value == "<null>" || value == "null"
}

func (a *App) componentUpdateConfig(component string) map[string]any {
	out := map[string]any{"component": component, "auto_check": true, "check_interval": defaultUpdateCheckInterval, "auto_update": false}
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

func (a *App) updateConfig() map[string]any {
	return map[string]any{
		"auto_check":           a.boolSetting(updateConfigAutoCheckKey, true),
		"auto_update":          a.boolSetting(updateConfigAutoUpdateKey, false),
		"check_interval":       a.intSetting(updateConfigCheckIntervalKey, defaultUpdateCheckInterval),
		"notify":               a.boolSetting(updateConfigNotifyKey, true),
		"mosdns_upgrade_mode":  a.modeSetting(updateConfigMosDNSUpgradeModeKey, "full", "full", "incremental", "reset"),
		"mihomo_upgrade_mode":  a.modeSetting(updateConfigMihomoUpgradeModeKey, "skip", "skip", "full"),
		"check_interval_label": updateIntervalLabel(a.intSetting(updateConfigCheckIntervalKey, defaultUpdateCheckInterval)),
	}
}

func (a *App) saveUpdateConfig(cfg map[string]any) {
	a.setSetting(updateConfigAutoCheckKey, strconv.FormatBool(updateBoolMapValue(cfg, "auto_check", true)))
	a.setSetting(updateConfigAutoUpdateKey, strconv.FormatBool(updateBoolMapValue(cfg, "auto_update", false)))
	a.setSetting(updateConfigNotifyKey, strconv.FormatBool(updateBoolMapValue(cfg, "notify", true)))
	a.setSetting(updateConfigCheckIntervalKey, strconv.Itoa(updateIntMapValue(cfg, "check_interval", defaultUpdateCheckInterval)))
	a.setSetting(updateConfigMosDNSUpgradeModeKey, updateStringMapValue(cfg, "mosdns_upgrade_mode", "full"))
	a.setSetting(updateConfigMihomoUpgradeModeKey, updateStringMapValue(cfg, "mihomo_upgrade_mode", "skip"))
}

func (a *App) boolSetting(key string, fallback bool) bool {
	value := strings.TrimSpace(a.setting(key, strconv.FormatBool(fallback)))
	if value == "" {
		return fallback
	}
	return isTruthy(value)
}

func (a *App) intSetting(key string, fallback int) int {
	value := strings.TrimSpace(a.setting(key, strconv.Itoa(fallback)))
	out, err := strconv.Atoi(value)
	if err != nil || out <= 0 {
		return fallback
	}
	return out
}

func (a *App) modeSetting(key, fallback string, allowed ...string) string {
	value := strings.ToLower(strings.TrimSpace(a.setting(key, fallback)))
	if oneOf(value, allowed...) {
		return value
	}
	return fallback
}

func updateBoolMapValue(values map[string]any, key string, fallback bool) bool {
	value, ok := values[key]
	if !ok {
		return fallback
	}
	parsed, err := structuredBoolValue(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func updateIntMapValue(values map[string]any, key string, fallback int) int {
	value, ok := values[key]
	if !ok {
		return fallback
	}
	parsed, err := structuredPositiveInt(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func updateStringMapValue(values map[string]any, key, fallback string) string {
	value, ok := values[key]
	if !ok {
		return fallback
	}
	out := strings.TrimSpace(fmtAny(value))
	if out == "" {
		return fallback
	}
	return out
}

func updateIntervalLabel(seconds int) string {
	switch seconds {
	case 12 * 60 * 60:
		return "12 小时"
	case 24 * 60 * 60:
		return "24 小时"
	case 3 * 24 * 60 * 60:
		return "3 天"
	case 7 * 24 * 60 * 60:
		return "7 天"
	default:
		return fmt.Sprintf("%d 秒", seconds)
	}
}

func (a *App) componentCurrentVersion(component string) string {
	component = normalizeComponent(component)
	target := a.componentTarget(component)
	if target == "" {
		return "unknown"
	}
	if _, err := os.Stat(target); err == nil {
		if version := componentBinaryVersion(component, target); version != "" {
			return version
		}
		return "installed"
	}
	if component == "mihomo" {
		legacyTarget := filepath.Join(a.DataDir, "data/binaries/mihomo/latest/mihomo")
		if _, err := os.Stat(legacyTarget); err == nil {
			if version := componentBinaryVersion(component, legacyTarget); version != "" {
				return version
			}
			return "installed"
		}
	}
	return "not-installed"
}

func componentBinaryVersion(component, target string) string {
	switch component {
	case "mosdns":
		if out, err := exec.Command(target, "version").CombinedOutput(); err == nil {
			return compactVersionOutput(string(out))
		}
	case "mihomo":
		if out, err := exec.Command(target, "-v").CombinedOutput(); err == nil {
			return compactVersionOutput(string(out))
		}
	}
	return ""
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

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest"`
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	HTMLURL     string        `json:"html_url"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
	} `json:"commit"`
}

func (a *App) fetchLatestRelease(owner, repo string) (githubRelease, error) {
	var release githubRelease
	err := a.fetchGitHubJSON(fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo), &release)
	return release, err
}

func (a *App) fetchReleaseByTag(owner, repo, tag string) (githubRelease, error) {
	var release githubRelease
	err := a.fetchGitHubJSON(fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, repo, tag), &release)
	return release, err
}

func (a *App) fetchReleases(owner, repo string) ([]githubRelease, error) {
	var releases []githubRelease
	err := a.fetchGitHubJSON(fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=20", owner, repo), &releases)
	return releases, err
}

func (a *App) fetchGitHubCommit(owner, repo, ref string) (githubCommit, error) {
	var commit githubCommit
	err := a.fetchGitHubJSON(fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", owner, repo, ref), &commit)
	return commit, err
}

func (a *App) fetchGitHubJSON(rawURL string, dst any) error {
	req, err := http.NewRequest(http.MethodGet, a.rewriteDownloadURL(rawURL), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "msf/"+a.Version)
	resp, err := a.downloadHTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("github api %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

func releaseAssetURL(release githubRelease, contains, suffix string) string {
	contains = strings.ToLower(contains)
	suffix = strings.ToLower(suffix)
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if contains != "" && !strings.Contains(name, contains) {
			continue
		}
		if suffix != "" && !strings.HasSuffix(name, suffix) {
			continue
		}
		return asset.BrowserDownloadURL
	}
	if len(release.Assets) > 0 {
		return release.Assets[0].BrowserDownloadURL
	}
	return ""
}

func selfUpdateAssetContainsFor(goos, goarch string) string {
	if goos == "" {
		goos = "linux"
	}
	switch goarch {
	case "amd64", "arm64":
		return goos + "-" + goarch
	default:
		return goos + "-" + goarch
	}
}

func (a *App) componentReleaseAssetURL(release githubRelease, component string) string {
	asset, ok := a.componentReleaseAsset(release, component)
	if !ok {
		return ""
	}
	return asset.BrowserDownloadURL
}

func (a *App) componentReleaseAsset(release githubRelease, component string) (githubAsset, bool) {
	want := downloadAssetName(a.componentDownloadURL(component))
	if want == "" {
		return githubAsset{}, false
	}
	for _, asset := range release.Assets {
		if strings.EqualFold(asset.Name, want) {
			return asset, true
		}
	}
	return githubAsset{}, false
}

func downloadAssetName(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	if i := strings.IndexAny(rawURL, "?#"); i >= 0 {
		rawURL = rawURL[:i]
	}
	rawURL = strings.TrimRight(rawURL, "/")
	if rawURL == "" {
		return ""
	}
	if i := strings.LastIndex(rawURL, "/"); i >= 0 {
		return rawURL[i+1:]
	}
	return rawURL
}

func versionDifferent(current, latest string) bool {
	current = strings.TrimPrefix(strings.TrimSpace(strings.ToLower(current)), "v")
	latest = strings.TrimPrefix(strings.TrimSpace(strings.ToLower(latest)), "v")
	return current != "" && latest != "" && current != latest && !strings.Contains(current, latest) && !strings.Contains(current, "dev")
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
