-- Observability Module Schema

-- Metric Sources
CREATE TABLE IF NOT EXISTS metric_sources (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL COMMENT '指标源名称',
    type VARCHAR(50) NOT NULL COMMENT '类型: prometheus, victoria-metrics',
    endpoint VARCHAR(255) NOT NULL COMMENT '指标源端点',
    credential_ref VARCHAR(100) COMMENT '凭据引用',
    business_scope_id BIGINT UNSIGNED COMMENT '业务范围ID',
    dept_id BIGINT UNSIGNED COMMENT '部门ID',
    status VARCHAR(20) NOT NULL DEFAULT 'active' COMMENT '状态: active, inactive',
    config JSON COMMENT '配置信息',
    remark TEXT COMMENT '备注',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_by VARCHAR(50) COMMENT '创建人',
    updated_by VARCHAR(50) COMMENT '更新人',
    INDEX idx_business_scope (business_scope_id),
    INDEX idx_dept (dept_id),
    INDEX idx_status (status),
    INDEX idx_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='指标源表';

-- Alert Rules
CREATE TABLE IF NOT EXISTS alert_rules (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    metric_source_id BIGINT UNSIGNED NOT NULL COMMENT '指标源ID',
    name VARCHAR(100) NOT NULL COMMENT '告警规则名称',
    business_scope_id BIGINT UNSIGNED COMMENT '业务范围ID',
    dept_id BIGINT UNSIGNED COMMENT '部门ID',
    environment VARCHAR(20) COMMENT '环境: prod, test, dev',
    promql TEXT NOT NULL COMMENT 'PromQL表达式',
    duration VARCHAR(20) COMMENT '持续时间, 如: 5m, 10m',
    severity VARCHAR(20) NOT NULL COMMENT '严重性: critical, warning, info',
    labels JSON COMMENT '标签',
    annotations JSON COMMENT '注解',
    notification_channel_ids JSON COMMENT '通知渠道ID列表',
    status VARCHAR(20) NOT NULL DEFAULT 'enabled' COMMENT '状态: enabled, disabled',
    remark TEXT COMMENT '备注',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_by VARCHAR(50) COMMENT '创建人',
    updated_by VARCHAR(50) COMMENT '更新人',
    INDEX idx_metric_source (metric_source_id),
    INDEX idx_business_scope (business_scope_id),
    INDEX idx_dept (dept_id),
    INDEX idx_environment (environment),
    INDEX idx_severity (severity),
    INDEX idx_status (status),
    FOREIGN KEY (metric_source_id) REFERENCES metric_sources(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警规则表';

-- Alert Records
CREATE TABLE IF NOT EXISTS alert_records (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    alert_rule_id BIGINT UNSIGNED NOT NULL COMMENT '告警规则ID',
    alert_rule_name VARCHAR(100) COMMENT '告警规则名称(冗余)',
    fired_at TIMESTAMP NOT NULL COMMENT '触发时间',
    resolved_at TIMESTAMP NULL COMMENT '恢复时间',
    severity VARCHAR(20) NOT NULL COMMENT '严重性: critical, warning, info',
    labels JSON COMMENT '标签',
    annotations JSON COMMENT '注解',
    notifications_sent JSON COMMENT '已发送通知记录',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_alert_rule (alert_rule_id),
    INDEX idx_fired_at (fired_at),
    INDEX idx_resolved_at (resolved_at),
    INDEX idx_severity (severity),
    INDEX idx_active_alerts (alert_rule_id, resolved_at),
    FOREIGN KEY (alert_rule_id) REFERENCES alert_rules(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警记录表';

-- Notification Channels
CREATE TABLE IF NOT EXISTS notification_channels (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL COMMENT '通知渠道名称',
    type VARCHAR(50) NOT NULL COMMENT '类型: email, dingtalk, wechat, slack',
    config JSON NOT NULL COMMENT '配置信息',
    business_scope_id BIGINT UNSIGNED COMMENT '业务范围ID',
    dept_id BIGINT UNSIGNED COMMENT '部门ID',
    status VARCHAR(20) NOT NULL DEFAULT 'active' COMMENT '状态: active, inactive',
    remark TEXT COMMENT '备注',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_by VARCHAR(50) COMMENT '创建人',
    updated_by VARCHAR(50) COMMENT '更新人',
    INDEX idx_business_scope (business_scope_id),
    INDEX idx_dept (dept_id),
    INDEX idx_type (type),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='通知渠道表';
