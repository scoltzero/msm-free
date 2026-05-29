package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ipv4Pattern      = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	queryTypePattern = regexp.MustCompile(`\b(A|AAAA|HTTPS|SVCB|PTR|TXT|MX|CNAME|SRV|NS|SOA)\b`)
	durationPattern  = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(ms|µs|us|s)\b`)
	answerPattern    = regexp.MustCompile(`\b(A|AAAA|CNAME):\s*([^,\s]+)`)
	keyValuePattern  = regexp.MustCompile(`(?i)(query_name|query|domain|client_ip|client|qtype|query_type|type|domain_set|rule|rcode|response_code|response|duration|elapsed|cost)[=:]\s*("?[^",\s]+"?)`)
)

func (a *App) mosDNSQueryDataset(limit int) []map[string]any {
	if limit <= 0 {
		limit = 5000
	}
	entries := mosDNSQueryEntries(a.serviceLogLines("mosdns", limit))
	if len(entries) == 0 {
		entries = mosDNSProxyQueryEntries(limit)
	}
	if len(entries) > limit {
		return entries[len(entries)-limit:]
	}
	return entries
}

func mosDNSProxyQueryEntries(limit int) []map[string]any {
	urls := []string{
		fmt.Sprintf("http://127.0.0.1:9099/api/query-logs?limit=%d", limit),
		fmt.Sprintf("http://127.0.0.1:9099/query-logs?limit=%d", limit),
		fmt.Sprintf("http://127.0.0.1:9099/plugins/query_log/get?limit=%d", limit),
	}
	for _, url := range urls {
		var raw any
		if proxyJSON(url, &raw) {
			if entries := normalizeMosDNSQueryPayload(raw); len(entries) > 0 {
				return entries
			}
		}
	}
	return nil
}

func normalizeMosDNSQueryPayload(raw any) []map[string]any {
	switch v := raw.(type) {
	case []any:
		return normalizeMosDNSQueryArray(v)
	case map[string]any:
		for _, key := range []string{"data", "logs", "items", "records", "queries"} {
			if arr, ok := v[key].([]any); ok {
				return normalizeMosDNSQueryArray(arr)
			}
			if nested, ok := v[key].(map[string]any); ok {
				if entries := normalizeMosDNSQueryPayload(nested); len(entries) > 0 {
					return entries
				}
			}
		}
	}
	return nil
}

func normalizeMosDNSQueryArray(items []any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for i, item := range items {
		switch v := item.(type) {
		case map[string]any:
			out = append(out, normalizeMosDNSQueryMap(v, i, ""))
		case string:
			out = append(out, parseMosDNSQueryLine(v, i))
		}
	}
	return out
}

func parseMosDNSQueryEntries(lines []string) []map[string]any {
	entries := make([]map[string]any, 0, len(lines))
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if start := strings.IndexByte(line, '{'); start >= 0 {
			var obj map[string]any
			if json.Unmarshal([]byte(line[start:]), &obj) == nil && len(obj) > 0 {
				entry := normalizeMosDNSQueryMap(obj, i, line)
				if entry["query_name"] != "" {
					entries = append(entries, entry)
					continue
				}
			}
		}
		entries = append(entries, parseMosDNSQueryLine(line, i))
	}
	return entries
}

func normalizeMosDNSQueryMap(obj map[string]any, index int, raw string) map[string]any {
	queryName := firstString(obj, "query_name", "query", "domain", "name", "qname", "host")
	clientIP := firstString(obj, "client_ip", "client", "client_addr", "remote_addr", "src")
	queryType := strings.ToUpper(firstString(obj, "query_type", "qtype", "type"))
	domainSet := firstString(obj, "domain_set", "rule", "matched_rule", "tag", "plugin", "matcher")
	responseCode := strings.ToUpper(firstString(obj, "response_code", "rcode", "response", "status"))
	duration := firstFloat(obj, "duration_ms", "elapsed_ms", "cost_ms", "latency_ms", "duration", "elapsed", "cost")
	answers := anySlice(obj["answers"])
	if len(answers) == 0 {
		answers = anySlice(obj["answer"])
	}
	if queryName == "" && raw != "" {
		queryName = extractDomainLike(raw)
	}
	if clientIP == "" && raw != "" {
		clientIP = firstIPv4(raw)
	}
	if queryType == "" {
		queryType = "A"
	}
	if domainSet == "" {
		domainSet = "unmatched_rule"
	}
	if responseCode == "" {
		responseCode = "NOERROR"
	}
	queryTime := firstString(obj, "query_time", "time", "timestamp", "created_at")
	if queryTime == "" {
		queryTime = time.Now().Add(time.Duration(index) * -time.Second).Format(time.RFC3339)
	}
	return map[string]any{
		"trace_id":      nonEmpty(firstString(obj, "trace_id", "id"), fmt.Sprintf("log-%d", index+1)),
		"query_time":    queryTime,
		"time":          queryTime,
		"query_name":    trimQueryName(queryName, raw),
		"domain":        trimQueryName(queryName, raw),
		"client_ip":     nonEmpty(clientIP, "127.0.0.1"),
		"client":        nonEmpty(clientIP, "127.0.0.1"),
		"query_type":    queryType,
		"type":          queryType,
		"domain_set":    domainSet,
		"rule":          domainSet,
		"response_code": responseCode,
		"response":      responseCode,
		"duration_ms":   duration,
		"answers":       answers,
		"raw":           nonEmpty(raw, fmt.Sprint(obj)),
	}
}

