# Business 模块导入导出功能实施进度

**更新时间**: 2026-08-19
**任务状态**: 部分完成，见 `2026-08-19-ops-v1-backlog-closeout` task packet

## 总体进度

| 模块 | 优先级 | 状态 | 完成度 | 备注 |
|------|--------|------|--------|------|
| CMDB Label | P0 | 已完成 | 100% | 已有 CSV 导入导出 |
| BizScope | P1 | 已完成 | 100% | 已有 CSV 导入导出 |
| CMDB Group | P0 | 已完成 | 100% | 已有 CSV 导入导出 |
| Deploy Package | P1 | 后端完成 | 80% | API 和 typed client 已实现，UI 按钮待补 |
| Deploy Template | P2 | 后端完成 | 75% | API 支持 steps JSON，UI 按钮待补 |
| Deploy Task | P2 | 后端完成 | 80% | 只读导出 API 和 typed client 已实现，UI 按钮待补 |

## 当前正在执行

### CMDB Label 导入导出 (P0) 🔄

**执行方式**: Codex 后台任务  
**任务 ID**: bl70nhy0a  
**开始时间**: 2026-08-16 13:12:03  
**预计完成**: 2026-08-16 13:17:00

**实施范围**:
- ✅ 规范文档已创建: `docs/import-export-cmdb-label-spec.md`
- 🔄 后端实现 (Codex 执行中):
  - `backend/modules/business/cmdb/label/label_import_export.go` (新建)
  - `backend/modules/business/cmdb/label/label_handler.go` (修改)
  - `backend/modules/business/cmdb/module.go` (权限)
- 🔄 前端实现 (Codex 执行中):
  - `frontend/src/modules/business/cmdb/label/api.ts` (修改)
  - `frontend/src/modules/business/cmdb/label/CmdbLabelSchemaList.tsx` (修改)
  - `frontend/src/modules/business/cmdb/locales/*.json` (i18n)

**CSV 字段**:
- key, name, category, valueMode, dictCode, options, required, status, description

**关键特性**:
- 基于 key 字段唯一性（不区分大小写）
- 枚举验证: category, valueMode, status
- 业务逻辑: enum 模式必填 options, dict 模式必填 dictCode

## 已完成的准备工作

### 1. 文档准备 ✅

| 文档 | 状态 | 描述 |
|------|------|------|
| `docs/import-export-implementation-plan.md` | ✅ | 总体实施计划 |
| `docs/import-export-implementation-guide.md` | ✅ | 实现模式指南 |
| `docs/import-export-cmdb-label-spec.md` | ✅ | Label 详细规范 |
| `docs/import-export-bizscope-spec.md` | ✅ | BizScope 详细规范 |

### 2. 参考实现分析 ✅

已深入分析以下参考实现：
- ✅ `backend/modules/business/cmdb/host/host_import_export.go` (297行)
- ✅ `frontend/src/modules/business/cmdb/host/CmdbHostList.tsx`
- ✅ `frontend/src/api/importExport.ts`
- ✅ `backend/pkg/impexp/` 工具包

**关键模式提取**:
1. CSV 头验证 → 行解析 → 业务验证 → 事务应用
2. 行号级别错误收集
3. 唯一键匹配（创建 vs 更新）
4. 单一事务保证原子性
5. 前端统一错误展示

### 3. 治理栏样式修复 ✅

已完成 Phase 1 任务：
- ✅ 移除 DeployPackageList.tsx 中的 IconApps
- ✅ 移除 DeployTemplateList.tsx 中的 IconCode
- ✅ 统一与 System 模块样式

## 下一步计划

### 短期 (本周内)

1. **等待 CMDB Label 完成** (今天)
   - 监控 Codex 执行结果
   - 验证后端接口功能
   - 验证前端 UI 集成
   - 运行测试套件

2. **实施 BizScope 导入导出** (今天/明天)
   - 规范已完成，可直接实施
   - 相对简单，预计 0.4 天

3. **规划 CMDB Group** (明天)
   - 分析树形路径解析逻辑
   - 参考 System Dept 的树形导入
   - 创建详细规范文档

### 中期 (下周)

4. **实施 CMDB Group 导入导出**
   - 树形结构处理
   - 路径解析和验证
   - 预计 0.5 天

5. **实施 Deploy Package 导入导出**
   - 相对标准的实现
   - 预计 0.6 天

