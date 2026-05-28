package server

import (
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func (a *App) handleUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(`select id,username,coalesce(email,''),coalesce(display_name,''),role,is_active,last_login,created_at,updated_at from users where deleted_at is null order by id`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		var last nullableTime
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.DisplayName, &u.Role, &u.IsActive, &last, &u.CreatedAt, &u.UpdatedAt); err == nil {
			if last.Valid {
				u.LastLogin = &last.Time
			}
			users = append(users, u)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": users, "users": users})
}

func (a *App) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(r) {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	var req struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if req.Role == "" {
		req.Role = "operator"
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(normalizePasswordForStorage(req.Password)), bcrypt.DefaultCost)
	now := time.Now()
	res, err := a.DB.Exec(`insert into users(username,password,email,display_name,role,is_active,created_at,updated_at) values(?,?,?,?,?,true,?,?)`, req.Username, string(hash), req.Email, req.DisplayName, req.Role, now, now)
	if err != nil {
		writeError(w, http.StatusBadRequest, "create_failed", err.Error())
		return
	}
	id, _ := res.LastInsertId()
	u, _ := a.userByID(id)
	writeJSON(w, http.StatusCreated, map[string]any{"success": true, "user": u})
}

func (a *App) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	u, err := a.userByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "user": u})
}

func (a *App) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(r) {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var req struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
		IsActive    bool   `json:"is_active"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	_, err := a.DB.Exec(`update users set email=?,display_name=?,role=?,is_active=?,updated_at=? where id=?`, req.Email, req.DisplayName, req.Role, req.IsActive, time.Now(), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "update_failed", err.Error())
		return
	}
	u, _ := a.userByID(id)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "user": u})
}

func (a *App) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(r) {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	_, err := a.DB.Exec(`update users set deleted_at=?,updated_at=? where id=?`, time.Now(), time.Now(), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "delete_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleResetUserPassword(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(r) {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var req struct {
		Password    string `json:"password"`
		NewPassword string `json:"new_password"`
	}
	_ = decodeJSON(r, &req)
	if req.Password == "" {
		req.Password = req.NewPassword
	}
	if req.Password == "" {
		req.Password = "admin123456"
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(normalizePasswordForStorage(req.Password)), bcrypt.DefaultCost)
	_, err := a.DB.Exec(`update users set password=?,updated_at=? where id=?`, string(hash), time.Now(), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "reset_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "password": req.Password})
}

func (a *App) handleToggleUser(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(r) {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	_, err := a.DB.Exec(`update users set is_active = not is_active, updated_at=? where id=?`, time.Now(), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "toggle_failed", err.Error())
		return
	}
	u, _ := a.userByID(id)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "user": u})
}

func (a *App) handleUserStats(w http.ResponseWriter, r *http.Request) {
	var total, active int
	_ = a.DB.QueryRow(`select count(*) from users where deleted_at is null`).Scan(&total)
	_ = a.DB.QueryRow(`select count(*) from users where deleted_at is null and is_active=true`).Scan(&active)
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "total": total, "active": active})
}

type nullableTime struct {
	Time  time.Time
	Valid bool
}

func (nt *nullableTime) Scan(value any) error {
	if value == nil {
		nt.Valid = false
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		nt.Time, nt.Valid = v, true
	case string:
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			nt.Time, nt.Valid = t, true
		}
	}
	return nil
}
