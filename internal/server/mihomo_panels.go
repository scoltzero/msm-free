package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func (a *App) handleProxyOverview(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.proxySnapshot()})
}

func (a *App) proxySnapshot() map[string]any {
	mihomo := a.mihomoSnapshot()
	mosdns := a.Services.Status("mosdns")
	return map[string]any{
		"core":     "mihomo",
		"mihomo":   mihomo,
		"mosdns":   mosdns,
		"services": map[string]any{"mihomo": mihomo["service"], "mosdns": mosdns},
		"ports":    mihomo["ports"],
		"healthy":  mihomo["controller_available"],
		"mode":     mihomo["mode"],
	}
}

func (a *App) mihomoSnapshot() map[string]any {
	return a.mihomoOverviewSnapshot()
}

func (a *App) mihomoFullSnapshot() map[string]any {
	cfg := a.mihomoConfigMap()
	service := a.Services.Status("mihomo")
	controllerCfg := a.mihomoControllerConfig()
	traffic := a.mihomoTrafficPayload()
	connections := a.mihomoConnectionsPayload(nil)
	proxies := a.mihomoProxiesPayload(nil)
	rules := a.mihomoRulesRuntime(nil)
	providers := a.mihomoProvidersPayload()
	version := a.mihomoVersion()
	if raw, ok, _ := a.mihomoControllerJSON(http.MethodGet, "/version", nil); ok {
		if m, ok := raw.(map[string]any); ok {
			version = firstNonEmpty(stringMapValue(m, "version"), stringMapValue(m, "premium"), version)
		}
	}
	ports := mihomoPortsFromConfig(cfg)
	health := map[string]any{
		"controller": a.tcpPortOpen("127.0.0.1", ports["controller"]),
		"http":       a.tcpPortOpen("127.0.0.1", ports["http"]),
		"socks":      a.tcpPortOpen("127.0.0.1", ports["socks"]),
		"mixed":      a.tcpPortOpen("127.0.0.1", ports["mixed"]),
		"redir":      a.tcpPortOpen("127.0.0.1", ports["redir"]),
		"tproxy":     a.tcpPortOpen("127.0.0.1", ports["tproxy"]),
	}
	snapshot := map[string]any{
		"service":              service,
		"status":               service.Status,
		"running":              service.Running,
		"installed":            service.Installed,
		"pid":                  service.PID,
		"cpu":                  service.CPU,
		"memory":               service.Memory,
		"uptime":               service.Uptime,
		"version":              version,
		"mode":                 firstNonEmpty(stringMapValue(controllerCfg, "mode"), stringMapValue(cfg, "mode"), "rule"),
		"log_level":            firstNonEmpty(stringMapValue(controllerCfg, "log-level"), stringMapValue(cfg, "log-level"), "info"),
		"allow_lan":            boolMapValue(controllerCfg, "allow-lan", boolMapValue(cfg, "allow-lan", true)),
		"external_controller":  a.mihomoControllerBase(),
		"controller_available": controllerCfg != nil,
		"ui_url":               "/ui/",
		"ports":                ports,
		"health":               health,
		"traffic":              traffic,
		"connections":          connections,
		"connection_count":     connections["total"],
		"proxies":              proxies,
		"proxy_group_count":    len(anyMapSlice(proxies["groups"])),
		"proxy_count":          len(anyMapSlice(proxies["proxy_list"])),
		"rules":                rules,
		"rule_count":           rules["total"],
		"providers":            providers,
		"proxy_provider_count": len(anyMapSlice(providers["proxy_providers"])),
		"rule_provider_count":  len(anyMapSlice(providers["rule_providers"])),
		"config":               map[string]any{"path": "configs/mihomo/config.yaml", "active": a.setting("mihomo.active_config", "config.yaml")},
	}
	stats := mihomoStatsFromSnapshot(snapshot)
	snapshot["stats"] = stats
	snapshot["uploadSpeed"] = stats["uploadSpeed"]
	snapshot["downloadSpeed"] = stats["downloadSpeed"]
	snapshot["upload_speed"] = stats["upload_speed"]
	snapshot["download_speed"] = stats["download_speed"]
	snapshot["uploadTotal"] = stats["uploadTotal"]
	snapshot["downloadTotal"] = stats["downloadTotal"]
	snapshot["upload_total"] = stats["upload_total"]
	snapshot["download_total"] = stats["download_total"]
	snapshot["activeConnections"] = stats["activeConnections"]
	snapshot["active_connections"] = stats["active_connections"]
	return snapshot
}

