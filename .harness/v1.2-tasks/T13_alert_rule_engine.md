# 任务包 T13: 告警规则引擎

**任务ID**: T13  
**任务名称**: 告警规则引擎  
**估计工作量**: 4天  
**优先级**: P0  
**负责人**: [待分配]  
**状态**: 未开始  

---

## 1. 任务目标

实现告警规则的完整管理系统，包括规则存储、规则类型支持（PromQL/阈值/模板）、规则验证。

**核心输出**:
- 告警规则 CRUD 功能
- 支持 3 种规则类型
- 规则模板系统
- 规则验证功能

---

## 2. 前置依赖

### 依赖任务
- [x] T12: Prometheus 数据源管理
- [x] T26: 数据库迁移脚本

### 依赖资源
- [ ] Prometheus 测试服务器
- [ ] 数据库 (MySQL 8.0+)

### 前置知识
- PromQL 语法
- 告警规则设计模式
- Go 模板引擎

---

## 3. 功能需求

### 核心功能
1. **规则类型支持**
   - PromQL 类型: 直接编写 PromQL 表达式
   - 阈值类型: 简单配置（指标名+阈值+操作符）
   - 模板类型: 使用预定义模板 + 参数替换

2. **规则管理**
   - CreateRule: 创建告警规则
   - GetRule: 获取规则详情
   - ListRules: 查询规则列表
   - UpdateRule: 更新规则配置
   - DeleteRule: 删除规则
   - EnableRule: 启用规则
   - DisableRule: 禁用规则

3. **规则验证**
   - ValidatePromQL: 验证 PromQL 语法
   - TestRule: 试运行规则，查看匹配结果
   - ValidateThreshold: 验证阈值配置

4. **规则模板**
   - ListTemplates: 列出所有模板
   - GetTemplate: 获取模板详情
   - RenderTemplate: 渲染模板（参数替换）

### 非功能需求
- **性能**: 规则查询 <100ms
- **安全**: 业务域隔离，权限控制
- **可靠性**: 规则验证必须通过才能保存

---

## 4. 技术规格

### 4.1 数据模型

```sql
-- 告警规则表（已在 T26 创建）
CREATE TABLE biz_observability_alert_rule (
    id                      BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    datasource_id           BIGINT UNSIGNED NOT NULL COMMENT '关联数据源',
    business_scope_id       BIGINT UNSIGNED NOT NULL COMMENT '业务域归属',
    
    -- 基本信息
    rule_name               VARCHAR(128) NOT NULL COMMENT '规则名称',
    display_name            VARCHAR(128) COMMENT '显示名称',
    description             TEXT COMMENT '规则描述',
    
    -- 规则类型
    rule_type               VARCHAR(32) NOT NULL COMMENT 'promql/threshold/template',
    
    -- PromQL 类型规则
    promql_expr             TEXT COMMENT 'PromQL 表达式',
    
    -- 阈值类型规则
    metric_name             VARCHAR(255) COMMENT '指标名称',
    threshold_operator      VARCHAR(16) COMMENT '>/>=/</<=/<>/==',
    threshold_value         DECIMAL(20, 4) COMMENT '阈值',
    aggregation             VARCHAR(32) COMMENT 'avg/sum/min/max/count',
    
    -- 模板类型规则
    template_id             BIGINT UNSIGNED COMMENT '关联规则模板',
    template_params         JSON COMMENT '模板参数',
    
    -- 评估配置
    evaluation_interval     INT DEFAULT 30 COMMENT '评估间隔(秒)',
    for_duration            INT DEFAULT 60 COMMENT '持续时长(秒)',
    
    -- 标签与注解
    labels                  JSON COMMENT '告警标签',
    annotations             JSON COMMENT '告警注解',
    
    -- 目标范围
    target_type             VARCHAR(32) COMMENT 'host/k8s_cluster/namespace/all',
    target_filters          JSON COMMENT '目标过滤条件',
    
    -- 严重级别
    severity                VARCHAR(16) NOT NULL COMMENT 'critical/warning/info',
    
    -- 通知配置
    notify_channels         JSON COMMENT '通知渠道 ["email","slack"]',
    notify_users            JSON COMMENT '通知用户ID列表',
    notify_groups           JSON COMMENT '通知用户组',
    
    -- 状态
    enabled                 BOOLEAN DEFAULT TRUE,
    last_evaluation_at      TIMESTAMP NULL,
    last_evaluation_result  VARCHAR(32) COMMENT 'ok/firing/pending/error',
    last_error              TEXT,
    
    created_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_by              VARCHAR(64),
    
    INDEX idx_datasource (datasource_id),
    INDEX idx_business_scope (business_scope_id),
    INDEX idx_enabled (enabled),
    INDEX idx_severity (severity),
    
    FOREIGN KEY (datasource_id) REFERENCES biz_observability_prometheus_datasource(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警规则';

-- 告警规则模板表
CREATE TABLE biz_observability_alert_rule_template (
    id                      BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    template_name           VARCHAR(128) NOT NULL,
    display_name            VARCHAR(128),
    description             TEXT,
    category                VARCHAR(64) COMMENT '分类: host/k8s/app/custom',
    
    -- 模板定义
    promql_template         TEXT COMMENT 'PromQL模板 (支持变量替换)',
    default_threshold       DECIMAL(20, 4),
    default_operator        VARCHAR(16),
    default_for_duration    INT DEFAULT 60,
    default_severity        VARCHAR(16),
    
    -- 参数定义
    parameters              JSON COMMENT '参数Schema',
    
    -- 示例
    example_usage           TEXT COMMENT '使用示例',
    
    -- 内置标识
    is_builtin              BOOLEAN DEFAULT FALSE COMMENT '内置模板',
    
    created_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    created_by              VARCHAR(64),
    
    UNIQUE KEY uk_template_name (template_name),
    INDEX idx_category (category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警规则模板';
```

