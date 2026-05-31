package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += int64(n)
	return n, err
}

func (r *statusRecorder) Flush() {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("response writer does not support hijack")
	}
	return h.Hijack()
}

func (r *statusRecorder) Push(target string, opts *http.PushOptions) error {
	if p, ok := r.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

func (r *statusRecorder) statusCode() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func (a *App) LogInfo(caller, msg string, fields map[string]any) {
	a.logMSM("info", caller, msg, fields)
}

func (a *App) LogWarn(caller, msg string, fields map[string]any) {
	a.logMSM("warn", caller, msg, fields)
}

func (a *App) LogError(caller, msg string, fields map[string]any) {
	a.logMSM("error", caller, msg, fields)
}

func (a *App) logMSM(level, caller, msg string, fields map[string]any) {
	if a == nil || a.DataDir == "" {
		return
	}
	entry := map[string]any{
		"level":  strings.ToLower(strings.TrimSpace(firstNonEmpty(level, "info"))),
		"time":   time.Now().Format("2006-01-02T15:04:05.000-0700"),
		"caller": caller,
		"msg":    msg,
	}
	for key, value := range fields {
		if key == "" || key == "level" || key == "time" || key == "caller" || key == "msg" {
			continue
		}
		entry[key] = value
	}
	b, err := json.Marshal(entry)
	if err != nil {
		return
	}
	path := filepath.Join(a.DataDir, "logs/msm.log")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	a.appLogMu.Lock()
	defer a.appLogMu.Unlock()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(b, '\n'))
}

func (a *App) logHTTPRequest(r *http.Request, status int, latency time.Duration) {
	if a == nil || r == nil {
		return
	}
	requestURI := r.URL.RequestURI()
	if requestURI == "" {
		requestURI = r.URL.Path
	}
	clientIP := clientIPFromRequest(r)
	line := fmt.Sprintf("[GIN] %s | %3d | %13s | %15s | %-7s %q",
		time.Now().Format("2006/01/02 - 15:04:05"),
		status,
		latency.String(),
		clientIP,
		r.Method,
		requestURI,
	)
	a.LogInfo("app/app.go:945", "gin", map[string]any{
		"message": line,
		"status":  status,
		"latency": latency.String(),
		"ip":      clientIP,
		"method":  r.Method,
		"path":    requestURI,
	})
}

func clientIPFromRequest(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		if idx := strings.Index(forwarded, ","); idx >= 0 {
			forwarded = forwarded[:idx]
		}
		return strings.TrimSpace(forwarded)
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
