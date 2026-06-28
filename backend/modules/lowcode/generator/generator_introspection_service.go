package generator

import (
	"errors"
	"pantheon-ops/backend/pkg/common"
	"regexp"
	"strings"

	"gorm.io/gorm"
)

type tableRow struct {
	TableName    string `gorm:"column:table_name"`
	TableComment string `gorm:"column:table_comment"`
	Engine       string `gorm:"column:engine"`
	TableRows    int64  `gorm:"column:table_rows"`
}

type columnRow struct {
	ColumnName    string `gorm:"column:column_name"`
	DataType      string `gorm:"column:data_type"`
	ColumnType    string `gorm:"column:column_type"`
	ColumnKey     string `gorm:"column:column_key"`
	IsNullable    string `gorm:"column:is_nullable"`
	Extra         string `gorm:"column:extra"`
	ColumnComment string `gorm:"column:column_comment"`
}

type generatorSchemaReader struct {
	db     *gorm.DB
	schema string
	close  func() error
}

func (s *GeneratorService) ListTables(datasourceID, keyword string) ([]TableOptionResp, error) {
	reader, err := s.openSchemaReader(datasourceID)
	if err != nil {
		return nil, err
	}
	if reader.close != nil {
		defer reader.close()
	}

	query := reader.db.Table("information_schema.tables").
		Select("table_name, table_comment, engine, table_rows").
		Where("table_schema = ?", reader.schema).
		Order("table_name asc")

	normalizedKeyword := strings.TrimSpace(keyword)
	if normalizedKeyword != "" {
		like := "%" + normalizedKeyword + "%"
		query = query.Where("(table_name like ? or table_comment like ?)", like, like)
	}

	var rows []tableRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]TableOptionResp, 0, len(rows))
	for _, row := range rows {
		items = append(items, TableOptionResp{
			TableName: row.TableName,
			Comment:   row.TableComment,
			Engine:    row.Engine,
			Rows:      row.TableRows,
		})
	}
	return items, nil
}

func (s *GeneratorService) PreviewTable(datasourceID, tableName string) (*TableSchemaPreviewResp, error) {
	normalizedTable := strings.TrimSpace(tableName)
	if normalizedTable == "" {
		return nil, common.NewBadRequest("generator.table.required")
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(normalizedTable) {
		return nil, common.NewBadRequest("generator.table.invalid")
	}

	reader, err := s.openSchemaReader(datasourceID)
	if err != nil {
		return nil, err
	}
	if reader.close != nil {
		defer reader.close()
	}

	var table tableRow
	if err := reader.db.Table("information_schema.tables").
		Select("table_name, table_comment, engine, table_rows").
		Where("table_schema = ? and table_name = ?", reader.schema, normalizedTable).
		Take(&table).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.NewNotFound("generator.table.not_found")
		}
		return nil, err
	}

	var columns []columnRow
	if err := reader.db.Table("information_schema.columns").
		Select("column_name, data_type, column_type, column_key, is_nullable, extra, column_comment").
		Where("table_schema = ? and table_name = ?", reader.schema, normalizedTable).
		Order("ordinal_position asc").
		Scan(&columns).Error; err != nil {
		return nil, err
	}

	fields := make([]ModuleFieldResp, 0, len(columns))
	for _, column := range columns {
		if isIgnoredGovernanceColumn(column.ColumnName) {
			continue
		}
		field := mapColumnToField(column)
		fields = append(fields, field)
	}

	resp := &TableSchemaPreviewResp{
		TableName:      table.TableName,
		TableComment:   table.TableComment,
		SuggestedName:  suggestModuleName(table.TableName),
		SuggestedScope: suggestScope(table.TableName),
		SuggestedTitle: suggestTitle(table.TableName, table.TableComment),
		Fields:         fields,
	}
	return resp, nil
}
