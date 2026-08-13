//nolint:revive // Dashboard service intentionally exposes a broad summary facade.
package platform

import (
	"sync"
	"time"

	"pantheon-base/pkg/authsession"
	"pantheon-base/pkg/common"

	"gorm.io/gorm"
)

const (
	summaryPeriodDays          = 7
	dynamicModuleTable         = "system_module_registration"
	dynamicModuleStatusActive  = 1
	authSecurityEventTableName = "system_auth_security_event"
	operationLogTableName      = "system_log_oper"
	condNotDeleted             = "deleted_at IS NULL"
)

type OrgGovernanceTask struct {
	TaskKey               string
	GovernanceScope       string
	GovernanceTag         string
	GovernanceAction      string
	GovernanceScopeLabel  string
	GovernanceTagLabel    string
	GovernanceActionLabel string
	DeptID                uint64
	DeptName              string
	PostName              string
	RelatedUserCount      int
}

type OrgGovernanceTaskLoader interface {
	ListOrgGovernanceTasks() ([]OrgGovernanceTask, error)
}

type DashboardServiceOption func(*DashboardService)

type DashboardService struct {
	db                      *gorm.DB
	orgGovernanceTaskLoader OrgGovernanceTaskLoader
}

func WithOrgGovernanceTaskLoader(loader OrgGovernanceTaskLoader) DashboardServiceOption {
	return func(s *DashboardService) {
		s.orgGovernanceTaskLoader = loader
	}
}

func NewDashboardService(db *gorm.DB, options ...DashboardServiceOption) *DashboardService {
	s := &DashboardService{db: db}
	for _, option := range options {
		option(s)
	}
	return s
}

