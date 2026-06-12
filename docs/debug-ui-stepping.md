# Debug UI & Stepping 设计

## 当前状态

断点调试已实现基本功能：设置/删除断点，执行时在断点行注入钩子捕获堆栈和全局变量，结果拼在 Output 文本区展示。

## UI 设计

### Debug 面板布局

```
┌────────────────────────────────────────────┐
│ Tab Bar                                     │
├────────────────────────────────────────────┤
│ [name] [label] [type]   [Save] [Run] [Debug]│
├────────────────────────────────────────────┤
│                                             │
│         CodeMirror Editor                   │
│                                             │
├────────────────────────────────────────────┤
│ Output / Debug              [Output][Debug] │
├────────────────────────────────────────────┤
│ ▶ Program output                            │
│   A -> C                                    │
│   A -> B                                    │
│                                             │
│ ● Breakpoint hit — line 11                  │
│   📋 Stack:                                  │
│     at <anonymous> (script.ts:12)           │
│     at <eval> (script.ts:13)               │
│   📦 Globals:                                │
│     console: [object Object]                │
│     Math: [object Math]                      │
├────────────────────────────────────────────┤
│ [▶ Continue] [⤵ Step Over] [⤵ Step Into]   │
│ [⤴ Step Out]                                │
└────────────────────────────────────────────┘
```

### 按钮位置

在 Output 区域上方添加 Debug 工具栏，包含：
- Run / Debug 切换（顶部已有）
- 调试控制：Continue / Step Over / Step Into / Step Out
- 断点结果在 Output 区域以结构化面板展示（非纯文本）

### 数据流

```
┌──────────────┐    POST /debug    ┌──────────────┐
│    Frontend   │ ───────────────► │   Backend     │
│              │                   │               │
│ set BP lines │                   │ esbuild build │
│ gutter click │                   │ instrument    │
│              │ ◄─────────────── │ QuickJS eval  │
│              │   {breakpoints:  │ sourcemap map │
│              │    [{line, stack,│               │
│              │      vars}]}     │               │
└──────────────┘                   └──────────────┘
```

## Stepping 实现方案

### 方案 A：无状态模拟（当前可做）

Step Over 通过设置临时下一行断点实现：
1. 用户点击 Step Over
2. 后端根据上次执行结果，找到当前函数的下一行
3. 设置临时断点（不持久化到 DB）
4. 重新执行，在下一行暂停

**限制：** 无法处理循环/分支中的多路径，每步需重新执行整个脚本。

### 方案 B：有状态 Debug Session（推荐，复杂度高）

1. 创建 `POST /api/scripts/{id}/session` — 启动调试会话
2. 服务端保持 QuickJS Runtime 存活（goroutine 挂起）
3. 前端通过 WebSocket 或轮询与会话通信
4. 命令：step/continue/stop/getvars
5. QuickJS 通过 `JS_SetInterruptHandler` 实现逐行中断

**状态机：**
```
RUNNING → (breakpoint hit) → PAUSED
PAUSED → step → RUNNING → (next line) → PAUSED
PAUSED → continue → RUNNING → (next BP) → PAUSED
PAUSED → stop → TERMINATED
```

**存储：** 会话状态（script_id, runtime, context, breakpoints）保存在内存 map 中，超时自动销毁。

### 方案 C：esbuild 级步进（中等复杂度）

编译时在每条语句间注入 `__dbg_step()` 钩子，执行时：
1. 每行执行前调用 `__dbg_step`
2. 检查是否在下一条用户可见语句上
3. 满足条件时截断执行并返回状态

**优势：** 不需要保持 Runtime 存活，每次请求执行到下一个断点或步进位置。
**劣势：** 每次步进都是完整重新执行，性能差（Hanoi 递归等场景）。

## 推荐路径

1. **第一阶段（本周）：** 完成 UI 面板重构，结构化展示堆栈/变量
2. **第二阶段：** 实现方案 C（esbuild 级无状态步进），每次重新执行到下一个步进点
3. **第三阶段：** 实现方案 B（有状态会话），真正的断点暂停和步进

## 前端 UI 待办

- [ ] Debug 面板分离（Output tab / Debug tab）
- [ ] 堆栈树状展示（文件名 + 行号）
- [ ] 变量表格式展示（name: type: value）
- [ ] Step 按钮（Continue / Step Over / Step Into / Step Out）
- [ ] 断点命中高亮行（CodeMirror 当前执行行标记）
