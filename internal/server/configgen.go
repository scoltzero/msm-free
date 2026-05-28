package server

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type SetupConfig struct {
	Username                 string `json:"username"`
	Password                 string `json:"password"`
	ConfirmPassword          string `json:"confirmPassword"`
	Email                    string `json:"email"`
	Timezone                 string `json:"timezone"`
	WebPort                  string `json:"webPort"`
	AMD64v3Enabled           bool   `json:"amd64v3_enabled"`
	SelectedInterface        string `json:"selected_interface"`
	MihomoCoreType           string `json:"mihomo_core_type"`
	AutoSetDNS               bool   `json:"auto_set_dns"`
	DNSOn                    string `json:"dns_on"`
	DNSOff                   string `json:"dns_off"`
	EnableIPv6               bool   `json:"enableIPv6"`
	FakeIPRangeV4            string `json:"fakeIPRangeV4"`
	FakeIPRangeV6            string `json:"fakeIPRangeV6"`
	LinuxProxyMode           string `json:"linux_proxy_mode"`
	NFTProxyPolicy           string `json:"nft_proxy_policy"`
	ProxyCore                string `json:"proxyCore"`
	MosDNSEnabled            bool   `json:"mosdnsEnabled"`
	SubscriptionURLs         string `json:"subscription_urls"`
	MihomoProxies            string `json:"mihomo_proxies"`
	GitHubProxyEnabled       bool   `json:"github_proxy_enabled"`
	GitHubHTTPSProxy         string `json:"github_https_proxy"`
	GitHubHTTPProxy          string `json:"github_http_proxy"`
	GitHubSocks5Proxy        string `json:"github_socks5_proxy"`
	GitHubAcceleratorEnabled bool   `json:"github_accelerator_enabled"`
	GitHubAcceleratorURL     string `json:"github_accelerator_url"`
}

func (c *SetupConfig) defaults() {
	if c.Timezone == "" {
		c.Timezone = "Asia/Shanghai"
	}
	if c.WebPort == "" {
		c.WebPort = "7777"
	}
	if c.MihomoCoreType == "" {
		c.MihomoCoreType = "meta"
	}
	if c.DNSOn == "" {
		c.DNSOn = "127.0.0.1"
	}
	if c.DNSOff == "" {
		c.DNSOff = "223.5.5.5"
	}
	if c.FakeIPRangeV4 == "" {
		c.FakeIPRangeV4 = "28.0.0.0/8"
	}
	if c.FakeIPRangeV6 == "" {
		c.FakeIPRangeV6 = "f2b0::/18"
	}
	if c.LinuxProxyMode == "" {
		c.LinuxProxyMode = "nft"
	}
	if c.NFTProxyPolicy == "" {
		c.NFTProxyPolicy = "direct_default"
	}
	if c.ProxyCore == "" || c.ProxyCore == "singbox" {
		c.ProxyCore = "mihomo"
	}
	c.MosDNSEnabled = true
}

func (a *App) ensureDefaultConfigs() error {
	cfg := SetupConfig{
		Timezone:          "Asia/Shanghai",
		WebPort:           "7777",
		SelectedInterface: "eth0",
		MihomoCoreType:    "meta",
		AutoSetDNS:        true,
		EnableIPv6:        true,
		ProxyCore:         "mihomo",
		MosDNSEnabled:     true,
	}
	cfg.defaults()
	files := map[string]string{
		"configs/app.yaml":                 a.renderAppYAML(cfg),
		"configs/mihomo/config.yaml":       a.renderMihomoYAML(cfg),
		"configs/mihomo/phone_config.yaml": a.renderMihomoPhoneYAML(cfg),
		"configs/mosdns/config.yaml":       a.renderMosDNSYAML(cfg),
		"configs/network/network.yaml":     a.renderNetworkYAML(cfg),
		"configs/network/network.nft":      a.renderNFT(cfg),
		"configs/singbox/config.json":      renderDisabledSingBoxJSON(),
	}
	for rel, content := range files {
		path := filepath.Join(a.DataDir, rel)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return err
			}
		}
	}
	return a.ensureMosDNSRuleFiles()
}

func (a *App) writeGeneratedConfigs(cfg SetupConfig) error {
	cfg.defaults()
	files := map[string]string{
		"configs/app.yaml":                 a.renderAppYAML(cfg),
		"configs/mihomo/config.yaml":       a.renderMihomoYAML(cfg),
		"configs/mihomo/phone_config.yaml": a.renderMihomoPhoneYAML(cfg),
		"configs/mosdns/config.yaml":       a.renderMosDNSYAML(cfg),
		"configs/network/network.yaml":     a.renderNetworkYAML(cfg),
		"configs/network/network.nft":      a.renderNFT(cfg),
		"configs/singbox/config.json":      renderDisabledSingBoxJSON(),
	}
	for rel, content := range files {
		if err := a.writeTextFile(rel, content); err != nil {
			return err
		}
	}
	return a.ensureMosDNSRuleFiles()
}

