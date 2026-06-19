package idgen

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewTenantID(t *testing.T) {
	id := NewTenantID()
	if len(id) != 24 {
		t.Errorf("len = %d, want 24", len(id))
	}
	if !strings.HasPrefix(id, KeyPrefixTenant) {
		t.Errorf("id = %s, want prefix %s", id, KeyPrefixTenant)
	}
}

func TestNewUserID(t *testing.T) {
	id := NewUserID()
	if len(id) != 24 {
		t.Errorf("len = %d, want 24", len(id))
	}
	if !strings.HasPrefix(id, KeyPrefixUser) {
		t.Errorf("id = %s, want prefix %s", id, KeyPrefixUser)
	}
}

func TestNewScriptID(t *testing.T) {
	id := NewScriptID()
	if len(id) != 24 {
		t.Errorf("len = %d, want 24", len(id))
	}
	if !strings.HasPrefix(id, KeyPrefixScript) {
		t.Errorf("id = %s, want prefix %s", id, KeyPrefixScript)
	}
}

func TestNewBreakpointID(t *testing.T) {
	id := NewBreakpointID()
	if len(id) != 24 {
		t.Errorf("len = %d, want 24", len(id))
	}
	if !strings.HasPrefix(id, KeyPrefixBreakpoint) {
		t.Errorf("id = %s, want prefix %s", id, KeyPrefixBreakpoint)
	}
}

func TestIDUniqueness(t *testing.T) {
	const n = 10000
	ids := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		id := NewScriptID()
		if ids[id] {
			t.Errorf("duplicate id: %s at iteration %d", id, i)
			return
		}
		ids[id] = true
	}
}

func TestIDConcurrent(t *testing.T) {
	const goroutines = 100
	const perGoroutine = 100

	var wg sync.WaitGroup
	ch := make(chan string, goroutines*perGoroutine)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				ch <- NewTenantID()
			}
		}()
	}
	wg.Wait()
	close(ch)

	ids := make(map[string]bool)
	for id := range ch {
		if ids[id] {
			t.Errorf("duplicate id in concurrent generation: %s", id)
			return
		}
		ids[id] = true
	}
	if len(ids) != goroutines*perGoroutine {
		t.Errorf("expected %d unique ids, got %d", goroutines*perGoroutine, len(ids))
	}
}

func TestIDLexicographicOrder(t *testing.T) {
	id1 := NewScriptID()
	time.Sleep(2 * time.Millisecond)
	id2 := NewScriptID()

	if id1 >= id2 {
		t.Errorf("id1=%s should be < id2=%s", id1, id2)
	}
}
