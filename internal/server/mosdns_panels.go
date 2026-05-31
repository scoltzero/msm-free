package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (a *App) mosDNSAPIBase() string {
	base := strings.TrimRight(a.setting("mosdns_api_endpoint", "http://127.0.0.1:9099"), "/")
	if base == "" {
		return "http://127.0.0.1:9099"
	}
	if _, err := url.ParseRequestURI(base); err != nil {
		return "http://127.0.0.1:9099"
	}
	return base
}

func (a *App) mosDNSAPIURL(path string) string {
	if path == "" {
		return a.mosDNSAPIBase()
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return a.mosDNSAPIBase() + path
}

func (a *App) mosDNSSnapshot(limit int) map[string]any {
	st := a.Services.Status("mosdns")
	entries := a.mosDNSQueryDataset(limit)
	audit := mosDNSAuditStats(entries)
	cache := mosDNSCacheSummary(entries)
	if remoteCache, ok := a.mosDNSProxyCache(); ok {
		if summary, ok := remoteCache["summary"].(map[string]any); ok {
			cache = summary
		} else {
			cache = remoteCache
		}
	}
	upstreamSummary := mosDNSUpstreamStats(entries)
	upstreamRows := anyMapSlice(upstreamSummary["upstreams"])
	remoteStats, remoteOK := a.mosDNSProxyStats()
	queryCount := len(entries)
	if remoteOK {
		if v, ok := firstNumeric(remoteStats, "query_count", "total_queries", "queries", "total"); ok {
			queryCount = int(v)
		}
	}
	clientCount := int(a.countTable("mosdns_clients"))
	if clientCount == 0 {
		clientCount = len(uniqueValues(entries, "client_ip", 0))
	}
	cacheEntries := cache["entries"]
	if cacheEntries == nil {
		cacheEntries = cache["size"]
	}
	detailedCache := map[string]any{
		"summary": cache,
		"entries": mosDNSCacheRows(entries),
		"items":   mosDNSCacheRows(entries),
		"caches": map[string]any{
			"fallback": map[string]any{
				"query_total": cache["query_total"],
				"hit_total":   cache["hit_total"],
				"hit_rate":    cache["hit_rate"],
			},
		},
	}
	if remoteCache, ok := a.mosDNSProxyCache(); ok {
		detailedCache = remoteCache
		if _, ok := detailedCache["summary"]; !ok {
			detailedCache["summary"] = cache
		}
		if _, ok := detailedCache["caches"]; !ok {
			detailedCache["caches"] = map[string]any{"remote": map[string]any{"query_total": queryCount, "hit_total": cacheEntries, "hit_rate": cache["hit_rate"]}}
		}
	}
	stats := map[string]any{
		"cpu_percent":         st.CPU,
		"process_rss_bytes":   normalizeMemoryBytes(st.Memory),
		"go_goroutines":       0,
		"go_gc_count":         0,
		"go_gc_duration_sec":  0,
		"go_threads":          0,
		"open_fds":            0,
		"max_fds":             0,
		"cache_query_total":   queryCount,
		"cache_hit_total":     numericAny(cache["hit_total"]),
		"average_duration_ms": numericAny(audit["average_duration_ms"]),
	}
	if remoteOK {
		for key, value := range remoteStats {
			stats[key] = value
		}
	}
	data := map[string]any{
		"service":                st,
		"status":                 st.Status,
		"running":                st.Running,
		"installed":              st.Installed,
		"pid":                    st.PID,
		"cpu":                    st.CPU,
		"memory":                 st.Memory,
		"uptime":                 st.Uptime,
		"version":                st.Version,
		"clients":                clientCount,
		"client_count":           clientCount,
		"client_ips":             a.countTable("mosdns_client_ips"),
		"switches":               a.mosDNSSwitchMap(),
		"api_endpoint":           a.mosDNSAPIBase(),
		"dns_listen":             ":53",
		"query_count":            queryCount,
		"cache_entries":          cacheEntries,
		"cache":                  cache,
		"detailed_cache":         detailedCache,
		"audit":                  audit,
		"audit_stats":            audit,
		"audit_ranks":            map[string]any{"domain": audit["top_domains"], "client": audit["top_clients"], "rule": audit["top_rules"], "domain_set": audit["top_rules"]},
		"stats":                  stats,
		"upstream_stats":         upstreamRows,
		"upstream_summary":       upstreamSummary,
		"upstream_stats_summary": upstreamSummary,
		"top_domains":            audit["top_domains"],
		"top_clients":            audit["top_clients"],
		"top_rules":              audit["top_rules"],
		"meta":                   mosDNSQueryMeta(entries),
		"source":                 "fallback",
	}
	if remoteOK {
		data["source"] = "mosdns_9099"
		data["remote"] = remoteStats
	}
	return data
}

func normalizeMemoryBytes(value int64) int64 {
	if value <= 0 {
		return 0
	}
	if value < 1_000_000 {
		return value * 1024 * 1024
	}
	return value
}

func numericAny(value any) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case float64:
		return v
	case json.Number:
		n, _ := v.Float64()
		return n
	case string:
		n, _ := strconv.ParseFloat(v, 64)
		return n
	default:
		return 0
	}
}

