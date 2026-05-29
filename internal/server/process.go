package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type ServiceManager struct {
	app   *App
	mu    sync.Mutex
	procs map[string]*exec.Cmd
}

type ServiceStatus struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Installed   bool   `json:"installed"`
	Running     bool   `json:"running"`
	PID         int    `json:"pid,omitempty"`
	Status      string `json:"status"`
	Version     string `json:"version,omitempty"`
	Uptime      int64  `json:"uptime,omitempty"`
	Memory      int64  `json:"memory,omitempty"`
	CPU         int64  `json:"cpu,omitempty"`
	BinaryPath  string `json:"binary_path,omitempty"`
	ConfigPath  string `json:"config_path,omitempty"`
	LogPath     string `json:"log_path,omitempty"`
	Error       string `json:"error,omitempty"`
}

func NewServiceManager(app *App) *ServiceManager {
	return &ServiceManager{app: app, procs: map[string]*exec.Cmd{}}
}

func (sm *ServiceManager) List() []ServiceStatus {
	return []ServiceStatus{sm.Status("mosdns"), sm.Status("mihomo")}
}

func (sm *ServiceManager) Status(name string) ServiceStatus {
	spec, err := sm.spec(name)
	if err != nil {
		return ServiceStatus{Name: name, Status: "unknown", Error: err.Error()}
	}
	status := ServiceStatus{Name: name, DisplayName: spec.DisplayName, BinaryPath: spec.Binary, ConfigPath: spec.Config, LogPath: spec.Stdout}
	if _, err := os.Stat(spec.Binary); err == nil {
		status.Installed = true
	}
	pid := readPID(spec.PIDFile)
	if pid > 0 && processAliveCross(pid) {
		status.Running = true
		status.PID = pid
		status.Status = "running"
		if metrics, ok := processResourceSnapshot(pid); ok {
			status.Uptime = metrics.Uptime
			status.Memory = metrics.Memory
			status.CPU = metrics.CPU
		} else if info, err := os.Stat(spec.PIDFile); err == nil {
			status.Uptime = int64(time.Since(info.ModTime()).Seconds())
		}
		return status
	}
	status.Status = "stopped"
	return status
}

