package platform

import (
	"sync"
	"time"

	"pantheon-ops/backend/pkg/authsession"
	"pantheon-ops/backend/pkg/common"

	"gorm.io/gorm"
)

const (
	summaryPeriodDays          = 7
	dynamicModuleTable         = "system_module_registration"
	dynamicModuleStatusActive  = 1
	authSecurityEventTableName = "system_auth_security_event"
	operationLogTableName      = "system_log_oper"
)

type OrgGovernanceTask struct {
	TaskKey               string
	GovernanceScope       string
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
	if err := s.loadSummaryCounts(resp, now, since, todayStart, idleMinutes, hasI18nTable, hasDynamicModuleTable, hasSecurityEventTable); err != nil {
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
	var recentLoginsRaw []rawLoginRow
	if err := s.db.Table("system_log_login").
		Select("id, username, ipaddr, browser, os, status, msg, login_time").
		Where("login_time >= ?", since).
		Order("login_time desc").
		Limit(20).
		Scan(&recentLoginsRaw).Error; err != nil {
		return nil, err
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

	if s.orgGovernanceTaskLoader != nil {
		tasks, err := s.orgGovernanceTaskLoader.ListOrgGovernanceTasks()
		if err != nil {
			return nil, err
		}
		resp.OrgGovernanceTaskCount = len(tasks)
		resp.OrgGovernanceTasks = make([]DashboardTodoResp, 0, len(tasks))
		for _, task := range tasks {
			resp.OrgGovernanceTasks = append(resp.OrgGovernanceTasks, DashboardTodoResp{
				TaskKey:          task.TaskKey,
				Domain:           task.GovernanceScope,
				ScopeLabel:       task.GovernanceScopeLabel,
				IssueLabel:       task.GovernanceTagLabel,
				ActionLabel:      task.GovernanceActionLabel,
				RelatedUserCount: task.RelatedUserCount,
			})
		}
	}

	return resp, nil
}

type summaryCountJob struct {
	count func() (int64, error)
	apply func(int64)
}

func (s *DashboardService) loadSummaryCounts(resp *SummaryResp, now, since, todayStart time.Time, idleMinutes int, hasI18nTable, hasDynamicModuleTable, hasSecurityEventTable bool) error {
	jobs := []summaryCountJob{
		{count: func() (int64, error) { return s.countTable("system_user", "deleted_at IS NULL") }, apply: func(value int64) { resp.TotalUsers = value }},
		{count: func() (int64, error) { return s.countTable("system_user", "deleted_at IS NULL AND status = ?", 1) }, apply: func(value int64) { resp.EnabledUsers = value }},
		{count: func() (int64, error) { return s.countTable("system_role", "deleted_at IS NULL") }, apply: func(value int64) { resp.TotalRoles = value }},
		{count: func() (int64, error) { return s.countTable("system_dept", "deleted_at IS NULL AND is_root = ?", 0) }, apply: func(value int64) { resp.TotalDepts = value }},
		{count: func() (int64, error) { return s.countTable("system_post", "deleted_at IS NULL") }, apply: func(value int64) { resp.TotalPosts = value }},
		{count: func() (int64, error) { return s.countTable("system_dict_type", "deleted_at IS NULL") }, apply: func(value int64) { resp.TotalDictTypes = value }},
		{count: func() (int64, error) { return s.countTable("system_setting", "") }, apply: func(value int64) { resp.TotalSettings = value }},
		{count: func() (int64, error) { return s.countTable("system_menu", "is_visible = ? AND type <> ?", 1, "F") }, apply: func(value int64) { resp.VisibleMenuCount = value }},
		{count: func() (int64, error) {
			return countQuery(authsession.ApplyActiveScope(s.db.Table("system_user_session"), "", now, idleMinutes))
		}, apply: func(value int64) { resp.ActiveSessionCount = value }},
		{count: func() (int64, error) {
			return s.countTable("system_log_login", "status = ? AND login_time >= ?", 1, since)
		}, apply: func(value int64) { resp.LoginSuccessCount = value }},
		{count: func() (int64, error) {
			return s.countTable("system_log_login", "status = ? AND login_time >= ?", 0, since)
		}, apply: func(value int64) { resp.LoginFailureCount = value }},
		{count: func() (int64, error) {
			return s.countTable(operationLogTableName, "created_at >= ?", todayStart)
		}, apply: func(value int64) { resp.TodayOperationCount = value }},
	}

	if hasI18nTable {
		jobs = append(jobs, summaryCountJob{
			count: func() (int64, error) { return s.countTable("system_i18n", "") },
			apply: func(value int64) { resp.TotalI18nEntries = value },
		})
	}

	if hasDynamicModuleTable {
		jobs = append(jobs, summaryCountJob{
			count: func() (int64, error) {
				return s.countTable(dynamicModuleTable, "status = ?", dynamicModuleStatusActive)
			},
			apply: func(value int64) { resp.ActiveModuleCount = value },
		})
	}

	if hasSecurityEventTable {
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
