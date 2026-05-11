# 错误码与多语言设计

更新时间：2026-04-28

类型：Design
归属层：platform
状态：Active

本文定义 Pantheon Base 的错误码与 i18n 责任边界。

目标是把下面这些问题一次讲清：

- 后端返回的是自然语言，还是错误 key？
- 前端 toast 应该直接显示 message，还是先翻译？
- 成功提示和失败提示分别由谁负责？
- fallback 规则是什么？
- 新增模块时，错误文案和 i18n key 由谁补？

如果这份文档不先定清楚，后续最容易变成：

- 后端有时返回中文，有时返回英文，有时返回 key
- 前端有时 `t(message)`，有时直接 `Message.error(message)`
- 成功文案和错误文案分散在几十个页面里
- 新模块只写接口，不补翻译 key
- 不同模块出现同义不同 key 的错误文案

## 1. 设计目标

- **后端只负责稳定的错误语义**
- **前端只负责最终展示语言**
- **错误 key 与 i18n key 统一**
- **成功/失败提示策略一致**
- **支持 fallback，不因缺失翻译直接崩 UI**

## 2. 统一响应原则

当前统一响应结构：

```go
type Response struct {
    Code    int
    Data    interface{}
    Message string
}
```

其中：

- `Code`：业务码
- `Data`：业务数据
- `Message`：**必须优先视为 i18n key**

## 3. 后端返回规则

## 3.1 后端不返回自然语言

后端默认不返回：

- 中文提示
- 英文提示
- 面向终端用户的完整展示文案

后端应该返回：

- 稳定错误 key
- 稳定成功 key（仅在确实需要时）

示例：

```text
param.invalid
permission.denied
user.login.error.not_found
user.role.required
refresh_token.invalid
```

## 3.2 后端错误分三层

| 层级 | 说明 | 示例 |
| :--- | :--- | :--- |
| `platform` | 基础设施错误 | `database.not_initialized` |
| `domain` | 业务规则错误 | `user.role.required` |
| `security` | 认证/授权错误 | `permission.denied` |

## 3.3 后端业务码规则

当前业务码：

| Code | 说明 |
| :--- | :--- |
| `200` | 成功 |
| `400` | 参数错误 |
| `401` | 未认证 / 认证失效 |
| `403` | 无权限 |
| `500` | 通用失败 |

短期允许继续使用这组码。

后续如果引入更细错误码体系，也不能破坏“`Message` 是 key”这一原则。

## 4. 前端展示规则

## 4.1 前端必须翻译 message

前端收到：

```json
{
  "code": 403,
  "message": "permission.denied"
}
```

默认展示应为：

```ts
t('permission.denied')
```

而不是直接显示：

```ts
"permission.denied"
```

## 4.2 当前运行时行为

当前请求层已经按“`message` 优先视为 key”收口，前端不再以内置英文自然语言作为默认提示。

当前规则：

- 如果后端返回的 `message` 命中 i18n key，则直接翻译展示
- 如果后端返回的是非 key 的原始传输文本，则统一回退到前端基础 key，而不是直接把英文 fallback 暴露给终端用户
- 网络异常、超时、浏览器层异常统一落到稳定的前端 key
- 开发环境允许在 fallback 文案后追加原始错误文本，用于定位联调问题；生产环境不直接暴露这类原始 transport message

## 4.3 正确行为

请求层应遵循：

1. 如果 `message` 是 key，则走 `t(message)`
2. 如果 key 缺失，则走 fallback
3. 如果是网络异常或非标准错误，再走默认前端 key

后端 handler 层应配套使用统一错误出口：

- 如果 Service 返回的是稳定 key，则原样返回；
- 如果拿到的是框架错误、数据库错误或其他非 key 文本，则不能直接透传给前端；
- 建议统一通过类似 `common.FailWithError(...)` 的出口，把非 key 错误收敛到稳定 fallback key。

## 5. 成功提示规则

## 5.1 成功提示优先由前端控制

成功提示不建议由后端自由返回。

原因：

- 前端最知道当前页面上下文
- 成功提示文案经常是页面级表达
- 后端成功提示很容易泛化成一堆重复 message

推荐：

- 新增成功：前端统一使用 `common.createSuccess`
- 更新成功：前端统一使用 `common.updateSuccess`
- 删除成功：前端统一使用 `common.deleteSuccess`

## 5.2 后端成功 message

后端成功返回里的 `message: "success"` 可以保留，但前端通常不直接拿它弹 toast。

## 6. i18n key 命名规则

## 6.1 通用 key

用于全局公共文案：

