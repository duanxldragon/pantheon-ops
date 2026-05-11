package deploy

import (
	"testing"

	"pantheon-ops/backend/pkg/testmysql"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func setupDeployTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)
	svc := NewDeployService(db)
	if err := svc.Migrate(); err != nil {
		t.Fatalf("migrate deploy tables: %v", err)
	}
	if err := db.AutoMigrate(&deployTestHost{}, &deployTestGroup{}); err != nil {
		t.Fatalf("migrate cmdb fixtures: %v", err)
	}
	return db
}

type deployTestHost struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement"`
	Hostname    string         `gorm:"size:128;not null"`
	IP          string         `gorm:"size:45;not null"`
	OS          string         `gorm:"size:32;not null"`
	Status      string         `gorm:"size:32;default:pending"`
	LabelValues datatypes.JSON `gorm:"type:json"`
	DeptID      uint64         `gorm:"column:dept_id"`
}

func (deployTestHost) TableName() string { return "biz_cmdb_host" }

type deployTestGroup struct {
	ID         uint64         `gorm:"primaryKey;autoIncrement"`
	ParentID   uint64         `gorm:"column:parent_id"`
	Name       string         `gorm:"size:128;not null"`
	Conditions datatypes.JSON `gorm:"type:json"`
}

func (deployTestGroup) TableName() string { return "biz_cmdb_group" }

func TestDeployTaskLifecycleCreatesHostDetailsAndSummarizesResult(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db)

	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name:           "nginx",
		Version:        "1.26",
		InstallCommand: "apt-get install -y nginx",
		Status:         "enabled",
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}

	host := deployTestHost{Hostname: "deploy-host-1", IP: "10.40.0.11", OS: "linux", Status: "online"}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}

	task, err := svc.CreateTask(CreateTaskRequest{
		Name:         "安装 Nginx",
		PackageID:    pkg.ID,
		TargetType:   "host",
		TargetIDs:    []uint64{host.ID},
		ExecutorType: "manual",
	}, "1")
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.Status != TaskStatusPending {
		t.Fatalf("expected pending task, got %s", task.Status)
	}

	started, err := svc.StartTask(task.ID, "1")
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
	svc := NewDeployService(db)

	pkg, err := svc.CreatePackage(CreatePackageRequest{Name: "node-exporter", Version: "1.8", InstallCommand: "install node-exporter"}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	matched := deployTestHost{
		Hostname:    "monitor-1",
		IP:          "10.40.0.21",
		OS:          "linux",
		Status:      "online",
		LabelValues: datatypes.JSON([]byte(`[{"key":"role","val":"monitor"}]`)),
	}
	other := deployTestHost{
		Hostname:    "app-1",
		IP:          "10.40.0.22",
		OS:          "linux",
		Status:      "online",
		LabelValues: datatypes.JSON([]byte(`[{"key":"role","val":"app"}]`)),
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
	}, "1")
	if err != nil {
		t.Fatalf("create group task: %v", err)
	}

	started, err := svc.StartTask(task.ID, "1")
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

func TestDeployTaskGroupTargetInheritsParentConditions(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db)

	pkg, err := svc.CreatePackage(CreatePackageRequest{Name: "node-exporter", Version: "1.9", InstallCommand: "install node-exporter"}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	xianMonitor := deployTestHost{
		Hostname:    "xian-monitor-1",
		IP:          "10.40.1.21",
		OS:          "linux",
		Status:      "online",
		LabelValues: datatypes.JSON([]byte(`[{"key":"region","val":"西安开发环境"},{"key":"role","val":"monitor"}]`)),
	}
	xianApp := deployTestHost{
		Hostname:    "xian-app-1",
		IP:          "10.40.1.22",
		OS:          "linux",
		Status:      "online",
		LabelValues: datatypes.JSON([]byte(`[{"key":"region","val":"西安开发环境"},{"key":"role","val":"app"}]`)),
	}
	shanghaiMonitor := deployTestHost{
		Hostname:    "shanghai-monitor-1",
		IP:          "10.40.1.23",
		OS:          "linux",
		Status:      "online",
		LabelValues: datatypes.JSON([]byte(`[{"key":"region","val":"上海开发环境"},{"key":"role","val":"monitor"}]`)),
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
	}, "1")
	if err != nil {
		t.Fatalf("create child group task: %v", err)
	}

	started, err := svc.StartTask(task.ID, "1")
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