func (a *App) mihomoOverviewSnapshot() map[string]any {
	cfg := a.mihomoConfigMap()
	service := a.Services.Status("mihomo")
	controllerCfg := a.mihomoControllerConfig()
	connections := a.mihomoConnectionsSummary()
	traffic := a.mihomoTrafficCachedPayload()
	version := a.mihomoVersion()
	if raw, ok, _ := a.mihomoControllerJSON(http.MethodGet, "/version", nil); ok {
		if m, ok := raw.(map[string]any); ok {
			version = firstNonEmpty(stringMapValue(m, "version"), stringMapValue(m, "premium"), version)
		}
	}
	ports := mihomoPortsFromConfig(cfg)
	health := map[string]any{
		"controller": a.tcpPortOpen("127.0.0.1", ports["controller"]),
		"http":       a.tcpPortOpen("127.0.0.1", ports["http"]),
		"socks":      a.tcpPortOpen("127.0.0.1", ports["socks"]),
		"mixed":      a.tcpPortOpen("127.0.0.1", ports["mixed"]),
		"redir":      a.tcpPortOpen("127.0.0.1", ports["redir"]),
		"tproxy":     a.tcpPortOpen("127.0.0.1", ports["tproxy"]),
	}
	counts := mihomoLocalConfigCounts(cfg)
	snapshot := map[string]any{
		"service":              service,
		"status":               service.Status,
		"running":              service.Running,
		"installed":            service.Installed,
		"pid":                  service.PID,
		"cpu":                  service.CPU,
		"memory":               service.Memory,
		"uptime":               service.Uptime,
		"version":              version,
		"mode":                 firstNonEmpty(stringMapValue(controllerCfg, "mode"), stringMapValue(cfg, "mode"), "rule"),
		"log_level":            firstNonEmpty(stringMapValue(controllerCfg, "log-level"), stringMapValue(cfg, "log-level"), "info"),
		"allow_lan":            boolMapValue(controllerCfg, "allow-lan", boolMapValue(cfg, "allow-lan", true)),
		"external_controller":  a.mihomoControllerBase(),
		"controller_available": controllerCfg != nil,
		"ui_url":               "/ui/",
		"zashboard_url":        "/ui/",
		"ports":                ports,
		"health":               health,
		"traffic":              traffic,
		"connections":          connections,
		"connection_count":     connections["total"],
		"proxy_group_count":    counts["proxy_group_count"],
		"proxy_count":          counts["proxy_count"],
		"rule_count":           counts["rule_count"],
		"proxy_provider_count": counts["proxy_provider_count"],
		"rule_provider_count":  counts["rule_provider_count"],
		"config":               map[string]any{"path": "configs/mihomo/config.yaml", "active": a.setting("mihomo.active_config", "config.yaml")},
		"lightweight":          true,
	}
	stats := mihomoStatsFromSnapshot(snapshot)
	snapshot["stats"] = stats
	snapshot["uploadSpeed"] = stats["uploadSpeed"]
	snapshot["downloadSpeed"] = stats["downloadSpeed"]
	snapshot["upload_speed"] = stats["upload_speed"]
	snapshot["download_speed"] = stats["download_speed"]
	snapshot["uploadTotal"] = stats["uploadTotal"]
	snapshot["downloadTotal"] = stats["downloadTotal"]
	snapshot["upload_total"] = stats["upload_total"]
	snapshot["download_total"] = stats["download_total"]
	snapshot["activeConnections"] = stats["activeConnections"]
	snapshot["active_connections"] = stats["active_connections"]
	return snapshot
}

func mihomoLocalConfigCounts(cfg map[string]any) map[string]any {
	return map[string]any{
		"proxy_group_count":    anyLen(cfg["proxy-groups"]),
		"proxy_count":          anyLen(cfg["proxies"]),
		"rule_count":           anyLen(cfg["rules"]),
		"proxy_provider_count": anyLen(cfg["proxy-providers"]),
		"rule_provider_count":  anyLen(cfg["rule-providers"]),
	}
}

func mihomoStatsFromSnapshot(snapshot map[string]any) map[string]any {
	traffic, _ := snapshot["traffic"].(map[string]any)
	connections, _ := snapshot["connections"].(map[string]any)
	connectionItems := anyMapSlice(connections["connections"])
	activeConnections := intAny(connections["active_count"], len(connectionItems))
	if activeConnections == 0 && len(connectionItems) > 0 {
		activeConnections = len(connectionItems)
	}
	downloadSpeed := numericMapValue(traffic, "down")
	if downloadSpeed == 0 {
		downloadSpeed = numericMapValue(traffic, "download")
	}
	uploadSpeed := numericMapValue(traffic, "up")
	if uploadSpeed == 0 {
		uploadSpeed = numericMapValue(traffic, "upload")
	}
	downloadTotal := numericMapValue(connections, "downloadTotal")
	if downloadTotal == 0 {
		downloadTotal = numericMapValue(connections, "download_total")
	}
	uploadTotal := numericMapValue(connections, "uploadTotal")
	if uploadTotal == 0 {
		uploadTotal = numericMapValue(connections, "upload_total")
	}
	proxyProviderCount := intAny(snapshot["proxy_provider_count"], 0)
	ruleProviderCount := intAny(snapshot["rule_provider_count"], 0)
	ruleCount := intAny(snapshot["rule_count"], 0)
	proxyGroupCount := intAny(snapshot["proxy_group_count"], 0)
	proxyCount := intAny(snapshot["proxy_count"], 0)
	stats := map[string]any{
		"status":               snapshot["status"],
		"running":              snapshot["running"],
		"version":              snapshot["version"],
		"pid":                  snapshot["pid"],
		"cpu":                  numericAny(snapshot["cpu"]),
		"cpu_percent":          numericAny(snapshot["cpu"]),
		"memory":               numericAny(snapshot["memory"]),
		"memory_bytes":         numericAny(snapshot["memory"]),
		"uptime":               snapshot["uptime"],
		"traffic":              traffic,
		"connections":          connections,
		"connection_count":     activeConnections,
		"connections_count":    activeConnections,
		"activeConnections":    activeConnections,
		"active_connections":   activeConnections,
		"downloadSpeed":        downloadSpeed,
		"download_speed":       downloadSpeed,
		"down":                 downloadSpeed,
		"uploadSpeed":          uploadSpeed,
		"upload_speed":         uploadSpeed,
		"up":                   uploadSpeed,
		"downloadTotal":        downloadTotal,
		"download_total":       downloadTotal,
		"uploadTotal":          uploadTotal,
		"upload_total":         uploadTotal,
		"proxyProviderCount":   proxyProviderCount,
		"proxy_provider_count": proxyProviderCount,
		"ruleProviderCount":    ruleProviderCount,
		"rule_provider_count":  ruleProviderCount,
		"ruleCount":            ruleCount,
		"rule_count":           ruleCount,
		"proxyGroupCount":      proxyGroupCount,
		"proxy_group_count":    proxyGroupCount,
		"proxyCount":           proxyCount,
		"proxy_count":          proxyCount,
		"controller_available": snapshot["controller_available"],
		"health":               snapshot["health"],
		"ports":                snapshot["ports"],
	}
	return stats
}