- `common.*`
- `auth.*`
- `permission.*`

## 6.2 模块级 key

用于模块页面、字段、业务规则：

- `system.user.*`
- `system.role.*`
- `system.menu.*`
- `system.permission.*`
- `auth.session.*`
- `biz.order.*`

补充约束：

- `system_i18n` 中的运行时翻译资产，**唯一性应按 `locale + key` 约束**，不能允许不同 `module` 各自保存同一个 key 的副本。
- `module` 用于资产归属、治理筛选和导出边界，不是运行时唯一键的一部分。
- 如果同一个 `key + locale` 在多个模块下重复出现，属于 `system/config` 国际化治理问题，必须收敛为单条 canonical 记录，而不是依赖缓存覆盖顺序“碰运气”生效。

## 6.3 错误 key

错误 key 推荐结构：

```text
{module}.{action}.error.{reason}
```

示例：

```text
user.login.error.not_found
user.login.error.disabled
user.login.error.password_wrong
user.update.error.protected
menu.delete.error.has_children
post.dept.required
post.dept.invalid
post.dept.root_forbidden
user.post.dept_required
user.post.dept_mismatch
```

### 6.3.1 通用错误 key

以下作为基础共用：

- `success`
- `param.invalid`
- `permission.denied`
- `permission.engine.not_initialized`
- `database.not_initialized`
- `network.error`
- `request.failed`

## 7. fallback 规则

## 7.1 前端翻译 fallback

优先级：

1. 远端语言包（`system_i18n` 运行时资产）
2. 本地 `fallbackResources`
3. 通用兜底 key
4. 最后才显示原始 key

当前内置 fallback locale：

- `zh-CN`
- `en-US`
- `ja-JP`
- `ko-KR`
- `fr-FR`

locale 资源治理门禁：

- 前端源码防回退到硬编码：`npm run check:i18n-hardcode`
- locale 完整性审计：`npm run audit:i18n-locales`
- 构建联调验证：`npm run build`

其中 `audit:i18n-locales` 以 `zh-CN` 为基准语言包、`en-US` 为参考语言包，检查：

- `missing`：目标 locale 缺失的 key
- `extra`：目标 locale 多出的 key
- `empty`：值为空或只含空白的 key
- `sameAsEn`：与英文值完全相同、需要人工复核是否未翻译的 key

新增语言、批量导入翻译、重命名 key、收口菜单/页头/导入导出等展示链路后，以上三条门禁必须一起通过，才能视为 `system/config` 国际化治理验收完成。

请求层与文件下载链路的默认回退 key 应统一使用：

- `request.failed`
- `network.error`
- `network.timeout`

不允许再直接把以下英文字符串作为默认用户提示：

- `Request Failed`
- `Network Error`
- `Refresh Failed`

## 7.2 网络错误 fallback

网络错误、超时、浏览器异常，不走后端 key。

统一前端 key：

- `network.error`
- `network.timeout`
- `request.failed`

## 8. 国际化后台治理能力

`system/config -> i18n` 管理端不应只停留在 CRUD，还应承担基础治理职责。

当前管理端应至少支持：

- 缺失 key 同步
- 缺失 locale 检测与占位补齐
- 重复 key 冲突审计
- 未使用 key 审计与清理
- 按模块导出翻译项
- 按模块导入翻译项
- 导入冲突阻断与明细反馈
- key 重命名预检与迁移报告导出

当前平台内置支持 locale：

- `zh-CN`
- `en-US`
- `ja-JP`
- `ko-KR`
- `fr-FR`

当前治理策略：

- 平台不为了“可能未来会用到”而持续预扩更多语言包
- 是否新增 locale，应由真实交付市场、客户要求、实施区域或合规要求驱动
- 在没有明确市场需求前，当前 5 个 locale 足以覆盖底座验证、演示和后续扩展链路

### 8.0 新增 locale 的准入与扩展流程

新增 locale 不应只理解为“复制一份语言文件”，而是一次 `system/config` 治理动作。

推荐准入条件：

- 已有明确目标市场或客户需求
- 已确认翻译责任人或外部翻译资源
- 已确认后续持续维护该 locale 的回归成本

执行流程：

1. 在前端新增对应 fallback locale 资源文件
2. 以现有基准语言包补齐全部 key，保证 key 集合一致
3. 在 `system/config -> i18n` 中导入或维护该 locale 的运行时翻译资产
4. 校验菜单、页头、按钮、空态、弹窗、导入导出反馈、错误 CSV、重命名迁移报告都可切换
5. 通过 `check:i18n-hardcode`、构建检查和 locale 缺失审计后再交付

