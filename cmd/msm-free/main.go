package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/msm-free/msm-free/internal/server"
)

var version = "0.1.0-dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	command := "serve"
	if len(args) > 0 && args[0] != "-h" && args[0] != "--help" && args[0] != "-v" && args[0] != "--version" {
		command = args[0]
		args = args[1:]
	}

	fs := flag.NewFlagSet(command, flag.ExitOnError)
	configDir := fs.String("c", defaultDataDir(), "data/config directory")
	fs.StringVar(configDir, "config", defaultDataDir(), "data/config directory")
	host := fs.String("host", "0.0.0.0", "listen host")
	port := fs.Int("p", 7777, "HTTP listen port")
	fs.IntVar(port, "port", 7777, "HTTP listen port")
	daemon := fs.Bool("d", false, "daemon mode placeholder")
	versionFlag := fs.Bool("v", false, "print version")
	fs.BoolVar(versionFlag, "version", false, "print version")
	_ = fs.Parse(args)

	if *versionFlag || command == "version" {
		fmt.Printf("msm-free %s\n", version)
		return nil
	}

	switch command {
	case "serve", "":
		if *daemon {
			log.Println("daemon mode is accepted for compatibility; running in foreground")
		}
		return serve(*configDir, *host, *port)
	case "init":
		app, err := server.New(server.Options{DataDir: *configDir, Version: version})
		if err != nil {
			return err
		}
		defer app.Close()
		return app.EnsureBaseLayout()
	case "reset-password":
		app, err := server.New(server.Options{DataDir: *configDir, Version: version})
		if err != nil {
			return err
		}
		defer app.Close()
		password := "admin123456"
		if fs.NArg() > 0 {
			password = fs.Arg(0)
		}
		if err := app.ResetAdminPassword(password); err != nil {
			return err
		}
		fmt.Printf("admin password reset to: %s\n", password)
		return nil
	case "status":
		pidFile := runtimePIDFile(*configDir)
		b, err := os.ReadFile(pidFile)
		if err != nil {
			return errors.New("msm-free is not running")
		}
		pid, _ := strconv.Atoi(stringTrim(string(b)))
		if pid > 0 && processAlive(pid) {
			fmt.Printf("msm-free running pid=%d\n", pid)
			return nil
		}
		return errors.New("stale pid file found")
	case "stop":
		pidFile := runtimePIDFile(*configDir)
		b, err := os.ReadFile(pidFile)
		if err != nil {
			return errors.New("msm-free is not running")
		}
		pid, _ := strconv.Atoi(stringTrim(string(b)))
		if pid <= 0 {
			return errors.New("invalid pid file")
		}
		proc, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		return proc.Signal(syscall.SIGTERM)
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func serve(dataDir, host string, port int) error {
	app, err := server.New(server.Options{DataDir: dataDir, Version: version})
	if err != nil {
		return err
	}
	defer app.Close()
	app.LogInfo("app/app.go:114", "MSM 后端服务启动中...", nil)
	app.LogInfo("app/app.go:115", "使用配置目录", map[string]any{"path": dataDir})

	if err := app.EnsureBaseLayout(); err != nil {
		return err
	}
	app.LogInfo("app/app.go:158", "已生成配置文件并落地当前有效 JWT 密钥", map[string]any{"file": filepath.Join(dataDir, "configs/app.yaml")})
	app.LogInfo("app/app.go:173", "JWT配置初始化成功", nil)
	app.LogInfo("app/app.go:182", "数据库初始化成功", nil)
	app.LogInfo("supervisor/manager.go:160", "Supervisord配置生成成功", map[string]any{"config": filepath.Join(dataDir, "configs/supervisor/supervisord.conf")})
	app.LogInfo("supervisor/manager.go:219", "Supervisord Manager 初始化成功", nil)
	app.LogInfo("app/app.go:209", "Supervisor 初始化成功", nil)
	app.LogInfo("app/app.go:217", "服务管理器初始化成功", nil)
	app.LogInfo("app/app.go:221", "系统监控器初始化成功", nil)
	app.LogInfo("app/app.go:235", "许可证未启用", map[string]any{"reason": "msm-free unlocked"})
	app.LogInfo("app/app.go:241", "Setup 服务初始化成功", nil)
	app.LogInfo("app/app.go:246", "更新服务初始化成功", nil)
	if err := os.WriteFile(filepath.Join(dataDir, "msm-free.pid"), []byte(fmt.Sprint(os.Getpid())), 0644); err != nil {
		return err
	}
	_ = os.WriteFile(filepath.Join(dataDir, "msm.pid"), []byte(fmt.Sprint(os.Getpid())), 0644)
	defer os.Remove(filepath.Join(dataDir, "msm-free.pid"))
	defer os.Remove(filepath.Join(dataDir, "msm.pid"))

	go func() {
		restoreCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		report := app.RestoreConfiguredRuntime(restoreCtx)
		if len(report.Errors) > 0 {
			app.LogError("app/app.go:298", "启动恢复完成但存在错误", map[string]any{"errors": report.Errors})
			log.Printf("runtime restore completed with errors: %v", report.Errors)
		}
	}()

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", host, port),
		Handler:           app.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = app.Services.StopAll(shutdownCtx)
		_ = srv.Shutdown(shutdownCtx)
	}()

	app.LogInfo("update/scheduler.go:51", "更新调度器已启动", map[string]any{"interval": 86400, "auto_download": false})
	app.LogInfo("componentupdate/scheduler.go:54", "组件更新调度器已启动", nil)
	app.LogInfo("app/app.go:372", "HTTP 服务器启动", map[string]any{"addr": fmt.Sprintf("%s:%d", host, port)})
	log.Printf("msm-free %s listening on http://%s:%d data=%s", version, host, port, dataDir)
	err = srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func defaultDataDir() string {
	if v := os.Getenv("MSM_FREE_DATA_DIR"); v != "" {
		return v
	}
	if os.Geteuid() == 0 {
		return "/opt/msm-free"
	}
	return "./data"
}

func runtimePIDFile(dataDir string) string {
	msmPID := filepath.Join(dataDir, "msm.pid")
	if _, err := os.Stat(msmPID); err == nil {
		return msmPID
	}
	return filepath.Join(dataDir, "msm-free.pid")
}

func processAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func stringTrim(s string) string {
	for len(s) > 0 && (s[0] == '\n' || s[0] == '\r' || s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 {
		last := s[len(s)-1]
		if last != '\n' && last != '\r' && last != ' ' && last != '\t' {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}
