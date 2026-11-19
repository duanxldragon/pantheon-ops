# BizScope 历史子要求裁决

日期：2026-11-17；独立code-reviewer：/root/history_closeout_review。

后继 Active BUSINESS_BIZSCOPE_MODULE_DESIGN.md（updated_at=2026-07-18）§6第170行明确列表路由list/详情view；§5第122–126行规定GET/list与options用list、GET/:id与主机读取用view。后继设计明确替代6/20中BizScope入口“不复用list”的子要求。保留目前frontend/backend seed契约，不机械替换权限，不新增page key。6/20其他数据范围要求继续开放；本裁决不替代整包验收。

同代理初次测试审查发现MEDIUM：commandFixture无真实TargetIDs且host已Failed，不能用其无副作用证明可写请求越权。主lane已改用previewFixture并新增范围内成功对照，最终测试与窄范围复审尚须完成。初次审查仅只读，未启动CLI/测试或改文件。