func (a *App) mosDNSProxyStats() (map[string]any, bool) {
	for _, path := range []string{"/api/stats", "/stats", "/webinfo", "/plugins/webinfo/get", "/metrics"} {
		var raw any
		if proxyJSON(a.mosDNSAPIURL(path), &raw) {
			if normalized := normalizeMapPayload(raw); len(normalized) > 0 {
				return normalized, true
			}
		}
	}
	return nil, false
}

func (a *App) mosDNSProxyCache() (map[string]any, bool) {
	for _, path := range []string{"/api/cache", "/cache", "/cache/detailed", "/plugins/cache/get"} {
		var raw any
		if proxyJSON(a.mosDNSAPIURL(path), &raw) {
			if normalized := normalizeMapPayload(raw); len(normalized) > 0 {
				if _, ok := normalized["entries"]; !ok {
					if rows, ok := normalized["caches"].([]any); ok {
						normalized["entries"] = len(rows)
					}
					if rows, ok := normalized["items"].([]any); ok {
						normalized["entries"] = len(rows)
					}
				}
				return normalized, true
			}
		}
	}
	return nil, false
}

func normalizeMapPayload(raw any) map[string]any {
	switch v := raw.(type) {
	case map[string]any:
		for _, key := range []string{"data", "result", "stats", "cache"} {
			if nested, ok := v[key].(map[string]any); ok {
				return nested
			}
		}
		return v
	default:
		return nil
	}
}

func firstNumeric(m map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		switch v := m[key].(type) {
		case float64:
			return v, true
		case int:
			return float64(v), true
		case int64:
			return float64(v), true
		case json.Number:
			n, err := v.Float64()
			return n, err == nil
		case string:
			n, err := strconv.ParseFloat(v, 64)
			if err == nil {
				return n, true
			}
		}
	}
	return 0, false
}

func (a *App) mosDNSClientIPSet() map[string]bool {
	rows, err := a.DB.Query(`select ip from mosdns_client_ips`)
	if err != nil {
		return map[string]bool{}
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var ip string
		if rows.Scan(&ip) == nil {
			out[ip] = true
		}
	}
	return out
}

func normalizeMosDNSClientStatus(status string, allowListed bool) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "allow", "allowed", "white", "whitelist", "client_ip", "proxy", "enabled":
		return "allow"
	case "deny", "denied", "black", "blacklist", "block":
		return "deny"
	case "disabled", "disable", "closed", "close", "off":
		return "disabled"
	case "unscanned", "unknown", "scan", "lan", "":
		if allowListed {
			return "allow"
		}
		return "unscanned"
	default:
		if allowListed {
			return "allow"
		}
		return "unscanned"
	}
}

func (a *App) setMosDNSClientIPAllowed(ip string, allowed bool, comment string) error {
	if strings.TrimSpace(ip) == "" {
		return nil
	}
	if allowed {
		_, err := a.DB.Exec(`insert into mosdns_client_ips(ip,comment,created_at,updated_at) values(?,?,?,?) on conflict(ip) do update set comment=excluded.comment,updated_at=excluded.updated_at`, ip, comment, time.Now(), time.Now())
		if err != nil {
			return err
		}
	} else {
		if _, err := a.DB.Exec(`delete from mosdns_client_ips where ip=?`, ip); err != nil {
			return err
		}
	}
	return a.rewriteMosDNSClientIPFile()
}

