package runtime

import (
	sm "github.com/go-sourcemap/sourcemap"
)

type RunResult struct {
	Output      string          `json:"output,omitempty"`
	Error       string          `json:"error,omitempty"`
	ErrorLine   int             `json:"errorLine,omitempty"`
	ErrorTSLine int             `json:"errorTSLine,omitempty"`
	Breakpoints []BreakpointHit `json:"breakpoints,omitempty"`
	HitCount    int             `json:"hitCount,omitempty"`
}

type BreakpointHit struct {
	Line int            `json:"line"`
	Name string         `json:"name,omitempty"`
	Vars map[string]any `json:"vars,omitempty"`
}

type BpLine struct {
	Line    int
	Enabled bool
}

type ScriptResolver func(name string) (source string, err error)

type Runner interface {
	Run(tsCode string, resolver ScriptResolver, timeoutMs int64) RunResult
	Debug(tsCode string, resolver ScriptResolver, bps []BpLine, skip int, timeoutMs int64) RunResult
}

var _ = (*sm.Consumer)(nil)
