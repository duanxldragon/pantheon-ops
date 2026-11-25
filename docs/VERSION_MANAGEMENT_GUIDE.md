---
title: Version Management Guide
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/designs/FOUNDATION_RELEASE_MODEL.md
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-11-14
---

# Pantheon Base - 版本管理指南

## 1. 当前版本与唯一命名

Pantheon Base 采用语义化版本和 Foundation Release 模型。版本身份由不可变 tag、release manifest、发布资产及校验和共同确定，不能从 `main` 的 HEAD、标题或历史版本数字推定。

| 对象                               | 规范                                     | 示例                                           |
| ---------------------------------- | ---------------------------------------- | ---------------------------------------------- |
| 仓库 VERSION                       | `<major>.<minor>.<patch>`                | `0.18.8`                                       |
| 正式 tag / manifest releaseVersion | `pantheon-base-v<major>.<minor>.<patch>` | `pantheon-base-v0.18.8`                        |
| GitHub Release 标题                | `Pantheon Base v<major>.<minor>.<patch>` | `Pantheon Base v0.18.8`                        |
| Release Line                       | `release/<major>.<minor>`                | `release/0.18`                                 |
| Foundation bundle                  | `foundation-release-<tag>.tgz`           | `foundation-release-pantheon-base-v0.18.8.tgz` |

