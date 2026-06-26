package runtime

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	sm "github.com/go-sourcemap/sourcemap"
	"github.com/evanw/esbuild/pkg/api"
)

var inlineSourcemapRE = regexp.MustCompile(`//# sourceMappingURL=data:application/json;base64,([^\s]+)`)
var gojaErrLineRE = regexp.MustCompile(`<eval>:(\d+):(\d+)`)

func buildJS(tsCode string, resolver ScriptResolver) (string, *sm.Consumer, error) {
	buildResult := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   tsCode,
			ResolveDir: "/",
			Loader:     api.LoaderTS,
		},
		Format:    api.FormatIIFE,
		Bundle:    true,
		Write:     false,
		Sourcemap: api.SourceMapInline,
		Plugins:   buildResolverPlugin(resolver),
	})

	if len(buildResult.Errors) > 0 {
		var errs []string
		for _, msg := range buildResult.Errors {
			errs = append(errs, msg.Text)
		}
		return "", nil, fmt.Errorf("Build error:\n%s", strings.Join(errs, "\n"))
	}

	if len(buildResult.OutputFiles) == 0 {
		return "", nil, fmt.Errorf("Build produced no output")
	}

	jsCode := string(buildResult.OutputFiles[0].Contents)
	mapper := extractSourceMap(jsCode)
	return jsCode, mapper, nil
}

func buildResolverPlugin(resolver ScriptResolver) []api.Plugin {
	return []api.Plugin{{
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
	}}
}

func extractSourceMap(jsCode string) *sm.Consumer {
	match := inlineSourcemapRE.FindStringSubmatch(jsCode)
	if match == nil {
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(match[1])
	if err != nil {
		return nil
	}
	mapper, _ := sm.Parse("", decoded)
	return mapper
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
		jsLine := findJSLineForTS(mapper, bp.Line)
		if jsLine > 0 {
			lines = append(lines, jsLine)
		}
	}
	sort.Ints(lines)
	return lines
}

func findJSLineForTS(mapper *sm.Consumer, tsLine int) int {
	var candidates []int
	for js := 1; js < 5000; js++ {
		_, _, sl, _, ok := mapper.Source(js, 0)
		if ok && (sl == tsLine || (tsLine > 1 && sl >= tsLine-1 && sl <= tsLine+1)) {
			candidates = append(candidates, js)
		}
	}
	return pickBestLine(candidates)
}

func pickBestLine(jsLines []int) int {
	if len(jsLines) == 0 {
		return 0
	}
	if len(jsLines) == 1 {
		return jsLines[0]
	}
	// Return the line closest to the median — avoids edge cases (comments/wrapper lines)
	return jsLines[len(jsLines)/2]
}

func instrumentCode(jsCode string, jsLines []int, lineVars map[int][]string) string {
	parts := strings.Split(jsCode, "\n")
	sort.Sort(sort.Reverse(sort.IntSlice(jsLines)))
	for _, line := range jsLines {
		idx := line - 1
		if idx >= 0 && idx < len(parts) {
			vars := lineVars[line]
			if len(vars) > 0 {
				varNames := make([]string, len(vars))
				for i, v := range vars {
					varNames[i] = fmt.Sprintf("%s:%s", v, v)
				}
				parts[idx] = fmt.Sprintf("__dbg(%d,{%s});", line, strings.Join(varNames, ",")) + parts[idx]
			} else {
				parts[idx] = fmt.Sprintf("__dbg(%d);", line) + parts[idx]
			}
		}
	}
	return strings.Join(parts, "\n")
}

func mapErrorLine(errMsg string, mapper *sm.Consumer) (string, int, int) {
	if mapper == nil {
		return errMsg, 0, 0
	}

	matches := gojaErrLineRE.FindStringSubmatch(errMsg)
	if matches == nil {
		lines := strings.SplitN(errMsg, "\n", 2)
		if len(lines) <= 1 {
			return errMsg, 0, 0
		}
		// Goja errors have "at <eval>:line:col" in the second line
		matches = gojaErrLineRE.FindStringSubmatch(lines[1])
		if matches == nil {
			return errMsg, 0, 0
		}
	}

	jsLine, _ := strconv.Atoi(matches[1])
	_, _, tsLine, _, ok := mapper.Source(jsLine, 0)
	if ok {
		return fmt.Sprintf("%s\n  [TS line %d, JS line %d]", errMsg, tsLine, jsLine), jsLine, tsLine
	}
	return fmt.Sprintf("%s\n  [JS line %d]", errMsg, jsLine), jsLine, 0
}
