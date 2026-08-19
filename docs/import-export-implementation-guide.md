# Business 模块导入导出功能实现指南

## 当前状态

**时间**: 2026-08-16
**问题**: Codex 服务连接失败（Reconnecting 5/5），无法通过 Codex 自动实现

## 实现优先级

基于 SRE 场景和功能完整度：

1. **CMDB Label** (P0) - 标签是资源分类的基础
2. **CMDB Group** (P1) - 主机分组是批量操作的前提
3. **BizScope** (P1) - 业务域是资源隔离的基础
4. **Deploy Package** (P2) - 部署包管理
5. **Deploy Template** (P2) - 部署模板管理
6. **Deploy Task** (P3) - 任务通常不需要导入导出（只导出日志）

## 参考实现

所有模块应参考 **CMDB Host** 的完整实现：
- 后端：`backend/modules/business/cmdb/host/host_import_export.go` (439行)
- 前端：`frontend/src/modules/business/cmdb/host/api.ts` (exportHosts/importHosts/downloadHostImportTemplate)

## 实现模式

### 后端（Go）

每个模块需要创建 `{module}_import_export.go` 文件，包含：

```go
// 1. CSV 表头常量
const (
    {Module}ExportCSVHeader = "field1,field2,field3,..."
    {Module}ImportMaxRows   = 5000
)

// 2. 导出函数
func (s *{Module}Service) Export(ctx context.Context, query *{Module}Query) ([]byte, error) {
    // 查询数据（限制 10000 行）
    // 转换为 CSV
    // 添加 UTF-8 BOM
    // 返回字节流
}

// 3. 导入函数
func (s *{Module}Service) Import(ctx context.Context, fileContent []byte) (*impexp.ImportResult, error) {
    // 解析 CSV（使用 impexp.ParseCSV）
    // 验证表头
    // 逐行验证和转换
    // 批量创建/更新（事务）
    // 返回 ImportResult
}

// 4. Handler 端点
// GET  /business/{module}/{resource}/export
// POST /business/{module}/{resource}/import
// GET  /business/{module}/{resource}/import-template
```

### 前端（TypeScript）

在 `api.ts` 中添加三个函数：

```typescript
// 1. 导出
export function export{Resources}(params?: {Module}ListQuery) {
  return downloadFile({
    url: '/business/{module}/{resource}/export',
    method: 'get',
    params: params as Record<string, unknown> | undefined,
    filename: '{module}-{resource}-export.csv',
  });
}

// 2. 下载模板
export function download{Module}ImportTemplate() {
  return downloadFile({
    url: '/business/{module}/{resource}/import-template',
    method: 'get',
    filename: '{module}-{resource}-import-template.csv',
  });
}

// 3. 导入
export function import{Resources}(file: File) {
  return uploadImportFile('/business/{module}/{resource}/import', file);
}
```

在列表页面组件中集成（参考 `CmdbHostList.tsx:159-166, 472-486`）。

## 模块详细规范

### 1. CMDB Label

**文件位置**:
- 后端：`backend/modules/business/cmdb/label/label_import_export.go`
- 前端：`frontend/src/modules/business/cmdb/label/api.ts`

**CSV 表头**:
```
key,name,category,valueMode,dictCode,options,required,status,description
```

**核心字段映射**:
- `key`: 标签键（唯一）
- `name`: 显示名称
- `category`: 分类（system/business/custom）
- `valueMode`: 值模式（single/multiple/freeform）
- `dictCode`: 字典代码（可选）
- `options`: JSON 数组字符串（预设选项）
- `required`: 是否必填（true/false）
- `status`: 状态（active/inactive）
- `description`: 描述

**验证规则**:
- `key`: 必填，唯一性检查
- `category`: 枚举值验证
- `valueMode`: 枚举值验证
- `options`: JSON 格式验证
- 导入策略：key 存在则更新，否则创建

**权限码**:
- `business:cmdb:label:export`
- `business:cmdb:label:import`

### 2. CMDB Group

**文件位置**:
- 后端：`backend/modules/business/cmdb/group/group_import_export.go`
- 前端：`frontend/src/modules/business/cmdb/group/api.ts`

**CSV 表头**:
```
code,name,parentCode,type,description,status
```

**核心字段映射**:
- `code`: 分组编码（唯一）
- `name`: 分组名称
- `parentCode`: 父分组编码（可选，用于树形结构）
- `type`: 分组类型（static/dynamic）
- `description`: 描述
- `status`: 状态（active/inactive）

**验证规则**:
- `code`: 必填，唯一性检查
- `parentCode`: 如果提供，必须存在且不能形成循环
- `type`: 枚举值验证
- 导入策略：code 存在则更新，否则创建
- 树形结构处理：先导入父节点，再导入子节点（可能需要两遍扫描）

**权限码**:
- `business:cmdb:group:export`
- `business:cmdb:group:import`

### 3. BizScope

**文件位置**:
- 后端：`backend/modules/business/bizscope/bizscope_import_export.go`
- 前端：`frontend/src/modules/business/bizscope/api.ts`

**CSV 表头**:
```
code,name,owner,environment,deptId,description,status
```