func (a *App) syncMosDNSClientAllowList(idOrKey, status string) error {
	var ip, name string
	err := a.DB.QueryRow(`select ip,coalesce(custom_name,hostname,'') from mosdns_clients where id=? or ip=? or mac=? order by id desc limit 1`, idOrKey, idOrKey, idOrKey).Scan(&ip, &name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	return a.setMosDNSClientIPAllowed(ip, status == "allow", name)
}

func mosDNSRulePatternsFromRequest(patterns []string, items []struct {
	Pattern string `json:"pattern"`
	Content string `json:"content"`
	Name    string `json:"name"`
	Value   string `json:"value"`
}) []string {
	out := append([]string{}, patterns...)
	for _, item := range items {
		value := firstNonEmpty(item.Pattern, item.Content, item.Value, item.Name)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func (a *App) mosDNSLogCapacity() int {
	n, err := strconv.Atoi(a.setting("mosdns_log_capacity", "5000"))
	if err != nil || n <= 0 {
		return 5000
	}
	if n > 50000 {
		return 50000
	}
	return n
}

func filterLogLines(lines []string, r *http.Request) []string {
	level := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("level")))
	query := strings.ToLower(strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("q"), r.URL.Query().Get("keyword"), r.URL.Query().Get("search"))))
	if level == "" && query == "" {
		return lines
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		lower := strings.ToLower(line)
		if level != "" && logLevelFromLine(line) != level {
			continue
		}
		if query != "" && !strings.Contains(lower, query) {
			continue
		}
		out = append(out, line)
	}
	return out
}

func structuredLogLines(lines []string) []map[string]any {
	out := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		if entry, ok := structuredJSONLogLine(line); ok {
			out = append(out, entry)
			continue
		}
		out = append(out, map[string]any{
			"time":    firstLogTime(line),
			"level":   logLevelFromLine(line),
			"message": line,
			"display": line,
			"raw":     line,
		})
	}
	return out
}

func structuredJSONLogLine(line string) (map[string]any, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "{") {
		return nil, false
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return nil, false
	}
	msg := stringMapValue(raw, "msg")
	caller := stringMapValue(raw, "caller")
	message := firstNonEmpty(
		stringMapValue(raw, "message"),
		stringMapValue(raw, "error"),
		stringMapValue(raw, "path"),
		stringMapValue(raw, "addr"),
		stringMapValue(raw, "endpoint"),
		stringMapValue(raw, "config"),
	)
	display := msg
	if caller != "" && msg != "" && message != "" {
		display = fmt.Sprintf("[%s] %s: %s", caller, msg, message)
	} else if caller != "" && msg != "" {
		display = fmt.Sprintf("[%s] %s", caller, msg)
	} else if message != "" {
		display = message
	}
	if display == "" {
		display = line
	}
	level := strings.ToLower(firstNonEmpty(stringMapValue(raw, "level"), "info"))
	return map[string]any{
		"time":    stringMapValue(raw, "time"),
		"level":   level,
		"message": display,
		"display": display,
		"caller":  caller,
		"msg":     msg,
		"raw":     line,
	}, true
}

func displayLogLine(line string) string {
	if entry, ok := structuredJSONLogLine(line); ok {
		return fmtAny(entry["display"])
	}
	return line
}

func mosDNSQueryRawLines(entries []map[string]any) []string {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		line := stringMapValue(entry, "raw")
		if line == "" || strings.HasPrefix(line, "map[") {
			line = fmt.Sprintf("time=%s client_ip=%s query_name=%s qtype=%s rule=%s rcode=%s",
				stringMapValue(entry, "query_time"),
				stringMapValue(entry, "client_ip"),
				stringMapValue(entry, "query_name"),
				stringMapValue(entry, "query_type"),
				stringMapValue(entry, "domain_set"),
				stringMapValue(entry, "response_code"),
			)
		}
		lines = append(lines, line)
	}
	return lines
}

func logLevelFromLine(line string) string {
	if entry, ok := structuredJSONLogLine(line); ok {
		return fmtAny(entry["level"])
	}
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "panic") || strings.Contains(lower, "fatal"):
		return "fatal"
	case strings.Contains(lower, "error") || strings.Contains(lower, "[err]"):
		return "error"
	case strings.Contains(lower, "warn"):
		return "warn"
	case strings.Contains(lower, "debug"):
		return "debug"
	default:
		return "info"
	}
}

