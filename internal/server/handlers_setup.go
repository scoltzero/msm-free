package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func (a *App) handleSetupCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"success":        true,
		"is_initialized": a.IsInitialized(),
		"needs_recovery": false,
	})
}

func (a *App) handleSetupSystemInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"system": map[string]any{
			"os":        runtime.GOOS,
			"arch":      runtime.GOARCH,
			"hostname":  hostname(),
			"local_ips": localIPs(),
		},
		"cpu": map[string]any{
			"model":           cpuModel(),
			"cores":           runtime.NumCPU(),
			"supportsAMD64v3": supportsAMD64v3(),
			"amd64v3_status":  amd64v3Status(),
		},
	})
}

func (a *App) handleSetupNetworkInterfaces(w http.ResponseWriter, r *http.Request) {
	ifaces, _ := net.Interfaces()
	var out []map[string]any
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		var ips []string
		for _, addr := range addrs {
			ips = append(ips, addr.String())
		}
		ip := primaryInterfaceIP(ips)
		out = append(out, map[string]any{
			"name":        iface.Name,
			"index":       iface.Index,
			"mac":         iface.HardwareAddr.String(),
			"flags":       iface.Flags.String(),
			"is_up":       iface.Flags&net.FlagUp != 0,
			"is_loopback": iface.Flags&net.FlagLoopback != 0,
			"addresses":   ips,
			"ip":          ip,
			"primary_ip":  ip,
			"speed":       interfaceSpeed(iface.Name),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "interfaces": out, "data": out})
}

func (a *App) handleSetupPrivilege(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"is_root": os.Geteuid() == 0,
		"uid":     os.Geteuid(),
		"message": "MosDNS 53 port and nftables require root on Linux",
	})
}

func (a *App) handleSetupGetConfig(w http.ResponseWriter, r *http.Request) {
	row := a.DB.QueryRow(`select username,email,web_port,amd64v3_enabled,selected_interface,mihomo_core_type,auto_set_dns,dns_on,dns_off,enable_ipv6,fake_ip_range_v4,fake_ip_range_v6,linux_proxy_mode,nft_proxy_policy,proxy_core,mos_dns_enabled,subscription_urls,mihomo_proxies,is_initialized from system_setups order by id desc limit 1`)
	var cfg SetupConfig
	var initialized bool
	err := row.Scan(&cfg.Username, &cfg.Email, &cfg.WebPort, &cfg.AMD64v3Enabled, &cfg.SelectedInterface, &cfg.MihomoCoreType, &cfg.AutoSetDNS, &cfg.DNSOn, &cfg.DNSOff, &cfg.EnableIPv6, &cfg.FakeIPRangeV4, &cfg.FakeIPRangeV6, &cfg.LinuxProxyMode, &cfg.NFTProxyPolicy, &cfg.ProxyCore, &cfg.MosDNSEnabled, &cfg.SubscriptionURLs, &cfg.MihomoProxies, &initialized)
	if err != nil {
		cfg.defaults()
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "config": cfg, "is_initialized": initialized})
}

