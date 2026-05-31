package server

import (
	"encoding/json"
	"os"
	"runtime"
	"strconv"
	"time"
)

type monitorNetworkSample struct {
	At      time.Time
	RXBytes uint64
	TXBytes uint64
}

func (a *App) monitorPayload() map[string]any {
	now := time.Now()
	system := a.monitorSystemSnapshot()
	hardware := a.monitorHardwareSnapshot()
	resource := a.monitorResourceSnapshot()
	network := a.monitorNetworkSnapshot(now)
	services := a.enhancedServiceList()

	return map[string]any{
		"time":      now,
		"timestamp": now.Unix(),
		"system":    system,
		"hardware":  hardware,
		"resource":  resource,
		"resources": resource,
		"network":   network,
		"services":  services,
		"stats": map[string]any{
			"system":   system,
			"hardware": hardware,
			"resource": resource,
			"network":  network,
			"services": services,
		},
		"ips":              system["local_ips"],
		"local_ips":        system["local_ips"],
		"hostname":         system["hostname"],
		"platform":         system["platform"],
		"os":               system["os"],
		"arch":             system["arch"],
		"uptime":           system["uptime"],
		"cpu_percent":      resource["cpu_percent"],
		"mem_percent":      resource["mem_percent"],
		"memory_percent":   resource["memory_percent"],
		"memory_used":      resource["memory_used"],
		"memory_total":     resource["memory_total"],
		"disk_percent":     resource["disk_percent"],
		"disk_used":        resource["disk_used"],
		"disk_total":       resource["disk_total"],
		"download_speed":   network["download_speed"],
		"upload_speed":     network["upload_speed"],
		"total_download":   network["total_download"],
		"total_upload":     network["total_upload"],
		"download_total":   network["total_download"],
		"upload_total":     network["total_upload"],
		"connections":      network["connections"],
		"connection_count": network["connection_count"],
	}
}

func (a *App) monitorSystemSnapshot() map[string]any {
	uptime := readSystemUptimeSeconds()
	platform := runtime.GOOS + "/" + runtime.GOARCH
	return map[string]any{
		"hostname":       hostname(),
		"platform":       platform,
		"os":             runtime.GOOS,
		"arch":           runtime.GOARCH,
		"go_version":     runtime.Version(),
		"local_ips":      localIPs(),
		"ips":            localIPs(),
		"data_dir":       a.DataDir,
		"version":        a.Version,
		"pid":            os.Getpid(),
		"is_root":        os.Geteuid() == 0,
		"uptime":         uptime,
		"uptime_seconds": uptime,
	}
}

func (a *App) monitorHardwareSnapshot() map[string]any {
	mem := readMemInfo()
	memTotal := mem["MemTotal"]
	memAvailable := mem["MemAvailable"]
	memUsed := uint64(0)
	if memTotal >= memAvailable {
		memUsed = memTotal - memAvailable
	}
	disk := diskUsage(a.DataDir)
	diskTotal := uint64FromAny(disk["total"])
	diskUsed := uint64FromAny(disk["used"])
	cpu := map[string]any{
		"model":             cpuModel(),
		"cores":             runtime.NumCPU(),
		"supports_amd64v3":  supportsAMD64v3(),
		"supportsAMD64v3":   supportsAMD64v3(),
		"amd64v3_supported": supportsAMD64v3(),
	}
	return map[string]any{
		"cpu":              cpu,
		"cpu_model":        cpu["model"],
		"cpu_cores":        cpu["cores"],
		"cores":            cpu["cores"],
		"supports_amd64v3": cpu["supports_amd64v3"],
		"supportsAMD64v3":  cpu["supportsAMD64v3"],
		"memory":           mem,
		"memory_total":     memTotal,
		"memory_available": memAvailable,
		"memory_used":      memUsed,
		"memory_percent":   percent(memUsed, memTotal),
		"total_memory":     memTotal,
		"mem_total":        memTotal,
		"mem_used":         memUsed,
		"mem_percent":      percent(memUsed, memTotal),
		"disk":             disk,
		"disk_total":       diskTotal,
		"disk_used":        diskUsed,
		"disk_free":        uint64FromAny(disk["free"]),
		"disk_percent":     numericAny(disk["percent"]),
		"storage_total":    diskTotal,
		"storage_used":     diskUsed,
		"storage_percent":  numericAny(disk["percent"]),
	}
}

