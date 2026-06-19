package idgen

import (
	"crypto/rand"
	"fmt"
	"time"
)

const (
	KeyPrefixTenant     = "001a"
	KeyPrefixUser       = "001b"
	KeyPrefixScript     = "001c"
	KeyPrefixBreakpoint = "001d"
)

func NewTenantID() string     { return newID(KeyPrefixTenant) }
func NewUserID() string       { return newID(KeyPrefixUser) }
func NewScriptID() string     { return newID(KeyPrefixScript) }
func NewBreakpointID() string { return newID(KeyPrefixBreakpoint) }

func newID(prefix string) string {
	ms := time.Now().UnixMilli()
	var r [4]byte
	if _, err := rand.Read(r[:]); err != nil {
		panic(fmt.Sprintf("idgen: crypto/rand failed: %v", err))
	}
	return fmt.Sprintf("%s%012x%08x", prefix, ms, r)
}
