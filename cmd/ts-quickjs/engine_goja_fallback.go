//go:build noquickjs

package main

import (
	"toy-platform/core/runtime"
)

func newEngine(engineType string) runtime.Runner {
	return &runtime.GojaEngine{}
}
