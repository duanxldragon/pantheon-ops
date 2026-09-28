---
title: Version Management Guide
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/designs/FOUNDATION_RELEASE_MODEL.md
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-09-03
---

# Pantheon Base - 版本管理指南

## 1. 版本策略概述

Pantheon Base 采用**语义化版本 + Foundation Release 模型**：

- **版本号格式**: `pantheon-base-v<major>.<minor>.<patch>`
- **当前版本**: `pantheon-base-v0.10.26`
- **Release Line**: `release/0.10`（兼容性元数据，非 Git 分支）
- **分支策略**: 仅保留 `main` 分支，release 通过不可变 tag 发布

---

## 2. 语义化版本规则

### 2.1 版本号含义

```
pantheon-base-v<major>.<minor>.<patch>
                 |       |       |
                 |       |       └─ 安全修复、Bug 修复、兼容性补丁
                 |       └───────── 新功能、治理增强、向后兼容的优化
                 └───────────────── 破坏性变更、基础契约变更
```

### 2.2 Major 版本（破坏性变更）

**何时升级 Major**:
- 基础契约变更（权限模型、i18n key 语义、菜单/路由契约）
- 共享层 API 不兼容变更
- 消费方式发生破坏性变化
- 需要消费仓大范围重构的改动

**示例**:
- `v0.10.25 → v1.0.0`: 权限模型从 Casbin 迁移到其他方案
- `v1.0.0 → v2.0.0`: 前端路由从 React Router v5 升级到 v6

### 2.3 Minor 版本（新功能/增强）

**何时升级 Minor**:
- 新增系统模块（如 MFA、SSO、审计增强）
- 新增共享组件或工具
- 架构优化（不影响消费仓）
- 设计系统工程化（如本次 PR #285）
- 质量门禁增强
- 性能优化

**示例**:
- `v0.10.25 → v0.11.0`: 新增 MFA 模块
- `v0.11.0 → v0.12.0`: 前端设计系统工程化（新增 2837 行文档 + Token 体系扩展）

**本次 PR #285 建议版本**: `v0.11.0`
- **理由**: 设计系统工程化属于重大增强，新增 4 个完整文档（2837 行）+ 9 个容器 Token + 机械门禁优化
- **影响范围**: 前端开发规范、设计协作流程、AI 生成质量
- **向后兼容**: 是（不影响现有业务代码）

### 2.4 Patch 版本（修复）

**何时升级 Patch**:
- 安全漏洞修复
- Bug 修复（不改变功能行为）
- 兼容性补丁
- 文档修正（Typo、链接失效）
- 依赖版本升级（安全修复）

**示例**:
- `v0.10.25 → v0.10.26`: 修复登录日志时区问题
- `v0.10.26 → v0.10.27`: 升级依赖修复安全漏洞

---

## 3. Release 发布流程

### 3.1 Release 前置条件

发布前必须满足以下条件：

#### 质量门禁 ✅
- [ ] GitHub Required Checks 全部通过
- [ ] CodeQL Security 无高危未解决问题
- [ ] SonarCloud Quality Gate 通过
- [ ] Dependabot 无高危依赖漏洞
- [ ] Secret Scan 无泄漏

#### 文档完整性 ✅
- [ ] Release Notes 已编写
- [ ] Consumer Impact Summary 已编写（如影响消费仓）
- [ ] Upgrade Notes 已编写（如需要升级步骤）
- [ ] CHANGELOG 已更新

#### 验证完成 ✅
- [ ] 冒烟测试通过
- [ ] 关键业务流程验证通过
- [ ] 跨主题/暗色模式验证通过（如涉及 UI 变更）

### 3.2 Release 发布步骤

#### Step 1: 准备 Release Notes

创建 `.harness/evidence/<task-id>/release-notes.md`:

