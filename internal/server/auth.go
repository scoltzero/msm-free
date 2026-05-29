package server

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type userContextKey struct{}

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func currentUser(r *http.Request) *User {
	if u, ok := r.Context().Value(userContextKey{}).(*User); ok {
		return u
	}
	return nil
}

func (a *App) authenticateRequest(r *http.Request) (*User, error) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		if token := r.URL.Query().Get("token"); token != "" {
			auth = "Bearer " + token
		}
	}
	if !strings.HasPrefix(auth, "Bearer ") {
		return nil, errors.New("missing bearer")
	}
	tokenStr := strings.TrimPrefix(auth, "Bearer ")
	if u, err := a.authenticateJWT(tokenStr); err == nil {
		return u, nil
	}
	return a.authenticateAPIToken(tokenStr)
}

func (a *App) authenticateJWT(tokenStr string) (*User, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		return a.Secret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid jwt")
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return a.userByID(claims.UserID)
}

func (a *App) authenticateAPIToken(token string) (*User, error) {
	hash := tokenHash(token)
	var userID int64
	err := a.DB.QueryRow(`select user_id from api_tokens where token_hash=? and revoked=false and (expires_at is null or expires_at > ?)`, hash, time.Now()).Scan(&userID)
	if err != nil {
		return nil, err
	}
	_, _ = a.DB.Exec(`update api_tokens set last_used_at=? where token_hash=?`, time.Now(), hash)
	return a.userByID(userID)
}

func (a *App) makeToken(u *User, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   u.ID,
		Username: u.Username,
		Role:     u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.Username,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.Secret)
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	u, hash, err := a.findUserByUsername(req.Username)
	if err != nil || !u.IsActive {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "用户名或密码错误")
		return
	}
	if !passwordMatches(hash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "用户名或密码错误")
		return
	}
	token, err := a.makeToken(u, 24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_error", err.Error())
		return
	}
	refresh := randomHex(32)
	_, _ = a.DB.Exec(`insert into refresh_tokens(user_id,token_hash,expires_at,created_at) values(?,?,?,?)`, u.ID, tokenHash(refresh), time.Now().Add(30*24*time.Hour), time.Now())
	_, _ = a.DB.Exec(`update users set last_login=?, failed_attempts=0 where id=?`, time.Now(), u.ID)
	a.audit(u, "login", "auth", "", true, "")
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "refresh_token": refresh, "user": u, "success": true})
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) handleRefresh(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	refresh := strings.TrimPrefix(auth, "Bearer ")
	if refresh == "" || refresh == auth {
		var req struct {
			RefreshToken string `json:"refresh_token"`
			Refresh      string `json:"refresh"`
			Token        string `json:"token"`
		}
		if r.Body != nil {
			defer r.Body.Close()
			_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req)
		}
		refresh = req.RefreshToken
		if refresh == "" {
			refresh = req.Refresh
		}
		if refresh == "" {
			refresh = req.Token
		}
		if refresh == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "refresh token required")
			return
		}
	}
	var userID int64
	err := a.DB.QueryRow(`select user_id from refresh_tokens where token_hash=? and revoked=false and expires_at > ?`, tokenHash(refresh), time.Now()).Scan(&userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "refresh token invalid")
		return
	}
	u, err := a.userByID(userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}
	token, err := a.makeToken(u, 24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": u, "success": true})
}

func (a *App) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "user": currentUser(r)})
}

func (a *App) handleProfile(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请提供认证令牌")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": u, "user": u})
}

