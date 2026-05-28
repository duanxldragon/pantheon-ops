package deploy

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"pantheon-ops/backend/modules/business/cmdb"
	"pantheon-ops/backend/pkg/testmysql"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func setupDeployTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	if err := svc.Migrate(); err != nil {
		t.Fatalf("migrate deploy tables: %v", err)
	}
	if err := db.AutoMigrate(&deployTestHost{}, &deployTestGroup{}, &deployTestBizScope{}); err != nil {
		t.Fatalf("migrate cmdb fixtures: %v", err)
	}
	return db
}

type deployTestHost struct {
	ID                  uint64         `gorm:"primaryKey;autoIncrement"`
	Hostname            string         `gorm:"size:128;not null"`
	IP                  string         `gorm:"size:45;not null"`
	SSHPort             int            `gorm:"column:ssh_port"`
	OS                  string         `gorm:"size:32;not null"`
	Status              string         `gorm:"size:32;default:pending"`
	LabelValues         datatypes.JSON `gorm:"type:json"`
	InstalledComponents datatypes.JSON `gorm:"column:installed_components;type:json"`
	BusinessScopeID     uint64         `gorm:"column:business_scope_id"`
	BusinessScopeName   string         `gorm:"column:business_scope_name"`
	DeptID              uint64         `gorm:"column:dept_id"`
	UpdatedAt           time.Time
	UpdatedBy           string         `gorm:"column:updated_by"`
	DeletedAt           gorm.DeletedAt `gorm:"index"`
}

func (deployTestHost) TableName() string { return "biz_cmdb_host" }

