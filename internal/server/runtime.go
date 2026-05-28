package server

import (
	"context"
	"log"
	"strings"
)

const nftDesiredKey = "network.nft.enabled"

type RuntimeRestoreReport struct {
	Initialized bool            `json:"initialized"`
	Services    []ServiceStatus `json:"services,omitempty"`
	NFT         map[string]any  `json:"nft,omitempty"`
	Errors      []string        `json:"errors,omitempty"`
}

func (a *App) SetConfiguredRuntimeDesired(cfg SetupConfig) {
	cfg.defaults()
	a.Services.setDesired("mihomo", strings.EqualFold(cfg.ProxyCore, "mihomo"))
	a.Services.setDesired("mosdns", cfg.MosDNSEnabled)
	a.setSetting(nftDesiredKey, boolSetting(shouldRestoreNFT(cfg)))
}

func (a *App) RestoreConfiguredRuntime(ctx context.Context) RuntimeRestoreReport {
	report := RuntimeRestoreReport{Initialized: a.IsInitialized()}
	if !report.Initialized {
		return report
	}
	a.backfillConfiguredRuntimeDesired()
	a.Services.StartEnabled(ctx)
	report.Services = a.Services.List()
	if a.setting(nftDesiredKey, "") == "true" {
		output, err := a.applyNFT(ctx)
		status := a.nftStatus()
		if output != "" {
			status["output"] = output
		}
		report.NFT = status
		if err != nil {
			msg := "failed to restore nftables: " + err.Error()
			report.Errors = append(report.Errors, msg)
			log.Print(msg)
		}
	}
	return report
}

func (a *App) backfillConfiguredRuntimeDesired() {
	cfg, ok := a.latestSetupConfig()
	if !ok {
		return
	}
	cfg.defaults()
	if a.setting(serviceDesiredKey("mihomo"), "") == "" {
		a.Services.setDesired("mihomo", strings.EqualFold(cfg.ProxyCore, "mihomo"))
	}
	if a.setting(serviceDesiredKey("mosdns"), "") == "" {
		a.Services.setDesired("mosdns", cfg.MosDNSEnabled)
	}
	if a.setting(nftDesiredKey, "") == "" {
		a.setSetting(nftDesiredKey, boolSetting(shouldRestoreNFT(cfg)))
	}
}

func (a *App) latestSetupConfig() (SetupConfig, bool) {
	row := a.DB.QueryRow(`select username,email,web_port,amd64v3_enabled,selected_interface,mihomo_core_type,auto_set_dns,dns_on,dns_off,enable_ipv6,fake_ip_range_v4,fake_ip_range_v6,linux_proxy_mode,nft_proxy_policy,proxy_core,mos_dns_enabled,subscription_urls,mihomo_proxies from system_setups order by id desc limit 1`)
	var cfg SetupConfig
	err := row.Scan(&cfg.Username, &cfg.Email, &cfg.WebPort, &cfg.AMD64v3Enabled, &cfg.SelectedInterface, &cfg.MihomoCoreType, &cfg.AutoSetDNS, &cfg.DNSOn, &cfg.DNSOff, &cfg.EnableIPv6, &cfg.FakeIPRangeV4, &cfg.FakeIPRangeV6, &cfg.LinuxProxyMode, &cfg.NFTProxyPolicy, &cfg.ProxyCore, &cfg.MosDNSEnabled, &cfg.SubscriptionURLs, &cfg.MihomoProxies)
	return cfg, err == nil
}

func shouldRestoreNFT(cfg SetupConfig) bool {
	mode := strings.ToLower(strings.TrimSpace(cfg.LinuxProxyMode))
	return mode == "" || mode == "nft" || mode == "nftables" || mode == "tproxy"
}

func boolSetting(ok bool) string {
	if ok {
		return "true"
	}
	return "false"
}