func (a *App) handleSetupPutConfig(w http.ResponseWriter, r *http.Request) {
	var cfg SetupConfig
	if err := decodeJSON(r, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	cfg.defaults()
	if cfg.Username == "" {
		cfg.Username = "root"
	}
	now := time.Now()
	_, err := a.DB.Exec(`insert into system_setups(created_at,updated_at,username,email,web_port,amd64v3_enabled,selected_interface,mihomo_core_type,auto_set_dns,dns_on,dns_off,enable_ipv6,fake_ip_range_v4,fake_ip_range_v6,linux_proxy_mode,nft_proxy_policy,proxy_core,mos_dns_enabled,subscription_urls,mihomo_proxies,github_proxy_enabled,github_https_proxy,github_http_proxy,github_socks5_proxy,github_accelerator_enabled,github_accelerator_url,is_initialized)
		values(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,true)`,
		now, now, cfg.Username, cfg.Email, cfg.WebPort, cfg.AMD64v3Enabled, cfg.SelectedInterface, cfg.MihomoCoreType, cfg.AutoSetDNS, cfg.DNSOn, cfg.DNSOff, cfg.EnableIPv6, cfg.FakeIPRangeV4, cfg.FakeIPRangeV6, cfg.LinuxProxyMode, cfg.NFTProxyPolicy, "mihomo", true, cfg.SubscriptionURLs, cfg.MihomoProxies, cfg.GitHubProxyEnabled, cfg.GitHubHTTPSProxy, cfg.GitHubHTTPProxy, cfg.GitHubSocks5Proxy, cfg.GitHubAcceleratorEnabled, cfg.GitHubAcceleratorURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "setup_error", err.Error())
		return
	}
	a.SetConfiguredRuntimeDesired(cfg)
	if err := a.writeGeneratedConfigs(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "config_error", err.Error())
		return
	}
	restarted := []string{}
	for _, name := range managedServiceNames() {
		st := a.Services.Status(name)
		if st.Running {
			if _, err := a.Services.Restart(r.Context(), name); err == nil {
				restarted = append(restarted, name)
			}
		}
	}
	missing := []string{}
	for _, name := range managedServiceNames() {
		if !a.Services.Status(name).Installed {
			missing = append(missing, name)
		}
	}
	networkReapply := shouldRestoreNFT(cfg)
	writeJSON(w, http.StatusOK, map[string]any{
		"success":                  true,
		"config":                   cfg,
		"restarted_services":       restarted,
		"needs_download":           len(missing) > 0,
		"download_component":       missing,
		"network_reapply_required": networkReapply,
	})
}

func (a *App) handleSetupInitialize(w http.ResponseWriter, r *http.Request) {
	var cfg SetupConfig
	if err := decodeJSON(r, &cfg); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	cfg.defaults()
	if cfg.Username == "" || cfg.Password == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "username and password are required")
		return
	}
	if err := a.EnsureBaseLayout(); err != nil {
		writeError(w, http.StatusInternalServerError, "layout_error", err.Error())
		return
	}
	if err := a.createOrUpdateAdmin(cfg.Username, cfg.Password, cfg.Email); err != nil {
		writeError(w, http.StatusInternalServerError, "user_error", err.Error())
		return
	}
	if err := a.writeGeneratedConfigs(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "config_error", err.Error())
		return
	}
	now := time.Now()
	_, err := a.DB.Exec(`insert into system_setups(created_at,updated_at,username,email,web_port,amd64v3_enabled,selected_interface,mihomo_core_type,auto_set_dns,dns_on,dns_off,enable_ipv6,fake_ip_range_v4,fake_ip_range_v6,linux_proxy_mode,nft_proxy_policy,proxy_core,mos_dns_enabled,subscription_urls,mihomo_proxies,github_proxy_enabled,github_https_proxy,github_http_proxy,github_socks5_proxy,github_accelerator_enabled,github_accelerator_url,is_initialized)
		values(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,true)`,
		now, now, cfg.Username, cfg.Email, cfg.WebPort, cfg.AMD64v3Enabled, cfg.SelectedInterface, cfg.MihomoCoreType, cfg.AutoSetDNS, cfg.DNSOn, cfg.DNSOff, cfg.EnableIPv6, cfg.FakeIPRangeV4, cfg.FakeIPRangeV6, cfg.LinuxProxyMode, cfg.NFTProxyPolicy, "mihomo", true, cfg.SubscriptionURLs, cfg.MihomoProxies, cfg.GitHubProxyEnabled, cfg.GitHubHTTPSProxy, cfg.GitHubHTTPProxy, cfg.GitHubSocks5Proxy, cfg.GitHubAcceleratorEnabled, cfg.GitHubAcceleratorURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "setup_error", err.Error())
		return
	}
	a.SetConfiguredRuntimeDesired(cfg)
	a.audit(nil, "setup.initialize", "system", cfg.Username, true, "")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "message": "initialized"})
}

func (a *App) handleSetupActivate(w http.ResponseWriter, r *http.Request) {
	report := RuntimeRestoreReport{Initialized: a.IsInitialized()}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		restored := a.RestoreConfiguredRuntime(ctx)
		if len(restored.Errors) > 0 {
			log.Printf("setup activation completed with errors: %s", strings.Join(restored.Errors, "; "))
		}
	}()
	writeJSON(w, http.StatusOK, map[string]any{
		"success":            true,
		"port_changed":       false,
		"port":               7777,
		"activation_pending": true,
		"runtime":            report,
	})
}