type deployTestBizScope struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement"`
	Code      string         `gorm:"size:64;not null"`
	Name      string         `gorm:"size:128;not null"`
	Status    string         `gorm:"size:16;not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (deployTestBizScope) TableName() string { return "biz_business_scope" }

type deployTestGroup struct {
	ID         uint64         `gorm:"primaryKey;autoIncrement"`
	ParentID   uint64         `gorm:"column:parent_id"`
	Name       string         `gorm:"size:128;not null"`
	Conditions datatypes.JSON `gorm:"type:json"`
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (deployTestGroup) TableName() string { return "biz_cmdb_group" }

type fakeDeploySSHRunner struct {
	script string
	stdout string
	stderr string
	err    error
}

func (r *fakeDeploySSHRunner) RunScript(script string) (string, string, error) {
	r.script = script
	return r.stdout, r.stderr, r.err
}

func (r *fakeDeploySSHRunner) Close() error {
	return nil
}

type multiStepDeploySSHRunner struct {
	scripts []string
	stdouts []string
	stderr  string
	errAt   int
}

func (r *multiStepDeploySSHRunner) RunScript(script string) (string, string, error) {
	r.scripts = append(r.scripts, script)
	index := len(r.scripts) - 1
	stdout := ""
	if index < len(r.stdouts) {
		stdout = r.stdouts[index]
	}
	if r.errAt >= 0 && index == r.errAt {
		return stdout, r.stderr, errors.New("step failed")
	}
	return stdout, r.stderr, nil
}

func (r *multiStepDeploySSHRunner) Close() error {
	return nil
}

func TestDeployTaskLifecycleCreatesHostDetailsAndSummarizesResult(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "nginx",
		Version:        "1.26",
		InstallCommand: "apt-get install -y nginx",
		Status:         "enabled",
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	scope := deployTestBizScope{Code: "his-dev", Name: "HIS 开发", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{Hostname: "deploy-host-1", IP: "10.40.0.11", OS: "linux", Status: "assigned", BusinessScopeID: scope.ID, BusinessScopeName: scope.Name}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "安装 Nginx",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      "host",
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    "manual",
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.Status != TaskStatusPending {
		t.Fatalf("expected pending task, got %s", task.Status)
	}

	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatalf("start task: %v", err)
	}
	if started.Status != TaskStatusRunning {
		t.Fatalf("expected running task, got %s", started.Status)
	}
	if len(started.Hosts) != 1 {
		t.Fatalf("expected one task host, got %d", len(started.Hosts))
	}
	if started.Hosts[0].HostIP != host.IP || started.Hosts[0].Status != TaskHostStatusRunning {
		t.Fatalf("unexpected task host snapshot: %+v", started.Hosts[0])
	}

	updated, err := svc.MarkHostResult(started.Hosts[0].ID, MarkHostResultRequest{
		Status: "success",
		Stdout: "nginx installed",
	}, "1")
	if err != nil {
		t.Fatalf("mark host result: %v", err)
	}
	if updated.Status != TaskHostStatusSuccess {
		t.Fatalf("expected host success, got %s", updated.Status)
	}
	var hostState deployTestHost
	if err := db.First(&hostState, host.ID).Error; err != nil {
		t.Fatalf("query host: %v", err)
	}
	if hostState.Status != "online" {
		t.Fatalf("expected host status online, got %s", hostState.Status)
	}
	var installed []deployInstalledComponent
	if err := json.Unmarshal(hostState.InstalledComponents, &installed); err != nil {
		t.Fatalf("unmarshal installed components: %v", err)
	}
	if len(installed) != 1 || installed[0].Name != "nginx" || installed[0].Version != "1.26" {
		t.Fatalf("unexpected installed components: %+v", installed)
	}

	detail, err := svc.GetTask(task.ID)
	if err != nil {
		t.Fatalf("get task detail: %v", err)
	}
	if detail.Status != TaskStatusSuccess {
		t.Fatalf("expected task success after all hosts succeeded, got %s", detail.Status)
	}
}

func TestDeployTaskUsesGroupTargets(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	pkg, err := svc.CreatePackage(CreatePackageRequest{Name: "node-exporter", Version: "1.8", InstallCommand: "install node-exporter"}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	scope := deployTestBizScope{Code: "monitor-dev", Name: "监控开发", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	matched := deployTestHost{
		Hostname:          "monitor-1",
		IP:                "10.40.0.21",
		OS:                "linux",
		Status:            "online",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
		LabelValues:       datatypes.JSON([]byte(`[{"key":"role","val":"monitor"}]`)),
	}
	other := deployTestHost{
		Hostname:          "app-1",
		IP:                "10.40.0.22",
		OS:                "linux",
		Status:            "online",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
		LabelValues:       datatypes.JSON([]byte(`[{"key":"role","val":"app"}]`)),
	}
	if err := db.Create(&matched).Error; err != nil {
		t.Fatalf("seed matched host: %v", err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("seed other host: %v", err)
	}
	group := deployTestGroup{
		Name:       "监控服务器",
		Conditions: datatypes.JSON([]byte(`{"operator":"AND","rules":[{"key":"role","op":"eq","val":"monitor"}]}`)),
	}
	if err := db.Create(&group).Error; err != nil {
		t.Fatalf("seed group: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:         "安装 Node Exporter",
		PackageID:    pkg.ID,
		TargetType:   "group",
		TargetIDs:    []uint64{group.ID},
		ExecutorType: "simulated",
	}, "1", nil)
	if err != nil {
		t.Fatalf("create group task: %v", err)
	}

	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatalf("start group task: %v", err)
	}
	if len(started.Hosts) != 1 {
		t.Fatalf("expected one matched host, got %d", len(started.Hosts))
	}
	if started.Hosts[0].HostID != matched.ID {
		t.Fatalf("expected matched host %d, got %+v", matched.ID, started.Hosts[0])
	}
}

func TestGetPackageReturnsDeploymentStats(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "nginx",
		Version:        "1.30.2",
		ExecutionMode:  "fixed",
		TemplateCode:   "nginx_systemd",
		SourceFileName: "nginx-1.30.2.tar.gz",
		Status:         "enabled",
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	scope := deployTestBizScope{Code: "his-dev", Name: "HIS 开发", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{Hostname: "pkg-detail-host", IP: "10.40.0.31", OS: "linux", Status: "assigned", BusinessScopeID: scope.ID, BusinessScopeName: scope.Name}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "安装 Nginx 组件",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      "host",
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    "manual",
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatalf("start task: %v", err)
	}
	if _, err := svc.MarkHostResult(started.Hosts[0].ID, MarkHostResultRequest{
		Status: "success",
		Stdout: "installed",
	}, "1"); err != nil {
		t.Fatalf("mark host result: %v", err)
	}

	detail, err := svc.GetPackage(pkg.ID)
	if err != nil {
		t.Fatalf("get package: %v", err)
	}
	if detail.LatestTaskName != "安装 Nginx 组件" {
		t.Fatalf("unexpected latest task name: %s", detail.LatestTaskName)
	}
	if detail.LatestTaskStatus != TaskStatusSuccess {
		t.Fatalf("unexpected latest task status: %s", detail.LatestTaskStatus)
	}
	if detail.LatestSuccessCount != 1 {
		t.Fatalf("unexpected latest success count: %d", detail.LatestSuccessCount)
	}
	if detail.LatestDeployedAt == nil {
		t.Fatal("expected latest deployed at to be populated")
	}
}

func TestDeployTaskGroupTargetInheritsParentConditions(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	pkg, err := svc.CreatePackage(CreatePackageRequest{Name: "node-exporter", Version: "1.9", InstallCommand: "install node-exporter"}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	scope := deployTestBizScope{Code: "xian-dev", Name: "西安开发", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	xianMonitor := deployTestHost{
		Hostname:          "xian-monitor-1",
		IP:                "10.40.1.21",
		OS:                "linux",
		Status:            "online",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
		LabelValues:       datatypes.JSON([]byte(`[{"key":"region","val":"西安开发环境"},{"key":"role","val":"monitor"}]`)),
	}
	xianApp := deployTestHost{
		Hostname:          "xian-app-1",
		IP:                "10.40.1.22",
		OS:                "linux",
		Status:            "online",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
		LabelValues:       datatypes.JSON([]byte(`[{"key":"region","val":"西安开发环境"},{"key":"role","val":"app"}]`)),
	}
	shanghaiMonitor := deployTestHost{
		Hostname:          "shanghai-monitor-1",
		IP:                "10.40.1.23",
		OS:                "linux",
		Status:            "online",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
		LabelValues:       datatypes.JSON([]byte(`[{"key":"region","val":"上海开发环境"},{"key":"role","val":"monitor"}]`)),
	}
	if err := db.Create(&xianMonitor).Error; err != nil {
		t.Fatalf("seed xian monitor host: %v", err)
	}
	if err := db.Create(&xianApp).Error; err != nil {
		t.Fatalf("seed xian app host: %v", err)
	}
	if err := db.Create(&shanghaiMonitor).Error; err != nil {
		t.Fatalf("seed shanghai monitor host: %v", err)
	}

	parent := deployTestGroup{
		Name:       "西安开发环境",
		Conditions: datatypes.JSON([]byte(`{"operator":"AND","rules":[{"key":"region","op":"eq","val":"西安开发环境"}]}`)),
	}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("seed parent group: %v", err)
	}
	child := deployTestGroup{
		ParentID:   parent.ID,
		Name:       "监控服务器",
		Conditions: datatypes.JSON([]byte(`{"operator":"AND","rules":[{"key":"role","op":"eq","val":"monitor"}]}`)),
	}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("seed child group: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:         "子分组安装 Node Exporter",
		PackageID:    pkg.ID,
		TargetType:   "group",
		TargetIDs:    []uint64{child.ID},
		ExecutorType: "simulated",
	}, "1", nil)
	if err != nil {
		t.Fatalf("create child group task: %v", err)
	}

	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatalf("start child group task: %v", err)
	}
	if len(started.Hosts) != 1 {
		t.Fatalf("expected one inherited matched host, got %d: %+v", len(started.Hosts), started.Hosts)
	}
	if started.Hosts[0].HostID != xianMonitor.ID {
		t.Fatalf("expected xian monitor host %d, got %+v", xianMonitor.ID, started.Hosts[0])
	}
}

func TestDeployTaskSSHExecutorRunsInstallAndWritesBackHostState(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	svc.sshRunnerFactory = func(host cmdbHostSnapshot, req StartTaskRequest) (deploySSHRunner, error) {
		if req.SSHUser != "root" {
			t.Fatalf("unexpected ssh user: %s", req.SSHUser)
		}
		return &fakeDeploySSHRunner{stdout: "nginx installed"}, nil
	}

	scope := deployTestBizScope{Code: "his-dev", Name: "HIS 开发", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "nginx",
		Version:        "1.30.2",
		InstallCommand: "echo nginx installed",
		Status:         "enabled",
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	host := deployTestHost{
		Hostname:          "deploy-host-ssh-1",
		IP:                "10.40.0.51",
		SSHPort:           22,
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "SSH 安装 Nginx",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      "host",
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeSSH,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	started, err := svc.StartTask(task.ID, StartTaskRequest{
		SSHUser:         "root",
		SSHPassword:     "secret",
		HostFingerprint: "SHA256:test",
		AuthMode:        "password",
	}, "1", nil)
	if err != nil {
		t.Fatalf("start ssh task: %v", err)
	}
	if started.Status != TaskStatusSuccess {
		t.Fatalf("expected success task, got %s", started.Status)
	}
	if len(started.Hosts) != 1 || started.Hosts[0].Status != TaskHostStatusSuccess {
		t.Fatalf("unexpected task hosts: %+v", started.Hosts)
	}
	if strings.TrimSpace(started.Hosts[0].Stdout) != "nginx installed" {
		t.Fatalf("unexpected stdout: %+v", started.Hosts[0])
	}

	var hostState deployTestHost
	if err := db.First(&hostState, host.ID).Error; err != nil {
		t.Fatalf("query host: %v", err)
	}
	if hostState.Status != "online" {
		t.Fatalf("expected host online, got %s", hostState.Status)
	}
	var installed []deployInstalledComponent
	if err := json.Unmarshal(hostState.InstalledComponents, &installed); err != nil {
		t.Fatalf("unmarshal installed components: %v", err)
	}
	if len(installed) != 1 || installed[0].Name != "nginx" || installed[0].Version != "1.30.2" {
		t.Fatalf("unexpected installed components: %+v", installed)
	}
}

func TestDeployPackageFixedTemplateRoundTrip(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "nginx",
		Version:        "1.30.2",
		ExecutionMode:  ExecutionModeFixed,
		TemplateCode:   TemplateCodeNginxSystemd,
		TemplateConfig: map[string]any{"scenario": "systemd"},
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create fixed template package: %v", err)
	}
	if pkg.TemplateCode != TemplateCodeNginxSystemd {
		t.Fatalf("expected template code %s, got %s", TemplateCodeNginxSystemd, pkg.TemplateCode)
	}
	if got := pkg.TemplateConfig["scenario"]; got != "systemd" {
		t.Fatalf("expected template config scenario=systemd, got %#v", got)
	}
}

func TestDeployTaskCreatesWithTemplateParams(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scope := deployTestBizScope{Code: "his-dev", Name: "HIS 开发", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "deploy-template-host-1",
		IP:                "10.40.0.71",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}

	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:          "nginx",
		Version:       "1.30.2",
		ExecutionMode: ExecutionModeFixed,
		TemplateCode:  TemplateCodeNginxSystemd,
		Status:        PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "固定模板安装 Nginx",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeSSH,
		TemplateParams: map[string]any{
			"installRoot": "/data/nginx",
			"serviceName": "nginx",
		},
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task with template params: %v", err)
	}
	if task.TemplateParams["installRoot"] != "/data/nginx" {
		t.Fatalf("expected installRoot template param, got %#v", task.TemplateParams)
	}
	if task.TemplateParams["serviceName"] != "nginx" {
		t.Fatalf("expected serviceName template param, got %#v", task.TemplateParams)
	}
}

func TestDeployTaskFixedTemplateSSHExecutorRendersScript(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	runner := &fakeDeploySSHRunner{stdout: "nginx template installed"}
	svc.sshRunnerFactory = func(host cmdbHostSnapshot, req StartTaskRequest) (deploySSHRunner, error) {
		return runner, nil
	}

	scope := deployTestBizScope{Code: "his-dev", Name: "HIS 开发", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "deploy-template-host-2",
		IP:                "10.40.0.72",
		SSHPort:           22,
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}

	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:          "nginx",
		Version:       "1.30.2",
		ExecutionMode: ExecutionModeFixed,
		TemplateCode:  TemplateCodeNginxSystemd,
		Status:        PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "固定模板 SSH 安装 Nginx",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeSSH,
		TemplateParams: map[string]any{
			"installRoot": "/data/nginx",
			"serviceName": "nginx",
		},
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	started, err := svc.StartTask(task.ID, StartTaskRequest{
		SSHUser:         "root",
		SSHPassword:     "secret",
		HostFingerprint: "SHA256:test",
		AuthMode:        "password",
	}, "1", nil)
	if err != nil {
		t.Fatalf("start task: %v", err)
	}
	if started.Status != TaskStatusSuccess {
		t.Fatalf("expected success task, got %s", started.Status)
	}
	if !strings.Contains(runner.script, "/data/nginx") {
		t.Fatalf("expected script to contain install root, got: %s", runner.script)
	}
	if !strings.Contains(runner.script, "nginx-1.30.2.tar.gz") {
		t.Fatalf("expected script to contain nginx tarball, got: %s", runner.script)
	}
	if !strings.Contains(runner.script, "nginx.service") {
		t.Fatalf("expected script to contain service name, got: %s", runner.script)
	}
	if len(started.Hosts[0].TraceSteps) < 4 {
		t.Fatalf("expected trace steps to be recorded, got %+v", started.Hosts[0].TraceSteps)
	}
	if phase := started.Hosts[0].TraceSteps[0]["phase"]; phase != "connect" {
		t.Fatalf("expected first trace phase connect, got %#v", phase)
	}

	var hostState deployTestHost
	if err := db.First(&hostState, host.ID).Error; err != nil {
		t.Fatalf("query host: %v", err)
	}
	if hostState.Status != "online" {
		t.Fatalf("expected host online, got %s", hostState.Status)
	}
}

func TestDeployTaskFixedTemplateRejectsMissingParams(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scope := deployTestBizScope{Code: "his-dev", Name: "HIS 开发", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "deploy-template-host-3",
		IP:                "10.40.0.73",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:          "nginx",
		Version:       "1.30.2",
		ExecutionMode: ExecutionModeFixed,
		TemplateCode:  TemplateCodeNginxSystemd,
		Status:        PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	_, err = svc.CreateTask(CreateTaskRequest{
		Name:            "错误模板参数任务",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeSSH,
		TemplateParams: map[string]any{
			"installRoot": "/data/nginx",
		},
	}, "1", nil)
	if err == nil || err.Error() != "deploytask.template_params_invalid" {
		t.Fatalf("expected template param invalid error, got %v", err)
	}
}

func TestDeployTaskFixedTemplateUsesUploadedSourceURL(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	runner := &fakeDeploySSHRunner{stdout: "nginx offline installed"}
	svc.sshRunnerFactory = func(host cmdbHostSnapshot, req StartTaskRequest) (deploySSHRunner, error) {
		return runner, nil
	}

	scope := deployTestBizScope{Code: "his-dev", Name: "HIS 开发", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "deploy-template-host-4",
		IP:                "10.40.0.74",
		SSHPort:           22,
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}

	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:            "nginx",
		Version:         "1.30.2",
		ExecutionMode:   ExecutionModeFixed,
		TemplateCode:    TemplateCodeNginxSystemd,
		SourceObjectKey: "deploy/package/nginx-1.30.2.tar.gz",
		SourceFileName:  "nginx-1.30.2.tar.gz",
		SourceURL:       "https://files.internal.local/deploy/nginx-1.30.2.tar.gz",
		Status:          PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "离线源码安装 Nginx",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeSSH,
		TemplateParams: map[string]any{
			"installRoot": "/data/nginx",
			"serviceName": "nginx",
		},
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	_, err = svc.StartTask(task.ID, StartTaskRequest{
		SSHUser:         "root",
		SSHPassword:     "secret",
		HostFingerprint: "SHA256:test",
		AuthMode:        "password",
	}, "1", nil)
	if err != nil {
		t.Fatalf("start task: %v", err)
	}
	if !strings.Contains(runner.script, `SOURCE_URL="https://files.internal.local/deploy/nginx-1.30.2.tar.gz"`) {
		t.Fatalf("expected uploaded source url in script, got: %s", runner.script)
	}
	if strings.Contains(runner.script, "https://nginx.org/download/") {
		t.Fatalf("expected offline script path without public download fallback, got: %s", runner.script)
	}
}

func TestDeployFixedTemplateScriptsRenderForBuiltInComponents(t *testing.T) {
	cases := []struct {
		name         string
		pkg          DeployPackage
		task         DeployTask
		expectScript []string
	}{
		{
			name: "mysql",
			pkg: DeployPackage{
				Name:            "mysql",
				Version:         "8.0.39",
				ExecutionMode:   ExecutionModeFixed,
				TemplateCode:    TemplateCodeMySQLSystemd,
				SourceObjectKey: "deploy/package/mysql-8.0.39.tar.xz",
				SourceFileName:  "mysql-8.0.39.tar.xz",
				SourceURL:       "https://files.internal.local/mysql-8.0.39.tar.xz",
			},
			task: DeployTask{
				Action:         TaskActionInstall,
				TemplateParams: datatypes.JSON([]byte(`{"installRoot":"/data/mysql","dataRoot":"/data/mysql/data","serviceName":"mysqld","port":"3306","rootPassword":"Secret_123"}`)),
			},
			expectScript: []string{"mysql-8.0.39.tar.xz", "ROOT_PASSWORD=\"Secret_123\"", "ExecStart=/data/mysql/bin/mysqld", "--connect-timeout 15 --max-time 300 --retry 2 --retry-delay 2", `systemctl stop "mysqld" >/dev/null 2>&1 || true`, `pkill -f "/data/mysql/bin/mysqld" >/dev/null 2>&1 || true`},
		},
		{
			name: "redis",
			pkg: DeployPackage{
				Name:          "redis",
				Version:       "7.2.5",
				ExecutionMode: ExecutionModeFixed,
				TemplateCode:  TemplateCodeRedisSystemd,
			},
			task: DeployTask{
				Action:         TaskActionInstall,
				TemplateParams: datatypes.JSON([]byte(`{"installRoot":"/data/redis","dataRoot":"/data/redis/data","serviceName":"redis","port":"6379","requirePassword":"Redis_123"}`)),
			},
			expectScript: []string{"download.redis.io/releases/redis-7.2.5.tar.gz", "requirepass Redis_123", "redis-server /data/redis/conf/redis.conf", "--connect-timeout 15 --max-time 300 --retry 2 --retry-delay 2", `systemctl stop "redis" >/dev/null 2>&1 || true`, `pkill -f "/data/redis/bin/redis-server" >/dev/null 2>&1 || true`},
		},
		{
			name: "minio",
			pkg: DeployPackage{
				Name:          "minio",
				Version:       "2025-05-24",
				ExecutionMode: ExecutionModeFixed,
				TemplateCode:  TemplateCodeMinIOSystemd,
			},
			task: DeployTask{
				Action:         TaskActionInstall,
				TemplateParams: datatypes.JSON([]byte(`{"installRoot":"/data/minio","dataRoot":"/data/minio/data","serviceName":"minio","apiPort":"9000","consolePort":"9001","rootUser":"minioadmin","rootPassword":"Minio_123"}`)),
			},
			expectScript: []string{"dl.minio.org.cn/server/minio/release/linux-amd64/minio", "MINIO_ROOT_USER=minioadmin", "ExecStart=/data/minio/bin/minio server", "--connect-timeout 15 --max-time 300 --retry 2 --retry-delay 2", `systemctl stop "minio" >/dev/null 2>&1 || true`, `pkill -f "/data/minio/bin/minio" >/dev/null 2>&1 || true`},
		},
		{
			name: "harbor",
			pkg: DeployPackage{
				Name:            "harbor",
				Version:         "2.11.1",
				ExecutionMode:   ExecutionModeFixed,
				TemplateCode:    TemplateCodeHarborOffline,
				SourceObjectKey: "deploy/package/harbor-offline-installer-v2.11.1.tgz",
				SourceFileName:  "harbor-offline-installer-v2.11.1.tgz",
				SourceURL:       "https://files.internal.local/harbor-offline-installer-v2.11.1.tgz",
			},
			task: DeployTask{
				Action:         TaskActionInstall,
				TemplateParams: datatypes.JSON([]byte(`{"installRoot":"/data/harbor","dataRoot":"/data/harbor/data","hostname":"harbor.local","httpPort":"8088","adminPassword":"Harbor_123"}`)),
			},
			expectScript: []string{"harbor-offline-installer-v2.11.1.tgz", "hostname: $HARBOR_HOSTNAME", "./install.sh"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			script, err := renderFixedTemplateScript(tc.pkg, tc.task)
			if err != nil {
				t.Fatalf("render script: %v", err)
			}
			for _, expected := range tc.expectScript {
				if !strings.Contains(script, expected) {
					t.Fatalf("expected script to contain %q, got: %s", expected, script)
				}
			}
		})
	}
}

func TestDeployFixedTemplateScriptsRequireMandatoryParams(t *testing.T) {
	err := validateTemplateParams(
		ExecutionModeFixed,
		TemplateCodeMySQLSystemd,
		nil,
		map[string]any{
			"installRoot": "/data/mysql",
			"dataRoot":    "/data/mysql/data",
			"serviceName": "mysqld",
			"port":        "3306",
		},
	)
	if err == nil || err.Error() != "deploytask.template_params_invalid" {
		t.Fatalf("expected template param invalid error, got %v", err)
	}
}

func TestDeployTaskTruncatesLargeExecutionLogs(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	largeStdout := strings.Repeat("x", 70000)
	runner := &fakeDeploySSHRunner{stdout: largeStdout}
	svc.sshRunnerFactory = func(host cmdbHostSnapshot, req StartTaskRequest) (deploySSHRunner, error) {
		return runner, nil
	}

	scope := deployTestBizScope{Code: "his-dev", Name: "HIS 开发", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "deploy-large-log-host",
		IP:                "10.40.0.95",
		SSHPort:           22,
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "redis",
		Version:        "7.2.5",
		InstallCommand: "echo install redis",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "超长日志任务",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeSSH,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	started, err := svc.StartTask(task.ID, StartTaskRequest{
		SSHUser:         "root",
		SSHPassword:     "secret",
		HostFingerprint: "SHA256:test",
		AuthMode:        "password",
	}, "1", nil)
	if err != nil {
		t.Fatalf("start task: %v", err)
	}
	if started.Status != TaskStatusSuccess {
		t.Fatalf("expected success task, got %s", started.Status)
	}
	if len(started.Hosts) != 1 {
		t.Fatalf("expected one task host, got %+v", started.Hosts)
	}
	if len(started.Hosts[0].Stdout) > 60000 {
		t.Fatalf("expected truncated stdout length <= 60000, got %d", len(started.Hosts[0].Stdout))
	}
	if !strings.Contains(started.Hosts[0].Stdout, "[truncated]") {
		t.Fatalf("expected truncation marker, got length=%d", len(started.Hosts[0].Stdout))
	}
}

func TestDeployTaskUninstallWritebackReturnsHostToAssignedAndRemovesComponent(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	svc.sshRunnerFactory = func(host cmdbHostSnapshot, req StartTaskRequest) (deploySSHRunner, error) {
		return &fakeDeploySSHRunner{stdout: "nginx removed"}, nil
	}

	scope := deployTestBizScope{Code: "his-dev", Name: "HIS 开发", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	installedJSON := datatypes.JSON([]byte(`[{"name":"nginx","version":"1.30.2"},{"name":"redis","version":"7.2"}]`))
	host := deployTestHost{
		Hostname:            "deploy-template-host-uninstall-1",
		IP:                  "10.40.0.75",
		SSHPort:             22,
		OS:                  "linux",
		Status:              "online",
		BusinessScopeID:     scope.ID,
		BusinessScopeName:   scope.Name,
		InstalledComponents: installedJSON,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}

	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:             "nginx",
		Version:          "1.30.2",
		ExecutionMode:    ExecutionModeFixed,
		TemplateCode:     TemplateCodeNginxSystemd,
		UninstallCommand: "systemctl stop nginx",
		Status:           PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "卸载 Nginx",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		Action:          TaskActionUninstall,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeSSH,
		TemplateParams: map[string]any{
			"serviceName": "nginx",
			"installRoot": "/data/nginx",
		},
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	started, err := svc.StartTask(task.ID, StartTaskRequest{
		SSHUser:         "root",
		SSHPassword:     "secret",
		HostFingerprint: "SHA256:test",
		AuthMode:        "password",
	}, "1", nil)
	if err != nil {
		t.Fatalf("start task: %v", err)
	}
	if started.Status != TaskStatusSuccess {
		t.Fatalf("expected success task, got %s", started.Status)
	}

	var hostState deployTestHost
	if err := db.First(&hostState, host.ID).Error; err != nil {
		t.Fatalf("query host: %v", err)
	}
	if hostState.Status != "online" {
		t.Fatalf("expected host remain online after uninstalling one of multiple components, got %s", hostState.Status)
	}
	var installed []deployInstalledComponent
	if err := json.Unmarshal(hostState.InstalledComponents, &installed); err != nil {
		t.Fatalf("unmarshal installed components: %v", err)
	}
	if len(installed) != 1 || installed[0].Name != "redis" {
		t.Fatalf("expected nginx removed and redis kept, got %+v", installed)
	}
}

func TestDeployTaskReinstallAllowsAssignedHost(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scope := deployTestBizScope{Code: "his-dev", Name: "HIS 开发", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "deploy-reinstall-host-1",
		IP:                "10.40.0.76",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:          "nginx",
		Version:       "1.30.2",
		ExecutionMode: ExecutionModeFixed,
		TemplateCode:  TemplateCodeNginxSystemd,
		Status:        PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "重装 Nginx",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		Action:          TaskActionReinstall,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeManual,
		TemplateParams: map[string]any{
			"serviceName": "nginx",
			"installRoot": "/data/nginx",
		},
	}, "1", nil)
	if err != nil {
		t.Fatalf("create reinstall task: %v", err)
	}
	if task.Action != TaskActionReinstall {
		t.Fatalf("expected reinstall action, got %s", task.Action)
	}
}

func TestUpsertHostInstalledComponentHandlesExistingJSONColumn(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	host := deployTestHost{
		Hostname:            "deploy-host-json-1",
		IP:                  "10.40.0.61",
		SSHPort:             22,
		OS:                  "linux",
		Status:              "online",
		InstalledComponents: datatypes.JSON([]byte(`[{"name":"redis","version":"7.2"}]`)),
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}

	now := time.Now()
	if err := svc.upsertHostInstalledComponent(host.ID, 99, "安装 Nginx", ExecutorTypeSSH, "nginx", "1.30.2", "1", now); err != nil {
		t.Fatalf("upsert host installed component: %v", err)
	}

	var hostState deployTestHost
	if err := db.First(&hostState, host.ID).Error; err != nil {
		t.Fatalf("query host: %v", err)
	}
	var installed []deployInstalledComponent
	if err := json.Unmarshal(hostState.InstalledComponents, &installed); err != nil {
		t.Fatalf("unmarshal installed components: %v", err)
	}
	if len(installed) != 2 {
		t.Fatalf("expected 2 installed components, got %+v", installed)
	}
	if installed[0].Name != "redis" || installed[1].Name != "nginx" {
		t.Fatalf("unexpected installed components order/content: %+v", installed)
	}
	if installed[1].DeployTaskID != 99 || installed[1].DeployTaskName != "安装 Nginx" || installed[1].ExecutorType != ExecutorTypeSSH {
		t.Fatalf("unexpected installed component metadata: %+v", installed[1])
	}
}

func TestDeployTaskDerivesPackageAndActionFromTemplate(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scope := deployTestBizScope{Code: "his-dev", Name: "HIS 开发", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "deploy-template-default-host",
		IP:                "10.40.0.91",
		OS:                "linux",
		Status:            "online",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:             "nginx",
		Version:          "1.30.2",
		ExecutionMode:    ExecutionModeFixed,
		TemplateCode:     TemplateCodeNginxSystemd,
		UninstallCommand: "systemctl stop nginx",
		Status:           PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	template, err := svc.CreateTemplate(CreateTemplateRequest{
		Name:          "nginx lifecycle",
		Version:       "v1",
		PackageID:     pkg.ID,
		DefaultAction: TaskActionUninstall,
		Status:        TemplateStatusEnabled,
		ParameterSchema: map[string]any{
			"installRoot": "/data/nginx",
			"serviceName": "nginx",
		},
	}, "1")
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "模板默认动作任务",
		TemplateID:      template.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.TemplateID != template.ID || task.TemplateName != template.Name {
		t.Fatalf("expected template metadata on task, got %+v", task)
	}
	if task.PackageID != pkg.ID || task.PackageName != pkg.Name {
		t.Fatalf("expected derived package from template, got %+v", task)
	}
	if task.Action != TaskActionUninstall {
		t.Fatalf("expected task action %s, got %s", TaskActionUninstall, task.Action)
	}
	if task.TemplateParams["installRoot"] != "/data/nginx" {
		t.Fatalf("expected template params copied from template, got %+v", task.TemplateParams)
	}
}

func TestDeployTemplateTaskSSHExecutorRunsAllTemplateSteps(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	runner := &multiStepDeploySSHRunner{stdouts: []string{"nginx installed", "redis installed"}, errAt: -1}
	svc.sshRunnerFactory = func(host cmdbHostSnapshot, req StartTaskRequest) (deploySSHRunner, error) {
		return runner, nil
	}

	scope := deployTestBizScope{Code: "his-dev", Name: "HIS 开发", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "deploy-template-steps-host",
		IP:                "10.40.0.92",
		SSHPort:           22,
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	nginxPkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:          "nginx",
		Version:       "1.30.2",
		ExecutionMode: ExecutionModeFixed,
		TemplateCode:  TemplateCodeNginxSystemd,
		Status:        PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create nginx package: %v", err)
	}
	redisPkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "redis",
		Version:        "7.2.0",
		InstallCommand: "echo install redis",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create redis package: %v", err)
	}
	template, err := svc.CreateTemplate(CreateTemplateRequest{
		Name:          "middleware bootstrap",
		Version:       "v1",
		PackageID:     nginxPkg.ID,
		DefaultAction: TaskActionInstall,
		Status:        TemplateStatusEnabled,
		ParameterSchema: map[string]any{
			"installRoot": "/data/nginx",
			"serviceName": "nginx",
		},
		Steps: []TemplateStepPayload{
			{
				StepCode:  "nginx_install",
				StepName:  "安装 Nginx",
				StepType:  "package",
				Action:    TaskActionInstall,
				PackageID: nginxPkg.ID,
				TemplateParams: map[string]any{
					"installRoot": "/data/nginx",
					"serviceName": "nginx",
				},
				Sort: 1,
			},
			{
				StepCode:  "redis_install",
				StepName:  "安装 Redis",
				StepType:  "package",
				Action:    TaskActionInstall,
				PackageID: redisPkg.ID,
				Sort:      2,
			},
		},
	}, "1")
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "模板多步骤 SSH 任务",
		TemplateID:      template.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeSSH,
		TemplateParams: map[string]any{
			"installRoot": "/data/nginx",
			"serviceName": "nginx",
		},
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	started, err := svc.StartTask(task.ID, StartTaskRequest{
		SSHUser:         "root",
		SSHPassword:     "secret",
		HostFingerprint: "SHA256:test",
		AuthMode:        "password",
	}, "1", nil)
	if err != nil {
		t.Fatalf("start task: %v", err)
	}
	if started.Status != TaskStatusSuccess {
		t.Fatalf("expected success task, got %s", started.Status)
	}
	if len(runner.scripts) != 2 {
		t.Fatalf("expected two rendered scripts, got %d", len(runner.scripts))
	}
	if !strings.Contains(runner.scripts[0], "nginx-1.30.2.tar.gz") {
		t.Fatalf("expected nginx script first, got %s", runner.scripts[0])
	}
	if !strings.Contains(runner.scripts[1], "install redis") {
		t.Fatalf("expected redis script second, got %s", runner.scripts[1])
	}
	if len(started.Hosts) != 1 {
		t.Fatalf("expected one host result, got %+v", started.Hosts)
	}
	traceSteps := started.Hosts[0].TraceSteps
	if len(traceSteps) < 5 {
		t.Fatalf("expected trace steps for multi-step execution, got %+v", traceSteps)
	}
	if !strings.Contains(started.Hosts[0].Stdout, "nginx installed") || !strings.Contains(started.Hosts[0].Stdout, "redis installed") {
		t.Fatalf("expected merged stdout, got %s", started.Hosts[0].Stdout)
	}

	var hostState deployTestHost
	if err := db.First(&hostState, host.ID).Error; err != nil {
		t.Fatalf("query host: %v", err)
	}
	if hostState.Status != "online" {
		t.Fatalf("expected host online, got %s", hostState.Status)
	}
	var installed []deployInstalledComponent
	if err := json.Unmarshal(hostState.InstalledComponents, &installed); err != nil {
		t.Fatalf("unmarshal installed components: %v", err)
	}
	if len(installed) != 2 {
		t.Fatalf("expected two installed components, got %+v", installed)
	}
}

func TestDeployTemplateTaskSSHExecutorRunsScriptStepsWithChecks(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	runner := &multiStepDeploySSHRunner{
		stdouts: []string{"precheck ok", "prepare ok", "postcheck ok", "install ok"},
		errAt:   -1,
	}
	svc.sshRunnerFactory = func(host cmdbHostSnapshot, req StartTaskRequest) (deploySSHRunner, error) {
		return runner, nil
	}

	scope := deployTestBizScope{Code: "his-dev", Name: "HIS 开发", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "deploy-script-step-host",
		IP:                "10.40.0.40",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "nginx",
		Version:        "1.30.2",
		InstallCommand: "echo install nginx to {{installRoot}}",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	template, err := svc.CreateTemplate(CreateTemplateRequest{
		Name:          "nginx prepare",
		Version:       "v1",
		PackageID:     pkg.ID,
		DefaultAction: TaskActionInstall,
		ExecutionMode: ExecutionModeOrchestrated,
		Status:        TemplateStatusEnabled,
		ParameterSchema: map[string]any{
			"installRoot": "/data/nginx",
			"serviceName": "nginx",
		},
		Steps: []TemplateStepPayload{
			{
				StepCode: "prepare",
				StepName: "安装前准备",
				StepType: TemplateStepTypeScript,
				Action:   TaskActionInstall,
				StepConfig: map[string]any{
					"precheckCommand":  `test "{{hostIp}}" = "10.40.0.40"`,
					"script":           `echo prepare {{businessScopeName}} {{installRoot}} {{serviceName}}`,
					"postcheckCommand": `test "{{serviceName}}" = "nginx"`,
				},
				Sort: 1,
			},
			{
				StepCode:  "install",
				StepName:  "安装 Nginx",
				StepType:  TemplateStepTypePackage,
				Action:    TaskActionInstall,
				PackageID: pkg.ID,
				Sort:      2,
			},
		},
	}, "1")
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "脚本步骤 SSH 任务",
		TemplateID:      template.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeSSH,
		TemplateParams: map[string]any{
			"installRoot": "/data/nginx",
			"serviceName": "nginx",
		},
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	started, err := svc.StartTask(task.ID, StartTaskRequest{
		SSHUser:         "root",
		SSHPassword:     "secret",
		HostFingerprint: "SHA256:test",
		AuthMode:        "password",
	}, "1", nil)
	if err != nil {
		t.Fatalf("start task: %v", err)
	}
	if started.Status != TaskStatusSuccess {
		t.Fatalf("expected success task, got %s", started.Status)
	}
	if len(runner.scripts) != 4 {
		t.Fatalf("expected four rendered scripts, got %d", len(runner.scripts))
	}
	if !strings.Contains(runner.scripts[0], `test "10.40.0.40" = "10.40.0.40"`) {
		t.Fatalf("expected rendered precheck, got %s", runner.scripts[0])
	}
	if !strings.Contains(runner.scripts[1], "echo prepare HIS 开发 /data/nginx nginx") {
		t.Fatalf("expected rendered script step, got %s", runner.scripts[1])
	}
	if !strings.Contains(runner.scripts[2], `test "nginx" = "nginx"`) {
		t.Fatalf("expected rendered postcheck, got %s", runner.scripts[2])
	}
	if !strings.Contains(runner.scripts[3], "install nginx") {
		t.Fatalf("expected package command last, got %s", runner.scripts[3])
	}
	if len(started.Hosts) != 1 {
		t.Fatalf("expected one host result, got %+v", started.Hosts)
	}
	tracePayload, _ := json.Marshal(started.Hosts[0].TraceSteps)
	traceText := string(tracePayload)
	if !strings.Contains(traceText, `"phase":"precheck"`) || !strings.Contains(traceText, `"phase":"postcheck"`) {
		t.Fatalf("expected precheck and postcheck trace, got %s", traceText)
	}
	if !strings.Contains(started.Hosts[0].Stdout, "prepare ok") || !strings.Contains(started.Hosts[0].Stdout, "install ok") {
		t.Fatalf("expected merged stdout, got %s", started.Hosts[0].Stdout)
	}
}

func TestDeployTemplateTaskSSHExecutorFailsOnScriptPrecheck(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	runner := &multiStepDeploySSHRunner{
		stdouts: []string{"precheck failed"},
		stderr:  "precheck stderr",
		errAt:   0,
	}
	svc.sshRunnerFactory = func(host cmdbHostSnapshot, req StartTaskRequest) (deploySSHRunner, error) {
		return runner, nil
	}

	scope := deployTestBizScope{Code: "his-dev", Name: "HIS 开发", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "deploy-precheck-fail-host",
		IP:                "10.40.0.41",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "nginx",
		Version:        "1.30.3",
		InstallCommand: "echo install nginx",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	template, err := svc.CreateTemplate(CreateTemplateRequest{
		Name:          "nginx precheck fail",
		Version:       "v1",
		PackageID:     pkg.ID,
		DefaultAction: TaskActionInstall,
		ExecutionMode: ExecutionModeOrchestrated,
		Status:        TemplateStatusEnabled,
		Steps: []TemplateStepPayload{
			{
				StepCode: "prepare",
				StepName: "安装前校验",
				StepType: TemplateStepTypeScript,
				Action:   TaskActionInstall,
				StepConfig: map[string]any{
					"precheckCommand": `exit 1`,
					"script":          `echo should-not-run`,
				},
				Sort: 1,
			},
		},
	}, "1")
	if err != nil {
		t.Fatalf("create template: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "前置校验失败任务",
		TemplateID:      template.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeSSH,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	started, err := svc.StartTask(task.ID, StartTaskRequest{
		SSHUser:         "root",
		SSHPassword:     "secret",
		HostFingerprint: "SHA256:test",
		AuthMode:        "password",
	}, "1", nil)
	if err != nil {
		t.Fatalf("start task: %v", err)
	}
	if started.Status != TaskStatusFailed {
		t.Fatalf("expected failed task, got %s", started.Status)
	}
	if len(runner.scripts) != 1 {
		t.Fatalf("expected precheck only, got %d", len(runner.scripts))
	}
	if len(started.Hosts) != 1 || started.Hosts[0].Status != TaskHostStatusFailed {
		t.Fatalf("expected failed host, got %+v", started.Hosts)
	}
	if !strings.Contains(started.Hosts[0].ErrorMessage, "precheck failed") {
		t.Fatalf("expected precheck failure in error message, got %s", started.Hosts[0].ErrorMessage)
	}
	tracePayload, _ := json.Marshal(started.Hosts[0].TraceSteps)
	if !strings.Contains(string(tracePayload), `"phase":"step_failed"`) {
		t.Fatalf("expected step_failed trace, got %s", string(tracePayload))
	}
}
