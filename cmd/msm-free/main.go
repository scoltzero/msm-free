package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
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
	prefix := fs.String("prefix", "/usr/local", "install prefix for uninstall")
	serviceName := fs.String("service-name", "msm-free", "systemd service name")
	aliasName := fs.String("alias-name", "msm", "compatibility CLI alias name")
	purge := fs.Bool("purge", false, "remove data directory during uninstall")
	wait := fs.Bool("wait", true, "wait for stop to complete")
	force := fs.Bool("force", false, "force kill process if graceful stop times out")
	timeout := fs.Duration("timeout", 15*time.Second, "stop/uninstall timeout")
	lines := fs.Int("n", 100, "log lines")
	fs.IntVar(lines, "lines", 100, "log lines")
	repo := fs.String("repo", defaultGitHubRepo(), "GitHub repository for update")
	updateURL := fs.String("url", "", "release tarball URL for update")
	daemon := fs.Bool("d", false, "daemon mode placeholder")
	versionFlag := fs.Bool("v", false, "print version")
	fs.BoolVar(versionFlag, "version", false, "print version")
	helpAll := fs.Bool("help-all", false, "print full help")
	_ = fs.Parse(args)

	if *versionFlag || command == "version" {
		fmt.Printf("msm-free %s\n", version)
		return nil
	}
	if command == "help" || *helpAll {
		printUsage()
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
		return printStatus(*configDir, *serviceName)
	case "stop":
		return stopRuntime(*configDir, *wait, *timeout, *force)
	case "restart":
		return restartRuntime(*configDir, *host, *port, *serviceName, *timeout, *force)
	case "logs":
		service := "msm"
		if fs.NArg() > 0 {
			service = fs.Arg(0)
		}
		return printLogs(*configDir, *serviceName, service, *lines)
	case "doctor":
		return runDoctor(*configDir, *serviceName)
	case "update":
		return updateRuntime(updateOptions{Repo: *repo, URL: *updateURL, Prefix: *prefix, DataDir: *configDir, ServiceName: *serviceName})
	case "service":
		action := ""
		if fs.NArg() > 0 {
			action = fs.Arg(0)
		}
		return serviceCommand(action, serviceOptions{Prefix: *prefix, DataDir: *configDir, Host: *host, Port: *port, ServiceName: *serviceName})
	case "license":
		action := "status"
		if fs.NArg() > 0 {
			action = fs.Arg(0)
		}
		return licenseCommand(action)
	case "uninstall":
		return uninstallRuntime(uninstallOptions{
			Prefix:      *prefix,
			DataDir:     *configDir,
			ServiceName: *serviceName,
			AliasName:   *aliasName,
			Purge:       *purge,
			Timeout:     *timeout,
		})
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func printUsage() {
	fmt.Print(`Usage:
  msm serve [--config /opt/msm-free] [--host 0.0.0.0] [--port 7777]
  msm init [--config /opt/msm-free]
  msm status [--config /opt/msm-free]
  msm restart [--config /opt/msm-free]
  msm stop [--config /opt/msm-free] [--timeout 15s] [--force]
  msm logs [--lines 100] [msm|mosdns|mihomo]
  msm doctor [--config /opt/msm-free]
  msm update [--repo scoltzero/msm-free] [--url https://.../msm-free-linux-amd64.tar.gz]
  msm uninstall [--config /opt/msm-free] [--prefix /usr/local] [--service-name msm-free] [--purge]
  msm reset-password [--config /opt/msm-free] [password]
  msm service install|uninstall [--config /opt/msm-free]
  msm license status|fingerprint
  msm version

Notes:
  msm-free and msm are the same CLI when the installer registers the msm alias.
  stop sends SIGTERM to the running msm-free process and waits for MosDNS/Mihomo child services to exit.
  uninstall removes the systemd unit and binary. It keeps the data directory unless --purge is provided.
`)
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

func stopRuntime(dataDir string, wait bool, timeout time.Duration, force bool) error {
	pidFile := runtimePIDFile(dataDir)
	b, err := os.ReadFile(pidFile)
	if err != nil {
		return errors.New("msm-free is not running")
	}
	pid, _ := strconv.Atoi(stringTrim(string(b)))
	if pid <= 0 {
		return errors.New("invalid pid file")
	}
	if !processAlive(pid) {
		removeRuntimePIDFiles(dataDir)
		return errors.New("stale pid file found")
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return err
	}
	if !wait {
		fmt.Printf("sent stop signal to msm-free pid=%d\n", pid)
		return nil
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for processAlive(pid) {
		if time.Now().After(deadline) {
			if force {
				_ = proc.Signal(syscall.SIGKILL)
				break
			}
			return fmt.Errorf("timed out waiting for msm-free pid=%d to stop", pid)
		}
		time.Sleep(200 * time.Millisecond)
	}
	removeRuntimePIDFiles(dataDir)
	fmt.Printf("msm-free stopped pid=%d\n", pid)
	return nil
}

func printStatus(dataDir, serviceName string) error {
	fmt.Printf("msm-free %s\n", version)
	fmt.Printf("data: %s\n", dataDir)
	pid := runtimePID(dataDir)
	running := pid > 0 && processAlive(pid)
	if running {
		fmt.Printf("app: running pid=%d\n", pid)
	} else {
		fmt.Println("app: stopped")
	}
	if systemdUnitExists(serviceName) {
		fmt.Printf("systemd: %s\n", strings.TrimSpace(commandOutput("systemctl", "is-active", serviceName)))
	}
	app, err := server.New(server.Options{DataDir: dataDir, Version: version})
	if err == nil {
		defer app.Close()
		for _, st := range app.Services.List() {
			state := "stopped"
			if st.Running {
				state = fmt.Sprintf("running pid=%d", st.PID)
			} else if !st.Installed {
				state = "not-installed"
			}
			fmt.Printf("%s: %s\n", st.Name, state)
		}
	}
	if !running && !strings.Contains(commandOutput("systemctl", "is-active", serviceName), "active") {
		return errors.New("msm-free is not running")
	}
	return nil
}

func restartRuntime(dataDir, host string, port int, serviceName string, timeout time.Duration, force bool) error {
	if systemdUnitExists(serviceName) {
		cmd := exec.Command("systemctl", "restart", serviceName)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
		fmt.Printf("restarted %s via systemd\n", serviceName)
		return nil
	}
	_ = stopRuntime(dataDir, true, timeout, force)
	return startDetached(dataDir, host, port)
}

func startDetached(dataDir, host string, port int) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "logs"), 0755); err != nil {
		return err
	}
	logPath := filepath.Join(dataDir, "logs", "msm-free.cli.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "serve", "--config", dataDir, "--host", host, "--port", strconv.Itoa(port))
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return err
	}
	if err := cmd.Process.Release(); err != nil {
		_ = logFile.Close()
		return err
	}
	fmt.Printf("started msm-free in background, log=%s\n", logPath)
	return nil
}

