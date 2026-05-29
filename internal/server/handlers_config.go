package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func (a *App) handleConfigTree(w http.ResponseWriter, r *http.Request) {
	root := r.URL.Query().Get("path")
	if root == "" {
		root = "configs"
	}
	nodes, err := a.fileTree(root, 4)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "tree": nodes, "data": nodes})
}

func (a *App) handleConfigFile(w http.ResponseWriter, r *http.Request) {
	rel := normalizeConfigRel(r.URL.Query().Get("path"))
	content, err := a.readTextFile(rel)
	if err != nil {
		writeError(w, http.StatusBadRequest, "read_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "path": rel, "content": content})
}

func (a *App) handleConfigFilePut(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path     string `json:"path"`
		FilePath string `json:"file_path"`
		Content  string `json:"content"`
		Comment  string `json:"comment"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Path == "" {
		req.Path = req.FilePath
	}
	req.Path = normalizeConfigRel(req.Path)
	if err := a.writeTextFile(req.Path, req.Content); err != nil {
		writeError(w, http.StatusBadRequest, "write_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleConfigFileCreate(w http.ResponseWriter, r *http.Request) {
	a.handleConfigFilePut(w, r)
}

func (a *App) handleConfigFileDelete(w http.ResponseWriter, r *http.Request) {
	rel := normalizeConfigRel(r.URL.Query().Get("path"))
	path, err := a.safePath(rel)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	if err := os.RemoveAll(path); err != nil {
		writeError(w, http.StatusBadRequest, "delete_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleConfigDirectory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	req.Path = normalizeConfigRel(req.Path)
	path, err := a.safePath(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		writeError(w, http.StatusBadRequest, "mkdir_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleConfigCopy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source     string `json:"source"`
		Target     string `json:"target"`
		SourcePath string `json:"source_path"`
		TargetPath string `json:"target_path"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Source == "" {
		req.Source = req.SourcePath
	}
	if req.Target == "" {
		req.Target = req.TargetPath
	}
	req.Source = normalizeConfigRel(req.Source)
	req.Target = normalizeConfigRel(req.Target)
	src, err := a.safePath(req.Source)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	dst, err := a.safePath(req.Target)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	info, err := os.Stat(src)
	if err != nil {
		writeError(w, http.StatusBadRequest, "copy_failed", err.Error())
		return
	}
	if err := copyFile(src, dst, info.Mode()); err != nil {
		writeError(w, http.StatusBadRequest, "copy_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleConfigRename(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source  string `json:"source"`
		Target  string `json:"target"`
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Source == "" {
		req.Source = req.OldPath
	}
	if req.Target == "" {
		req.Target = req.NewPath
	}
	req.Source = normalizeConfigRel(req.Source)
	req.Target = normalizeConfigRel(req.Target)
	src, err := a.safePath(req.Source)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	dst, err := a.safePath(req.Target)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	if err := os.Rename(src, dst); err != nil {
		writeError(w, http.StatusBadRequest, "rename_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleConfigValidate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path     string `json:"path"`
		Service  string `json:"service"`
		FilePath string `json:"file_path"`
		Content  string `json:"content"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Path == "" {
		req.Path = req.FilePath
	}
	if req.Path == "" && req.Service != "" {
		req.Path = filepath.ToSlash(filepath.Join("configs", normalizeServiceName(req.Service), "config.yaml"))
	}
	req.Path = normalizeConfigRel(req.Path)
	if req.Content == "" && req.Path != "" {
		req.Content, _ = a.readTextFile(req.Path)
	}
	if strings.HasSuffix(req.Path, ".yaml") || strings.HasSuffix(req.Path, ".yml") {
		var v any
		if err := yaml.Unmarshal([]byte(req.Content), &v); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"success": true, "valid": false, "error": err.Error()})
			return
		}
	}
	if strings.HasSuffix(req.Path, ".json") {
		var v any
		if err := json.Unmarshal([]byte(req.Content), &v); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"success": true, "valid": false, "error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "valid": true})
}

func (a *App) handleConfigDownload(w http.ResponseWriter, r *http.Request) {
	rel := normalizeConfigRel(r.URL.Query().Get("path"))
	path, err := a.safePath(rel)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	serveLocalFile(w, r, path)
}

func (a *App) handleConfigUpload(w http.ResponseWriter, r *http.Request) {
	rel := normalizeConfigRel(r.URL.Query().Get("path"))
	if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(64 << 20); err != nil {
			writeError(w, http.StatusBadRequest, "bad_upload", err.Error())
			return
		}
		if rel == "" {
			rel = normalizeConfigRel(r.FormValue("path"))
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_upload", err.Error())
			return
		}
		defer file.Close()
		if rel == "" {
			rel = filepath.ToSlash(filepath.Join("configs", header.Filename))
		}
		rel = normalizeConfigRel(rel)
		path, err := a.safePath(rel)
		if err != nil {
			writeError(w, http.StatusBadRequest, "path_error", err.Error())
			return
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			writeError(w, http.StatusInternalServerError, "mkdir_failed", err.Error())
			return
		}
		out, err := os.Create(path)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "create_failed", err.Error())
			return
		}
		defer out.Close()
		if _, err := io.Copy(out, io.LimitReader(file, 64<<20)); err != nil {
			writeError(w, http.StatusInternalServerError, "upload_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "path": rel})
		return
	}
	if rel == "" {
		rel = "configs/uploaded-" + strconv.FormatInt(time.Now().Unix(), 10)
	}
	rel = normalizeConfigRel(rel)
	path, err := a.safePath(rel)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "mkdir_failed", err.Error())
		return
	}
	out, err := os.Create(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}
	defer out.Close()
	_, err = io.Copy(out, io.LimitReader(r.Body, 64<<20))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "upload_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "path": rel})
}

func (a *App) handleHistoryCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Service  string `json:"service"`
		Path     string `json:"path"`
		FilePath string `json:"file_path"`
		Content  string `json:"content"`
		Comment  string `json:"comment"`
		IsStable bool   `json:"is_stable"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Path == "" {
		req.Path = req.FilePath
	}
	req.Path = normalizeConfigRel(req.Path)
	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "path required")
		return
	}
	if req.Service == "" {
		req.Service = serviceFromPath(req.Path)
	}
	if req.Content == "" {
		req.Content, _ = a.readTextFile(req.Path)
	}
	user := "system"
	if u := currentUser(r); u != nil {
		user = u.Username
	}
	now := nowString()
	res, err := a.DB.Exec(`insert into config_histories(service,file_path,content,comment,is_stable,created_by,created_at,updated_at) values(?,?,?,?,?,?,?,?)`,
		req.Service, req.Path, req.Content, req.Comment, req.IsStable, user, now, now)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	id, _ := res.LastInsertId()
	writeJSON(w, http.StatusCreated, map[string]any{"success": true, "id": id})
}

func (a *App) handleHistory(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(`select id,service,file_path,comment,is_stable,created_by,created_at from config_histories where deleted_at is null order by id desc limit 200`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	defer rows.Close()
	var items []map[string]any
	for rows.Next() {
		var id int64
		var service, path, comment, by, created string
		var stable bool
		_ = rows.Scan(&id, &service, &path, &comment, &stable, &by, &created)
		items = append(items, map[string]any{"id": id, "service": service, "file_path": path, "comment": comment, "is_stable": stable, "created_by": by, "created_at": created})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items})
}

func (a *App) handleHistoryGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var item = map[string]any{}
	var content, service, path, comment, by, created string
	var stable bool
	err := a.DB.QueryRow(`select service,file_path,content,comment,is_stable,created_by,created_at from config_histories where id=?`, id).Scan(&service, &path, &content, &comment, &stable, &by, &created)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "history not found")
		return
	}
	item["id"] = id
	item["service"] = service
	item["file_path"] = path
	item["content"] = content
	item["comment"] = comment
	item["is_stable"] = stable
	item["created_by"] = by
	item["created_at"] = created
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": item})
}

func (a *App) handleHistoryRollback(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var path, content string
	err := a.DB.QueryRow(`select file_path,content from config_histories where id=?`, id).Scan(&path, &content)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "history not found")
		return
	}
	if err := a.writeTextFile(path, content); err != nil {
		writeError(w, http.StatusBadRequest, "rollback_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleHistoryStar(w http.ResponseWriter, r *http.Request) {
	_, err := a.DB.Exec(`update config_histories set is_stable = not is_stable where id=?`, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "star_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleHistoryDelete(w http.ResponseWriter, r *http.Request) {
	_, err := a.DB.Exec(`update config_histories set deleted_at=? where id=?`, time.Now(), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "delete_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleHistoryCompare(w http.ResponseWriter, r *http.Request) {
	leftID := firstNonEmpty(r.URL.Query().Get("from"), r.URL.Query().Get("left"), r.URL.Query().Get("id1"))
	rightID := firstNonEmpty(r.URL.Query().Get("to"), r.URL.Query().Get("right"), r.URL.Query().Get("id2"))
	leftContent, leftLabel := a.historyContentForCompare(leftID, r.URL.Query().Get("path"))
	rightContent, rightLabel := a.historyContentForCompare(rightID, r.URL.Query().Get("path"))
	diff := simpleUnifiedDiff(leftLabel, rightLabel, leftContent, rightContent)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "diff": diff, "data": map[string]any{"diff": diff, "left": leftLabel, "right": rightLabel}})
}

func (a *App) handleConfigBackup(w http.ResponseWriter, r *http.Request) {
	root := filepath.Join(a.DataDir, "configs")
	b, err := zipDir(root)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "backup_failed", err.Error())
		return
	}
	name := "configs-" + time.Now().Format("20060102-150405") + ".zip"
	rel := filepath.ToSlash(filepath.Join("backups", name))
	path, err := a.safePath(rel)
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "mkdir_failed", err.Error())
		return
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "backup_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"name": name, "path": rel, "size": len(b), "created_at": nowString()}})
}