func (sm *ServiceManager) Start(ctx context.Context, name string) (ServiceStatus, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	spec, err := sm.spec(name)
	if err != nil {
		return sm.Status(name), err
	}
	if st := sm.Status(name); st.Running {
		sm.setDesired(name, true)
		return st, nil
	}
	if _, err := os.Stat(spec.Binary); err != nil {
		return sm.Status(name), fmt.Errorf("%s binary not installed at %s", name, spec.Binary)
	}
	if err := os.MkdirAll(filepath.Dir(spec.Stdout), 0755); err != nil {
		return sm.Status(name), err
	}
	stdout, err := os.OpenFile(spec.Stdout, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return sm.Status(name), err
	}
	stderr, err := os.OpenFile(spec.Stderr, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		stdout.Close()
		return sm.Status(name), err
	}
	// Managed services must outlive the HTTP request that triggered them.
	// Stop/Restart owns shutdown through PID files and process signals.
	cmd := exec.CommandContext(context.Background(), spec.Binary, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
	if err := cmd.Start(); err != nil {
		stdout.Close()
		stderr.Close()
		return sm.Status(name), err
	}
	sm.procs[name] = cmd
	_ = os.WriteFile(spec.PIDFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644)
	go func() {
		_ = cmd.Wait()
		stdout.Close()
		stderr.Close()
		_ = os.Remove(spec.PIDFile)
		sm.mu.Lock()
		delete(sm.procs, name)
		sm.mu.Unlock()
	}()
	time.Sleep(300 * time.Millisecond)
	st := sm.Status(name)
	if st.Running {
		sm.setDesired(name, true)
	}
	return st, nil
}

func (sm *ServiceManager) Stop(ctx context.Context, name string) (ServiceStatus, error) {
	return sm.stop(ctx, name, true)
}

func (sm *ServiceManager) stop(ctx context.Context, name string, persistDesired bool) (ServiceStatus, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	spec, err := sm.spec(name)
	if err != nil {
		return sm.Status(name), err
	}
	pid := readPID(spec.PIDFile)
	if pid <= 0 {
		if persistDesired {
			sm.setDesired(name, false)
		}
		return sm.Status(name), nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		_ = os.Remove(spec.PIDFile)
		if persistDesired {
			sm.setDesired(name, false)
		}
		return sm.Status(name), nil
	}
	_ = proc.Signal(syscall.SIGTERM)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !processAliveCross(pid) {
			_ = os.Remove(spec.PIDFile)
			if persistDesired {
				sm.setDesired(name, false)
			}
			return sm.Status(name), nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	_ = proc.Signal(syscall.SIGKILL)
	_ = os.Remove(spec.PIDFile)
	if persistDesired {
		sm.setDesired(name, false)
	}
	return sm.Status(name), nil
}

func (sm *ServiceManager) Restart(ctx context.Context, name string) (ServiceStatus, error) {
	_, _ = sm.stop(ctx, name, false)
	return sm.Start(ctx, name)
}

func (sm *ServiceManager) StopAll(ctx context.Context) error {
	var errs []string
	for _, name := range []string{"mosdns", "mihomo"} {
		if _, err := sm.stop(ctx, name, false); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func (sm *ServiceManager) StartEnabled(ctx context.Context) {
	for _, name := range []string{"mosdns", "mihomo"} {
		if sm.app.setting(serviceDesiredKey(name), "") != "true" {
			continue
		}
		if _, err := sm.Start(ctx, name); err != nil {
			log.Printf("failed to restore %s service: %v", name, err)
		}
	}
}

func (sm *ServiceManager) setDesired(name string, enabled bool) {
	value := "false"
	if enabled {
		value = "true"
	}
	sm.app.setSetting(serviceDesiredKey(name), value)
}

func serviceDesiredKey(name string) string {
	if name == "proxy" || name == "clash" {
		name = "mihomo"
	}
	return "service." + name + ".enabled"
}

type serviceSpec struct {
	DisplayName string
	Binary      string
	Args        []string
	Dir         string
	Config      string
	Stdout      string
	Stderr      string
	PIDFile     string
}

func (sm *ServiceManager) spec(name string) (serviceSpec, error) {
	root := sm.app.DataDir
	switch name {
	case "mihomo", "proxy":
		bin := firstExisting(
			filepath.Join(root, "data/binaries/mihomo/mihomo"),
			filepath.Join(root, "data/binaries/mihomo/latest/mihomo"),
			filepath.Join(root, "data/binaries/mihomo/mihomo-linux-amd64"),
		)
		cfg := filepath.Join(root, "configs/mihomo/config.yaml")
		return serviceSpec{
			DisplayName: "Mihomo",
			Binary:      bin,
			Args:        []string{"-d", filepath.Join(root, "configs/mihomo"), "-f", cfg},
			Dir:         filepath.Join(root, "configs/mihomo"),
			Config:      cfg,
			Stdout:      filepath.Join(root, "logs/mihomo.out.log"),
			Stderr:      filepath.Join(root, "logs/mihomo.err.log"),
			PIDFile:     filepath.Join(root, "data/mihomo.pid"),
		}, nil
	case "mosdns":
		bin := firstExisting(
			filepath.Join(root, "data/binaries/mosdns/mosdns"),
			filepath.Join(root, "data/binaries/mosdns/latest/mosdns"),
		)
		cfgDir := filepath.Join(root, "configs/mosdns")
		return serviceSpec{
			DisplayName: "MosDNS",
			Binary:      bin,
			Args:        []string{"start", "--dir", cfgDir},
			Dir:         cfgDir,
			Config:      filepath.Join(cfgDir, "config.yaml"),
			Stdout:      filepath.Join(root, "logs/mosdns.out.log"),
			Stderr:      filepath.Join(root, "logs/mosdns.err.log"),
			PIDFile:     filepath.Join(root, "data/mosdns.pid"),
		}, nil
	default:
		return serviceSpec{}, fmt.Errorf("unknown service %s", name)
	}
}

func firstExisting(paths ...string) string {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return paths[0]
}

func readPID(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	return pid
}

func processAliveCross(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

func tailFile(path string, maxLines int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	buf := make([]string, 0, maxLines)
	for scanner.Scan() {
		line := scanner.Text()
		if len(buf) == maxLines {
			copy(buf, buf[1:])
			buf[maxLines-1] = line
		} else {
			buf = append(buf, line)
		}
	}
	return buf, scanner.Err()
}
