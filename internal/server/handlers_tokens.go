package server

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
)

func (a *App) handleAPITokens(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	rows, err := a.DB.Query(`select id,name,last_used_at,expires_at,created_at,revoked from api_tokens where user_id=? order by id desc`, u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	defer rows.Close()
	var items []map[string]any
	for rows.Next() {
		var id int64
		var name string
		var last, expires, created sql.NullTime
		var revoked bool
		_ = rows.Scan(&id, &name, &last, &expires, &created, &revoked)
		items = append(items, map[string]any{
			"id": id, "name": name, "last_used_at": nullableTimeString(last), "expires_at": nullableTimeString(expires),
			"created_at": nullableTimeString(created), "revoked": revoked,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items})
}

func (a *App) handleCreateAPIToken(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		Name      string `json:"name"`
		ExpiresIn int    `json:"expires_in"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Name == "" {
		req.Name = "API Token"
	}
	token := "msmf_" + randomHex(32)
	var expires any
	if req.ExpiresIn > 0 {
		expires = time.Now().Add(time.Duration(req.ExpiresIn) * time.Second)
	} else if strings.TrimSpace(req.ExpiresAt) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(req.ExpiresAt)); err == nil {
			expires = t
		} else {
			writeError(w, http.StatusBadRequest, "bad_request", "expires_at must be RFC3339")
			return
		}
	}
	res, err := a.DB.Exec(`insert into api_tokens(user_id,name,token_hash,expires_at,created_at,revoked) values(?,?,?,?,?,false)`, u.ID, req.Name, tokenHash(token), expires, time.Now())
	if err != nil {
		a.audit(u, "api_token.create", "api_tokens", req.Name, false, err.Error())
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	id, _ := res.LastInsertId()
	a.audit(u, "api_token.create", "api_tokens", req.Name, true, "")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "token": token, "data": map[string]any{"id": id, "name": req.Name, "token": token, "expires_at": expires}})
}

func (a *App) handleRevokeAPIToken(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	res, err := a.DB.Exec(`update api_tokens set revoked=true where id=? and user_id=?`, r.PathValue("id"), u.ID)
	if err != nil {
		a.audit(u, "api_token.revoke", "api_tokens", r.PathValue("id"), false, err.Error())
		writeError(w, http.StatusBadRequest, "revoke_failed", err.Error())
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeError(w, http.StatusNotFound, "not_found", "token not found")
		return
	}
	a.audit(u, "api_token.revoke", "api_tokens", r.PathValue("id"), true, "")
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
