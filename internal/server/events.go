package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (a *App) registerEvents(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/events", a.handleEvents)
	mux.HandleFunc("GET /api/v1/events/monitor", a.handleEvents)
	mux.HandleFunc("GET /api/v1/events/proxy", a.handleProxyEvents)
	mux.HandleFunc("GET /api/v1/events/mosdns", a.handleMosDNSEvents)
	mux.HandleFunc("GET /api/v1/events/mihomo", a.handleMihomoEvents)
	mux.HandleFunc("GET /api/v1/events/logs/{service}", a.handleLogEvents)
}

func (a *App) handleEvents(w http.ResponseWriter, r *http.Request) {
	a.sseLoopNamed(w, r, time.Second, "monitor", func() any {
		return map[string]any{
			"time":     time.Now(),
			"services": a.Services.List(),
			"ips":      localIPs(),
		}
	})
}

func (a *App) handleProxyEvents(w http.ResponseWriter, r *http.Request) {
	a.sseLoopNamed(w, r, time.Second, "proxy", func() any {
		data := a.proxySnapshot()
		data["time"] = time.Now()
		return data
	})
}

func (a *App) handleMosDNSEvents(w http.ResponseWriter, r *http.Request) {
	a.sseLoopNamed(w, r, 2*time.Second, "mosdns", func() any {
		data := a.mosDNSSnapshot(2000)
		data["time"] = time.Now()
		data["audit_ranks"] = map[string]any{"domain": data["top_domains"], "client": data["top_clients"], "domain_set": data["top_rules"]}
		return data
	})
}

func (a *App) handleMihomoEvents(w http.ResponseWriter, r *http.Request) {
	a.sseLoopNamed(w, r, 2*time.Second, "mihomo", func() any {
		data := a.mihomoSnapshot()
		data["time"] = time.Now()
		return data
	})
}

func (a *App) handleLogEvents(w http.ResponseWriter, r *http.Request) {
	service := normalizeServiceName(r.PathValue("service"))
	last := ""
	a.sseLoop(w, r, 1500*time.Millisecond, func() any {
		lines := filterLogLines(a.serviceLogLines(service, queryInt(r, "lines", 40)), r)
		content := strings.Join(lines, "\n")
		if content == last {
			return map[string]any{"service": service, "line": "", "lines": []string{}, "logs": []any{}, "content": content}
		}
		last = content
		line := ""
		if len(lines) > 0 {
			line = lines[len(lines)-1]
		}
		return map[string]any{"service": service, "line": line, "lines": lines, "logs": structuredLogLines(lines), "content": content}
	})
}

func (a *App) sseLoop(w http.ResponseWriter, r *http.Request, interval time.Duration, payload func() any) {
	a.sseLoopNamed(w, r, interval, "", payload)
}

func (a *App) sseLoopNamed(w http.ResponseWriter, r *http.Request, interval time.Duration, event string, payload func() any) {
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, _ := w.(http.Flusher)
	enc := json.NewEncoder(w)
	send := func(v any) bool {
		if event != "" {
			fmt.Fprintf(w, "event: %s\n", event)
		}
		fmt.Fprint(w, "data: ")
		if err := enc.Encode(v); err != nil {
			return false
		}
		fmt.Fprint(w, "\n")
		if flusher != nil {
			flusher.Flush()
		}
		return true
	}
	if !send(payload()) {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if !send(payload()) {
				return
			}
		}
	}
}