func printLogs(dataDir, serviceName, service string, lines int) error {
	if lines <= 0 {
		lines = 100
	}
	service = normalizeCLIService(service)
	if service == "msm" && commandExists("journalctl") && systemdUnitExists(serviceName) {
		cmd := exec.Command("journalctl", "-u", serviceName, "-n", strconv.Itoa(lines), "--no-pager")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	paths := cliLogPaths(dataDir, service)
	found := false
	for _, path := range paths {
		if !fileExists(path) {
			continue
		}
		found = true
		fmt.Printf("==> %s <==\n", path)
		for _, line := range tailFile(path, lines) {
			fmt.Println(line)
		}
	}
	if !found {
		return fmt.Errorf("no log files found for %s", service)
	}
	return nil
}

func runDoctor(dataDir, serviceName string) error {
	fmt.Printf("msm-free doctor\n")
	fmt.Printf("version: %s\n", version)
	fmt.Printf("data: %s\n", dataDir)
	fmt.Printf("root: %t\n", os.Geteuid() == 0)
	if systemdUnitExists(serviceName) {
		fmt.Printf("systemd %s: %s\n", serviceName, strings.TrimSpace(commandOutput("systemctl", "is-active", serviceName)))
	} else {
		fmt.Printf("systemd %s: not-installed\n", serviceName)
	}
	app, err := server.New(server.Options{DataDir: dataDir, Version: version})
	if err != nil {
		return err
	}
	defer app.Close()
	for _, dir := range []string{"configs/mosdns", "configs/mihomo", "configs/network", "logs", "database", "data/binaries"} {
		path := filepath.Join(dataDir, dir)
		status := "ok"
		if _, err := os.Stat(path); err != nil {
			status = err.Error()
		}
		fmt.Printf("dir %-24s %s\n", dir, status)
	}
	for _, st := range app.Services.List() {
		status := st.Status
		if !st.Installed {
			status = "not-installed"
		}
		fmt.Printf("service %-8s %-13s pid=%d binary=%s\n", st.Name, status, st.PID, st.BinaryPath)
	}
	for _, item := range []struct {
		Name string
		Port string
	}{
		{"web", "7777"}, {"mihomo-controller", "9090"}, {"mosdns-api", "9099"}, {"dns", "53"},
		{"http", "7890"}, {"socks", "7891"}, {"mixed", "7892"}, {"tproxy", "7896"}, {"redirect", "7877"},
	} {
		fmt.Printf("port %-18s :%-5s %s\n", item.Name, item.Port, portStatus(item.Port))
	}
	for _, name := range []string{"nft", "ip", "ss"} {
		if commandExists(name) {
			fmt.Printf("command %-8s ok\n", name)
		} else {
			fmt.Printf("command %-8s missing\n", name)
		}
	}
	return nil
}

type uninstallOptions struct {
	Prefix      string
	DataDir     string
	ServiceName string
	AliasName   string
	Purge       bool
	Timeout     time.Duration
}

func uninstallRuntime(opts uninstallOptions) error {
	if os.Geteuid() != 0 {
		return errors.New("uninstall must be run as root")
	}
	if isUnraidRuntime() {
		return errors.New("on Unraid, remove msm-free from the WebGUI plugin page; application data is kept under /mnt/user/appdata/msm-free")
	}
	if opts.Prefix == "" {
		opts.Prefix = "/usr/local"
	}
	if opts.DataDir == "" {
		opts.DataDir = defaultDataDir()
	}
	if opts.ServiceName == "" {
		opts.ServiceName = "msm-free"
	}
	if opts.AliasName == "" {
		opts.AliasName = "msm"
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 15 * time.Second
	}

	servicePath := filepath.Join("/etc/systemd/system", opts.ServiceName+".service")
	if commandExists("systemctl") && fileExists(servicePath) {
		_ = runQuiet("systemctl", "stop", opts.ServiceName)
		_ = runQuiet("systemctl", "disable", opts.ServiceName)
		if err := os.Remove(servicePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		_ = runQuiet("systemctl", "daemon-reload")
		_ = runQuiet("systemctl", "reset-failed", opts.ServiceName)
	} else {
		_ = stopRuntime(opts.DataDir, true, opts.Timeout, true)
	}

	binDest := filepath.Join(opts.Prefix, "bin", "msm-free")
	if err := os.Remove(binDest); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	removeAliasIfOwned(filepath.Join(opts.Prefix, "bin", opts.AliasName), binDest)

	if opts.Purge {
		if err := safeRemoveAll(opts.DataDir); err != nil {
			return err
		}
		fmt.Printf("removed msm-free binary, service, and data directory: %s\n", opts.DataDir)
		return nil
	}
	fmt.Printf("removed msm-free binary and service\n")
	fmt.Printf("kept data directory: %s\n", opts.DataDir)
	return nil
}

type updateOptions struct {
	Repo        string
	URL         string
	Prefix      string
	DataDir     string
	ServiceName string
}

func updateRuntime(opts updateOptions) error {
	if os.Geteuid() != 0 {
		return errors.New("update must be run as root")
	}
	if isUnraidRuntime() {
		return errors.New("on Unraid, update msm-free from the WebGUI plugin page instead of the Linux tarball updater")
	}
	if opts.Repo == "" {
		opts.Repo = defaultGitHubRepo()
	}
	if opts.URL == "" {
		opts.URL = "https://github.com/" + opts.Repo + "/releases/latest/download/msm-free-linux-amd64.tar.gz"
	}
	if _, err := url.ParseRequestURI(opts.URL); err != nil {
		return fmt.Errorf("invalid update URL: %w", err)
	}
	tmp, err := os.MkdirTemp("", "msm-free-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	archivePath := filepath.Join(tmp, "msm-free-linux-amd64.tar.gz")
	fmt.Printf("downloading %s\n", opts.URL)
	if err := downloadFile(opts.URL, archivePath); err != nil {
		return err
	}
	if err := extractTarGZ(archivePath, tmp); err != nil {
		return err
	}
	installScript, err := findFile(tmp, "install.sh")
	if err != nil {
		return err
	}
	args := []string{installScript, "--prefix", opts.Prefix, "--data-dir", opts.DataDir, "--service-name", opts.ServiceName}
	cmd := exec.Command("sh", args...)
	cmd.Dir = filepath.Dir(installScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("running installer from %s\n", filepath.Dir(installScript))
	return cmd.Run()
}

type serviceOptions struct {
	Prefix      string
	DataDir     string
	Host        string
	Port        int
	ServiceName string
}

func serviceCommand(action string, opts serviceOptions) error {
	switch action {
	case "install":
		return installSystemdService(opts)
	case "uninstall", "remove":
		return removeSystemdService(opts.ServiceName)
	case "status":
		if !systemdUnitExists(opts.ServiceName) {
			return fmt.Errorf("systemd service %s is not installed", opts.ServiceName)
		}
		cmd := exec.Command("systemctl", "status", opts.ServiceName, "--no-pager")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return errors.New("usage: msm service install|uninstall|status")
	}
}

func installSystemdService(opts serviceOptions) error {
	if os.Geteuid() != 0 {
		return errors.New("service install must be run as root")
	}
	if isUnraidRuntime() {
		return errors.New("on Unraid, use /etc/rc.d/rc.msm-free and the WebGUI plugin page instead of systemd service install")
	}
	if opts.ServiceName == "" {
		opts.ServiceName = "msm-free"
	}
	binDest := filepath.Join(opts.Prefix, "bin", "msm-free")
	if !fileExists(binDest) {
		if exe, err := os.Executable(); err == nil {
			binDest = exe
		}
	}
	if err := os.MkdirAll(opts.DataDir, 0755); err != nil {
		return err
	}
	servicePath := filepath.Join("/etc/systemd/system", opts.ServiceName+".service")
	body := fmt.Sprintf(`[Unit]
Description=msm-free service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=%s
Environment=MSM_FREE_DATA_DIR=%s
ExecStart=%s serve --config %s --host %s --port %d
Restart=on-failure
RestartSec=2
TimeoutStopSec=30
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
`, opts.DataDir, opts.DataDir, binDest, opts.DataDir, opts.Host, opts.Port)
	if err := os.WriteFile(servicePath, []byte(body), 0644); err != nil {
		return err
	}
	if err := runVisible("systemctl", "daemon-reload"); err != nil {
		return err
	}
	if err := runVisible("systemctl", "enable", opts.ServiceName); err != nil {
		return err
	}
	fmt.Printf("installed systemd service: %s\n", opts.ServiceName)
	return nil
}

func removeSystemdService(serviceName string) error {
	if os.Geteuid() != 0 {
		return errors.New("service uninstall must be run as root")
	}
	if isUnraidRuntime() {
		return errors.New("on Unraid, remove msm-free from the WebGUI plugin page instead of systemd service uninstall")
	}
	servicePath := filepath.Join("/etc/systemd/system", serviceName+".service")
	_ = runQuiet("systemctl", "stop", serviceName)
	_ = runQuiet("systemctl", "disable", serviceName)
	if err := os.Remove(servicePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	_ = runQuiet("systemctl", "daemon-reload")
	fmt.Printf("removed systemd service: %s\n", serviceName)
	return nil
}

func licenseCommand(action string) error {
	switch action {
	case "status", "":
		fmt.Println("license: free/unlocked")
		return nil
	case "fingerprint":
		host, _ := os.Hostname()
		sum := sha256.Sum256([]byte(host + "|" + defaultDataDir()))
		fmt.Println(hex.EncodeToString(sum[:]))
		return nil
	case "activate", "deactivate", "bind", "unbind", "info":
		fmt.Println("license: free/unlocked; commercial license commands are not required in msm-free")
		return nil
	default:
		return errors.New("usage: msm license status|fingerprint")
	}
}

func removeRuntimePIDFiles(dataDir string) {
	_ = os.Remove(filepath.Join(dataDir, "msm.pid"))
	_ = os.Remove(filepath.Join(dataDir, "msm-free.pid"))
}

func removeAliasIfOwned(aliasPath, binPath string) {
	target, err := os.Readlink(aliasPath)
	if err != nil {
		return
	}
	if target == binPath || target == filepath.Base(binPath) {
		_ = os.Remove(aliasPath)
	}
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func runQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func safeRemoveAll(path string) error {
	clean := filepath.Clean(path)
	switch clean {
	case "", ".", "/", "/opt", "/usr", "/usr/local", "/mnt", "/mnt/user", "/mnt/cache":
		return fmt.Errorf("refusing to purge unsafe data directory: %s", path)
	}
	return os.RemoveAll(clean)
}

func runtimePID(dataDir string) int {
	b, err := os.ReadFile(runtimePIDFile(dataDir))
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(stringTrim(string(b)))
	return pid
}

func systemdUnitExists(serviceName string) bool {
	return commandExists("systemctl") && fileExists(filepath.Join("/etc/systemd/system", serviceName+".service"))
}

func commandOutput(name string, args ...string) string {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out))
	}
	return strings.TrimSpace(string(out))
}

func normalizeCLIService(service string) string {
	switch strings.ToLower(strings.TrimSpace(service)) {
	case "", "app", "msm-free", "web", "server":
		return "msm"
	case "proxy", "clash":
		return "mihomo"
	default:
		return strings.ToLower(strings.TrimSpace(service))
	}
}

func cliLogPaths(dataDir, service string) []string {
	switch normalizeCLIService(service) {
	case "mosdns":
		return []string{
			filepath.Join(dataDir, "logs", "mosdns.out.log"),
			filepath.Join(dataDir, "logs", "mosdns.err.log"),
			filepath.Join(dataDir, "logs", "mosdns.log"),
			filepath.Join(dataDir, "configs", "logs", "mosdns.log"),
		}
	case "mihomo":
		return []string{
			filepath.Join(dataDir, "logs", "mihomo.out.log"),
			filepath.Join(dataDir, "logs", "mihomo.err.log"),
			filepath.Join(dataDir, "logs", "mihomo.log"),
		}
	default:
		return []string{
			filepath.Join(dataDir, "logs", "msm.log"),
			filepath.Join(dataDir, "logs", "msm-free.unraid.log"),
			filepath.Join(dataDir, "logs", "msm-free.cli.log"),
		}
	}
}

func tailFile(path string, n int) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if n > 0 && len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

func portStatus(port string) string {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", port), 500*time.Millisecond)
	if err != nil {
		return "closed"
	}
	_ = conn.Close()
	return "open"
}

func downloadFile(rawURL, dest string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	res, err := client.Get(rawURL)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("download failed: HTTP %d", res.StatusCode)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, res.Body)
	return err
}

func extractTarGZ(archivePath, dest string) error {
	f, err := os.Open(archivePath)
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
	base, _ := filepath.Abs(dest)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.Clean(hdr.Name))
		abs, _ := filepath.Abs(target)
		if abs != base && !strings.HasPrefix(abs, base+string(filepath.Separator)) {
			return fmt.Errorf("archive path escapes destination: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(abs, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(abs, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
}

func findFile(root, name string) (string, error) {
	var found string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found != "" {
			return err
		}
		if !d.IsDir() && d.Name() == name {
			found = path
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("%s not found under %s", name, root)
	}
	return found, nil
}

func runVisible(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func defaultGitHubRepo() string {
	if v := strings.TrimSpace(os.Getenv("MSM_FREE_GITHUB_REPO")); v != "" {
		return v
	}
	return "scoltzero/msm-free"
}

func isUnraidRuntime() bool {
	if fileExists("/etc/unraid-version") || fileExists("/usr/local/sbin/emhttp") || fileExists("/boot/config/plugins") {
		return true
	}
	if strings.Contains(strings.ToLower(os.Getenv("UNRAID_VERSION")), "unraid") {
		return true
	}
	return false
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
