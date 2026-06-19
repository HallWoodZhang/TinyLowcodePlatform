package db

import "time"

type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Label     string    `json:"label"`
	Betamap   string    `json:"betamap"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type User struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenantId"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"passwordHash,omitempty"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Script struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenantId"`
	Name      string    `json:"name"`
	Label     string    `json:"label"`
	Type      string    `json:"type"`
	TSCode    string    `json:"tsCode"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ScriptSummary struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenantId"`
	Name      string    `json:"name"`
	Label     string    `json:"label"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Breakpoint struct {
	ScriptID string `json:"scriptId"`
	Line     int    `json:"line"`
	Enabled  bool   `json:"enabled"`
}

type TenantStore interface {
	CreateTenant(name, label, betamap string) (*Tenant, error)
	GetTenant(id string) (*Tenant, error)
	GetTenantByName(name string) (*Tenant, error)
	ListTenants() ([]Tenant, error)
	UpdateTenant(id string, name, label, betamap *string) (*Tenant, error)
	DeleteTenant(id string) error
}

type UserStore interface {
	CreateUser(tenantID, username, passwordHash, role string) (*User, error)
	GetUser(id string) (*User, error)
	GetUserByTenantAndUsername(tenantName, username string) (*User, error)
	ListUsersByTenant(tenantID string) ([]User, error)
	ListAllUsers() ([]User, error)
	DeleteUser(id string) error
	UpdateUserPassword(id, passwordHash string) error
}

type ScriptStore interface {
	ListScripts(tenantID string) ([]ScriptSummary, error)
	ListAllScripts() ([]ScriptSummary, error)
	GetScript(id string) (*Script, error)
	GetScriptByName(tenantID, name string) (*Script, error)
	CreateScript(tenantID, name, label, scriptType, tsCode string) (*Script, error)
	UpdateScript(id string, name, label, scriptType, tsCode *string) (*Script, error)
	DeleteScript(id string) error
}

type BreakpointStore interface {
	ListBreakpoints(scriptID string) ([]Breakpoint, error)
	SetBreakpoint(scriptID string, line int, enabled bool) error
	DeleteBreakpoint(scriptID string, line int) error
}

type Store interface {
	TenantStore
	UserStore
	ScriptStore
	BreakpointStore
}
