package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (a *App) handleMosDNSAdguardRules(w http.ResponseWriter, r *http.Request) {
	items := a.ruleSourcesForCompat("adguard", "")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items, "items": items, "rules": items, "total": len(items)})
}

func (a *App) handleMosDNSAdguardRuleCreate(w http.ResponseWriter, r *http.Request) {
	var req mosDNSRuleSource
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	req.SourceType = "adguard"
	req.Type = "adguard"
	source, status, err := a.createCompatRuleSource(req)
	if err != nil {
		writeError(w, status, "write_failed", err.Error())
		return
	}
	writeJSON(w, status, map[string]any{"success": true, "data": mosDNSRuleSourceCompatMap(source, false)})
}

func (a *App) handleMosDNSAdguardRulePut(w http.ResponseWriter, r *http.Request) {
	a.handleCompatRuleSourcePut(w, r, "adguard", false)
}

func (a *App) handleMosDNSAdguardRuleDelete(w http.ResponseWriter, r *http.Request) {
	a.handleCompatRuleSourceDelete(w, r, "adguard")
}

func (a *App) handleMosDNSAdguardUpdate(w http.ResponseWriter, r *http.Request) {
	a.updateCompatRuleSources(w, "adguard")
}

func (a *App) handleMosDNSGeositeRules(w http.ResponseWriter, r *http.Request) {
	typ := normalizeMosDNSGeositeType(r.URL.Query().Get("type"))
	items := a.ruleSourcesForCompat("srs", typ)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items, "items": items, "rules": items, "total": len(items)})
}

func (a *App) handleMosDNSGeositeRulePut(w http.ResponseWriter, r *http.Request) {
	typ := normalizeMosDNSGeositeType(r.PathValue("type"))
	name := strings.TrimSpace(r.PathValue("name"))
	var req mosDNSRuleSource
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Name == "" {
		req.Name = name
	}
	if req.Type == "" {
		req.Type = typ
	}
	req.Type = normalizeMosDNSGeositeType(req.Type)
	req.SourceType = "srs"
	if current, ok := a.findMosDNSGeositeSource(typ, name); ok {
		req.ID = current.ID
		req.ConfigPath = current.ConfigPath
		if req.Files == "" {
			req.Files = current.Files
		}
		if req.LastUpdated == "" {
			req.LastUpdated = current.LastUpdated
		}
		if req.RuleCount == 0 {
			req.RuleCount = current.RuleCount
		}
	}
	source, status, err := a.createCompatRuleSource(req)
	if err != nil {
		writeError(w, status, "write_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": mosDNSRuleSourceCompatMap(source, true)})
}

func (a *App) handleMosDNSGeositeRuleDelete(w http.ResponseWriter, r *http.Request) {
	typ := normalizeMosDNSGeositeType(r.PathValue("type"))
	name := strings.TrimSpace(r.PathValue("name"))
	source, ok := a.findMosDNSGeositeSource(typ, name)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "rule source not found")
		return
	}
	if err := a.deleteCompatRuleSource(source); err != nil {
		writeError(w, http.StatusBadRequest, "delete_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": mosDNSRuleSourceCompatMap(source, true)})
}

func (a *App) handleMosDNSGeositeRuleUpdate(w http.ResponseWriter, r *http.Request) {
	typ := normalizeMosDNSGeositeType(r.PathValue("type"))
	name := strings.TrimSpace(r.PathValue("name"))
	source, ok := a.findMosDNSGeositeSource(typ, name)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "rule source not found")
		return
	}
	updated, err := a.updateMosDNSRuleSource(source)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"success": false, "error": err.Error(), "data": mosDNSRuleSourceCompatMap(source, true)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": mosDNSRuleSourceCompatMap(updated, true)})
}

func (a *App) ruleSourcesForCompat(sourceType, typ string) []map[string]any {
	var out []map[string]any
	for _, source := range a.mosDNSRuleSources() {
		if source.SourceType != sourceType {
			continue
		}
		if sourceType == "srs" && typ != "" && normalizeMosDNSGeositeType(source.Type) != typ {
			continue
		}
		out = append(out, mosDNSRuleSourceCompatMap(source, sourceType == "srs"))
	}
	sort.SliceStable(out, func(i, j int) bool {
		left := compatRuleOrder(fmtAny(out[i]["type"]), fmtAny(out[i]["name"]))
		right := compatRuleOrder(fmtAny(out[j]["type"]), fmtAny(out[j]["name"]))
		if left == right {
			return fmtAny(out[i]["name"]) < fmtAny(out[j]["name"])
		}
		return left < right
	})
	return out
}

