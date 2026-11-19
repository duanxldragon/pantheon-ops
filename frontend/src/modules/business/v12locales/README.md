# V1.2 业务 i18n 汇总 locale

本目录是 V1.2 新增业务模块（filedist、cicd/argocd、k8s 扩展、observability、
service catalog）的 i18n key 汇总清单。历史上这些 key 由生成器产物
`frontend/src/i18n/resources/business/*.ts` 手工维护；迁移到 foundation
v0.10.25 消费流程时抽回模块 locale，使 `rebuild:from-base` 的
`generateBusinessFallbacks` 步骤可以完整重现 key 集合。

E01/E02 的九个生命周期/同步观察 canonical 错误及 E01 新增的 44 个 UI key 提供中文和英文文案。
fr-FR、ja-JP、ko-KR 的这些新增 key 暂使用明确的英文 fallback，尚未声称完成对应语言翻译；正式 overlay 重建会装配这些声明式资源。
