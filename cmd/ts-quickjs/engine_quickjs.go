//go:build !noquickjs

package main

import (
	"tiny-lowcode-platform/core/runtime"
)

func newEngine(engineType string) runtime.Runner {
	if engineType == "goja" {
		return &runtime.GojaEngine{}
	}
	return &runtime.QuickJSEngine{}
}