func (a *App) mihomoControllerBase() string {
	base := strings.TrimRight(a.setting("mihomo_controller_endpoint", ""), "/")
	if base == "" {
		cfg := a.mihomoConfigMap()
		controller := firstNonEmpty(stringMapValue(cfg, "external-controller"), "127.0.0.1:9090")
		controller = strings.TrimSpace(controller)
		if strings.HasPrefix(controller, ":") {
			controller = "127.0.0.1" + controller
		}
		if strings.HasPrefix(controller, "0.0.0.0:") {
			controller = "127.0.0.1:" + strings.TrimPrefix(controller, "0.0.0.0:")
		}
		if !strings.Contains(controller, "://") {
			controller = "http://" + controller
		}
		base = strings.TrimRight(controller, "/")
	}
	if _, err := url.ParseRequestURI(base); err != nil {
		return "http://127.0.0.1:9090"
	}
	return base
}

func (a *App) mihomoControllerURL(path string) string {
	if path == "" {
		return a.mihomoControllerBase()
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return a.mihomoControllerBase() + path
}

func (a *App) mihomoSecret() string {
	if secret := a.setting("mihomo_controller_secret", ""); secret != "" {
		return secret
	}
	return stringMapValue(a.mihomoConfigMap(), "secret")
}

func (a *App) mihomoControllerJSON(method, path string, body []byte) (any, bool, error) {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	req, err := http.NewRequest(method, a.mihomoControllerURL(path), bytes.NewReader(body))
	if err != nil {
		return nil, false, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if secret := a.mihomoSecret(); secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, false, fmt.Errorf("mihomo controller http %d", resp.StatusCode)
	}
	if resp.Body == http.NoBody || resp.ContentLength == 0 {
		return map[string]any{"ok": true}, true, nil
	}
	var raw any
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&raw); err != nil {
		return map[string]any{"ok": true}, true, nil
	}
	return raw, true, nil
}

func (a *App) mihomoControllerMap(path string) (map[string]any, bool) {
	raw, ok, _ := a.mihomoControllerJSON(http.MethodGet, path, nil)
	if !ok {
		return nil, false
	}
	switch v := raw.(type) {
	case map[string]any:
		return v, true
	default:
		return map[string]any{"data": v}, true
	}
}

func (a *App) mihomoControllerConfig() map[string]any {
	if cfg, ok := a.mihomoControllerMap("/configs"); ok {
		return cfg
	}
	return nil
}

func (a *App) mihomoConfigMap() map[string]any {
	cfg := map[string]any{}
	_ = readYAMLFile(filepath.Join(a.DataDir, "configs/mihomo/config.yaml"), &cfg)
	return cfg
}

func mihomoPortsFromConfig(cfg map[string]any) map[string]int {
	ports := map[string]int{
		"http": 7890, "socks": 7891, "mixed": 7892, "redir": 7877, "tproxy": 7896, "dns": 6666, "controller": 9090,
	}
	ports["http"] = intMapValue(cfg, "port", ports["http"])
	ports["socks"] = intMapValue(cfg, "socks-port", ports["socks"])
	ports["mixed"] = intMapValue(cfg, "mixed-port", ports["mixed"])
	ports["redir"] = intMapValue(cfg, "redir-port", ports["redir"])
	ports["tproxy"] = intMapValue(cfg, "tproxy-port", ports["tproxy"])
	if dns, ok := cfg["dns"].(map[string]any); ok {
		ports["dns"] = portFromListen(firstNonEmpty(stringMapValue(dns, "listen"), "0.0.0.0:6666"), ports["dns"])
	}
	ports["controller"] = portFromListen(firstNonEmpty(stringMapValue(cfg, "external-controller"), "127.0.0.1:9090"), ports["controller"])
	return ports
}

const mihomoTrafficCacheTTL = 2 * time.Second

func (a *App) mihomoTrafficPayload() map[string]any {
	if cached, ok := a.cachedMihomoTraffic(); ok {
		return cached
	}
	payload := a.fetchMihomoTrafficPayload()
	a.storeMihomoTraffic(payload)
	return payload
}

func (a *App) mihomoTrafficCachedPayload() map[string]any {
	if cached, ok := a.cachedMihomoTraffic(); ok {
		return cached
	}
	go a.refreshMihomoTrafficCache()
	return zeroMihomoTrafficPayload()
}

func (a *App) refreshMihomoTrafficCache() {
	payload := a.fetchMihomoTrafficPayload()
	a.storeMihomoTraffic(payload)
}

func (a *App) cachedMihomoTraffic() (map[string]any, bool) {
	a.mihomoTrafficMu.Lock()
	defer a.mihomoTrafficMu.Unlock()
	if a.mihomoTrafficCache == nil || a.mihomoTrafficAt.IsZero() || time.Since(a.mihomoTrafficAt) > mihomoTrafficCacheTTL {
		return nil, false
	}
	return cloneAnyMap(a.mihomoTrafficCache), true
}

func (a *App) storeMihomoTraffic(payload map[string]any) {
	a.mihomoTrafficMu.Lock()
	defer a.mihomoTrafficMu.Unlock()
	a.mihomoTrafficCache = cloneAnyMap(payload)
	a.mihomoTrafficAt = time.Now()
}

