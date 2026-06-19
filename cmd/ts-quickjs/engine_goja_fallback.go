//go:build noquickjs

package main

import (
	"tiny-lowcode-platform/core/runtime"
)

func newEngine(engineType string) runtime.Runner {
	return &runtime.GojaEngine{}
}
