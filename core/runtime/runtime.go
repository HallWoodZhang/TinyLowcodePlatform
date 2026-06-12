package runtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	sm "github.com/go-sourcemap/sourcemap"
	"github.com/evanw/esbuild/pkg/api"
	qjs "github.com/quickjs-go/quickjs-go"
)

type RunResult struct {
	Output      string          `json:"output,omitempty"`
	Error       string          `json:"error,omitempty"`
	Breakpoints []BreakpointHit `json:"breakpoints,omitempty"`
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
	Debug(tsCode string, resolver ScriptResolver, bps []BpLine, timeoutMs int64) RunResult
}

type Engine struct{}

var stackLineRE = regexp.MustCompile(`script\.ts:(\d+)`)
var inlineSourcemapRE = regexp.MustCompile(`//# sourceMappingURL=data:application/json;base64,([^\s]+)`)

func (e *Engine) Run(tsCode string, resolver ScriptResolver, timeoutMs int64) RunResult {
	result := e.build(tsCode, resolver)
	if len(result.Errors) > 0 {
		var errs []string
		for _, msg := range result.Errors {
			errs = append(errs, msg.Text)
		}
		return RunResult{Error: "Build error:\n" + strings.Join(errs, "\n")}
	}
	if len(result.OutputFiles) == 0 {
		return RunResult{Error: "Build produced no output"}
	}

	jsCode := string(result.OutputFiles[0].Contents)
	match := inlineSourcemapRE.FindStringSubmatch(jsCode)
	var mapper *sm.Consumer
	if match != nil {
		decoded, _ := base64.StdEncoding.DecodeString(match[1])
		mapper = loadSourceMap(decoded)
	}

	return e.executeWithTimeout(jsCode, mapper, timeoutMs)
}

func (e *Engine) Debug(tsCode string, resolver ScriptResolver, bps []BpLine, timeoutMs int64) RunResult {
	buildResult := e.build(tsCode, resolver)
	if len(buildResult.Errors) > 0 {
		var errs []string
		for _, msg := range buildResult.Errors {
			errs = append(errs, msg.Text)
		}
		return RunResult{Error: "Build error:\n" + strings.Join(errs, "\n")}
	}
	if len(buildResult.OutputFiles) == 0 {
		return RunResult{Error: "Build produced no output"}
	}

	jsCode := string(buildResult.OutputFiles[0].Contents)

	match := inlineSourcemapRE.FindStringSubmatch(jsCode)
	var mapper *sm.Consumer
	if match != nil {
		decoded, _ := base64.StdEncoding.DecodeString(match[1])
		mapper = loadSourceMap(decoded)
	}

	jsLines := mapTSBreakpointsToJS(mapper, bps)
	if len(jsLines) > 0 {
		jsCode = instrumentCode(jsCode, jsLines)
	}

	runResult := e.executeWithTimeout(jsCode, mapper, timeoutMs)
	for i := range runResult.Breakpoints {
		if _, _, sl, _, ok := mapper.Source(runResult.Breakpoints[i].Line, 0); ok {
			runResult.Breakpoints[i].Line = sl
		}
	}
	return runResult
}

func (e *Engine) build(tsCode string, resolver ScriptResolver) api.BuildResult {
	return api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   tsCode,
			ResolveDir: "/",
			Loader:     api.LoaderTS,
		},
		Format:    api.FormatIIFE,
		Bundle:    true,
		Write:     false,
		Sourcemap: api.SourceMapInline,
		Plugins: []api.Plugin{{
			Name: "script-resolver",
			Setup: func(build api.PluginBuild) {
				build.OnResolve(api.OnResolveOptions{Filter: `.*`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					if args.Kind != api.ResolveJSImportStatement || resolver == nil {
						return api.OnResolveResult{}, nil
					}
					_, err := resolver(args.Path)
					if err != nil {
						return api.OnResolveResult{}, nil
					}
					return api.OnResolveResult{Path: args.Path, Namespace: "script"}, nil
				})
				build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "script"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					source, err := resolver(args.Path)
					if err != nil {
						return api.OnLoadResult{}, nil
					}
					return api.OnLoadResult{Contents: &source, Loader: api.LoaderTS}, nil
				})
			},
		}},
	})
}

func mapTSBreakpointsToJS(mapper *sm.Consumer, bps []BpLine) []int {
	if mapper == nil || len(bps) == 0 {
		return nil
	}
	var lines []int
	for _, bp := range bps {
		if !bp.Enabled {
			continue
		}
		// Find JS line that maps to this TS line
		jsLine := findJSLineForTS(mapper, bp.Line)
		if jsLine > 0 {
			lines = append(lines, jsLine)
		}
	}
	sort.Ints(lines)
	return lines
}

