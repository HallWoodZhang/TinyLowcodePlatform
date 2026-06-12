# 前端编辑器选型

## 背景

原始前端使用普通 `<textarea>` 作为代码编辑区域，缺乏语法高亮和行号显示。

## 方案探索

### 第一版：自定义高亮叠加层

使用透明 `<textarea>` 叠加在 `<pre><code>` 高亮层之上，通过 JavaScript 实时解析代码并注入 `<span>` 标签实现语法着色。

**问题：**
- 两个 DOM 元素的字体渲染存在细微差异（浏览器对 textarea 和 pre 的字体度量不同）
- `<span>` 标签在高亮层中可见（文本选区穿透）
- 滚动同步存在延迟，视觉闪烁
- 维护成本高（正则表达式需覆盖 TypeScript/SQL 语法边缘情况）

### 第二版：纯文本编辑器

移除高亮层，仅保留 `<textarea>` + 左侧行号 gutter。

**问题：** 失去语法高亮功能，代码可读性差。

### 最终方案：CodeMirror 5

采用成熟的开源代码编辑器 [CodeMirror 5](https://codemirror.net/5/)，通过 CDN 加载。

**优势：**
- 零依赖安装（CDN `<script>` 标签即可）
- 内置行号、语法高亮、主题支持
- TypeScript 通过 JavaScript mode 获得关键字符号着色
- SQL mode 覆盖 80+ 关键字高亮
- 自动括号配对（`autoCloseBrackets`）
- 快捷键绑定（`Ctrl+S` 保存、`Ctrl+Enter` 执行）
- 选区感知（SQL editor 支持选中部分执行）

## 技术细节

| 项目 | 值 |
|---|---|
| 库 | CodeMirror 5.65.18 |
| CDN | cdnjs.cloudflare.com |
| TS 模式 | `mode: 'javascript'` |
| SQL 模式 | `mode: 'text/x-sql'` |
| 主题 | `material-darker`（Catppuccin Mocha 色系） |
| 字体 | JetBrains Mono / Fira Code / Cascadia Code |

## 对构建的影响

无。CDN 资源在浏览器端加载，`go:embed` 仅嵌入 `index.html`，二进制体积不受影响。离线环境需自行托管 CDN 文件。