### 4.2 核心接口

```go
// backend/modules/business/observability/alert/rule_service.go

package alert

import (
    "context"
    "encoding/json"
    "fmt"
    "strings"
    
    "gorm.io/gorm"
)

// RuleService manages alert rules
type RuleService struct {
    db               *gorm.DB
    datasourceSvc    *DatasourceService
}

// NewRuleService creates a rule service
func NewRuleService(db *gorm.DB, datasourceSvc *DatasourceService) *RuleService {
    return &RuleService{
        db:            db,
        datasourceSvc: datasourceSvc,
    }
}

// CreateRule creates a new alert rule
func (s *RuleService) CreateRule(ctx context.Context, req *CreateRuleRequest) (*AlertRule, error) {
    // 1. 验证数据源
    datasource, err := s.datasourceSvc.GetByID(ctx, req.DatasourceID)
    if err != nil {
        return nil, fmt.Errorf("datasource not found: %w", err)
    }
    
    // 2. 构建规则
    rule := &AlertRule{
        DatasourceID:      req.DatasourceID,
        BusinessScopeID:   req.BusinessScopeID,
        RuleName:          req.RuleName,
        DisplayName:       req.DisplayName,
        Description:       req.Description,
        RuleType:          req.RuleType,
        EvaluationInterval: req.EvaluationInterval,
        ForDuration:       req.ForDuration,
        Severity:          req.Severity,
        Enabled:           req.Enabled,
    }
    
    // 3. 根据类型设置规则
    switch req.RuleType {
    case "promql":
        rule.PromQLExpr = req.PromQLExpr
        // 验证 PromQL
        if err := s.validatePromQL(ctx, datasource, req.PromQLExpr); err != nil {
            return nil, fmt.Errorf("invalid promql: %w", err)
        }
        
    case "threshold":
        rule.MetricName = req.MetricName
        rule.ThresholdOperator = req.ThresholdOperator
        rule.ThresholdValue = req.ThresholdValue
        rule.Aggregation = req.Aggregation
        
    case "template":
        rule.TemplateID = req.TemplateID
        paramsJSON, _ := json.Marshal(req.TemplateParams)
        rule.TemplateParams = paramsJSON
        
        // 渲染模板生成 PromQL
        promql, err := s.renderTemplate(ctx, req.TemplateID, req.TemplateParams)
        if err != nil {
            return nil, fmt.Errorf("render template failed: %w", err)
        }
        rule.PromQLExpr = promql
        
    default:
        return nil, fmt.Errorf("unsupported rule type: %s", req.RuleType)
    }
    
    // 4. 设置标签和注解
    if req.Labels != nil {
        labelsJSON, _ := json.Marshal(req.Labels)
        rule.Labels = labelsJSON
    }
    if req.Annotations != nil {
        annotationsJSON, _ := json.Marshal(req.Annotations)
        rule.Annotations = annotationsJSON
    }
    
    // 5. 设置通知配置
    if req.NotifyChannels != nil {
        channelsJSON, _ := json.Marshal(req.NotifyChannels)
        rule.NotifyChannels = channelsJSON
    }
    
    // 6. 设置目标范围
    rule.TargetType = req.TargetType
    if req.TargetFilters != nil {
        filtersJSON, _ := json.Marshal(req.TargetFilters)
        rule.TargetFilters = filtersJSON
    }
    
    // 7. 保存规则
    if err := s.db.Create(rule).Error; err != nil {
        return nil, err
    }
    
    return rule, nil
}

// GetRule retrieves a rule by ID
func (s *RuleService) GetRule(ctx context.Context, id uint64) (*AlertRule, error) {
    var rule AlertRule
    if err := s.db.First(&rule, id).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, fmt.Errorf("rule not found")
        }
        return nil, err
    }
    return &rule, nil
}

// ListRules lists rules with filters
func (s *RuleService) ListRules(ctx context.Context, req *ListRulesRequest) ([]*AlertRule, int64, error) {
    query := s.db.Model(&AlertRule{})
    
    // 业务域过滤
    if req.BusinessScopeID != 0 {
        query = query.Where("business_scope_id = ?", req.BusinessScopeID)
    }
    
    // 启用状态过滤
    if req.Enabled != nil {
        query = query.Where("enabled = ?", *req.Enabled)
    }
    
    // 严重级别过滤
    if req.Severity != "" {
        query = query.Where("severity = ?", req.Severity)
    }
    
    // 规则类型过滤
    if req.RuleType != "" {
        query = query.Where("rule_type = ?", req.RuleType)
    }
    
    // 关键词搜索
    if req.Keyword != "" {
        query = query.Where("rule_name LIKE ? OR display_name LIKE ?",
            "%"+req.Keyword+"%", "%"+req.Keyword+"%")
    }
    
    // 计数
    var total int64
    query.Count(&total)
    
    // 分页
    var rules []*AlertRule
    err := query.Offset(req.Offset).Limit(req.Limit).
        Order("created_at DESC").
        Find(&rules).Error
    
    return rules, total, err
}

// UpdateRule updates a rule
func (s *RuleService) UpdateRule(ctx context.Context, id uint64, req *UpdateRuleRequest) error {
    var rule AlertRule
    if err := s.db.First(&rule, id).Error; err != nil {
        return fmt.Errorf("rule not found")
    }
    
    updates := make(map[string]interface{})
    
    if req.DisplayName != nil {
        updates["display_name"] = *req.DisplayName
    }
    if req.Description != nil {
        updates["description"] = *req.Description
    }
    if req.PromQLExpr != nil {
        // 验证 PromQL
        datasource, _ := s.datasourceSvc.GetByID(ctx, rule.DatasourceID)
        if err := s.validatePromQL(ctx, datasource, *req.PromQLExpr); err != nil {
            return fmt.Errorf("invalid promql: %w", err)
        }
        updates["promql_expr"] = *req.PromQLExpr
    }
    if req.Severity != nil {
        updates["severity"] = *req.Severity
    }
    if req.Enabled != nil {
        updates["enabled"] = *req.Enabled
    }
    
    return s.db.Model(&rule).Updates(updates).Error
}

// DeleteRule deletes a rule
func (s *RuleService) DeleteRule(ctx context.Context, id uint64) error {
    result := s.db.Delete(&AlertRule{}, id)
    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected == 0 {
        return fmt.Errorf("rule not found")
    }
    return nil
}

// ValidateRule validates a rule without saving
func (s *RuleService) ValidateRule(ctx context.Context, req *CreateRuleRequest) error {
    datasource, err := s.datasourceSvc.GetByID(ctx, req.DatasourceID)
    if err != nil {
        return fmt.Errorf("datasource not found: %w", err)
    }
    
    var promql string
    
    switch req.RuleType {
    case "promql":
        promql = req.PromQLExpr
    case "threshold":
        promql = s.buildThresholdQuery(req)
    case "template":
        promql, err = s.renderTemplate(ctx, req.TemplateID, req.TemplateParams)
        if err != nil {
            return err
        }
    }
    
    return s.validatePromQL(ctx, datasource, promql)
}

// TestRule tests a rule and returns matched results
func (s *RuleService) TestRule(ctx context.Context, id uint64) (interface{}, error) {
    rule, err := s.GetRule(ctx, id)
    if err != nil {
        return nil, err
    }
    
    datasource, err := s.datasourceSvc.GetByID(ctx, rule.DatasourceID)
    if err != nil {
        return nil, err
    }
    
    // 执行查询
    promClient, err := s.datasourceSvc.GetPrometheusClient(datasource)
    if err != nil {
        return nil, err
    }
    
    result, warnings, err := promClient.Query(ctx, rule.PromQLExpr, time.Now())
    if err != nil {
        return nil, err
    }
    
    return map[string]interface{}{
        "result":   result,
        "warnings": warnings,
    }, nil
}

// validatePromQL validates PromQL syntax
func (s *RuleService) validatePromQL(ctx context.Context, datasource *PrometheusDatasource, promql string) error {
    if promql == "" {
        return fmt.Errorf("promql expression is empty")
    }
    
    // 使用 Prometheus API 验证语法
    promClient, err := s.datasourceSvc.GetPrometheusClient(datasource)
    if err != nil {
        return err
    }
    
    // 执行查询（限制1个结果）
    _, warnings, err := promClient.Query(ctx, promql, time.Now())
    if err != nil {
        return fmt.Errorf("promql syntax error: %w", err)
    }
    
    if len(warnings) > 0 {
        return fmt.Errorf("promql warnings: %v", warnings)
    }
    
    return nil
}

// buildThresholdQuery builds PromQL from threshold config
func (s *RuleService) buildThresholdQuery(req *CreateRuleRequest) string {
    query := fmt.Sprintf("%s(%s)", req.Aggregation, req.MetricName)
    
    if req.TargetFilters != nil {
        labels := []string{}
        for k, v := range req.TargetFilters {
            labels = append(labels, fmt.Sprintf(`%s="%s"`, k, v))
        }
        if len(labels) > 0 {
            query = fmt.Sprintf("%s(%s{%s})", req.Aggregation, req.MetricName, strings.Join(labels, ","))
        }
    }
    
    return query
}

// renderTemplate renders a template with parameters
func (s *RuleService) renderTemplate(ctx context.Context, templateID uint64, params map[string]interface{}) (string, error) {
    var template AlertRuleTemplate
    if err := s.db.First(&template, templateID).Error; err != nil {
        return "", fmt.Errorf("template not found")
    }
    
    promql := template.PromQLTemplate
    
    // 简单的变量替换（使用 {{varname}} 语法）
    for k, v := range params {
        placeholder := fmt.Sprintf("{{%s}}", k)
        promql = strings.ReplaceAll(promql, placeholder, fmt.Sprintf("%v", v))
    }
    
    return promql, nil
}

// ListTemplates lists all rule templates
func (s *RuleService) ListTemplates(ctx context.Context, category string) ([]*AlertRuleTemplate, error) {
    query := s.db.Model(&AlertRuleTemplate{})
    
    if category != "" {
        query = query.Where("category = ?", category)
    }
    
    var templates []*AlertRuleTemplate
    err := query.Order("category, template_name").Find(&templates).Error
    
    return templates, err
}
```

