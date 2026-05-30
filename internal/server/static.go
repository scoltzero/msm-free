package server

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:embed web/dist
var frontendFS embed.FS

func (a *App) registerStatic(mux *http.ServeMux) {
	mux.HandleFunc("GET /ui", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", http.StatusFound)
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
			serveZashboardIndex(w, abs)
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

func serveZashboardIndex(w http.ResponseWriter, path string) {
	body, err := os.ReadFile(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	html := string(body)
	if !strings.Contains(html, "msm-free-zashboard-auto-backend") {
		if strings.Contains(html, "</head>") {
			html = strings.Replace(html, "</head>", zashboardAutoBackendScript+"</head>", 1)
		} else {
			html = zashboardAutoBackendScript + html
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(html))
}

func serveJavaScript(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(body))
}

const zashboardAutoBackendScript = `<script id="msm-free-zashboard-auto-backend">
;(function () {
  try {
    if (!window.localStorage) return
    var listKey = "setup/api-list"
    var activeKey = "setup/active-uuid"
    var host = window.location.hostname || "127.0.0.1"
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
      password: "",
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
  <title>Mihomo Dashboard</title>
  <style>
    :root{color-scheme:light dark;font-family:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#f6f7fb;color:#111827}
    body{margin:0;min-height:100vh;background:linear-gradient(180deg,#f8fafc,#eef2f7)}
    .shell{max-width:1180px;margin:0 auto;padding:28px}
    header{display:flex;justify-content:space-between;gap:18px;align-items:center;margin-bottom:22px}
    h1{font-size:24px;line-height:1.2;margin:0}
    p{margin:6px 0 0;color:#64748b}
    .grid{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:14px}
    .card{background:rgba(255,255,255,.86);border:1px solid rgba(148,163,184,.28);border-radius:8px;padding:16px;box-shadow:0 10px 30px rgba(15,23,42,.08)}
    .span2{grid-column:span 2}
    .span4{grid-column:span 4}
    .label{font-size:12px;color:#64748b;margin-bottom:8px}
    .value{font-size:22px;font-weight:700;word-break:break-word}
    button,a.button{border:0;border-radius:8px;background:#111827;color:white;padding:10px 14px;text-decoration:none;cursor:pointer;font-size:14px}
    button.secondary{background:#e2e8f0;color:#111827}
    table{width:100%;border-collapse:collapse;font-size:13px}
    th,td{text-align:left;padding:10px;border-bottom:1px solid #e2e8f0;vertical-align:top}
    th{color:#64748b;font-weight:600}
    .toolbar{display:flex;gap:10px;align-items:center;flex-wrap:wrap}
    .ok{color:#047857}.bad{color:#b91c1c}.muted{color:#64748b}
    @media(max-width:860px){.grid{grid-template-columns:1fr}.span2,.span4{grid-column:auto}header{align-items:flex-start;flex-direction:column}}
    @media(prefers-color-scheme:dark){:root{background:#0f172a;color:#e5e7eb}body{background:linear-gradient(180deg,#111827,#020617)}.card{background:rgba(15,23,42,.88);border-color:rgba(148,163,184,.18)}th,td{border-color:#1f2937}p,.label,.muted{color:#94a3b8}button.secondary{background:#1f2937;color:#e5e7eb}}
  </style>
</head>
<body>
  <main class="shell">
    <header>
      <div>
        <h1>Mihomo Dashboard</h1>
        <p>内置轻量面板。安装 zashboard 后会自动优先显示真实 Dashboard。</p>
      </div>
      <div class="toolbar">
        <button id="refresh">刷新</button>
        <a class="button" href="/mihomo/proxies">返回代理节点</a>
      </div>
    </header>
    <section class="grid">
      <div class="card"><div class="label">运行模式</div><div class="value" id="mode">-</div></div>
      <div class="card"><div class="label">HTTP / Mixed</div><div class="value" id="ports">-</div></div>
      <div class="card"><div class="label">下载</div><div class="value" id="down">0 B/s</div></div>
      <div class="card"><div class="label">上传</div><div class="value" id="up">0 B/s</div></div>
      <div class="card span2"><div class="label">代理组</div><table><thead><tr><th>名称</th><th>类型</th><th>当前</th></tr></thead><tbody id="proxies"><tr><td colspan="3" class="muted">加载中</td></tr></tbody></table></div>
      <div class="card span2"><div class="label">连接</div><table><thead><tr><th>Host</th><th>规则</th><th>链路</th></tr></thead><tbody id="connections"><tr><td colspan="3" class="muted">加载中</td></tr></tbody></table></div>
      <div class="card span4"><div class="label">状态</div><div id="status" class="muted">等待刷新</div></div>
    </section>
  </main>
  <script>
    const token = localStorage.getItem("msm-token") || "";
    const headers = token ? { Authorization: "Bearer " + token } : {};
    const fmt = n => {
      n = Number(n || 0);
      if (n > 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + " MB/s";
      if (n > 1024) return (n / 1024).toFixed(1) + " KB/s";
      return n.toFixed(0) + " B/s";
    };
    async function api(path) {
      const res = await fetch("/api/v1" + path, { headers });
      const json = await res.json();
      return json.data ?? json;
    }
    async function refresh() {
      const status = document.getElementById("status");
      try {
        const [configs, proxies, traffic, connections] = await Promise.all([
          api("/mihomo/controller/configs"),
          api("/mihomo/proxies"),
          api("/mihomo/traffic"),
          api("/mihomo/connections")
        ]);
        document.getElementById("mode").textContent = configs.mode || "-";
        document.getElementById("ports").textContent = (configs.port || "-") + " / " + (configs["mixed-port"] || "-");
        document.getElementById("down").textContent = fmt(traffic.down);
        document.getElementById("up").textContent = fmt(traffic.up);
        const proxyRows = Object.values(proxies.proxies || {}).filter(p => p && Array.isArray(p.all)).slice(0, 12);
        document.getElementById("proxies").innerHTML = proxyRows.length ? proxyRows.map(p => "<tr><td>" + (p.name || "-") + "</td><td>" + (p.type || "-") + "</td><td>" + (p.now || "-") + "</td></tr>").join("") : "<tr><td colspan=\"3\" class=\"muted\">暂无代理组</td></tr>";
        const connRows = (connections.connections || []).slice(0, 12);
        document.getElementById("connections").innerHTML = connRows.length ? connRows.map(c => "<tr><td>" + ((c.metadata && (c.metadata.host || c.metadata.destinationIP)) || "-") + "</td><td>" + (c.rule || "-") + "</td><td>" + ((c.chains || []).join(" / ") || "-") + "</td></tr>").join("") : "<tr><td colspan=\"3\" class=\"muted\">暂无连接</td></tr>";
        status.textContent = "已刷新";
        status.className = "ok";
      } catch (err) {
        status.textContent = err.message || String(err);
        status.className = "bad";
      }
    }
    document.getElementById("refresh").addEventListener("click", refresh);
    refresh();
    setInterval(refresh, 5000);
  </script>
</body>
</html>`

func serveLocalFile(w http.ResponseWriter, r *http.Request, path string) {
	if _, err := os.Stat(path); err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	http.ServeFile(w, r, filepath.Clean(path))
}
