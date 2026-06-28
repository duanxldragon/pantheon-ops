package org

import (
	"fmt"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/impexp"

	"gorm.io/gorm/clause"
)

// dept_export.go - Export functions for dept module

// ExportDepts exports department list to CSV
func (s *DeptService) ExportDepts(query *DeptListQuery) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	depts, err := s.listDeptsForExport(query)
	if err != nil {
		return nil, err
	}
	childCountByDept, err := s.loadDeptChildCounts()
	if err != nil {
		return nil, err
	}
	postCountByDept, err := s.loadDeptPostCounts()
	if err != nil {
		return nil, err
	}
	userCountByDept, err := s.loadDeptUserCounts()
	if err != nil {
		return nil, err
	}
	pathByID, _, err := impexp.BuildDeptPathMaps(s.db)
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(depts))
	for _, dept := range depts {
		if dept.IsRoot == common.StatusFlagYes {
			continue
		}
		governanceTags := buildDeptGovernanceTags(dept, childCountByDept[dept.ID], postCountByDept[dept.ID])
		governanceBlockedBy := buildDeptDeleteBlockers(childCountByDept[dept.ID], postCountByDept[dept.ID], userCountByDept[dept.ID])
		governanceActions := buildDeptGovernanceActions(governanceTags, governanceBlockedBy)
		rows = append(rows, []string{
			pathByID[dept.ParentID],
			dept.DeptName,
			fmt.Sprintf("%d", dept.Sort),
			dept.Leader,
			dept.Phone,
			dept.Email,
			fmt.Sprintf("%d", dept.Status),
			pathByID[dept.ID],
			fmt.Sprintf("%d", childCountByDept[dept.ID]),
			fmt.Sprintf("%d", postCountByDept[dept.ID]),
			fmt.Sprintf("%d", userCountByDept[dept.ID]),
			"dept",
			strings.Join(governanceTags, "|"),
			fmt.Sprintf("%d", impexp.CountGovernanceProblems(governanceTags, map[string]struct{}{"leaderless": {}, "no-post": {}, "empty": {}})),
			strings.Join(governanceBlockedBy, "|"),
			strings.Join(governanceActions, "|"),
			impexp.GovernanceScopeLabel("dept"),
			impexp.GovernanceTagLabels(governanceTags),
			impexp.GovernanceBlockedByLabels(governanceBlockedBy),
			impexp.GovernanceActionLabels(governanceActions),
		})
	}

	return &impexp.CSVFile{
		Filename: "system-dept-export.csv",
		Headers:  append([]string{"parentDeptPath", "deptName", "sort", "leader", "phone", "email", "status", "deptPath", "childDeptCount", "postCount", "userCount"}, impexp.GovernanceExportHeaders...),
		Rows:     rows,
	}, nil
}

// BuildDeptImportTemplate generates import template CSV
func (s *DeptService) BuildDeptImportTemplate() *impexp.CSVFile {
	return &impexp.CSVFile{
		Filename: "system-dept-import-template.csv",
		Headers:  []string{"parentDeptPath", "deptName", "sort", "leader", "phone", "email", "status"},
		Rows: [][]string{
			{"#说明：保留第一行表头；parentDeptPath 使用部门导出的完整路径；根节点下创建部门时通常填写 Pantheon Base；status 使用 1=启用、2=禁用。", "", "", "", "", "", ""},
			{"#Pantheon Base", "研发中心", "10", "张三", "13800138000", "rd@example.com", "1"},
		},
	}
}

// ExportGovernanceTasks exports governance tasks to CSV
func (s *DeptService) ExportGovernanceTasks(query *DeptGovernanceTaskQuery) (*impexp.CSVFile, error) {
	items, err := s.ListGovernanceTasks(query)
	if err != nil {
		return nil, err
	}
	snapshotAt := time.Now().Format(time.RFC3339)
	querySummary := buildGovernanceTaskQuerySummary(query)
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			snapshotAt,
			querySummary,
			item.TaskKey,
			item.GovernanceScope,
			item.GovernanceScopeLabel,
			item.GovernanceTag,
			item.GovernanceTagLabel,
			item.GovernanceBlockedBy,
			item.GovernanceBlockedByLabel,
			item.GovernanceAction,
			item.GovernanceActionLabel,
			fmt.Sprintf("%d", item.DeptID),
			item.DeptName,
			item.DeptPath,
			fmt.Sprintf("%d", item.PostID),
			item.PostName,
			fmt.Sprintf("%d", item.RelatedUserCount),
			fmt.Sprintf("%d", item.ResourceStatus),
		})
	}
	return &impexp.CSVFile{
		Filename: "system-org-governance-tasks.csv",
		Headers: []string{
			"governanceSnapshotAt", "governanceQuerySummary", "taskKey",
			"governanceScope", "governanceScopeLabel",
			"governanceTag", "governanceTagLabel",
			"governanceBlockedBy", "governanceBlockedByLabel",
			"governanceAction", "governanceActionLabel",
			"deptId", "deptName", "deptPath",
			"postId", "postName",
			"relatedUserCount", "resourceStatus",
		},
		Rows: rows,
	}, nil
}

// listDeptsForExport lists departments with optional filtering
func (s *DeptService) listDeptsForExport(query *DeptListQuery) ([]SystemDept, error) {
	var depts []SystemDept
	sortColumn, sortDesc := normalizeDeptSort(query)
	if err := s.db.Model(&SystemDept{}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: sortColumn}, Desc: sortDesc}).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "id"}, Desc: false}).
		Find(&depts).Error; err != nil {
		return nil, err
	}
	postCountByDept, err := s.loadDeptPostCounts()
	if err != nil {
		return nil, err
	}
	return filterDeptTreeNodes(depts, query, postCountByDept), nil
}
