package server

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func (a *App) handleUsers(w http.ResponseWriter, r *http.Request) {
	conds := []string{"deleted_at is null"}
	args := []any{}
	if search := strings.TrimSpace(r.URL.Query().Get("search")); search != "" {
		conds = append(conds, "(username like ? or email like ? or display_name like ?)")
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	if role := strings.TrimSpace(r.URL.Query().Get("role")); role != "" && role != "all" {
		conds = append(conds, "role=?")
		args = append(args, role)
	}
	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" && status != "all" {
		switch strings.ToLower(status) {
		case "active", "enabled":
			conds = append(conds, "is_active=true")
		case "inactive", "disabled":
			conds = append(conds, "is_active=false")
		}
	}
	where := strings.Join(conds, " and ")
	var total int
	_ = a.DB.QueryRow(`select count(*) from users where `+where, args...).Scan(&total)
	page, pageSize := pageParams(r, 20)
	query := `select id,username,coalesce(email,''),coalesce(display_name,''),role,is_active,last_login,created_at,updated_at from users where ` + where + ` order by id limit ? offset ?`
	queryArgs := append(args, pageSize, (page-1)*pageSize)
	rows, err := a.DB.Query(query, queryArgs...)
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
	stats := a.userStats()
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": users, "users": users, "stats": stats, "pagination": pagination(page, pageSize, total), "meta": map[string]any{"pagination": pagination(page, pageSize, total)}})
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
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || len(req.Username) < 3 {
		writeError(w, http.StatusBadRequest, "bad_request", "username must be at least 3 characters")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "bad_request", "password must be at least 8 characters")
		return
	}
	if req.Role == "" {
		req.Role = "operator"
	}
	if !validUserRole(req.Role) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid role")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(normalizePasswordForStorage(req.Password)), bcrypt.DefaultCost)
	now := time.Now()
	res, err := a.DB.Exec(`insert into users(username,password,email,display_name,role,is_active,created_at,updated_at) values(?,?,?,?,?,true,?,?)`, req.Username, string(hash), req.Email, req.DisplayName, req.Role, now, now)
	if err != nil {
		a.audit(currentUser(r), "user.create", "users", req.Username, false, err.Error())
		writeError(w, http.StatusBadRequest, "create_failed", err.Error())
		return
	}
	id, _ := res.LastInsertId()
	u, _ := a.userByID(id)
	a.audit(currentUser(r), "user.create", "users", req.Username, true, "")
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
		IsActive    *bool  `json:"is_active"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	old, err := a.userByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	if req.Role == "" {
		req.Role = old.Role
	}
	if !validUserRole(req.Role) {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid role")
		return
	}
	active := old.IsActive
	if req.IsActive != nil {
		active = *req.IsActive
	}
	if old.Role == "admin" && (!active || req.Role != "admin") && a.activeAdminCount() <= 1 {
		writeError(w, http.StatusBadRequest, "last_admin", "cannot disable or demote the last active admin")
		return
	}
	_, err = a.DB.Exec(`update users set email=?,display_name=?,role=?,is_active=?,updated_at=? where id=?`, req.Email, req.DisplayName, req.Role, active, time.Now(), id)
	if err != nil {
		a.audit(currentUser(r), "user.update", "users", strconv.FormatInt(id, 10), false, err.Error())
		writeError(w, http.StatusBadRequest, "update_failed", err.Error())
		return
	}
	u, _ := a.userByID(id)
	a.audit(currentUser(r), "user.update", "users", strconv.FormatInt(id, 10), true, "")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "user": u})
}

func (a *App) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(r) {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if u := currentUser(r); u != nil && u.ID == id {
		writeError(w, http.StatusBadRequest, "self_delete", "cannot delete current user")
		return
	}
	target, err := a.userByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	if target.Role == "admin" && a.adminCount() <= 1 {
		writeError(w, http.StatusBadRequest, "last_admin", "cannot delete the last admin")
		return
	}
	_, err = a.DB.Exec(`update users set deleted_at=?,updated_at=? where id=?`, time.Now(), time.Now(), id)
	if err != nil {
		a.audit(currentUser(r), "user.delete", "users", strconv.FormatInt(id, 10), false, err.Error())
		writeError(w, http.StatusBadRequest, "delete_failed", err.Error())
		return
	}
	a.audit(currentUser(r), "user.delete", "users", strconv.FormatInt(id, 10), true, "")
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
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "bad_request", "password must be at least 8 characters")
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(normalizePasswordForStorage(req.Password)), bcrypt.DefaultCost)
	_, err := a.DB.Exec(`update users set password=?,updated_at=? where id=?`, string(hash), time.Now(), id)
	if err != nil {
		a.audit(currentUser(r), "user.reset_password", "users", strconv.FormatInt(id, 10), false, err.Error())
		writeError(w, http.StatusBadRequest, "reset_failed", err.Error())
		return
	}
	a.audit(currentUser(r), "user.reset_password", "users", strconv.FormatInt(id, 10), true, "")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "password": req.Password, "plain_password": req.Password})
}

func (a *App) handleToggleUser(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(r) {
		writeError(w, http.StatusForbidden, "forbidden", "admin required")
		return
	}
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	target, err := a.userByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	if target.Role == "admin" && target.IsActive && a.activeAdminCount() <= 1 {
		writeError(w, http.StatusBadRequest, "last_admin", "cannot disable the last active admin")
		return
	}
	_, err = a.DB.Exec(`update users set is_active = not is_active, updated_at=? where id=?`, time.Now(), id)
	if err != nil {
		a.audit(currentUser(r), "user.toggle_active", "users", strconv.FormatInt(id, 10), false, err.Error())
		writeError(w, http.StatusBadRequest, "toggle_failed", err.Error())
		return
	}
	u, _ := a.userByID(id)
	a.audit(currentUser(r), "user.toggle_active", "users", strconv.FormatInt(id, 10), true, "")
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "user": u})
}

func (a *App) handleUserStats(w http.ResponseWriter, r *http.Request) {
	stats := a.userStats()
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": stats, "total": stats["total"], "active": stats["active"]})
}

func (a *App) handleAuditLogs(w http.ResponseWriter, r *http.Request) {
	conds := []string{"1=1"}
	args := []any{}
	if user := strings.TrimSpace(r.URL.Query().Get("username")); user != "" {
		conds = append(conds, "username like ?")
		args = append(args, "%"+user+"%")
	}
	if action := strings.TrimSpace(r.URL.Query().Get("action")); action != "" {
		conds = append(conds, "action like ?")
		args = append(args, "%"+action+"%")
	}
	if target := strings.TrimSpace(r.URL.Query().Get("target")); target != "" {
		conds = append(conds, "target like ?")
		args = append(args, "%"+target+"%")
	}
	where := strings.Join(conds, " and ")
	var total int
	_ = a.DB.QueryRow(`select count(*) from audit_logs where `+where, args...).Scan(&total)
	page, pageSize := pageParams(r, 50)
	query := `select id,user_id,username,action,target,detail,success,error,ip_address,created_at from audit_logs where ` + where + ` order by id desc limit ? offset ?`
	rows, err := a.DB.Query(query, append(args, pageSize, (page-1)*pageSize)...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	defer rows.Close()
	var items []map[string]any
	for rows.Next() {
		var id int64
		var userID sql.NullInt64
		var username, action, target, detail, errText, ip sql.NullString
		var success bool
		var created sql.NullTime
		_ = rows.Scan(&id, &userID, &username, &action, &target, &detail, &success, &errText, &ip, &created)
		items = append(items, map[string]any{
			"id": id, "user_id": nullableInt64(userID), "username": username.String, "action": action.String,
			"target": target.String, "detail": detail.String, "success": success, "error": errText.String,
			"ip_address": ip.String, "created_at": nullableTimeString(created),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": items, "items": items, "audit_logs": items, "pagination": pagination(page, pageSize, total), "meta": map[string]any{"pagination": pagination(page, pageSize, total)}})
}

func validUserRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin", "operator", "viewer", "guest":
		return true
	default:
		return false
	}
}

func (a *App) adminCount() int {
	return userRowCount(a.DB, "role='admin'")
}

func (a *App) activeAdminCount() int {
	return userRowCount(a.DB, "role='admin' and is_active=true")
}

func (a *App) userStats() map[string]any {
	total := userRowCount(a.DB, "")
	active := userRowCount(a.DB, "is_active=true")
	inactive := userRowCount(a.DB, "is_active=false")
	stats := map[string]any{"total": total, "active": active, "inactive": inactive, "disabled": inactive, "roles": map[string]int{}}
	roles := map[string]int{"admin": 0, "operator": 0, "viewer": 0, "guest": 0}
	rows, err := a.DB.Query(`select role,count(*) from users where deleted_at is null group by role`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var role string
			var count int
			_ = rows.Scan(&role, &count)
			roles[role] = count
			stats[role] = count
		}
	}
	stats["roles"] = roles
	return stats
}

func nullableInt64(v sql.NullInt64) any {
	if !v.Valid {
		return nil
	}
	return v.Int64
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
