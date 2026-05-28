package server

import "net/http"

func (a *App) registerSingBoxRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/singbox/status", a.handleSingBoxUnavailable)
	mux.HandleFunc("POST /api/v1/singbox/start", a.handleSingBoxUnavailable)
	mux.HandleFunc("POST /api/v1/singbox/stop", a.handleSingBoxUnavailable)
	mux.HandleFunc("POST /api/v1/singbox/restart", a.handleSingBoxUnavailable)
	mux.HandleFunc("POST /api/v1/singbox/install", a.handleSingBoxUnavailable)
	mux.HandleFunc("GET /api/v1/singbox/versions", a.handleSingBoxVersions)
	mux.HandleFunc("POST /api/v1/singbox/version", a.handleSingBoxUnavailable)
	mux.HandleFunc("GET /api/v1/singbox/configs", a.handleSingBoxConfigs)
	mux.HandleFunc("GET /api/v1/singbox/config", a.handleSingBoxConfig)
	mux.HandleFunc("POST /api/v1/singbox/config", a.handleSingBoxConfigSwitch)
	mux.HandleFunc("PUT /api/v1/singbox/config/{path...}", a.handleSingBoxUnavailable)
	mux.HandleFunc("POST /api/v1/singbox/config/{id}/rollback", a.handleSingBoxConfigRollback)
	mux.HandleFunc("POST /api/v1/singbox/validate", a.handleSingBoxValidate)

	mux.HandleFunc("GET /api/v1/sing-box/status", a.handleSingBoxUnavailable)
	mux.HandleFunc("POST /api/v1/sing-box/start", a.handleSingBoxUnavailable)
	mux.HandleFunc("POST /api/v1/sing-box/stop", a.handleSingBoxUnavailable)
	mux.HandleFunc("POST /api/v1/sing-box/restart", a.handleSingBoxUnavailable)
	mux.HandleFunc("POST /api/v1/sing-box/install", a.handleSingBoxUnavailable)
	mux.HandleFunc("GET /api/v1/sing-box/versions", a.handleSingBoxVersions)
	mux.HandleFunc("POST /api/v1/sing-box/version", a.handleSingBoxUnavailable)
	mux.HandleFunc("GET /api/v1/sing-box/configs", a.handleSingBoxConfigs)
	mux.HandleFunc("GET /api/v1/sing-box/config", a.handleSingBoxConfig)
	mux.HandleFunc("POST /api/v1/sing-box/config", a.handleSingBoxConfigSwitch)
	mux.HandleFunc("PUT /api/v1/sing-box/config/{path...}", a.handleSingBoxUnavailable)
	mux.HandleFunc("POST /api/v1/sing-box/config/{id}/rollback", a.handleSingBoxConfigRollback)
	mux.HandleFunc("POST /api/v1/sing-box/validate", a.handleSingBoxValidate)
}

func (a *App) handleSingBoxUnavailable(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"name":         "singbox",
			"display_name": "Sing-Box",
			"installed":    false,
			"running":      false,
			"status":       "disabled",
			"message":      "sing-box is not implemented in msm-free x86 first version",
		},
	})
}

func (a *App) handleSingBoxVersions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": []any{}})
}

func (a *App) handleSingBoxConfigs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": []any{}})
}

func (a *App) handleSingBoxConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "content": "", "data": map[string]any{"content": ""}})
}

func (a *App) handleSingBoxConfigSwitch(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"config": "disabled", "status": "disabled"}})
}

func (a *App) handleSingBoxConfigRollback(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": map[string]any{"id": r.PathValue("id"), "status": "disabled"}})
}

func (a *App) handleSingBoxValidate(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "valid": true})
}
