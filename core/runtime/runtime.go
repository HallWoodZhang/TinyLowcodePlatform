//go:build !noquickjs

package runtime

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	qjs "github.com/quickjs-go/quickjs-go"
)

type QuickJSEngine struct{}

func (e *QuickJSEngine) Run(tsCode string, resolver ScriptResolver, timeoutMs int64) RunResult {
	jsCode, mapper, err := buildJS(tsCode, resolver)
	if err != nil {
		return RunResult{Error: err.Error()}
	}
	result := e.execute(jsCode)
	// Map JS error line to TS if available
	if result.Error != "" && mapper != nil {
		result.Error, result.ErrorLine, result.ErrorTSLine = mapErrorLineWithOffset(result.Error, mapper, -1)
	}
	return result
}

func (e *QuickJSEngine) Debug(tsCode string, resolver ScriptResolver, bps []BpLine, skip int, timeoutMs int64) RunResult {
	jsCode, mapper, err := buildJS(tsCode, resolver)
	if err != nil {
		return RunResult{Error: err.Error()}
	}
	jsLines := mapTSBreakpointsToJS(mapper, bps)
	if len(jsLines) > 0 {
		jsCode = instrumentCode(jsCode, jsLines, nil)
	}
	result := e.execute(jsCode)
	// Map error line
	if result.Error != "" && mapper != nil {
		result.Error, result.ErrorLine, result.ErrorTSLine = mapErrorLineWithOffset(result.Error, mapper, -1)
	}
	for i := range result.Breakpoints {
		if mapper != nil {
			if _, _, sl, _, ok := mapper.Source(result.Breakpoints[i].Line, 0); ok {
				result.Breakpoints[i].Line = sl
			}
		}
	}
	return result
}

func (e *QuickJSEngine) execute(jsCode string) RunResult {
	runtime := qjs.NewRuntime()
	defer runtime.Free()

	jsCtx := runtime.NewContext()
	defer jsCtx.Free()

	var output strings.Builder
	var bpOutput strings.Builder

	console := jsCtx.Object()
	console.Set("log", jsCtx.Function(func(ctx *qjs.Context, this qjs.Value, args []qjs.Value) qjs.Value {
		var parts []string
		for _, arg := range args {
			parts = append(parts, arg.String())
		}
		output.WriteString(strings.Join(parts, " "))
		output.WriteString("\n")
		return ctx.Null()
	}))
	console.Set("error", jsCtx.Function(func(ctx *qjs.Context, this qjs.Value, args []qjs.Value) qjs.Value {
		var parts []string
		for _, arg := range args {
			parts = append(parts, arg.String())
		}
		output.WriteString("[ERR] " + strings.Join(parts, " "))
		output.WriteString("\n")
		return ctx.Null()
	}))
	jsCtx.Globals().Set("console", console)

	jsCtx.Globals().Set("__dbg", jsCtx.Function(func(ctx *qjs.Context, this qjs.Value, args []qjs.Value) qjs.Value {
		if len(args) > 0 {
			bpOutput.WriteString(fmt.Sprintf("__DBG_LINE__:%s\n", args[0].String()))
		}
		return ctx.Null()
	}))

	// Wrap in try-catch to capture Error.stack (QuickJS supports it)
	wrappedCode := "try {\n" + jsCode + "\n} catch(__e) { console.log('__TRACE__:' + __e.toString() + '\\n' + (__e.stack || '')); }"

	result, err := jsCtx.EvalFile(wrappedCode, qjs.EVAL_GLOBAL, "script.ts")
	defer func() {
		if result.IsObject() {
			result.Free()
		}
	}()

	outStr := output.String()

	// Check for caught exception in console output
	if idx := strings.Index(outStr, "__TRACE__:"); idx >= 0 {
		trace := strings.TrimSpace(outStr[idx+len("__TRACE__:"):])
		cleanOutput := strings.TrimSpace(outStr[:idx])
		if cleanOutput == "" {
			cleanOutput = "(no output before error)"
		}
		return RunResult{
			Output: cleanOutput,
			Error:  "Runtime error:\n" + trace,
		}
	}

	// If EvalFile itself returned an error (syntax error etc.)
	if err != nil {
		return RunResult{
			Output: outStr,
			Error:  "Runtime error:\n" + err.Error(),
		}
	}

	runResult := RunResult{}
	if output.Len() > 0 {
		runResult.Output = outStr
	} else if result.IsUndefined() {
		runResult.Output = "undefined"
	} else {
		runResult.Output = result.String()
	}
	if bpOutput.Len() > 0 {
		runResult.Breakpoints = parseBpHits(bpOutput.String())
	}
	return runResult
}

func parseBpHits(raw string) []BreakpointHit {
	var hits []BreakpointHit
	var current *BreakpointHit

	lines := strings.Split(raw, "\n")
	for _, l := range lines {
		if strings.HasPrefix(l, "__DBG_LINE__:") {
			if current != nil {
				hits = append(hits, *current)
			}
			lineStr := strings.TrimPrefix(l, "__DBG_LINE__:")
			line, _ := strconv.Atoi(lineStr)
			current = &BreakpointHit{Line: line}
		} else if current != nil && strings.HasPrefix(l, "__DBG_STACK__:") {
			current.Name = strings.TrimPrefix(l, "__DBG_STACK__:")
		} else if current != nil && strings.HasPrefix(l, "__DBG_GLOBALS__:") {
			current.Vars = make(map[string]any)
			gs := strings.TrimPrefix(l, "__DBG_GLOBALS__:")
			var rawVars map[string]string
			if err := json.Unmarshal([]byte(gs), &rawVars); err == nil {
				for k, v := range rawVars {
					current.Vars[k] = v
				}
			}
		}
	}
	if current != nil {
		hits = append(hits, *current)
	}
	return hits
}