func parseMosDNSQueryLine(line string, index int) map[string]any {
	values := map[string]string{}
	for _, match := range keyValuePattern.FindAllStringSubmatch(line, -1) {
		if len(match) >= 3 {
			values[strings.ToLower(match[1])] = strings.Trim(match[2], `"`)
		}
	}
	queryName := values["query_name"]
	if queryName == "" {
		queryName = values["query"]
	}
	if queryName == "" {
		queryName = values["domain"]
	}
	if queryName == "" {
		queryName = extractDomainLike(line)
	}
	clientIP := values["client_ip"]
	if clientIP == "" {
		clientIP = values["client"]
	}
	if clientIP == "" {
		clientIP = firstIPv4(line)
	}
	queryType := strings.ToUpper(values["query_type"])
	if queryType == "" {
		queryType = strings.ToUpper(values["qtype"])
	}
	if queryType == "" {
		queryType = strings.ToUpper(values["type"])
	}
	if queryType == "" {
		if m := queryTypePattern.FindStringSubmatch(line); len(m) > 1 {
			queryType = strings.ToUpper(m[1])
		}
	}
	domainSet := values["domain_set"]
	if domainSet == "" {
		domainSet = values["rule"]
	}
	responseCode := strings.ToUpper(values["response_code"])
	if responseCode == "" {
		responseCode = strings.ToUpper(values["rcode"])
	}
	if responseCode == "" {
		responseCode = strings.ToUpper(values["response"])
	}
	duration := parseDurationMS(values["duration"])
	if duration == 0 {
		duration = parseDurationMS(values["elapsed"])
	}
	if duration == 0 {
		duration = parseDurationMS(values["cost"])
	}
	if duration == 0 {
		duration = firstDurationMS(line)
	}
	answers := parseAnswers(line)
	if queryType == "" {
		queryType = "A"
	}
	if domainSet == "" {
		domainSet = "unmatched_rule"
	}
	if responseCode == "" {
		responseCode = "NOERROR"
	}
	queryTime := time.Now().Add(time.Duration(index) * -time.Second).Format(time.RFC3339)
	return map[string]any{
		"trace_id":      fmt.Sprintf("log-%d", index+1),
		"query_time":    queryTime,
		"time":          queryTime,
		"query_name":    trimQueryName(queryName, line),
		"domain":        trimQueryName(queryName, line),
		"client_ip":     nonEmpty(clientIP, "127.0.0.1"),
		"client":        nonEmpty(clientIP, "127.0.0.1"),
		"query_type":    queryType,
		"type":          queryType,
		"domain_set":    domainSet,
		"rule":          domainSet,
		"response_code": responseCode,
		"response":      responseCode,
		"duration_ms":   duration,
		"answers":       answers,
		"raw":           line,
	}
}

func filterMosDNSQueryEntries(entries []map[string]any, r *http.Request) []map[string]any {
	q := r.URL.Query()
	search := strings.TrimSpace(firstNonEmpty(q.Get("search"), q.Get("keyword"), q.Get("domain"), q.Get("query")))
	client := strings.TrimSpace(firstNonEmpty(q.Get("client"), q.Get("client_ip")))
	queryType := strings.ToUpper(strings.TrimSpace(firstNonEmpty(q.Get("type"), q.Get("query_type"))))
	rule := strings.TrimSpace(firstNonEmpty(q.Get("rule"), q.Get("domain_set")))
	response := strings.ToUpper(strings.TrimSpace(firstNonEmpty(q.Get("response"), q.Get("response_code"))))
	exact := q.Get("exact") == "true"
	filtered := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		name := stringMapValue(entry, "query_name")
		if search != "" {
			if exact && !strings.EqualFold(name, search) {
				continue
			}
			if !exact && !strings.Contains(strings.ToLower(name), strings.ToLower(search)) {
				continue
			}
		}
		if client != "" && stringMapValue(entry, "client_ip") != client {
			continue
		}
		if queryType != "" && strings.ToUpper(stringMapValue(entry, "query_type")) != queryType {
			continue
		}
		if rule != "" && stringMapValue(entry, "domain_set") != rule {
			continue
		}
		if response != "" && strings.ToUpper(stringMapValue(entry, "response_code")) != response {
			continue
		}
		filtered = append(filtered, entry)
	}
	field := strings.TrimSpace(firstNonEmpty(q.Get("sort_field"), q.Get("sort")))
	order := strings.ToLower(q.Get("sort_order"))
	if field == "" {
		field = "query_time"
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		left := stringMapValue(filtered[i], field)
		right := stringMapValue(filtered[j], field)
		if order == "asc" {
			return left < right
		}
		return left > right
	})
	return filtered
}

