package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSetupInitializeLoginAndGeneratedConfigs(t *testing.T) {
	app := newTestApp(t)
	body := map[string]any{
		"username":             "root",
		"password":             "86558781",
		"confirmPassword":      "86558781",
		"webPort":              "17777",
		"selected_interface":   "eth0",
		"subscription_urls":    "机场A|https://example.com/a.yaml\nhttps://example.com/b.yaml",
		"mihomo_proxies":       "vless://c20c7f96-e1a1-4203-bffa-888ef04959fd@example.com:443?encryption=none&security=reality&type=tcp&sni=gateway.icloud.com&fp=chrome&pbk=abc&sid=123&flow=xtls-rprx-vision#USxDMITan4pro-vless_reality_vision",
		"enableIPv6":           true,
		"nft_proxy_policy":     "direct_default",
		"fakeIPRangeV4":        "28.0.0.0/8",
		"linux_proxy_mode":     "nft",
		"mihomo_core_type":     "meta",
		"auto_set_dns":         true,
		"proxyCore":            "mihomo",
		"mosdnsEnabled":        true,
		"github_proxy_enabled": false,
	}
	res := requestJSON(t, app, http.MethodPost, "/api/v1/setup/initialize", "", body)
	if res.Code != http.StatusOK {
		t.Fatalf("initialize status=%d body=%s", res.Code, res.Body.String())
	}
	if !app.IsInitialized() {
		t.Fatal("app should be initialized")
	}
	if got := app.setting(serviceDesiredKey("mihomo"), ""); got != "true" {
		t.Fatalf("mihomo should be enabled after setup, got %q", got)
	}
	if got := app.setting(serviceDesiredKey("mosdns"), ""); got != "true" {
		t.Fatalf("mosdns should be enabled after setup, got %q", got)
	}
	if got := app.setting(nftDesiredKey, ""); got != "true" {
		t.Fatalf("nftables should be enabled after setup, got %q", got)
	}
	login := requestJSON(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"username": "root", "password": "86558781"})
	if login.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", login.Code, login.Body.String())
	}
	var loginBody map[string]any
	_ = json.Unmarshal(login.Body.Bytes(), &loginBody)
	if loginBody["token"] == "" {
		t.Fatal("login response missing token")
	}
	cfg, err := os.ReadFile(filepath.Join(app.DataDir, "configs/mihomo/config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(cfg)
	for _, want := range []string{"proxy-providers:", "msm_manual:", "https://example.com/a.yaml", "机场A", "机场1", "tproxy-port: 7896", "listen: 0.0.0.0:6666", "fake-ip-range: 28.0.0.1/8", "UrlTest: &UrlTest", "proxies: [DIRECT], include-all: true, include-all-proxies: true, include-all-providers: true", "name: 机场节点, type: select, proxies: [DIRECT], include-all: true, include-all-proxies: true, include-all-providers: true"} {
		if !strings.Contains(text, want) {
			t.Fatalf("mihomo config missing %q:\n%s", want, text)
		}
	}
	manualProvider, err := os.ReadFile(filepath.Join(app.DataDir, "configs/mihomo/proxy_providers/msm_manual.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	manualText := string(manualProvider)
	for _, want := range []string{"proxies:", "type: vless", "server: example.com", "port: 443", "uuid: c20c7f96-e1a1-4203-bffa-888ef04959fd", "reality-opts:", "public-key: abc", "short-id: \"123\"", "flow: xtls-rprx-vision"} {
		if !strings.Contains(manualText, want) {
			t.Fatalf("manual provider missing %q:\n%s", want, manualText)
		}
	}
	mosdnsConfig, err := os.ReadFile(filepath.Join(app.DataDir, "configs/mosdns/config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	mosdnsText := string(mosdnsConfig)
	for _, want := range []string{"sequence_6666", `listen: ":53"`, `listen: ":66"`, `listen: ":77"`, "listen: 127.0.0.1:5656"} {
		if !strings.Contains(mosdnsText, want) {
			t.Fatalf("mosdns config missing %q:\n%s", want, mosdnsText)
		}
	}
	mssbFiles := map[string][]string{
		"configs/mosdns/sub_config/forward_1.yaml":        {"udp://127.0.0.1:6666", `listen: ":2222"`},
		"configs/mosdns/sub_config/forward_nocn.yaml":     {`listen: ":3333"`},
		"configs/mosdns/sub_config/forward_nocn_ecs.yaml": {`listen: ":4444"`},
		"configs/mosdns/sub_config/for_singbox.yaml":      {`listen: ":7778"`, `listen: ":8888"`},
		"configs/mosdns/sub_config/forward_2.yaml":        {"127.0.0.1:5656", "entry: sequence_6666"},
	}
	for rel, wants := range mssbFiles {
		b, err := os.ReadFile(filepath.Join(app.DataDir, rel))
		if err != nil {
			t.Fatal(err)
		}
		got := string(b)
		for _, want := range wants {
			if !strings.Contains(got, want) {
				t.Fatalf("%s missing %q:\n%s", rel, want, got)
			}
		}
	}
	nft, err := os.ReadFile(filepath.Join(app.DataDir, "configs/network/network.nft"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(nft), `iifname { "lo", "eth0" }`) || !strings.Contains(string(nft), "tproxy to :7896") || !strings.Contains(string(nft), "redirect to :7877") || !strings.Contains(string(nft), "28.0.0.0/8") || !strings.Contains(string(nft), "set dns_ipv4 {\n    type ipv4_addr\n    flags interval") {
		t.Fatalf("nft template not rendered correctly:\n%s", nft)
	}
}

func TestSupportsAMD64v3AcceptsABMAsLZCNTCompat(t *testing.T) {
	cpuInfo := `processor : 0
vendor_id : GenuineIntel
model name : 12th Gen Intel(R) Core(TM) i9-12900HK
flags : fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush mmx fxsr sse sse2 ss ht syscall nx rdtscp lm constant_tsc rep_good nopl xtopology cpuid tsc_known_freq pni pclmulqdq ssse3 fma cx16 pcid sse4_1 sse4_2 movbe popcnt aes xsave avx f16c rdrand hypervisor lahf_lm abm 3dnowprefetch cpuid_fault invpcid_single pti ssbd ibrs ibpb stibp fsgsbase bmi1 avx2 bmi2 invpcid rdseed adx clflushopt xsaveopt arat umip
`
	if !supportsAMD64v3Flags(cpuInfo) {
		t.Fatal("expected Intel/KVM flags with abm to satisfy AMD64 v3 lzcnt requirement")
	}
}

func TestSupportsAMD64v3RequiresLZCNTOrABM(t *testing.T) {
	cpuInfo := `flags : avx avx2 bmi1 bmi2 fma movbe xsave`
	if supportsAMD64v3Flags(cpuInfo) {
		t.Fatal("expected AMD64 v3 detection to fail without lzcnt or abm")
	}
}

func TestSetupDefaultsMatchWebUIFakeIPRanges(t *testing.T) {
	var cfg SetupConfig
	cfg.defaults()
	if cfg.FakeIPRangeV4 != "28.0.0.0/8" {
		t.Fatalf("FakeIPRangeV4 default mismatch, got %q", cfg.FakeIPRangeV4)
	}
	if cfg.FakeIPRangeV6 != "f2b0::/18" {
		t.Fatalf("FakeIPRangeV6 default mismatch, got %q", cfg.FakeIPRangeV6)
	}
	if got := normalizeMihomoFakeIPv4Range(cfg.FakeIPRangeV4); got != "28.0.0.1/8" {
		t.Fatalf("mihomo fake-ip range normalization mismatch, got %q", got)
	}
	if got := fakeIPv4RouteCIDR(cfg.FakeIPRangeV4); got != "28.0.0.0/8" {
		t.Fatalf("IPv4 route CIDR mismatch, got %q", got)
	}
	if got := fakeIPv6RouteCIDR(""); got != "f2b0::/18" {
		t.Fatalf("IPv6 route CIDR default mismatch, got %q", got)
	}
}

func TestEnsureSetupProviderArtifactsBackfillsManualProvider(t *testing.T) {
	app := newTestApp(t)
	cfg := SetupConfig{
		SelectedInterface: "eth0",
		SubscriptionURLs:  "imm|https://example.com/imm.yaml",
		MihomoProxies:     "trojan://pass@example.org:443?sni=example.org#manual-node",
		ProxyCore:         "mihomo",
		MosDNSEnabled:     true,
	}
	cfg.defaults()
	if err := app.writeTextFile("configs/mihomo/config.yaml", "mode: rule\nproxy-providers: {}\n"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(app.DataDir, "configs/mihomo/proxy_providers/msm_manual.yaml")); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := app.ensureSetupProviderArtifacts(cfg); err != nil {
		t.Fatal(err)
	}
	manualProvider, err := os.ReadFile(filepath.Join(app.DataDir, "configs/mihomo/proxy_providers/msm_manual.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manualProvider), "name: manual-node") || !strings.Contains(string(manualProvider), "type: trojan") {
		t.Fatalf("manual provider not backfilled:\n%s", string(manualProvider))
	}
	config, err := os.ReadFile(filepath.Join(app.DataDir, "configs/mihomo/config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	configText := string(config)
	for _, want := range []string{"proxy-providers:", "msm_manual:", "imm", "https://example.com/imm.yaml", "./proxy_providers/imm.yaml"} {
		if !strings.Contains(configText, want) {
			t.Fatalf("mihomo provider section missing %q:\n%s", want, configText)
		}
	}
}

func TestNewLogEventLinesOnlyReturnsFreshNonDuplicateRows(t *testing.T) {
	previous := []string{"a", "b", "c"}
	if got := newLogEventLines(previous, []string{"a", "b", "c"}); len(got) != 0 {
		t.Fatalf("unchanged logs should not emit rows, got %#v", got)
	}
	got := newLogEventLines(previous, []string{"b", "c", "d"})
	if len(got) != 1 || got[0] != "d" {
		t.Fatalf("rotated tail should emit only new row, got %#v", got)
	}
	got = newLogEventLines(previous, []string{"x", "y"})
	if len(got) != 2 || got[0] != "x" || got[1] != "y" {
		t.Fatalf("truncated/replaced log should emit current rows, got %#v", got)
	}
}

func TestSafePathRejectsEscape(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.safePath("../etc/passwd"); err == nil {
		t.Fatal("expected path escape to fail")
	}
	if _, err := app.safePath("data/binaries/mihomo/mihomo"); err == nil {
		t.Fatal("expected non-config/log/backup path to fail")
	}
	if _, err := app.safePath("configs/mihomo/config.yaml"); err != nil {
		t.Fatalf("expected config path to pass: %v", err)
	}
}

func TestStaticSetupRouteServesFreshSPAEntry(t *testing.T) {
	app := newTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/setup", nil)
	res := httptest.NewRecorder()
	app.Router().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("setup route status=%d body=%s", res.Code, res.Body.String())
	}
	if got := res.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("setup route should disable HTML cache, got %q", got)
	}
	if body := res.Body.String(); !strings.Contains(body, "msm-free-spa-recovery") || !strings.Contains(body, "id=\"root\"") {
		t.Fatalf("setup route did not serve recovered SPA index:\n%s", body)
	}
}

func TestSetupNetworkInterfacesResponseMatchesExportedWebUI(t *testing.T) {
	app := newTestApp(t)
	res := requestJSON(t, app, http.MethodGet, "/api/v1/setup/network-interfaces", "", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("network interfaces status=%d body=%s", res.Code, res.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, ok := body["data"].([]any)
	if !ok {
		t.Fatalf("exported WebUI expects data array, got %#v", body["data"])
	}
	interfaces, ok := body["interfaces"].([]any)
	if !ok || len(interfaces) != len(data) {
		t.Fatalf("interfaces and data should both expose the same array: %#v", body)
	}
	if len(data) > 0 {
		item, ok := data[0].(map[string]any)
		if !ok || item["name"] == nil || item["ip"] == nil || item["speed"] == nil {
			t.Fatalf("interface rows should include name/ip/speed for setup UI: %#v", data[0])
		}
	}
}

func TestCompatibilityLayoutMatchesMSMTreeShape(t *testing.T) {
	app := newTestApp(t)
	for _, rel := range []string{
		"database/msm.db",
		"logs/msm.log",
		"logs/supervisor/supervisord.log",
		"configs/logs/mosdns.log",
		"configs/supervisor/supervisord.conf",
		"configs/supervisor/services/mihomo.ini",
		"configs/supervisor/services/mosdns.ini",
		"configs/network/history/.keep",
		"configs/mosdns/cache/.keep",
		"configs/mosdns/unpack/.keep",
		"configs/mihomo/config.yaml.backup",
	} {
		if _, err := os.Stat(filepath.Join(app.DataDir, rel)); err != nil {
			t.Fatalf("expected MSM-compatible layout path %s: %v", rel, err)
		}
	}
}

func TestStructuredMSMJSONLogsFormatLikeOriginal(t *testing.T) {
	app := newTestApp(t)
	app.LogInfo("app/app.go:114", "MSM 后端服务启动中...", nil)
	app.LogInfo("app/app.go:945", "gin", map[string]any{"message": `[GIN] 2026/05/31 - 12:58:40 | 200 | 1ms | 192.168.10.223 | GET      "/api/v1/logs/msm/stats"`})
	logs := requestJSON(t, app, http.MethodGet, "/api/v1/logs/msm?lines=10", tokenForRole(t, app, "admin"), nil)
	if logs.Code != http.StatusOK {
		t.Fatalf("logs status=%d body=%s", logs.Code, logs.Body.String())
	}
	body := logs.Body.String()
	if !strings.Contains(body, "[app/app.go:114] MSM 后端服务启动中...") || !strings.Contains(body, "[app/app.go:945] gin: [GIN]") {
		t.Fatalf("structured MSM JSON logs were not formatted for WebUI:\n%s", body)
	}
	if !strings.Contains(body, `"level":"info"`) || !strings.Contains(body, `"raw":"{`) {
		t.Fatalf("structured MSM JSON logs should preserve level and raw JSON:\n%s", body)
	}
}

func TestSetupDownloadSkipIfExists(t *testing.T) {
	app := newTestApp(t)
	target := app.componentTarget("mihomo")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/setup/download/mihomo?skip_if_exists=1", nil)
	res := httptest.NewRecorder()
	app.Router().ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("download status=%d body=%s", res.Code, res.Body.String())
	}
	if body := res.Body.String(); !strings.Contains(body, `"status":"skipped"`) {
		t.Fatalf("expected skipped download event, got:\n%s", body)
	}
}

func TestLicenseStatusIsFreeUnlocked(t *testing.T) {
	app := newTestApp(t)
	res := requestJSON(t, app, http.MethodGet, "/api/v1/license-activation/status", "", nil)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "unlocked") || !strings.Contains(res.Body.String(), "free") {
		t.Fatalf("license response not free/unlocked: %s", res.Body.String())
	}
}

func TestMosDNSObservabilityAndRuleCategories(t *testing.T) {
	app := newTestApp(t)
	token := tokenForRole(t, app, "admin")
	lines := []string{
		`{"query_name":"chatgpt.com","client_ip":"192.168.10.223","query_type":"A","domain_set":"my_fakeiprule","response_code":"NOERROR","duration_ms":0.08,"answers":["A: 28.0.0.218"]}`,
		`client_ip=192.168.10.235 query_name=google.com qtype=A rule=unmatched_rule rcode=NOERROR A: 142.250.192.142 (TTL: 5s) 0.06ms`,
		`client_ip=192.168.10.223 query_name=www.google.com qtype=HTTPS rule=BANHTTPS rcode=NOERROR 0.01ms`,
	}
	entries := mosDNSQueryEntries(lines)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	meta := mosDNSQueryMeta(entries)
	if !containsString(meta["query_types"].([]string), "HTTPS") {
		t.Fatalf("meta missing HTTPS: %#v", meta)
	}
	stats := mosDNSAuditStats(entries)
	if stats["fakeip_queries"].(int) == 0 || stats["blocked_queries"].(int) == 0 {
		t.Fatalf("stats should include fakeip and blocked queries: %#v", stats)
	}
	cache := mosDNSCacheSummary(entries)
	if cache["entries"].(int) == 0 {
		t.Fatalf("cache summary should include fakeip answer: %#v", cache)
	}
	res := requestJSON(t, app, http.MethodGet, "/api/v1/mosdns/rules/categories", token, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("categories status=%d body=%s", res.Code, res.Body.String())
	}
	for _, want := range []string{"redirect", "adguard", "online"} {
		if !strings.Contains(res.Body.String(), want) {
			t.Fatalf("categories missing %s: %s", want, res.Body.String())
		}
	}
}

func TestMosDNSRuleSourceManagementAndUpdate(t *testing.T) {
	app := newTestApp(t)
	token := tokenForRole(t, app, "admin")
	ruleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("domain:example.com\nkeyword:ads\n# ignored\nfull:test.example\n"))
	}))
	defer ruleServer.Close()

	create := requestJSON(t, app, http.MethodPost, "/api/v1/mosdns/rule-sets", token, map[string]any{
		"source_type": "adguard",
		"name":        "unit-adguard",
		"type":        "adguard",
		"url":         ruleServer.URL + "/rules.txt",
		"enabled":     true,
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create source status=%d body=%s", create.Code, create.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(create.Body.Bytes(), &created)
	source := created["data"].(map[string]any)
	id := source["id"].(string)
	list := requestJSON(t, app, http.MethodGet, "/api/v1/mosdns/rule-sets?source_type=adguard", token, nil)
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), "unit-adguard") || !strings.Contains(list.Body.String(), ruleServer.URL) {
		t.Fatalf("rule source list missing created source: status=%d body=%s", list.Code, list.Body.String())
	}
	update := requestJSON(t, app, http.MethodPost, "/api/v1/mosdns/rule-sets/"+id+"/update", token, nil)
	if update.Code != http.StatusOK || !strings.Contains(update.Body.String(), `"success":true`) || !strings.Contains(update.Body.String(), `"rule_count":3`) {
		t.Fatalf("rule source update failed: status=%d body=%s", update.Code, update.Body.String())
	}
	local := filepath.Join(app.DataDir, "configs/mosdns/adguard", id+".rules")
	b, err := os.ReadFile(local)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "domain:example.com") {
		t.Fatalf("downloaded rule file mismatch: %s", string(b))
	}
	cats := requestJSON(t, app, http.MethodGet, "/api/v1/mosdns/rules/categories", token, nil)
	if cats.Code != http.StatusOK || !strings.Contains(cats.Body.String(), `"id":"adguard"`) {
		t.Fatalf("categories missing adguard source count: %s", cats.Body.String())
	}
	del := requestJSON(t, app, http.MethodDelete, "/api/v1/mosdns/rule-sets/"+id+"?delete_file=true", token, nil)
	if del.Code != http.StatusOK {
		t.Fatalf("delete source status=%d body=%s", del.Code, del.Body.String())
	}
	if _, err := os.Stat(local); !os.IsNotExist(err) {
		t.Fatalf("expected local rule file deleted, err=%v", err)
	}
}

func TestMosDNS9099TakesPriorityForQueryLogsAndOverview(t *testing.T) {
	app := newTestApp(t)
	token := tokenForRole(t, app, "admin")
	if err := os.WriteFile(filepath.Join(app.DataDir, "logs/mosdns.out.log"), []byte(`client_ip=192.168.10.9 query_name=local.example qtype=A rule=local rcode=NOERROR`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/query-logs":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{
				{"query_name": "remote.example", "client_ip": "192.168.10.8", "query_type": "A", "domain_set": "my_fakeiprule", "response_code": "NOERROR", "answers": []string{"A: 28.0.0.2"}},
			}})
		case "/stats":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"query_count": 77}})
		case "/cache":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"entries": 12, "hit_rate": 88}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()
	app.setSetting("mosdns_api_endpoint", api.URL)

	logs := requestJSON(t, app, http.MethodGet, "/api/v1/mosdns/query-logs", token, nil)
	if logs.Code != http.StatusOK || !strings.Contains(logs.Body.String(), "remote.example") || strings.Contains(logs.Body.String(), "local.example") {
		t.Fatalf("query logs should prefer 9099: status=%d body=%s", logs.Code, logs.Body.String())
	}
	overview := requestJSON(t, app, http.MethodGet, "/api/v1/mosdns/overview", token, nil)
	if overview.Code != http.StatusOK || !strings.Contains(overview.Body.String(), `"source":"mosdns_9099"`) || !strings.Contains(overview.Body.String(), `"query_count":77`) || !strings.Contains(overview.Body.String(), `"upstream_stats":[`) || !strings.Contains(overview.Body.String(), `"detailed_cache"`) || !strings.Contains(overview.Body.String(), `"audit_stats"`) {
		t.Fatalf("overview should include 9099 stats: status=%d body=%s", overview.Code, overview.Body.String())
	}
}