func (a *App) monitorResourceSnapshot() map[string]any {
	mem := readMemInfo()
	memTotal := mem["MemTotal"]
	memAvailable := mem["MemAvailable"]
	memUsed := uint64(0)
	if memTotal >= memAvailable {
		memUsed = memTotal - memAvailable
	}
	disk := diskUsage(a.DataDir)
	diskTotal := uint64FromAny(disk["total"])
	diskUsed := uint64FromAny(disk["used"])
	cpuPercent := sampleCPUPercent()
	memPercent := percent(memUsed, memTotal)
	diskPercent := numericAny(disk["percent"])
	return map[string]any{
		"cpu_percent":    cpuPercent,
		"cpu":            cpuPercent,
		"memory_total":   memTotal,
		"memory_free":    memAvailable,
		"memory_used":    memUsed,
		"memory_percent": memPercent,
		"mem_total":      memTotal,
		"mem_free":       memAvailable,
		"mem_used":       memUsed,
		"mem_percent":    memPercent,
		"disk_total":     diskTotal,
		"disk_free":      uint64FromAny(disk["free"]),
		"disk_used":      diskUsed,
		"disk_percent":   diskPercent,
		"goroutines":     runtime.NumGoroutine(),
	}
}

func (a *App) monitorNetworkSnapshot(now time.Time) map[string]any {
	interfaces := readNetworkCounters()
	var rxTotal, txTotal uint64
	for _, item := range interfaces {
		if stringMapValue(item, "name") == "lo" {
			continue
		}
		rxTotal += uint64FromAny(item["rx_bytes"])
		txTotal += uint64FromAny(item["tx_bytes"])
	}

	var downloadSpeed, uploadSpeed float64
	a.monitorMu.Lock()
	last := a.monitorNetworkLast
	if !last.At.IsZero() && now.After(last.At) {
		elapsed := now.Sub(last.At).Seconds()
		if elapsed > 0 {
			if rxTotal >= last.RXBytes {
				downloadSpeed = float64(rxTotal-last.RXBytes) / elapsed
			}
			if txTotal >= last.TXBytes {
				uploadSpeed = float64(txTotal-last.TXBytes) / elapsed
			}
		}
	}
	a.monitorNetworkLast = monitorNetworkSample{At: now, RXBytes: rxTotal, TXBytes: txTotal}
	a.monitorMu.Unlock()

	connectionCount, mihomoDownloadTotal, mihomoUploadTotal := a.monitorMihomoConnectionSummary()
	return map[string]any{
		"local_ips":             localIPs(),
		"interfaces":            interfaces,
		"rx_bytes":              rxTotal,
		"tx_bytes":              txTotal,
		"total_download":        rxTotal,
		"total_upload":          txTotal,
		"download_total":        rxTotal,
		"upload_total":          txTotal,
		"download_speed":        downloadSpeed,
		"upload_speed":          uploadSpeed,
		"down_speed":            downloadSpeed,
		"up_speed":              uploadSpeed,
		"downloadSpeed":         downloadSpeed,
		"uploadSpeed":           uploadSpeed,
		"connections":           connectionCount,
		"connection_count":      connectionCount,
		"mihomo_download_total": mihomoDownloadTotal,
		"mihomo_upload_total":   mihomoUploadTotal,
		"mihomo_downloadTotal":  mihomoDownloadTotal,
		"mihomo_uploadTotal":    mihomoUploadTotal,
	}
}

func (a *App) monitorMihomoConnectionSummary() (float64, float64, float64) {
	raw, ok := a.mihomoControllerMap("/connections")
	if !ok {
		return 0, 0, 0
	}
	return float64(len(anySlice(raw["connections"]))), numericMapValue(raw, "downloadTotal"), numericMapValue(raw, "uploadTotal")
}

func uint64FromAny(value any) uint64 {
	switch v := value.(type) {
	case uint64:
		return v
	case uint:
		return uint64(v)
	case uint32:
		return uint64(v)
	case int:
		if v > 0 {
			return uint64(v)
		}
	case int64:
		if v > 0 {
			return uint64(v)
		}
	case float64:
		if v > 0 {
			return uint64(v)
		}
	case json.Number:
		n, _ := v.Int64()
		if n > 0 {
			return uint64(n)
		}
	case string:
		n, _ := strconv.ParseUint(v, 10, 64)
		return n
	}
	return 0
}
