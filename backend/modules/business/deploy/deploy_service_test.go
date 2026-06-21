package deploy

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"pantheon-ops/backend/modules/business/cmdb"
	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/testmysql"

	"github.com/gin-gonic/gin"
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

func stringPtr(value string) *string {
	return &value
}

func uint64Ptr(value uint64) *uint64 {
	return &value
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
	if task.Status != TaskStatusDraft {
		t.Fatalf("expected draft task, got %s", task.Status)
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
	}, "1", nil)
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

	detail, err := svc.GetTask(task.ID, nil)
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
	}, "1", nil); err != nil {
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
	if err == nil || err.Error() != errDeployTaskTemplateParamsInvalid {
		t.Fatalf("expected template param invalid error, got %v", err)
	}
}

func TestStartDeployTaskReturnsCanonicalSSHValidationErrors(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scope := deployTestBizScope{Code: "ssh-validate-scope", Name: "SSH Validate Scope", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "ssh-validate-host",
		IP:                "10.40.0.111",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "ssh-validate-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "SSH 校验任务",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeSSH,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	_, err = svc.StartTask(task.ID, StartTaskRequest{
		SSHUser:     "root",
		SSHPassword: "secret",
	}, "1", nil)
	if err == nil || err.Error() != errDeployTaskSSHHostKeyRequired {
		t.Fatalf("expected host key required error, got %v", err)
	}

	_, err = svc.StartTask(task.ID, StartTaskRequest{
		HostFingerprint: "SHA256:test",
		SSHPassword:     "secret",
	}, "1", nil)
	if err == nil || err.Error() != errDeployTaskSSHUserRequired {
		t.Fatalf("expected ssh user required error, got %v", err)
	}

	_, err = svc.StartTask(task.ID, StartTaskRequest{
		SSHUser:         "root",
		HostFingerprint: "SHA256:test",
		AuthMode:        "password",
	}, "1", nil)
	if err == nil || err.Error() != errDeployTaskSSHPasswordRequired {
		t.Fatalf("expected ssh password required error, got %v", err)
	}

	_, err = svc.StartTask(task.ID, StartTaskRequest{
		SSHUser:         "root",
		HostFingerprint: "SHA256:test",
		AuthMode:        "private_key",
	}, "1", nil)
	if err == nil || err.Error() != errDeployTaskSSHPrivateKeyRequired {
		t.Fatalf("expected ssh private key required error, got %v", err)
	}
}

func TestStartDeployTaskReturnsCanonicalExecutionPlanValidationErrors(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scope := deployTestBizScope{Code: "ssh-plan-scope", Name: "SSH Plan Scope", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "ssh-plan-host",
		IP:                "10.40.0.112",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}

	fixedPkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:          "ssh-plan-fixed-pkg",
		Version:       "1.0.0",
		ExecutionMode: ExecutionModeFixed,
		TemplateCode:  TemplateCodeNginxSystemd,
		Status:        PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create fixed package: %v", err)
	}
	fixedTask, err := svc.CreateTask(CreateTaskRequest{
		Name:            "SSH 模板参数缺失任务",
		PackageID:       fixedPkg.ID,
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
		t.Fatalf("create fixed task: %v", err)
	}
	if err := db.Model(&DeployTask{}).Where("id = ?", fixedTask.ID).Update("template_params", datatypes.JSON([]byte(`{"installRoot":"/data/nginx"}`))).Error; err != nil {
		t.Fatalf("corrupt template params: %v", err)
	}
	_, err = svc.StartTask(fixedTask.ID, StartTaskRequest{
		SSHUser:         "root",
		SSHPassword:     "secret",
		HostFingerprint: "SHA256:test",
		AuthMode:        "password",
	}, "1", nil)
	if err == nil || err.Error() != errDeployTaskTemplateParamsInvalid {
		t.Fatalf("expected template params invalid on start, got %v", err)
	}

	plainPkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "ssh-plan-plain-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create plain package: %v", err)
	}
	plainTask, err := svc.CreateTask(CreateTaskRequest{
		Name:            "SSH 命令缺失任务",
		PackageID:       plainPkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeSSH,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create plain task: %v", err)
	}
	if err := db.Model(&DeployPackage{}).Where("id = ?", plainPkg.ID).Update("install_command", "").Error; err != nil {
		t.Fatalf("clear install command: %v", err)
	}
	_, err = svc.StartTask(plainTask.ID, StartTaskRequest{
		SSHUser:         "root",
		SSHPassword:     "secret",
		HostFingerprint: "SHA256:test",
		AuthMode:        "password",
	}, "1", nil)
	if err == nil || err.Error() != errDeployTaskInstallCommandRequired {
		t.Fatalf("expected install command required on start, got %v", err)
	}

	missingSourcePkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:            "ssh-plan-source-pkg",
		Version:         "1.0.0",
		ExecutionMode:   ExecutionModeFixed,
		TemplateCode:    TemplateCodeHarborOffline,
		SourceObjectKey: "deploy/package/harbor-offline.tgz",
		SourceFileName:  "harbor-offline.tgz",
		Status:          PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create missing source package: %v", err)
	}
	missingSourceTask, err := svc.CreateTask(CreateTaskRequest{
		Name:            "SSH 缺少源码来源任务",
		PackageID:       missingSourcePkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeSSH,
		TemplateParams: map[string]any{
			"installRoot":   "/data/harbor",
			"dataRoot":      "/data/harbor/data",
			"hostname":      "harbor.internal.local",
			"httpPort":      "8088",
			"adminPassword": "Harbor_123",
		},
	}, "1", nil)
	if err != nil {
		t.Fatalf("create missing source task: %v", err)
	}
	_, err = svc.StartTask(missingSourceTask.ID, StartTaskRequest{
		SSHUser:         "root",
		SSHPassword:     "secret",
		HostFingerprint: "SHA256:test",
		AuthMode:        "password",
	}, "1", nil)
	if err == nil || err.Error() != errDeployTaskPackageSourceMissing {
		t.Fatalf("expected package source missing on start, got %v", err)
	}
}

