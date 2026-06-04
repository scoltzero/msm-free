package server

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type DownloadEvent struct {
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	Message  string `json:"message"`
}

func (a *App) componentDownloadURL(component string) string {
	mihomoCoreType, amd64v3 := a.componentDownloadOptions()
	return componentDownloadURLFor(component, runtime.GOOS, runtime.GOARCH, mihomoCoreType, amd64v3)
}

func (a *App) componentDownloadOptions() (string, bool) {
	if a == nil || a.DB == nil {
		return "meta", false
	}
	var coreType string
	var amd64v3 bool
	err := a.DB.QueryRow(`select coalesce(mihomo_core_type,'meta'), amd64v3_enabled from system_setups order by id desc limit 1`).Scan(&coreType, &amd64v3)
	if err != nil {
		return "meta", false
	}
	return normalizeMihomoCoreType(coreType), amd64v3
}

func componentDownloadURLFor(component, goos, goarch, mihomoCoreType string, amd64v3 bool) string {
	switch component {
	case "mihomo":
		if goos != "linux" {
			return ""
		}
		arch := mihomoAssetArch(goarch, amd64v3)
		if arch == "" {
			return ""
		}
		return fmt.Sprintf("https://github.com/baozaodetudou/mssb/releases/download/mihomo/mihomo-%s-linux-%s.tar.gz", normalizeMihomoCoreType(mihomoCoreType), arch)
	case "mosdns":
		if goos != "linux" {
			return ""
		}
		arch := mosDNSAssetArch(goarch, amd64v3)
		if arch == "" {
			return ""
		}
		return fmt.Sprintf("https://github.com/baozaodetudou/mssb/releases/download/mosdns/mosdns-linux-%s.zip", arch)
	case "zashboard", "ui":
		return "https://github.com/Zephyruso/zashboard/releases/latest/download/dist.zip"
	default:
		return ""
	}
}

func normalizeMihomoCoreType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "alpha":
		return "alpha"
	default:
		return "meta"
	}
}

func mihomoAssetArch(goarch string, amd64v3 bool) string {
	switch goarch {
	case "amd64":
		if amd64v3 {
			return "amd64v3"
		}
		return "amd64"
	case "arm64":
		return "arm64"
	default:
		return ""
	}
}

func mosDNSAssetArch(goarch string, amd64v3 bool) string {
	switch goarch {
	case "amd64":
		if amd64v3 {
			return "amd64-v3"
		}
		return "amd64"
	case "arm64":
		return "arm64"
	default:
		return ""
	}
}

func (a *App) installComponent(component string, emit func(DownloadEvent)) error {
	if runtime.GOOS != "linux" && component != "zashboard" && component != "ui" {
		emit(DownloadEvent{Status: "skipped", Progress: 100, Message: "binary download is linux-only; place binary manually on this platform"})
		return nil
	}
	target := a.componentTarget(component)
	if target == "" {
		return fmt.Errorf("unknown component %s", component)
	}
	if _, err := os.Stat(target); err == nil {
		emit(DownloadEvent{Status: "running", Progress: 5, Message: component + " already installed; refreshing files"})
	}
	url := a.componentDownloadURL(component)
	if url == "" {
		return fmt.Errorf("no download URL for %s on %s/%s", component, runtime.GOOS, runtime.GOARCH)
	}
	emit(DownloadEvent{Status: "running", Progress: 5, Message: "downloading " + url})
	tmp := filepath.Join(a.DataDir, "data", component+".download")
	_ = os.Remove(tmp)
	if err := a.downloadFile(url, tmp, emit); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	emit(DownloadEvent{Status: "running", Progress: 60, Message: "extracting"})
	if strings.HasSuffix(url, ".zip") {
		if err := unzip(tmp, filepath.Dir(target)); err != nil {
			return err
		}
	} else {
		if err := untarGz(tmp, filepath.Dir(target)); err != nil {
			return err
		}
	}
	if component == "zashboard" || component == "ui" {
		if err := patchZashboardIndex(filepath.Dir(target)); err != nil {
			return err
		}
	}
	_ = os.Remove(tmp)
	_ = chmodExecutables(filepath.Dir(target))
	emit(DownloadEvent{Status: "completed", Progress: 100, Message: component + " installed"})
	return nil
}

func (a *App) componentTarget(component string) string {
	switch component {
	case "mihomo":
		return filepath.Join(a.DataDir, "data/binaries/mihomo/mihomo")
	case "mosdns":
		return filepath.Join(a.DataDir, "data/binaries/mosdns/mosdns")
	case "zashboard", "ui":
		return filepath.Join(a.DataDir, "configs/mihomo/ui/index.html")
	default:
		return ""
	}
}

