package main

import (
	platform "github.com/duanxldragon/pantheon-base/backend/modules/platform"
	dept "github.com/duanxldragon/pantheon-base/backend/modules/system/org/dept"
	"gorm.io/gorm"
)

// platformDeptGovernanceTaskLoader is the composition-root adapter that lets the
// platform dashboard surface org governance tasks without the platform module
// importing the system/org/dept implementation directly.
//
// Boundary rule: REPOSITORY_LAYOUT.md §8.2 — `platform` may depend on `system/*`
// only through a public contract or an adapter owned by the composition root.
// Keeping the adapter in cmd/server (the shell entrypoint) is that adapter: the
// gate `scripts/harness/check-boundaries.mjs` fails if the import moves back
// under backend/modules/platform.
type platformDeptGovernanceTaskLoader struct {
	db *gorm.DB
}

var _ platform.OrgGovernanceTaskLoader = platformDeptGovernanceTaskLoader{}

func (l platformDeptGovernanceTaskLoader) ListOrgGovernanceTasks() ([]platform.OrgGovernanceTask, error) {
	orgSvc := dept.NewDeptService(l.db)
	tasks, err := orgSvc.ListGovernanceTasks(&dept.DeptGovernanceTaskQuery{})
	if err != nil {
		return nil, err
	}
	result := make([]platform.OrgGovernanceTask, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, platform.OrgGovernanceTask{
			TaskKey:               task.TaskKey,
			GovernanceScope:       task.GovernanceScope,
			GovernanceTag:         task.GovernanceTag,
			GovernanceAction:      task.GovernanceAction,
			GovernanceScopeLabel:  task.GovernanceScopeLabel,
			GovernanceTagLabel:    task.GovernanceTagLabel,
			GovernanceActionLabel: task.GovernanceActionLabel,
			DeptID:                task.DeptID,
			DeptName:              task.DeptName,
			PostName:              task.PostName,
			RelatedUserCount:      task.RelatedUserCount,
		})
	}
	return result, nil
}