func TestStartDeployTaskPersistsCanonicalSSHRunnerErrors(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	svc.sshRunnerFactory = func(host cmdbHostSnapshot, req StartTaskRequest) (deploySSHRunner, error) {
		return nil, errors.New(errDeployTaskSSHConnectFailed)
	}

	scope := deployTestBizScope{Code: "ssh-runner-scope", Name: "SSH Runner Scope", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "ssh-runner-host",
		IP:                "10.40.0.113",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "ssh-runner-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "SSH 连接失败任务",
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
	if started.Status != TaskStatusFailed {
		t.Fatalf("expected failed task, got %s", started.Status)
	}
	if len(started.Hosts) != 1 {
		t.Fatalf("expected one task host, got %+v", started.Hosts)
	}
	if started.Hosts[0].ErrorMessage != errDeployTaskSSHConnectFailed {
		t.Fatalf("expected canonical ssh connect failed error, got %+v", started.Hosts[0])
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
	if err == nil || err.Error() != errDeployTaskTemplateParamsInvalid {
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

func TestDeployTaskUpdateRebuildsTemplateSnapshotAndScope(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scopeA := deployTestBizScope{Code: "his-dev-a", Name: "HIS 开发 A", Status: "active"}
	scopeB := deployTestBizScope{Code: "his-dev-b", Name: "HIS 开发 B", Status: "active"}
	if err := db.Create(&scopeA).Error; err != nil {
		t.Fatalf("seed scope A: %v", err)
	}
	if err := db.Create(&scopeB).Error; err != nil {
		t.Fatalf("seed scope B: %v", err)
	}
	hostA := deployTestHost{
		Hostname:          "deploy-update-host-a",
		IP:                "10.40.0.101",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scopeA.ID,
		BusinessScopeName: scopeA.Name,
	}
	hostB := deployTestHost{
		Hostname:          "deploy-update-host-b",
		IP:                "10.40.0.102",
		OS:                "linux",
		Status:            "online",
		BusinessScopeID:   scopeB.ID,
		BusinessScopeName: scopeB.Name,
	}
	if err := db.Create(&hostA).Error; err != nil {
		t.Fatalf("seed host A: %v", err)
	}
	if err := db.Create(&hostB).Error; err != nil {
		t.Fatalf("seed host B: %v", err)
	}

	pkgA, err := svc.CreatePackage(CreatePackageRequest{
		Name:          "nginx",
		Version:       "1.30.2",
		ExecutionMode: ExecutionModeFixed,
		TemplateCode:  TemplateCodeNginxSystemd,
		Status:        PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package A: %v", err)
	}
	pkgB, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "redis",
		Version:        "7.2.0",
		InstallCommand: "echo install redis",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package B: %v", err)
	}
	templateA, err := svc.CreateTemplate(CreateTemplateRequest{
		Name:          "nginx install",
		Version:       "v1",
		PackageID:     pkgA.ID,
		DefaultAction: TaskActionInstall,
		Status:        TemplateStatusEnabled,
		ParameterSchema: map[string]any{
			"installRoot": "/data/nginx",
			"serviceName": "nginx",
		},
	}, "1")
	if err != nil {
		t.Fatalf("create template A: %v", err)
	}
	templateB, err := svc.CreateTemplate(CreateTemplateRequest{
		Name:          "redis uninstall",
		Version:       "v2",
		PackageID:     pkgB.ID,
		DefaultAction: TaskActionUninstall,
		Status:        TemplateStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create template B: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "待编辑任务",
		TemplateID:      templateA.ID,
		BusinessScopeID: scopeA.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{hostA.ID},
		ExecutorType:    ExecutorTypeSSH,
		TemplateParams: map[string]any{
			"installRoot": "/data/nginx",
			"serviceName": "nginx",
		},
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	updated, err := svc.UpdateTask(task.ID, UpdateTaskRequest{
		Name:            stringPtr("切换模板后的任务"),
		TemplateID:      uint64Ptr(templateB.ID),
		BusinessScopeID: uint64Ptr(scopeB.ID),
		TargetType:      stringPtr(TargetTypeHost),
		TargetIDs:       []uint64{hostB.ID},
		ExecutorType:    stringPtr(ExecutorTypeManual),
		Remark:          stringPtr("updated by test"),
	}, "1", nil)
	if err != nil {
		t.Fatalf("update task: %v", err)
	}
	if updated.Name != "切换模板后的任务" {
		t.Fatalf("unexpected task name: %s", updated.Name)
	}
	if updated.TemplateID != templateB.ID || updated.TemplateName != templateB.Name || updated.TemplateVersion != templateB.Version {
		t.Fatalf("expected template B snapshot, got %+v", updated)
	}
	if updated.PackageID != pkgB.ID || updated.PackageName != pkgB.Name || updated.PackageVersion != pkgB.Version {
		t.Fatalf("expected package B snapshot, got %+v", updated)
	}
	if updated.Action != TaskActionUninstall {
		t.Fatalf("expected default action from template B, got %s", updated.Action)
	}
	if updated.BusinessScopeID != scopeB.ID || updated.BusinessScopeName != scopeB.Name {
		t.Fatalf("expected scope B snapshot, got %+v", updated)
	}
	if len(updated.TargetIDs) != 1 || updated.TargetIDs[0] != hostB.ID {
		t.Fatalf("expected updated host targets, got %+v", updated.TargetIDs)
	}
	if updated.ExecutorType != ExecutorTypeManual {
		t.Fatalf("expected executor manual, got %s", updated.ExecutorType)
	}
	if updated.ExecutionMode != templateB.ExecutionMode {
		t.Fatalf("expected execution mode from template B, got %s", updated.ExecutionMode)
	}
	if updated.Remark != "updated by test" {
		t.Fatalf("unexpected remark: %s", updated.Remark)
	}
}

func TestDeployTaskUpdateSwitchesToDirectPackageAndTemplateParams(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scope := deployTestBizScope{Code: "his-direct", Name: "HIS 直连", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "deploy-update-direct-host",
		IP:                "10.40.0.111",
		OS:                "linux",
		Status:            "online",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}

	templatePkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:          "nginx",
		Version:       "1.30.2",
		ExecutionMode: ExecutionModeFixed,
		TemplateCode:  TemplateCodeNginxSystemd,
		Status:        PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create template package: %v", err)
	}
	directPkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "redis",
		Version:        "7.2.1",
		InstallCommand: "echo install redis",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create direct package: %v", err)
	}
	template, err := svc.CreateTemplate(CreateTemplateRequest{
		Name:          "nginx install",
		Version:       "v1",
		PackageID:     templatePkg.ID,
		DefaultAction: TaskActionInstall,
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
		Name:            "模板任务转组件任务",
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

	updated, err := svc.UpdateTask(task.ID, UpdateTaskRequest{
		TemplateID:   uint64Ptr(0),
		PackageID:    uint64Ptr(directPkg.ID),
		Action:       stringPtr(TaskActionUpgrade),
		ExecutorType: stringPtr(ExecutorTypeSimulated),
		TemplateParams: &map[string]any{
			"action": TaskActionUpgrade,
		},
	}, "1", nil)
	if err != nil {
		t.Fatalf("update task to direct package: %v", err)
	}
	if updated.TemplateID != 0 || updated.TemplateName != "" || updated.TemplateVersion != "" {
		t.Fatalf("expected cleared template snapshot, got %+v", updated)
	}
	if updated.PackageID != directPkg.ID || updated.PackageName != directPkg.Name || updated.PackageVersion != directPkg.Version {
		t.Fatalf("expected direct package snapshot, got %+v", updated)
	}
	if updated.Action != TaskActionUpgrade {
		t.Fatalf("expected explicit upgrade action, got %s", updated.Action)
	}
	if updated.ExecutorType != ExecutorTypeSimulated {
		t.Fatalf("expected simulated executor, got %s", updated.ExecutorType)
	}
	if updated.ExecutionMode != directPkg.ExecutionMode {
		t.Fatalf("expected execution mode from direct package, got %s", updated.ExecutionMode)
	}
	if updated.TemplateParams["action"] != TaskActionUpgrade {
		t.Fatalf("expected template params to be replaced, got %+v", updated.TemplateParams)
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
func TestDeployTaskVisibilityRespectsHostDataScope(t *testing.T) {
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
	host := deployTestHost{
		Hostname:          "deploy-host-scope",
		IP:                "10.40.0.61",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
		DeptID:            20,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "范围内外可见性测试",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      "host",
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    "manual",
	}, "1", &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 20})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 20})
	if err != nil {
		t.Fatalf("start task: %v", err)
	}

	visibleList, err := svc.ListTasks(TaskQuery{Page: 1, PageSize: 10}, &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 20})
	if err != nil {
		t.Fatalf("list visible tasks: %v", err)
	}
	if visibleList.Total != 1 {
		t.Fatalf("expected one visible task, got %d", visibleList.Total)
	}

	hiddenList, err := svc.ListTasks(TaskQuery{Page: 1, PageSize: 10}, &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 10})
	if err != nil {
		t.Fatalf("list hidden tasks: %v", err)
	}
	if hiddenList.Total != 0 {
		t.Fatalf("expected no hidden tasks, got %d", hiddenList.Total)
	}

	if _, err := svc.GetTask(started.ID, &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 10}); err == nil {
		t.Fatal("expected scoped get task to fail")
	} else if !isDeployTaskForbidden(err) {
		t.Fatalf("expected forbidden scoped get task error, got %v", err)
	}
}

func TestDeployTaskCreateReturnsCanonicalValidationErrors(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scope := deployTestBizScope{Code: "create-validate", Name: "Create Validate", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "create-validate-host",
		IP:                "10.40.0.81",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	disabledPkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "create-validate-disabled",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusDisabled,
	}, "1")
	if err != nil {
		t.Fatalf("create disabled package: %v", err)
	}

	cases := []struct {
		name string
		req  CreateTaskRequest
		want string
	}{
		{
			name: "name required",
			req: CreateTaskRequest{
				PackageID:       disabledPkg.ID,
				BusinessScopeID: scope.ID,
				TargetType:      TargetTypeHost,
				TargetIDs:       []uint64{host.ID},
				ExecutorType:    ExecutorTypeManual,
			},
			want: "business.deploy.task.nameRequired",
		},
		{
			name: "target required",
			req: CreateTaskRequest{
				Name:            "missing targets",
				PackageID:       disabledPkg.ID,
				BusinessScopeID: scope.ID,
				TargetType:      TargetTypeHost,
				ExecutorType:    ExecutorTypeManual,
			},
			want: "business.deploy.task.targetRequired",
		},
		{
			name: "invalid target type",
			req: CreateTaskRequest{
				Name:            "invalid target type",
				PackageID:       disabledPkg.ID,
				BusinessScopeID: scope.ID,
				TargetType:      "bogus",
				TargetIDs:       []uint64{host.ID},
				ExecutorType:    ExecutorTypeManual,
			},
			want: "business.deploy.task.invalidTargetType",
		},
		{
			name: "scope required",
			req: CreateTaskRequest{
				Name:         "missing scope",
				PackageID:    disabledPkg.ID,
				TargetType:   TargetTypeHost,
				TargetIDs:    []uint64{host.ID},
				ExecutorType: ExecutorTypeManual,
			},
			want: "business.deploy.task.scopeRequired",
		},
		{
			name: "scope invalid",
			req: CreateTaskRequest{
				Name:            "invalid scope",
				PackageID:       disabledPkg.ID,
				BusinessScopeID: 999999,
				TargetType:      TargetTypeHost,
				TargetIDs:       []uint64{host.ID},
				ExecutorType:    ExecutorTypeManual,
			},
			want: "business.deploy.task.scopeInvalid",
		},
		{
			name: "invalid executor type",
			req: CreateTaskRequest{
				Name:            "invalid executor",
				PackageID:       disabledPkg.ID,
				BusinessScopeID: scope.ID,
				TargetType:      TargetTypeHost,
				TargetIDs:       []uint64{host.ID},
				ExecutorType:    "bogus",
			},
			want: "business.deploy.task.invalidExecutorType",
		},
		{
			name: "invalid action",
			req: CreateTaskRequest{
				Name:            "invalid action",
				PackageID:       disabledPkg.ID,
				BusinessScopeID: scope.ID,
				Action:          "bogus",
				TargetType:      TargetTypeHost,
				TargetIDs:       []uint64{host.ID},
				ExecutorType:    ExecutorTypeManual,
			},
			want: "business.deploy.task.invalidAction",
		},
		{
			name: "package required",
			req: CreateTaskRequest{
				Name:            "missing package",
				BusinessScopeID: scope.ID,
				TargetType:      TargetTypeHost,
				TargetIDs:       []uint64{host.ID},
				ExecutorType:    ExecutorTypeManual,
			},
			want: "business.deploy.task.packageRequired",
		},
		{
			name: "package disabled",
			req: CreateTaskRequest{
				Name:            "disabled package",
				PackageID:       disabledPkg.ID,
				BusinessScopeID: scope.ID,
				TargetType:      TargetTypeHost,
				TargetIDs:       []uint64{host.ID},
				ExecutorType:    ExecutorTypeManual,
			},
			want: "business.deploy.task.packageDisabled",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreateTask(tc.req, "1", nil)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("expected %s, got %v", tc.want, err)
			}
		})
	}
}