func (a *App) handleProfileUpdate(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "请提供认证令牌")
		return
	}
	var req struct {
		Email       string `json:"email"`
		DisplayName string `json:"display_name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	_, err := a.DB.Exec(`update users set email=?,display_name=?,updated_at=? where id=?`, req.Email, req.DisplayName, time.Now(), u.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "update_failed", err.Error())
		return
	}
	updated, err := a.userByID(u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": updated, "user": updated})
}

func (a *App) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	_, hash, err := a.findUserByUsername(u.Username)
	if err != nil || !passwordMatches(hash, req.OldPassword) {
		writeError(w, http.StatusBadRequest, "invalid_password", "旧密码错误")
		return
	}
	newHash, _ := bcrypt.GenerateFromPassword([]byte(normalizePasswordForStorage(req.NewPassword)), bcrypt.DefaultCost)
	_, err = a.DB.Exec(`update users set password=?, updated_at=? where id=?`, string(newHash), time.Now(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (a *App) requireAdmin(r *http.Request) bool {
	u := currentUser(r)
	return u != nil && u.Role == "admin"
}

func (a *App) authorizeRequest(u *User, r *http.Request) bool {
	if u == nil || !u.IsActive {
		return false
	}
	role := strings.ToLower(strings.TrimSpace(u.Role))
	if role == "" {
		role = "guest"
	}
	if role == "admin" {
		return true
	}
	path := r.URL.Path
	method := r.Method
	if path == "/api/v1/auth/me" || path == "/api/v1/profile" || path == "/api/v1/profile/password" {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/api-tokens") {
		return role != "guest"
	}
	if role == "operator" {
		return operatorAllows(method, path)
	}
	if role == "viewer" {
		return viewerAllows(method, path)
	}
	if role == "guest" {
		return guestAllows(method, path)
	}
	return false
}

func operatorAllows(method, path string) bool {
	deniedPrefixes := []string{
		"/api/v1/users",
		"/api/v1/settings",
		"/api/v1/setup/reset",
		"/api/v1/license-activation/activate",
		"/api/v1/license-activation/deactivate",
		"/api/v1/license-activation/refresh",
	}
	for _, prefix := range deniedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	if strings.HasPrefix(path, "/api/v1/history/") && (method == http.MethodPost || method == http.MethodDelete) {
		return false
	}
	return true
}

func viewerAllows(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	deniedPrefixes := []string{
		"/api/v1/users",
		"/api/v1/settings",
		"/api/v1/logs",
		"/api/v1/events/logs",
		"/api/v1/mosdns/logs",
		"/api/v1/mosdns/query-log",
		"/api/v1/mosdns/query-logs",
		"/api/v1/mosdns/audit",
	}
	for _, prefix := range deniedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	return strings.HasPrefix(path, "/api/v1/version") ||
		strings.HasPrefix(path, "/api/v1/monitor") ||
		strings.HasPrefix(path, "/api/v1/services") ||
		strings.HasPrefix(path, "/api/v1/config") ||
		strings.HasPrefix(path, "/api/v1/history") ||
		strings.HasPrefix(path, "/api/v1/mosdns") ||
		strings.HasPrefix(path, "/api/v1/mihomo") ||
		strings.HasPrefix(path, "/api/v1/network/info") ||
		strings.HasPrefix(path, "/api/v1/netlink/nftables/status") ||
		strings.HasPrefix(path, "/api/v1/system/diagnostics") ||
		strings.HasPrefix(path, "/api/v1/update") ||
		strings.HasPrefix(path, "/api/v1/component-updates")
}

func guestAllows(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	return path == "/api/v1/version" ||
		path == "/api/v1/auth/me" ||
		path == "/api/v1/profile" ||
		path == "/api/v1/monitor/system" ||
		path == "/api/v1/monitor/hardware" ||
		path == "/api/v1/license-activation/status"
}

func withUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userContextKey{}, u)
}

func passwordMatches(hash, supplied string) bool {
	candidates := []string{supplied, normalizePasswordForStorage(supplied)}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(candidate)) == nil {
			return true
		}
	}
	return false
}

func normalizePasswordForStorage(password string) string {
	if isSHA256Hex(password) {
		return strings.ToLower(password)
	}
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

func isSHA256Hex(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return false
	}
	return true
}

func isNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
