package server

import (
	"bufio"
	"context"
	"database/sql"
	"net"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type discoveredMosDNSClient struct {
	IP        string
	MAC       string
	Hostname  string
	Interface string
	Source    string
	Online    bool
}

func (a *App) scanMosDNSClientSources(iface string) ([]map[string]string, map[string]any) {
	iface = strings.TrimSpace(iface)
	refreshNeighborCache(iface)
	clients := map[string]discoveredMosDNSClient{}
	sources := map[string]int{}
	warnings := []string{}
	add := func(c discoveredMosDNSClient) {
		c.IP = strings.TrimSpace(c.IP)
		if !isUsefulClientIP(c.IP) {
			return
		}
		if iface != "" && c.Interface != "" && c.Interface != iface {
			return
		}
		if c.Source == "" {
			c.Source = "scan"
		}
		current, ok := clients[c.IP]
		if !ok {
			clients[c.IP] = c
			sources[c.Source]++
			return
		}
		if current.MAC == "" && c.MAC != "" {
			current.MAC = c.MAC
		}
		if current.Hostname == "" && c.Hostname != "" {
			current.Hostname = c.Hostname
		}
		if current.Interface == "" && c.Interface != "" {
			current.Interface = c.Interface
		}
		current.Online = current.Online || c.Online
		if !strings.Contains(current.Source, c.Source) {
			current.Source = current.Source + "," + c.Source
		}
		clients[c.IP] = current
		sources[c.Source]++
	}
	for _, item := range scanNeighborTables(iface) {
		add(discoveredMosDNSClient{
			IP: item["ip"], MAC: item["mac"], Hostname: item["hostname"], Interface: item["interface"], Source: firstNonEmpty(item["source"], "neigh"), Online: true,
		})
	}
	for _, item := range scanProcNetARP(iface) {
		add(item)
	}
	for _, ip := range a.mosDNSClientIPSetMerged() {
		add(discoveredMosDNSClient{IP: ip, Source: "allowlist", Online: false})
	}
	for _, entry := range a.mosDNSQueryDataset(2000) {
		add(discoveredMosDNSClient{IP: stringMapValue(entry, "client_ip"), Hostname: stringMapValue(entry, "client"), Source: "dnslog", Online: false})
	}
	out := make([]map[string]string, 0, len(clients))
	for _, c := range clients {
		online := "false"
		if c.Online {
			online = "true"
		}
		out = append(out, map[string]string{
			"ip": c.IP, "mac": c.MAC, "hostname": firstNonEmpty(c.Hostname, c.IP), "interface": c.Interface, "source": c.Source, "online": online,
		})
	}
	sortDiscoveredClients(out)
	return out, map[string]any{"sources": sources, "warnings": warnings}
}

func sortDiscoveredClients(items []map[string]string) {
	sortIP := func(ip string) string {
		parsed := net.ParseIP(ip)
		if parsed == nil {
			return ip
		}
		if v4 := parsed.To4(); v4 != nil {
			return string(v4)
		}
		return parsed.String()
	}
	sort.SliceStable(items, func(i, j int) bool {
		return sortIP(items[i]["ip"]) < sortIP(items[j]["ip"])
	})
}

func (a *App) upsertMosDNSScannedClient(item map[string]string, allowIPs map[string]bool, now time.Time) error {
	ip := strings.TrimSpace(item["ip"])
	if !isUsefulClientIP(ip) {
		return nil
	}
	mac := strings.TrimSpace(item["mac"])
	hostname := strings.TrimSpace(item["hostname"])
	iface := strings.TrimSpace(item["interface"])
	source := firstNonEmpty(strings.TrimSpace(item["source"]), "scan")
	online := strings.EqualFold(item["online"], "true") || source == "neigh" || strings.Contains(source, "arp")
	status := "unscanned"
	var id int64
	var currentType string
	err := a.DB.QueryRow(`select id,coalesce(type,'') from mosdns_clients where ip=? order by case when coalesce(mac,'')='' then 1 else 0 end,id desc limit 1`, ip).Scan(&id, &currentType)
	if err == nil {
		status = normalizeMosDNSClientStatus(currentType, allowIPs[ip])
		if allowIPs[ip] {
			status = "allow"
		}
		_, err = a.DB.Exec(`update mosdns_clients set mac=coalesce(nullif(?,''),mac),hostname=coalesce(nullif(?,''),hostname),source=?,type=?,last_seen_at=?,last_scan_at=?,interface=coalesce(nullif(?,''),interface),is_online=?,updated_at=? where id=?`,
			mac, hostname, source, status, now, now, iface, online, now, id)
		return err
	}
	if err != sql.ErrNoRows {
		return err
	}
	if allowIPs[ip] {
		status = "allow"
	}
	_, err = a.DB.Exec(`insert into mosdns_clients(mac,ip,hostname,source,type,first_seen_at,last_seen_at,last_scan_at,interface,is_online,created_at,updated_at)
		values(?,?,?,?,?,?,?,?,?,?,?,?)`,
		mac, ip, hostname, source, status, now, now, now, iface, online, now, now)
	return err
}

func scanNeighbors() []map[string]string {
	return scanNeighborTables("")
}

func scanNeighborTables(iface string) []map[string]string {
	var items []map[string]string
	for _, args := range [][]string{{"ip", "neigh"}, {"arp", "-an"}} {
		out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
		if err != nil {
			continue
		}
		items = append(items, parseNeighborOutput(string(out), iface, args[0])...)
	}
	return dedupeClientMaps(items)
}

func parseNeighborOutput(out, iface, source string) []map[string]string {
	var items []map[string]string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		item := map[string]string{"ip": strings.Trim(fields[0], "()"), "mac": "", "hostname": "", "interface": "", "source": source}
		for i, f := range fields {
			switch f {
			case "dev":
				if i+1 < len(fields) {
					item["interface"] = fields[i+1]
				}
			case "lladdr", "at":
				if i+1 < len(fields) {
					item["mac"] = fields[i+1]
				}
			case "on":
				if source == "arp" && i+1 < len(fields) {
					item["interface"] = fields[i+1]
				}
			}
		}
		if iface != "" && item["interface"] != "" && item["interface"] != iface {
			continue
		}
		if isUsefulClientIP(item["ip"]) {
			items = append(items, item)
		}
	}
	return items
}

