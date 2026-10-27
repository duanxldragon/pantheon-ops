---
title: 业务字典接入指南
doc_type: Design
layer: system/config / business/*
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-05-05
---

# 业务字典接入指南

English version: [BUSINESS_DICT_INTEGRATION_GUIDE.en.md](./BUSINESS_DICT_INTEGRATION_GUIDE.en.md)

本文定义业务模块如何使用 `system/config/dict`，避免业务模块各自硬编码枚举或把业务字典混进 system 底座语义。

---

## 1. 边界

`system/config/dict` 提供字典治理能力。

`business/*` 定义业务字典语义。

例如：

- 字典类型 `cmdb_host_status` 的治理能力属于 `system/config`。
- “主机状态有哪些值、哪些状态允许删除”属于 `business/cmdb`。

## 2. 命名规范

业务字典类型建议使用：

- `{business_domain}_{object}_{field}`

示例：

- `cmdb_host_status`
- `cmdb_host_lifecycle_status`
- `cmdb_vendor_type`
- `cmdb_vendor_status`

禁止：

- 使用过短类型名，例如 `status`。
- 把业务字典命名为 `system_*`。
- 前端直接硬编码自然语言选项。

## 3. 接入方式

业务模块接入字典时必须说明：

- 字典类型。
- 默认字典项。
- 是否允许运行期新增。
- 是否允许禁用已被业务数据引用的字典项。
- 导入导出时使用 label 还是 value。

## 4. 引用保护

业务字典被业务数据引用后，应优先采用软保护：

- 禁用前检查是否存在启用业务数据引用。
- 删除前检查引用数量。
- 如果当前未实现强引用检查，必须在设计文档中标记为待补强约束。

## 5. CMDB 示例

| 字典类型 | 用途 | 推荐值 |
| :--- | :--- | :--- |
| `cmdb_host_status` | 主机运行状态 | `online`、`offline`、`maintenance` |
| `cmdb_host_lifecycle_status` | 生命周期 | `planned`、`active`、`retired` |
| `cmdb_vendor_type` | 供应商类型 | `cloud`、`idc`、`hardware`、`service` |
| `cmdb_vendor_status` | 供应商状态 | `enabled`、`disabled` |

## 6. 验收

- 页面筛选、表单、详情展示均来自字典或明确固定枚举。
- 新增业务字典已补 i18n key。
- 字典禁用或删除不会静默破坏业务数据。
- 业务模块设计文档已声明字典依赖。