```markdown
# Release Notes: pantheon-base-v0.11.0

## 发布信息
- **版本**: v0.11.0
- **Release Line**: release/0.11
- **发布日期**: 2026-09-03
- **上一版本**: v0.10.26

## 核心变更

### 🎨 前端设计系统工程化
- 新增 4 个工程文档（2837 行）
- Token 体系扩展（+9 个容器 token，+28% 覆盖率）
- UI 模式库（12 类完整代码模板）
- 机械门禁优化（白名单机制）

### 📚 文档体系
- `COMPONENT_STYLING_GUIDE.md`: BEM 命名、Token 使用
- `UI_PATTERN_LIBRARY.md`: 12 类 UI 模式模板
- `DESIGN_ENGINEERING_GUIDE.md`: 设计协作流程
- `TOKEN_MIGRATION_GUIDE.md`: Token 迁移指南

## 消费仓影响

### Pantheon Ops
- **影响等级**: 低
- **需要升级**: 否（向后兼容）
- **建议操作**: 可选升级以获得设计规范支持

### 影响分析
- ✅ 不涉及 API 契约变更
- ✅ 不涉及权限模型变更
- ✅ 不涉及 i18n key 变更
- ✅ 不涉及菜单/路由变更

## 升级说明

### 从 v0.10.x 升级
1. 无需代码变更
2. 可选：参考新文档优化前端开发流程
3. 可选：参考 Token 迁移指南优化现有组件

### 破坏性变更
无

## 验证结论
- ✅ GitHub Required Checks: 全部通过
- ✅ CodeQL Security: 无高危问题
- ✅ SonarCloud Quality Gate: 通过
- ✅ Frontend Unit Tests: 通过
- ✅ 机械门禁: 0 个违规

## 相关 PR
- #285: Frontend Design System Engineering Alignment
```

#### Step 2: 切换到 main 分支

```bash
cd D:\workspace\go\pantheon-platform\pantheon-base
git checkout main
git pull origin main
```

#### Step 3: 合并 PR

```bash
# 确保所有检查通过
gh pr checks 285

# Squash 合并 PR
gh pr merge 285 --squash --delete-branch

# 拉取最新 main
git pull origin main
```

#### Step 4: 打标签

```bash
# 打标签
git tag -a pantheon-base-v0.11.0 -m "Release v0.11.0: Frontend Design System Engineering Alignment

🎨 前端设计系统工程化
- Complete design system documentation (2837 lines)
- Container token system (+9 tokens, +28% coverage)
- UI pattern library (12 patterns with full code templates)
- Mechanical gate optimization (whitelist mechanism)
- AI-friendly design constraints and templates

📚 新增文档
- COMPONENT_STYLING_GUIDE.md (682 lines)
- UI_PATTERN_LIBRARY.md (845 lines)
- DESIGN_ENGINEERING_GUIDE.md (890 lines)
- TOKEN_MIGRATION_GUIDE.md (420 lines)

✅ 质量验证
- All GitHub checks passed
- CodeQL security cleared
- SonarCloud quality gate passed
- Zero contract violations

🔗 相关 PR: #285

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"

# 推送标签
git push origin pantheon-base-v0.11.0
```

#### Step 5: 创建 GitHub Release

```bash
# 方式 1: 使用 gh CLI
gh release create pantheon-base-v0.11.0 \
  --title "Pantheon Base v0.11.0 - Frontend Design System Engineering" \
  --notes-file .harness/evidence/2026-09-03-frontend-design-alignment/release-notes.md \
  --latest

# 方式 2: 手动在 GitHub 网页创建
# 访问: https://github.com/duanxldragon/pantheon-base/releases/new
# 选择标签: pantheon-base-v0.11.0
# 填写 Release Notes
```

### 3.3 Release 资产

每个 Release 应包含：
- **Tag**: 不可变标签（`pantheon-base-v<x.y.z>`）
- **Release Notes**: 变更说明
- **Consumer Impact**: 消费仓影响分析
- **Upgrade Guide**: 升级指南
- **Verification Evidence**: 验证证据（`.harness/evidence/`）

