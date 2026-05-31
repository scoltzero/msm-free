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
type authIdentityContextKey struct{}

type AuthIdentity struct {
	User       *User
	AuthType   string
	TokenID    int64
	TokenScope string
}

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

func currentIdentity(r *http.Request) *AuthIdentity {
	if identity, ok := r.Context().Value(authIdentityContextKey{}).(*AuthIdentity); ok {
		return identity
	}
	if u := currentUser(r); u != nil {
		return &AuthIdentity{User: u, AuthType: "jwt", TokenScope: "admin"}
	}
	return nil
}

func (a *App) authenticateRequest(r *http.Request) (*AuthIdentity, error) {
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
	if identity, err := a.authenticateJWT(tokenStr); err == nil {
		return identity, nil
	}
	return a.authenticateAPIToken(tokenStr)
}

func (a *App) authenticateJWT(tokenStr string) (*AuthIdentity, error) {
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
	u, err := a.userByID(claims.UserID)
	if err != nil {
		return nil, err
	}
	return &AuthIdentity{User: u, AuthType: "jwt", TokenScope: "admin"}, nil
}

func (a *App) authenticateAPIToken(token string) (*AuthIdentity, error) {
	hash := tokenHash(token)
	var userID int64
	var tokenID int64
	var scope string
	err := a.DB.QueryRow(`select id,user_id,coalesce(scope,'admin') from api_tokens where token_hash=? and revoked=false and (expires_at is null or expires_at > ?)`, hash, time.Now()).Scan(&tokenID, &userID, &scope)
	if err != nil {
		return nil, err
	}
	_, _ = a.DB.Exec(`update api_tokens set last_used_at=? where token_hash=?`, time.Now(), hash)
	u, err := a.userByID(userID)
	if err != nil {
		return nil, err
	}
	return &AuthIdentity{User: u, AuthType: "api_token", TokenID: tokenID, TokenScope: normalizeTokenScope(scope)}, nil
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

func (a *App) authorizeRequest(identity *AuthIdentity, r *http.Request) bool {
	if identity == nil || identity.User == nil || !identity.User.IsActive {
		return false
	}
	u := identity.User
	if !roleAllows(u.Role, r.Method, r.URL.Path) {
		return false
	}
	if identity.AuthType != "api_token" {
		return true
	}
	return tokenScopeAllows(identity.TokenScope, r.Method, r.URL.Path)
}

func roleAllows(role, method, path string) bool {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		role = "guest"
	}
	if role == "admin" {
		return true
	}
	if selfServiceAllows(method, path) {
		return true
	}
	if path == "/api/v1/settings/profile" || path == "/api/v1/settings/appearance" {
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
		"/api/v1/daemon",
		"/api/v1/users",
		"/api/v1/audit-logs",
		"/api/v1/api-tokens",
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
		"/api/v1/audit-logs",
		"/api/v1/api-tokens",
	}
	for _, prefix := range deniedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	return strings.HasPrefix(path, "/api/v1/version") ||
		strings.HasPrefix(path, "/api/v1/monitor") ||
		strings.HasPrefix(path, "/api/v1/services") ||
		strings.HasPrefix(path, "/api/v1/logs") ||
		strings.HasPrefix(path, "/api/v1/events/logs") ||
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

func normalizeTokenScope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "read", "operate", "admin":
		return strings.ToLower(strings.TrimSpace(scope))
	default:
		return "admin"
	}
}

func validTokenScope(scope string) bool {
	switch normalizeTokenScope(scope) {
	case "read", "operate", "admin":
		return strings.TrimSpace(scope) == "" || strings.EqualFold(strings.TrimSpace(scope), normalizeTokenScope(scope))
	default:
		return false
	}
}

func tokenScopeAllows(scope, method, path string) bool {
	switch normalizeTokenScope(scope) {
	case "admin":
		return true
	case "operate":
		return selfServiceAllows(method, path) ||
			path == "/api/v1/settings/profile" ||
			path == "/api/v1/settings/appearance" ||
			operatorAllows(method, path)
	case "read":
		return readScopeAllows(method, path)
	default:
		return false
	}
}

func selfServiceAllows(method, path string) bool {
	return path == "/api/v1/auth/me" ||
		path == "/api/v1/profile" ||
		path == "/api/v1/profile/password"
}

func readScopeAllows(method, path string) bool {
	if method != http.MethodGet {
		return false
	}
	if path == "/api/v1/auth/me" || path == "/api/v1/profile" || path == "/api/v1/settings/profile" || path == "/api/v1/settings/appearance" {
		return true
	}
	if path == "/api/v1/version" ||
		strings.HasPrefix(path, "/api/v1/monitor") ||
		strings.HasPrefix(path, "/api/v1/services") ||
		strings.HasPrefix(path, "/api/v1/logs") ||
		strings.HasPrefix(path, "/api/v1/events/logs") ||
		strings.HasPrefix(path, "/api/v1/update/status") ||
		strings.HasPrefix(path, "/api/v1/update/releases") ||
		strings.HasPrefix(path, "/api/v1/component-updates") {
		return true
	}
	for _, prefix := range []string{
		"/api/v1/mosdns/status",
		"/api/v1/mosdns/overview",
		"/api/v1/mosdns/stats",
		"/api/v1/mosdns/metrics",
		"/api/v1/mosdns/version",
		"/api/v1/mosdns/logs",
		"/api/v1/mosdns/query-log",
		"/api/v1/mosdns/query-logs",
		"/api/v1/mosdns/audit",
		"/api/v1/mosdns/cache/detailed",
		"/api/v1/mosdns/upstream/stats",
		"/api/v1/mihomo/status",
		"/api/v1/mihomo/overview",
		"/api/v1/mihomo/dashboard",
		"/api/v1/mihomo/summary",
		"/api/v1/mihomo/stats",
		"/api/v1/mihomo/traffic",
		"/api/v1/mihomo/logs",
		"/api/v1/proxy/status",
		"/api/v1/proxy/overview",
		"/api/v1/proxy/stats",
		"/api/v1/proxy/traffic",
		"/api/v1/proxy/logs",
	} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
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
	ctx = context.WithValue(ctx, userContextKey{}, u)
	return context.WithValue(ctx, authIdentityContextKey{}, &AuthIdentity{User: u, AuthType: "jwt", TokenScope: "admin"})
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