func (a *App) handleConfigBackups(w http.ResponseWriter, r *http.Request) {
	dir, err := a.safePath("backups")
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	entries, _ := os.ReadDir(dir)
	var rows []map[string]any
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".zip") {
			continue
		}
		info, _ := entry.Info()
		rows = append(rows, map[string]any{"name": entry.Name(), "path": filepath.ToSlash(filepath.Join("backups", entry.Name())), "size": info.Size(), "created_at": info.ModTime().Format(time.RFC3339)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": rows, "backups": rows})
}

func (a *App) handleConfigBackupDownload(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(firstNonEmpty(r.URL.Query().Get("name"), r.URL.Query().Get("path")))
	if name == "." || name == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "backup name required")
		return
	}
	path, err := a.safePath(filepath.ToSlash(filepath.Join("backups", name)))
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+name)
	serveLocalFile(w, r, path)
}

func (a *App) handleConfigRestore(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	name := filepath.Base(firstNonEmpty(req.Name, req.Path))
	if name == "." || name == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "backup name required")
		return
	}
	src, err := a.safePath(filepath.ToSlash(filepath.Join("backups", name)))
	if err != nil {
		writeError(w, http.StatusBadRequest, "path_error", err.Error())
		return
	}
	if err := restoreZipToDir(src, filepath.Join(a.DataDir, "configs")); err != nil {
		writeError(w, http.StatusBadRequest, "restore_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"restored": name}})
}

func zipDir(root string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = w.Write(b)
		return err
	})
	if err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func restoreZipToDir(src, dest string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zr.Close()
	cleanDest, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	for _, file := range zr.File {
		name := filepath.Clean(filepath.ToSlash(file.Name))
		if name == "." || strings.HasPrefix(name, "../") || filepath.IsAbs(name) {
			continue
		}
		target := filepath.Join(cleanDest, filepath.FromSlash(name))
		absTarget, err := filepath.Abs(target)
		if err != nil || (!strings.HasPrefix(absTarget, cleanDest+string(os.PathSeparator)) && absTarget != cleanDest) {
			continue
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(absTarget, file.FileInfo().Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(absTarget), 0755); err != nil {
			return err
		}
		rc, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(absTarget, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, file.FileInfo().Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		rc.Close()
		out.Close()
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

func (a *App) historyContentForCompare(id, path string) (string, string) {
	if strings.TrimSpace(id) != "" {
		var content, filePath string
		if err := a.DB.QueryRow(`select content,file_path from config_histories where id=?`, id).Scan(&content, &filePath); err == nil {
			return content, "history:" + id + ":" + filePath
		}
	}
	path = normalizeConfigRel(path)
	if path == "" {
		return "", "empty"
	}
	content, _ := a.readTextFile(path)
	return content, "current:" + path
}

func simpleUnifiedDiff(leftLabel, rightLabel, left, right string) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")
	var out strings.Builder
	out.WriteString("--- " + leftLabel + "\n")
	out.WriteString("+++ " + rightLabel + "\n")
	headerLen := out.Len()
	max := len(leftLines)
	if len(rightLines) > max {
		max = len(rightLines)
	}
	for i := 0; i < max; i++ {
		var l, r string
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		if l == r {
			continue
		}
		out.WriteString(fmt.Sprintf("@@ line %d @@\n", i+1))
		if i < len(leftLines) {
			out.WriteString("-" + l + "\n")
		}
		if i < len(rightLines) {
			out.WriteString("+" + r + "\n")
		}
	}
	if out.Len() == headerLen {
		out.WriteString(" no changes\n")
	}
	return out.String()
}

func normalizeConfigRel(rel string) string {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" {
		return rel
	}
	if strings.HasPrefix(rel, "configs/") || strings.HasPrefix(rel, "logs/") || strings.HasPrefix(rel, "backups/") {
		return rel
	}
	switch {
	case rel == "mihomo" || rel == "mosdns" || rel == "network" || rel == "singbox" || rel == "sing-box":
		return "configs/" + strings.ReplaceAll(rel, "sing-box", "singbox")
	case strings.HasPrefix(rel, "mihomo/") || strings.HasPrefix(rel, "mosdns/") || strings.HasPrefix(rel, "network/") || strings.HasPrefix(rel, "singbox/") || strings.HasPrefix(rel, "sing-box/"):
		rel = strings.Replace(rel, "sing-box/", "singbox/", 1)
		return "configs/" + rel
	default:
		return rel
	}
}
