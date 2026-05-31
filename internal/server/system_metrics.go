package server

import (
	"bufio"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type procMetrics struct {
	Uptime int64
	Memory int64
	CPU    int64
}

type cpuTimes struct {
	total uint64
	idle  uint64
}

func processResourceSnapshot(pid int) (procMetrics, bool) {
	if runtime.GOOS != "linux" || pid <= 0 {
		return procMetrics{}, false
	}
	stat, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return procMetrics{}, false
	}
	text := string(stat)
	end := strings.LastIndex(text, ")")
	if end < 0 || end+2 >= len(text) {
		return procMetrics{}, false
	}
	fields := strings.Fields(text[end+2:])
	if len(fields) < 20 {
		return procMetrics{}, false
	}
	utime, _ := strconv.ParseUint(fields[11], 10, 64)
	stime, _ := strconv.ParseUint(fields[12], 10, 64)
	startTicks, _ := strconv.ParseUint(fields[19], 10, 64)
	uptime := readSystemUptimeSeconds()
	const clockTicks = 100
	startSeconds := float64(startTicks) / clockTicks
	elapsed := uptime - startSeconds
	if elapsed < 1 {
		elapsed = 1
	}
	cpuSeconds := float64(utime+stime) / clockTicks
	cpuPercent := int64(cpuSeconds * 100 / elapsed)
	rss := readProcRSSBytes(pid)
	return procMetrics{Uptime: int64(elapsed), Memory: int64(rss), CPU: cpuPercent}, true
}

func readProcRSSBytes(pid int) uint64 {
	file, err := os.Open(filepath.Join("/proc", strconv.Itoa(pid), "status"))
	if err != nil {
		return 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "VmRSS:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			value, _ := strconv.ParseUint(fields[1], 10, 64)
			return value * 1024
		}
	}
	return 0
}

func readSystemUptimeSeconds() float64 {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(b))
	if len(fields) == 0 {
		return 0
	}
	value, _ := strconv.ParseFloat(fields[0], 64)
	return value
}

func readCPUTimes() (cpuTimes, error) {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuTimes{}, err
	}
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[0] != "cpu" {
			continue
		}
		var total uint64
		for _, field := range fields[1:] {
			value, _ := strconv.ParseUint(field, 10, 64)
			total += value
		}
		idle, _ := strconv.ParseUint(fields[4], 10, 64)
		if len(fields) > 5 {
			iowait, _ := strconv.ParseUint(fields[5], 10, 64)
			idle += iowait
		}
		return cpuTimes{total: total, idle: idle}, nil
	}
	return cpuTimes{}, errors.New("cpu line not found")
}

func sampleCPUPercent() float64 {
	if runtime.GOOS != "linux" {
		return 0
	}
	a, err := readCPUTimes()
	if err != nil {
		return 0
	}
	time.Sleep(120 * time.Millisecond)
	b, err := readCPUTimes()
	if err != nil || b.total <= a.total {
		return 0
	}
	total := b.total - a.total
	idle := b.idle - a.idle
	if total == 0 || idle > total {
		return 0
	}
	return float64(total-idle) * 100 / float64(total)
}

func readNetworkCounters() []map[string]any {
	b, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return nil
	}
	var rows []map[string]any
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		name := strings.TrimSpace(parts[0])
		fields := strings.Fields(parts[1])
		if name == "" || len(fields) < 16 {
			continue
		}
		rx, _ := strconv.ParseUint(fields[0], 10, 64)
		tx, _ := strconv.ParseUint(fields[8], 10, 64)
		rows = append(rows, map[string]any{"name": name, "rx_bytes": rx, "tx_bytes": tx})
	}
	return rows
}

func diskUsage(path string) map[string]any {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free
	return map[string]any{"ok": true, "total": total, "free": free, "used": used, "percent": percent(used, total)}
}

func tcpPortOpen(port int) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 150*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func diagnosticPortRows() []map[string]any {
	defs := []struct {
		Service string
		Port    int
		Desc    string
	}{
		{"msm", 7777, "Web UI"},
		{"mosdns", 53, "DNS 服务入口"},
		{"mosdns", 2222, "国内 DNS"},
		{"mosdns", 3333, "国外转发"},
		{"mosdns", 4444, "国外缓存 DNS"},
		{"mosdns", 5656, "主分流服务器"},
		{"mosdns", 6666, "Mihomo DNS 对接"},
		{"mosdns", 8888, "内部 DNS"},
		{"mosdns", 9099, "统计接口"},
		{"mihomo", 7890, "HTTP 代理"},
		{"mihomo", 7891, "SOCKS5 代理"},
		{"mihomo", 7892, "Mixed 代理"},
		{"mihomo", 7896, "TProxy"},
		{"mihomo", 7877, "Redirect"},
		{"mihomo", 9090, "Zashboard"},
	}
	rows := make([]map[string]any, 0, len(defs))
	for _, def := range defs {
		inUse := tcpPortOpen(def.Port)
		status := "free"
		if inUse {
			status = "ok"
		}
		rows = append(rows, map[string]any{
			"service":          def.Service,
			"port":             def.Port,
			"protocol":         "tcp",
			"name":             def.Desc,
			"description":      def.Desc,
			"status":           status,
			"in_use":           inUse,
			"owner_process":    "",
			"owner_pid":        nil,
			"expected_owner":   def.Service,
			"expected_pid":     nil,
			"config_path_hint": "",
			"notes":            "",
		})
	}
	return rows
}
