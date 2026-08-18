# Business 模块导入导出功能实现计划

**任务**: 补全导入导出功能 (#3)  
**优先级**: 高  
**预计工期**: 3 天  
**开始日期**: 2026-08-16

## 现状分析

### ✅ 已实现（参考模板）
- **CMDB Host**: 完整的导入导出实现
  - 后端: `backend/modules/business/cmdb/host/host_import_export.go` (297行)
  - 前端: `frontend/src/modules/business/cmdb/host/api.ts` + `CmdbHostList.tsx`
  - 功能: 导出CSV、下载模板、导入主机

### ❌ 待实现（6个模块）

| 模块 | 优先级 | 复杂度 | 预计工期 |
|------|--------|--------|----------|
| 1. CMDB Group | P0 | 中 | 0.5天 |
| 2. CMDB Label | P0 | 低 | 0.4天 |
| 3. BizScope | P1 | 低 | 0.4天 |
| 4. Deploy Package | P1 | 中 | 0.6天 |
| 5. Deploy Template | P2 | 高 | 0.8天 |
| 6. Deploy Task | P2 | 低 | 0.3天 |

**总计**: 约 3 天

## 实现策略

### 阶段 1: 准备阶段（0.5天）
1. 深入分析 CMDB Host 实现作为模板
2. 提取可复用的代码模式和工具函数
3. 确定每个模块的字段映射和验证规则
4. 创建统一的导入导出接口规范

### 阶段 2: 核心实现（2天）
按优先级顺序实现：

#### 2.1 CMDB Label (0.4天) - 最简单，先突破
**数据模型**:
```go
type CmdbLabel struct {
    ID          uint
    Key         string  // 标签键
    Value       string  // 标签值
    Description string
    Color       string
    Category    string
}
```

**CSV 字段**:
- key, value, description, color, category

**验证规则**:
- key+value 组合唯一性
- category 枚举值检查

---

#### 2.2 BizScope (0.4天) - 业务域
**数据模型**:
```go
type BizScope struct {
    ID          uint
    Code        string  // 唯一编码
    Name        string
    Owner       string
    Environment string  // dev/test/prod
    Status      string  // active/inactive
}
```

**CSV 字段**:
- code, name, owner, environment, status, remark

**验证规则**:
- code 唯一性
- environment 枚举值 (dev/test/prod)
- status 枚举值 (active/inactive)

---

#### 2.3 CMDB Group (0.5天) - 主机分组（树形结构）
**数据模型**:
```go
type CmdbGroup struct {
    ID          uint
    Name        string
    ParentID    *uint   // 父分组ID
    Path        string  // 树形路径
    Description string
}
```

**CSV 字段**:
- name, parent_path, description

**验证规则**:
- name 在同级唯一
- parent_path 必须存在
- 防止循环引用

**复杂点**:
- 需要解析树形路径（如 "根分组/子分组/孙分组"）
- 导入时需要按层级顺序创建

---

#### 2.4 Deploy Package (0.6天) - 部署包
**数据模型**:
```go
type DeployPackage struct {
    ID              uint
    Name            string
    Version         string
    InstallCommand  string
    UninstallCommand string
    ExecutionMode   string  // shell/ansible
    SourceURL       string
}
```

**CSV 字段**:
- name, version, install_command, uninstall_command, execution_mode, source_url, description

**验证规则**:
- name+version 组合唯一性
- execution_mode 枚举值
- 命令格式验证

---

#### 2.5 Deploy Task (0.3天) - 部署任务（只读导出）
**数据模型**:
```go
type DeployTask struct {
    ID              uint
    PackageID       uint
    Action          string  // install/uninstall
    TargetType      string  // host/group
    Status          string
    CreatedAt       time.Time
}
```

**功能**: 仅导出（不支持导入，任务应该通过UI创建）

**CSV 字段**:
- task_id, package_name, action, target_type, status, created_at, updated_at

---

#### 2.6 Deploy Template (0.8天) - 部署模板（最复杂）
**数据模型**:
```go
type DeployTemplate struct {
    ID          uint
    Name        string
    Code        string  // 唯一编码
    Steps       JSON    // 步骤数组
    Variables   JSON    // 变量定义
}
```

**复杂点**:
- Steps 是 JSON 结构，CSV 难以表达
- 可能需要特殊格式（JSON 序列化字符串）

**方案**:
1. 简化版：只导出导入基本信息（name, code, description）
2. 完整版：使用 JSON 格式导出（不是 CSV）

**建议**: 先实现简化版 CSV，后续扩展为 JSON 格式

---

### 阶段 3: 集成测试（0.5天）
1. 编写单元测试（参考 `host_import_export_test.go`）
2. 前端 UI 集成测试
3. 端到端导入导出测试
4. 错误处理测试

## 技术实现细节

### 后端实现模式

每个模块创建 `{module}_import_export.go` 文件：

```go
package module

import (
    "github.com/your-org/pantheon-ops/backend/pkg/impexp"
)

// 1. 定义 CSV 行结构
type ModuleCSVRow struct {
    Field1 string `csv:"field1"`
    Field2 string `csv:"field2"`
}

// 2. 导出函数
func (s *Service) Export(ctx context.Context, query *ListQuery) ([]byte, error) {
    // 查询数据
    items, err := s.List(ctx, query)
    if err != nil {
        return nil, err
    }
    
    // 转换为 CSV 行
    rows := make([]ModuleCSVRow, len(items))
    for i, item := range items {
        rows[i] = toCSVRow(item)
    }
    
    // 生成 CSV
    return impexp.WriteCSV(rows)
}

// 3. 导入函数
func (s *Service) Import(ctx context.Context, fileData []byte) (*ImportResult, error) {
    // 解析 CSV
    rows, err := impexp.ParseCSV[ModuleCSVRow](fileData)
    if err != nil {
        return nil, err
    }
    
    // 验证并导入
    result := &ImportResult{}
    for i, row := range rows {
        if err := s.importRow(ctx, row); err != nil {
            result.Failed++
            result.Errors = append(result.Errors, ImportError{
                Row: i + 2,
                Message: err.Error(),
            })
        } else {
            result.Created++ // or result.Updated++
        }
    }
    
    return result, nil
}

// 4. 下载模板
func DownloadImportTemplate() ([]byte, error) {
    template := []ModuleCSVRow{
        {Field1: "example1", Field2: "example2"},
    }
    return impexp.WriteCSV(template)
}
```

### 前端实现模式

在 `api.ts` 中添加：

```typescript
// 导出
export function exportModules(params?: ListQuery) {
  return downloadFile({
    url: '/business/module/export',
    method: 'get',
    params,
    filename: 'module-export.csv',
  });
}

// 下载模板
export function downloadModuleImportTemplate() {
  return downloadFile({
    url: '/business/module/import-template',
    method: 'get',
    filename: 'module-import-template.csv',
  });
}

// 导入
export function importModules(file: File) {
  return uploadImportFile('/business/module/import', file);
}
```

在 `List.tsx` 中添加按钮：

```tsx
<ListHeaderActions>
  <ImportCsvButton
    onImport={handleImport}
    onDownloadTemplate={handleDownloadTemplate}
    disabled={!hasPerm('business:module:import')}
  />
  <Button
    type="primary"
    icon={<IconDownload />}
    onClick={handleExport}
    disabled={!hasPerm('business:module:export')}
  >
    {t('common.export')}
  </Button>
</ListHeaderActions>
```

## 可复用组件和工具

### 后端
1. **`pkg/impexp/csv.go`** - CSV 读写（含 BOM、防注入）
2. **`pkg/impexp/import_helpers.go`** - 导入辅助函数
3. **CMDB Host 实现** - 完整参考模板

### 前端
1. **`api/importExport.ts`** - `downloadFile`, `uploadImportFile`
2. **`components/patterns/actions/ImportCsvButton.tsx`** - 标准导入按钮
3. **CMDB Host List** - UI 集成参考

## 质量标准

### 后端
- ✅ 行号级别的错误收集
- ✅ 事务保证原子性
- ✅ CSV 注入防护
- ✅ 导出限流（10,000行）
- ✅ 导入限流（5,000行）
- ✅ UTF-8 BOM 支持
- ✅ 单元测试覆盖

### 前端
- ✅ 权限控制（export/import 权限）
- ✅ 错误提示国际化
- ✅ 导入结果反馈（成功/失败/错误详情）
- ✅ 文件类型验证
- ✅ 进度提示

## 风险和缓解

### 风险 1: Deploy Template 结构复杂
**缓解**: 先实现简化版（仅基本信息），后续扩展为 JSON 格式

### 风险 2: CMDB Group 树形结构处理
**缓解**: 参考 System Dept 的树形导入实现（已有成熟方案）

### 风险 3: 工期可能超预期
**缓解**: 
- 优先实现 P0/P1 模块
- P2 模块（Deploy Template/Task）可以后置
- 使用 Codex 加速开发

## 实施顺序（建议）

**Day 1**:
1. ✅ 分析 CMDB Host 实现（0.5h）
2. 实现 CMDB Label（3h）
3. 实现 BizScope（3h）

**Day 2**:
4. 实现 CMDB Group（4h）
5. 实现 Deploy Package（4h）

**Day 3**:
6. 实现 Deploy Task（2h）
7. 实现 Deploy Template 简化版（4h）
8. 集成测试（2h）

## 成功标准

- [ ] 6 个模块全部实现导入导出功能
- [ ] 前端 UI 集成完成（导入按钮 + 导出按钮）
- [ ] 后端 API 端点完整（/export, /import, /import-template）
- [ ] 权限控制正确配置
- [ ] 单元测试通过
- [ ] 端到端测试通过
- [ ] 错误处理完善
- [ ] 国际化文案完整

## 下一步行动

1. 使用 codegraph 深入分析 CMDB Host 的实现细节
2. 创建第一个模块（CMDB Label）的实现计划
3. 委托 Codex 实现后端代码
4. 委托 Codex 实现前端集成
5. 测试验证

---

**准备就绪，等待执行指令。**