---

## 4. 版本号决策流程图

```
┌─────────────────────────────────────┐
│  有破坏性变更？                       │
│  (契约不兼容/需要消费仓大范围重构)     │
└──────────┬──────────────────────────┘
           │
    ┌──────▼──────┐
    │   是        │   否
    │             │
    ▼             ▼
┌────────┐   ┌─────────────────────────────┐
│ Major  │   │ 新增功能/增强/优化？          │
│ v1.0.0 │   │ (新模块/新文档/架构优化)       │
└────────┘   └──────────┬──────────────────┘
                        │
                 ┌──────▼──────┐
                 │   是        │   否
                 │             │
                 ▼             ▼
            ┌────────┐   ┌─────────┐
            │ Minor  │   │ Patch   │
            │ v0.11.0│   │ v0.10.27│
            └────────┘   └─────────┘
```

---

## 5. CHANGELOG 维护

### 5.1 CHANGELOG 格式

`CHANGELOG.md` 采用 [Keep a Changelog](https://keepachangelog.com/) 格式：

```markdown
# Changelog

All notable changes to Pantheon Base will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.11.0] - 2026-09-03

### Added
- 完整的前端设计系统工程文档（2837 行）
- 9 个容器 Token（交互/展示/操作三层语义）
- 12 类 UI 模式完整代码模板
- Token 迁移白名单机制

### Changed
- 机械门禁优化（语义色白名单 + 精细调整间距白名单）
- DESIGN.md 文档顺序更新

### Fixed
- N/A

## [0.10.26] - 2026-09-02

### Added
- Foundation release consumer pipeline

### Fixed
- Repository layout contract frontend library entrypoint

## [0.10.25] - 2026-09-01
...
```

### 5.2 CHANGELOG 更新时机

- **每个 PR 合并后**: 更新 `[Unreleased]` 章节
- **Release 发布时**: 将 `[Unreleased]` 内容移到新版本章节

---

## 6. 版本发布检查清单

### 发布前检查 ✅
- [ ] 所有 PR 已合并到 `main`
- [ ] GitHub Required Checks 全部通过
- [ ] Release Notes 已编写
- [ ] Consumer Impact Summary 已编写（如适用）
- [ ] CHANGELOG 已更新
- [ ] 版本号已确定（Major/Minor/Patch）

### 发布操作 ✅
- [ ] 切换到 `main` 分支
- [ ] 拉取最新代码
- [ ] 打标签（`git tag -a pantheon-base-v<x.y.z>`）
- [ ] 推送标签（`git push origin pantheon-base-v<x.y.z>`）
- [ ] 创建 GitHub Release

### 发布后验证 ✅
- [ ] GitHub Release 页面正确显示
- [ ] Release Notes 完整
- [ ] 标签不可变（不能删除/修改）
- [ ] 通知相关团队

### 消费仓同步（如适用）✅
- [ ] 更新 `pantheon-ops` 的 `foundation-release.lock.json`
- [ ] 运行 `npm run check:base-sync`
- [ ] 验证业务 overlay 兼容性
- [ ] 提交消费仓 PR

---

## 7. 常见场景

### 场景 1: 安全漏洞修复

```bash
# 修复漏洞并合并 PR
gh pr merge <pr-number> --squash

# 打 Patch 版本
git tag -a pantheon-base-v0.10.27 -m "Security: Fix XSS vulnerability in user input"
git push origin pantheon-base-v0.10.27

# 创建 Release
gh release create pantheon-base-v0.10.27 \
  --title "Pantheon Base v0.10.27 - Security Fix" \
  --notes "🔒 修复用户输入 XSS 漏洞"
```

### 场景 2: 新增系统模块

```bash
# 合并 MFA 功能 PR
gh pr merge <pr-number> --squash

# 打 Minor 版本
git tag -a pantheon-base-v0.11.0 -m "Feature: Add MFA/TOTP support"
git push origin pantheon-base-v0.11.0

# 创建 Release（包含升级说明）
gh release create pantheon-base-v0.11.0 \
  --title "Pantheon Base v0.11.0 - MFA Support" \
  --notes-file release-notes.md
```

### 场景 3: 破坏性架构变更

```bash
# 合并权限模型重构 PR
gh pr merge <pr-number> --squash

# 打 Major 版本
git tag -a pantheon-base-v1.0.0 -m "BREAKING: Refactor permission model"
git push origin pantheon-base-v1.0.0

# 创建 Release（必须包含详细的升级指南）
gh release create pantheon-base-v1.0.0 \
  --title "Pantheon Base v1.0.0 - Permission Model v2" \
  --notes-file release-notes.md \
  --discussion-category "Announcements"
```

---

## 8. 版本命名约定

### 8.1 标签命名

- **格式**: `pantheon-base-v<major>.<minor>.<patch>`
- **示例**: 
  - ✅ `pantheon-base-v0.11.0`
  - ✅ `pantheon-base-v1.0.0`
  - ❌ `v0.11.0` (缺少项目前缀)
  - ❌ `pantheon-base-0.11.0` (缺少 v 前缀)

### 8.2 Release Line

- **格式**: `release/<major>.<minor>`
- **用途**: 兼容性元数据，记录在 consumer 的 `foundation-release.lock.json`
- **示例**: `release/0.10`, `release/0.11`, `release/1.0`
- **注意**: Release Line 不是 Git 分支，仅用于语义标记

### 8.3 Release Notes 标题

- **格式**: `Pantheon Base v<x.y.z> - <简短描述>`
- **示例**:
  - `Pantheon Base v0.11.0 - Frontend Design System Engineering`
  - `Pantheon Base v0.10.27 - Security Fix`
  - `Pantheon Base v1.0.0 - Permission Model v2`

---

## 9. 参考文档

- [FOUNDATION_RELEASE_MODEL.md](./designs/FOUNDATION_RELEASE_MODEL.md): Foundation Release 模型详解
- [PLATFORM_CONTRACT.md](./contracts/PLATFORM_CONTRACT.md): 平台契约
- [Semantic Versioning 2.0.0](https://semver.org/): 语义化版本规范
- [Keep a Changelog](https://keepachangelog.com/): CHANGELOG 格式规范

---

## 10. FAQ

### Q1: 何时应该升级 Major 版本？
**A**: 只有在发生破坏性变更时才升级 Major 版本，例如：
- 权限模型不兼容变更
- i18n key 语义变更
- 共享层 API 不兼容变更
- 需要消费仓大范围重构

大部分情况下，应该通过向后兼容的方式实现新功能（Minor 版本）。

### Q2: 本次 PR #285 应该升级到什么版本？
**A**: 建议 `v0.11.0`（Minor 版本），理由：
- 新增重大功能（设计系统工程化）
- 2837 行新文档 + Token 体系扩展
- 向后兼容（不影响现有业务代码）

### Q3: 如何回滚到旧版本？
**A**: 
```bash
# 消费仓回滚到旧版本
# 编辑 docs/PROJECT_INHERITANCE.md
base_version: pantheon-base-v0.10.26  # 改为旧版本

# 运行同步检查
npm run check:base-sync
```

### Q4: 是否需要维护多个 release 分支？
**A**: 不需要。Pantheon Base 只维护 `main` 分支，release 通过不可变 tag 发布。如需修复旧版本漏洞，可以：
1. 从旧 tag 创建临时分支
2. 修复并打新 patch tag
3. 删除临时分支

### Q5: 消费仓何时需要升级？
**A**: 
- **Patch 版本**: 尽快升级（安全修复）
- **Minor 版本**: 按需升级（新功能/优化）
- **Major 版本**: 计划升级（需要重构）