func mosDNSRuleSourceCompatMap(source mosDNSRuleSource, geosite bool) map[string]any {
	typ := source.Type
	if geosite {
		typ = publicMosDNSGeositeType(typ)
	}
	return map[string]any{
		"id":                    source.ID,
		"name":                  source.Name,
		"type":                  typ,
		"files":                 source.Files,
		"url":                   source.URL,
		"enabled":               source.Enabled,
		"auto_update":           source.AutoUpdate,
		"update_interval_hours": source.UpdateIntervalHours,
		"rule_count":            source.RuleCount,
		"last_updated":          source.LastUpdated,
		"source_type":           source.SourceType,
		"config_path":           source.ConfigPath,
		"local_path":            source.LocalPath,
		"file_size":             source.FileSize,
		"warning":               source.Warning,
	}
}

func (a *App) createCompatRuleSource(req mosDNSRuleSource) (mosDNSRuleSource, int, error) {
	req.normalizeForCreate(req.SourceType)
	if req.Name == "" || req.URL == "" {
		return req, http.StatusBadRequest, fmt.Errorf("name and url are required")
	}
	if req.SourceType == "srs" {
		req.Type = normalizeMosDNSGeositeType(req.Type)
	}
	if req.ID == "" {
		if req.SourceType == "adguard" {
			req.ID = randomHex(16)
		} else {
			req.ID = mosDNSRuleSourceID(req)
		}
	}
	if req.LastUpdated == "" {
		req.LastUpdated = time.Now().Format(time.RFC3339)
	}
	if req.UpdateIntervalHours <= 0 {
		req.UpdateIntervalHours = 24
	}
	if req.Files == "" {
		req.Files = mosDNSRuleSourceLocalRel(req)
		req.Files = strings.TrimPrefix(strings.TrimPrefix(req.Files, "configs/mosdns/"), "mosdns/")
	}
	configRel := mosDNSRuleSourceConfigRel(req)
	items, err := a.readMosDNSRuleSourceConfig(configRel, req.SourceType)
	if err != nil {
		return req, http.StatusBadRequest, err
	}
	replaced := false
	for i := range items {
		if items[i].ID == req.ID || items[i].Name == req.Name {
			items[i] = req
			replaced = true
			break
		}
	}
	if !replaced {
		items = append(items, req)
	}
	if err := a.writeMosDNSRuleSourceConfig(configRel, items); err != nil {
		return req, http.StatusBadRequest, err
	}
	source, _ := a.findMosDNSRuleSource(req.ID)
	if source.ID == "" {
		source = req
		source.ConfigPath = configRel
		source.hydrateRuntimeFields()
	}
	status := http.StatusOK
	if !replaced {
		status = http.StatusCreated
	}
	return source, status, nil
}

func (a *App) handleCompatRuleSourcePut(w http.ResponseWriter, r *http.Request, sourceType string, geosite bool) {
	id := r.PathValue("id")
	current, ok := a.findMosDNSRuleSource(id)
	if !ok || current.SourceType != sourceType {
		writeError(w, http.StatusNotFound, "not_found", "rule source not found")
		return
	}
	var req mosDNSRuleSource
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Name != "" {
		current.Name = req.Name
	}
	if req.Type != "" {
		current.Type = req.Type
	}
	if sourceType == "srs" {
		current.Type = normalizeMosDNSGeositeType(current.Type)
	}
	if req.Files != "" {
		current.Files = req.Files
	}
	if req.URL != "" {
		current.URL = req.URL
	}
	current.Enabled = req.Enabled
	current.AutoUpdate = req.AutoUpdate
	if req.UpdateIntervalHours > 0 {
		current.UpdateIntervalHours = req.UpdateIntervalHours
	}
	if req.RuleCount > 0 {
		current.RuleCount = req.RuleCount
	}
	if req.LastUpdated != "" {
		current.LastUpdated = req.LastUpdated
	}
	if err := a.replaceMosDNSRuleSource(current); err != nil {
		writeError(w, http.StatusBadRequest, "write_failed", err.Error())
		return
	}
	source, _ := a.findMosDNSRuleSource(current.ID)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": mosDNSRuleSourceCompatMap(source, geosite)})
}

