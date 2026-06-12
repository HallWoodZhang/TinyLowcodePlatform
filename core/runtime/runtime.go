package runtime

import (
	"context"
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

type Runner interface {
	Run(tsCode string, timeoutMs int64) RunResult
}

type Engine struct{}

func (e *Engine) Run(tsCode string, timeoutMs int64) RunResult {
	transformResult := api.Transform(tsCode, api.TransformOptions{
		Loader:    api.LoaderTS,
		Format:    api.FormatIIFE,
		Sourcemap: api.SourceMapExternal,
	})
	if len(transformResult.Errors) > 0 {
		var errs []string
		for _, msg := range transformResult.Errors {
			errs = append(errs, msg.Text)
		}
		return RunResult{Error: "TypeScript compilation error:\n" + strings.Join(errs, "\n")}
	}

	jsCode := string(transformResult.Code)
	mapper := loadSourceMap(transformResult.Map)

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

var stackLineRE = regexp.MustCompile(`script\.ts:(\d+)`)

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
