package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	mux.HandleFunc("GET /api/v1/component-updates/{component}/config", a.handleComponentUpdateConfig)
	mux.HandleFunc("PUT /api/v1/component-updates/{component}/config", a.handleComponentUpdateConfigPut)
}

func (a *App) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	item := a.selfUpdateState()
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": item})
}

func (a *App) selfUpdateState() map[string]any {
	item := map[string]any{
		"component":       "msm-free",
		"current_version": a.Version,
		"latest_version":  a.Version,
		"has_update":      false,
		"status":          "idle",
		"progress":        0,
	}
	row := a.DB.QueryRow(`select current_version,latest_version,has_update,status,progress,coalesce(error_message,''),coalesce(download_url,''),coalesce(release_notes,''),last_check_time from update_info where component='msm-free' order by id desc limit 1`)
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
	}
	return item
}

func (a *App) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	release, err := a.fetchLatestRelease("scoltzero", "msm-free")
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": a.selfUpdateState()})
		return
	}
	downloadURL := releaseAssetURL(release, selfUpdateAssetContainsFor(runtime.GOOS, runtime.GOARCH), ".tar.gz")
	hasUpdate := versionDifferent(a.Version, release.TagName)
	now := time.Now()
	_, _ = a.DB.Exec(`insert into update_info(component,current_version,latest_version,has_update,status,progress,error_message,download_url,release_notes,last_check_time,created_at,updated_at)
		values('msm-free',?,?,?,?,?,?,?,?,?,?,?)
		on conflict(id) do nothing`,
		a.Version, release.TagName, hasUpdate, "checked", 0, "", downloadURL, release.Body, now, now, now)
	_, _ = a.DB.Exec(`update update_info set current_version=?,latest_version=?,has_update=?,status='checked',progress=0,error_message='',download_url=?,release_notes=?,last_check_time=?,updated_at=? where component='msm-free'`,
		a.Version, release.TagName, hasUpdate, downloadURL, release.Body, now, now)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"component":       "msm-free",
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
	releases, err := a.fetchReleases("scoltzero", "msm-free")
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": []any{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": releases})
}

func (a *App) handleUpdateDownload(w http.ResponseWriter, r *http.Request) {
	state := a.selfUpdateState()
	rawURL := strings.TrimSpace(fmt.Sprint(state["download_url"]))
	if rawURL == "" {
		release, err := a.fetchLatestRelease("scoltzero", "msm-free")
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
	last := DownloadEvent{Status: "running", Progress: 5, Message: "starting"}
	err := a.downloadFile(rawURL, dest, func(ev DownloadEvent) {
		last = ev
		_, _ = a.DB.Exec(`update update_info set status='downloading',progress=?,error_message='',updated_at=? where component='msm-free'`, ev.Progress, nowString())
	})
	if err != nil {
		_, _ = a.DB.Exec(`update update_info set status='failed',progress=?,error_message=?,updated_at=? where component='msm-free'`, last.Progress, err.Error(), nowString())
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": a.selfUpdateState()})
		return
	}
	_, _ = a.DB.Exec(`update update_info set status='downloaded',progress=100,error_message='',download_url=?,updated_at=? where component='msm-free'`, rawURL, nowString())
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"path": dest, "event": last}})
}

func (a *App) handleUpdateInstall(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "message": "更新包已下载后可由安装脚本或 Unraid 插件覆盖当前二进制；运行中的进程不做热替换。", "data": a.selfUpdateState()})
}

func (a *App) handleUpdateCancel(w http.ResponseWriter, r *http.Request) {
	_, _ = a.DB.Exec(`update update_info set status='idle',progress=0,updated_at=? where component='msm-free'`, nowString())
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
	if remote, err := a.componentRemoteInfo(component); err == nil {
		state["latest_version"] = remote.TagName
		state["download_url"] = firstNonEmpty(a.componentReleaseAssetURL(remote, component), a.componentDownloadURL(component))
		state["release_body"] = remote.Body
		state["has_update"] = versionDifferent(fmt.Sprint(state["current_version"]), remote.TagName)
	}
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
		component, a.componentCurrentVersion(component), "latest", true, a.componentDownloadURL(component), "running", 5, now, now)
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
		"component":       component,
		"current_version": current,
		"latest_version":  "latest",
		"has_update":      !installed,
		"download_url":    a.componentDownloadURL(component),
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

func (a *App) componentRemoteInfo(component string) (githubRelease, error) {
	switch normalizeComponent(component) {
	case "mosdns":
		return a.fetchReleaseByTag("baozaodetudou", "mssb", "mosdns")
	case "mihomo":
		return a.fetchReleaseByTag("baozaodetudou", "mssb", "mihomo")
	case "zashboard":
		commit, err := a.fetchGitHubCommit("Zephyruso", "zashboard", "gh-pages")
		if err != nil {
			return githubRelease{}, err
		}
		return githubRelease{TagName: "gh-pages-" + shortSHA(commit.SHA), Name: "zashboard gh-pages", Body: commit.Commit.Message, Assets: []githubAsset{{Name: "gh-pages.zip", BrowserDownloadURL: a.componentDownloadURL("zashboard")}}}, nil
	default:
		return githubRelease{}, fmt.Errorf("unknown component %s", component)
	}
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

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
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
	req.Header.Set("User-Agent", "msm-free/"+a.Version)
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
	want := downloadAssetName(a.componentDownloadURL(component))
	if want == "" {
		return ""
	}
	for _, asset := range release.Assets {
		if strings.EqualFold(asset.Name, want) {
			return asset.BrowserDownloadURL
		}
	}
	return ""
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
	return current != "" && latest != "" && current != latest && !strings.Contains(current, "dev")
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
