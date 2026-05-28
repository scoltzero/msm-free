package server

import (
	"fmt"
	"net/netip"
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
		c.FakeIPRangeV4 = "28.0.0.1/8"
	}
	if c.FakeIPRangeV6 == "" {
		c.FakeIPRangeV6 = "2001:2::/64"
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
	if err := a.ensureMSSBTemplateDefaults(false); err != nil {
		return err
	}
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
	if err := a.ensureMSSBTemplateDefaults(true); err != nil {
		return err
	}
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
	if template, ok := mssbTemplateText("mihomo/config.yaml"); ok {
		return renderMihomoTemplate(template, cfg)
	}
	return renderMihomoFallbackYAML(cfg)
}

func (a *App) renderMihomoPhoneYAML(cfg SetupConfig) string {
	return a.renderMihomoYAML(cfg)
}

func renderMihomoTemplate(template string, cfg SetupConfig) string {
	content := template
	ipv6 := boolYAML(cfg.EnableIPv6)
	content = strings.Replace(content, "interface-name: eth0", "interface-name: "+selectedInterface(cfg.SelectedInterface), 1)
	content = strings.ReplaceAll(content, "ipv6: true", "ipv6: "+ipv6)
	content = strings.Replace(content, "external-ui: /mssb/mihomo/ui", "external-ui: ui", 1)
	content = strings.Replace(content, "fake-ip-range: 28.0.0.1/8", "fake-ip-range: "+normalizeMihomoFakeIPv4Range(cfg.FakeIPRangeV4), 1)
	return replaceMihomoProxyProviders(content, renderProxyProvidersYAML(parseSubscriptionProviders(cfg.SubscriptionURLs)))
}

func renderMihomoFallbackYAML(cfg SetupConfig) string {
	providerYAML := renderProxyProvidersYAML(parseSubscriptionProviders(cfg.SubscriptionURLs))
	ipv6 := boolYAML(cfg.EnableIPv6)
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
geodata-mode: true
geodata-loader: standard
geo-auto-update: true
geo-update-interval: 24
find-process-mode: strict
global-client-fingerprint: chrome
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
  - {name: 手动切换, type: select, proxies: [DIRECT], include-all-providers: true}
  - {name: 漏网之鱼, type: select, proxies: [节点选择, 全球直连]}
  - {name: 全球直连, type: select, proxies: [DIRECT]}
rules:
  - DOMAIN-SUFFIX,lan,全球直连
  - IP-CIDR,10.0.0.0/8,全球直连,no-resolve
  - IP-CIDR,172.16.0.0/12,全球直连,no-resolve
  - IP-CIDR,192.168.0.0/16,全球直连,no-resolve
  - IP-CIDR,127.0.0.0/8,全球直连,no-resolve
  - IP-CIDR,8.8.8.8/32,节点选择
  - IP-CIDR,1.1.1.1/32,节点选择
  - MATCH,节点选择
%s`, selectedInterface(cfg.SelectedInterface), ipv6, ipv6, normalizeMihomoFakeIPv4Range(cfg.FakeIPRangeV4), providerYAML)
}

func renderProxyProvidersYAML(providers map[string]string) string {
	if len(providers) == 0 {
		return "proxy-providers: {}\n"
	}
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
		return b.String()
	}
	return "proxy-providers: {}\n"
}

func (a *App) renderMosDNSYAML(cfg SetupConfig) string {
	if template, ok := mssbTemplateText("mosdns/config.yaml"); ok {
		logPath := filepath.ToSlash(filepath.Join(a.DataDir, "logs/mosdns.log"))
		content := strings.Replace(template, `file: "/tmp/mosdns.log"`, fmt.Sprintf(`file: "%s"`, logPath), 1)
		if !strings.Contains(content, "tag: udp_main") {
			content = strings.Replace(content, "\n  - tag: sequence_requery", mssbMainSplitServerYAML()+"\n  - tag: sequence_requery", 1)
		}
		content = strings.ReplaceAll(content, "28.0.0.0/8", fakeIPv4RouteCIDR(cfg.FakeIPRangeV4))
		content = strings.ReplaceAll(content, "2001:2::/64", fakeIPv6RouteCIDR(cfg.FakeIPRangeV6))
		return content
	}
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
`, filepath.ToSlash(filepath.Join(a.DataDir, "logs/mosdns.log")))
}

