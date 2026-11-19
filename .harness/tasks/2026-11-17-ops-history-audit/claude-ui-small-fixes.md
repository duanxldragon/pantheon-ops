低风险修复任务，工作目录 D:/workspace/go/pantheon-platform/pantheon-ops。使用用户现有默认模型，不启动子代理。你不是唯一执行者，不要还原/覆盖他人改动。先读 AGENTS.md、../pantheon-base/DESIGN.md、C:/Users/xiaolong/.codex/skills/impeccable/SKILL.md，并按现有 operational admin 控件风格保持 UI 行为和可访问性。

你仅拥有以下三个文件：frontend/src/modules/business/deploy/task/DeployTargetSelector.tsx、frontend/src/modules/business/shared/targetSelector.ts、frontend/tests/smoke/business/service/ServiceCatalog.nav.spec.ts；以及新建 .harness/evidence/2026-11-17-ops-history-audit/claude-ui-small-fixes.md。不要改其它代码、状态、lock、配置，不提交/发布。不运行命令（Codex 统一验证）。

当前 npm run build 实际失败：DeployTargetSelector.tsx 31/32 TARGET_SELECTION_BUDGET 未导入；targetSelector.ts 108 initialHostIds 未使用。这是刚提取 shared hook 的低级回归。补导入，并恢复旧选择器对 initialHostIds 的初始化兼容；显式 initial 的 scopeId 错配必须清空，不可将跨域旧意图保留。参考 git diff 的原文件不是必需，你可直接读取当前代码。保留 scope 切换清空、abort/generation 防旧响应、provider revalidation，不为消除 warning 直接删除兼容属性。

最后一份真实日志 .harness/evidence/2026-11-16-ops-navigation-repair/codex-navigation-spec-final.log：第三用例 sidebar click services 严格模式失败，因为 服务目录/服务 与 Kubernetes/服务 同名。修复测试 locator，按真实服务目录祖先或对应菜单路径/节点 id 限定；不要 first() 掩盖，不能缩减 30 项入口或取消真实点击/URL/tab/back/forward/reload 断言。可以读取前半段 walkEntry 的已有按 id 菜单定位并复用；不要造新 helper 大抽象。

只输出事实：修改文件、修复原因、未由自己运行的验证。渲染与 npm/typecheck/Playwright 由 Codex 执行，禁止自称已验证或改任务 completed。