func mosDNSQueryMeta(entries []map[string]any) map[string]any {
	domains := uniqueValues(entries, "query_name", 500)
	clients := uniqueValues(entries, "client_ip", 200)
	types := uniqueValues(entries, "query_type", 50)
	rules := uniqueValues(entries, "domain_set", 200)
	responses := uniqueValues(entries, "response_code", 50)
	if len(types) == 0 {
		types = []string{"A", "AAAA", "HTTPS"}
	}
	if len(responses) == 0 {
		responses = []string{"NOERROR", "NXDOMAIN", "SERVFAIL"}
	}
	return map[string]any{
		"domains":        domains,
		"clients":        clients,
		"types":          types,
		"query_types":    types,
		"rules":          rules,
		"responses":      responses,
		"response_codes": responses,
	}
}

func mosDNSAuditStats(entries []map[string]any) map[string]any {
	blocked := 0
	fakeIP := 0
	for _, entry := range entries {
		rule := strings.ToLower(stringMapValue(entry, "domain_set"))
		resp := strings.ToUpper(stringMapValue(entry, "response_code"))
		if strings.Contains(rule, "ban") || strings.Contains(rule, "block") || strings.Contains(rule, "ad") || resp == "NXDOMAIN" || resp == "REFUSED" {
			blocked++
		}
		if entryHasFakeIP(entry) {
			fakeIP++
		}
	}
	return map[string]any{
		"total_queries":   len(entries),
		"blocked_queries": blocked,
		"fakeip_queries":  fakeIP,
		"direct_queries":  len(entries) - fakeIP,
		"top_clients":     mosDNSRank(entries, "client_ip", 10),
		"top_domains":     mosDNSRank(entries, "query_name", 10),
		"top_rules":       mosDNSRank(entries, "domain_set", 10),
	}
}

func mosDNSRank(entries []map[string]any, key string, limit int) []map[string]any {
	counts := map[string]int{}
	for _, entry := range entries {
		value := stringMapValue(entry, key)
		if value == "" {
			continue
		}
		counts[value]++
	}
	rows := make([]map[string]any, 0, len(counts))
	for value, count := range counts {
		rows = append(rows, map[string]any{"name": value, "value": value, "count": count})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i]["count"].(int) > rows[j]["count"].(int)
	})
	if limit > 0 && len(rows) > limit {
		return rows[:limit]
	}
	return rows
}

func mosDNSCacheSummary(entries []map[string]any) map[string]any {
	cached := len(mosDNSCacheRows(entries))
	hitRate := 0.0
	if len(entries) > 0 {
		hitRate = float64(cached) * 100 / float64(len(entries))
	}
	return map[string]any{"entries": cached, "hit_rate": hitRate}
}

func mosDNSCacheRows(entries []map[string]any) []map[string]any {
	seen := map[string]bool{}
	var rows []map[string]any
	for _, entry := range entries {
		name := stringMapValue(entry, "query_name")
		if name == "" || seen[name] {
			continue
		}
		if len(anySlice(entry["answers"])) == 0 && !entryHasFakeIP(entry) {
			continue
		}
		seen[name] = true
		rows = append(rows, map[string]any{
			"domain":       name,
			"query_type":   stringMapValue(entry, "query_type"),
			"rule":         stringMapValue(entry, "domain_set"),
			"answers":      entry["answers"],
			"cached_until": "",
		})
	}
	if len(rows) > 200 {
		return rows[:200]
	}
	return rows
}

func mosDNSUpstreamStats(entries []map[string]any) map[string]any {
	groups := []struct {
		Name string
		Want func(map[string]any) bool
	}{
		{"fakeip", func(e map[string]any) bool { return entryHasFakeIP(e) }},
		{"direct", func(e map[string]any) bool {
			rule := strings.ToLower(stringMapValue(e, "domain_set"))
			return !entryHasFakeIP(e) && (strings.Contains(rule, "white") || strings.Contains(rule, "direct") || strings.Contains(rule, "unmatched") || rule == "")
		}},
		{"blocked", func(e map[string]any) bool {
			rule := strings.ToLower(stringMapValue(e, "domain_set"))
			return strings.Contains(rule, "ban") || strings.Contains(rule, "block") || strings.Contains(rule, "ad")
		}},
	}
	var upstreams []map[string]any
	covered := 0
	for _, group := range groups {
		count := 0
		for _, entry := range entries {
			if group.Want(entry) {
				count++
			}
		}
		covered += count
		upstreams = append(upstreams, map[string]any{"name": group.Name, "count": count, "ok": true})
	}
	upstreams = append(upstreams, map[string]any{"name": "other", "count": maxInt(0, len(entries)-covered), "ok": true})
	return map[string]any{"upstreams": upstreams, "total": len(entries), "response_codes": mosDNSRank(entries, "response_code", 10)}
}

