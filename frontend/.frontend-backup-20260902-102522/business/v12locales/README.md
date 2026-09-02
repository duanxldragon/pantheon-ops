# V1.2 业务 i18n 汇总 locale

本目录是 V1.2 新增业务模块（filedist、cicd/argocd、k8s 扩展、observability、
service catalog）的 i18n key 汇总清单。历史上这些 key 由生成器产物
`frontend/src/i18n/resources/business/*.ts` 手工维护；迁移到 foundation
v0.10.25 消费流程时抽回模块 locale，使 `rebuild:from-base` 的
`generateBusinessFallbacks` 步骤可以完整重现 key 集合。