func (a *App) handleSetupReset(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(r) {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	var req struct {
		DeleteBinaries   bool `json:"delete_binaries"`
		DeleteComponents bool `json:"delete_components"`
	}
	if err := decodeJSON(r, &req); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	_ = a.Services.StopAll(r.Context())
	_, _ = a.DB.Exec(`delete from system_setups`)
	a.Services.setDesired("mihomo", false)
	a.Services.setDesired("mosdns", false)
	a.setSetting(nftDesiredKey, "false")
	deletedBinaries := false
	if req.DeleteBinaries || req.DeleteComponents {
		_ = os.RemoveAll(filepath.Join(a.DataDir, "data/binaries/mihomo"))
		_ = os.RemoveAll(filepath.Join(a.DataDir, "data/binaries/mosdns"))
		deletedBinaries = true
	}
	a.audit(currentUser(r), "setup.reset", "system", fmt.Sprintf("delete_binaries=%t", deletedBinaries), true, "")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "deleted_binaries": deletedBinaries})
}

func (a *App) handleSetupDownload(w http.ResponseWriter, r *http.Request) {
	component := r.PathValue("component")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	enc := json.NewEncoder(w)
	flusher, _ := w.(http.Flusher)
	emit := func(ev DownloadEvent) {
		fmt.Fprint(w, "data: ")
		_ = enc.Encode(ev)
		fmt.Fprint(w, "\n")
		if flusher != nil {
			flusher.Flush()
		}
	}
	if isTruthy(r.URL.Query().Get("skip_if_exists")) {
		if target := a.componentTarget(component); target != "" {
			if _, err := os.Stat(target); err == nil {
				emit(DownloadEvent{Status: "skipped", Progress: 100, Message: component + " already installed"})
				return
			}
		}
	}
	err := a.installComponent(component, emit)
	if err != nil {
		emit(DownloadEvent{Status: "failed", Progress: 0, Message: err.Error()})
	}
}

func isTruthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func primaryInterfaceIP(addresses []string) string {
	for _, addr := range addresses {
		host := addr
		if strings.Contains(addr, "/") {
			if ip, _, err := net.ParseCIDR(addr); err == nil {
				host = ip.String()
			}
		}
		ip := net.ParseIP(host)
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		if ip.To4() != nil {
			return ip.String()
		}
	}
	for _, addr := range addresses {
		host := addr
		if strings.Contains(addr, "/") {
			if ip, _, err := net.ParseCIDR(addr); err == nil {
				host = ip.String()
			}
		}
		ip := net.ParseIP(host)
		if ip != nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
			return ip.String()
		}
	}
	return ""
}

func interfaceSpeed(name string) string {
	if runtime.GOOS != "linux" || name == "" {
		return "unknown"
	}
	b, err := os.ReadFile(filepath.Join("/sys/class/net", name, "speed"))
	if err != nil {
		return "unknown"
	}
	value := strings.TrimSpace(string(b))
	if value == "" || value == "-1" {
		return "unknown"
	}
	return value + " Mbps"
}

func hostname() string {
	h, _ := os.Hostname()
	return h
}

func cpuModel() string {
	if runtime.GOOS != "linux" {
		return runtime.GOARCH
	}
	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return runtime.GOARCH
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(strings.ToLower(line), "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return runtime.GOARCH
}

func supportsAMD64v3() bool {
	if runtime.GOARCH != "amd64" {
		return false
	}
	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return false
	}
	flags := strings.ToLower(string(b))
	required := []string{"avx", "avx2", "bmi1", "bmi2", "fma", "lzcnt", "movbe", "xsave"}
	for _, f := range required {
		if !strings.Contains(flags, f) {
			return false
		}
	}
	return true
}

func amd64v3Status() string {
	if runtime.GOARCH != "amd64" {
		return "unnecessary"
	}
	if supportsAMD64v3() {
		return "supported"
	}
	return "unsupported"
}