约束：

- 不允许只补数据库语言包而不补前端 fallback 资源
- 不允许只补页面主文案而漏掉菜单、错误提示、导入导出和治理报表
- 不允许在没有明确业务依据时无限制扩语种，避免形成长期维护负债

### 8.1 重复 key 冲突

同一个 `key` 如果被多个模块复用，运行时语言包会按最终装载结果覆盖，容易产生“一个 key，多处语义”问题。

因此管理端需要审计：

- 同一 `key` 是否跨多个 `module`
- 同一 `key` 是否跨多个 `group`
- 同一 `key` 在不同模块下的值是否已经分叉

这类问题属于 `system/config` 的国际化治理问题，不应交给业务模块各自兜底。

### 8.2 未使用 key

未使用 key 指数据库中存在，但当前运行时代码未再引用的翻译键。

治理要求：

- 后端扫描代码引用并产出未使用 key 列表
- 扫描时应排除 i18n 词典文件、种子文件和测试文件，避免误判
- 清理动作优先支持按模块执行，避免一次性全量误删

未使用 key 不建议直接删除，推荐生命周期：

1. `active`
2. `observing`
3. `archived`
4. `delete`

建议规则：

- 首次识别未使用时，先进入观察期
- 观察期满后仍未被代码重新引用，再进入归档态
- 删除动作只允许针对已归档项，并要求二次确认
- 如果观察期或归档期内重新被代码引用，应自动恢复为 `active`
- 观察 / 归档 / 删除三类治理动作都应写入统一审计
- 审计详情至少要能回看：操作者、模块、生命周期状态流转、受影响 key 列表、受影响记录数
- `system/audit` 只负责承接与展示，生命周期判断和执行仍归属 `system/config -> i18n`

### 8.3 按模块导出

翻译导出不应只有全量导出，还应支持：

- 按 `module` 导出
- 作为模块迁移、模块交付和模块审计的辅助能力

这保证 `system/config` 可以为后续业务域提供独立的翻译资产边界。

当前导入导出治理还应满足：

- 导出文件列头走 i18n key 翻译，不把英文列名写死在运行时
- 导入结果摘要支持多语言展示 `created / updated / failed / row errors`
- 错误 CSV 使用统一 i18n key 列头和下载文件名
- 如果导入内容与现有 canonical 记录冲突，必须阻断并明确提示冲突行、冲突 key、归属模块等信息

### 8.4 冲突修复辅助

国际化治理不应只报问题，还应提供修复辅助信息。

对于重复 key 冲突，管理端应至少提供：

- 按冲突 key 直接回到列表筛选
- 按模块给出建议重命名 key
- 明确提示这是“建议”，不是自动改名

原因是自动改 key 会同时影响：

- 数据库存量翻译记录
- 前后端代码引用
- 导入导出资产

因此平台只负责给出稳定的修复辅助，不直接替业务域做破坏性改写。

### 8.5 长期占位值告警

占位值如 `[some.key]` 可以作为短期缺失翻译兜底，但不能长期留在系统里。

治理要求：

- 后端审计应识别“占位值超过阈值天数仍未修复”的记录
- 这类记录应按模块、key、locale 输出
- 管理端应支持从告警项直接定位到列表修复

默认可先使用 30 天作为疑似遗留阈值，后续再视平台治理要求调整。

### 8.6 key 重命名工作流

重复 key 的治理最终通常会落到 key 重命名，但这不应做成“后台自动改源码”的危险操作。

推荐工作流：

1. 后端预检
2. 展示受影响翻译记录、locale、目标 key 冲突情况
3. 扫描并展示源码引用文件
4. 由开发者先完成代码引用迁移
5. 在管理端显式确认后，再执行数据库中的 key 重命名

约束：

- 管理端只更新 `system_i18n` 中的翻译资产
- 管理端不直接修改仓库源码
- 如果目标 key 已存在，必须阻止执行
- 如果仍存在源码引用，必须要求显式确认

这符合 `system/config` 的职责边界：它负责配置与翻译资产治理，不负责远程篡改应用源码。

当前管理端生成的重命名迁移报告还应覆盖：

- 受影响模块
- 受影响 locale
- 目标 key 是否已存在
- 代码引用文件与命中次数
- 建议替换前后值
- 开发者执行清单

### 8.7 生成器与前端源码防回归

国际化治理不只针对运行时数据库资产，也要约束前端源码与代码生成链路。

当前规则：