func mssbMainSplitServerYAML() string {
	return `
  - tag: forward_all_in
    type: forward
    args:
      concurrent: 1
      upstreams:
        - addr: "udp://127.0.0.1:5656"

  - tag: udp_main
    type: udp_server
    args:
      entry: sequence_6666
      listen: 127.0.0.1:5656

  - tag: tcp_main
    type: tcp_server
    args:
      entry: sequence_6666
      listen: 127.0.0.1:5656
      idle_timeout: 720
`
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
		"fake_ipv4":       []string{fakeIPv4RouteCIDR(cfg.FakeIPRangeV4)},
		"fake_ipv6":       []string{fakeIPv6RouteCIDR(cfg.FakeIPRangeV6)},
		"tproxy_port":     7896,
		"ipv4":            map[string]any{"enable": true, "listen_port": 7877},
		"ipv6":            map[string]any{"enable": cfg.EnableIPv6, "listen_port": 7877},
	}
	b, _ := yaml.Marshal(v)
	return string(b)
}

func (a *App) renderNFT(cfg SetupConfig) string {
	ifaceSet := nftInterfaceSet(cfg.SelectedInterface)
	return fmt.Sprintf(`#!/usr/sbin/nft -f
flush ruleset

table inet msm_free {
  set local_ipv4 {
    type ipv4_addr
    flags interval
    elements = { 0.0.0.0/8, 10.0.0.0/8, 127.0.0.0/8, 169.254.0.0/16, 172.16.0.0/12, 192.168.0.0/16, 224.0.0.0/4, 240.0.0.0/4 }
  }

  set local_ipv6 {
    type ipv6_addr
    flags interval
    elements = { ::ffff:0.0.0.0/96, 64:ff9b::/96, 100::/64, 2001::/32, 2001:10::/28, 2001:20::/28, 2001:db8::/32, 2002::/16, fc00::/7, fe80::/10 }
  }

  set china_dns_ipv4 {
    type ipv4_addr
    elements = { 221.130.33.60, 223.5.5.5, 223.6.6.6, 119.29.29.29, 119.28.28.28, 114.114.114.114, 114.114.115.115 }
  }

  set china_dns_ipv6 {
    type ipv6_addr
    elements = { 2400:3200::1, 2400:3200:baba::1, 2402:4e00:: }
  }

  set dns_ipv4 {
    type ipv4_addr
    elements = { 8.8.8.8/32, 8.8.4.4/32, 1.0.0.1/32, 1.1.1.1/32, 9.9.9.9/32 }
  }

  set dns_ipv6 {
    type ipv6_addr
    elements = { 2001:4860:4860::8888/128, 2001:4860:4860::8844/128, 2606:4700:4700::1111/128, 2606:4700:4700::1001/128 }
  }

  set fake_ipv4 {
    type ipv4_addr
    flags interval
    elements = { %s }
  }

  set fake_ipv6 {
    type ipv6_addr
    flags interval
    elements = { %s }
  }

  chain nat-prerouting {
    type nat hook prerouting priority dstnat; policy accept;
    fib daddr type { unspec, local, anycast, multicast } return
    ip daddr @local_ipv4 return
    ip6 daddr @local_ipv6 return
    ip daddr @china_dns_ipv4 return
    ip6 daddr @china_dns_ipv6 return
    udp dport { 123 } return
    udp dport { 53 } accept
    ip daddr @dns_ipv4 meta l4proto tcp redirect to :7877
    ip6 daddr @dns_ipv6 meta l4proto tcp redirect to :7877
    iifname { %s } meta l4proto tcp redirect to :7877
  }

  chain nat-output {
    type nat hook output priority filter; policy accept;
    fib daddr type { unspec, local, anycast, multicast } return
    ip daddr @fake_ipv4 meta l4proto tcp redirect to :7877
    ip6 daddr @fake_ipv6 meta l4proto tcp redirect to :7877
  }

  chain proxy-tproxy {
    fib daddr type { unspec, local, anycast, multicast } return
    ip daddr @local_ipv4 return
    ip6 daddr @local_ipv6 return
    ip daddr @china_dns_ipv4 return
    ip6 daddr @china_dns_ipv6 return
    udp dport { 123 } return
    udp dport { 53 } accept
    meta l4proto udp meta mark set 1 tproxy to :7896 accept
  }

  chain proxy-mark {
    fib daddr type { unspec, local, anycast, multicast } return
    ip daddr @local_ipv4 return
    ip6 daddr @local_ipv6 return
    ip daddr @china_dns_ipv4 return
    ip6 daddr @china_dns_ipv6 return
    udp dport { 123 } return
    udp dport { 53 } accept
    meta mark set 1
  }

  chain mangle-output {
    type route hook output priority mangle; policy accept;
    meta l4proto udp skgid != 1 ct direction original goto proxy-mark
  }

  chain mangle-prerouting {
    type filter hook prerouting priority mangle; policy accept;
    ip daddr @dns_ipv4 meta l4proto udp ct direction original goto proxy-tproxy
    ip6 daddr @dns_ipv6 meta l4proto udp ct direction original goto proxy-tproxy
    iifname { %s } meta l4proto udp ct direction original goto proxy-tproxy
  }
}
`, fakeIPv4RouteCIDR(cfg.FakeIPRangeV4), fakeIPv6RouteCIDR(cfg.FakeIPRangeV6), ifaceSet, ifaceSet)
}

