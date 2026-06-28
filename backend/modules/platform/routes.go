package platform

import (
	"pantheon-ops/backend/internal/middleware"
	"pantheon-ops/backend/pkg/database"

	dept "pantheon-ops/backend/modules/system/org/dept"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type platformDeptGovernanceTaskLoader struct {
	db *gorm.DB
}

func (l platformDeptGovernanceTaskLoader) ListOrgGovernanceTasks() ([]OrgGovernanceTask, error) {
	orgSvc := dept.NewDeptService(l.db)
	tasks, err := orgSvc.ListGovernanceTasks(&dept.DeptGovernanceTaskQuery{})
	if err != nil {
		return nil, err
	}
	result := make([]OrgGovernanceTask, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, OrgGovernanceTask{
			TaskKey:               task.TaskKey,
			GovernanceScope:       task.GovernanceScope,
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

func RegisterPlatformRoutes(r *gin.RouterGroup, db *gorm.DB) {
	tokenMiddleware := middleware.TokenAuthMiddleware(database.RDB)

	dashboardSvc := NewDashboardService(db, WithOrgGovernanceTaskLoader(platformDeptGovernanceTaskLoader{db: db}))
	dashboardHandler := NewDashboardHandler(dashboardSvc)

	dashboardGroup := r.Group("/dashboard").Use(tokenMiddleware)
	{
		dashboardGroup.GET("/summary", dashboardHandler.GetSummary)
	}

	RegisterHealthRoutes(r, db)
}
