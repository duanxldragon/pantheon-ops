# Release Notes — base-v0.9.1

发布日期：2026-07-26
Base commit：`71f0bedafd60639f68197310cea8b01833b1d0b2`（main）
Release line：`0.9`
性质：过渡发行版（interim foundation release），用于解锁 pantheon-ops 业务开发；base-v1.0.0 仍按 V1 冻结收尾计划（P0–P6）推进。

## 与 base-v0.9.0 的关系

base-v0.9.0（2026-07-22，`0f9a803`）已切但未被任何消费仓采用，且三份 notes 为空、不含 7-25 之后合并的质量清零成果。base-v0.9.1 取代 v0.9.0 作为 0.9 线的实际消费版本。

## 主要变更主题（相对 pantheon-ops 当前锁定的 base-v0.8.11，2026-07-08）

- **安全强化**：refresh token 轮换、SecureAction 运行时、CSV 注入防护、上传安全修复；GitHub 安全告警（CodeQL/Dependabot/Secret scanning）三项归零（2026-07-25）。
- **代码质量清零**：SonarCloud 存量 812 → 128（剩余均为 S3776 认知复杂度等 code smell，已移交维护者裁量）；BUG / VULNERABILITY 双零；main 分支 Quality Gate OK（2026-07-25 存档）。
- **审计与 i18n**：审计操作标题 i18n 化（约 86 处）。
- **CI/门禁增强**：golangci-lint 新代码门禁（quality.yml PR/merge_group `--new-from-rev`）、S8545 action 化、encoding/UI 双机械门禁、visual contract 检查器。
- **重复率与常量化治理**：S1192 字符串常量化（跨约 53 文件）、CPD 治理。

> 精确变更清单见同目录 `changes-since-v0.8.11.txt`（`git log base-v0.8.11..base-v0.9.1 --oneline` 生成）。

## 验证结论

- GitHub required checks：main 持续绿（发布前人工核验）。
- CodeQL：无未解释的 error/critical 告警（2026-07-25 归零存档）。
- SonarCloud：**平台维护中不可达**。采用 2026-07-25 存档证据（QG OK、BUG/VULN 双零）替代实时查询，剩余 128 项 code smell 不阻塞过渡版。豁免记录：`.harness/evidence/release-base-v0.9.1/sonarcloud-gate-exemption.md`。
- release-gate.yml 本次未 dispatch（Gate 3 因 SonarCloud 维护必然失败）；SonarCloud 恢复后应在 `71f0bed` 上补跑一次做回填验证。

## 已知未包含项（留给 base-v1.0.0）

- V1 审查发现的 12 项后端修复（token 吊销、data-scope 孤儿、MFA secret 残留、优雅停机等）。
- 冒烟 16 项流程缺口补齐。
- css:S4666 检查器重锚与 CSS 漂移修复（现渲染已被档案截图旁证为验收意图）。
- SonarCloud 剩余 128 项 code smell 清零。
