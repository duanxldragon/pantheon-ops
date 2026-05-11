# Platform 壳层 PR Checklist 片段

更新时间：2026-04-30

类型：Acceptance
归属层：platform
状态：Active

以下内容可直接复制到后续 `platform` 壳层 PR 描述中。

适用范围：

- `frontend/src/core/layout/index.tsx`
- `frontend/src/core/layout/index.css`
- 动态菜单渲染链路
- 菜单图标映射
- 顶部横版导航
- 左侧竖版导航
- 与导航直接相邻的品牌区、页签区、顶部栏布局

---

```md
## Platform Shell Checklist

- [ ] 本次改动属于 `platform` 壳层，而不是单个 `system/*` 页面内容微调
- [ ] 已附双模式验收文档链接
- [ ] 双模式验收文档基于 `docs/acceptances/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md`
- [ ] 已同时验证竖版展开、竖版折叠、横版主导航、横版弹出子菜单
- [ ] 已执行 `npm run build`
- [ ] 已补固定扫描结果摘要：旧右栏 / 原生浮层 / 静态 Modal API / 双模式链路
- [ ] 已更新矩阵状态为 `Target` 或 `Pending`
- [ ] 若存在 `Pending`，已附矩阵文档链接和挂账位置
- [ ] 本次提交未把“代码已修改，待验收”误写成“已收口”
```

---

## 推荐搭配

- PR 正文模板：`docs/acceptances/PLATFORM_SHELL_PR_TEMPLATE.md`
- PR 正文样例：`docs/archive/PLATFORM_SHELL_PR_SAMPLE_20260430_LAYOUT_UNIFICATION.md`
- 双模式验收模板：`docs/acceptances/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md`
- 双模式验收基准样例：`docs/archive/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_20260430_LAYOUT_UNIFICATION.md`