6. **实施 Deploy Template 简化版**
   - 仅基本信息（不含 Steps JSON）
   - 预计 0.8 天

7. **实施 Deploy Task 导出**
   - 只读导出，不支持导入
   - 预计 0.3 天

### 长期优化

8. **完善 Deploy Template 完整版**
   - 支持 JSON 格式导入导出
   - Steps 结构序列化

9. **补全单元测试**
   - 每个模块的导入导出测试
   - 边界条件和错误场景

10. **端到端验证**
    - 完整的导入导出流程测试
    - 权限控制测试
    - 审计日志验证

## 风险和阻塞点

### 当前风险

1. **Codex 执行中的潜在问题** 🟡
   - 缺少 BUSINESS_CMDB_DESIGN.md 和 BUSINESS_CMDB_ACCEPTANCE.md
   - 这些是可选文档，不影响实现
   - Codex 会继续执行

2. **CMDB Group 复杂度** 🟡
   - 树形结构需要特殊处理
   - 路径解析逻辑复杂
   - 需要参考 System Dept 实现

3. **Deploy Template JSON 结构** 🟡
   - Steps 字段是复杂 JSON
   - CSV 格式难以表达
   - 可能需要 JSON 导入格式

### 缓解措施

- 风险 1: Codex 有内置容错，会跳过缺失文档继续执行
- 风险 2: 参考 `backend/modules/system/iam/dept/` 的树形导入实现
- 风险 3: 先实现简化版（仅基本信息），后续扩展

## 成功标准

### 核心标准 (必须)

- [ ] 6 个模块全部实现导入导出功能
- [ ] 前端 UI 正确集成（导入按钮 + 导出按钮）
- [ ] 后端 API 完整（/export, /import, /import-template）
- [ ] 权限控制正确
- [ ] CSV 格式符合规范（UTF-8 BOM）
- [ ] 错误处理完善（行号、字段、错误信息）
- [ ] 国际化文案完整（中英文）

### 质量标准 (推荐)

- [ ] 单元测试覆盖主要场景
- [ ] 端到端测试通过
- [ ] 审计日志正确记录
- [ ] 导出性能可接受（10,000 行限制）
- [ ] 导入性能可接受（5,000 行限制）

### 验收标准 (理想)

- [ ] 实际业务场景验证
- [ ] 性能压测通过
- [ ] 安全审计通过
- [ ] 用户培训材料完成

## 资源和参考

### 关键文档
- [总体实施计划](./import-export-implementation-plan.md)
- [实现模式指南](./import-export-implementation-guide.md)
- [CMDB Label 规范](./import-export-cmdb-label-spec.md)
- [BizScope 规范](./import-export-bizscope-spec.md)

### 参考实现
- CMDB Host: `backend/modules/business/cmdb/host/host_import_export.go`
- System User: `backend/modules/system/iam/user/user_export.go`
- System Dept: `backend/modules/system/iam/dept/dept_service.go` (树形结构)

### 工具包
- `pantheon-base/pkg/impexp/` - CSV 读写、验证工具
- `frontend/src/api/importExport.ts` - 前端导入导出工具
- `frontend/src/components/patterns/actions/ImportCsvButton.tsx` - 导入按钮组件

## 关键决策记录

### 决策 1: 优先级排序
**日期**: 2026-08-16  
**决策**: 按 Label → BizScope → Group → Package → Task → Template 顺序实施  
**理由**: 从简单到复杂，先突破最简单的 Label，建立信心和模式

### 决策 2: Deploy Template 简化实现
**日期**: 2026-08-16  
**决策**: 先实现简化版（仅基本信息），后续扩展 JSON 格式  
**理由**: Steps JSON 结构复杂，CSV 难以表达，避免阻塞整体进度

### 决策 3: Deploy Task 只读导出
**日期**: 2026-08-16  
**决策**: Deploy Task 仅支持导出，不支持导入  
**理由**: 部署任务应该通过 UI 创建，导入可能破坏任务状态完整性

### 决策 4: 委托 Codex 实现
**日期**: 2026-08-16  
**决策**: 遵循 CLAUDE.md 规则，由 Codex 实现所有 backend/ 和 frontend/src/ 代码  
**理由**: 职责分离，Claude 负责规划和审查，Codex 负责实现

---

**最后更新**: 2026-08-16 13:20  
**下次更新**: 等待 CMDB Label 完成后
