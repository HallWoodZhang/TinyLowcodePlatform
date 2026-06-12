# TypeScript Sourcemap 错误映射

## 背景

QuickJS 执行编译后的 JavaScript 代码时，运行时错误只包含 JS 行号（如 `script.ts:2`），对 TypeScript 开发者极不友好。需要将 JS 行号映射回原始 TS 行号。

## 方案

```
TypeScript 源码
       │
       ▼
  esbuild Transform
  (Loader: TS, Format: IIFE, Sourcemap: External)
       │
       ├──► JS 代码 ──► QuickJS EvalFile("script.ts")
       │                        │
       │                        ▼
       │               JS 运行时错误 + Stack 信息
       │                        │
       │                        ▼
       │               parse  Stack 提取 JS 行号
       │                        │
       ▼                        ▼
   Sourcemap JSON ──► go-sourcemap.Source(jsLine, 0)
                              │
                              ▼
                     原始 TS 行号 → 拼入 Error 消息
```

## 关键组件

### 1. esbuild 配置

```go
api.Transform(tsCode, api.TransformOptions{
    Loader:    api.LoaderTS,
    Format:    api.FormatIIFE,
    Sourcemap: api.SourceMapExternal,  // 生成标准 VLQ sourcemap JSON
})
```

- `FormatIIFE`：不依赖 CommonJS，QuickJS 兼容
- `SourceMapExternal`：sourcemap JSON 单独输出在 `transformResult.Map`

### 2. QuickJS 错误元信息

```go
result, err := jsCtx.EvalFile(jsCode, qjs.EVAL_GLOBAL, "script.ts")
```

- 使用 `EvalFile` 并传入文件名 `"script.ts"`，错误 Stack 中会包含文件名和行号
- 返回的 `err` 可类型断言为 `*qjs.Error`，包含 `Stack` 字段：

```
Stack: "    at <anonymous> (script.ts:2)\n    at <eval> (script.ts:3)\n"
```

### 3. Stack 解析

正则提取第一个 `script.ts:N` 中的行号：

```go
var stackLineRE = regexp.MustCompile(`script\.ts:(\d+)`)
match := stackLineRE.FindStringSubmatch(qjsErr.Stack)
jsLine, _ := strconv.Atoi(match[1])
```

### 4. Sourcemap 映射

```go
import sm "github.com/go-sourcemap/sourcemap"

mapper, _ := sm.Parse("", transformResult.Map)
_, _, srcLine, _, ok := mapper.Source(jsLine, 0)
if ok {
    // srcLine 就是 TS 原始行号（1-indexed）
}
```

> 注意：`go-sourcemap` 库内部使用 1-indexed 行号，因此 stack 提取的行号无需 -1 转换，直接传入即可。

### 5. 最终输出

```go
return fmt.Sprintf("%s\n    at TypeScript line %d", msg, srcLine)
```

## 依赖

| 库 | 用途 |
|---|---|
| `github.com/evanw/esbuild` | TS 编译 + sourcemap 生成 |
| `github.com/go-sourcemap/sourcemap` | VLQ sourcemap 解析 |
| `github.com/quickjs-go/quickjs-go` | JS 运行时 + 错误 Stack |

## 已知限制

- esbuild IIFE 格式会删除注释（只保留实际代码），但 sourcemap 行映射保持正确
- 多行表达式的映射精度为行级（列级映射预留但未使用）
- QuickJS 的 Stack 信息依赖 `EvalFile` 传入的文件名，必须使用带文件名的 API