**核心字段映射**:
- `code`: 业务域编码（唯一）
- `name`: 业务域名称
- `owner`: 负责人
- `environment`: 环境（dev/test/prod）
- `deptId`: 部门ID
- `description`: 描述
- `status`: 状态（active/inactive）

**验证规则**:
- `code`: 必填，唯一性检查
- `environment`: 枚举值验证
- `deptId`: 部门存在性验证
- 导入策略：code 存在则更新，否则创建

**权限码**:
- `business:bizscope:export`
- `business:bizscope:import`

### 4. Deploy Package

**文件位置**:
- 后端：`backend/modules/business/deploy/package/package_import_export.go`
- 前端：`frontend/src/modules/business/deploy/api.ts`

**CSV 表头**:
```
name,version,category,installCommand,uninstallCommand,executionMode,sourceUrl,status,description
```

**核心字段映射**:
- `name`: 软件包名称
- `version`: 版本号
- `category`: 分类（middleware/database/application等）
- `installCommand`: 安装命令
- `uninstallCommand`: 卸载命令
- `executionMode`: 执行模式（script/ansible）
- `sourceUrl`: 源地址
- `status`: 状态（active/inactive）
- `description`: 描述

**验证规则**:
- `name+version`: 组合唯一性
- `executionMode`: 枚举值验证
- 导入策略：name+version 存在则更新，否则创建

**权限码**:
- `business:deploy:package:export`
- `business:deploy:package:import`

### 5. Deploy Template

**文件位置**:
- 后端：`backend/modules/business/deploy/template/template_import_export.go`
- 前端：`frontend/src/modules/business/deploy/api.ts`

**CSV 表头**:
```
code,name,packageName,packageVersion,preCheckSteps,mainSteps,postCheckSteps,status,description
```

**注意**: 步骤字段（preCheckSteps, mainSteps, postCheckSteps）是 JSON 数组，导入导出时需要序列化/反序列化。

**核心字段映射**:
- `code`: 模板编码（唯一）
- `name`: 模板名称
- `packageName`: 关联软件包名称
- `packageVersion`: 关联软件包版本
- `preCheckSteps`: 前置检查步骤（JSON）
- `mainSteps`: 主执行步骤（JSON）
- `postCheckSteps`: 后置检查步骤（JSON）
- `status`: 状态（active/inactive）
- `description`: 描述

**验证规则**:
- `code`: 必填，唯一性检查
- `packageName+packageVersion`: 软件包存在性验证
- JSON 字段：格式验证
- 导入策略：code 存在则更新，否则创建

**权限码**:
- `business:deploy:template:export`
- `business:deploy:template:import`

## 通用实现要点

### 安全措施（必须）
1. **CSV 注入防护**: 使用 `impexp.EscapeCSVInjection()` 处理所有单元格
2. **行数限制**: 导出最多 10000 行，导入最多 5000 行
3. **UTF-8 BOM**: 导出时添加 BOM，确保 Excel 正确识别
4. **事务保护**: 导入操作必须在事务中执行，失败全部回滚
5. **审计日志**: 使用 `common.SetAuditMetadata()` 记录所有变更

### 错误处理（必须）
1. **行号级别错误收集**: 使用 `impexp.ImportResult` 结构
2. **国际化错误消息**: 所有错误消息使用 i18n key
3. **详细错误信息**: 记录失败的行号、字段名、错误原因

### 前端集成（必须）
1. **导入按钮**: 使用 `<ImportCsvButton>` 组件
2. **导出按钮**: 集成到 `TableBatchActionBar` 的 `prefixActions`
3. **权限控制**: 使用 `hasPerm()` 检查权限
4. **进度提示**: 导入时显示 loading 状态，完成后显示成功/失败统计

## 测试验证

每个模块实现后需要验证：

1. **导出功能**:
   - 有数据时导出成功，文件可用 Excel 打开
   - 无数据时返回只有表头的文件
   - 字段值包含逗号、引号、换行符时正确转义
   
2. **导入模板**:
   - 下载的模板与导出格式一致
   
3. **导入功能**:
   - 合法数据导入成功
   - 重复数据触发更新逻辑
   - 非法数据返回详细错误（行号 + 字段 + 原因）
   - 部分失败时返回成功/失败统计

4. **权限控制**:
   - 无权限时前端按钮隐藏
   - 无权限时后端返回 403

## 下一步行动

### 选项 A: 等待 Codex 服务恢复
等待 Codex 连接问题解决，然后使用简化的 Codex 命令执行实现。

### 选项 B: 手动实现
如果时间紧急，可以手动实现：
1. 复制 `host_import_export.go` → `label_import_export.go`
2. 全局替换关键词（Host → LabelSchema, host → label）
3. 修改 CSV 表头和字段映射
4. 在 handler 中注册路由
5. 重复以上步骤完成其他模块

### 选项 C: 分阶段实现
先完成最重要的 CMDB Label 和 BizScope，验证流程后再批量完成其他模块。

---

**创建时间**: 2026-08-16
**状态**: 待执行
**依赖**: Codex 服务可用 或 人工实现