func (s *DashboardService) GetSummary() (*SummaryResp, error) {
	if s.db == nil {
		return nil, common.NewBadRequest("database.not_initialized")
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

	hasI18nTable := s.db.Migrator().HasTable("system_i18n")
	hasDynamicModuleTable := s.db.Migrator().HasTable(dynamicModuleTable)
	hasSecurityEventTable := s.db.Migrator().HasTable(authSecurityEventTableName)
	if err := s.loadSummaryCounts(resp, summaryCountsParams{
		Now:                   now,
		Since:                 since,
		TodayStart:            todayStart,
		IdleMinutes:           idleMinutes,
		HasI18nTable:          hasI18nTable,
		HasDynamicModuleTable: hasDynamicModuleTable,
		HasSecurityEventTable: hasSecurityEventTable,
	}); err != nil {
		return nil, err
	}

	if err := s.loadLoginActivity(resp, since); err != nil {
		return nil, err
	}
	if err := s.loadOrgGovernanceTasks(resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *DashboardService) loadLoginActivity(resp *SummaryResp, since time.Time) error {
	var lastSuccessfulLoginAt time.Time
	if err := s.db.Table("system_log_login").
		Select("login_time").
		Where("status = ?", 1).
		Order("login_time desc").
		Limit(1).
		Scan(&lastSuccessfulLoginAt).Error; err != nil {
		return err
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
	var recentLoginsRaw []rawLoginRow
	if err := s.db.Table("system_log_login").
		Select("id, username, ipaddr, browser, os, status, msg, login_time").
		Where("login_time >= ?", since).
		Order("login_time desc").
		Limit(20).
		Scan(&recentLoginsRaw).Error; err != nil {
		return err
	}
	resp.RecentLogins = make([]RecentLoginActivityResp, 0, len(recentLoginsRaw))
	for _, r := range recentLoginsRaw {
		resp.RecentLogins = append(resp.RecentLogins, RecentLoginActivityResp{
			ID:        r.ID,
			Username:  r.Username,
			Ipaddr:    r.Ipaddr,
			Browser:   r.Browser,
			OS:        r.OS,
			Status:    r.Status,
			Msg:       r.Msg,
			LoginTime: r.LoginTime.Format(time.RFC3339),
		})
	}
	return nil
}

func (s *DashboardService) loadOrgGovernanceTasks(resp *SummaryResp) error {
	if s.orgGovernanceTaskLoader == nil {
		return nil
	}
	tasks, err := s.orgGovernanceTaskLoader.ListOrgGovernanceTasks()
	if err != nil {
		return err
	}
	resp.OrgGovernanceTaskCount = len(tasks)
	resp.OrgGovernanceTasks = make([]DashboardTodoResp, 0, len(tasks))
	for _, task := range tasks {
		resp.OrgGovernanceTasks = append(resp.OrgGovernanceTasks, buildDashboardTodo(task))
	}
	return nil
}

func buildDashboardTodo(task OrgGovernanceTask) DashboardTodoResp {
	return DashboardTodoResp{
		TaskKey:          task.TaskKey,
		Domain:           task.GovernanceScope,
		Issue:            task.GovernanceTag,
		Action:           task.GovernanceAction,
		ScopeLabel:       task.GovernanceScopeLabel,
		IssueLabel:       task.GovernanceTagLabel,
		ActionLabel:      task.GovernanceActionLabel,
		ResourceLabel:    dashboardTodoResourceLabel(task),
		RelatedUserCount: task.RelatedUserCount,
		RoutePath:        "/system/dept",
		RouteStateDeptID: task.DeptID,
	}
}

func dashboardTodoResourceLabel(task OrgGovernanceTask) string {
	if task.GovernanceScope != "post" || task.PostName == "" {
		return task.DeptName
	}
	if task.DeptName == "" {
		return task.PostName
	}
	return task.PostName + " / " + task.DeptName
}

type summaryCountJob struct {
	count func() (int64, error)
	apply func(int64)
}

type summaryCountsParams struct {
	Now                   time.Time
	Since                 time.Time
	TodayStart            time.Time
	IdleMinutes           int
	HasI18nTable          bool
	HasDynamicModuleTable bool
	HasSecurityEventTable bool
}

func (s *DashboardService) loadSummaryCounts(resp *SummaryResp, params summaryCountsParams) error {
	jobs := []summaryCountJob{
		{count: func() (int64, error) { return s.countTable("system_user", condNotDeleted) }, apply: func(value int64) { resp.TotalUsers = value }},
		{count: func() (int64, error) { return s.countTable("system_user", "deleted_at IS NULL AND status = ?", 1) }, apply: func(value int64) { resp.EnabledUsers = value }},
		{count: func() (int64, error) { return s.countTable("system_role", condNotDeleted) }, apply: func(value int64) { resp.TotalRoles = value }},
		{count: func() (int64, error) { return s.countTable("system_dept", "deleted_at IS NULL AND is_root = ?", 0) }, apply: func(value int64) { resp.TotalDepts = value }},
		{count: func() (int64, error) { return s.countTable("system_post", condNotDeleted) }, apply: func(value int64) { resp.TotalPosts = value }},
		{count: func() (int64, error) { return s.countTable("system_dict_type", condNotDeleted) }, apply: func(value int64) { resp.TotalDictTypes = value }},
		{count: func() (int64, error) { return s.countTable("system_setting", "") }, apply: func(value int64) { resp.TotalSettings = value }},
		{count: func() (int64, error) { return s.countTable("system_menu", "is_visible = ? AND type <> ?", 1, "F") }, apply: func(value int64) { resp.VisibleMenuCount = value }},
		{count: func() (int64, error) {
			return countQuery(authsession.ApplyActiveScope(s.db.Table("system_user_session"), "", params.Now, params.IdleMinutes))
		}, apply: func(value int64) { resp.ActiveSessionCount = value }},
		{count: func() (int64, error) {
			return s.countTable("system_log_login", "status = ? AND login_time >= ?", 1, params.Since)
		}, apply: func(value int64) { resp.LoginSuccessCount = value }},
		{count: func() (int64, error) {
			return s.countTable("system_log_login", "status = ? AND login_time >= ?", 0, params.Since)
		}, apply: func(value int64) { resp.LoginFailureCount = value }},
		{count: func() (int64, error) {
			return s.countTable(operationLogTableName, "oper_time >= ?", params.TodayStart)
		}, apply: func(value int64) { resp.TodayOperationCount = value }},
	}

	if params.HasI18nTable {
		jobs = append(jobs, summaryCountJob{
			count: func() (int64, error) { return s.countTable("system_i18n", "") },
			apply: func(value int64) { resp.TotalI18nEntries = value },
		})
	}

	if params.HasDynamicModuleTable {
		jobs = append(jobs, summaryCountJob{
			count: func() (int64, error) {
				return s.countTable(dynamicModuleTable, "status = ?", dynamicModuleStatusActive)
			},
			apply: func(value int64) { resp.ActiveModuleCount = value },
		})
	}

	if params.HasSecurityEventTable {
		jobs = append(jobs,
			summaryCountJob{
				count: func() (int64, error) { return s.countTable(authSecurityEventTableName, "") },
				apply: func(value int64) { resp.TotalSecurityEventCount = value },
			},
			summaryCountJob{
				count: func() (int64, error) {
					return s.countTable(authSecurityEventTableName, "acknowledged_at IS NULL")
				},
				apply: func(value int64) { resp.PendingSecurityEventCount = value },
			},
		)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	errs := make([]error, 0, len(jobs))

	for _, job := range jobs {
		wg.Add(1)
		go func(j summaryCountJob) {
			defer wg.Done()
			count, err := j.count()
			if err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}
			mu.Lock()
			j.apply(count)
			mu.Unlock()
		}(job)
	}
	wg.Wait()

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (s *DashboardService) countTable(tableName string, where string, args ...interface{}) (int64, error) {
	var count int64
	query := s.db.Table(tableName)
	if where != "" {
		query = query.Where(where, args...)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func countQuery(q *gorm.DB) (int64, error) {
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
