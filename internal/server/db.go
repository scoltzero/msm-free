package server

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID          int64      `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email,omitempty"`
	DisplayName string     `json:"display_name,omitempty"`
	Role        string     `json:"role"`
	IsActive    bool       `json:"is_active"`
	LastLogin   *time.Time `json:"last_login,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (a *App) migrate() error {
	stmts := []string{
		`pragma journal_mode = wal`,
		`create table if not exists settings (key text primary key, value text not null, updated_at datetime)`,
		`create table if not exists users (
			id integer primary key autoincrement,
			username text not null unique,
			password text not null,
			email text,
			display_name text,
			role text default 'operator',
			is_active numeric default true,
			last_login datetime,
			failed_attempts integer default 0,
			locked_until datetime,
			created_at datetime,
			updated_at datetime,
			deleted_at datetime
		)`,
		`create table if not exists refresh_tokens (
			id integer primary key autoincrement,
			user_id integer not null,
			token_hash text not null,
			expires_at datetime,
			revoked numeric default false,
			created_at datetime
		)`,
		`create table if not exists api_tokens (
			id integer primary key autoincrement,
			user_id integer not null,
			name text not null,
			token_hash text not null,
			last_used_at datetime,
			expires_at datetime,
			created_at datetime,
			revoked numeric default false
		)`,
		`create table if not exists audit_logs (
			id integer primary key autoincrement,
			user_id integer,
			username text,
			action text,
			target text,
			detail text,
			success numeric,
			error text,
			ip_address text,
			created_at datetime
		)`,
		`create table if not exists system_setups (
			id integer primary key autoincrement,
			created_at datetime,
			updated_at datetime,
			username text not null,
			email text,
			web_port text default '7777',
			amd64v3_enabled numeric default false,
			selected_interface text,
			singbox_core_type text default '',
			mihomo_core_type text default 'meta',
			auto_set_dns numeric default true,
			dns_on text default '127.0.0.1',
			dns_off text default '223.5.5.5',
			enable_ipv6 numeric default true,
			fake_ip_range_v4 text default '28.0.0.0/8',
			fake_ip_range_v6 text default 'f2b0::/18',
			linux_proxy_mode text default 'nft',
			nft_proxy_policy text default 'direct_default',
			proxy_core text default 'mihomo',
			mos_dns_enabled numeric default true,
			subscription_urls text,
			mihomo_proxies text,
			github_proxy_enabled numeric default false,
			github_https_proxy text,
			github_http_proxy text,
			github_socks5_proxy text,
			github_accelerator_enabled numeric default false,
			github_accelerator_url text,
			is_initialized numeric default false
		)`,
		`create table if not exists config_histories (
			id integer primary key autoincrement,
			service text not null,
			file_path text not null,
			content text,
			comment text,
			is_stable numeric default false,
			created_by text default 'admin',
			created_at datetime,
			updated_at datetime,
			deleted_at datetime
		)`,
		`create table if not exists mosdns_clients (
			id integer primary key autoincrement,
			mac text,
			ip text not null,
			hostname text,
			vendor text,
			custom_name text,
			custom_desc text,
			source text,
			type text,
			query_count integer default 0,
			first_seen_at datetime,
			last_seen_at datetime,
			last_scan_at datetime,
			interface text,
			is_online numeric default false,
			created_at datetime,
			updated_at datetime
		)`,
		`create unique index if not exists idx_mac_ip on mosdns_clients(mac, ip)`,
		`create table if not exists mosdns_client_ips (
			id integer primary key autoincrement,
			ip text not null unique,
			comment text,
			created_at datetime,
			updated_at datetime
		)`,
		`create table if not exists mosdns_switch_states (
			id integer primary key autoincrement,
			switch_key text not null unique,
			enabled numeric,
			created_at datetime,
			updated_at datetime
		)`,
		`create table if not exists update_info (
			id integer primary key autoincrement,
			component text default 'msm-free',
			current_version text,
			latest_version text,
			has_update numeric default false,
			status text default 'idle',
			progress integer default 0,
			error_message text,
			download_url text,
			release_notes text,
			last_check_time datetime,
			created_at datetime,
			updated_at datetime
		)`,
		`create table if not exists component_update_info (
			id integer primary key autoincrement,
			component text not null unique,
			current_version text,
			latest_version text,
			has_update numeric default false,
			download_url text,
			release_body text,
			status text default 'idle',
			progress integer default 0,
			error_message text,
			last_check_time datetime,
			created_at datetime,
			updated_at datetime
		)`,
		`create table if not exists component_update_config (
			id integer primary key autoincrement,
			component text not null unique,
			auto_check numeric default true,
			check_interval integer default 86400,
			auto_update numeric default false,
			created_at datetime,
			updated_at datetime
		)`,
	}
	for _, stmt := range stmts {
		if _, err := a.DB.Exec(stmt); err != nil {
			return fmt.Errorf("migrate %q: %w", stmt, err)
		}
	}
	return nil
}

func (a *App) IsInitialized() bool {
	var ok bool
	err := a.DB.QueryRow(`select is_initialized from system_setups order by id desc limit 1`).Scan(&ok)
	return err == nil && ok
}

func (a *App) ResetAdminPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(normalizePasswordForStorage(password)), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	res, err := a.DB.Exec(`update users set password=?, updated_at=? where role='admin' and deleted_at is null`, string(hash), time.Now())
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		_, err = a.DB.Exec(`insert into users(username,password,role,is_active,created_at,updated_at) values('admin',?,'admin',true,?,?)`, string(hash), time.Now(), time.Now())
	}
	return err
}

func (a *App) createOrUpdateAdmin(username, password, email string) error {
	if username == "" || password == "" {
		return errors.New("username and password are required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(normalizePasswordForStorage(password)), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := time.Now()
	_, err = a.DB.Exec(`insert into users(username,password,email,display_name,role,is_active,created_at,updated_at)
		values(?,?,?,?,?,?,?,?)
		on conflict(username) do update set password=excluded.password,email=excluded.email,role='admin',is_active=true,updated_at=excluded.updated_at,deleted_at=null`,
		username, string(hash), email, username, "admin", true, now, now)
	return err
}

func (a *App) findUserByUsername(username string) (*User, string, error) {
	row := a.DB.QueryRow(`select id, username, password, coalesce(email,''), coalesce(display_name,''), role, is_active, last_login, created_at, updated_at
		from users where username=? and deleted_at is null`, username)
	var u User
	var password string
	var last sql.NullTime
	if err := row.Scan(&u.ID, &u.Username, &password, &u.Email, &u.DisplayName, &u.Role, &u.IsActive, &last, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, "", err
	}
	if last.Valid {
		u.LastLogin = &last.Time
	}
	return &u, password, nil
}

func (a *App) userByID(id int64) (*User, error) {
	row := a.DB.QueryRow(`select id, username, coalesce(email,''), coalesce(display_name,''), role, is_active, last_login, created_at, updated_at
		from users where id=? and deleted_at is null`, id)
	var u User
	var last sql.NullTime
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.DisplayName, &u.Role, &u.IsActive, &last, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	if last.Valid {
		u.LastLogin = &last.Time
	}
	return &u, nil
}

func (a *App) audit(user *User, action, target, detail string, success bool, errText string) {
	var userID any
	var username string
	if user != nil {
		userID = user.ID
		username = user.Username
	}
	_, _ = a.DB.Exec(`insert into audit_logs(user_id,username,action,target,detail,success,error,created_at) values(?,?,?,?,?,?,?,?)`,
		userID, username, action, target, detail, success, errText, time.Now())
}