func (a *App) handleCompatRuleSourceDelete(w http.ResponseWriter, r *http.Request, sourceType string) {
	source, ok := a.findMosDNSRuleSource(r.PathValue("id"))
	if !ok || source.SourceType != sourceType {
		writeError(w, http.StatusNotFound, "not_found", "rule source not found")
		return
	}
	if err := a.deleteCompatRuleSource(source); err != nil {
		writeError(w, http.StatusBadRequest, "delete_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": mosDNSRuleSourceCompatMap(source, false)})
}

func (a *App) deleteCompatRuleSource(source mosDNSRuleSource) error {
	items, err := a.readMosDNSRuleSourceConfig(source.ConfigPath, source.SourceType)
	if err != nil {
		return err
	}
	next := make([]mosDNSRuleSource, 0, len(items))
	for _, item := range items {
		if item.ID != source.ID && item.Name != source.Name {
			next = append(next, item)
		}
	}
	return a.writeMosDNSRuleSourceConfig(source.ConfigPath, next)
}

func (a *App) updateCompatRuleSources(w http.ResponseWriter, sourceType string) {
	var updated []map[string]any
	var failures []map[string]string
	for _, source := range a.mosDNSRuleSources() {
		if source.SourceType != sourceType || !source.Enabled {
			continue
		}
		next, err := a.updateMosDNSRuleSource(source)
		if err != nil {
			failures = append(failures, map[string]string{"id": source.ID, "name": source.Name, "error": err.Error()})
			continue
		}
		updated = append(updated, mosDNSRuleSourceCompatMap(next, sourceType == "srs"))
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": len(failures) == 0, "data": map[string]any{"updated": updated, "failures": failures}, "updated": updated, "failures": failures})
}

func (a *App) findMosDNSGeositeSource(typ, name string) (mosDNSRuleSource, bool) {
	typ = normalizeMosDNSGeositeType(typ)
	name = strings.TrimSpace(name)
	for _, source := range a.mosDNSRuleSources() {
		if source.SourceType != "srs" {
			continue
		}
		if typ != "" && normalizeMosDNSGeositeType(source.Type) != typ {
			continue
		}
		if source.Name == name || source.ID == name || safeRuleFileName(source.Name) == safeRuleFileName(name) {
			return source, true
		}
	}
	return mosDNSRuleSource{}, false
}

func normalizeMosDNSGeositeType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	switch value {
	case "geosite_cn", "geositecn", "cn":
		return "geositecn"
	case "geosite_no_cn", "geosite_no_cn_", "geositenocn", "geolocation_!cn", "geolocation-!cn", "no_cn":
		return "geositenocn"
	case "geoip_cn", "geoipcn":
		return "geoipcn"
	case "cuscn":
		return "cuscn"
	case "cusnocn", "custom_no_cn":
		return "cusnocn"
	default:
		return value
	}
}

func publicMosDNSGeositeType(value string) string {
	switch normalizeMosDNSGeositeType(value) {
	case "geositecn":
		return "geosite_cn"
	case "geositenocn":
		return "geosite_no_cn"
	case "geoipcn":
		return "geoip_cn"
	default:
		return value
	}
}

func compatRuleOrder(typ, name string) int {
	switch normalizeMosDNSGeositeType(typ) {
	case "geositecn":
		return 10
	case "geositenocn":
		return 20
	case "geoipcn":
		return 30
	case "cuscn":
		return 40
	case "cusnocn":
		if strings.EqualFold(name, "tiktok") {
			return 60
		}
		return 50
	case "adguard":
		switch name {
		case "httpdns":
			return 10
		case "pcdn1":
			return 20
		case "pcdn2":
			return 30
		}
	}
	return 100
}

func (a *App) jsonSettingWithFileFallback(key, rel string, fallback any) any {
	if value := a.jsonSetting(key, nil); value != nil {
		return value
	}
	return a.readJSONFile(rel, fallback)
}

func (a *App) readJSONFile(rel string, fallback any) any {
	content, err := a.readTextFile(rel)
	if err != nil || strings.TrimSpace(content) == "" {
		return fallback
	}
	var out any
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		return fallback
	}
	return out
}

func (a *App) writeJSONFile(rel string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	path, err := a.safePath(rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0644)
}