func (a *App) fetchMihomoTrafficPayload() map[string]any {
	if raw, ok := a.mihomoControllerMap("/traffic"); ok {
		return map[string]any{
			"up":       numericMapValue(raw, "up"),
			"down":     numericMapValue(raw, "down"),
			"upload":   numericMapValue(raw, "up"),
			"download": numericMapValue(raw, "down"),
			"raw":      raw,
		}
	}
	return zeroMihomoTrafficPayload()
}

func zeroMihomoTrafficPayload() map[string]any {
	return map[string]any{"up": 0, "down": 0, "upload": 0, "download": 0, "raw": map[string]any{}}
}

func cloneAnyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (a *App) mihomoConnectionsSummary() map[string]any {
	raw, ok := a.mihomoControllerMap("/connections")
	if !ok {
		return map[string]any{
			"total": 0, "active_count": 0, "downloadTotal": 0, "uploadTotal": 0, "download_total": 0, "upload_total": 0,
		}
	}
	total := len(anySlice(raw["connections"]))
	return map[string]any{
		"total":          total,
		"active_count":   total,
		"downloadTotal":  numericMapValue(raw, "downloadTotal"),
		"uploadTotal":    numericMapValue(raw, "uploadTotal"),
		"download_total": numericMapValue(raw, "downloadTotal"),
		"upload_total":   numericMapValue(raw, "uploadTotal"),
	}
}

func (a *App) mihomoConnectionsPayload(r *http.Request) map[string]any {
	raw, ok := a.mihomoControllerMap("/connections")
	if !ok {
		raw = map[string]any{"connections": []any{}, "downloadTotal": 0, "uploadTotal": 0}
	}
	connections := normalizeMihomoConnectionList(anySlice(raw["connections"]))
	filtered := filterMihomoConnections(connections, r)
	page, limit := 1, len(filtered)
	if r != nil {
		page = queryInt(r, "page", 1)
		limit = queryInt(r, "page_size", queryInt(r, "limit", len(filtered)))
	}
	if limit <= 0 {
		limit = 100
	}
	total := len(filtered)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	items := filtered[start:end]
	return map[string]any{
		"connections":    items,
		"items":          items,
		"total":          total,
		"active_count":   len(connections),
		"downloadTotal":  numericMapValue(raw, "downloadTotal"),
		"uploadTotal":    numericMapValue(raw, "uploadTotal"),
		"download_total": numericMapValue(raw, "downloadTotal"),
		"upload_total":   numericMapValue(raw, "uploadTotal"),
		"pagination": map[string]any{
			"page": page, "limit": limit, "page_size": limit, "total": total, "total_pages": (total + limit - 1) / limit,
		},
		"raw": raw,
	}
}

func normalizeMihomoConnectionList(items []any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for i, item := range items {
		conn, ok := item.(map[string]any)
		if !ok {
			continue
		}
		metadata, _ := conn["metadata"].(map[string]any)
		chains := stringSlice(conn["chains"])
		host := firstNonEmpty(stringMapValue(metadata, "host"), stringMapValue(metadata, "destinationIP"), stringMapValue(metadata, "destinationPort"))
		id := firstNonEmpty(stringMapValue(conn, "id"), fmt.Sprintf("conn-%d", i+1))
		normalized := map[string]any{
			"id":               id,
			"host":             host,
			"network":          strings.ToLower(firstNonEmpty(stringMapValue(metadata, "network"), stringMapValue(metadata, "netWork"))),
			"type":             stringMapValue(metadata, "type"),
			"inbound":          stringMapValue(metadata, "type"),
			"source_ip":        stringMapValue(metadata, "sourceIP"),
			"source_port":      stringMapValue(metadata, "sourcePort"),
			"destination_ip":   stringMapValue(metadata, "destinationIP"),
			"destination_port": stringMapValue(metadata, "destinationPort"),
			"process":          firstNonEmpty(stringMapValue(metadata, "process"), stringMapValue(metadata, "processPath")),
			"rule":             stringMapValue(conn, "rule"),
			"rule_payload":     stringMapValue(conn, "rulePayload"),
			"chains":           chains,
			"chain":            strings.Join(chains, " / "),
			"download":         numericMapValue(conn, "download"),
			"upload":           numericMapValue(conn, "upload"),
			"start":            stringMapValue(conn, "start"),
			"metadata":         metadata,
			"raw":              conn,
		}
		out = append(out, normalized)
	}
	return out
}

func filterMihomoConnections(items []map[string]any, r *http.Request) []map[string]any {
	if r == nil {
		return items
	}
	q := r.URL.Query()
	search := strings.ToLower(strings.TrimSpace(firstNonEmpty(q.Get("search"), q.Get("q"), q.Get("keyword"), q.Get("host"))))
	network := strings.ToLower(strings.TrimSpace(firstNonEmpty(q.Get("network"), q.Get("protocol"))))
	inbound := strings.TrimSpace(q.Get("inbound"))
	rule := strings.TrimSpace(q.Get("rule"))
	chain := strings.TrimSpace(q.Get("chain"))
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if search != "" && !strings.Contains(strings.ToLower(strings.Join([]string{
			stringMapValue(item, "host"),
			stringMapValue(item, "source_ip"),
			stringMapValue(item, "destination_ip"),
			stringMapValue(item, "process"),
			stringMapValue(item, "rule"),
			stringMapValue(item, "chain"),
		}, " ")), search) {
			continue
		}
		if network != "" && network != "all" && stringMapValue(item, "network") != network {
			continue
		}
		if inbound != "" && inbound != "all" && stringMapValue(item, "inbound") != inbound {
			continue
		}
		if rule != "" && !strings.EqualFold(stringMapValue(item, "rule"), rule) {
			continue
		}
		if chain != "" && !strings.Contains(stringMapValue(item, "chain"), chain) {
			continue
		}
		filtered = append(filtered, item)
	}
	sortMihomoRows(filtered, q.Get("sort"), q.Get("sort_order"))
	return filtered
}

