package server

import (
	"embed"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

//go:embed web/dist
var frontendFS embed.FS

func (a *App) registerStatic(mux *http.ServeMux) {
	mux.HandleFunc("GET /ui", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, a.zashboardSetupURL(r), http.StatusFound)
	})
	mux.HandleFunc("GET /ui/{path...}", a.handleMihomoUIAsset)

	sub, err := fs.Sub(frontendFS, "web/dist")
	if err != nil {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("msm-free frontend is not built"))
		})
		return
	}
	fileServer := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, http.StatusNotFound, "not_found", "API endpoint not found")
			return
		}
		if r.URL.Path != "/" {
			rel := strings.TrimPrefix(r.URL.Path, "/")
			if rel == "index.html" {
				serveFrontendIndex(w, sub)
				return
			}
			if _, err := fs.Stat(sub, rel); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		serveFrontendIndex(w, sub)
	})
}

func serveFrontendIndex(w http.ResponseWriter, fsys fs.FS) {
	body, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "frontend index.html not found")
		return
	}
	html := string(body)
	if !strings.Contains(html, "msm-free-spa-recovery") {
		if strings.Contains(html, "</head>") {
			html = strings.Replace(html, "</head>", frontendSPARecoveryScript+"</head>", 1)
		} else {
			html = frontendSPARecoveryScript + html
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(html))
}

func (a *App) handleMihomoUIAsset(w http.ResponseWriter, r *http.Request) {
	uiDir := filepath.Join(a.DataDir, "configs/mihomo/ui")
	rel := strings.TrimPrefix(r.PathValue("path"), "/")
	if rel == "" {
		rel = "index.html"
	}
	path := filepath.Join(uiDir, filepath.Clean(rel))
	base, _ := filepath.Abs(uiDir)
	abs, _ := filepath.Abs(path)
	if abs != base && !strings.HasPrefix(abs, base+string(filepath.Separator)) {
		writeError(w, http.StatusBadRequest, "path_error", "invalid ui asset path")
		return
	}
	if rel == "registerSW.js" {
		serveJavaScript(w, zashboardRegisterSWCleanupScript)
		return
	}
	if rel == "sw.js" {
		serveJavaScript(w, zashboardNoopServiceWorkerScript)
		return
	}
	if _, err := os.Stat(abs); err == nil {
		if rel == "index.html" {
			a.serveZashboardIndex(w, r, abs)
			return
		}
		http.ServeFile(w, r, abs)
		return
	}
	if rel != "index.html" {
		writeError(w, http.StatusNotFound, "not_found", "ui asset not found")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(mihomoFallbackDashboardHTML))
}

