package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	qjs "github.com/quickjs-go/quickjs-go"
)

type RunResult struct {
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

func RunTSCode(tsCode string, timeoutMs int64) RunResult {
	transformResult := api.Transform(tsCode, api.TransformOptions{
		Loader: api.LoaderTS,
		Format: api.FormatCommonJS,
	})
	if len(transformResult.Errors) > 0 {
		var errs []string
		for _, e := range transformResult.Errors {
			errs = append(errs, e.Text)
		}
		return RunResult{Error: "TypeScript compilation error:\n" + strings.Join(errs, "\n")}
	}
	jsCode := string(transformResult.Code)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	ch := make(chan RunResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				ch <- RunResult{Error: fmt.Sprintf("runtime panic: %v", r)}
			}
		}()
		ch <- executeJS(jsCode)
	}()

	select {
	case result := <-ch:
		return result
	case <-ctx.Done():
		return RunResult{Error: fmt.Sprintf("execution timed out after %dms", timeoutMs)}
	}
}

func executeJS(jsCode string) RunResult {
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

	result, err := jsCtx.Eval(jsCode, qjs.EVAL_GLOBAL)
	if err != nil {
		return RunResult{
			Output: output.String(),
			Error:  "JavaScript runtime error: " + err.Error(),
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
