package runtime

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	sm "github.com/go-sourcemap/sourcemap"
	"github.com/evanw/esbuild/pkg/api"
	qjs "github.com/quickjs-go/quickjs-go"
)

type RunResult struct {
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

type ScriptResolver func(name string) (source string, err error)

type Runner interface {
	Run(tsCode string, resolver ScriptResolver, timeoutMs int64) RunResult
}

type Engine struct{}

var stackLineRE = regexp.MustCompile(`script\.ts:(\d+)`)
var inlineSourcemapRE = regexp.MustCompile(`//# sourceMappingURL=data:application/json;base64,([^\s]+)`)

func (e *Engine) Run(tsCode string, resolver ScriptResolver, timeoutMs int64) RunResult {
	var sourcemapBytes []byte

	result := api.Build(api.BuildOptions{
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
					if args.Kind != api.ResolveJSImportStatement {
						return api.OnResolveResult{}, nil
					}
					if resolver == nil {
						return api.OnResolveResult{}, nil
					}
					_, err := resolver(args.Path)
					if err != nil {
						return api.OnResolveResult{}, nil
					}
					return api.OnResolveResult{
						Path:      args.Path,
						Namespace: "script",
					}, nil
				})

				build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "script"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
					source, err := resolver(args.Path)
					if err != nil {
						return api.OnLoadResult{}, nil
					}
					return api.OnLoadResult{
						Contents: &source,
						Loader:   api.LoaderTS,
					}, nil
				})
			},
		}},
	})

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

	// Extract inline source map
	match := inlineSourcemapRE.FindStringSubmatch(jsCode)
	if match != nil {
		decoded, err := base64.StdEncoding.DecodeString(match[1])
		if err == nil {
			sourcemapBytes = decoded
		}
	}
	mapper := loadSourceMap(sourcemapBytes)

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

	result, err := jsCtx.EvalFile(jsCode, qjs.EVAL_GLOBAL, "script.ts")
	if err != nil {
		return RunResult{
			Output: output.String(),
			Error:  "JavaScript runtime error: " + mapJSError(err, mapper),
		}
	}
	defer result.Free()

	if output.Len() > 0 {
		return RunResult{Output: output.String()}
	}
	if result.IsUndefined() {
		return RunResult{Output: "undefined"}
	}
	return RunResult{Output: result.String()}
}
