package runtime

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dop251/goja"
)

type GojaEngine struct{}

func (e *GojaEngine) Run(tsCode string, resolver ScriptResolver, timeoutMs int64) RunResult {
	jsCode, _, err := buildJS(tsCode, resolver)
	if err != nil {
		return RunResult{Error: err.Error()}
	}
	return e.execute(jsCode, timeoutMs)
}

func (e *GojaEngine) Debug(tsCode string, resolver ScriptResolver, bps []BpLine, timeoutMs int64) RunResult {
	jsCode, mapper, err := buildJS(tsCode, resolver)
	if err != nil {
		return RunResult{Error: err.Error()}
	}
	jsLines := mapTSBreakpointsToJS(mapper, bps)

	// Build variable name map for each breakpoint line
	lineVars := make(map[int][]string)
	for _, jsLine := range jsLines {
		vars := FindScopeVars(jsCode, jsLine)
		if len(vars) > 0 {
			lineVars[jsLine] = vars
		}
	}

	if len(jsLines) > 0 {
		jsCode = instrumentCode(jsCode, jsLines, lineVars)
	}
	result := e.execute(jsCode, timeoutMs)
	for i := range result.Breakpoints {
		if mapper != nil {
			if _, _, sl, _, ok := mapper.Source(result.Breakpoints[i].Line, 0); ok {
				result.Breakpoints[i].Line = sl
			}
		}
	}
	return result
}

func (e *GojaEngine) execute(jsCode string, timeoutMs int64) RunResult {
	vm := goja.New()

	var output strings.Builder
	var bpOutput strings.Builder

	vm.Set("console", map[string]interface{}{
		"log": func(args ...interface{}) {
			var parts []string
			for _, a := range args {
				parts = append(parts, fmt.Sprint(a))
			}
			output.WriteString(strings.Join(parts, " ") + "\n")
		},
		"error": func(args ...interface{}) {
			var parts []string
			for _, a := range args {
				parts = append(parts, fmt.Sprint(a))
			}
			output.WriteString("[ERR] " + strings.Join(parts, " ") + "\n")
		},
	})

	vm.Set("__dbg", func(call goja.FunctionCall) goja.Value {
		line := 0
		if len(call.Arguments) > 0 {
			line = int(call.Arguments[0].ToInteger())
		}
		stack := vm.CaptureCallStack(50, nil)
		var stackLines []string
		for _, frame := range stack {
			stackLines = append(stackLines, fmt.Sprintf("    at %s (%s:%d)", frame.FuncName(), frame.Position().Filename, frame.Position().Line))
		}

		var localVars []string
		if len(call.Arguments) > 1 {
			varsObj := call.Arguments[1].ToObject(vm)
			for _, key := range varsObj.Keys() {
				v := varsObj.Get(key)
				if v != nil {
					s := v.String()
					if len(s) > 200 {
						s = s[:200] + "..."
					}
					localVars = append(localVars, fmt.Sprintf("%s: %s", key, s))
				}
			}
		}

		bpOutput.WriteString(fmt.Sprintf("__DBG_LINE__:%d\n", line))
		bpOutput.WriteString(fmt.Sprintf("__DBG_STACK__:%s\n", strings.Join(stackLines, "\\n")))
		bpOutput.WriteString(fmt.Sprintf("__DBG_GLOBALS__:%s\n", strings.Join(localVars, "|")))
		vm.Interrupt("__BP_STOP__")
		return goja.Undefined()
	})

	timer := time.AfterFunc(time.Duration(timeoutMs)*time.Millisecond, func() {
		vm.Interrupt("execution timed out")
	})
	defer timer.Stop()

	val, err := vm.RunString(jsCode)
	if err != nil {
		if bpOutput.Len() > 0 {
			runResult := RunResult{}
			if output.Len() > 0 {
				runResult.Output = output.String()
			}
			runResult.Breakpoints = parseGojaBpHits(bpOutput.String())
			return runResult
		}
		return RunResult{
			Output: output.String(),
			Error:  "Runtime error: " + err.Error(),
		}
	}

	runResult := RunResult{}
	if output.Len() > 0 {
		runResult.Output = output.String()
	} else if val != nil {
		runResult.Output = val.String()
	} else {
		runResult.Output = "undefined"
	}

	if bpOutput.Len() > 0 {
		runResult.Breakpoints = parseGojaBpHits(bpOutput.String())
	}
	return runResult
}

func parseGojaBpHits(raw string) []BreakpointHit {
	var hits []BreakpointHit
	var current *BreakpointHit

	lines := strings.Split(raw, "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "__DBG_LINE__:") {
			if current != nil {
				hits = append(hits, *current)
			}
			lineStr := strings.TrimPrefix(l, "__DBG_LINE__:")
			var line int
			fmt.Sscanf(lineStr, "%d", &line)
			current = &BreakpointHit{Line: line}
		} else if current != nil && strings.HasPrefix(l, "__DBG_STACK__:") {
			current.Name = strings.TrimPrefix(l, "__DBG_STACK__:")
		} else if current != nil && strings.HasPrefix(l, "__DBG_GLOBALS__:") {
			current.Vars = make(map[string]any)
			rawVars := strings.TrimPrefix(l, "__DBG_GLOBALS__:")
			if rawVars != "" {
				for _, pair := range strings.Split(rawVars, "|") {
					parts := strings.SplitN(pair, ":", 2)
					if len(parts) == 2 {
						current.Vars[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
					}
				}
			}
		}
	}
	if current != nil {
		hits = append(hits, *current)
	}
	return hits
}

var _ = sort.IntSlice(nil)