func renderDisabledSingBoxJSON() string {
	return `{
  "log": {
    "level": "info",
    "timestamp": true
  },
  "dns": {
    "servers": [
      {
        "tag": "local",
        "address": "223.5.5.5"
      }
    ]
  },
  "inbounds": [],
  "outbounds": [
    {
      "type": "direct",
      "tag": "direct"
    }
  ],
  "route": {
    "rules": [],
    "final": "direct"
  }
}
`
}

func (a *App) renderAppYAML(cfg SetupConfig) string {
	return fmt.Sprintf(`server:
  host: 0.0.0.0
  port: %s
  mode: release
  enable_https: false
system:
  timezone: %s
jwt:
  secret: %s
`, cfg.WebPort, cfg.Timezone, string(a.Secret))
}

func (a *App) renderMihomoYAML(cfg SetupConfig) string {
	providers := parseSubscriptionProviders(cfg.SubscriptionURLs)
	providerYAML := "proxy-providers: {}\n"
	if len(providers) > 0 {
		var b strings.Builder
		b.WriteString("proxy-providers:\n")
		keys := make([]string, 0, len(providers))
		for k := range providers {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, tag := range keys {
			url := providers[tag]
			b.WriteString(fmt.Sprintf("  %q:\n    type: http\n    url: %q\n    interval: 3600\n    path: ./proxy_providers/%s.yaml\n    health-check:\n      enable: true\n      url: http://detectportal.firefox.com/success.txt\n      interval: 120\n", tag, url, safeFilename(tag)))
		}
		providerYAML = b.String()
	}
	ipv6 := "false"
	if cfg.EnableIPv6 {
		ipv6 = "true"
	}
	return fmt.Sprintf(`# msm-free generated Mihomo config
mode: rule
log-level: info
unified-delay: true
tcp-concurrent: true
interface-name: %s
ipv6: %s
udp: true
port: 7890
socks-port: 7891
mixed-port: 7892
redir-port: 7877
tproxy-port: 7896
allow-lan: true
bind-address: "*"
routing-mark: 1
external-controller: :9090
external-ui: ui
external-ui-url: https://github.com/Zephyruso/zashboard/archive/refs/heads/gh-pages.zip
profile:
  store-selected: true
  store-fake-ip: true
sniffer:
  enable: false
tun:
  enable: false
dns:
  enable: true
  listen: 0.0.0.0:6666
  ipv6: %s
  enhanced-mode: fake-ip
  fake-ip-range: %s
  fake-ip-filter:
    - "*"
    - +.lan
  default-nameserver:
    - 127.0.0.1:8888
proxy-groups:
  - {name: 节点选择, type: select, proxies: [手动切换, 全球直连]}
  - {name: 手动切换, type: select, proxies: [DIRECT], include-all: true}
  - {name: 漏网之鱼, type: select, proxies: [节点选择, 全球直连]}
  - {name: 全球直连, type: select, proxies: [DIRECT]}
rules:
  - DOMAIN-SUFFIX,lan,全球直连
  - IP-CIDR,10.0.0.0/8,全球直连,no-resolve
  - IP-CIDR,172.16.0.0/12,全球直连,no-resolve
  - IP-CIDR,192.168.0.0/16,全球直连,no-resolve
  - IP-CIDR,127.0.0.0/8,全球直连,no-resolve
  - MATCH,节点选择
%s`, cfg.SelectedInterface, ipv6, ipv6, strings.TrimSuffix(cfg.FakeIPRangeV4, "/8")+"/8", providerYAML)
}

func (a *App) renderMihomoPhoneYAML(cfg SetupConfig) string {
	cfg.FakeIPRangeV4 = "28.0.0.1/8"
	return a.renderMihomoYAML(cfg)
}

func (a *App) renderMosDNSYAML(cfg SetupConfig) string {
	return fmt.Sprintf(`log:
  level: warn
  file: "%s"
api:
  http: "0.0.0.0:9099"
include:
  - "sub_config/switch.yaml"
  - "sub_config/forward_local.yaml"
  - "sub_config/forward_remote.yaml"
plugins:
  - tag: blocklist
    type: domain_set
    args:
      files:
        - "rules/blocklist.txt"
  - tag: client_ip
    type: ip_set
    args:
      files:
        - "client_ip.txt"
  - tag: forward_local
    type: forward
    args:
      concurrent: 2
      upstreams:
        - addr: "udp://223.5.5.5"
        - addr: "udp://119.29.29.29"
  - tag: forward_remote
    type: forward
    args:
      concurrent: 2
      upstreams:
        - addr: "udp://127.0.0.1:6666"
  - tag: sequence_main
    type: sequence
    args:
      - matches:
          - qname $blocklist
        exec: reject 0
      - matches:
          - client_ip $client_ip
        exec: $forward_remote
      - exec: $forward_local
  - tag: udp_all
    type: udp_server
    args:
      entry: sequence_main
      listen: ":53"
      enable_audit: true
  - tag: tcp_all
    type: tcp_server
    args:
      entry: sequence_main
      listen: ":53"
      enable_audit: true
`, filepath.ToSlash(filepath.Join(a.DataDir, "configs/logs/mosdns.log")))
}

func (a *App) renderNetworkYAML(cfg SetupConfig) string {
	v := map[string]any{
		"mode":            "tproxy",
		"proxy_policy":    cfg.NFTProxyPolicy,
		"interface":       cfg.SelectedInterface,
		"mark":            1,
		"table":           100,
		"allow_dns":       true,
		"dns_ports":       []int{53, 5353},
		"system_dns_on":   cfg.DNSOn,
		"system_dns_off":  cfg.DNSOff,
		"auto_system_dns": cfg.AutoSetDNS,
		"fake_ipv4":       []string{cfg.FakeIPRangeV4},
		"fake_ipv6":       []string{cfg.FakeIPRangeV6},
		"tproxy_port":     7896,
		"ipv4":            map[string]any{"enable": true, "listen_port": 7877},
		"ipv6":            map[string]any{"enable": cfg.EnableIPv6, "listen_port": 7877},
	}
	b, _ := yaml.Marshal(v)
	return string(b)
}

func (a *App) renderNFT(cfg SetupConfig) string {
	iface := cfg.SelectedInterface
	if iface == "" {
		iface = "eth0"
	}
	return fmt.Sprintf(`#!/usr/sbin/nft -f
flush ruleset

table inet msm_free {
  set local_ipv4 {
    type ipv4_addr
    flags interval
    elements = { 0.0.0.0/8, 10.0.0.0/8, 127.0.0.0/8, 169.254.0.0/16, 172.16.0.0/12, 192.168.0.0/16, 224.0.0.0/4, 240.0.0.0/4 }
  }

  chain prerouting {
    type filter hook prerouting priority mangle; policy accept;
    iifname "%s" ip daddr @local_ipv4 return
    iifname "%s" meta l4proto { tcp, udp } tproxy to :7896 meta mark set 1 accept
  }

  chain output {
    type route hook output priority mangle; policy accept;
    ip daddr @local_ipv4 return
    meta l4proto { tcp, udp } meta mark set 1 accept
  }
}
`, iface, iface)
}

func (a *App) ensureMosDNSRuleFiles() error {
	files := map[string]string{
		"configs/mosdns/client_ip.txt":                  "",
		"configs/mosdns/rules/blocklist.txt":            "",
		"configs/mosdns/sub_config/switch.yaml":         "plugins: []\n",
		"configs/mosdns/sub_config/forward_local.yaml":  "plugins: []\n",
		"configs/mosdns/sub_config/forward_remote.yaml": "plugins: []\n",
	}
	for rel, content := range files {
		path := filepath.Join(a.DataDir, rel)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return err
			}
		}
	}
	for i := 1; i <= 13; i++ {
		key := fmt.Sprintf("switch%d", i)
		_, _ = a.DB.Exec(`insert or ignore into mosdns_switch_states(switch_key,enabled,created_at,updated_at) values(?,?,?,?)`, key, i != 3, nowString(), nowString())
	}
	return nil
}

func parseSubscriptionProviders(input string) map[string]string {
	out := map[string]string{}
	idx := 0
	for _, item := range strings.Fields(input) {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		tag, url := "", item
		if strings.Contains(item, "|") {
			parts := strings.SplitN(item, "|", 2)
			tag, url = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		}
		if tag == "" {
			idx++
			tag = fmt.Sprintf("机场%d", idx)
		}
		if url != "" {
			out[tag] = url
		}
	}
	return out
}

func safeFilename(s string) string {
	repl := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_", " ", "_")
	return repl.Replace(s)
}