func (a *App) mihomoProxiesPayload(r *http.Request) map[string]any {
	rawProxies, ok := a.mihomoControllerMap("/proxies")
	if !ok {
		rawProxies = map[string]any{"proxies": map[string]any{}}
	}
	rawProviders, ok := a.mihomoControllerMap("/providers/proxies")
	if !ok {
		rawProviders = map[string]any{"providers": map[string]any{}}
	}
	proxyMap, groups, proxies := normalizeMihomoProxies(rawProxies)
	if r != nil {
		search := strings.ToLower(strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("search"), r.URL.Query().Get("q"))))
		if search != "" {
			proxies = filterMihomoProxyList(proxies, search)
			groups = filterMihomoProxyList(groups, search)
			proxyMap = filterMihomoProxyMap(proxyMap, search)
		}
	}
	return map[string]any{
		"groups":       groups,
		"proxy_groups": groups,
		"proxy_list":   proxies,
		"nodes":        proxies,
		"proxies":      proxyMap,
		"providers":    normalizeProviderMap(rawProviders["providers"]),
		"raw":          rawProxies,
	}
}

func normalizeMihomoProxies(raw map[string]any) (map[string]any, []map[string]any, []map[string]any) {
	proxyMap, _ := raw["proxies"].(map[string]any)
	byName := map[string]any{}
	var groups []map[string]any
	var proxies []map[string]any
	groupTypes := map[string]bool{"Selector": true, "URLTest": true, "Fallback": true, "LoadBalance": true, "Relay": true}
	for name, value := range proxyMap {
		item, ok := value.(map[string]any)
		if !ok {
			continue
		}
		all := stringSlice(item["all"])
		row := map[string]any{
			"name":          firstNonEmpty(stringMapValue(item, "name"), name),
			"type":          stringMapValue(item, "type"),
			"now":           stringMapValue(item, "now"),
			"all":           all,
			"all_count":     len(all),
			"udp":           boolMapValue(item, "udp", false),
			"delay":         latestProxyDelay(item),
			"history":       item["history"],
			"icon":          stringMapValue(item, "icon"),
			"hidden":        boolMapValue(item, "hidden", false),
			"alive":         boolMapValue(item, "alive", true),
			"provider":      firstNonEmpty(stringMapValue(item, "provider"), stringMapValue(item, "providerName"), stringMapValue(item, "provider-name")),
			"provider_name": firstNonEmpty(stringMapValue(item, "providerName"), stringMapValue(item, "provider-name"), stringMapValue(item, "provider")),
			"raw":           item,
		}
		if row["provider_name"] != "" {
			row["provider-name"] = row["provider_name"]
		}
		byName[stringMapValue(row, "name")] = row
		if groupTypes[stringMapValue(item, "type")] || len(stringSlice(item["all"])) > 0 {
			groups = append(groups, row)
		} else {
			proxies = append(proxies, row)
		}
	}
	sort.Slice(groups, func(i, j int) bool { return stringMapValue(groups[i], "name") < stringMapValue(groups[j], "name") })
	sort.Slice(proxies, func(i, j int) bool { return stringMapValue(proxies[i], "name") < stringMapValue(proxies[j], "name") })
	return byName, groups, proxies
}

func (a *App) mihomoRulesRuntime(r *http.Request) map[string]any {
	raw, ok := a.mihomoControllerMap("/rules")
	if !ok {
		raw = map[string]any{"rules": []any{}}
	}
	rules := normalizeMihomoRules(anySlice(raw["rules"]))
	if r != nil {
		q := r.URL.Query()
		search := strings.ToLower(strings.TrimSpace(firstNonEmpty(q.Get("search"), q.Get("q"), q.Get("keyword"))))
		typ := strings.TrimSpace(q.Get("type"))
		proxy := strings.TrimSpace(q.Get("proxy"))
		provider := strings.TrimSpace(q.Get("provider"))
		filtered := make([]map[string]any, 0, len(rules))
		for _, rule := range rules {
			if search != "" && !strings.Contains(strings.ToLower(strings.Join([]string{
				stringMapValue(rule, "type"), stringMapValue(rule, "payload"), stringMapValue(rule, "proxy"), stringMapValue(rule, "provider"),
			}, " ")), search) {
				continue
			}
			if typ != "" && typ != "all" && !strings.EqualFold(stringMapValue(rule, "type"), typ) {
				continue
			}
			if proxy != "" && proxy != "all" && stringMapValue(rule, "proxy") != proxy {
				continue
			}
			if provider != "" && provider != "all" && stringMapValue(rule, "provider") != provider {
				continue
			}
			filtered = append(filtered, rule)
		}
		rules = filtered
		sortMihomoRows(rules, q.Get("sort"), q.Get("sort_order"))
	}
	page, limit := 1, len(rules)
	if r != nil {
		page = queryInt(r, "page", 1)
		limit = queryInt(r, "page_size", queryInt(r, "limit", len(rules)))
	}
	if limit <= 0 {
		limit = 200
	}
	total := len(rules)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	items := rules[start:end]
	return map[string]any{
		"rules": items, "items": items, "total": total, "raw": raw,
		"pagination": map[string]any{"page": page, "limit": limit, "page_size": limit, "total": total, "total_pages": (total + limit - 1) / limit},
	}
}

