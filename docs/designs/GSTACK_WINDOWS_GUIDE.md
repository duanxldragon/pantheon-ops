# gstack Windows 使用清单

更新时间：2026-04-20

类型：Design
归属层：platform
状态：Active

本文只描述 Pantheon 本地开发场景下，如何在 Windows 上稳定使用 gstack 内置浏览器完成冒烟或页面巡检。

## 1. 适用范围

- 平台层 / 系统域页面冒烟
- 本地前后端联调
- 需要使用 gstack 内置 Chrome 采集截图、console error、页面快照

默认约定：

- Pantheon 本地浏览器页面链路测试默认使用 gstack。
- 组织架构、访问控制、按钮权限、无权限态、截图留证这类真实页面验收默认走 `browse chain` 或 gstack Browser。
- Playwright 仅作为 CI/API smoke 或用户明确要求时的补充工具，不作为 Windows 本地验收默认入口。

## 2. 当前环境结论

在当前 Windows 环境中，**不需要额外 setup 才能使用 gstack 完成本地冒烟**。

已确认可直接使用的能力：

- `browse`
- `qa-only`
- `open-gstack-browser`
- `setup-browser-cookies`（按需）

## 3. 已确认路径

- gstack 仓库：`C:\Users\xiaolong\.gstack\repos\gstack`
- 浏览器可执行文件：`C:\Users\xiaolong\.gstack\repos\gstack\browse\dist\browse.exe`

## 4. 推荐使用方式

### 4.1 优先使用单条 `browse chain`

Windows 下多次拆开的 browse 调用更容易出现上下文漂移。
建议把“打开登录页 → 注入 token → 跳转目标页 → 等待 → 截图 / snapshot”放在一条 `chain` 中。

推荐模式：

```powershell
browse chain "goto http://127.0.0.1:5173/login | wait --networkidle 15000 | storage set pantheon_access_token <AT> | storage set pantheon_refresh_token <RT> | goto http://127.0.0.1:5173/dashboard | wait --networkidle 15000 | console --errors | screenshot out.png | snapshot -i"
```

### 4.2 先通过 API 登录，再注入 token

对于后台系统页，本地场景下最稳定的做法不是手动输入表单，而是：

1. 调 `POST /api/v1/auth/login`
2. 取 `accessToken` / `refreshToken`
3. 注入到 localStorage
4. 再跳到目标页

## 5. 常见问题

### 5.1 `spawn EPERM`

现象：

- `browse.exe` 启动时报 `spawn EPERM`

处理建议：

- 在 Windows 上允许提权执行 gstack 浏览器
- 一旦批准后，后续相同前缀命令可复用授权

### 5.2 `No active page`

现象：

- `wait`、`console`、`snapshot` 报 `No active page`

根因：

- 浏览器上下文在前一步中被关闭或漂移

处理建议：

- 重新执行整条 `browse chain`
- 避免把登录、storage、页面跳转拆成多次独立调用

### 5.3 截图超时

现象：

- `locator.screenshot: Timeout 5000ms exceeded`

处理建议：

- 先看 `url` 与 `console --errors` 是否已正常
- 如果页面本身正常，可单独补跑一次 `screenshot`

## 6. 哪些 setup 是按需的

### 6.1 `setup-browser-cookies`

仅在以下场景需要：

- 你想复用本机真实 Chromium/Chrome 登录态
- 不想通过 API 注入 token

本地后台冒烟不是必需步骤。

### 6.2 `open-gstack-browser`

仅在以下场景需要：

- 你想看到可视化浏览器窗口
- 你需要边看边操作或向他人演示

纯脚本化 QA / smoke 不是必需步骤。

## 7. Pantheon 本地冒烟建议顺序

1. 确认后端 `:8080` 在线
2. 确认前端 `:5173` 在线
3. 调登录接口拿 token
4. 用单条 `browse chain` 跑目标页
5. 同步保存原始输出、截图和汇总 JSON

## 8. 结论

- 当前 Pantheon 的 Windows 环境已经具备 gstack 冒烟能力
- 无需额外安装新的浏览器集成才能完成本地系统页巡检
- 重点不是“再做 setup”，而是遵循稳定执行方式：**单条 chain、必要时提权、尽量用 API 注入登录态**
