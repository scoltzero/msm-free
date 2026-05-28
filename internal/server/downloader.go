package server

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
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

func componentDownloadURL(component string) string {
	switch component {
	case "mihomo":
		return "https://github.com/baozaodetudou/mssb/releases/download/mihomo/mihomo-meta-linux-amd64.tar.gz"
	case "mosdns":
		return "https://github.com/baozaodetudou/mssb/releases/download/mosdns/mosdns-linux-amd64.zip"
	case "zashboard", "ui":
		return "https://github.com/Zephyruso/zashboard/archive/refs/heads/gh-pages.zip"
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
		emit(DownloadEvent{Status: "skipped", Progress: 100, Message: component + " already installed"})
		return nil
	}
	url := componentDownloadURL(component)
	if url == "" {
		return fmt.Errorf("no download URL for %s", component)
	}
	emit(DownloadEvent{Status: "running", Progress: 5, Message: "downloading " + url})
	tmp := filepath.Join(a.DataDir, "data", component+".download")
	_ = os.Remove(tmp)
	if err := downloadFile(url, tmp); err != nil {
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

func downloadFile(url, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
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
	_, err = io.Copy(out, resp.Body)
	return err
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
