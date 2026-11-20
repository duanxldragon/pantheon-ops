# 业务导航矩阵 — NAVIGATION_MATRIX

日期：2026-11-16
任务：`2026-11-16-ops-navigation-repair`
工具：此表记录目标业务菜单块（Sort 1..8 连续、系统治理之前）与逐叶导航回检查矩阵。数据源：各 seed 后端 Sort / Path / RouteName / PagePerm，配套前端路由。

## 业务块总序（Sort 1..8，连续）

| #   | 侧栏标签 (zh-CN) | Top Sort | 菜单路径                    | 顶级组件/Route                    | 主要子项（Sort→页）                                                         |
| --- | ---------------- | -------- | --------------------------- | --------------------------------- | --------------------------------------------------------------------------- |
| 1   | CMDB             | 1        | `/business/cmdb`            | cmdb M                            | host(Sort1→`/host`)、group(2→`/group`)、label(3→`/label`)                   |
| 2   | 业务域           | 2        | `/business/bizscope`        | bizscope                          | 业务域 + 环境子项                                                           |
| 3   | 服务目录         | 3        | `/business/service-catalog` | `business/service/ServiceCatalog` | applications(3)、services(4)、instances(5) —— 页内 Tabs 受 URL 控制         |
| 4   | 安装部署         | 4        | `/business/deploy`          | deploy M                          | package(1)、template(2)、task(4)                                            |
| 5   | 文件分发         | 5        | `/business/filedist/files`  | `business/filedist/FiledistList`  | 单页 C                                                                      |
| 6   | Kubernetes       | 6        | `/business/k8s`             | k8s M                             | cluster、workload、service、**release(应用发布)**、ingress、storage、backup |
| 7   | ArgoCD 应用交付  | 7        | `/business/cicd/argocd`     | `business/cicd/argocd/ArgoCDList` | 单页 C（servers/repositories/applications tabs）                            |
| 8   | 可观测中心       | 8        | `/business/observability`   | observability M                   | monitoring(1)、alerts(2)、logs(3)、traces(4)                                |

> ArgoCD 标签 = **`ArgoCD 应用交付`**（GitOps 明确，与 K8s release `应用发布` 区隔，侧栏 + 页 hero 双重一致）。仅 K8s release 子项保留 `应用发布`。

## 服务目录 3 页 / 旧树 404 修复

- 收敛后唯一入口 `/business/service-catalog`（保留旧 `id159 /business/service` 的菜单 id 与全部角色绑定含 tenant_id；连续 seed 幂等；只处理 `module=business.service` 旧菜单；不删同名自定义菜单）。
- 页内 Tabs：`/applications`、`/services`、`/instances` —— `activeTab` 由 URL 派生、`onChange` 写回 URL；深链/刷新/前进后退一致。
- 回归：迁移测试（真实旧树 fixture，Codex 用真实 MySQL 跑） + Playwright `ServiceCatalog.nav.spec.ts`（确认无 `/business/service*` 404、Tab⇄URL 同步、深链 reload 保持激活 Tab）。

## 逐叶导航回检查矩阵（Codex 独立验收执行）

| 叶入口              | 期望 URL                                 | 期望 200 / 无 404 | 深链刷新 | 前进后退 |
| ------------------- | ---------------------------------------- | ----------------- | -------- | -------- |
| CMDB → 主机         | `/business/cmdb/host`                    | ✓                 | ✓        | ✓        |
| CMDB → 分组         | `/business/cmdb/group`                   | ✓                 | ✓        | ✓        |
| CMDB → 标签         | `/business/cmdb/label`                   | ✓                 | ✓        | ✓        |
| 业务域              | `/business/bizscope`                     | ✓                 | ✓        | ✓        |
| 服务目录 → 应用     | `/business/service-catalog/applications` | ✓                 | ✓        | ✓        |
| 服务目录 → 服务     | `/business/service-catalog/services`     | ✓                 | ✓        | ✓        |
| 服务目录 → 服务实例 | `/business/service-catalog/instances`    | ✓                 | ✓        | ✓        |
| 安装部署 → 包       | `/business/deploy/package`               | ✓                 | ✓        | ✓        |
| 安装部署 → 模板     | `/business/deploy/template`              | ✓                 | ✓        | ✓        |
| 安装部署 → 任务     | `/business/deploy/task`                  | ✓                 | ✓        | ✓        |
| 文件分发            | `/business/filedist/files`               | ✓                 | ✓        | ✓        |
| Kubernetes → 集群   | `/business/k8s/cluster`                  | ✓                 | ✓        | ✓        |
| Kubernetes → 负载   | `/business/k8s/workload`                 | ✓                 | ✓        | ✓        |
| Kubernetes → 服务   | `/business/k8s/service`                  | ✓                 | ✓        | ✓        |
| Kubernetes → 发布   | `/business/k8s/release`                  | ✓                 | ✓        | ✓        |
| Kubernetes → 网关   | `/business/k8s/ingress`                  | ✓                 | ✓        | ✓        |
| Kubernetes → 存储   | `/business/k8s/storage`                  | ✓                 | ✓        | ✓        |
| Kubernetes → 备份   | `/business/k8s/backup`                   | ✓                 | ✓        | ✓        |
| ArgoCD 应用交付     | `/business/cicd/argocd`                  | ✓                 | ✓        | ✓        |
| 可观测 → 监控       | `/business/observability/monitoring`     | ✓                 | ✓        | ✓        |
| 可观测 → 告警中心   | `/business/observability/alerts`         | ✓                 | ✓        | ✓        |
| 可观测 → 日志       | `/business/observability/logs`           | ✓                 | ✓        | ✓        |
| 可观测 → 链路       | `/business/observability/traces`         | ✓                 | ✓        | ✓        |

## 渲染截图（Codex 验收产物）

- 桌面 1440x900：完整侧栏业务块连续 + 服务目录 Tabs 高亮 + 全部业务叶入口。
- 移动 390x844：折叠侧栏 + 逐叶进入导航无横向滚动 / 无隐藏固定底部。
- 截图存放：由 Codex 在真实升级旧库重启后产出（本任务契约；Claude 侧无真实库/禁 localhost curl，不在此目录伪造渲染证据）。
