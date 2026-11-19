# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\service\ServiceCatalog.nav.spec.ts >> Business navigation repair >> every business parent/leaf entry loads the expected main content
- Location: tests\smoke\business\service\ServiceCatalog.nav.spec.ts:229:3

# Error details

```
Test timeout of 180000ms exceeded.
```

# Page snapshot

```yaml
- generic [ref=f31e3]:
    - complementary [ref=f31e4]:
        - generic [ref=f31e5]:
            - generic [ref=f31e6]:
                - generic [ref=f31e7]: P
                - generic [ref=f31e8]: Pantheon Base
            - menu [ref=f31e12]:
                - generic [ref=f31e13]:
                    - menuitem [ref=f31e14] [cursor=pointer]:
                        - button "工作台" [ref=f31e15]
                    - button "CMDB" [ref=f31e25] [cursor=pointer]
                    - button "业务域" [ref=f31e40] [cursor=pointer]
                    - button "服务目录" [ref=f31e55] [cursor=pointer]
                    - button "安装部署" [ref=f31e70] [cursor=pointer]
                    - menuitem [ref=f31e79] [cursor=pointer]:
                        - button "文件分发" [ref=f31e80]
                    - button "Kubernetes" [ref=f31e92] [cursor=pointer]
                    - menuitem [ref=f31e101] [cursor=pointer]:
                        - button "ArgoCD 应用交付" [ref=f31e102]
                    - generic [ref=f31e108]:
                        - button "可观测中心" [ref=f31e111] [cursor=pointer]
                        - generic [ref=f31e120]:
                            - menuitem [ref=f31e121] [cursor=pointer]:
                                - button "监控中心" [ref=f31e124]
                            - menuitem [ref=f31e133] [cursor=pointer]:
                                - button "告警中心" [ref=f31e136]
                            - menuitem [ref=f31e142] [cursor=pointer]:
                                - button "日志中心" [ref=f31e145]
                            - menuitem [ref=f31e152] [cursor=pointer]:
                                - button "链路追踪" [active] [ref=f31e155]
                    - button "访问控制" [ref=f31e164] [cursor=pointer]
                    - button "组织架构" [ref=f31e178] [cursor=pointer]
                    - button "安全审计" [ref=f31e193] [cursor=pointer]
                    - button "低代码平台" [ref=f31e205] [cursor=pointer]
                    - button "平台配置" [ref=f31e217] [cursor=pointer]
    - generic [ref=f31e226]:
        - generic [ref=f31e227]:
            - generic [ref=f31e228]:
                - button "切换导航" [ref=f31e229] [cursor=pointer]
                - list [ref=f31e233]:
                    - listitem [ref=f31e234]: 首页
                    - listitem [ref=f31e238]: 可观测中心
                    - listitem [ref=f31e242]: 链路追踪
            - generic [ref=f31e243]:
                - button "平台功能台" [ref=f31e245]:
                    - generic [ref=f31e248]: 搜索菜单、页面、功能…
                    - generic [ref=f31e249]: Ctrl K
                - button "通知中心" [ref=f31e251] [cursor=pointer]
                - button "平台偏好" [ref=f31e255] [cursor=pointer]
                - 'button "用户: Administrator" [ref=f31e260] [cursor=pointer]':
                    - generic [ref=f31e261]: A
                    - generic [ref=f31e263]:
                        - generic [ref=f31e264]: Administrator
                        - generic [ref=f31e265]: admin
        - tablist "已打开页面" [ref=f31e266]:
            - tab "工作台" [ref=f31e267] [cursor=pointer]
            - tab "存储管理" [ref=f31e271] [cursor=pointer]
            - tab "Velero 备份" [ref=f31e273] [cursor=pointer]
            - tab "ArgoCD 应用交付" [ref=f31e275] [cursor=pointer]
            - tab "监控中心" [ref=f31e277] [cursor=pointer]
            - tab "告警中心" [ref=f31e279] [cursor=pointer]
            - tab "日志中心" [ref=f31e281] [cursor=pointer]
            - tab "链路追踪" [selected] [ref=f31e283] [cursor=pointer]
        - main [ref=f31e285]:
            - generic [ref=f31e288]:
                - generic [ref=f31e290]:
                    - generic [ref=f31e291]:
                        - generic [ref=f31e292]: 运维平台 / 链路追踪
                        - generic [ref=f31e293]: 链路追踪
                    - generic [ref=f31e295]:
                        - generic [ref=f31e296]:
                            - generic [ref=f31e297]: 数据源
                            - strong [ref=f31e298]: "0"
                        - generic [ref=f31e299]:
                            - generic [ref=f31e300]: 命中数
                            - strong [ref=f31e301]: "0"
                - search "筛选" [ref=f31e303]:
                    - generic [ref=f31e305]:
                        - combobox "数据源" [ref=f31e307] [cursor=pointer]:
                            - generic [ref=f31e309]:
                                - textbox "数据源"
                                - generic [ref=f31e310]: 数据源
                        - generic [ref=f31e316]:
                            - button "搜索" [ref=f31e318] [cursor=pointer]
                            - button "停止查询" [disabled] [ref=f31e322]
                - generic [ref=f31e324]:
                    - combobox "编辑方式" [disabled] [ref=f31e327]:
                        - generic "目录条件" [ref=f31e328]:
                            - generic [ref=f31e329]:
                                - textbox [disabled] [aria-hidden]
                                - generic [ref=f31e330]: 目录条件
                    - generic [ref=f31e335]:
                        - button "历史与视图" [ref=f31e337] [cursor=pointer]
                        - button "保存视图" [disabled] [ref=f31e341]
                - generic [ref=f31e345]:
                    - generic [ref=f31e346]:
                        - text: 开始时间（UTC）
                        - textbox "开始时间（UTC）" [ref=f31e347]: 2026-11-17T07:11
                    - generic [ref=f31e348]:
                        - text: 结束时间（UTC）
                        - textbox "结束时间（UTC）" [ref=f31e349]: 2026-11-17T07:26
                    - generic [ref=f31e350]: 单次范围最多 24 小时，响应最多 8 MiB，最多展示 100 条链路
                - generic [ref=f31e351]: 暂无可用链路追踪数据源。
        - generic [ref=f31e361]: Pantheon Platform · Enterprise Admin Foundation
```