- 生成器输出应坚持 key-first，不应把 `List / Create / Update / Delete` 这类展示文案直接固化到生成结果中
- 字段模板占位、页面动作名、审计标题等生成内容应优先来自 i18n key
- 前端源码需要通过 `frontend/scripts/check-i18n-hardcode.mjs` 做展示型硬编码扫描
- `frontend/package.json` 的 `prebuild` 必须执行 `check:i18n-hardcode`，作为国际化防回归门禁

## 7.3 未知错误 fallback

当后端返回未知 key：

- 先尝试 `t(message)`
- 如果翻译结果仍等于原 key，则显示：
  - 开发环境：原 key + 兜底文案
  - 生产环境：通用错误文案

## 8. 前后端责任边界

## 8.1 后端负责

- 定义业务码
- 定义错误 key
- 保证 key 稳定
- 不返回面向用户的自然语言

## 8.2 前端负责

- 翻译 key
- 选择展示方式（toast / inline / form error / empty state）
- 提供 fallback
- 补本地最小语言包

## 8.3 模块负责

新增模块时，模块 owner 必须同步补：

- 错误 key 清单
- 菜单和页面文案 key
- 表单字段 key
- 按钮文案 key

## 9. 错误展示方式规范

不是所有错误都应该用 toast。

## 9.1 toast

适合：

- 删除失败
- 保存失败
- 网络异常
- 权限不足提示

## 9.2 表单项错误

适合：

- 字段必填
- 邮箱格式错误
- 密码不合法

## 9.3 页面级错误

适合：

- 403
- 404
- 500
- 首屏加载失败

## 9.4 空态提示

适合：

- 搜索无结果
- 无数据
- 未配置

## 10. 请求层规范

请求层统一负责：

- 401 refresh 尝试
- 标准响应拆包
- 错误 key 翻译
- 默认错误提示

请求层不应负责：

- 业务成功 toast
- 表单字段级错误提示
- 页面专属提示逻辑

## 11. 登录与认证错误规范

以下错误属于认证域：

- `user.login.error.not_found`
- `user.login.error.disabled`
- `user.login.error.password_wrong`
- `refresh_token.invalid`
- `refresh_token.expired`
- `refresh_token.rotated`

后续 `auth` 模块独立后，建议逐步统一为：

- `auth.login.error.*`
- `auth.refresh.error.*`
- `auth.session.error.*`

当前阶段允许保留旧 key，但文档和新代码要向新命名靠拢。

## 12. 权限错误规范

统一使用：

- `permission.denied`
- `permission.role.invalid`
- `permission.policy.exists`

页面级无权限与接口级无权限都可以使用同一个 key，但前端展示方式不同。

## 13. 模块新增时必须补的内容

新增模块时，必须同步补：

- 后端错误 key
- 前端页面文案 key
- 本地 fallback 资源
- 数据库 i18n seed（如使用）
- 文档说明

## 14. 当前落地差距

当前已经做到：

- 后端大部分错误返回 key
- 前端有 fallbackResources
- 前端启动会拉远端语言包
- 请求层已统一优先按 `t(message)` 翻译
- 网络错误、超时和 transport error 已统一收口到稳定 key
- 前端源码已有 `check:i18n-hardcode`，locale 资源已有 `audit:i18n-locales`

当前仍缺：

- 成功/失败提示方式统一规范落地
- `auth` 拆分后的错误 key 收口
- 运行时远端语言包与本地 fallback 资源之间的新增 key 回填仍需持续同步

补充约束：

- 审计日志中的 `error_msg` 如果保存的是错误 key，前端详情页也应优先按 i18n key 翻译展示，而不是直接裸显 key。
- 平台层当前已为每个请求透传或生成 `X-Request-ID`；后续如果请求层做问题上报或下载错误摘要，也应把该标识纳入可追踪链路。
- 对于“接口返回 200 但业务结果中 `applied=false` / 存在校验错误”的批量导入场景，允许通过审计元数据单独把操作日志标记为失败，以保证审计语义真实。

## 15. 验收清单

当以下问题都能明确回答时，说明本设计完整：

- 后端是否只返回 key？
- 前端是否统一翻译 message？
- 网络错误是否有独立 key？
- 模块新增时是否知道要补哪些 i18n key？
- 成功提示是否由前端主导？
- 未知 key 是否有 fallback？

## 16. 下一份建议补的文档

如果继续推进文档设计，下一批建议是：

- `docs/acceptances/ACCEPTANCE_CHECKLIST.md`
- `docs/designs/FRONTEND_COMPONENT_PLAN.md`

因为错误与 i18n 边界定完后，下一步应该解决：

- 前端组件层如何承接这些文案与状态规范；
- 阶段性交付如何统一验收。