func normalizeMihomoRules(items []any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for i, item := range items {
		switch v := item.(type) {
		case map[string]any:
			out = append(out, map[string]any{
				"id": i + 1, "index": i + 1,
				"type":     firstNonEmpty(stringMapValue(v, "type"), stringMapValue(v, "ruleType")),
				"payload":  firstNonEmpty(stringMapValue(v, "payload"), stringMapValue(v, "rulePayload")),
				"proxy":    firstNonEmpty(stringMapValue(v, "proxy"), stringMapValue(v, "adapter")),
				"provider": stringMapValue(v, "provider"),
				"raw":      v,
			})
		case string:
			parts := strings.Split(v, ",")
			row := map[string]any{"id": i + 1, "index": i + 1, "raw": v}
			if len(parts) > 0 {
				row["type"] = strings.TrimSpace(parts[0])
			}
			if len(parts) > 1 {
				row["payload"] = strings.TrimSpace(parts[1])
			}
			if len(parts) > 2 {
				row["proxy"] = strings.TrimSpace(parts[2])
			}
			out = append(out, row)
		}
	}
	return out
}

func (a *App) mihomoProvidersPayload() map[string]any {
	proxy := a.mihomoProxyProvidersPayload()
	rule := a.mihomoRuleProvidersPayload()
	return map[string]any{"proxy_providers": proxy["items"], "rule_providers": rule["items"], "proxy": proxy, "rule": rule}
}

func (a *App) mihomoProxyProvidersPayload() map[string]any {
	cfg := a.mihomoConfigMap()
	configProviders := normalizeConfigProviders(cfg["proxy-providers"])
	raw, ok := a.mihomoControllerMap("/providers/proxies")
	runtime := map[string]map[string]any{}
	if ok {
		runtime = normalizeProviderMap(raw["providers"])
	}
	items := mergeProviders(configProviders, runtime, "proxy")
	return map[string]any{"proxy-providers": cfg["proxy-providers"], "items": items, "providers": items, "runtime": runtime}
}

func (a *App) mihomoRuleProvidersPayload() map[string]any {
	cfg := a.mihomoConfigMap()
	configProviders := normalizeConfigProviders(cfg["rule-providers"])
	raw, ok := a.mihomoControllerMap("/providers/rules")
	runtime := map[string]map[string]any{}
	if ok {
		runtime = normalizeProviderMap(raw["providers"])
	}
	items := mergeProviders(configProviders, runtime, "rule")
	return map[string]any{"rule-providers": cfg["rule-providers"], "items": items, "providers": items, "runtime": runtime}
}

func (a *App) handleMihomoProxyProviderGet(w http.ResponseWriter, r *http.Request) {
	a.writeMihomoProviderGet(w, r.PathValue("name"), "proxy-providers", "/providers/proxies/")
}

func (a *App) handleMihomoProxyProviderPut(w http.ResponseWriter, r *http.Request) {
	a.writeMihomoProviderUpsert(w, r, r.PathValue("name"), "proxy-providers")
}

func (a *App) handleMihomoProxyProviderDelete(w http.ResponseWriter, r *http.Request) {
	a.writeMihomoProviderDelete(w, r.PathValue("name"), "proxy-providers")
}

func (a *App) handleMihomoProxyProviderUpdate(w http.ResponseWriter, r *http.Request) {
	a.writeMihomoProviderRuntimeUpdate(w, r.PathValue("name"), "proxy")
}

func (a *App) handleMihomoRuleProviders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": a.mihomoRuleProvidersPayload()})
}

func (a *App) handleMihomoRuleProvidersPut(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := a.updateMihomoConfigSections(req, "rule-providers"); err != nil {
		writeError(w, http.StatusBadRequest, "write_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "restart_required": true, "data": a.mihomoRuleProvidersPayload()})
}

func (a *App) handleMihomoRuleProviderGet(w http.ResponseWriter, r *http.Request) {
	a.writeMihomoProviderGet(w, r.PathValue("name"), "rule-providers", "/providers/rules/")
}

func (a *App) handleMihomoRuleProviderPut(w http.ResponseWriter, r *http.Request) {
	a.writeMihomoProviderUpsert(w, r, r.PathValue("name"), "rule-providers")
}

func (a *App) handleMihomoRuleProviderDelete(w http.ResponseWriter, r *http.Request) {
	a.writeMihomoProviderDelete(w, r.PathValue("name"), "rule-providers")
}

func (a *App) handleMihomoRuleProviderUpdate(w http.ResponseWriter, r *http.Request) {
	a.writeMihomoProviderRuntimeUpdate(w, r.PathValue("name"), "rule")
}

func (a *App) writeMihomoProviderGet(w http.ResponseWriter, name, section, runtimePrefix string) {
	cfgProviders := normalizeConfigProviders(a.mihomoConfigMap()[section])
	item, ok := cfgProviders[name]
	if raw, runtimeOK, _ := a.mihomoControllerJSON(http.MethodGet, runtimePrefix+url.PathEscape(name), nil); runtimeOK {
		item["runtime"] = raw
		ok = true
	}
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "provider not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": item})
}