当前正式版本以 [VERSION](../VERSION)、[GitHub Latest](https://github.com/duanxldragon/pantheon-base/releases/latest) 和匹配的不可变资产为准。Release Line 是兼容性元数据；仓库持续开发分支为 `main`。

预发布 tag 可使用已有的 `-alpha.N`、`-beta.N`、`-rc.N` 格式，标题保留完整后缀，例如 `Pantheon Base v0.19.0-rc.1`。消费仓的稳定版入口只接受已发布、非 draft、非 prerelease 的正式版本。

publisher 对上述预发布显式传入 `--prerelease` 和 `--latest=false`。稳定版创建保留 GitHub 原有默认行为；发行或历史补录后仍须复核并固定正确的 Latest。本次标题规范化没有引入稳定版上传完成后的自动推广策略。

## 2. 历史命名的处理

历史 `base-v*`、`backend/v*`、裸 `v*` 等标签属于旧命名体系。它们保留原 tag、commit、资产和发布说明，展示时用 `Legacy` 标识原始身份，不转换为当前 Foundation 版本号。

- 当前 Foundation 发行只使用 `pantheon-base-vX.Y.Z`；发布脚本拒绝旧前缀、裸版本及不完整版本号。
- 历史版本的数字大小不参与当前 Foundation 版本排序；旧 `v1.0` 如存在，不表示当前 Foundation 已发布 `pantheon-base-v1.0.0`。
- 历史版本不会仅为整理名称而改为 prerelease，也不会重新打标签、删除标签或覆盖资产。
- GitHub Latest 必须指向当前经过验证的稳定 Foundation 发行，不能由历史命名或后台补录自动决定。
- GitHub 标题整理仅改变展示名；发布说明文件、release notes 正文和不可变发行身份继续保留。历史发布正文的补充说明必须明确为命名说明，不能改写当时验证结论。

例如历史 `backend/v0.13.1` 可显示为 `Legacy backend/v0.13.1`；当前 `pantheon-base-v0.18.8` 显示为 `Pantheon Base v0.18.8`。两者不构成同一版本序列的重编号。

## 3. 版本号决策

| 升级类型 | 触发条件                                                            |
| -------- | ------------------------------------------------------------------- |
| Major    | 基础契约、共享 API 或消费方式发生不兼容变更，需要明确迁移和回滚方案 |
| Minor    | 向后兼容的新共享能力、治理增强、架构或性能优化                      |
| Patch    | 安全修复、缺陷修复、兼容性补丁和文档修正                            |

先明确消费仓影响，再决定版本号。`pantheon-base-v1.0.0` 是未来可用的规范身份示例，不是对历史裸 `v1.0` 的映射，也不是当前发布声明。

## 4. 发布前置条件

- 目标 commit 已固定；VERSION、manifest、release notes、upgrade notes 和 consumer impact 一致。
- CI、Quality、Security、Actionlint、Full Smoke、SonarCloud 及 Release Gate 等适用门禁已通过。
- 高风险共享改动完成独立 review；涉及 UI 时保留渲染和状态验证证据。
- 发布工作树干净；已有 tag 如存在，必须解析到同一个目标 commit。
- 发行资产通过 SHA256、manifest、归档身份和凭证检查。
- task packet 和 evidence 链路完整，未通过的外部或人工门禁显式保留。

已发布版本不可重复发布。修复发行内容时创建新的 patch 版本，不覆盖旧 bundle、repo.tar 或校验和。

## 5. 标准发布入口

通过 [Foundation Release Cut workflow](../.github/workflows/foundation-release-cut.yml) 或仓库脚本执行。workflow 要求维护者预先创建绑定目标 commit 的 tag；本地 publisher 在权限满足时可创建并推送 tag。两条路径都核验版本、目标 commit、门禁和资产。

以下在 Base 根目录执行。变量必须先设置为已确认的未发布版本、兼容性 line、固定 commit 和实际说明目录；不能把已发布的 `pantheon-base-v0.18.8` 当作新发行重跑。

```powershell
npm run release:foundation:cut -- `
  --release-version $taskReleaseVersion `
  --release-line $taskReleaseLine `
  --base-commit $taskTargetCommit `
  --release-notes (Get-Content -Raw -LiteralPath (Join-Path $taskNotesDir 'release-notes.md')) `
  --upgrade-notes (Get-Content -Raw -LiteralPath (Join-Path $taskNotesDir 'upgrade-notes.md')) `
  --consumer-impact (Get-Content -Raw -LiteralPath (Join-Path $taskNotesDir 'consumer-impact.md'))

node scripts/foundation-release/check-release-assets.mjs `
  --release-version $taskReleaseVersion --dist dist/foundation-releases --local-only

npm run release:foundation:publish -- `
  --release-version $taskReleaseVersion `
  --release-line $taskReleaseLine `
  --target-commit $taskTargetCommit `
  --repo duanxldragon/pantheon-base --dry-run
```

`--dry-run` 核验并生成本地发布计划，不创建远端 tag 或 GitHub Release。审阅计划和门禁后，在已获发行授权的环境用相同参数移除 `--dry-run` 发布；直接 `gh release create` 不替代这些检查。

每个正式发行包含四个资产：Foundation bundle、bundle 的 `.sha256`、`repo.tar` 和 `repo.tar.sha256`。发布后重新核验远端 tag commit、资产校验和、manifest、三个说明正文和 GitHub Latest 指向。

## 6. CHANGELOG 与发布说明

[CHANGELOG.md](../CHANGELOG.md) 使用 [Keep a Changelog](https://keepachangelog.com/) 格式。开发变更进入 `[Unreleased]`，发布时归入实际版本章节。

发布准备材料和 `releases/<tag>/` 中的 manifest、release notes、upgrade notes、consumer impact 与目标 commit 绑定。已冻结的发布说明只追加新版本，详见 [发布说明索引](./release-notes/README.md)。GitHub 展示标题统一为 `Pantheon Base vX.Y.Z`，内容描述放在正文中。

## 7. 消费仓升级与回滚

升级在 Ops 根目录执行，以正式 GitHub 资产为来源，不读取本地 Base 目录或比较 Base/Ops Git 工作树。

```powershell
npm run fetch:base-release -- --release pantheon-base-v0.18.8
npm run rebuild:from-base -- --target .tmp/clean-base-overlay
node scripts/business-overlay/check-business-overlay.mjs --root .tmp/clean-base-overlay
npm run check:shared-drift -- --root .tmp/clean-base-overlay
```

继续执行完整 CGO race、前端和业务验证，保留原工作树备份并审阅安装计划后再替换活动树。活动 lock、overlay report 和继承 marker 必须匹配实际安装版本；验证候选不能单独证明活动升级完成。

回滚同样从指定的旧正式版本资产重建并验证，不能只编辑继承文档或修改 lock 的版本字符串。生产数据迁移和回滚须遵守独立备份、演练及授权门禁。

## 8. 参考

- [Foundation Release Model](./designs/FOUNDATION_RELEASE_MODEL.md)
- [Platform Contract](./contracts/PLATFORM_CONTRACT.md)
- [Semantic Versioning](https://semver.org/)
- [发布说明索引](./release-notes/README.md)
