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
	if cfg, ok := a.latestSetupConfig(); ok {
		cfg.defaults()
		if err := a.ensureSetupProviderArtifacts(cfg); err != nil {
			report.Errors = append(report.Errors, "failed to sync setup providers: "+err.Error())
		}
	}
	a.backfillConfiguredRuntimeDesired()
	report.Errors = append(report.Errors, a.Services.StartEnabled(ctx)...)
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

func (a *App) ensureSetupProviderArtifacts(cfg SetupConfig) error {
	cfg.defaults()
	providers := parseSubscriptionProviders(cfg.SubscriptionURLs)
	includeManual := hasMihomoManualProxies(cfg.MihomoProxies)
	if len(providers) == 0 && !includeManual {
		return nil
	}
	if manual := renderMihomoManualProviderYAML(cfg.MihomoProxies); strings.TrimSpace(manual) != "" {
		if old, err := a.readTextFile("configs/mihomo/proxy_providers/msm_manual.yaml"); err != nil || old != manual {
			if err := a.writeTextFile("configs/mihomo/proxy_providers/msm_manual.yaml", manual); err != nil {
				return err
			}
		}
	}
	config, err := a.readTextFile("configs/mihomo/config.yaml")
	if err != nil {
		return err
	}
	providerYAML := renderProxyProvidersYAML(providers, includeManual)
	patched := replaceMihomoProxyProviders(config, providerYAML)
	if patched != config {
		if err := a.writeTextFile("configs/mihomo/config.yaml", patched); err != nil {
			return err
		}
	}
	return nil
}