func scanProcNetARP(iface string) []discoveredMosDNSClient {
	file, err := os.Open("/proc/net/arp")
	if err != nil {
		return nil
	}
	defer file.Close()
	var out []discoveredMosDNSClient
	scanner := bufio.NewScanner(file)
	first := true
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if first {
			first = false
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		if iface != "" && fields[5] != iface {
			continue
		}
		out = append(out, discoveredMosDNSClient{IP: fields[0], MAC: fields[3], Interface: fields[5], Source: "proc_arp", Online: true})
	}
	return out
}

func refreshNeighborCache(iface string) {
	for _, target := range interfaceBroadcasts(iface) {
		ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
		args := []string{"-b", "-c", "1", "-W", "1", target}
		_ = exec.CommandContext(ctx, "ping", args...).Run()
		cancel()
	}
}

func interfaceBroadcasts(iface string) []string {
	out, err := exec.Command("ip", "-o", "-4", "addr", "show", "scope", "global").CombinedOutput()
	if err != nil {
		return nil
	}
	var targets []string
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		name := fields[1]
		if iface != "" && name != iface {
			continue
		}
		for i, field := range fields {
			if field == "brd" && i+1 < len(fields) && net.ParseIP(fields[i+1]) != nil {
				targets = append(targets, fields[i+1])
			}
		}
	}
	return targets
}

func (a *App) mosDNSClientIPSetMerged() []string {
	seen := map[string]bool{}
	for ip := range a.mosDNSClientIPSet() {
		if isUsefulClientIP(ip) {
			seen[ip] = true
		}
	}
	for _, rel := range []string{"configs/mosdns/client_ip.txt", "configs/mosdns/rule/client_ip.txt"} {
		content, _ := a.readTextFile(rel)
		for _, ip := range splitNonEmptyLines(content) {
			if isUsefulClientIP(ip) {
				seen[ip] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for ip := range seen {
		out = append(out, ip)
	}
	sort.Strings(out)
	return out
}

func dedupeClientMaps(items []map[string]string) []map[string]string {
	byIP := map[string]map[string]string{}
	for _, item := range items {
		ip := item["ip"]
		if !isUsefulClientIP(ip) {
			continue
		}
		current := byIP[ip]
		if current == nil {
			byIP[ip] = item
			continue
		}
		for _, key := range []string{"mac", "hostname", "interface"} {
			if current[key] == "" && item[key] != "" {
				current[key] = item[key]
			}
		}
		if !strings.Contains(current["source"], item["source"]) {
			current["source"] += "," + item["source"]
		}
	}
	out := make([]map[string]string, 0, len(byIP))
	for _, item := range byIP {
		out = append(out, item)
	}
	sortDiscoveredClients(out)
	return out
}

func isUsefulClientIP(ip string) bool {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil || parsed.IsLoopback() || parsed.IsUnspecified() || parsed.IsMulticast() {
		return false
	}
	if parsed.IsLinkLocalUnicast() || parsed.IsLinkLocalMulticast() {
		return false
	}
	return true
}
