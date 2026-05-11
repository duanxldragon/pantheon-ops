# 上传与存储设计

更新时间：2026-04-29

类型：Design
归属层：system/config
状态：Active

本文定义 `system/config -> upload` 子域与平台统一上传能力的边界。

它重点回答：

- 上传能力为什么归属 `system/config`
- 配置、运行时写文件、对象访问 URL 各自由谁负责
- 本地存储和 S3-compatible 存储如何切换
- 验收时应该看哪些风险点

---

## 1. 模块定位

上传能力是平台公共基础能力。

其中：

- 配置归属 `system/config`
- 运行时文件处理归属平台公共包
- 业务模块通过统一入口复用，不自行发明上传协议

当前不是每个模块各自上传，而是：

- 一个统一上传入口
- 一套统一配置
- 一套统一大小、类型、路径和 URL 生成规则

---

## 2. 边界

### 2.1 upload 负责

- 上传配置读取
- 文件大小限制
- 文件类型白名单
- 本地落盘路径解析
- 本地文件访问 URL 生成
- S3-compatible 对象存储上传
- 统一上传 API

### 2.2 upload 不负责

- 业务文件元数据业务语义
- 各模块自己的附件生命周期
- 业务审批与归档规则

### 2.3 协作边界

- `system/config -> setting` 负责配置值
- `backend/pkg/upload` 负责运行时处理
- `frontend` 负责通过统一接口上传
- 业务模块只负责声明 `scope` 并消费返回结果

---

## 3. 当前运行时能力

当前统一上传入口：

- `POST /api/v1/system/upload`

当前本地文件访问入口：

- `GET /api/v1/system/upload/files/*filepath`

当前前端已接入：

- 个人中心头像上传
- 用户管理头像上传
- 通用 `uploadSystemFile(file, scope)` 请求封装

---

## 4. 配置模型

上传配置来源于 `system_setting`，归属 `system/config -> upload` 分组。

当前核心配置项：

- `upload.storage_driver`
- `upload.max_file_size`
- `upload.allowed_types`
- `upload.local_path`
- `upload.public_base_url`
- `upload.s3_endpoint`
- `upload.s3_bucket`
- `upload.s3_region`
- `upload.s3_access_key_id`
- `upload.s3_secret_access_key`

### 4.1 当前驱动

支持：

- `local`
- `s3`

兼容：

- `s3-compatible` 可归一到 `s3`

### 4.2 当前默认语义

- `local`：文件写入本地路径，再通过平台路由访问
- `s3`：文件写入对象存储，直接返回对象访问 URL 或拼装后的公开 URL

---

## 5. 本地存储设计

### 5.1 路径约束

本地模式下必须受：

- `upload.local_path`

约束。

要求：

- 不允许目录穿越
- 不允许解析到配置根目录之外
- 生成的对象键和物理路径必须可追溯

### 5.2 URL 生成

本地模式默认通过平台文件访问入口暴露：

- `/api/v1/system/upload/files/*filepath`

如果配置了：

- `upload.public_base_url`

则允许按公开基地址生成最终 URL。

---

## 6. S3-compatible 存储设计

### 6.1 需要的配置

- endpoint
- bucket
- region
- access key id
- secret access key

### 6.2 运行时要求

- 初始化 bucket
- 写入对象
- 生成对象访问 URL

### 6.3 错误语义

至少应明确区分：

- endpoint 未配置
- bucket 未配置
- credentials 未配置
- endpoint 非法
- bucket 初始化失败
- 上传失败

---

## 7. 文件校验规则

上传入口至少做四类校验：

1. 文件是否存在
2. 文件大小是否超限
3. 文件扩展名/类型是否允许
4. 目标路径是否安全

这四类校验都不应散落在业务模块中重复实现。

---

## 8. scope 设计

上传接口支持通过 `scope` 区分业务场景。

例如：

- `general`
- `profile/avatar`

原则：

- `scope` 用于标识场景，不等于权限模型
- 不能把 `scope` 直接当成安全边界
- 真正安全仍要靠登录态、接口权限和上传配置校验

---

## 9. 安全与审计

### 9.1 安全要求

- 上传接口至少要求登录态
- 文件大小和类型不允许由前端单方面控制
- 本地路径必须防目录穿越
- 敏感对象存储凭据必须加密存储

### 9.2 审计要求

上传动作应进入统一操作审计。

至少记录：

- 上传人
- scope
- 文件名
- 文件大小
- 驱动类型
- 结果状态

目标：

- 能回溯“谁上传了什么，写到哪里，是否成功”

---

## 10. 前端消费约束

前端不应：

- 拼接本地文件路径
- 自己决定允许的文件大小
- 自己决定允许的后缀类型

前端应：

- 通过统一上传接口发送文件
- 使用后端返回的对象 URL / objectKey
- 将失败提示仍视为 `message key` 翻译链路的一部分

---

## 11. 验收要求

上传能力后续至少固定检查：

### 11.1 基础链路

- `/system/setting` 中 upload 分组可读可写
- `POST /system/upload` 可工作
- 本地驱动下文件 URL 可访问
- S3 驱动下对象 URL 生成正确

### 11.2 校验链路

- 超大小文件被拒绝
- 非允许类型文件被拒绝
- 非法路径被拒绝
- S3 缺配置时错误语义明确

### 11.3 消费链路

- 个人中心头像上传可用
- 用户管理头像上传可用
- 上传结果不会回退成英文硬编码错误提示

### 11.4 审计链路

- 上传动作有统一审计
- 失败场景也能留下可解释记录

---

## 12. 当前结论

上传能力已经不是“设置页里顺带放几个配置项”。

它是：

- 配置归属 `system/config`
- 运行时归属平台公共能力
- 被多个模块复用的统一上传子域

后续必须继续保持这一点，不能回退成各模块各写一套上传逻辑。
