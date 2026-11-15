# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-pages.spec.ts >> Deploy business module smoke >> deploy template supports script steps and task params render in detail
- Location: tests\smoke\business\deploy\deploy-pages.spec.ts:486:3

# Error details

```
Test timeout of 30000ms exceeded.
```

# Page snapshot

```yaml
- generic [ref=e3]:
    - complementary [ref=e4]:
        - generic [ref=e5]:
            - generic [ref=e6]:
                - generic [ref=e7]: P
                - generic [ref=e8]: Pantheon Base
            - menu [ref=e12]
    - generic [ref=e13]:
        - generic [ref=e14]:
            - generic [ref=e15]:
                - button "切换导航" [ref=e16] [cursor=pointer]
                - list [ref=e20]:
                    - listitem [ref=e21]: 首页
                    - listitem [ref=e25]: 任务模板
            - generic [ref=e26]:
                - button "平台功能台" [ref=e28]:
                    - generic [ref=e31]: 搜索菜单、页面、功能…
                    - generic [ref=e32]: Ctrl K
                - button "平台偏好" [ref=e34] [cursor=pointer]
                - 'button "用户: 用户" [ref=e39] [cursor=pointer]':
                    - generic [ref=e40]: 用
                    - generic [ref=e42]: 用户
        - tablist "已打开页面" [ref=e44]
        - main [ref=e45]:
            - generic [ref=e47]:
                - generic [ref=e52]: 加载失败
                - generic [ref=e53]: 当前区块暂时无法返回数据，请重试或稍后再查看。
                - button "重试" [ref=e55] [cursor=pointer]
        - generic [ref=e56]: Pantheon Platform · Enterprise Admin Foundation
```