---

## 5. 实现指南

### 5.1 目录结构

```
backend/modules/business/observability/alert/
├── rule_model.go          - 规则数据模型
├── rule_repository.go     - 数据访问层
├── rule_service.go        - 业务逻辑层
├── rule_dto.go            - 请求/响应 DTO
├── template_service.go    - 模板服务
└── rule_service_test.go   - 单元测试
```

### 5.2 关键步骤

1. **实现 rule_model.go**
   - 定义 AlertRule 和 AlertRuleTemplate 结构体

2. **实现 rule_service.go**
   - CreateRule: 创建规则 + 验证
   - GetRule/ListRules: 查询规则
   - UpdateRule: 更新规则
   - DeleteRule: 删除规则
   - ValidateRule: 验证规则
   - TestRule: 试运行规则

3. **实现 template_service.go**
   - 内置模板初始化
   - 模板渲染逻辑

4. **实现单元测试**
   - 测试规则 CRUD
   - 测试规则验证
   - 测试模板渲染

---

## 6. 测试要求

### 6.1 单元测试

**必须覆盖**:
- [ ] CreateRule (3种类型)
- [ ] GetRule
- [ ] ListRules (各种过滤条件)
- [ ] UpdateRule
- [ ] DeleteRule
- [ ] ValidateRule
- [ ] TestRule
- [ ] buildThresholdQuery
- [ ] renderTemplate

