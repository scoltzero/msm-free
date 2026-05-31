package server

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (a *App) handleAPITokens(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(r) {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	rows, err := a.DB.Query(`select t.id,t.user_id,coalesce(u.username,''),t.name,coalesce(t.scope,'admin'),t.last_used_at,t.expires_at,t.created_at,t.revoked from api_tokens t left join users u on u.id=t.user_id order by t.id desc`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	defer rows.Close()
	var items []map[string]any
	for rows.Next() {
		var id, userID int64
		var username, name, scope string
		var last, expires, created sql.NullTime
		var revoked bool
		_ = rows.Scan(&id, &userID, &username, &name, &scope, &last, &expires, &created, &revoked)
		items = append(items, map[string]any{
			"id": id, "user_id": userID, "username": username, "name": name, "scope": normalizeTokenScope(scope), "last_used_at": nullableTimeString(last), "expires_at": nullableTimeString(expires),
			"created_at": nullableTimeString(created), "revoked": revoked,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items})
}

func (a *App) handleCreateAPIToken(w http.ResponseWriter, r *http.Request) {
	admin := currentUser(r)
	if !a.requireAdmin(r) {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	var req struct {
		Name      string `json:"name"`
		UserID    int64  `json:"user_id"`
		Username  string `json:"username"`
		Scope     string `json:"scope"`
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
	if !validTokenScope(req.Scope) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid token scope")
		return
	}
	req.Scope = normalizeTokenScope(req.Scope)
	target, err := a.apiTokenTargetUser(req.UserID, req.Username, admin)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
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
	res, err := a.DB.Exec(`insert into api_tokens(user_id,name,token_hash,scope,expires_at,created_at,revoked) values(?,?,?,?,?,?,false)`, target.ID, req.Name, tokenHash(token), req.Scope, expires, time.Now())
	if err != nil {
		a.audit(admin, "api_token.create", "api_tokens", req.Name, false, err.Error())
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	id, _ := res.LastInsertId()
	a.audit(admin, "api_token.create", "api_tokens", req.Name, true, "")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "token": token, "data": map[string]any{"id": id, "user_id": target.ID, "username": target.Username, "name": req.Name, "scope": req.Scope, "expires_at": expires}})
}

func (a *App) handleRevokeAPIToken(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if !a.requireAdmin(r) {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid token id")
		return
	}
	res, err := a.DB.Exec(`update api_tokens set revoked=true where id=?`, id)
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

func (a *App) apiTokenTargetUser(userID int64, username string, fallback *User) (*User, error) {
	if userID > 0 {
		return a.userByID(userID)
	}
	username = strings.TrimSpace(username)
	if username != "" {
		u, _, err := a.findUserByUsername(username)
		return u, err
	}
	if fallback == nil {
		return nil, sql.ErrNoRows
	}
	return fallback, nil
}
