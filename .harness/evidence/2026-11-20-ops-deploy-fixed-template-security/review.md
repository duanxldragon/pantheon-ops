# T02 Review — 固定部署模板参数与卸载路径安全

## 结论

代码级发现已修复并通过对应用例；本检查点不宣称整包验收。真实 SSH V7、独立评审和维护者终验仍为 open gap。

## Findings（严重度从高到低）

1. **HIGH — 包版本未进入共享安全门。** MySQL `MYSQL_VERSION="%s"` 与 Redis fallback `REDIS_VERSION="%s"` 直接插值 `pkg.Version`，旧包行可携带 `"; touch ...; #`。已修复：`renderFixedTemplateScript` 统一校验包版本，并允许合法 semver `+`。用例 `TestFixedTemplateRejectsUnsafePackageVersion`、`TestFixedTemplateAllowsSemverBuildMetadataPackageVersion`。
2. **HIGH — 渲染期卸载危险根检查可绕过。** 校验从 `templateParams["action"]` 读取动作，旧任务参数通常不含该键，`task.Action=uninstall` 时不会触发危险根检查。已修复：渲染使用 `task.Action` 构造有效校验视图；创建/快照校验也注入归一化动作。用例 `TestFixedTemplateRenderRejectsUnsafePersistedUninstall`。
3. **HIGH — Redis 密码 heredoc 命令替换。** `requirePassword` 仅拒绝空白后被裸插入未加引号的 heredoc，`$(id)` 可执行。已修复：`REDIS_PASSWORD` 使用 `shellQuoteSingle`，配置文件通过 `printf` 追加；密码内容保持不透明。用例 `TestFixedTemplateRedisPasswordStaysOpaque`。
4. **MEDIUM — 七模板覆盖不均。** 原用例只覆盖部分模板。已新增七模板合法参数表与逐模板非法元数据表；既有 MySQL/Redis/MinIO/Harbor/JDK/Prometheus 兼容回归同时通过。
5. **OPEN — 真实 SSH 注入与指纹校验未执行。** 无授权隔离 SSH 目标，V7 不能以单元测试代替。保留为 residual，不代签。

## Machine Readable

```json
{
  "taskId": "2026-11-20-ops-deploy-fixed-template-security",
  "verdict": "changes requested",
  "reviewScope": "self findings-first review of the local implementation checkpoint",
  "findings": [
    {
      "id": "T02-F1",
      "severity": "high",
      "status": "fixed",
      "summary": "pkg.Version reached MySQL/Redis shell literals without validation"
    },
    {
      "id": "T02-F2",
      "severity": "high",
      "status": "fixed",
      "summary": "render-time uninstall check read action from params instead of task.Action"
    },
    {
      "id": "T02-F3",
      "severity": "high",
      "status": "fixed",
      "summary": "Redis requirePassword was interpolated raw into an unquoted heredoc"
    },
    {
      "id": "T02-F4",
      "severity": "medium",
      "status": "fixed",
      "summary": "safety coverage did not exercise all seven built-in fixed templates"
    },
    {
      "id": "T02-F5",
      "severity": "high",
      "status": "open",
      "summary": "V7 isolated SSH injection/fingerprint acceptance not executed"
    }
  ],
  "linkage": {
    "taskManifest": ".harness/tasks/2026-11-20-ops-deploy-fixed-template-security/manifest.json",
    "evidence": ".harness/evidence/2026-11-20-ops-deploy-fixed-template-security/commands.json",
    "reviewFile": ".harness/evidence/2026-11-20-ops-deploy-fixed-template-security/review.md",
    "changeRef": "none",
    "planRefs": [
      ".harness/tasks/2026-11-20-ops-review-remediation/plan.md"
    ]
  },
  "updatedAt": "2026-11-22T04:05:18Z"
}
```

## 验证映射

| 发现 | 回归 |
| --- | --- |
| F1 | `TestFixedTemplateRejectsUnsafePackageVersion`, `TestFixedTemplateAllowsSemverBuildMetadataPackageVersion` |
| F2 | `TestFixedTemplateSnapshotValidationUsesEffectiveUninstallAction`, `TestFixedTemplateRenderRejectsUnsafePersistedUninstall` |
| F3 | `TestFixedTemplateRedisPasswordStaysOpaque` |
| F4 | `TestFixedTemplateValidParamsForAllBuiltInTemplates`, `TestFixedTemplateRejectsUnsafeMetadataForAllBuiltInTemplates` |

## 未覆盖

- V7 真实 SSH 注入无副作用与指纹不符拒绝。
- 独立 code/security review、完整 race、维护者最终功能验收。
