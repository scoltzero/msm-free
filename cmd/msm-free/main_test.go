package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestStopRuntimeTerminatesPIDAndRemovesPIDFiles(t *testing.T) {
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep command not available")
	}
	dataDir := t.TempDir()
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	})

	pidFile := filepath.Join(dataDir, "msm-free.pid")
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		t.Fatal(err)
	}
	if err := stopRuntime(dataDir, true, 3*time.Second, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Fatalf("pid file should be removed after stop, err=%v", err)
	}
}

func TestSafeRemoveAllRejectsBroadPaths(t *testing.T) {
	for _, path := range []string{"", ".", "/", "/opt", "/usr/local", "/mnt/user"} {
		if err := safeRemoveAll(path); err == nil {
			t.Fatalf("safeRemoveAll(%q) should reject broad path", path)
		}
	}
}
