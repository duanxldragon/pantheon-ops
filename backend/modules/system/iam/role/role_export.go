package iam

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/impexp"

	"gorm.io/gorm/clause"
)

func (s *RoleService) ExportRoles(query *RoleListQuery) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	roles, err := s.listRolesForExport(query)
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(roles))
	for _, role := range roles {
		rows = append(rows, []string{
			role.RoleName,
			role.RoleKey,
			fmt.Sprintf("%d", role.Sort),
			fmt.Sprintf("%d", role.Status),
			role.CreatedAt.Format(time.RFC3339),
		})
	}

	return &impexp.CSVFile{
		Filename: "system-role-export.csv",
		Headers:  []string{"roleName", "roleKey", "sort", "status", "createdAt"},
		Rows:     rows,
	}, nil
}

func (s *RoleService) listRolesForExport(query *RoleListQuery) ([]SystemRole, error) {
	var roles []SystemRole
	db := s.db.Model(&SystemRole{})
	if query != nil {
		if strings.TrimSpace(query.RoleName) != "" {
			db = db.Where("role_name LIKE ?", fmt.Sprintf("%%%s%%", strings.TrimSpace(query.RoleName)))
		}
		if strings.TrimSpace(query.RoleKey) != "" {
			db = db.Where("role_key LIKE ?", fmt.Sprintf("%%%s%%", strings.TrimSpace(query.RoleKey)))
		}
		if query.Status != nil && (*query.Status == 1 || *query.Status == 2) {
			db = db.Where("status = ?", *query.Status)
		}
	}

	sortColumn, sortDesc := normalizeRoleSort(query)
	if err := db.
		Order(clause.OrderByColumn{Column: clause.Column{Name: sortColumn}, Desc: sortDesc}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: false}).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}