func (a *App) writeMihomoProviderUpsert(w http.ResponseWriter, r *http.Request, name, section string) {
	var req map[string]any
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	providerName := firstNonEmpty(name, stringMapValue(req, "name"), stringMapValue(req, "tag"))
	provider, err := normalizeProviderRequest(providerName, req, section)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := a.upsertMihomoProvider(section, providerName, provider); err != nil {
		writeError(w, http.StatusBadRequest, "write_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "restart_required": true, "data": provider})
}

func (a *App) writeMihomoProviderDelete(w http.ResponseWriter, name, section string) {
	if strings.TrimSpace(name) == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "provider name required")
		return
	}
	cfg := a.mihomoConfigMap()
	providers := normalizeConfigProviders(cfg[section])
	delete(providers, name)
	cfg[section] = providerConfigMap(providers)
	if err := a.writeMihomoConfigMap(cfg); err != nil {
		writeError(w, http.StatusBadRequest, "write_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "restart_required": true})
}

func (a *App) writeMihomoProviderRuntimeUpdate(w http.ResponseWriter, name, kind string) {
	if name == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "provider name required")
		return
	}
	paths := []string{}
	if kind == "proxy" {
		paths = []string{"/providers/proxies/" + url.PathEscape(name) + "/healthcheck", "/providers/proxies/" + url.PathEscape(name)}
	} else {
		paths = []string{"/providers/rules/" + url.PathEscape(name)}
	}
	var lastErr error
	for _, path := range paths {
		if raw, ok, err := a.mihomoControllerJSON(http.MethodPut, path, nil); ok {
			writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": raw})
			return
		} else {
			lastErr = err
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": false, "warning": errString(lastErr, "mihomo controller unavailable"), "data": map[string]any{"updated": false}})
}

func (a *App) upsertMihomoProvider(section, name string, provider map[string]any) error {
	cfg := a.mihomoConfigMap()
	providers := normalizeConfigProviders(cfg[section])
	providers[name] = provider
	cfg[section] = providerConfigMap(providers)
	return a.writeMihomoConfigMap(cfg)
}

func normalizeProviderRequest(name string, req map[string]any, section string) (map[string]any, error) {
	raw := firstNonEmpty(stringMapValue(req, "value"), stringMapValue(req, "subscription"), stringMapValue(req, "input"))
	if raw != "" && stringMapValue(req, "url") == "" {
		tag, u := parseTaggedURL(raw)
		if name == "" {
			name = tag
		}
		req["url"] = u
	}
	if name == "" {
		return nil, fmt.Errorf("provider name required")
	}
	u := strings.TrimSpace(stringMapValue(req, "url"))
	if u == "" {
		return nil, fmt.Errorf("provider url required")
	}
	provider := map[string]any{}
	for k, v := range req {
		if k == "name" || k == "tag" || k == "value" || k == "subscription" || k == "input" {
			continue
		}
		provider[k] = v
	}
	if provider["type"] == nil {
		provider["type"] = "http"
	}
	if provider["url"] == nil {
		provider["url"] = u
	}
	if provider["path"] == nil {
		dir := "proxy_providers"
		ext := ".yaml"
		if section == "rule-providers" {
			dir = "rules"
			ext = ".mrs"
			if provider["behavior"] == nil {
				provider["behavior"] = "classical"
			}
			if provider["format"] == nil {
				provider["format"] = "yaml"
				ext = ".yaml"
			}
		}
		provider["path"] = filepath.ToSlash(filepath.Join("./"+dir, sanitizeProviderName(name)+ext))
	}
	if provider["interval"] == nil {
		provider["interval"] = 86400
	}
	return provider, nil
}

func parseTaggedURL(raw string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(raw), "|", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	u := strings.TrimSpace(raw)
	parsed, _ := url.Parse(u)
	name := parsed.Hostname()
	if name == "" {
		name = "provider"
	}
	return name, u
}

func sanitizeProviderName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", ":", "-", "|", "-")
	name = replacer.Replace(name)
	name = strings.Trim(name, ".-")
	if name == "" {
		return "provider"
	}
	return name
}

func (a *App) writeMihomoConfigMap(cfg map[string]any) error {
	if old, err := a.readTextFile("configs/mihomo/config.yaml"); err == nil {
		a.createConfigHistory("mihomo", "configs/mihomo/config.yaml", old, "auto backup before Mihomo config update", "system")
	}
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return a.writeTextFile("configs/mihomo/config.yaml", string(b))
}

func normalizeConfigProviders(raw any) map[string]map[string]any {
	out := map[string]map[string]any{}
	switch providers := raw.(type) {
	case map[string]any:
		for name, value := range providers {
			if item, ok := value.(map[string]any); ok {
				item["name"] = name
				out[name] = item
			}
		}
	case map[any]any:
		for key, value := range providers {
			name := fmt.Sprint(key)
			if item, ok := value.(map[string]any); ok {
				item["name"] = name
				out[name] = item
			}
		}
	}
	return out
}

func providerConfigMap(providers map[string]map[string]any) map[string]any {
	out := map[string]any{}
	for name, provider := range providers {
		cp := map[string]any{}
		for k, v := range provider {
			if k == "name" || k == "runtime" || k == "source" || k == "provider_type" {
				continue
			}
			cp[k] = v
		}
		out[name] = cp
	}
	return out
}

func normalizeProviderMap(raw any) map[string]map[string]any {
	out := map[string]map[string]any{}
	if providers, ok := raw.(map[string]any); ok {
		for name, value := range providers {
			if item, ok := value.(map[string]any); ok {
				item["name"] = firstNonEmpty(stringMapValue(item, "name"), name)
				out[name] = item
			}
		}
	}
	return out
}

func mergeProviders(config, runtime map[string]map[string]any, kind string) []map[string]any {
	names := map[string]bool{}
	for name := range config {
		names[name] = true
	}
	for name := range runtime {
		names[name] = true
	}
	sorted := make([]string, 0, len(names))
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)
	items := make([]map[string]any, 0, len(sorted))
	for _, name := range sorted {
		item := map[string]any{"name": name, "provider_type": kind}
		for k, v := range config[name] {
			item[k] = v
		}
		if rt, ok := runtime[name]; ok {
			item["runtime"] = rt
			item["updated_at"] = firstNonEmpty(stringMapValue(rt, "updatedAt"), stringMapValue(rt, "updated_at"))
			item["vehicle_type"] = stringMapValue(rt, "vehicleType")
			item["source"] = "config+controller"
		} else {
			item["source"] = "config"
		}
		items = append(items, item)
	}
	return items
}

