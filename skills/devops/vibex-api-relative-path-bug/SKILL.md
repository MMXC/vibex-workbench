---
name: vibex-api-relative-path-bug
description: vibex-api-relative-path-bug — skill for devops category
category: devops
triggers:
- deploy
- CI/CD
- Cloudflare
- webhook
- monitoring
- Next.js QA
- vibex api relative path bug
related_skills:
- systematic-debugging
- test-driven-development
- gstack-browse
---
repo_tracked: true


# VibeX API 相对路径 404 Bug

## 触发场景

前端 API 调用使用**相对路径**（`/api/v1/...`），而页面部署在 `vibex-app.pages.dev`（Next.js 静态导出模式）。浏览器将相对路径拼到当前域名，导致请求打到错误的地址。

## 症状

- API 返回 404，响应体是 Next.js 404 HTML 页面
- `_redirects` 文件存在但对 `/api/*` 规则不生效
- 实际后端 `api.vibex.top` 返回 200/401（正常）
- 不同 API 有不同的成功率（有些碰巧在 pages.dev 上有同名静态文件）

## 根因

代码中使用了默认参数空字符串：

```typescript
// ❌ 错误：baseUrl 默认为 ''，导致相对路径
function createDDSAPI(baseUrl = '') {
  const base = `${baseUrl}/api/v1/dds`; // → '/api/v1/dds'
}
return useCallback(() => createDDSAPI(), [])() // 没传 baseUrl
```

浏览器解析相对路径 `GET /api/v1/dds/chapters`：
- 页面在 `https://vibex-app.pages.dev/design/dds-canvas/`
- 浏览器拼接 → `https://vibex-app.pages.dev/api/v1/dds/chapters` ❌
- 应该是 → `https://api.vibex.top/api/v1/dds/chapters` ✅

## 为什么 _redirects 不生效

Next.js `output: 'export'` 模式下，Cloudflare Pages 会优先让 Next.js 自己的静态文件服务处理请求：
- 有同名静态文件 → 返回该文件（`/api/quality/metrics` 有同名文件所以通）
- 无同名文件 → Next.js 404，永远不读 `_redirects`

## 修复

引入统一的 API 配置，用绝对路径：

```typescript
import { API_CONFIG } from '@/lib/api-config';

return useCallback(() => createDDSAPI(API_CONFIG.baseURL), [])() as DDSAPIClient;
//                                         ^^^^^^^^^^^^^^^^^^^^^^^^
//                                         'https://api.vibex.top/api'
```

## 验证

修复后检查 Network 面板，所有 API 请求都应该是 `api.vibex.top` 而非 `vibex-app.pages.dev`。

## 预防

所有 API client factory 函数必须要求传入绝对 baseUrl，或者直接依赖 `API_CONFIG`，禁止使用相对路径。