func (a *App) serveZashboardIndex(w http.ResponseWriter, r *http.Request, path string) {
	body, err := os.ReadFile(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	html := string(body)
	if !strings.Contains(html, "msm-free-zashboard-auto-backend") {
		script := a.zashboardAutoBackendScript(r)
		if strings.Contains(html, "</head>") {
			html = strings.Replace(html, "</head>", script+"</head>", 1)
		} else {
			html = script + html
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(html))
}

func (a *App) zashboardSetupURL(r *http.Request) string {
	return "/ui/#/setup?" + a.zashboardSetupQuery(r)
}

func (a *App) zashboardExternalSetupURL(r *http.Request) string {
	host := requestHostName(r)
	return "http://" + net.JoinHostPort(host, "9090") + "/ui/#/setup?" + a.zashboardSetupQuery(r)
}

func (a *App) zashboardSetupQuery(r *http.Request) string {
	host := requestHostName(r)
	q := "hostname=" + urlQueryEscape(host) + "&port=9090&disableUpgradeCore=1"
	if secret := a.mihomoSecret(); secret != "" {
		q += "&secret=" + urlQueryEscape(secret)
	}
	return q
}

func requestHostName(r *http.Request) string {
	host := r.Host
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwarded != "" {
		host = forwarded
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if host == "" {
		return "127.0.0.1"
	}
	return host
}

func serveJavaScript(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(body))
}

func (a *App) zashboardAutoBackendScript(r *http.Request) string {
	secret := a.mihomoSecret()
	if secret != "" {
		secret = urlQueryEscape(secret)
	}
	return strings.ReplaceAll(strings.ReplaceAll(zashboardAutoBackendScriptTemplate, "__MSM_FREE_HOST__", urlQueryEscape(requestHostName(r))), "__MSM_FREE_SECRET__", secret)
}

func urlQueryEscape(value string) string {
	return url.QueryEscape(value)
}

const zashboardAutoBackendScriptTemplate = `<script id="msm-free-zashboard-auto-backend">
;(function () {
  try {
    if (!window.localStorage) return
    var listKey = "setup/api-list"
    var activeKey = "setup/active-uuid"
    var presetHost = decodeURIComponent("__MSM_FREE_HOST__")
    var presetSecret = decodeURIComponent("__MSM_FREE_SECRET__")
    var host = presetHost || window.location.hostname || "127.0.0.1"
    var loopback = function (value) {
      return value === "127.0.0.1" || value === "localhost" || value === "::1" || value === "0.0.0.0"
    }
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
    var active = localStorage.getItem(activeKey) || ""
    var activeItem = list.find(function (item) { return item && item.uuid === active })
    if (activeItem && (!loopback(activeItem.host) || loopback(host))) return

    var matching = list.find(function (item) {
      return item && item.protocol === "http" && item.host === host && String(item.port) === "9090" && !item.secondaryPath
    })
    if (matching) {
      localStorage.setItem(activeKey, matching.uuid)
      return
    }

    var id = activeItem && activeItem.uuid ? activeItem.uuid : "msm-free-" + host.replace(/[^a-zA-Z0-9]/g, "-") + "-9090"
    var entry = {
      protocol: "http",
      secondaryPath: "",
      host: host,
      port: "9090",
      password: presetSecret,
      label: "msm-free",
      disableUpgradeCore: true,
      disableTunMode: false,
      uuid: id
    }
    if (activeItem) {
      Object.assign(activeItem, entry)
    } else {
      list.unshift(entry)
    }
    localStorage.setItem(listKey, JSON.stringify(list))
    localStorage.setItem(activeKey, id)
  } catch (err) {
    console.warn("msm-free zashboard backend preset failed", err)
  }
})()
</script>
`

const frontendSPARecoveryScript = `<script id="msm-free-spa-recovery">
;(function () {
  var reloadKey = "msm-free-spa-reload-at"
  function reloadOnce() {
    try {
      var last = Number(sessionStorage.getItem(reloadKey) || "0")
      if (Date.now() - last < 15000) return
      sessionStorage.setItem(reloadKey, String(Date.now()))
    } catch (_) {}
    var reload = function () { window.location.reload() }
    try {
      if ("caches" in window) {
        caches.keys().then(function (keys) {
          return Promise.all(keys.map(function (key) { return caches.delete(key) }))
        }).finally(reload)
        return
      }
    } catch (_) {}
    reload()
  }
  window.addEventListener("vite:preloadError", function (event) {
    event.preventDefault()
    reloadOnce()
  })
  window.addEventListener("unhandledrejection", function (event) {
    var reason = event && event.reason
    var message = String((reason && (reason.message || reason.stack)) || reason || "")
    if (message.indexOf("Failed to fetch dynamically imported module") !== -1 ||
        message.indexOf("Importing a module script failed") !== -1 ||
        message.indexOf("error loading dynamically imported module") !== -1) {
      reloadOnce()
    }
  })
})()
</script>
`

const zashboardRegisterSWCleanupScript = `;(function () {
  if (!("serviceWorker" in navigator)) return
  navigator.serviceWorker.getRegistrations().then(function (registrations) {
    registrations.forEach(function (registration) {
      if (registration.scope && registration.scope.indexOf("/ui/") !== -1) registration.unregister()
    })
  }).catch(function () {})
})()
`

const zashboardNoopServiceWorkerScript = `self.addEventListener("install", function (event) {
  self.skipWaiting()
})
self.addEventListener("activate", function (event) {
  event.waitUntil((async function () {
    if (self.caches) {
      var keys = await caches.keys()
      await Promise.all(keys.map(function (key) { return caches.delete(key) }))
    }
    if (self.registration && self.registration.unregister) await self.registration.unregister()
    if (self.clients && self.clients.claim) await self.clients.claim()
    if (self.clients && self.clients.matchAll) {
      var clients = await self.clients.matchAll({ type: "window" })
      clients.forEach(function (client) {
        if (client.url && client.url.indexOf("/ui/") !== -1 && client.navigate) client.navigate(client.url)
      })
    }
  })())
})
`

const mihomoFallbackDashboardHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>zashboard 未安装</title>
  <style>
    :root{color-scheme:light dark;font-family:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#f6f7fb;color:#111827}
    body{margin:0;min-height:100vh;display:grid;place-items:center;background:#f6f7fb}
    .panel{width:min(560px,calc(100vw - 40px));background:rgba(255,255,255,.92);border:1px solid rgba(148,163,184,.35);border-radius:8px;padding:24px;box-shadow:0 18px 45px rgba(15,23,42,.10)}
    h1{font-size:22px;margin:0 0 10px}
    p{margin:0 0 14px;color:#64748b;line-height:1.6}
    code{background:#eef2f7;border-radius:6px;padding:2px 6px}
    a{color:#0284c7;text-decoration:none}
    @media(prefers-color-scheme:dark){:root{background:#020617;color:#e5e7eb}body{background:#020617}.panel{background:rgba(15,23,42,.92);border-color:rgba(148,163,184,.25)}p{color:#94a3b8}code{background:#111827}}
  </style>
</head>
<body>
  <main class="panel">
    <h1>zashboard 还没有安装</h1>
    <p>Mihomo 的运行态 Web UI 现在由 <code>Zephyruso/zashboard</code> 接管。当前数据目录没有找到 <code>configs/mihomo/ui/index.html</code>，所以不能显示真实 zashboard。</p>
    <p>请在初始化流程下载 Mihomo 时同时安装 zashboard，或在组件更新页安装 <code>zashboard</code>。安装完成后再次打开 <code>/ui/</code> 会直接加载官方 zashboard 静态页面，并自动连接当前主机的 <code>9090</code> Mihomo 控制端口。</p>
    <p><a href="/settings">返回系统设置</a></p>
  </main>
</body>
</html>`

func serveLocalFile(w http.ResponseWriter, r *http.Request, path string) {
	if _, err := os.Stat(path); err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	http.ServeFile(w, r, filepath.Clean(path))
}
