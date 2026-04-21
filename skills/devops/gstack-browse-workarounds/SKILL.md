---
name: gstack-browse-workarounds
description: gstack-browse-workarounds — skill for devops category
category: devops
triggers:
- deploy
- CI/CD
- Cloudflare
- webhook
- monitoring
- Next.js QA
- gstack browse workarounds
related_skills:
- systematic-debugging
- test-driven-development
- gstack-browse
---
repo_tracked: true


# gstack browse Workarounds

## Chromium Sandbox 报错（root 用户）

**症状：**
```
[err] Running as root without --no-sandbox is not supported.
Browser logs: Chromium sandboxing failed!
```

**原因：** `browser-manager.ts` 通过 `process.env.CI` 或 `process.env.CONTAINER` 检测容器环境并自动加 `--no-sandbox`，但某些场景下这两个 env var 都没设。

**解决：** 每次调用 `browse` 命令前加 `CI=true`：
```bash
CI=true /root/.openclaw/skills/gstack-browse/bin/browse goto https://...
CI=true /root/.openclaw/skills/gstack-browse/bin/browse snapshot -i
```

## browse 命令是每次独立启动

- `browse` 每次调用都启动一个新的 server 进程，不是 daemon
- 每个命令都需要 `CI=true` 前缀
- 服务器启动需要 ~2-3 秒，`goto` 后建议 `sleep 5` 再 snapshot

## 认证处理

- Playwright session 在跨页面导航后会**丢失 auth 状态**
- 每个测试 session 开始时先完整登录一次
- 可用 `cookie-import-browser` 从已有浏览器导入 cookie 避免重复登录

## 常用 QA 命令模板

```bash
# 1. 启动 + 登录
CI=true $B goto https://vibex-app.pages.dev/auth/ && sleep 3
CI=true $B fill @e1 "y760283407@outlook.com"
CI=true $B fill @e2 "12345678"
CI=true $B click @e3 && sleep 5

# 2. 页面快照
CI=true $B snapshot -i

# 3. 查 console 错误
CI=true $B console --errors

# 4. 查 404 网络请求
CI=true $B network 2>&1 | grep "404"

# 5. 截图
CI=true $B screenshot /tmp/screen.png

# 6. reload 后等待
CI=true $B reload && sleep 5 && CI=true $B snapshot -i
```
## 点击超时问题（普遍模式）

**症状：** `click @eN` 对多种元素超时（5秒），包括：Tab 组件、抽屉内的按钮、工具栏 action 按钮。

**原因：** React/Vue 等 SPA 组件对 Playwright 标准 click 事件（包含 pointer/preventDefault 等）不响应，或元素被动画遮挡。

**通用解法：** 用 JS 直接调用 `click()`，永远不要信任 `$B click` 对 SPA 组件的可用性：

```bash
# 找按钮并点击（最安全）
$B js "
const btns = Array.from(document.querySelectorAll('button'));
const btn = btns.find(b => b.textContent.includes('确认所有节点'));
if(btn) btn.click();
else console.log('not found');
"

# Tab 切换
$B js "
const tabs = document.querySelectorAll('[role=tab]');
const tab = Array.from(tabs).find(t => t.textContent.includes('流程'));
if(tab) tab.click();
"

# 填表单后点击发送（同理）
$B js "
const btns = Array.from(document.querySelectorAll('button'));
const btn = btns.find(b => b.textContent.trim() === '发送' || b.textContent.includes('发送需求'));
if(btn) btn.click();
"
```

**注意：** URL 参数路由（如 `?tab=flow`）通常**不生效**。SPA 用 JS 状态而非 URL 控制 Tab，必须用 JS 点击。

**ref 在 JS 点击后仍可用** — `snapshot` 会实时重算 ref ID，但 JS 点击后要 `sleep 2-3` 等 React 渲染完成再 snapshot。

## Drawer/Modal 内容被 snapshot 遗漏

**症状：** `$B snapshot -i` 的 accessibility tree 不显示 drawer/modal 内的元素，但页面明显有内容。

**原因：** 某些 drawer 用 `aria-hidden` 或 CSS `display` 隐藏，accessibility tree 不收录。

**解决：** 用 `-C` 标志强制显示所有内容：

```bash
$B snapshot -C 2>&1 | grep -i "需求\|抽屉\|描述"
```

`-C` 会显示被隐藏层的内容，ref ID 也不同（带 @c 前缀的 cursor-pointer 元素），需要用 `$B js` 点击。

## Console buffer 不清除问题

**症状：** 访问 A 页面时有 console error，之后导航到 B 页面，console 里还有 A 的错误。

**原因：** Playwright console buffer 在页面导航后**不自动清除**。

**解决：** 每次导航到新页面后，先 `--clear` 再等 3 秒再查 `--errors`：
```bash
$B console --clear
$B goto https://...
sleep 3
$B console --errors
```

## 环境变量速查

| 用途 | Env Var |
|------|---------|
| 触发 no-sandbox | `CI=true` |
| 额外 Chromium 参数 | `BROWSE_EXTRA_ARGS="--proxy-server=socks5://..."` |
| Playwright 浏览器路径 | `PLAYWRIGHT_BROWSERS_PATH=~/.cache/ms-playwright` |
| browse server 脚本 | `BROWSE_SERVER_SCRIPT=/root/.openclaw/skills/gstack-browse/src/server.ts` |

---

_更新于 2026-04-16：Tab 点击超时 + JS workaround + console buffer 不清除问题发现_