func TestDeployTaskUpdateReturnsCanonicalValidationErrors(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scope := deployTestBizScope{Code: "update-validate", Name: "Update Validate", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "update-validate-host",
		IP:                "10.40.0.82",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "update-validate-package",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "update validation task",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	cases := []struct {
		name string
		req  UpdateTaskRequest
		want string
	}{
		{
			name: "invalid target type",
			req:  UpdateTaskRequest{TargetType: stringPtr("bogus")},
			want: "business.deploy.task.invalidTargetType",
		},
		{
			name: "invalid action",
			req:  UpdateTaskRequest{Action: stringPtr("bogus")},
			want: "business.deploy.task.invalidAction",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.UpdateTask(task.ID, tc.req, "1", nil)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("expected %s, got %v", tc.want, err)
			}
		})
	}
}

func TestDraftDeployTaskVisibilityRespectsResolvedTargets(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "redis",
		Version:        "7.2",
		InstallCommand: "install redis",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	scope := deployTestBizScope{Code: "redis-dev", Name: "Redis 开发", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "draft-scope-host",
		IP:                "10.40.0.88",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
		DeptID:            30,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "待启动范围任务",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 30})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.Status != TaskStatusDraft {
		t.Fatalf("expected draft task, got %s", task.Status)
	}

	visibleList, err := svc.ListTasks(TaskQuery{Page: 1, PageSize: 10}, &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 30})
	if err != nil {
		t.Fatalf("list visible draft tasks: %v", err)
	}
	if visibleList.Total != 1 {
		t.Fatalf("expected one visible draft task, got %d", visibleList.Total)
	}

	hiddenList, err := svc.ListTasks(TaskQuery{Page: 1, PageSize: 10}, &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 31})
	if err != nil {
		t.Fatalf("list hidden draft tasks: %v", err)
	}
	if hiddenList.Total != 0 {
		t.Fatalf("expected no draft task outside scope, got %d", hiddenList.Total)
	}

	if _, err := svc.GetTask(task.ID, &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 31}); err == nil {
		t.Fatal("expected draft scoped get task to fail")
	} else if !isDeployTaskForbidden(err) {
		t.Fatalf("expected forbidden draft scoped get task error, got %v", err)
	}
}

