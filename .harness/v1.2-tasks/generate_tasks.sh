#!/bin/bash
# V1.2 任务包批量生成脚本

# 任务包定义（格式: ID|名称|工作量|优先级|依赖|简要描述）
TASKS=(
    "T2|ArgoCD Service层实现|3天|P0|T1|实现ArgoCD Server/Repository/Application的Service层和Repository层"
    "T3|ArgoCD Reconciler|3天|P0|T1,T2|实现ArgoCD状态同步Reconciler，每30秒同步Application状态"
    "T4|ArgoCD API接口|2天|P0|T2|实现ArgoCD的所有HTTP API接口和路由注册"
    "T5|ArgoCD前端UI|4天|P0|T4|实现ArgoCD前端完整UI（服务器/仓库/Application管理）"
    "T6|Velero Client封装|3天|P0|无|封装Velero CRD客户端，支持Backup/Restore/Schedule操作"
    "T7|Velero Service层实现|3天|P0|T6|实现Velero Server/BackupPlan/Backup/Restore的Service层"
    "T8|Velero Reconciler|3天|P0|T6,T7|实现Velero状态同步Reconciler，每60秒同步备份/恢复状态"
    "T9|Velero灾难演练|2天|P1|T7|实现灾难恢复演练功能（临时NS恢复+验证+清理）"
    "T10|Velero API接口|2天|P0|T7|实现Velero的所有HTTP API接口"
    "T11|Velero前端UI|4天|P0|T10|实现Velero前端完整UI（服务器/策略/备份/恢复/演练）"
    "T12|Prometheus数据源管理|2天|P0|无|实现Prometheus数据源注册、连通性测试、PromQL查询代理"
    "T14|告警评估器|3天|P0|T13|实现告警评估引擎（每30秒评估规则、触发/恢复告警、计算指纹）"
    "T15|多渠道通知|3天|P0|T13|实现多渠道通知发送（Email/Slack/钉钉/企业微信/Webhook）"
    "T16|告警API接口|2天|P0|T13,T14,T15|实现告警系统的所有HTTP API接口"
    "T17|告警前端UI|4天|P0|T16|实现告警前端完整UI（数据源/规则/告警记录/通知渠道/静默）"
    "T18|文件存储管理|2天|P1|无|实现文件上传/存储/SHA-256计算/元数据管理"
    "T19|SFTP传输引擎|4天|P1|T18|实现SFTP并发传输引擎（分块传输/断点续传/进度追踪/校验）"
    "T20|文件分发调度器|2天|P1|T19|实现文件分发任务调度器（Worker池/并发控制/队列管理）"
    "T21|文件分发API接口|2天|P1|T18,T19,T20|实现文件分发的所有HTTP API接口"
    "T22|文件分发前端UI|3天|P1|T21|实现文件分发前端完整UI（文件库/任务/实时进度/事件）"
    "T23|服务实例状态联动|3天|P0|无|激活部署→服务实例和K8s→服务实例的状态联动逻辑"
    "T24|服务实例健康检查|2天|P1|T23|实现服务实例健康检查器（VM HTTP检查+K8s Workload检查）"
    "T25|服务实例前端完善|3天|P0|T23|完善服务实例前端CRUD功能和详情页展示"
    "T26|数据库迁移脚本|2天|P0|无|创建V1.2所有新增表的数据库迁移脚本（22张表）"
    "T27|单元测试|8天|P0|所有开发任务|为所有新增代码编写单元测试，覆盖率>70%"
    "T28|集成测试|5天|P0|所有开发任务|编写ArgoCD/Velero/告警/文件分发的端到端集成测试"
    "T29|真机验收|3天|P0|T28|在真实环境验收所有V1.2功能"
    "T30|文档编写|5天|P0|所有功能|编写V1.2用户手册、API文档、运维手册、发布说明"
)

# 创建任务包目录
mkdir -p .harness/v1.2-tasks

# 生成每个任务包
for task in "${TASKS[@]}"; do
    IFS='|' read -r id name days priority deps desc <<< "$task"

    filename=".harness/v1.2-tasks/${id}_$(echo $name | tr '[:upper:]' '[:lower:]' | tr ' ' '_' | tr -d '()').md"

    echo "生成任务包: $filename"

    cat > "$filename" << EOF
# 任务包 $id: $name

**任务ID**: $id
**任务名称**: $name
**估计工作量**: $days
**优先级**: $priority
**负责人**: [待分配]
**状态**: 未开始

---

## 1. 任务目标

$desc

---

## 2. 前置依赖

### 依赖任务
$(if [ "$deps" != "无" ] && [ "$deps" != "所有开发任务" ]; then
    IFS=',' read -ra DEP_ARRAY <<< "$deps"
    for dep in "\${DEP_ARRAY[@]}"; do
        echo "- [ ] $dep"
    done
else
    echo "- 无（基础任务）"
fi)

### 依赖资源
- [ ] 待补充

### 前置知识
- 待补充

---

## 3. 功能需求

### 核心功能
1. 待补充
2. 待补充
3. 待补充

### 非功能需求
- **性能**: 待补充
- **安全**: 待补充
- **可靠性**: 待补充

---

## 4. 技术规格

### 4.1 数据模型
\`\`\`sql
-- 待补充
\`\`\`

### 4.2 核心接口
\`\`\`go
// 待补充
\`\`\`

---

## 5. 实现指南

### 5.1 目录结构
\`\`\`
待补充
\`\`\`

### 5.2 关键步骤
1. 待补充
2. 待补充

---

## 6. 测试要求

### 6.1 单元测试
**必须覆盖**: 待补充
**覆盖率要求**: >70%

### 6.2 集成测试
**测试场景**: 待补充

---

## 7. 验收标准

### 7.1 代码质量
- [ ] \`go build\` 编译通过
- [ ] \`go test\` 测试通过
- [ ] \`go vet\` 无告警

### 7.2 功能完整性
- [ ] 核心功能完成

---

## 8. 交付物清单

### 必须交付
- [ ] 源代码
- [ ] 单元测试
- [ ] 实现说明文档

---

## 9. 风险与注意事项

### 技术风险
- 待补充

### 注意事项
- 待补充

---

## 10. 参考资料

- 待补充

---

## 11. 变更记录

| 日期 | 变更内容 | 变更人 |
|------|---------|--------|
| 2026-08-28 | 创建任务包 | 脚本生成 |

---

## 12. 问题追踪

### 待解决问题
- [ ] 待补充

### 已解决问题
(暂无)
EOF

done

echo "所有任务包生成完成！"
echo "已生成文件列表:"
ls -1 .harness/v1.2-tasks/*.md
