package generator

import (
	"fmt"
	"strings"
	"unicode"
)

func mapColumnToField(column columnRow) ModuleFieldResp {
	fieldType, dictCode, enumOptions := mapColumnType(column)
	required := strings.EqualFold(column.IsNullable, "NO") && !strings.Contains(strings.ToLower(column.Extra), "auto_increment")
	unique := column.ColumnKey == "UNI"
	label := humanizeLabel(column.ColumnName, column.ColumnComment)
	labelEn := humanizeEnglishLabel(column.ColumnName)
	placeholder := ""
	placeholderEn := ""
	if fieldType == "string" || fieldType == "text" {
		placeholder = fmt.Sprintf("请输入%s", label)
		placeholderEn = fmt.Sprintf("Enter %s", strings.ToLower(labelEn))
	}
	if fieldType == "enum" {
		placeholder = fmt.Sprintf("请选择%s", label)
		placeholderEn = fmt.Sprintf("Select %s", strings.ToLower(labelEn))
	}

	validation := &FieldValidationResp{
		Required: required,
		Unique:   unique,
	}
	if len(enumOptions) > 0 {
		validation.Enum = make([]string, 0, len(enumOptions))
		for _, item := range enumOptions {
			validation.Enum = append(validation.Enum, item.Value)
		}
	}
	if !required && !unique && len(validation.Enum) == 0 {
		validation = nil
	}

	return ModuleFieldResp{
		Name:          toCamel(column.ColumnName),
		Type:          fieldType,
		Label:         label,
		LabelEn:       labelEn,
		Required:      required,
		Searchable:    shouldFieldBeSearchable(column.ColumnName, fieldType),
		Sortable:      fieldType != "text" && fieldType != "relation",
		VisibleInList: fieldType != "text",
		VisibleInForm: true,
		Placeholder:   placeholder,
		PlaceholderEn: placeholderEn,
		HelpText:      strings.TrimSpace(column.ColumnComment),
		HelpTextEn:    "",
		DictCode:      dictCode,
		EnumOptions:   enumOptions,
		Validation:    validation,
	}
}

func mapColumnType(column columnRow) (string, string, []EnumOptionResp) {
	dataType := strings.ToLower(strings.TrimSpace(column.DataType))
	columnType := strings.ToLower(strings.TrimSpace(column.ColumnType))
	columnName := strings.ToLower(strings.TrimSpace(column.ColumnName))

	switch dataType {
	case "varchar", "char":
		if dictCode, options, ok := inferConventionalEnumField(columnName); ok {
			return "enum", dictCode, options
		}
		return "string", "", nil
	case "text", "mediumtext", "longtext":
		return "text", "", nil
	case "tinyint":
		if strings.HasPrefix(columnType, "tinyint(1)") {
			return "bool", "", nil
		}
		return "int", "", nil
	case "smallint", "mediumint", "int", "bigint":
		return "int", "", nil
	case "decimal", "float", "double":
		return "float", "", nil
	case "date", "datetime", "timestamp":
		return "date", "", nil
	case "enum", "set":
		options := parseEnumOptions(columnType)
		return "enum", "", options
	default:
		return "string", "", nil
	}
}

func inferConventionalEnumField(columnName string) (string, []EnumOptionResp, bool) {
	switch columnName {
	case "environment":
		return "environment", []EnumOptionResp{
			{Value: "dev", Label: "开发", LabelEn: "Development"},
			{Value: "test", Label: "测试", LabelEn: "Test"},
			{Value: "staging", Label: "预发", LabelEn: "Staging"},
			{Value: "prod", Label: "生产", LabelEn: "Production"},
		}, true
	case "status":
		return "status", []EnumOptionResp{
			{Value: "active", Label: "启用", LabelEn: "Active"},
			{Value: "inactive", Label: "停用", LabelEn: "Inactive"},
		}, true
	default:
		return "", nil, false
	}
}

func parseEnumOptions(columnType string) []EnumOptionResp {
	start := strings.Index(columnType, "(")
	end := strings.LastIndex(columnType, ")")
	if start < 0 || end <= start {
		return nil
	}
	raw := columnType[start+1 : end]
	parts := strings.Split(raw, ",")
	items := make([]EnumOptionResp, 0, len(parts))
	for _, part := range parts {
		value := strings.Trim(part, "' ")
		if value == "" {
			continue
		}
		items = append(items, EnumOptionResp{
			Value:   value,
			Label:   strings.ReplaceAll(value, "_", " "),
			LabelEn: humanizeEnglishLabel(value),
		})
	}
	return items
}

