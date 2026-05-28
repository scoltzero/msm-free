package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	for _, want := range []string{"proxy-providers:", "https://example.com/a.yaml", "机场A", "机场1", "tproxy-port: 7896", "listen: 0.0.0.0:6666", "fake-ip-range: 28.0.0.1/8"} {
		if !strings.Contains(text, want) {
			t.Fatalf("mihomo config missing %q:\n%s", want, text)
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
		"configs/mosdns/sub_config/for_singbox.yaml":      {`listen: ":7777"`, `listen: ":8888"`},
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
	if !strings.Contains(string(nft), `iifname { "lo", "eth0" }`) || !strings.Contains(string(nft), "tproxy to :7896") || !strings.Contains(string(nft), "redirect to :7877") || !strings.Contains(string(nft), "28.0.0.0/8") {
		t.Fatalf("nft template not rendered correctly:\n%s", nft)
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
