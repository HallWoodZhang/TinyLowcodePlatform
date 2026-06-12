# BugFix #001: QuickJS 执行 TypeScript 报 module is not defined

## 版本

v0.x (2026-06-13)

## 现象

调用 `POST /api/scripts/{id}/run` 执行 TypeScript 脚本时返回错误：

```json
{"error": "JavaScript runtime error: ReferenceError: 'module' is not defined"}
```

无论何种 TS 代码均无法执行。

## 根因

`core/runtime/runtime.go` 中使用 esbuild 将 TypeScript 编译为 JavaScript 时，输出格式指定为 `api.FormatCommonJS`：

```go
transformResult := api.Transform(tsCode, api.TransformOptions{
    Loader: api.LoaderTS,
    Format: api.FormatCommonJS, // ← 问题所在
})
```

CommonJS 格式会生成如下包装代码：

```javascript
var __commonJS = (cb, mod) => function __require() {
    return mod || (0, cb[Object.keys(cb)[0]])((mod = { exports: {} }).exports, mod), mod.exports
};
```

其中引用了 `module` 和 `exports` 全局对象，这些对象仅在 Node.js / 浏览器 bundler 环境中存在。

本项目使用 [QuickJS](https://bellard.org/quickjs/) 作为 JS 运行时，QuickJS 是一个轻量级嵌入式引擎，**不提供** CommonJS 模块系统，`module` / `exports` 均未定义，因此执行时抛出 `ReferenceError`。

## 修复

将 esbuild 输出格式从 `api.FormatCommonJS` 改为 `api.FormatIIFE`：

```go
Format: api.FormatIIFE, // IIFE 自执行函数，无模块依赖
```

IIFE 格式将代码包装为立即执行函数表达式：

```javascript
(() => {
    console.log(1 + 2);
})();
```

不依赖任何外部模块系统，与 QuickJS 兼容。

## 影响范围

- 仅影响 `core/runtime/runtime.go`
- 对简单脚本（无 import/export）完全透明
- 使用 `import` / `export` 的脚本不适用（esbuild 会报错），后续可考虑 bundling 支持

## 验证

```bash
curl -X POST http://127.0.0.1:9720/api/scripts \
  -H 'Content-Type: application/json' \
  -d '{"name":"test","label":"T","type":"ts","tsCode":"console.log(1+2);"}'

curl -X POST http://127.0.0.1:9720/api/scripts/{id}/run
# → {"output":"3\n"}
```

## 相关文件

| 文件 | 变更 |
|---|---|
| `core/runtime/runtime.go:27` | `FormatCommonJS` → `FormatIIFE` |
| `docs/bugfix-001-quickjs-commonjs-module.md` | 本文档 |