func isIgnoredGovernanceColumn(columnName string) bool {
	switch strings.ToLower(strings.TrimSpace(columnName)) {
	case "id", "created_at", "updated_at", "deleted_at":
		return true
	default:
		return false
	}
}

func suggestModuleName(tableName string) string {
	normalized := strings.TrimSpace(strings.ToLower(tableName))
	if strings.HasPrefix(normalized, "system_") {
		return strings.TrimPrefix(normalized, "system_")
	}
	normalized = strings.TrimPrefix(normalized, "biz_")
	return strings.ReplaceAll(normalized, "_", "/")
}

func suggestScope(tableName string) string {
	normalized := strings.TrimSpace(strings.ToLower(tableName))
	if strings.HasPrefix(normalized, "system_") {
		return "system"
	}
	return "business"
}

func suggestTitle(tableName string, comment string) string {
	trimmed := strings.TrimSpace(comment)
	if trimmed != "" {
		return trimmed
	}
	return humanizeLabel(strings.TrimPrefix(strings.TrimPrefix(tableName, "biz_"), "system_"), "")
}

func humanizeLabel(name string, comment string) string {
	if trimmed := strings.TrimSpace(comment); trimmed != "" {
		return trimmed
	}
	if label, ok := conventionalChineseFieldLabels[strings.ToLower(strings.TrimSpace(name))]; ok {
		return label
	}
	normalized := strings.ReplaceAll(strings.TrimSpace(name), "_", " ")
	if normalized == "" {
		return name
	}
	runes := []rune(normalized)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func humanizeEnglishLabel(name string) string {
	if label, ok := conventionalEnglishFieldLabels[strings.ToLower(strings.TrimSpace(name))]; ok {
		return label
	}
	normalized := strings.NewReplacer("_", " ", "-", " ").Replace(strings.TrimSpace(name))
	if normalized == "" {
		return name
	}
	words := strings.Fields(normalized)
	for index, word := range words {
		if strings.ToUpper(word) == word && len(word) <= 4 {
			continue
		}
		words[index] = strings.ToLower(word)
	}
	result := strings.Join(words, " ")
	runes := []rune(result)
	if len(runes) == 0 {
		return result
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func toCamel(value string) string {
	parts := strings.Split(strings.TrimSpace(strings.ToLower(value)), "_")
	if len(parts) == 0 {
		return value
	}
	builder := strings.Builder{}
	for index, part := range parts {
		if part == "" {
			continue
		}
		if index == 0 {
			builder.WriteString(part)
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		builder.WriteString(string(runes))
	}
	return builder.String()
}

var conventionalChineseFieldLabels = map[string]string{
	"arch":             "架构",
	"cluster_name":     "集群名称",
	"code":             "编码",
	"display_name":     "显示名称",
	"email":            "邮箱",
	"environment":      "环境",
	"host_code":        "主机编码",
	"hostname":         "主机名",
	"idc_code":         "机房编码",
	"ip_address":       "IP 地址",
	"kernel_version":   "内核版本",
	"lifecycle_status": "生命周期状态",
	"maintainer_team":  "维护团队",
	"name":             "名称",
	"os_family":        "系统家族",
	"os_name":          "操作系统",
	"owner_name":       "负责人",
	"owner_user_id":    "负责人用户 ID",
	"phone":            "手机号",
	"provider":         "云厂商",
	"purpose":          "用途",
	"region_code":      "区域编码",
	"remark":           "备注",
	"sort":             "排序",
	"ssh_port":         "SSH 端口",
	"status":           "状态",
}

var conventionalEnglishFieldLabels = map[string]string{
	"idc_code":      "IDC code",
	"ip_address":    "IP address",
	"os_family":     "OS family",
	"os_name":       "OS name",
	"owner_user_id": "Owner user ID",
	"ssh_port":      "SSH port",
}

func shouldFieldBeSearchable(name string, fieldType string) bool {
	if fieldType == "text" || fieldType == "relation" {
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(name))
	return strings.Contains(normalized, "name") ||
		strings.Contains(normalized, "code") ||
		strings.Contains(normalized, "title") ||
		strings.Contains(normalized, "status") ||
		strings.Contains(normalized, "phone") ||
		strings.Contains(normalized, "email")
}