func (a *App) ensureMosDNSRuleFiles() error {
	files := map[string]string{
		"configs/mosdns/client_ip.txt":      "",
		"configs/mosdns/rule/blocklist.txt": "",
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
	defaults := defaultMosDNSSwitchStates()
	for i := 1; i <= 15; i++ {
		key := fmt.Sprintf("switch%d", i)
		_, _ = a.DB.Exec(`insert or ignore into mosdns_switch_states(switch_key,enabled,created_at,updated_at) values(?,?,?,?)`, key, defaults[key], nowString(), nowString())
	}
	return nil
}

func replaceMihomoProxyProviders(content, providersYAML string) string {
	marker := "\n# 节点订阅\nproxy-providers:"
	if idx := strings.Index(content, marker); idx >= 0 {
		return strings.TrimRight(content[:idx], "\n") + "\n\n# 节点订阅\n" + strings.TrimRight(providersYAML, "\n") + "\n"
	}
	if idx := strings.LastIndex(content, "\nproxy-providers:"); idx >= 0 {
		return strings.TrimRight(content[:idx], "\n") + "\n\n" + strings.TrimRight(providersYAML, "\n") + "\n"
	}
	return strings.TrimRight(content, "\n") + "\n\n" + strings.TrimRight(providersYAML, "\n") + "\n"
}

func boolYAML(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func selectedInterface(iface string) string {
	if iface == "" {
		return "eth0"
	}
	return iface
}

func normalizeMihomoFakeIPv4Range(v string) string {
	v = strings.TrimSpace(v)
	switch v {
	case "", "28.0.0.0/8":
		return "28.0.0.1/8"
	default:
		return v
	}
}

func fakeIPv4RouteCIDR(v string) string {
	v = normalizeMihomoFakeIPv4Range(v)
	if p, err := netip.ParsePrefix(v); err == nil {
		return p.Masked().String()
	}
	if addr, err := netip.ParseAddr(v); err == nil && addr.Is4() {
		return netip.PrefixFrom(addr, 8).Masked().String()
	}
	return "28.0.0.0/8"
}

func fakeIPv6RouteCIDR(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		v = "2001:2::/64"
	}
	if p, err := netip.ParsePrefix(v); err == nil {
		return p.Masked().String()
	}
	if addr, err := netip.ParseAddr(v); err == nil && addr.Is6() {
		return netip.PrefixFrom(addr, 64).Masked().String()
	}
	return "2001:2::/64"
}

func nftInterfaceSet(iface string) string {
	seen := map[string]bool{}
	var values []string
	for _, item := range []string{"lo", selectedInterface(iface)} {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		values = append(values, fmt.Sprintf("%q", item))
	}
	return strings.Join(values, ", ")
}

func defaultMosDNSSwitchStates() map[string]bool {
	return map[string]bool{
		"switch1": true, "switch2": false, "switch3": true, "switch4": true, "switch5": true,
		"switch6": false, "switch7": true, "switch8": false, "switch9": true, "switch10": false,
		"switch11": true, "switch12": false, "switch13": true, "switch14": true, "switch15": true,
	}
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