func TestMosDNSClientMoveSyncsWhitelistFile(t *testing.T) {
	app := newTestApp(t)
	token := tokenForRole(t, app, "admin")
	create := requestJSON(t, app, http.MethodPost, "/api/v1/mosdns/clients", token, map[string]any{
		"ip": "192.168.10.88", "mac": "00:11:22:33:44:55", "hostname": "unit-client",
	})
	if create.Code != http.StatusOK {
		t.Fatalf("create client status=%d body=%s", create.Code, create.Body.String())
	}
	move := requestJSON(t, app, http.MethodPost, "/api/v1/mosdns/clients/192.168.10.88/move", token, map[string]string{"status": "allow"})
	if move.Code != http.StatusOK {
		t.Fatalf("move status=%d body=%s", move.Code, move.Body.String())
	}
	listFile, err := os.ReadFile(filepath.Join(app.DataDir, "configs/mosdns/client_ip.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(listFile), "192.168.10.88") {
		t.Fatalf("client_ip.txt missing moved client: %s", string(listFile))
	}
	clients := requestJSON(t, app, http.MethodGet, "/api/v1/mosdns/clients?status=allow", token, nil)
	if clients.Code != http.StatusOK || !strings.Contains(clients.Body.String(), `"status":"allow"`) {
		t.Fatalf("allow clients response mismatch: status=%d body=%s", clients.Code, clients.Body.String())
	}
	disable := requestJSON(t, app, http.MethodPost, "/api/v1/mosdns/clients/192.168.10.88/move", token, map[string]string{"status": "disabled"})
	if disable.Code != http.StatusOK {
		t.Fatalf("disable status=%d body=%s", disable.Code, disable.Body.String())
	}
	listFile, err = os.ReadFile(filepath.Join(app.DataDir, "configs/mosdns/client_ip.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(listFile), "192.168.10.88") {
		t.Fatalf("client_ip.txt should remove disabled client: %s", string(listFile))
	}
}

func TestMosDNSRuleReorderAndClear(t *testing.T) {
	app := newTestApp(t)
	token := tokenForRole(t, app, "admin")
	reorder := requestJSON(t, app, http.MethodPut, "/api/v1/mosdns/rules/whitelist/reorder", token, map[string]any{
		"items": []map[string]string{{"pattern": "domain:b.example"}, {"pattern": "domain:a.example"}},
	})
	if reorder.Code != http.StatusOK {
		t.Fatalf("reorder status=%d body=%s", reorder.Code, reorder.Body.String())
	}
	content, err := os.ReadFile(filepath.Join(app.DataDir, "configs/mosdns/rule/whitelist.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(content); !strings.Contains(got, "domain:b.example\ndomain:a.example\n") {
		t.Fatalf("rule order not persisted: %s", got)
	}
	clear := requestJSON(t, app, http.MethodDelete, "/api/v1/mosdns/rules/whitelist/all", token, nil)
	if clear.Code != http.StatusOK {
		t.Fatalf("clear status=%d body=%s", clear.Code, clear.Body.String())
	}
	content, err = os.ReadFile(filepath.Join(app.DataDir, "configs/mosdns/rule/whitelist.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(content)) != "" {
		t.Fatalf("rule clear not persisted: %s", string(content))
	}
}

func TestMosDNSRoutingTaskGeneratesRuleFiles(t *testing.T) {
	app := newTestApp(t)
	token := tokenForRole(t, app, "admin")
	api := httptest.NewServer(http.NotFoundHandler())
	defer api.Close()
	app.setSetting("mosdns_api_endpoint", api.URL)
	lines := strings.Join([]string{
		`{"query_name":"fake.example","client_ip":"192.168.10.2","query_type":"A","domain_set":"my_fakeiprule","response_code":"NOERROR","answers":["A: 28.0.0.9"]}`,
		`client_ip=192.168.10.3 query_name=real.example qtype=A rule=unmatched_rule rcode=NOERROR A: 223.5.5.5`,
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(app.DataDir, "logs/mosdns.out.log"), []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}
	start := requestJSON(t, app, http.MethodPost, "/api/v1/mosdns/system/routing/start", token, nil)
	if start.Code != http.StatusOK || !strings.Contains(start.Body.String(), `"status":"completed"`) {
		t.Fatalf("routing start status=%d body=%s", start.Code, start.Body.String())
	}
	fakeRules, err := os.ReadFile(filepath.Join(app.DataDir, "configs/mosdns/gen/fakeiprule.txt"))
	if err != nil {
		t.Fatal(err)
	}
	realRules, err := os.ReadFile(filepath.Join(app.DataDir, "configs/mosdns/gen/realiprule.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(fakeRules), "domain:fake.example") || !strings.Contains(string(realRules), "domain:real.example") {
		t.Fatalf("routing files mismatch fake=%s real=%s", string(fakeRules), string(realRules))
	}
}

func TestMosDNSConfigSaveCreatesHistory(t *testing.T) {
	app := newTestApp(t)
	token := tokenForRole(t, app, "admin")
	put := requestJSON(t, app, http.MethodPut, "/api/v1/mosdns/config/file", token, map[string]string{
		"path": "configs/mosdns/config.yaml", "content": "log:\n  level: info\n",
	})
	if put.Code != http.StatusOK || !strings.Contains(put.Body.String(), "restart_required") {
		t.Fatalf("config put status=%d body=%s", put.Code, put.Body.String())
	}
	var count int
	if err := app.DB.QueryRow(`select count(*) from config_histories where service='mosdns' and file_path='configs/mosdns/config.yaml'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected MosDNS config save to create history")
	}
}

func TestMihomoControllerBackedOverviewAndTraffic(t *testing.T) {
	app := newTestApp(t)
	token := tokenForRole(t, app, "admin")
	api := newFakeMihomoController(t)
	defer api.Close()
	app.setSetting("mihomo_controller_endpoint", api.URL)

	overview := requestJSON(t, app, http.MethodGet, "/api/v1/mihomo/overview", token, nil)
	if overview.Code != http.StatusOK || !strings.Contains(overview.Body.String(), `"version":"v1.19.9"`) || !strings.Contains(overview.Body.String(), `"connection_count":2`) || !strings.Contains(overview.Body.String(), `"proxy_provider_count":1`) {
		t.Fatalf("mihomo overview should use controller data: status=%d body=%s", overview.Code, overview.Body.String())
	}
	traffic := requestJSON(t, app, http.MethodGet, "/api/v1/mihomo/traffic", token, nil)
	if traffic.Code != http.StatusOK || !strings.Contains(traffic.Body.String(), `"download":4096`) || !strings.Contains(traffic.Body.String(), `"upload":1024`) {
		t.Fatalf("mihomo traffic mismatch: status=%d body=%s", traffic.Code, traffic.Body.String())
	}
	stats := requestJSON(t, app, http.MethodGet, "/api/v1/mihomo/stats", token, nil)
	if stats.Code != http.StatusOK || !strings.Contains(stats.Body.String(), `"downloadSpeed":4096`) || !strings.Contains(stats.Body.String(), `"uploadSpeed":1024`) || !strings.Contains(stats.Body.String(), `"downloadTotal":8192`) || !strings.Contains(stats.Body.String(), `"activeConnections":2`) {
		t.Fatalf("mihomo stats should expose overview-compatible counters: status=%d body=%s", stats.Code, stats.Body.String())
	}
	proxy := requestJSON(t, app, http.MethodGet, "/api/v1/proxy/overview", token, nil)
	if proxy.Code != http.StatusOK || !strings.Contains(proxy.Body.String(), `"core":"mihomo"`) || !strings.Contains(proxy.Body.String(), `"mode":"rule"`) {
		t.Fatalf("proxy overview mismatch: status=%d body=%s", proxy.Code, proxy.Body.String())
	}
}

func TestNetworkInfoProvidesOverviewLocalIPShape(t *testing.T) {
	app := newTestApp(t)
	token := tokenForRole(t, app, "admin")
	if err := os.MkdirAll(filepath.Join(app.DataDir, "configs/network"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app.DataDir, "configs/network/network.yaml"), []byte("interface: lo\nmode: tproxy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := requestJSON(t, app, http.MethodGet, "/api/v1/network/info", token, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("network info status=%d body=%s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), `"data"`) || !strings.Contains(res.Body.String(), `"localIP"`) || !strings.Contains(res.Body.String(), `"interfaces"`) || !strings.Contains(res.Body.String(), `"selected_interface":"lo"`) {
		t.Fatalf("network info should provide frontend overview shape: %s", res.Body.String())
	}
}

func TestMihomoConnectionsProxiesRulesAndClose(t *testing.T) {
	app := newTestApp(t)
	token := tokenForRole(t, app, "admin")
	api := newFakeMihomoController(t)
	defer api.Close()
	app.setSetting("mihomo_controller_endpoint", api.URL)

	connections := requestJSON(t, app, http.MethodGet, "/api/v1/mihomo/connections?search=google&page=1&page_size=1", token, nil)
	if connections.Code != http.StatusOK || !strings.Contains(connections.Body.String(), `"total":1`) || !strings.Contains(connections.Body.String(), `"host":"google.com"`) {
		t.Fatalf("connection filtering mismatch: status=%d body=%s", connections.Code, connections.Body.String())
	}
	closeAll := requestJSON(t, app, http.MethodDelete, "/api/v1/mihomo/connections", token, nil)
	if closeAll.Code != http.StatusOK || !strings.Contains(closeAll.Body.String(), `"success":true`) {
		t.Fatalf("connection close failed: status=%d body=%s", closeAll.Code, closeAll.Body.String())
	}
	proxies := requestJSON(t, app, http.MethodGet, "/api/v1/mihomo/proxies?search=proxy", token, nil)
	if proxies.Code != http.StatusOK || !strings.Contains(proxies.Body.String(), `"name":"Proxy"`) || !strings.Contains(proxies.Body.String(), `"name":"proxy-a"`) || !strings.Contains(proxies.Body.String(), `"proxy_list"`) || !strings.Contains(proxies.Body.String(), `"Proxy":{`) {
		t.Fatalf("proxy list mismatch: status=%d body=%s", proxies.Code, proxies.Body.String())
	}
	selectProxy := requestJSON(t, app, http.MethodPut, "/api/v1/mihomo/proxies/Proxy", token, map[string]string{"name": "proxy-a"})
	if selectProxy.Code != http.StatusOK || !strings.Contains(selectProxy.Body.String(), `"updated":true`) {
		t.Fatalf("proxy select mismatch: status=%d body=%s", selectProxy.Code, selectProxy.Body.String())
	}
	rules := requestJSON(t, app, http.MethodGet, "/api/v1/mihomo/rules?type=DOMAIN-SUFFIX", token, nil)
	if rules.Code != http.StatusOK || !strings.Contains(rules.Body.String(), `"payload":"google.com"`) || strings.Contains(rules.Body.String(), "geoip") {
		t.Fatalf("rules filtering mismatch: status=%d body=%s", rules.Code, rules.Body.String())
	}
}

func TestMihomoProviderConfigManagementCreatesHistory(t *testing.T) {
	app := newTestApp(t)
	token := tokenForRole(t, app, "admin")
	api := newFakeMihomoController(t)
	defer api.Close()
	app.setSetting("mihomo_controller_endpoint", api.URL)

	put := requestJSON(t, app, http.MethodPut, "/api/v1/mihomo/proxy-providers/airport", token, map[string]any{
		"url":      "https://example.com/sub.yaml",
		"interval": 3600,
	})
	if put.Code != http.StatusOK || !strings.Contains(put.Body.String(), `"restart_required":true`) {
		t.Fatalf("proxy provider put status=%d body=%s", put.Code, put.Body.String())
	}
	cfg, err := os.ReadFile(filepath.Join(app.DataDir, "configs/mihomo/config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), "airport:") || !strings.Contains(string(cfg), "https://example.com/sub.yaml") {
		t.Fatalf("proxy provider not persisted:\n%s", string(cfg))
	}
	update := requestJSON(t, app, http.MethodPost, "/api/v1/mihomo/proxy-providers/airport/update", token, nil)
	if update.Code != http.StatusOK || !strings.Contains(update.Body.String(), `"healthcheck":true`) {
		t.Fatalf("proxy provider update status=%d body=%s", update.Code, update.Body.String())
	}
	rulePut := requestJSON(t, app, http.MethodPut, "/api/v1/mihomo/rule-providers/directlist", token, map[string]any{
		"url":      "https://example.com/rules.yaml",
		"behavior": "domain",
	})
	if rulePut.Code != http.StatusOK || !strings.Contains(rulePut.Body.String(), `"restart_required":true`) {
		t.Fatalf("rule provider put status=%d body=%s", rulePut.Code, rulePut.Body.String())
	}
	list := requestJSON(t, app, http.MethodGet, "/api/v1/mihomo/providers", token, nil)
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"name":"airport"`) || !strings.Contains(list.Body.String(), `"name":"directlist"`) {
		t.Fatalf("providers list mismatch: status=%d body=%s", list.Code, list.Body.String())
	}
	var count int
	if err := app.DB.QueryRow(`select count(*) from config_histories where service='mihomo' and file_path='configs/mihomo/config.yaml'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected Mihomo provider config updates to create history")
	}
	del := requestJSON(t, app, http.MethodDelete, "/api/v1/mihomo/proxy-providers/airport", token, nil)
	if del.Code != http.StatusOK || !strings.Contains(del.Body.String(), `"restart_required":true`) {
		t.Fatalf("proxy provider delete status=%d body=%s", del.Code, del.Body.String())
	}
}

func TestMihomoConfigAndLogPanelCompatibility(t *testing.T) {
	app := newTestApp(t)
	token := tokenForRole(t, app, "admin")
	logPath := filepath.Join(app.DataDir, "logs/mihomo.out.log")
	if err := os.WriteFile(logPath, []byte("[info] started\n[warning] proxy provider failed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	logs := requestJSON(t, app, http.MethodGet, "/api/v1/mihomo/logs?level=warn&search=provider", token, nil)
	if logs.Code != http.StatusOK || !strings.Contains(logs.Body.String(), "proxy provider failed") || strings.Contains(logs.Body.String(), "started") {
		t.Fatalf("mihomo logs filtering mismatch: status=%d body=%s", logs.Code, logs.Body.String())
	}
	put := requestJSON(t, app, http.MethodPut, "/api/v1/mihomo/config/config.yaml", token, map[string]any{"content": "mode: rule\nmixed-port: 7892\n"})
	if put.Code != http.StatusOK || !strings.Contains(put.Body.String(), `"restart_required":true`) {
		t.Fatalf("mihomo config path put status=%d body=%s", put.Code, put.Body.String())
	}
	var count int
	if err := app.DB.QueryRow(`select count(*) from config_histories where service='mihomo' and file_path='configs/mihomo/config.yaml'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("expected Mihomo config path save to create history")
	}
	files := requestJSON(t, app, http.MethodGet, "/api/v1/mihomo/config/files", token, nil)
	if files.Code != http.StatusOK || !strings.Contains(files.Body.String(), "config.yaml") {
		t.Fatalf("mihomo config files mismatch: status=%d body=%s", files.Code, files.Body.String())
	}
}

func TestRolePermissionsMatchMSMModel(t *testing.T) {
	app := newTestApp(t)
	operator := tokenForRole(t, app, "operator")
	viewer := tokenForRole(t, app, "viewer")
	guest := tokenForRole(t, app, "guest")
	if res := requestJSON(t, app, http.MethodPut, "/api/v1/settings", operator, map[string]string{"theme": "dark"}); res.Code != http.StatusForbidden {
		t.Fatalf("operator should not update settings, status=%d body=%s", res.Code, res.Body.String())
	}
	if res := requestJSON(t, app, http.MethodGet, "/api/v1/monitor/system", viewer, nil); res.Code != http.StatusOK {
		t.Fatalf("viewer should read monitor, status=%d body=%s", res.Code, res.Body.String())
	}
	if res := requestJSON(t, app, http.MethodPut, "/api/v1/config/file", viewer, map[string]string{"path": "configs/mihomo/config.yaml", "content": ""}); res.Code != http.StatusForbidden {
		t.Fatalf("viewer should not update config, status=%d body=%s", res.Code, res.Body.String())
	}
	if res := requestJSON(t, app, http.MethodGet, "/api/v1/logs/mihomo", viewer, nil); res.Code != http.StatusForbidden {
		t.Fatalf("viewer should not read logs, status=%d body=%s", res.Code, res.Body.String())
	}
	if res := requestJSON(t, app, http.MethodGet, "/api/v1/services", guest, nil); res.Code != http.StatusForbidden {
		t.Fatalf("guest should not read service status, status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestConfigCompareBackupAndDiagnostics(t *testing.T) {
	app := newTestApp(t)
	token := tokenForRole(t, app, "admin")
	create := requestJSON(t, app, http.MethodPost, "/api/v1/history", token, map[string]any{"path": "configs/mihomo/config.yaml", "content": "a: 1\n", "comment": "old"})
	if create.Code != http.StatusCreated {
		t.Fatalf("history create status=%d body=%s", create.Code, create.Body.String())
	}
	if err := app.writeTextFile("configs/mihomo/config.yaml", "a: 2\n"); err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	_ = json.Unmarshal(create.Body.Bytes(), &body)
	compare := requestJSON(t, app, http.MethodGet, "/api/v1/history/compare?from="+strconvID(body["id"])+"&path=configs/mihomo/config.yaml", token, nil)
	if compare.Code != http.StatusOK || !strings.Contains(compare.Body.String(), "-a: 1") || !strings.Contains(compare.Body.String(), "+a: 2") {
		t.Fatalf("compare did not return useful diff: status=%d body=%s", compare.Code, compare.Body.String())
	}
	backup := requestJSON(t, app, http.MethodPost, "/api/v1/config/backup", token, map[string]any{})
	if backup.Code != http.StatusOK || !strings.Contains(backup.Body.String(), ".zip") {
		t.Fatalf("backup status=%d body=%s", backup.Code, backup.Body.String())
	}
	diag := requestJSON(t, app, http.MethodGet, "/api/v1/system/diagnostics", token, nil)
	if diag.Code != http.StatusOK || !strings.Contains(diag.Body.String(), "配置目录") || !strings.Contains(diag.Body.String(), "端口占用") {
		t.Fatalf("diagnostics incomplete: status=%d body=%s", diag.Code, diag.Body.String())
	}
	var diagBody map[string]any
	if err := json.Unmarshal(diag.Body.Bytes(), &diagBody); err != nil {
		t.Fatal(err)
	}
	if diagBody["overall_status"] == "" || diagBody["system_info"] == nil {
		t.Fatalf("diagnostics should expose direct system page fields: %#v", diagBody)
	}
	checks, ok := diagBody["checks"].([]any)
	if !ok || len(checks) == 0 {
		t.Fatalf("diagnostics checks should be a non-empty array: %#v", diagBody["checks"])
	}
	first, ok := checks[0].(map[string]any)
	if !ok {
		t.Fatalf("diagnostics check row should be an object: %#v", checks[0])
	}
	if _, ok := first["details"].(string); !ok {
		t.Fatalf("diagnostics details must be a string for React rendering: %#v", first["details"])
	}
	switch first["status"] {
	case "success", "warning", "error":
	default:
		t.Fatalf("diagnostics status should use WebUI enum, got %#v", first["status"])
	}
}

func TestBasicManagementServiceLogsAndConfigAPIs(t *testing.T) {
	app := newTestApp(t)
	token := tokenForRole(t, app, "admin")
	services := requestJSON(t, app, http.MethodGet, "/api/v1/services", token, nil)
	if services.Code != http.StatusOK || !strings.Contains(services.Body.String(), "desired_enabled") || !strings.Contains(services.Body.String(), "health_ports") {
		t.Fatalf("services compatibility response incomplete: status=%d body=%s", services.Code, services.Body.String())
	}
	stopAll := requestJSON(t, app, http.MethodPost, "/api/v1/services/stop-all?wait=1&timeout_ms=100", token, nil)
	if stopAll.Code != http.StatusOK || !strings.Contains(stopAll.Body.String(), "mosdns") || !strings.Contains(stopAll.Body.String(), "mihomo") {
		t.Fatalf("stop-all response incomplete: status=%d body=%s", stopAll.Code, stopAll.Body.String())
	}
	if err := os.WriteFile(filepath.Join(app.DataDir, "logs/mihomo.out.log"), []byte("[info] started\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app.DataDir, "logs/mihomo.err.log"), []byte("[warn] proxy provider failed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	logs := requestJSON(t, app, http.MethodGet, "/api/v1/logs/mihomo?level=warn&search=provider&page=1&page_size=1", token, nil)
	if logs.Code != http.StatusOK || !strings.Contains(logs.Body.String(), "proxy provider failed") || !strings.Contains(logs.Body.String(), "pagination") {
		t.Fatalf("logs filtering response mismatch: status=%d body=%s", logs.Code, logs.Body.String())
	}
	download := requestJSON(t, app, http.MethodGet, "/api/v1/logs/mihomo/download?format=zip", token, nil)
	if download.Code != http.StatusOK || !strings.Contains(download.Header().Get("Content-Type"), "application/zip") {
		t.Fatalf("logs zip download mismatch: status=%d content-type=%s", download.Code, download.Header().Get("Content-Type"))
	}
	clear := requestJSON(t, app, http.MethodDelete, "/api/v1/logs/mihomo", token, nil)
	if clear.Code != http.StatusOK {
		t.Fatalf("logs clear failed: status=%d body=%s", clear.Code, clear.Body.String())
	}
	if b, _ := os.ReadFile(filepath.Join(app.DataDir, "logs/mihomo.err.log")); len(b) != 0 {
		t.Fatalf("stderr log should be cleared, got %q", string(b))
	}
	if err := os.WriteFile(filepath.Join(app.DataDir, "logs/msm.log"), []byte("[info] msm web request\n"), 0644); err != nil {
		t.Fatal(err)
	}
	defaultLogs := requestJSON(t, app, http.MethodGet, "/api/v1/logs?lines=20", token, nil)
	if defaultLogs.Code != http.StatusOK || !strings.Contains(defaultLogs.Body.String(), `"service":"msm"`) || !strings.Contains(defaultLogs.Body.String(), "msm web request") {
		t.Fatalf("default logs route should return msm logs: status=%d body=%s", defaultLogs.Code, defaultLogs.Body.String())
	}
	stats := requestJSON(t, app, http.MethodGet, "/api/v1/logs/msm/stats", token, nil)
	if stats.Code != http.StatusOK || !strings.Contains(stats.Body.String(), `"total":`) || !strings.Contains(stats.Body.String(), `"service":"msm"`) {
		t.Fatalf("msm log stats should include default app log lines: status=%d body=%s", stats.Code, stats.Body.String())
	}
	put := requestJSON(t, app, http.MethodPut, "/api/v1/config/file", token, map[string]any{"path": "configs/mihomo/config.yaml", "content": "mode: rule\n", "comment": "test save"})
	if put.Code != http.StatusOK || !strings.Contains(put.Body.String(), "history_id") || !strings.Contains(put.Body.String(), "restart_required") {
		t.Fatalf("config save response incomplete: status=%d body=%s", put.Code, put.Body.String())
	}
	history := requestJSON(t, app, http.MethodGet, "/api/v1/history?service=mihomo&page=1&page_size=1", token, nil)
	if history.Code != http.StatusOK || !strings.Contains(history.Body.String(), "pagination") || !strings.Contains(history.Body.String(), "configs/mihomo/config.yaml") {
		t.Fatalf("history filter response mismatch: status=%d body=%s", history.Code, history.Body.String())
	}
}

func TestBasicManagementUsersTokensSettingsAndDiagnostics(t *testing.T) {
	app := newTestApp(t)
	token := tokenForRole(t, app, "admin")
	badRole := requestJSON(t, app, http.MethodPost, "/api/v1/users", token, map[string]any{"username": "badrole", "password": "12345678", "role": "root"})
	if badRole.Code != http.StatusBadRequest {
		t.Fatalf("invalid role should fail, status=%d body=%s", badRole.Code, badRole.Body.String())
	}
	create := requestJSON(t, app, http.MethodPost, "/api/v1/users", token, map[string]any{"username": "op1", "password": "12345678", "role": "operator", "display_name": "Operator"})
	if create.Code != http.StatusCreated {
		t.Fatalf("user create failed: status=%d body=%s", create.Code, create.Body.String())
	}
	list := requestJSON(t, app, http.MethodGet, "/api/v1/users?search=op1&role=operator&page=1&page_size=5", token, nil)
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), "pagination") || !strings.Contains(list.Body.String(), "op1") {
		t.Fatalf("user list filters mismatch: status=%d body=%s", list.Code, list.Body.String())
	}
	selfDelete := requestJSON(t, app, http.MethodDelete, "/api/v1/users/1", token, nil)
	if selfDelete.Code != http.StatusBadRequest || !strings.Contains(selfDelete.Body.String(), "self_delete") {
		t.Fatalf("self delete should be blocked: status=%d body=%s", selfDelete.Code, selfDelete.Body.String())
	}
	tokenCreate := requestJSON(t, app, http.MethodPost, "/api/v1/api-tokens", token, map[string]any{"name": "test-token", "expires_in": 60})
	if tokenCreate.Code != http.StatusOK || !strings.Contains(tokenCreate.Body.String(), "msmf_") {
		t.Fatalf("api token create failed: status=%d body=%s", tokenCreate.Code, tokenCreate.Body.String())
	}
	audit := requestJSON(t, app, http.MethodGet, "/api/v1/audit-logs?action=user.create", token, nil)
	if audit.Code != http.StatusOK || !strings.Contains(audit.Body.String(), "user.create") || !strings.Contains(audit.Body.String(), "pagination") {
		t.Fatalf("audit logs response mismatch: status=%d body=%s", audit.Code, audit.Body.String())
	}
	appearancePut := requestJSON(t, app, http.MethodPut, "/api/v1/settings/appearance", token, map[string]any{"theme": "dark", "language": "zh-CN"})
	if appearancePut.Code != http.StatusOK {
		t.Fatalf("appearance put failed: status=%d body=%s", appearancePut.Code, appearancePut.Body.String())
	}
	appearance := requestJSON(t, app, http.MethodGet, "/api/v1/settings/appearance", token, nil)
	if appearance.Code != http.StatusOK || !strings.Contains(appearance.Body.String(), "dark") {
		t.Fatalf("appearance get mismatch: status=%d body=%s", appearance.Code, appearance.Body.String())
	}
	diagRun := requestJSON(t, app, http.MethodPost, "/api/v1/system/diagnostics/run", token, nil)
	if diagRun.Code != http.StatusOK || !strings.Contains(diagRun.Body.String(), "服务状态") || !strings.Contains(diagRun.Body.String(), "recent_errors") {
		t.Fatalf("diagnostics run incomplete: status=%d body=%s", diagRun.Code, diagRun.Body.String())
	}
	diagDownload := requestJSON(t, app, http.MethodGet, "/api/v1/system/diagnostics/download", token, nil)
	if diagDownload.Code != http.StatusOK || !strings.Contains(diagDownload.Header().Get("Content-Disposition"), "diagnostics") {
		t.Fatalf("diagnostics download mismatch: status=%d headers=%v", diagDownload.Code, diagDownload.Header())
	}
	setup := requestJSON(t, app, http.MethodPut, "/api/v1/setup/config", token, map[string]any{
		"username":           "root",
		"selected_interface": "eth0",
		"subscription_urls":  "imm|https://example.com/sub.yaml",
		"proxy_core":         "mihomo",
		"mos_dns_enabled":    true,
		"auto_set_dns":       false,
		"enable_ipv6":        false,
		"mihomo_proxies":     "trojan://pass@example.org:443?sni=example.org#manual-node",
	})
	if setup.Code != http.StatusOK || !strings.Contains(setup.Body.String(), "network_reapply_required") || !strings.Contains(setup.Body.String(), "download_component") {
		t.Fatalf("setup config compatibility response mismatch: status=%d body=%s", setup.Code, setup.Body.String())
	}
	if !strings.Contains(setup.Body.String(), `"proxy_core":"mihomo"`) || !strings.Contains(setup.Body.String(), `"enable_ipv6":false`) {
		t.Fatalf("setup config response should expose snake_case WebUI fields: status=%d body=%s", setup.Code, setup.Body.String())
	}
	setupGet := requestJSON(t, app, http.MethodGet, "/api/v1/setup/config", token, nil)
	if setupGet.Code != http.StatusOK {
		t.Fatalf("setup config get failed: status=%d body=%s", setupGet.Code, setupGet.Body.String())
	}
	var setupBody map[string]any
	if err := json.Unmarshal(setupGet.Body.Bytes(), &setupBody); err != nil {
		t.Fatal(err)
	}
	data, ok := setupBody["data"].(map[string]any)
	if !ok {
		t.Fatalf("setup config get should expose data object: %#v", setupBody["data"])
	}
	if data["selected_interface"] != "eth0" || data["proxy_core"] != "mihomo" || data["enable_ipv6"] != false {
		t.Fatalf("setup config data mismatch: %#v", data)
	}
	if !strings.Contains(fmtAny(data["subscription_urls"]), "imm|https://example.com/sub.yaml") || !strings.Contains(fmtAny(data["mihomo_proxies"]), "manual-node") {
		t.Fatalf("setup config should preserve subscription and manual node text: %#v", data)
	}
	manualProvider, err := os.ReadFile(filepath.Join(app.DataDir, "configs/mihomo/proxy_providers/msm_manual.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manualProvider), "name: manual-node") || !strings.Contains(string(manualProvider), "type: trojan") {
		t.Fatalf("manual proxy provider was not generated from settings setup config:\n%s", string(manualProvider))
	}
}

func newFakeMihomoController(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/version":
			_ = json.NewEncoder(w).Encode(map[string]any{"version": "v1.19.9", "premium": "v1.19.9"})
		case "/configs":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"mode": "rule", "log-level": "info", "allow-lan": true,
				"mixed-port": 7892, "redir-port": 7877, "tproxy-port": 7896, "external-controller": ":9090",
			})
		case "/traffic":
			_ = json.NewEncoder(w).Encode(map[string]any{"up": 1024, "down": 4096})
		case "/connections":
			if r.Method == http.MethodDelete {
				_ = json.NewEncoder(w).Encode(map[string]any{"closed": true})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"downloadTotal": 8192, "uploadTotal": 2048,
				"connections": []map[string]any{
					{
						"id": "c1", "rule": "RuleSet", "rulePayload": "google", "chains": []string{"Proxy", "proxy-a"}, "download": 100, "upload": 10, "start": "2026-05-30T10:00:00Z",
						"metadata": map[string]any{"host": "google.com", "network": "tcp", "type": "TProxy", "sourceIP": "192.168.10.2", "destinationIP": "28.0.0.8", "destinationPort": "443"},
					},
					{
						"id": "c2", "rule": "DIRECT", "chains": []string{"DIRECT"}, "download": 50, "upload": 5,
						"metadata": map[string]any{"host": "local.lan", "network": "udp", "type": "Redirect", "sourceIP": "192.168.10.3", "destinationIP": "192.168.10.1", "destinationPort": "53"},
					},
				},
			})
		case "/proxies":
			_ = json.NewEncoder(w).Encode(map[string]any{"proxies": map[string]any{
				"Proxy":   map[string]any{"name": "Proxy", "type": "Selector", "now": "proxy-a", "all": []string{"proxy-a", "DIRECT"}},
				"proxy-a": map[string]any{"name": "proxy-a", "type": "Shadowsocks", "udp": true, "history": []map[string]any{{"delay": 42}}},
				"DIRECT":  map[string]any{"name": "DIRECT", "type": "Direct"},
			}})
		case "/proxies/Proxy":
			_ = json.NewEncoder(w).Encode(map[string]any{"updated": true})
		case "/rules":
			_ = json.NewEncoder(w).Encode(map[string]any{"rules": []any{
				map[string]any{"type": "DOMAIN-SUFFIX", "payload": "google.com", "proxy": "Proxy", "provider": "geosite"},
				map[string]any{"type": "GEOIP", "payload": "CN", "proxy": "DIRECT"},
			}})
		case "/providers/proxies":
			_ = json.NewEncoder(w).Encode(map[string]any{"providers": map[string]any{
				"airport": map[string]any{"name": "airport", "vehicleType": "HTTP", "updatedAt": "2026-05-30T10:00:00Z", "proxies": []any{}},
			}})
		case "/providers/proxies/airport":
			_ = json.NewEncoder(w).Encode(map[string]any{"name": "airport", "vehicleType": "HTTP", "updatedAt": "2026-05-30T10:00:00Z", "proxies": []any{}})
		case "/providers/proxies/airport/healthcheck":
			_ = json.NewEncoder(w).Encode(map[string]any{"healthcheck": true, "updatedAt": "2026-05-30T10:00:00Z"})
		case "/providers/rules":
			_ = json.NewEncoder(w).Encode(map[string]any{"providers": map[string]any{
				"directlist": map[string]any{"name": "directlist", "vehicleType": "HTTP", "updatedAt": "2026-05-30T10:00:00Z", "rules": []any{}},
			}})
		case "/providers/rules/directlist":
			_ = json.NewEncoder(w).Encode(map[string]any{"name": "directlist", "vehicleType": "HTTP", "updatedAt": "2026-05-30T10:00:00Z", "rules": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	app, err := New(Options{DataDir: t.TempDir(), Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	if err := app.EnsureBaseLayout(); err != nil {
		t.Fatal(err)
	}
	return app
}

func tokenForRole(t *testing.T, app *App, role string) string {
	t.Helper()
	now := time.Now()
	res, err := app.DB.Exec(`insert into users(username,password,role,is_active,created_at,updated_at) values(?,?,?,?,?,?)`, role+"_user", "x", role, true, now, now)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	u, err := app.userByID(id)
	if err != nil {
		t.Fatal(err)
	}
	token, err := app.makeToken(u, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func strconvID(v any) string {
	switch x := v.(type) {
	case float64:
		return strconv.Itoa(int(x))
	case int:
		return strconv.Itoa(x)
	case string:
		return x
	default:
		return ""
	}
}

func requestJSON(t *testing.T, app *App, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var rbody *bytes.Reader
	if body == nil {
		rbody = bytes.NewReader(nil)
	} else {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rbody = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rbody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	app.Router().ServeHTTP(res, req)
	return res
}
