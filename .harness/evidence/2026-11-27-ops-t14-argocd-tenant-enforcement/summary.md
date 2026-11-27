# T14 ArgoCD 候选验证

- 最新 Ops `02088145`、Base `v0.19.1`。主体代码取自 T14 未提交工作树，候选源码 tree 为 `c14a380e`。
- 13 个变更文件局限于业务 ArgoCD、测试和业务租户身份辅助包。沿用已合入的第一阶段数据迁移。其他模块留在原开发树。
- 真实 UTC MySQL、独立 Redis 与 Windows CGO 定向 race 通过，40 条通过事件。
- 当前完整 race、独立复核、托管检查和外部 provider 验收分别留痕。

### 候选验证结论

- 业务边界、GORM SQL 约束、干净重建、共享漂移与 59 项 overlay 测试通过。
- Windows 完整 `go test -race ./...` 达到 44 分钟 10 秒的本次监督时限，退出 124。输出已记录 16 个通过包，后续包结果未形成。
- 完整 race 日志 SHA-256 为 `17cf36a48ad4eef7cbcd6c14fa04efa507cf158593d4bcbbc1545994f8e9436f`。候选保留在本地，独立评审和 GitHub 交付仍开放。

### 2026-11-27 串行全量复验

- Windows MinGW CGO、隔离 UTC MySQL 与两套独立 Redis 完成 `go test -race ./...`。78 个测试包通过，15 个无测试包，exit 0，耗时 450 秒。
- 本次使用临时 MySQL 实例，关闭 redo 和 doublewrite 加速反复建库。外部 ArgoCD provider 与生产耐久性仍独立验收。
- 验证树 `32bf1732be751abdeda5e532546a7ab633468f0f`，日志 SHA-256 `2742e1a06e23211440e23e4f1a95284c4bbd8a78e15374fb1dc35aeda71600c0`。高风险独立评审和 GitHub 必过检查仍开放。

### Go Lint 修订后的全量验证

- 补充租户 context 及隔离数据库 fixture 导出符号的 Go 文档注释。
- 与 GitHub 同版本 golangci-lint v2.6.2 对新增差异报告 0 issues。Windows MinGW CGO 使用同样的临时 UTC MySQL 与两套独立 Redis，完整 `go test -race ./...` 通过 78 个测试包，耗时 389 秒。验证树 `b7c933eb84889aacdc095a8d0dc61c31b5e319d1`，日志 SHA-256 `7c104d85bbbc416ac4de3306932f916076badaa1e6197f3fed5a99674b158c42`。
