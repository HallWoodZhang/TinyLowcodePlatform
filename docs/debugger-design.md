# 断点调试系统

## 设计目标

在 TypeScript 源码行设置断点，执行时自动在断点处捕获位置信息，返回前端展示。

由于 QuickJS 无原生调试协议，采用**代码插桩 + 钩子函数捕获**方案：编译后在断点对应 JS 行注入 `__dbg(line)` 调用，执行时 QuickJS 全局函数捕获命中行号，再通过 sourcemap 映射回 TS 行号。

## 数据模型

新增 `breakpoints` 表：

```sql
CREATE TABLE breakpoints (
    script_id INTEGER NOT NULL,
    line      INTEGER NOT NULL,
    enabled   INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (script_id, line),
    FOREIGN KEY (script_id) REFERENCES scripts(id) ON DELETE CASCADE
);
```

- 复合主键 `(script_id, line)` 确保每行最多一个断点
- CASCADE 删除：脚本删除时自动清理其断点

## API

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/scripts/{id}/breakpoints` | 列出断点 |
| POST | `/api/scripts/{id}/breakpoints` | 设置/更新断点 `{line, enabled}` |
| DELETE | `/api/scripts/{id}/breakpoints/{line}` | 删除断点 |
| POST | `/api/scripts/{id}/debug` | 调试执行 |

## 执行流程

```
TS 源码 + 断点行号列表
        │
        ▼
  esbuild Build (bundle + sourcemap inline)
        │
        ├──► JS 代码
        └──► Sourcemap (base64 inline)
                │
                ▼
        mapTSBreakpointsToJS()
        ┌──────────────────┐
        │ for each TS line: │
        │   search mapper   │
        │   JS(1..5000,0)   │
        │   → TS line match │
        └──────────────────┘
                │
                ▼
        instrumentCode()
        ┌──────────────────────┐
        │ Split JS by \n       │
        │ Insert __dbg(line);  │
        │ from bottom up        │  ← 避免行号偏移
        └──────────────────────┘
                │
                ▼
        QuickJS EvalFile("script.ts")
        ┌──────────────────────┐
        │ Global.__dbg = fn    │
        │  → bpOutput += line  │
        └──────────────────────┘
                │
                ▼
        parseBpHits() → extract JS lines
                │
                ▼
        Sourcemap.Source(jsLine, 0) → TS lines
                │
                ▼
        RunResult.Breakpoints
```

## 关键实现

### TS → JS 行映射 (`findJSLineForTS`)

sourcemap 只提供 JS → TS 单向映射。逆向查找需遍历 JS 行：

```go
func findJSLineForTS(mapper, tsLine) int {
    for js := 1; js < 5000; js++ {
        _, _, sl, _, ok := mapper.Source(js, 0)
        if ok && sl == tsLine { return js }
    }
    return 0
}
```

上限 5000 行覆盖绝大多数场景。

### 插桩注入 (`instrumentCode`)

从下往上插入避免行号偏移：

```javascript
// 原始 JS (IIFE)
(() => {
  const a = 1;         // JS line 2
  const b = 2;         // JS line 3
})();

// 插桩后 (断点在 TS line 2 → JS line 3)
(() => {
  const a = 1;
  __dbg(3);const b = 2;  // ← 钩子注入在行首
})();
```

### 钩子捕获 (`__dbg`)

QuickJS 全局函数，将命中行号写入特定格式字符串：

```go
jsCtx.Globals().Set("__dbg", jsCtx.Function(func(...) {
    bpOutput.WriteString(fmt.Sprintf("__DBG__:%s;", arg.String()))
}))
```

执行后正则解析：`regexp.MustCompile("__DBG__:(\d+);")`

## 限制与后续优化

- **变量捕获**：当前只捕获行号，不捕获变量值。后续可通过 QuickJS 的 `JS_GetGlobalVar` / scope 遍历实现
- **单步执行**：需要 QuickJS 的 `JS_InterruptHandler` 或在每条语句间注入钩子
- **条件断点**：前端传入条件表达式，执行时 `eval(condition)` 判断是否命中
- **跨脚本断点**：导入的脚本也可设置断点，需将依赖脚本的 breakpoints 一并传入