func (a *App) handleMihomoUIConfig(w http.ResponseWriter, r *http.Request) {
	cfg := a.mihomoConfigMap()
	ports := mihomoPortsFromConfig(cfg)
	host := r.Host
	if strings.Contains(host, ":") {
		host, _, _ = net.SplitHostPort(host)
	}
	if host == "" {
		host = "127.0.0.1"
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{
		"url":        "/ui/",
		"controller": fmt.Sprintf("http://%s:%d", host, ports["controller"]),
		"host":       host,
		"port":       ports["controller"],
		"secret":     a.mihomoSecret(),
		"zashboard":  "/ui/",
	}})
}

func (a *App) downloadConfigDir(w http.ResponseWriter, rel, filename string) {
	root, err := a.safePath(rel)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	b, err := zipDir(root)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "zip_failed", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	_, _ = w.Write(b)
}

func (a *App) uploadConfigZip(w http.ResponseWriter, r *http.Request, destRel string) {
	if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		writeError(w, http.StatusBadRequest, "bad_upload", "multipart file required")
		return
	}
	if err := r.ParseMultipartForm(128 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "bad_upload", err.Error())
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_upload", err.Error())
		return
	}
	defer file.Close()
	tmp, err := os.CreateTemp("", "msm-free-config-*.zip")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "temp_failed", err.Error())
		return
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := io.Copy(tmp, io.LimitReader(file, 128<<20)); err != nil {
		tmp.Close()
		writeError(w, http.StatusInternalServerError, "upload_failed", err.Error())
		return
	}
	tmp.Close()
	dest, err := a.safePath(destRel)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	if err := restoreZipToDir(tmpPath, dest); err != nil {
		writeError(w, http.StatusBadRequest, "restore_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "restart_required": true, "data": map[string]any{"restart_required": true}})
}

func filterMihomoProxyList(items []map[string]any, search string) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(strings.Join([]string{
			stringMapValue(item, "name"), stringMapValue(item, "type"), stringMapValue(item, "now"),
		}, " ")), search) {
			out = append(out, item)
		}
	}
	return out
}

func filterMihomoProxyMap(items map[string]any, search string) map[string]any {
	out := map[string]any{}
	for name, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if strings.Contains(strings.ToLower(strings.Join([]string{
			name, stringMapValue(m, "name"), stringMapValue(m, "type"), stringMapValue(m, "now"), stringMapValue(m, "provider_name"),
		}, " ")), search) {
			out[name] = item
		}
	}
	return out
}

func latestProxyDelay(item map[string]any) float64 {
	history := anySlice(item["history"])
	if len(history) == 0 {
		return 0
	}
	last, _ := history[len(history)-1].(map[string]any)
	return numericMapValue(last, "delay")
}

func sortMihomoRows(rows []map[string]any, field, order string) {
	field = firstNonEmpty(field, "id")
	order = strings.ToLower(order)
	sort.SliceStable(rows, func(i, j int) bool {
		if field == "download" || field == "upload" || field == "delay" || field == "id" || field == "index" {
			li := numericMapValue(rows[i], field)
			ri := numericMapValue(rows[j], field)
			if order == "asc" {
				return li < ri
			}
			return li > ri
		}
		lv := stringMapValue(rows[i], field)
		rv := stringMapValue(rows[j], field)
		if order == "asc" {
			return lv < rv
		}
		return lv > rv
	})
}

func anyLen(v any) int {
	switch items := v.(type) {
	case []map[string]any:
		return len(items)
	case []any:
		return len(items)
	case []string:
		return len(items)
	case map[string]any:
		return len(items)
	case map[string]string:
		return len(items)
	default:
		return 0
	}
}

func anyMapSlice(v any) []map[string]any {
	switch items := v.(type) {
	case []map[string]any:
		return items
	case []any:
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

func stringSlice(v any) []string {
	switch items := v.(type) {
	case []string:
		return items
	case []any:
		out := make([]string, 0, len(items))
		for _, item := range items {
			if s := strings.TrimSpace(fmt.Sprint(item)); s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		items = strings.TrimSpace(items)
		if items == "" {
			return nil
		}
		return []string{items}
	default:
		return nil
	}
}

func intMapValue(m map[string]any, key string, fallback int) int {
	switch v := m[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func numericMapValue(m map[string]any, key string) float64 {
	if n, ok := firstNumeric(m, key); ok {
		return n
	}
	return 0
}

func intAny(value any, fallback int) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, err := v.Int64()
		if err == nil {
			return int(n)
		}
	case string:
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return fallback
}

func boolMapValue(m map[string]any, key string, fallback bool) bool {
	switch v := m[key].(type) {
	case bool:
		return v
	case string:
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func portFromListen(listen string, fallback int) int {
	listen = strings.TrimSpace(listen)
	if listen == "" {
		return fallback
	}
	if strings.HasPrefix(listen, ":") {
		listen = "127.0.0.1" + listen
	}
	_, port, err := net.SplitHostPort(listen)
	if err != nil {
		parts := strings.Split(listen, ":")
		port = parts[len(parts)-1]
	}
	if n, err := strconv.Atoi(port); err == nil && n > 0 {
		return n
	}
	return fallback
}

func (a *App) tcpPortOpen(host string, port int) bool {
	if port <= 0 {
		return false
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func errString(err error, fallback string) string {
	if err != nil {
		return err.Error()
	}
	return fallback
}
