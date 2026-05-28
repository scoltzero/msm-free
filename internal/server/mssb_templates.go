package server

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed mssb_templates/**
var mssbTemplates embed.FS

func mssbTemplateText(rel string) (string, bool) {
	b, err := mssbTemplates.ReadFile(filepath.ToSlash(filepath.Join("mssb_templates", rel)))
	if err != nil {
		return "", false
	}
	return string(b), true
}

func (a *App) ensureMSSBTemplateDefaults(overwriteStructural bool) error {
	root := "mssb_templates"
	return fs.WalkDir(mssbTemplates, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(path, root+"/")
		if rel == "mosdns/config.yaml" || strings.HasPrefix(rel, "mihomo/") {
			return nil
		}
		destRel := filepath.ToSlash(filepath.Join("configs", rel))
		dest := filepath.Join(a.DataDir, destRel)
		if !overwriteStructural || !isMSSBStructuralTemplate(rel) {
			if _, err := os.Stat(dest); err == nil {
				return nil
			}
		}
		b, err := mssbTemplates.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		return os.WriteFile(dest, b, 0644)
	})
}

func isMSSBStructuralTemplate(rel string) bool {
	switch {
	case rel == "mosdns/config.yaml":
		return true
	case strings.HasPrefix(rel, "mosdns/sub_config/"):
		return true
	case strings.HasPrefix(rel, "mosdns/adguard/"):
		return true
	case strings.HasPrefix(rel, "mosdns/webinfo/"):
		return true
	case strings.HasPrefix(rel, "mosdns/nft/"):
		return true
	case strings.HasPrefix(rel, "mosdns/srs/") && strings.HasSuffix(rel, ".json"):
		return true
	case strings.HasSuffix(rel, "_settings.json"), strings.HasSuffix(rel, "_overrides.json"):
		return true
	case rel == "mihomo/config.yaml", rel == "mihomo/phone_config.yaml":
		return true
	default:
		return false
	}
}
