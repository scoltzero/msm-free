package server

import (
	"database/sql"
	"net/http"
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
	}
	_, err := a.DB.Exec(`insert into api_tokens(user_id,name,token_hash,expires_at,created_at,revoked) values(?,?,?,?,?,false)`, u.ID, req.Name, tokenHash(token), expires, time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "token": token})
}

func (a *App) handleRevokeAPIToken(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	_, err := a.DB.Exec(`update api_tokens set revoked=true where id=? and user_id=?`, r.PathValue("id"), u.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "revoke_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