**覆盖率要求**: >75%

### 6.2 集成测试

**测试场景**:
1. 创建 PromQL 类型规则
2. 创建阈值类型规则
3. 创建模板类型规则
4. 验证 PromQL 语法错误
5. 试运行规则查看结果
6. 更新规则并验证
7. 删除规则

---

## 7. 验收标准

### 7.1 功能完整性
- [ ] 支持 3 种规则类型
- [ ] 规则 CRUD 完整
- [ ] 规则验证功能
- [ ] 规则测试功能
- [ ] 模板系统功能

### 7.2 数据完整性
- [ ] 规则必须关联数据源
- [ ] 规则必须归属业务域
- [ ] PromQL 必须验证通过
- [ ] 模板参数必须完整

---

## 8. 交付物清单

### 必须交付
- [ ] `rule_model.go`
- [ ] `rule_service.go`
- [ ] `rule_dto.go`
- [ ] `template_service.go`
- [ ] `rule_service_test.go`
- [ ] 内置模板初始化脚本

---

## 9. 参考资料

- Prometheus PromQL 文档: https://prometheus.io/docs/prometheus/latest/querying/basics/
- 告警规则最佳实践: https://prometheus.io/docs/practices/alerting/

---

## 10. 变更记录

| 日期 | 变更内容 | 变更人 |
|------|---------|--------|
| 2026-08-28 | 创建任务包 | Claude |