func (a *App) downloadFile(rawURL, dest string, emit func(DownloadEvent)) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	finalURL := a.rewriteDownloadURL(rawURL)
	client := a.downloadHTTPClient()
	resp, err := client.Get(finalURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	var written int64
	total := resp.ContentLength
	buf := make([]byte, 128*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				return err
			}
			written += int64(n)
			if emit != nil && total > 0 {
				progress := 5 + int(float64(written)*50/float64(total))
				if progress > 55 {
					progress = 55
				}
				emit(DownloadEvent{Status: "running", Progress: progress, Message: fmt.Sprintf("downloaded %d/%d bytes", written, total)})
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return nil
}

func downloadFile(rawURL, dest string) error {
	app := &App{}
	return app.downloadFile(rawURL, dest, nil)
}

func (a *App) downloadHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if a != nil && a.DB != nil {
		if proxy := a.downloadProxyURL(); proxy != nil {
			transport.Proxy = http.ProxyURL(proxy)
		}
	}
	return &http.Client{Timeout: 10 * time.Minute, Transport: transport}
}

func (a *App) downloadProxyURL() *url.URL {
	var enabled bool
	var httpsProxy, httpProxy, socks5Proxy sql.NullString
	err := a.DB.QueryRow(`select github_proxy_enabled,github_https_proxy,github_http_proxy,github_socks5_proxy from system_setups order by id desc limit 1`).Scan(&enabled, &httpsProxy, &httpProxy, &socks5Proxy)
	if err != nil || !enabled {
		return nil
	}
	raw := strings.TrimSpace(httpsProxy.String)
	if raw == "" {
		raw = strings.TrimSpace(httpProxy.String)
	}
	if raw == "" {
		raw = strings.TrimSpace(socks5Proxy.String)
	}
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return nil
	}
	return u
}

func (a *App) rewriteDownloadURL(raw string) string {
	if a == nil || a.DB == nil || (!strings.Contains(raw, "github.com/") && !strings.Contains(raw, "githubusercontent.com/")) {
		return raw
	}
	var enabled bool
	var accelerator sql.NullString
	err := a.DB.QueryRow(`select github_accelerator_enabled,github_accelerator_url from system_setups order by id desc limit 1`).Scan(&enabled, &accelerator)
	if err != nil || !enabled {
		return raw
	}
	prefix := strings.TrimRight(strings.TrimSpace(accelerator.String), "/")
	if prefix == "" {
		return raw
	}
	return prefix + "/" + raw
}

func untarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		name := filepath.Base(h.Name)
		if name == "" || name == "." {
			continue
		}
		path := filepath.Join(dest, name)
		if h.FileInfo().IsDir() {
			_ = os.MkdirAll(path, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, h.FileInfo().Mode())
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		out.Close()
	}
	return nil
}

func unzip(src, dest string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		rel := stripFirstPathComponent(f.Name)
		if rel == "" {
			continue
		}
		path := filepath.Join(dest, rel)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(path, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, f.FileInfo().Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func stripFirstPathComponent(p string) string {
	p = filepath.ToSlash(p)
	parts := strings.SplitN(p, "/", 2)
	if len(parts) == 1 {
		return parts[0]
	}
	return parts[1]
}

func patchZashboardIndex(uiDir string) error {
	indexPath := filepath.Join(uiDir, "index.html")
	body, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}
	html := string(body)
	if strings.Contains(html, "msm-free-zashboard-disk-backend") {
		return nil
	}
	if strings.Contains(html, "</head>") {
		html = strings.Replace(html, "</head>", zashboardDiskAutoBackendScript+"</head>", 1)
	} else {
		html = zashboardDiskAutoBackendScript + html
	}
	return os.WriteFile(indexPath, []byte(html), 0644)
}

const zashboardDiskAutoBackendScript = `<script id="msm-free-zashboard-disk-backend">
;(function () {
  try {
    if (!window.localStorage) return
    var host = window.location.hostname || "127.0.0.1"
    var listKey = "setup/api-list"
    var activeKey = "setup/active-uuid"
    var raw = localStorage.getItem(listKey)
    var list = []
    if (raw) {
      try {
        var parsed = JSON.parse(raw)
        if (Array.isArray(parsed)) list = parsed
      } catch (_) {
        list = []
      }
    }
    var id = "msm-free-" + host.replace(/[^a-zA-Z0-9]/g, "-") + "-9090"
    var entry = {
      protocol: "http",
      secondaryPath: "",
      host: host,
      port: "9090",
      password: "",
      label: "msm-free",
      disableUpgradeCore: true,
      disableTunMode: false,
      uuid: id
    }
    var existing = list.find(function (item) { return item && item.uuid === id })
    if (existing) {
      Object.assign(existing, entry)
    } else {
      list.unshift(entry)
    }
    localStorage.setItem(listKey, JSON.stringify(list))
    localStorage.setItem(activeKey, id)
  } catch (err) {
    console.warn("msm-free zashboard disk backend preset failed", err)
  }
})()
</script>
`

func chmodExecutables(dir string) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := strings.ToLower(d.Name())
		if name == "mihomo" || name == "mosdns" || strings.Contains(name, "mihomo") || strings.Contains(name, "mosdns") {
			_ = os.Chmod(path, 0755)
		}
		return nil
	})
}
