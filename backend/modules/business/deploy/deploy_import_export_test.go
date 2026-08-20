package deploy

import (
	"testing"

	"pantheon-base/modules/business/cmdb"
)

func TestDeployPackageImportAndExport(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	result, err := svc.ImportPackages([][]string{
		deployPackageCSVHeaders,
		{"demo", "1.0.0", "demo package", "echo install", "", "fixed", "", "{}", "", "", "", "enabled"},
	}, "1")
	if err != nil || !result.Applied || result.Created != 1 {
		t.Fatalf("unexpected import result: %#v, %v", result, err)
	}
	file, err := svc.ExportPackages(PackageQuery{})
	if err != nil || len(file.Rows) != 1 || file.Rows[0][0] != "demo" {
		t.Fatalf("unexpected export: %#v, %v", file, err)
	}
}

func TestDeployTaskExportScopesRows(t *testing.T) {
	db := setupDeployTestDB(t)
	svc := NewDeployService(db, cmdb.NewDeployCMDBCapability(db))
	scopeID, hostID := seedDeployReliabilityFixture(t, db, "export-host", "10.40.0.210")
	pkg, err := svc.CreatePackage(CreatePackageRequest{Name: "export-pkg", Version: "1.0.0", InstallCommand: "echo install"}, "1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.CreateTask(CreateTaskRequest{Name: "export task", PackageID: pkg.ID, BusinessScopeID: scopeID, TargetType: TargetTypeHost, TargetIDs: []uint64{hostID}}, "1", nil); err != nil {
		t.Fatal(err)
	}
	file, err := svc.ExportTasks(TaskQuery{}, nil)
	if err != nil || len(file.Rows) != 1 || file.Rows[0][1] != "export task" {
		t.Fatalf("unexpected export: %#v, %v", file, err)
	}
}