func findJSLineForTS(mapper *sm.Consumer, tsLine int) int {
	// find the exact or closest higher JS line that maps to this TS line
	for js := 1; js < 5000; js++ {
		_, _, sl, _, ok := mapper.Source(js, 0)
		if ok && (sl == tsLine || (tsLine > 1 && sl >= tsLine-1 && sl <= tsLine+1)) {
			return js
		}
	}
	return 0
}

func instrumentCode(jsCode string, jsLines []int) string {
	parts := strings.Split(jsCode, "\n")
	// Insert hooks from bottom up so line numbers don't shift
	sort.Sort(sort.Reverse(sort.IntSlice(jsLines)))
	for _, line := range jsLines {
		idx := line - 1
		if idx >= 0 && idx < len(parts) {
			parts[idx] = fmt.Sprintf("__dbg(%d);", line) + parts[idx]
		}
	}
	return strings.Join(parts, "\n")
}

func (e *Engine) executeWithTimeout(jsCode string, mapper *sm.Consumer, timeoutMs int64) RunResult {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	ch := make(chan RunResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				ch <- RunResult{Error: fmt.Sprintf("runtime panic: %v", r)}
			}
		}()
		ch <- executeJS(jsCode, mapper)
	}()

	select {
	case result := <-ch:
		return result
	case <-ctx.Done():
		return RunResult{Error: fmt.Sprintf("execution timed out after %dms", timeoutMs)}
	}
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

func loadSourceMap(data []byte) *sm.Consumer {
	consumer, err := sm.Parse("", data)
	if err != nil || consumer == nil {
		return nil
	}
	return consumer
}

func mapJSError(err error, mapper *sm.Consumer) string {
	msg := err.Error()
	if mapper == nil {
		return msg
	}
	qjsErr, ok := err.(*qjs.Error)
	if !ok {
		return msg
	}
	match := stackLineRE.FindStringSubmatch(qjsErr.Stack)
	if match == nil {
		return msg
	}
	jsLine, err := strconv.Atoi(match[1])
	if err != nil || jsLine < 1 {
		return msg
	}
	_, _, srcLine, _, ok := mapper.Source(jsLine, 0)
	if !ok {
		return msg
	}
	return fmt.Sprintf("%s\n    at TypeScript line %d", msg, srcLine)
}

func executeJS(jsCode string, mapper *sm.Consumer) RunResult {
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
	console.Set("warn", jsCtx.Function(func(ctx *qjs.Context, this qjs.Value, args []qjs.Value) qjs.Value {
		var parts []string
		for _, arg := range args {
			parts = append(parts, arg.String())
		}
		output.WriteString("[WARN] " + strings.Join(parts, " "))
		output.WriteString("\n")
		return ctx.Null()
	}))
	jsCtx.Globals().Set("console", console)

	jsCtx.Globals().Set("__dbg", jsCtx.Function(func(ctx *qjs.Context, this qjs.Value, args []qjs.Value) qjs.Value {
		line := "0"
		if len(args) > 0 {
			line = args[0].String()
		}
		// Capture stack trace
		stack := ""
		errObj, _ := ctx.Eval("new Error().stack", qjs.EVAL_GLOBAL)
		if !errObj.IsUndefined() && !errObj.IsException() {
			stack = errObj.String()
		}
		// Capture global variables
		globals, _ := ctx.Eval("(function(){var g=globalThis;var r={};try{Object.keys(g).slice(0,50).forEach(function(k){try{var v=g[k];if(typeof v!=='function')r[k]=String(v);}catch(e){}});}catch(e){}return JSON.stringify(r);})()", qjs.EVAL_GLOBAL)
		gStr := ""
		if !globals.IsUndefined() && !globals.IsException() {
			gStr = globals.String()
		}

		bpOutput.WriteString(fmt.Sprintf("__DBG_LINE__:%s\n", line))
		bpOutput.WriteString(fmt.Sprintf("__DBG_STACK__:%s\n", strings.ReplaceAll(stack, "\n", "\\n")))
		bpOutput.WriteString(fmt.Sprintf("__DBG_GLOBALS__:%s\n", gStr))
		return ctx.ThrowInternalError("__BP_STOP__")
	}))

	result, err := jsCtx.EvalFile(jsCode, qjs.EVAL_GLOBAL, "script.ts")
	if err != nil {
		if bpOutput.Len() > 0 {
			// Breakpoint stop: return captured data
			runResult := RunResult{}
			if output.Len() > 0 {
				runResult.Output = output.String()
			}
			runResult.Breakpoints = parseBpHits(bpOutput.String())
			return runResult
		}
		return RunResult{
			Output: output.String(),
			Error:  "JavaScript runtime error: " + mapJSError(err, mapper),
		}
	}
	defer result.Free()

	runResult := RunResult{}
	if output.Len() > 0 {
		runResult.Output = output.String()
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
