package server

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var allowedConfigRoots = []string{"configs", "logs", "backups"}

func (a *App) safePath(rel string) (string, error) {
	if rel == "" || rel == "." || rel == "/" {
		return a.DataDir, nil
	}
	rel = strings.TrimPrefix(rel, "/")
	clean := filepath.Clean(rel)
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", errors.New("invalid path")
	}
	ok := false
	for _, root := range allowedConfigRoots {
		if clean == root || strings.HasPrefix(clean, root+string(filepath.Separator)) {
			ok = true
			break
		}
	}
	if !ok {
		return "", errors.New("path outside allowed roots")
	}
	full := filepath.Join(a.DataDir, clean)
	base, _ := filepath.Abs(a.DataDir)
	abs, _ := filepath.Abs(full)
	if abs != base && !strings.HasPrefix(abs, base+string(filepath.Separator)) {
		return "", errors.New("path escapes data dir")
	}
	return abs, nil
}

func (a *App) readTextFile(rel string) (string, error) {
	path, err := a.safePath(rel)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(path)
	return string(b), err
}

func (a *App) writeTextFile(rel, content string) error {
	path, err := a.safePath(rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if old, err := os.ReadFile(path); err == nil {
		_, _ = a.DB.Exec(`insert into config_histories(service,file_path,content,comment,created_by,created_at,updated_at) values(?,?,?,?,?,?,?)`,
			serviceFromPath(rel), rel, string(old), "auto backup before save", "system", nowString(), nowString())
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func serviceFromPath(path string) string {
	switch {
	case strings.Contains(path, "mosdns"):
		return "mosdns"
	case strings.Contains(path, "mihomo"):
		return "mihomo"
	case strings.Contains(path, "network"):
		return "network"
	default:
		return "system"
	}
}

type FileNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Type     string     `json:"type"`
	Size     int64      `json:"size,omitempty"`
	Modified string     `json:"modified,omitempty"`
	Children []FileNode `json:"children,omitempty"`
}

func (a *App) fileTree(rel string, depth int) ([]FileNode, error) {
	root, err := a.safePath(rel)
	if err != nil {
		return nil, err
	}
	var nodes []FileNode
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		childRel := filepath.ToSlash(filepath.Join(rel, entry.Name()))
		node := FileNode{Name: entry.Name(), Path: childRel, Size: info.Size(), Modified: info.ModTime().Format("2006-01-02 15:04:05")}
		if entry.IsDir() {
			node.Type = "directory"
			if depth > 0 {
				children, _ := a.fileTree(childRel, depth-1)
				node.Children = children
			}
		} else {
			node.Type = "file"
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func copyFile(src, dst string, mode fs.FileMode) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, b, mode)
}