func firstLogTime(line string) string {
	if entry, ok := structuredJSONLogLine(line); ok {
		return fmtAny(entry["time"])
	}
	fields := strings.Fields(line)
	for _, field := range fields {
		field = strings.Trim(field, "[]")
		if _, err := time.Parse(time.RFC3339, field); err == nil {
			return field
		}
		if t, err := time.Parse("2006-01-02", field); err == nil {
			return t.Format(time.RFC3339)
		}
	}
	return ""
}

func (a *App) mosDNSRoutingState() map[string]any {
	state, ok := a.jsonSetting("mosdns_routing_task", defaultMosDNSRoutingState()).(map[string]any)
	if !ok {
		return defaultMosDNSRoutingState()
	}
	return state
}

func (a *App) generateMosDNSRoutingRules() (map[string]any, error) {
	state := defaultMosDNSRoutingState()
	state["running"] = true
	state["enabled"] = true
	state["status"] = "running"
	state["progress"] = 10
	a.storeJSONSetting("mosdns_routing_task", state)
	entries := a.mosDNSQueryDataset(10000)
	fakeSet := map[string]bool{}
	realSet := map[string]bool{}
	counts := map[string]int{}
	for _, entry := range entries {
		domain := stringMapValue(entry, "query_name")
		if domain == "" {
			continue
		}
		counts[domain]++
		if entryHasFakeIP(entry) {
			fakeSet[domain] = true
			continue
		}
		realSet[domain] = true
	}
	fakeDomains := sortedKeys(fakeSet)
	realDomains := sortedKeys(realSet)
	top := topDomainLines(counts, 200)
	files := map[string][]string{
		"fakeiprule.txt":  prefixedDomainLines(fakeDomains),
		"fakeiplist.txt":  fakeDomains,
		"realiprule.txt":  prefixedDomainLines(realDomains),
		"realiplist.txt":  realDomains,
		"top_domains.txt": top,
	}
	for name, lines := range files {
		rel := filepath.ToSlash(filepath.Join("configs/mosdns/gen", name))
		content := strings.Join(lines, "\n")
		if content != "" {
			content += "\n"
		}
		if err := a.writeTextFile(rel, content); err != nil {
			state["status"] = "failed"
			state["running"] = false
			state["error"] = err.Error()
			a.storeJSONSetting("mosdns_routing_task", state)
			return state, err
		}
	}
	state["running"] = false
	state["status"] = "completed"
	state["progress"] = 100
	state["last_run_at"] = time.Now().Format(time.RFC3339)
	state["fakeip_count"] = len(fakeDomains)
	state["realip_count"] = len(realDomains)
	state["rules"] = []map[string]any{
		{"name": "fakeiprule.txt", "count": len(fakeDomains), "path": "configs/mosdns/gen/fakeiprule.txt"},
		{"name": "realiprule.txt", "count": len(realDomains), "path": "configs/mosdns/gen/realiprule.txt"},
	}
	a.storeJSONSetting("mosdns_routing_task", state)
	return state, nil
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func prefixedDomainLines(domains []string) []string {
	out := make([]string, 0, len(domains))
	for _, domain := range domains {
		if strings.Contains(domain, ":") {
			out = append(out, domain)
		} else {
			out = append(out, "domain:"+domain)
		}
	}
	return out
}

func topDomainLines(counts map[string]int, limit int) []string {
	type row struct {
		domain string
		count  int
	}
	rows := make([]row, 0, len(counts))
	for domain, count := range counts {
		rows = append(rows, row{domain: domain, count: count})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].count > rows[j].count })
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, fmt.Sprintf("%s %d", row.domain, row.count))
	}
	return out
}

func (a *App) storeJSONSetting(key string, value any) {
	b, err := json.Marshal(value)
	if err != nil {
		return
	}
	a.setSetting(key, string(b))
}

func (a *App) createConfigHistory(service, path, content, comment, username string) {
	if username == "" {
		username = "system"
	}
	now := nowString()
	_, _ = a.DB.Exec(`insert into config_histories(service,file_path,content,comment,is_stable,created_by,created_at,updated_at) values(?,?,?,?,?,?,?,?)`,
		service, path, content, comment, false, username, now, now)
}

func currentUsername(r *http.Request) string {
	if user := currentUser(r); user != nil {
		return user.Username
	}
	return "system"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