func TestListDeployTasksIncludesHostExecutionSummary(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scope := deployTestBizScope{Code: "task-summary-scope", Name: "任务汇总作用域", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	hostA := deployTestHost{
		Hostname:          "task-summary-host-a",
		IP:                "10.40.0.135",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	hostB := deployTestHost{
		Hostname:          "task-summary-host-b",
		IP:                "10.40.0.136",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&hostA).Error; err != nil {
		t.Fatalf("seed host A: %v", err)
	}
	if err := db.Create(&hostB).Error; err != nil {
		t.Fatalf("seed host B: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "task-summary-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "任务汇总任务",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{hostA.ID, hostB.ID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	now := time.Now()
	hostRows := []DeployTaskHost{
		{
			TaskID:     task.ID,
			HostID:     hostA.ID,
			Hostname:   hostA.Hostname,
			HostIP:     hostA.IP,
			OS:         hostA.OS,
			Status:     TaskHostStatusSuccess,
			StartedAt:  &now,
			FinishedAt: &now,
			UpdatedBy:  "1",
		},
		{
			TaskID:    task.ID,
			HostID:    hostB.ID,
			Hostname:  hostB.Hostname,
			HostIP:    hostB.IP,
			OS:        hostB.OS,
			Status:    TaskHostStatusFailed,
			StartedAt: &now,
			UpdatedBy: "1",
		},
	}
	if err := db.Create(&hostRows).Error; err != nil {
		t.Fatalf("seed task hosts: %v", err)
	}
	if err := db.Model(&DeployTask{}).Where("id = ?", task.ID).Updates(map[string]any{
		"status":     TaskStatusRunning,
		"started_at": &now,
		"updated_by": "1",
		"updated_at": now,
	}).Error; err != nil {
		t.Fatalf("update task status: %v", err)
	}

	list, err := svc.ListTasks(TaskQuery{Page: 1, PageSize: 10}, nil)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(list.Items) == 0 {
		t.Fatal("expected task list to contain items")
	}

	var summary *TaskResponse
	for i := range list.Items {
		if list.Items[i].ID == task.ID {
			summary = &list.Items[i]
			break
		}
	}
	if summary == nil {
		t.Fatalf("expected task %d in list", task.ID)
	}
	if summary.HostCount != 2 {
		t.Fatalf("expected hostCount=2, got %d", summary.HostCount)
	}
	if summary.SuccessCount != 1 {
		t.Fatalf("expected successCount=1, got %d", summary.SuccessCount)
	}
	if summary.FailedCount != 1 {
		t.Fatalf("expected failedCount=1, got %d", summary.FailedCount)
	}
	if summary.RunningCount != 0 {
		t.Fatalf("expected runningCount=0, got %d", summary.RunningCount)
	}
	if len(summary.Hosts) != 2 {
		t.Fatalf("expected 2 host snapshots in list summary, got %d", len(summary.Hosts))
	}
}

func TestDeleteDeployTaskRemovesPendingTaskAndHosts(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scope := deployTestBizScope{Code: "delete-task-scope", Name: "删除任务作用域", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "delete-task-host",
		IP:                "10.40.0.131",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "delete-task-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "待删除部署任务",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	now := time.Now()
	hostRow := DeployTaskHost{
		TaskID:    task.ID,
		HostID:    host.ID,
		Hostname:  host.Hostname,
		HostIP:    host.IP,
		OS:        host.OS,
		Status:    TaskHostStatusPending,
		StartedAt: &now,
		UpdatedBy: "1",
	}
	if err := db.Create(&hostRow).Error; err != nil {
		t.Fatalf("create task host: %v", err)
	}

	if err := svc.DeleteTask(task.ID, "1", nil); err != nil {
		t.Fatalf("delete task: %v", err)
	}

	if _, err := svc.GetTask(task.ID, nil); err == nil {
		t.Fatal("expected deleted task to be hidden")
	} else if !isDeployTaskNotFound(err) {
		t.Fatalf("expected deleted task not found error, got %v", err)
	}

	var hostCount int64
	if err := db.Model(&DeployTaskHost{}).Where("task_id = ?", task.ID).Count(&hostCount).Error; err != nil {
		t.Fatalf("count task hosts: %v", err)
	}
	if hostCount != 0 {
		t.Fatalf("expected task hosts deleted, got %d", hostCount)
	}
}

func TestDeleteDeployTaskRejectsRunningTask(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scope := deployTestBizScope{Code: "delete-locked-scope", Name: "删除锁定作用域", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "delete-locked-host",
		IP:                "10.40.0.132",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "delete-locked-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "锁定部署任务",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if _, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil); err != nil {
		t.Fatalf("start task: %v", err)
	}

	err = svc.DeleteTask(task.ID, "1", nil)
	if err == nil || err.Error() != errDeployTaskInvalidDeleteState {
		t.Fatalf("expected %s, got %v", errDeployTaskInvalidDeleteState, err)
	}
}

func TestStartDeployTaskAllowsDraftTask(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scope := deployTestBizScope{Code: "draft-start-scope", Name: "草稿启动作用域", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "draft-start-host",
		IP:                "10.40.0.133",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "draft-start-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "草稿启动任务",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatalf("start draft task: %v", err)
	}
	if started.Status != TaskStatusRunning {
		t.Fatalf("expected running task, got %s", started.Status)
	}
	if len(started.Hosts) != 1 {
		t.Fatalf("expected one task host, got %d", len(started.Hosts))
	}
	if started.Hosts[0].HostID != host.ID || started.Hosts[0].Status != TaskHostStatusRunning {
		t.Fatalf("unexpected task host snapshot: %+v", started.Hosts[0])
	}
}

func TestCancelDeployTaskRejectsDraftTask(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scope := deployTestBizScope{Code: "draft-cancel-scope", Name: "草稿取消作用域", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "draft-cancel-host",
		IP:                "10.40.0.133",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "draft-cancel-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "草稿取消任务",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	_, err = svc.CancelTask(task.ID, "1", nil)
	if err == nil || err.Error() != "business.deploy.task.invalidCancelState" {
		t.Fatalf("expected business.deploy.task.invalidCancelState, got %v", err)
	}
}

func TestMarkHostResultRejectsFailedWithoutErrorMessage(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scope := deployTestBizScope{Code: "failed-reason-scope", Name: "失败原因作用域", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "failed-reason-host",
		IP:                "10.40.0.134",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "failed-reason-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "失败原因任务",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatalf("start task: %v", err)
	}
	if len(started.Hosts) != 1 {
		t.Fatalf("expected one host, got %+v", started.Hosts)
	}

	_, err = svc.MarkHostResult(started.Hosts[0].ID, MarkHostResultRequest{
		Status: TaskHostStatusFailed,
		Stdout: "failed without reason",
	}, "1", nil)
	if err == nil || err.Error() != "business.deploy.taskHost.markFailed.reasonRequired" {
		t.Fatalf("expected business.deploy.taskHost.markFailed.reasonRequired, got %v", err)
	}
}

func TestDeployTaskDetailHandlerReturnsForbiddenAndNotFound(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	handler := NewDeployHandler(svc)

	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "handler-nginx",
		Version:        "1.0.0",
		InstallCommand: "install nginx",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	scope := deployTestBizScope{Code: "handler-scope", Name: "Handler Scope", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "handler-host",
		IP:                "10.40.0.91",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
		DeptID:            21,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "Handler Detail Task",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 21})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	forbiddenRouter := gin.New()
	forbiddenRouter.GET("/tasks/:id", func(c *gin.Context) {
		c.Set(common.DataScopeContextKey, &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 22})
		handler.GetTask(c)
	})
	forbiddenRecorder := httptest.NewRecorder()
	forbiddenRequest := httptest.NewRequest(http.MethodGet, "/tasks/"+strconv.FormatUint(task.ID, 10), nil)
	forbiddenRouter.ServeHTTP(forbiddenRecorder, forbiddenRequest)
	if forbiddenRecorder.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden status, got %d", forbiddenRecorder.Code)
	}
	var forbiddenResp common.Response
	if err := json.Unmarshal(forbiddenRecorder.Body.Bytes(), &forbiddenResp); err != nil {
		t.Fatalf("decode forbidden response: %v", err)
	}
	if forbiddenResp.Code != common.CodeForbidden || forbiddenResp.Message != errDeployTaskForbidden.Error() {
		t.Fatalf("unexpected forbidden response: %+v", forbiddenResp)
	}

	notFoundRouter := gin.New()
	notFoundRouter.GET("/tasks/:id", handler.GetTask)
	notFoundRecorder := httptest.NewRecorder()
	notFoundRequest := httptest.NewRequest(http.MethodGet, "/tasks/999999", nil)
	notFoundRouter.ServeHTTP(notFoundRecorder, notFoundRequest)
	if notFoundRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected not found status, got %d", notFoundRecorder.Code)
	}
	var notFoundResp common.Response
	if err := json.Unmarshal(notFoundRecorder.Body.Bytes(), &notFoundResp); err != nil {
		t.Fatalf("decode not found response: %v", err)
	}
	if notFoundResp.Code != common.CodeNotFound || notFoundResp.Message != errDeployTaskNotFound.Error() {
		t.Fatalf("unexpected not found response: %+v", notFoundResp)
	}
}

func TestDeployTaskActionHandlersReturnCanonicalStatusCodes(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	handler := NewDeployHandler(svc)

	scope := deployTestBizScope{Code: "handler-action-scope", Name: "Handler Action Scope", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "handler-action-host",
		IP:                "10.40.0.92",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "handler-action-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "Handler Action Task",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	draftTask, err := svc.CreateTask(CreateTaskRequest{
		Name:            "Handler Draft Cancel Task",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create draft task: %v", err)
	}
	started, err := svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
	if err != nil {
		t.Fatalf("start task: %v", err)
	}

	startRouter := gin.New()
	startRouter.POST("/tasks/:id/start", handler.StartTask)
	startRecorder := httptest.NewRecorder()
	startRequest := httptest.NewRequest(http.MethodPost, "/tasks/"+strconv.FormatUint(started.ID, 10)+"/start", nil)
	startRouter.ServeHTTP(startRecorder, startRequest)
	if startRecorder.Code != http.StatusConflict {
		t.Fatalf("expected start conflict status, got %d", startRecorder.Code)
	}
	var startResp common.Response
	if err := json.Unmarshal(startRecorder.Body.Bytes(), &startResp); err != nil {
		t.Fatalf("decode start response: %v", err)
	}
	if startResp.Code != http.StatusConflict || startResp.Message != "business.deploy.task.invalidStartState" {
		t.Fatalf("unexpected start response: %+v", startResp)
	}

	cancelRouter := gin.New()
	cancelRouter.POST("/tasks/:id/cancel", handler.CancelTask)
	cancelRecorder := httptest.NewRecorder()
	cancelRequest := httptest.NewRequest(http.MethodPost, "/tasks/"+strconv.FormatUint(draftTask.ID, 10)+"/cancel", nil)
	cancelRouter.ServeHTTP(cancelRecorder, cancelRequest)
	if cancelRecorder.Code != http.StatusConflict {
		t.Fatalf("expected cancel conflict status, got %d", cancelRecorder.Code)
	}
	var cancelResp common.Response
	if err := json.Unmarshal(cancelRecorder.Body.Bytes(), &cancelResp); err != nil {
		t.Fatalf("decode cancel response: %v", err)
	}
	if cancelResp.Code != http.StatusConflict || cancelResp.Message != "business.deploy.task.invalidCancelState" {
		t.Fatalf("unexpected cancel response: %+v", cancelResp)
	}

	updateRouter := gin.New()
	updateRouter.PUT("/tasks/:id", handler.UpdateTask)
	updateRecorder := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(
		http.MethodPut,
		"/tasks/"+strconv.FormatUint(started.ID, 10),
		strings.NewReader(`{"name":"locked update"}`),
	)
	updateRequest.Header.Set("Content-Type", "application/json")
	updateRouter.ServeHTTP(updateRecorder, updateRequest)
	if updateRecorder.Code != http.StatusConflict {
		t.Fatalf("expected update conflict status, got %d", updateRecorder.Code)
	}
	var updateResp common.Response
	if err := json.Unmarshal(updateRecorder.Body.Bytes(), &updateResp); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updateResp.Code != http.StatusConflict || updateResp.Message != errDeployTaskInvalidUpdateState {
		t.Fatalf("unexpected update response: %+v", updateResp)
	}

	deleteRouter := gin.New()
	deleteRouter.DELETE("/tasks/:id", handler.DeleteTask)
	deleteRecorder := httptest.NewRecorder()
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/tasks/"+strconv.FormatUint(started.ID, 10), nil)
	deleteRouter.ServeHTTP(deleteRecorder, deleteRequest)
	if deleteRecorder.Code != http.StatusConflict {
		t.Fatalf("expected delete conflict status, got %d", deleteRecorder.Code)
	}
	var deleteResp common.Response
	if err := json.Unmarshal(deleteRecorder.Body.Bytes(), &deleteResp); err != nil {
		t.Fatalf("decode delete response: %v", err)
	}
	if deleteResp.Code != http.StatusConflict || deleteResp.Message != errDeployTaskInvalidDeleteState {
		t.Fatalf("unexpected delete response: %+v", deleteResp)
	}

	markRouter := gin.New()
	markRouter.POST("/task-hosts/:id/result", handler.MarkHostResult)
	markRecorder := httptest.NewRecorder()
	markRequest := httptest.NewRequest(
		http.MethodPost,
		"/task-hosts/"+strconv.FormatUint(started.Hosts[0].ID, 10)+"/result",
		strings.NewReader(`{"status":"failed","stdout":"no reason"}`),
	)
	markRequest.Header.Set("Content-Type", "application/json")
	markRouter.ServeHTTP(markRecorder, markRequest)
	var markResp common.Response
	if err := json.Unmarshal(markRecorder.Body.Bytes(), &markResp); err != nil {
		t.Fatalf("decode mark response: %v", err)
	}
	if markResp.Code != common.CodeParamInvalid || markResp.Message != "business.deploy.taskHost.markFailed.reasonRequired" {
		t.Fatalf("unexpected mark response: %+v", markResp)
	}
}

func TestDeployTaskCreateAndUpdateReturnTargetScopeCanonicalErrors(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scopeA := deployTestBizScope{Code: "scope-a", Name: "Scope A", Status: "active"}
	scopeB := deployTestBizScope{Code: "scope-b", Name: "Scope B", Status: "active"}
	if err := db.Create(&scopeA).Error; err != nil {
		t.Fatalf("seed scope A: %v", err)
	}
	if err := db.Create(&scopeB).Error; err != nil {
		t.Fatalf("seed scope B: %v", err)
	}
	hostA := deployTestHost{
		Hostname:          "scope-host-a",
		IP:                "10.40.0.94",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scopeA.ID,
		BusinessScopeName: scopeA.Name,
	}
	hostB := deployTestHost{
		Hostname:          "scope-host-b",
		IP:                "10.40.0.95",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scopeB.ID,
		BusinessScopeName: scopeB.Name,
	}
	if err := db.Create(&hostA).Error; err != nil {
		t.Fatalf("seed host A: %v", err)
	}
	if err := db.Create(&hostB).Error; err != nil {
		t.Fatalf("seed host B: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "scope-target-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	_, err = svc.CreateTask(CreateTaskRequest{
		Name:            "out-of-scope host task",
		PackageID:       pkg.ID,
		BusinessScopeID: scopeA.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{hostB.ID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err == nil || err.Error() != "business.deploy.task.targetOutOfScope" {
		t.Fatalf("expected business.deploy.task.targetOutOfScope, got %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "update target scope task",
		PackageID:       pkg.ID,
		BusinessScopeID: scopeA.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{hostA.ID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	_, err = svc.UpdateTask(task.ID, UpdateTaskRequest{
		BusinessScopeID: uint64Ptr(scopeA.ID),
		TargetIDs:       []uint64{hostB.ID},
	}, "1", nil)
	if err == nil || err.Error() != "business.deploy.task.targetOutOfScope" {
		t.Fatalf("expected update out-of-scope error, got %v", err)
	}
}

func TestDeployTaskCreateAndUpdateReturnTargetStatusCanonicalError(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))

	scope := deployTestBizScope{Code: "status-mismatch-scope", Name: "Status Mismatch Scope", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "status-mismatch-host",
		IP:                "10.40.0.95",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "status-mismatch-package",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	_, err = svc.CreateTask(CreateTaskRequest{
		Name:            "invalid uninstall task",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		Action:          TaskActionUninstall,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err == nil || err.Error() != "business.deploy.task.targetStatusMismatch" {
		t.Fatalf("expected create target status mismatch error, got %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "valid install task",
		PackageID:       pkg.ID,
		BusinessScopeID: scope.ID,
		Action:          TaskActionInstall,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create valid task: %v", err)
	}
	uninstall := TaskActionUninstall
	_, err = svc.UpdateTask(task.ID, UpdateTaskRequest{Action: &uninstall}, "1", nil)
	if err == nil || err.Error() != "business.deploy.task.targetStatusMismatch" {
		t.Fatalf("expected update target status mismatch error, got %v", err)
	}
}

func TestDeployTaskMutationHandlersReturnCanonicalValidationErrors(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	handler := NewDeployHandler(svc)

	scope := deployTestBizScope{Code: "handler-validate-scope", Name: "Handler Validate Scope", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "handler-validate-host",
		IP:                "10.40.0.93",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	enabledPkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "handler-validate-enabled",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create enabled package: %v", err)
	}
	disabledPkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "handler-validate-disabled",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusDisabled,
	}, "1")
	if err != nil {
		t.Fatalf("create disabled package: %v", err)
	}
	task, err := svc.CreateTask(CreateTaskRequest{
		Name:            "Handler Validate Task",
		PackageID:       enabledPkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeManual,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	createRouter := gin.New()
	createRouter.POST("/tasks", handler.CreateTask)
	createRecorder := httptest.NewRecorder()
	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/tasks",
		strings.NewReader(`{"name":"disabled package task","packageId":`+strconv.FormatUint(disabledPkg.ID, 10)+`,"businessScopeId":`+strconv.FormatUint(scope.ID, 10)+`,"targetType":"host","targetIds":[`+strconv.FormatUint(host.ID, 10)+`],"executorType":"manual"}`),
	)
	createRequest.Header.Set("Content-Type", "application/json")
	createRouter.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected create bad request, got %d", createRecorder.Code)
	}
	var createResp common.Response
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createResp.Code != common.CodeParamInvalid || createResp.Message != "business.deploy.task.packageDisabled" {
		t.Fatalf("unexpected create response: %+v", createResp)
	}

	createInvalidActionRecorder := httptest.NewRecorder()
	createInvalidActionRequest := httptest.NewRequest(
		http.MethodPost,
		"/tasks",
		strings.NewReader(`{"name":"invalid action task","packageId":`+strconv.FormatUint(enabledPkg.ID, 10)+`,"businessScopeId":`+strconv.FormatUint(scope.ID, 10)+`,"action":"bogus","targetType":"host","targetIds":[`+strconv.FormatUint(host.ID, 10)+`],"executorType":"manual"}`),
	)
	createInvalidActionRequest.Header.Set("Content-Type", "application/json")
	createRouter.ServeHTTP(createInvalidActionRecorder, createInvalidActionRequest)
	if createInvalidActionRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid action bad request, got %d", createInvalidActionRecorder.Code)
	}
	var createInvalidActionResp common.Response
	if err := json.Unmarshal(createInvalidActionRecorder.Body.Bytes(), &createInvalidActionResp); err != nil {
		t.Fatalf("decode invalid action response: %v", err)
	}
	if createInvalidActionResp.Code != common.CodeParamInvalid || createInvalidActionResp.Message != "business.deploy.task.invalidAction" {
		t.Fatalf("unexpected invalid action response: %+v", createInvalidActionResp)
	}

	updateRouter := gin.New()
	updateRouter.PUT("/tasks/:id", handler.UpdateTask)
	updateRecorder := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(
		http.MethodPut,
		"/tasks/"+strconv.FormatUint(task.ID, 10),
		strings.NewReader(`{"executorType":"bogus"}`),
	)
	updateRequest.Header.Set("Content-Type", "application/json")
	updateRouter.ServeHTTP(updateRecorder, updateRequest)
	if updateRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected update bad request, got %d", updateRecorder.Code)
	}
	var updateResp common.Response
	if err := json.Unmarshal(updateRecorder.Body.Bytes(), &updateResp); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if updateResp.Code != common.CodeParamInvalid || updateResp.Message != "business.deploy.task.invalidExecutorType" {
		t.Fatalf("unexpected update response: %+v", updateResp)
	}

	updateInvalidTargetRecorder := httptest.NewRecorder()
	updateInvalidTargetRequest := httptest.NewRequest(
		http.MethodPut,
		"/tasks/"+strconv.FormatUint(task.ID, 10),
		strings.NewReader(`{"targetType":"bogus"}`),
	)
	updateInvalidTargetRequest.Header.Set("Content-Type", "application/json")
	updateRouter.ServeHTTP(updateInvalidTargetRecorder, updateInvalidTargetRequest)
	if updateInvalidTargetRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid target type bad request, got %d", updateInvalidTargetRecorder.Code)
	}
	var updateInvalidTargetResp common.Response
	if err := json.Unmarshal(updateInvalidTargetRecorder.Body.Bytes(), &updateInvalidTargetResp); err != nil {
		t.Fatalf("decode invalid target type response: %v", err)
	}
	if updateInvalidTargetResp.Code != common.CodeParamInvalid || updateInvalidTargetResp.Message != "business.deploy.task.invalidTargetType" {
		t.Fatalf("unexpected invalid target type response: %+v", updateInvalidTargetResp)
	}
}

func TestDeployTaskMutationHandlersReturnCanonicalScopeValidationErrors(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	handler := NewDeployHandler(svc)

	scope := deployTestBizScope{Code: "handler-scope-validation", Name: "Handler Scope Validation", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "handler-scope-host",
		IP:                "10.40.0.96",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "handler-scope-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	createRouter := gin.New()
	createRouter.POST("/tasks", handler.CreateTask)
	createRecorder := httptest.NewRecorder()
	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/tasks",
		strings.NewReader(`{"name":"missing scope task","packageId":`+strconv.FormatUint(pkg.ID, 10)+`,"targetType":"host","targetIds":[`+strconv.FormatUint(host.ID, 10)+`],"executorType":"manual"}`),
	)
	createRequest.Header.Set("Content-Type", "application/json")
	createRouter.ServeHTTP(createRecorder, createRequest)
	if createRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected create scope bad request, got %d", createRecorder.Code)
	}
	var createResp common.Response
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create scope response: %v", err)
	}
	if createResp.Code != common.CodeParamInvalid || createResp.Message != "business.deploy.task.scopeRequired" {
		t.Fatalf("unexpected create scope response: %+v", createResp)
	}
}

func TestDeployTaskMutationHandlersReturnCanonicalTargetStatusValidationError(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	handler := NewDeployHandler(svc)

	scope := deployTestBizScope{Code: "handler-status-validation", Name: "Handler Status Validation", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "handler-status-host",
		IP:                "10.40.0.97",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "handler-status-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	router := gin.New()
	router.POST("/tasks", handler.CreateTask)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/tasks",
		strings.NewReader(`{"name":"invalid status task","packageId":`+strconv.FormatUint(pkg.ID, 10)+`,"businessScopeId":`+strconv.FormatUint(scope.ID, 10)+`,"action":"uninstall","targetType":"host","targetIds":[`+strconv.FormatUint(host.ID, 10)+`],"executorType":"manual"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected target status bad request, got %d", recorder.Code)
	}
	var resp common.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode target status response: %v", err)
	}
	if resp.Code != common.CodeParamInvalid || resp.Message != "business.deploy.task.targetStatusMismatch" {
		t.Fatalf("unexpected target status response: %+v", resp)
	}
}

func TestStartDeployTaskHandlerReturnsCanonicalValidationErrors(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	handler := NewDeployHandler(svc)

	scope := deployTestBizScope{Code: "handler-start-validate", Name: "Handler Start Validate", Status: "active"}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname:          "handler-start-host",
		IP:                "10.40.0.114",
		OS:                "linux",
		Status:            "assigned",
		BusinessScopeID:   scope.ID,
		BusinessScopeName: scope.Name,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}

	startRouter := gin.New()
	startRouter.POST("/tasks/:id/start", handler.StartTask)

	plainPkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "handler-start-plain-pkg",
		Version:        "1.0.0",
		InstallCommand: "echo install",
		Status:         PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create plain package: %v", err)
	}
	plainTask, err := svc.CreateTask(CreateTaskRequest{
		Name:            "Handler SSH Start Task",
		PackageID:       plainPkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeSSH,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create plain task: %v", err)
	}

	missingFingerprintRecorder := httptest.NewRecorder()
	missingFingerprintRequest := httptest.NewRequest(
		http.MethodPost,
		"/tasks/"+strconv.FormatUint(plainTask.ID, 10)+"/start",
		strings.NewReader(`{"sshUser":"root","sshPassword":"secret","authMode":"password"}`),
	)
	missingFingerprintRequest.Header.Set("Content-Type", "application/json")
	startRouter.ServeHTTP(missingFingerprintRecorder, missingFingerprintRequest)
	if missingFingerprintRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected missing fingerprint bad request, got %d", missingFingerprintRecorder.Code)
	}
	var missingFingerprintResp common.Response
	if err := json.Unmarshal(missingFingerprintRecorder.Body.Bytes(), &missingFingerprintResp); err != nil {
		t.Fatalf("decode missing fingerprint response: %v", err)
	}
	if missingFingerprintResp.Code != common.CodeParamInvalid || missingFingerprintResp.Message != errDeployTaskSSHHostKeyRequired {
		t.Fatalf("unexpected missing fingerprint response: %+v", missingFingerprintResp)
	}

	fixedPkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:          "handler-start-fixed-pkg",
		Version:       "1.0.0",
		ExecutionMode: ExecutionModeFixed,
		TemplateCode:  TemplateCodeNginxSystemd,
		Status:        PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create fixed package: %v", err)
	}
	fixedTask, err := svc.CreateTask(CreateTaskRequest{
		Name:            "Handler Fixed Start Task",
		PackageID:       fixedPkg.ID,
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
		t.Fatalf("create fixed task: %v", err)
	}
	if err := db.Model(&DeployTask{}).Where("id = ?", fixedTask.ID).Update("template_params", datatypes.JSON([]byte(`{"installRoot":"/data/nginx"}`))).Error; err != nil {
		t.Fatalf("corrupt template params: %v", err)
	}

	templateRecorder := httptest.NewRecorder()
	templateRequest := httptest.NewRequest(
		http.MethodPost,
		"/tasks/"+strconv.FormatUint(fixedTask.ID, 10)+"/start",
		strings.NewReader(`{"sshUser":"root","sshPassword":"secret","hostFingerprint":"SHA256:test","authMode":"password"}`),
	)
	templateRequest.Header.Set("Content-Type", "application/json")
	startRouter.ServeHTTP(templateRecorder, templateRequest)
	if templateRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected template validation bad request, got %d", templateRecorder.Code)
	}
	var templateResp common.Response
	if err := json.Unmarshal(templateRecorder.Body.Bytes(), &templateResp); err != nil {
		t.Fatalf("decode template validation response: %v", err)
	}
	if templateResp.Code != common.CodeParamInvalid || templateResp.Message != errDeployTaskTemplateParamsInvalid {
		t.Fatalf("unexpected template validation response: %+v", templateResp)
	}

	commandTask, err := svc.CreateTask(CreateTaskRequest{
		Name:            "Handler Command Start Task",
		PackageID:       plainPkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeSSH,
	}, "1", nil)
	if err != nil {
		t.Fatalf("create command task: %v", err)
	}
	if err := db.Model(&DeployPackage{}).Where("id = ?", plainPkg.ID).Update("install_command", "").Error; err != nil {
		t.Fatalf("clear install command: %v", err)
	}

	commandRecorder := httptest.NewRecorder()
	commandRequest := httptest.NewRequest(
		http.MethodPost,
		"/tasks/"+strconv.FormatUint(commandTask.ID, 10)+"/start",
		strings.NewReader(`{"sshUser":"root","sshPassword":"secret","hostFingerprint":"SHA256:test","authMode":"password"}`),
	)
	commandRequest.Header.Set("Content-Type", "application/json")
	startRouter.ServeHTTP(commandRecorder, commandRequest)
	if commandRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected install command validation bad request, got %d", commandRecorder.Code)
	}
	var commandResp common.Response
	if err := json.Unmarshal(commandRecorder.Body.Bytes(), &commandResp); err != nil {
		t.Fatalf("decode install command response: %v", err)
	}
	if commandResp.Code != common.CodeParamInvalid || commandResp.Message != errDeployTaskInstallCommandRequired {
		t.Fatalf("unexpected install command response: %+v", commandResp)
	}

	missingSourcePkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:            "handler-start-source-pkg",
		Version:         "1.0.0",
		ExecutionMode:   ExecutionModeFixed,
		TemplateCode:    TemplateCodeHarborOffline,
		SourceObjectKey: "deploy/package/harbor-offline.tgz",
		SourceFileName:  "harbor-offline.tgz",
		Status:          PackageStatusEnabled,
	}, "1")
	if err != nil {
		t.Fatalf("create missing source package: %v", err)
	}
	missingSourceTask, err := svc.CreateTask(CreateTaskRequest{
		Name:            "Handler Missing Source Task",
		PackageID:       missingSourcePkg.ID,
		BusinessScopeID: scope.ID,
		TargetType:      TargetTypeHost,
		TargetIDs:       []uint64{host.ID},
		ExecutorType:    ExecutorTypeSSH,
		TemplateParams: map[string]any{
			"installRoot":   "/data/harbor",
			"dataRoot":      "/data/harbor/data",
			"hostname":      "harbor.internal.local",
			"httpPort":      "8088",
			"adminPassword": "Harbor_123",
		},
	}, "1", nil)
	if err != nil {
		t.Fatalf("create missing source task: %v", err)
	}

	missingSourceRecorder := httptest.NewRecorder()
	missingSourceRequest := httptest.NewRequest(
		http.MethodPost,
		"/tasks/"+strconv.FormatUint(missingSourceTask.ID, 10)+"/start",
		strings.NewReader(`{"sshUser":"root","sshPassword":"secret","hostFingerprint":"SHA256:test","authMode":"password"}`),
	)
	missingSourceRequest.Header.Set("Content-Type", "application/json")
	startRouter.ServeHTTP(missingSourceRecorder, missingSourceRequest)
	if missingSourceRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected missing source bad request, got %d", missingSourceRecorder.Code)
	}
	var missingSourceResp common.Response
	if err := json.Unmarshal(missingSourceRecorder.Body.Bytes(), &missingSourceResp); err != nil {
		t.Fatalf("decode missing source response: %v", err)
	}
	if missingSourceResp.Code != common.CodeParamInvalid || missingSourceResp.Message != errDeployTaskPackageSourceMissing {
		t.Fatalf("unexpected missing source response: %+v", missingSourceResp)
	}
}
