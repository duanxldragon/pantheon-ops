# i18n 模块设计

更新时间：2026-04-29

类型：Design
归属层：system/config
状态：Active

本文定义 `system/config -> i18n` 子域的独立边界。

它重点解决四个问题：

- `i18n` 在 Pantheon 中到底属于什么层级
- 运行时翻译资产、前端 fallback、后端 message key 各自由谁负责
- `i18n` 为什么不能只理解成“翻译 CRUD 页”
- 新增 locale、重命名 key、清理未使用 key 时，验收应该看什么

---

## 1. 模块定位

`i18n` 属于 `system/config` 子域。

它的职责不是“帮页面显示中文/英文”这么简单，而是：

- 维护运行时翻译资产
- 为前端和导入导出链路提供稳定语言包
- 承接 key 生命周期治理
- 为新模块接入提供 namespace 和翻译收口能力

它不是：

- 页面文案随手编辑器
- 业务模块可以随意绕过的附属页
- 只面向前端的工具页

---

## 2. 边界

### 2.1 `i18n` 负责

- 语言包读取
- 翻译项 CRUD
- 导入、导出、模板下载
- 缺失 locale 检测
- 缺失 key 同步
- builtin locale 资产回填
- key 重命名预览与执行
- 未使用 key 的观察、归档、删除

### 2.2 `i18n` 不负责

- 后端业务码设计，这属于错误语义设计
- 页面最终展示布局，这属于前端页面
- 菜单、权限、模块边界本身，这些属于 `system/iam` 或模块契约
- 业务模块自己的语义定义，业务模块只是在这里接入翻译资产

### 2.3 协作边界

- 后端返回稳定的 `message key`
- 前端负责 `t(message)` 与 fallback
- `i18n` 子域负责让 key 和翻译资产“可治理、可查、可迁移”

---

## 3. 运行时模型

## 3.1 资产来源

运行时翻译优先级：

1. `system_i18n` 中的远端翻译资产
2. 前端本地 `fallbackResources`
3. 通用兜底 key
4. 最后才裸显原始 key

## 3.2 当前 locale 基线

当前内置 fallback locale：

- `zh-CN`
- `en-US`
- `ja-JP`
- `ko-KR`
- `fr-FR`

原则：

- 不为了想象中的未来市场预扩 locale
- 新增 locale 必须由真实交付需求驱动

## 3.3 唯一性约束

`system_i18n` 运行时资产的唯一性应按：

- `locale + key`

而不是：

- `module + locale + key`

原因：

- `module` 用于归属和治理筛选
- `key + locale` 才是运行时真实查找维度

---

## 4. 页面与接口

## 4.1 前端页面

固定页面：

- `/system/i18n`

页面归属：

- `system/config`

当前页面权限：

- `system:i18n:list`

当前动作权限：

- `system:i18n:create`
- `system:i18n:update`
- `system:i18n:delete`
- `system:i18n:export`
- `system:i18n:import`
- `system:i18n:refresh`

## 4.2 后端接口

公共读取：

- `GET /api/v1/system/i18n/pack`

管理治理：

- `GET /api/v1/system/i18n/list`
- `GET /api/v1/system/i18n/:id`
- `POST /api/v1/system/i18n`
- `PUT /api/v1/system/i18n/:id`
- `DELETE /api/v1/system/i18n/:id`
- `POST /api/v1/system/i18n/batch-delete`
- `POST /api/v1/system/i18n/export`
- `POST /api/v1/system/i18n/import`
- `POST /api/v1/system/i18n/cache/refresh`
- `POST /api/v1/system/i18n/sync-keys`
- `GET /api/v1/system/i18n/overview`
- `GET /api/v1/system/i18n/audit`
- `GET /api/v1/system/i18n/missing-locales`
- `POST /api/v1/system/i18n/rename/preview`
- `POST /api/v1/system/i18n/rename`
- `POST /api/v1/system/i18n/fill-missing-locales`
- `POST /api/v1/system/i18n/hydrate-builtin-locales`
- `POST /api/v1/system/i18n/cleanup-unused`
- `POST /api/v1/system/i18n/lifecycle/observe`
- `POST /api/v1/system/i18n/lifecycle/archive`
- `POST /api/v1/system/i18n/lifecycle/delete`

---

## 5. 生命周期治理

`i18n` 的难点不是新增一条翻译，而是“如何不让资产越积越乱”。

当前生命周期应至少分成四步：

1. 发现缺失
2. 补齐缺失
3. 观察未使用
4. 归档或删除

### 5.1 缺失治理

包括：

- 缺失 key 同步
- 缺失 locale 检测
- 占位补齐

目标：

- 阻断“模块已接入，但某个 locale 全是空白”

### 5.2 重命名治理

包括：

- rename preview
- rename execute
- 迁移报告

目标：

- 避免 key 重构靠手改碰运气

### 5.3 未使用治理

包括：

- observe
- archive
- delete

目标：

- 让清理动作有缓冲，不直接删除潜在仍在使用的 key

---

## 6. 新增 locale 的准入规则

新增 locale 必须同时满足：

- 有明确市场、客户或交付需求
- 有翻译责任归属
- 有后续维护人

新增后至少完成：

1. 本地 fallback 资源补齐
2. `system_i18n` 运行时资产导入
3. `check:i18n-hardcode`
4. `audit:i18n-locales`
5. `npm run build`
6. 页面与导入导出链路回归

---

## 7. 与错误码设计的关系

`i18n` 不能脱离错误语义设计单独存在。

配合关系：

- 后端负责稳定 `message key`
- 前端负责翻译和展示
- `i18n` 负责让这些 key 有稳定资产可查

因此：

- 后端不应返回自然语言
- 前端不应直接裸显 key
- `i18n` 管理端也不应把“翻译资产”误当成“业务规则定义”

---

## 8. 风险点

### 8.1 常见风险

- 同义不同 key
- 同 key 多模块重复
- locale 集合不一致
- fallback 退回英文硬编码
- 导入后部分成功但没有明确报告
- key 重命名后前后端不一致

### 8.2 质量门禁

至少保留：

- `npm run check:i18n-hardcode`
- `npm run audit:i18n-locales`
- `npm run build`

如果影响导入导出链路，还应补：

- 错误 CSV 列头语言校验
- 结果摘要语言切换校验
- 重命名报告语言校验

---

## 9. 验收要求

后续 `i18n` 验收至少看五层：

1. `/system/i18n` 页面可打开
2. 页面权限和动作权限正确
3. 远端语言包能正确加载
4. 生命周期治理链路有明确结果
5. fallback 不回退到硬编码英文提示

固定检查项：

- `/system/i18n`
- `GET /system/i18n/pack`
- 导入
- 导出
- 刷新缓存
- rename preview / rename
- missing locale 检测
- unused lifecycle

---

## 10. 当前结论

`i18n` 已经是 `system/config` 下的独立治理子域。

后续必须坚持：

- 把它当成运行时资产治理系统，而不是简单翻译录入页
- 把 locale、key、fallback、生命周期和验收门禁放在同一套规则里

否则新增模块越多，翻译资产就越容易失控。