func (a *App) mosDNSRuleSets() []map[string]any {
	roots := []string{"configs/mosdns/rule", "configs/mosdns/srs", "configs/mosdns/adguard"}
	var rows []map[string]any
	for _, root := range roots {
		abs, err := a.safePath(root)
		if err != nil {
			continue
		}
		entries, err := os.ReadDir(abs)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			rel := filepath.ToSlash(filepath.Join(root, entry.Name()))
			content, _ := a.readTextFile(rel)
			lines := splitNonEmptyLines(content)
			rows = append(rows, map[string]any{
				"name":       entry.Name(),
				"path":       rel,
				"type":       filepath.Base(root),
				"count":      len(lines),
				"updated_at": fileModTime(filepath.Join(abs, entry.Name())),
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return fmt.Sprint(rows[i]["path"]) < fmt.Sprint(rows[j]["path"]) })
	return rows
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			switch v := value.(type) {
			case string:
				if strings.TrimSpace(v) != "" {
					return strings.TrimSpace(v)
				}
			case float64:
				return strconv.FormatFloat(v, 'f', -1, 64)
			case int:
				return strconv.Itoa(v)
			}
		}
	}
	return ""
}

func firstFloat(m map[string]any, keys ...string) float64 {
	for _, key := range keys {
		switch v := m[key].(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case string:
			return parseDurationMS(v)
		}
	}
	return 0
}

func anySlice(v any) []any {
	switch x := v.(type) {
	case []any:
		return x
	case []string:
		out := make([]any, 0, len(x))
		for _, item := range x {
			out = append(out, item)
		}
		return out
	case string:
		if x == "" {
			return nil
		}
		return []any{x}
	default:
		return nil
	}
}

func parseDurationMS(value string) float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if v, err := strconv.ParseFloat(value, 64); err == nil {
		return v
	}
	return firstDurationMS(value)
}

func firstDurationMS(line string) float64 {
	match := durationPattern.FindStringSubmatch(line)
	if len(match) < 3 {
		return 0
	}
	value, _ := strconv.ParseFloat(match[1], 64)
	switch strings.ToLower(match[2]) {
	case "s":
		return value * 1000
	case "µs", "us":
		return value / 1000
	default:
		return value
	}
}

func parseAnswers(line string) []any {
	var out []any
	for _, match := range answerPattern.FindAllStringSubmatch(line, -1) {
		if len(match) >= 3 {
			out = append(out, map[string]any{"type": match[1], "value": strings.Trim(match[2], ",")})
		}
	}
	return out
}

func firstIPv4(line string) string {
	for _, ip := range ipv4Pattern.FindAllString(line, -1) {
		if strings.HasPrefix(ip, "127.") {
			continue
		}
		return ip
	}
	if ip := ipv4Pattern.FindString(line); ip != "" {
		return ip
	}
	return ""
}

func trimQueryName(name, fallback string) string {
	name = strings.Trim(strings.TrimSpace(name), ".")
	if name == "" {
		name = extractDomainLike(fallback)
	}
	if len(name) > 180 {
		return name[:180]
	}
	return name
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func stringMapValue(m map[string]any, key string) string {
	if value, ok := m[key]; ok {
		switch v := value.(type) {
		case string:
			return v
		case fmt.Stringer:
			return v.String()
		default:
			return fmt.Sprint(v)
		}
	}
	return ""
}

func uniqueValues(entries []map[string]any, key string, limit int) []string {
	seen := map[string]bool{}
	var out []string
	for _, entry := range entries {
		value := stringMapValue(entry, key)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	sort.Strings(out)
	return out
}

func entryHasFakeIP(entry map[string]any) bool {
	for _, ans := range anySlice(entry["answers"]) {
		if strings.Contains(fmt.Sprint(ans), "28.") || strings.Contains(fmt.Sprint(ans), "f2b0:") {
			return true
		}
	}
	rule := strings.ToLower(stringMapValue(entry, "domain_set"))
	return strings.Contains(rule, "fakeip") || strings.Contains(rule, "greylist")
}

func fileModTime(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return ""
	}
	return info.ModTime().Format(time.RFC3339)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
