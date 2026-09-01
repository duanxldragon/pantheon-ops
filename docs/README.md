# Pantheon-Ops 文档目录

本目录包含 Pantheon-Ops 项目的所有文档。

---

## 📚 文档结构

```
docs/
├── README.md                          # 本文件 - 文档索引
├── README.en.md                       # 英文文档索引
│
├── designs/                           # 设计文档
│   └── ... (其他设计文档)
│
├── audits/                            # 审计报告
│   └── v1.0/                          # V1.0 审计归档
│       ├── pantheon-ops_v1.0冻结前综合评审报告.md
│       ├── pantheon-ops_v1.0业务模块关联性与闭环检查报告.md
│
├── reviews/                           # 评审文档
│   └── (评审记录)
│
├── PROJECT_INHERITANCE.md             # 项目继承说明
├── PROJECT_INHERITANCE.en.md          # 项目继承说明（英文）
├── TASK_PACKET_OPS_TEMPLATE.md       # 任务包模板
├── TASK_PACKET_OPS_TEMPLATE.en.md    # 任务包模板（英文）
└── V1_RELEASE_CHECKLIST.md            # V1.0 发布检查表
```

---

## 📖 快速导航

### 核心文档

| 文档 | 描述 | 状态 |
|------|------|------|
| [V1_RELEASE_CHECKLIST.md](./V1_RELEASE_CHECKLIST.md) | V1.0 发布检查表 | ✅ 最新 |
| [PROJECT_INHERITANCE.md](./PROJECT_INHERITANCE.md) | 项目继承说明 | ✅ 最新 |
| [TASK_PACKET_OPS_TEMPLATE.md](./TASK_PACKET_OPS_TEMPLATE.md) | 任务包模板 | ✅ 最新 |

### V1.2 设计文档

| 文档 | 描述 | 状态 |
|------|------|------|
| [designs/V1.2_COMPLETE_DESIGN.md](../.harness/v1.2-docs/V1.2_COMPLETE_DESIGN.md) | V1.2 完整设计（第1部分）| ✅ 已完成 |
| [designs/V1.2_COMPLETE_DESIGN_PART2.md](../.harness/v1.2-docs/V1.2_COMPLETE_DESIGN_PART2.md) | V1.2 完整设计（第2部分）| ✅ 已完成 |
| [designs/V1.2_COMPLETE_DESIGN_PART3.md](../.harness/v1.2-docs/V1.2_COMPLETE_DESIGN_PART3.md) | V1.2 完整设计（第3部分）| ✅ 已完成 |

**V1.2 核心模块**:
- ArgoCD GitOps 集成
- Velero 备份恢复
- 告警规则引擎
- 文件分发系统

### V1.0 审计报告

| 文档 | 描述 | 日期 |
|------|------|------|
| [audits/v1.0/pantheon-ops_v1.0冻结前综合评审报告.md](./audits/v1.0/pantheon-ops_v1.0冻结前综合评审报告.md) | V1.0 综合评审报告 | 2026-08-28 |
| [audits/v1.0/pantheon-ops_v1.0业务模块关联性与闭环检查报告.md](./audits/v1.0/pantheon-ops_v1.0业务模块关联性与闭环检查报告.md) | 业务模块闭环检查 | 2026-08-28 |

---

## 🎯 按版本查看

### V1.0 (Current)
- ✅ 核心闭环完成
- ✅ CMDB + 部署 + K8s
- ⚠️ 服务实例/监控骨架存在
- ❌ ArgoCD/Velero 已下架

**相关文档**:
- [V1.0 发布检查表](./V1_RELEASE_CHECKLIST.md)
- [V1.0 综合评审报告](./audits/v1.0/pantheon-ops_v1.0冻结前综合评审报告.md)
- [V1.0 闭环检查报告](./audits/v1.0/pantheon-ops_v1.0业务模块关联性与闭环检查报告.md)

### V1.2 (Planning)
- 📐 ArgoCD 真实集成
- 📐 Velero 真实集成
- 📐 告警规则引擎
- 📐 文件分发系统

**相关文档**:
- [V1.2 完整设计方案](../.harness/v1.2-docs/V1.2_COMPLETE_DESIGN.md)

---

## 📝 文档类型说明

### 设计文档 (designs/)
包含系统架构、模块设计、API 设计等技术规格文档。

### 审计报告 (audits/)
包含版本审计、评审报告、修复记录等质量保证文档。

### 评审文档 (reviews/)
包含代码评审、设计评审等过程记录文档。

---

## 🔍 搜索指南

### 查找特定主题
- **部署相关**: `grep -r "部署" docs/`
- **K8s相关**: `grep -r "kubernetes\|k8s" docs/`
- **告警相关**: `grep -r "告警\|alert" docs/`

### 查找特定版本
- **V1.0 文档**: `docs/audits/v1.0/`
- **V1.2 文档**: `docs/designs/V1.2*`

---

## 📌 维护说明

### 添加新文档
1. 确定文档类型（设计/审计/评审）
2. 放入对应目录
3. 更新本索引文件

### 归档旧文档
1. 按版本创建归档目录
2. 移动过时文档到归档目录
3. 更新索引中的状态标记

### 文档命名规范
- 设计文档: `模块名_DESIGN.md`
- 审计报告: `项目名_版本_审计类型.md`
- 评审记录: `评审日期_评审主题.md`

---

## 📮 联系方式

- **项目负责人**: [待定]
- **技术架构**: Claude
- **文档维护**: [待定]

---

**最后更新**: 2026-08-28
**文档版本**: v1.0
