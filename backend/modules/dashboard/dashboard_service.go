package dashboard

import (
	"errors"
	"time"

	auth "pantheon-ops/backend/modules/auth"
	dynamicmodule "pantheon-ops/backend/modules/system/dynamicmodule"
	dept "pantheon-ops/backend/modules/system/org/dept"
	"pantheon-ops/backend/pkg/authsession"

	"gorm.io/gorm"
)

const summaryPeriodDays = 7

type DashboardService struct {
	db *gorm.DB
}

func NewDashboardService(db *gorm.DB) *DashboardService {
	return &DashboardService{db: db}
}

func (s *DashboardService) GetSummary() (*SummaryResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	now := time.Now()
	since := now.AddDate(0, 0, -summaryPeriodDays)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	idleMinutes := authsession.LoadSessionIdleMinutes(s.db, authsession.DefaultSessionIdleMinutes)
	if err := authsession.CleanupInactiveSessions(s.db, now, idleMinutes); err != nil {
		return nil, err
	}
	resp := &SummaryResp{
		PeriodDays: summaryPeriodDays,
	}

	if err := s.db.Table("system_user").Where("deleted_at IS NULL").Count(&resp.TotalUsers).Error; err != nil {
		return nil, err
	}
	if err := s.db.Table("system_user").Where("deleted_at IS NULL AND status = ?", 1).Count(&resp.EnabledUsers).Error; err != nil {
		return nil, err
	}
	if err := s.db.Table("system_role").Where("deleted_at IS NULL").Count(&resp.TotalRoles).Error; err != nil {
		return nil, err
	}
	if err := s.db.Table("system_dept").Where("deleted_at IS NULL AND is_root = ?", 0).Count(&resp.TotalDepts).Error; err != nil {
		return nil, err
	}
	if err := s.db.Table("system_post").Where("deleted_at IS NULL").Count(&resp.TotalPosts).Error; err != nil {
		return nil, err
	}
	if err := s.db.Table("system_dict_type").Where("deleted_at IS NULL").Count(&resp.TotalDictTypes).Error; err != nil {
		return nil, err
	}
	if err := s.db.Table("system_setting").Count(&resp.TotalSettings).Error; err != nil {
		return nil, err
	}
	if s.db.Migrator().HasTable("system_i18n") {
		if err := s.db.Table("system_i18n").Count(&resp.TotalI18nEntries).Error; err != nil {
			return nil, err
		}
	}
	if s.db.Migrator().HasTable(dynamicmodule.ModuleRegistration{}.TableName()) {
		if err := s.db.Table(dynamicmodule.ModuleRegistration{}.TableName()).
			Where("status = ?", dynamicmodule.ModuleStatusActive).
			Count(&resp.ActiveModuleCount).Error; err != nil {
			return nil, err
		}
	}
	if err := s.db.Table("system_menu").Where("is_visible = ? AND type <> ?", 1, "F").Count(&resp.VisibleMenuCount).Error; err != nil {
		return nil, err
	}
	if err := authsession.ApplyActiveScope(s.db.Table("system_user_session"), "", now, idleMinutes).
		Count(&resp.ActiveSessionCount).Error; err != nil {
		return nil, err
	}
	if err := s.db.Table("system_log_login").
		Where("status = ? AND login_time >= ?", 1, since).
		Count(&resp.LoginSuccessCount).Error; err != nil {
		return nil, err
	}
	if err := s.db.Table("system_log_login").
		Where("status = ? AND login_time >= ?", 0, since).
		Count(&resp.LoginFailureCount).Error; err != nil {
		return nil, err
	}
	if s.db.Migrator().HasTable(auth.SystemAuthSecurityEvent{}.TableName()) {
		if err := s.db.Table(auth.SystemAuthSecurityEvent{}.TableName()).
			Where("created_at >= ?", since).
			Count(&resp.TotalSecurityEventCount).Error; err != nil {
			return nil, err
		}
		if err := s.db.Table(auth.SystemAuthSecurityEvent{}.TableName()).
			Where("acknowledged_at IS NULL AND created_at >= ?", since).
			Count(&resp.PendingSecurityEventCount).Error; err != nil {
			return nil, err
		}
	}
	if err := s.db.Table("system_log_oper").
		Where("oper_time >= ?", todayStart).
		Count(&resp.TodayOperationCount).Error; err != nil {
		return nil, err
	}

	var lastSuccessfulLoginAt time.Time
	if err := s.db.Table("system_log_login").
		Select("login_time").
		Where("status = ?", 1).
		Order("login_time desc").
		Limit(1).
		Scan(&lastSuccessfulLoginAt).Error; err != nil {
		return nil, err
	}
	if !lastSuccessfulLoginAt.IsZero() {
		resp.LastSuccessfulLoginAt = lastSuccessfulLoginAt.Format(time.RFC3339)
	}

	type rawLoginRow struct {
		ID        uint64    `gorm:"column:id"`
		Username  string    `gorm:"column:username"`
		Ipaddr    string    `gorm:"column:ipaddr"`
		Browser   string    `gorm:"column:browser"`
		OS        string    `gorm:"column:os"`
		Status    int       `gorm:"column:status"`
		Msg       string    `gorm:"column:msg"`
		LoginTime time.Time `gorm:"column:login_time"`
	}

	var rawRows []rawLoginRow
	if err := s.db.Table("system_log_login").
		Select("id, username, ipaddr, browser, os, status, msg, login_time").
		Order("login_time desc, id desc").
		Limit(8).
		Scan(&rawRows).Error; err != nil {
		return nil, err
	}

	resp.RecentLogins = make([]RecentLoginActivityResp, 0, len(rawRows))
	for _, row := range rawRows {
		resp.RecentLogins = append(resp.RecentLogins, RecentLoginActivityResp{
			ID:        row.ID,
			Username:  row.Username,
			Ipaddr:    row.Ipaddr,
			Browser:   row.Browser,
			OS:        row.OS,
			Status:    row.Status,
			Msg:       row.Msg,
			LoginTime: row.LoginTime.Format(time.RFC3339),
		})
	}

	orgSvc := dept.NewDeptService(s.db)
	orgTasks, err := orgSvc.ListGovernanceTasks(&dept.DeptGovernanceTaskQuery{})
	if err != nil {
		return nil, err
	}
	resp.OrgGovernanceTaskCount = len(orgTasks)
	resp.OrgGovernanceTasks = make([]DashboardTodoResp, 0, minInt(len(orgTasks), 6))
	for _, task := range orgTasks {
		if len(resp.OrgGovernanceTasks) >= 6 {
			break
		}
		resourceLabel := task.DeptName
		if task.GovernanceScope == "post" && task.PostName != "" {
			resourceLabel = task.PostName + " / " + task.DeptName
		}
		resp.OrgGovernanceTasks = append(resp.OrgGovernanceTasks, DashboardTodoResp{
			TaskKey:          task.TaskKey,
			Domain:           "system.org",
			ScopeLabel:       task.GovernanceScopeLabel,
			IssueLabel:       task.GovernanceTagLabel,
			ActionLabel:      task.GovernanceActionLabel,
			ResourceLabel:    resourceLabel,
			RelatedUserCount: task.RelatedUserCount,
			RoutePath:        "/system/dept",
			RouteStateDeptID: task.DeptID,
		})
	}

	return resp, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
