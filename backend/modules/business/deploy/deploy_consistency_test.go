package deploy

import (
	"flag"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
)

var requireFixedTemplateAssets = flag.Bool("require-deploy-assets", false, "verify the packaged deploy asset directory")

func TestFixedTemplateAssetCatalog(t *testing.T) {
	catalog := FixedTemplateAssets()
	covered := make(map[string]bool, len(catalog))
	for _, item := range catalog {
		if covered[item.TemplateCode] {
			t.Fatalf("duplicate template %s in the asset catalog", item.TemplateCode)
		}
		covered[item.TemplateCode] = true
		if item.NeedsOwnPackage == (len(item.AssetPrefixes) != 0) {
			t.Fatalf("template %s must either declare preset asset prefixes or require an operator package", item.TemplateCode)
		}
	}
	for _, code := range fixedTemplateCodes() {
		if !covered[code] {
			t.Fatalf("fixed template %s is missing from FixedTemplateAssets", code)
		}
	}
	if len(covered) != len(fixedTemplateCodes()) {
		t.Fatal("asset catalog contains an unknown fixed template")
	}
}

func TestFixedTemplateAssetCoverage(t *testing.T) {
	if !*requireFixedTemplateAssets {
		t.Skip("archive verification runs through npm run check:deploy-assets")
	}
	dir, err := FixedTemplateAssetDir()
	if err != nil {
		t.Fatalf("resolve asset dir: %v", err)
	}
	names, err := listFixedAssetNames(dir)
	if err != nil {
		t.Fatalf("read asset dir %s: %v", dir, err)
	}
	if missing := UnbackedFixedTemplates(FixedTemplateAssets(), names); len(missing) != 0 {
		t.Fatalf("fixed templates declare preset packages missing from %s: %v (ship the archive or mark NeedsOwnPackage)", dir, missing)
	}
}

func TestFixedTemplateAssetCoverageReportsMissingPackage(t *testing.T) {
	catalog := []FixedTemplateAsset{
		{TemplateCode: TemplateCodeNginxSystemd, AssetPrefixes: []string{"nginx-"}},
		{TemplateCode: TemplateCodeMySQLSystemd, AssetPrefixes: []string{"mysql-"}},
		{TemplateCode: TemplateCodeMinIOSystemd, NeedsOwnPackage: true},
	}
	missing := UnbackedFixedTemplates(catalog, []string{"nginx-1.31.9.tar.gz"})
	if len(missing) != 1 || missing[0] != TemplateCodeMySQLSystemd {
		t.Fatalf("expected only %s to be reported, got %v", TemplateCodeMySQLSystemd, missing)
	}
	missing = UnbackedFixedTemplates(catalog, []string{"nginx-1.31.9.tar.gz", "mysql-8.0.36.tar.gz"})
	if len(missing) != 0 {
		t.Fatalf("expected no missing templates once archives exist, got %v", missing)
	}
}

func deployExecutorFixture(t *testing.T) (*DeployService, uint64) {
	t.Helper()
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	pkg, err := svc.CreatePackage(CreatePackageRequest{
		Name: "exec-nginx", Version: "1.26", InstallCommand: "apt-get install -y nginx", Status: "enabled",
	}, "1")
	if err != nil {
		t.Fatalf("create package: %v", err)
	}
	scope := deployTestBizScope{Code: "exec-scope", Name: "Exec scope", Status: "active", DeptID: 10}
	if err := db.Create(&scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	host := deployTestHost{
		Hostname: "exec-host", IP: "10.42.0.11", OS: "linux", Status: "assigned",
		BusinessScopeID: scope.ID, BusinessScopeName: scope.Name, DeptID: 10,
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("seed host: %v", err)
	}
	task, err := svc.CreateTask(CreateTaskRequest{
		Name: "exec-task", PackageID: pkg.ID, BusinessScopeID: scope.ID,
		TargetType: "host", TargetIDs: []uint64{host.ID}, ExecutorType: "manual",
	}, "1", nil)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	return svc, task.ID
}

func TestCreateTaskRejectsAgentExecutor(t *testing.T) {
	svc, taskID := deployExecutorFixture(t)
	var task DeployTask
	if err := svc.db.First(&task, taskID).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	_, err := svc.CreateTask(CreateTaskRequest{
		Name: "agent-task", PackageID: task.PackageID, BusinessScopeID: task.BusinessScopeID,
		TargetType: "host", TargetIDs: parseUint64JSON(task.TargetIDs), ExecutorType: ExecutorTypeAgent,
	}, "1", nil)
	if err == nil || err.Error() != errDeployExecutorTypeUnsupported {
		t.Fatalf("expected %s, got %v", errDeployExecutorTypeUnsupported, err)
	}
}

func TestUpdateTaskRejectsAgentExecutor(t *testing.T) {
	svc, taskID := deployExecutorFixture(t)
	agent := ExecutorTypeAgent
	if _, err := svc.updateTask(taskID, UpdateTaskRequest{ExecutorType: &agent}, "1", nil); err == nil || err.Error() != errDeployExecutorTypeUnsupported {
		t.Fatalf("expected %s, got %v", errDeployExecutorTypeUnsupported, err)
	}
}

func TestLegacyAgentTaskStaysReadableButCannotStart(t *testing.T) {
	svc, taskID := deployExecutorFixture(t)
	if err := svc.db.Model(&DeployTask{}).Where("id = ?", taskID).
		Update("executor_type", ExecutorTypeAgent).Error; err != nil {
		t.Fatalf("seed legacy agent row: %v", err)
	}

	detail, err := svc.GetTask(taskID, nil)
	if err != nil {
		t.Fatalf("legacy agent task must stay readable: %v", err)
	}
	if detail.ExecutorType != ExecutorTypeAgent {
		t.Fatalf("expected stored agent type, got %s", detail.ExecutorType)
	}

	renamed := "legacy-agent-renamed"
	updated, err := svc.updateTask(taskID, UpdateTaskRequest{Name: &renamed}, "1", nil)
	if err != nil {
		t.Fatalf("renaming a legacy agent task must not rewrite its type: %v", err)
	}
	if updated.ExecutorType != ExecutorTypeAgent || updated.Name != renamed {
		t.Fatalf("legacy row changed unexpectedly: %+v", updated)
	}

	if _, err := svc.StartTask(taskID, StartTaskRequest{}, "1", nil); err == nil || err.Error() != errDeployExecutorTypeUnsupported {
		t.Fatalf("expected %s when starting a legacy agent task, got %v", errDeployExecutorTypeUnsupported, err)
	}
}
