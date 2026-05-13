package dynamicmodule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gorm.io/gorm"
	"pantheon-ops/backend/pkg/testmysql"

	"pantheon-ops/backend/internal/scaffold"
)

func TestRegisterGeneratedModuleBusinessOnly(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	req := newGeneratedModuleRequest("system", "alert", "系统告警", "system_alert")

	if _, _, _, err := service.RegisterGeneratedModule(req); err == nil || err.Error() != "module.generate.business_only" {
		t.Fatalf("expected business-only error, got %v", err)
	}
}

func TestRegisterGeneratedModuleWritesRegistries(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	req := newGeneratedModuleRequest("business", "ticket", "工单管理", "biz_ticket")

	registration, writtenFiles, summary, err := service.RegisterGeneratedModule(req)
	if err != nil {
		t.Fatalf("register generated module: %v", err)
	}

	if registration.Name != "business.ticket" {
		t.Fatalf("unexpected module name: %s", registration.Name)
	}
	if registration.Status != ModuleStatusPendingActivation {
		t.Fatalf("unexpected module status: %d", registration.Status)
	}
	if len(writtenFiles) != 4 {
		t.Fatalf("unexpected written file count: %d", len(writtenFiles))
	}
	if summary == nil {
		t.Fatal("expected registration summary")
	}
	if summary.ModuleKey != "business.ticket" {
		t.Fatalf("unexpected summary module key: %s", summary.ModuleKey)
	}
	if summary.RoutePath != "/business/ticket" {
		t.Fatalf("unexpected route path: %s", summary.RoutePath)
	}
	if summary.ComponentKey != "business/ticket/TicketList" {
		t.Fatalf("unexpected component key: %s", summary.ComponentKey)
	}
	if summary.ParentMenuSource != "top_level" {
		t.Fatalf("unexpected parent menu source: %s", summary.ParentMenuSource)
	}
	if !summary.RequiresRestart || !summary.RequiresFrontendBuild {
		t.Fatalf("expected restart/build flags to be true: %+v", summary)
	}

	assertFileContains(t, filepath.Join(workspaceRoot, "backend", "modules", "business", "generated_registry.go"), `ticket "pantheon-ops/backend/modules/business/ticket"`)
	assertFileContains(t, filepath.Join(workspaceRoot, "backend", "modules", "business", "generated_registry.go"), "ticket.InitTicketModule")
	assertFileContains(t, filepath.Join(workspaceRoot, "frontend", "src", "modules", "generated", "business.ts"), "TicketModule")
	assertFileContains(t, filepath.Join(workspaceRoot, "frontend", "src", "core", "router", "generatedComponentRegistry.ts"), "'business/ticket/TicketList'")
}

func TestRegisterGeneratedModulePersistsActivationDiagnostics(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	req := newGeneratedModuleRequest("business", "ticket", "工单管理", "biz_ticket")

	if _, _, _, err := service.RegisterGeneratedModule(req); err != nil {
		t.Fatalf("register generated module: %v", err)
	}

	var registration ModuleRegistration
	if err := db.Where("name = ?", "business.ticket").First(&registration).Error; err != nil {
		t.Fatalf("load registration: %v", err)
	}
	if registration.Status != ModuleStatusPendingActivation {
		t.Fatalf("expected pending activation, got %d", registration.Status)
	}
	if registration.LastError != "" {
		t.Fatalf("expected empty last error, got %q", registration.LastError)
	}
	if strings.TrimSpace(registration.LastVerifiedAt) == "" {
		t.Fatal("expected last verified time to be persisted")
	}
	if !strings.Contains(registration.LastVerificationResult, `"code":"pending_activation"`) {
		t.Fatalf("expected pending activation verification result, got %s", registration.LastVerificationResult)
	}
	if !strings.Contains(registration.LastVerificationResult, `"code":"backend_registry"`) {
		t.Fatalf("expected backend registry verification result, got %s", registration.LastVerificationResult)
	}
}

func TestRegisterGeneratedModuleBuildsInferredParentSummary(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustCreateSystemMenuTable(t, db)
	mustInsertSystemMenuPath(t, db, "/business/cmdb")

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	req := newGeneratedModuleRequest("business", "cmdb/vendor", "供应商管理", "biz_cmdb_vendor")

	_, _, summary, err := service.RegisterGeneratedModule(req)
	if err != nil {
		t.Fatalf("register generated module: %v", err)
	}

	if summary == nil {
		t.Fatal("expected registration summary")
	}
	if summary.RoutePath != "/business/cmdb/vendor" {
		t.Fatalf("unexpected route path: %s", summary.RoutePath)
	}
	if summary.RouteName != "business-cmdb-vendor" {
		t.Fatalf("unexpected route name: %s", summary.RouteName)
	}
	if summary.ComponentKey != "business/cmdb/vendor/CmdbVendorList" {
		t.Fatalf("unexpected component key: %s", summary.ComponentKey)
	}
	if summary.PermissionPrefix != "business:cmdb:vendor" {
		t.Fatalf("unexpected permission prefix: %s", summary.PermissionPrefix)
	}
	if summary.ParentMenuPath != "/business/cmdb" {
		t.Fatalf("unexpected parent menu path: %s", summary.ParentMenuPath)
	}
	if summary.ParentMenuSource != "inferred" {
		t.Fatalf("unexpected parent menu source: %s", summary.ParentMenuSource)
	}
	if !summary.ParentMenuExists {
		t.Fatal("expected inferred parent menu to exist")
	}
	assertHasVerification(t, summary.Verifications, "parent_menu", "pass")
}

func TestRegisterGeneratedModuleBuildsGovernanceContractSummary(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	req := newGeneratedModuleRequest("business", "cmdb/host", "主机管理", "biz_cmdb_host")
	req.Schema.TemplateVersion = "v1"
	req.Schema.EnableDataScope = true
	req.Schema.DataScopeMode = "dept"
	req.Schema.Dependencies = []scaffold.ModuleDependency{{Module: "cmdb/vendor", Required: true, Reason: "host needs vendor"}}
	req.Schema.Relations = []scaffold.ModuleRelation{{
		Name:         "hostVendor",
		Type:         "lookup",
		TargetModule: "cmdb/vendor",
		LocalField:   "vendorId",
		TargetField:  "id",
	}}

	_, _, summary, err := service.RegisterGeneratedModule(req)
	if err != nil {
		t.Fatalf("register generated module: %v", err)
	}
	if summary.Contract.TemplateVersion != "v1" {
		t.Fatalf("unexpected template version: %s", summary.Contract.TemplateVersion)
	}
	if !summary.Contract.DataScopeEnabled || summary.Contract.DataScopeMode != "dept" {
		t.Fatalf("unexpected data scope contract: %+v", summary.Contract)
	}
	if summary.Contract.DependencyCount != 1 || summary.Contract.RelationCount != 1 {
		t.Fatalf("unexpected contract counts: %+v", summary.Contract)
	}
	if len(summary.Contract.Dependencies) != 1 || summary.Contract.Dependencies[0].Module != "cmdb/vendor" {
		t.Fatalf("unexpected contract dependencies: %+v", summary.Contract.Dependencies)
	}
	if len(summary.Contract.Relations) != 1 || summary.Contract.Relations[0].Name != "hostVendor" {
		t.Fatalf("unexpected contract relations: %+v", summary.Contract.Relations)
	}
	assertHasVerification(t, summary.Verifications, "contract_governance", "pass")
}

func TestRegisterGeneratedModuleNormalizesExplicitParentMenu(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustCreateSystemMenuTable(t, db)
	mustInsertSystemMenuPath(t, db, "/business/cmdb")

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	req := newGeneratedModuleRequest("business", "cmdb/host", "主机管理", "biz_cmdb_host")
	req.Schema.ParentMenu = "business/cmdb"

	_, _, summary, err := service.RegisterGeneratedModule(req)
	if err != nil {
		t.Fatalf("register generated module: %v", err)
	}
	if summary.ParentMenuPath != "/business/cmdb" {
		t.Fatalf("unexpected normalized parent menu path: %s", summary.ParentMenuPath)
	}
	if summary.ParentMenuSource != "explicit" {
		t.Fatalf("unexpected parent menu source: %s", summary.ParentMenuSource)
	}
	if !summary.ParentMenuExists {
		t.Fatal("expected normalized explicit parent menu to exist")
	}
}

func TestUnregisterAndRegisterManagedModule_RewritesRegistriesWithoutPurgingSource(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustCreateSystemMenuTable(t, db)
	mustCreateSystemRolePermissionTable(t, db)

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	req := newGeneratedModuleRequest("business", "asset", "资产管理", "biz_asset")
	if _, _, _, err := service.RegisterGeneratedModule(req); err != nil {
		t.Fatalf("register generated module: %v", err)
	}

	moduleDir := filepath.Join(workspaceRoot, "backend", "modules", "business", "asset")
	if err := service.UnregisterModule("business.asset", false, false); err != nil {
		t.Fatalf("unregister generated module: %v", err)
	}
	if _, err := os.Stat(moduleDir); err != nil {
		t.Fatalf("expected generated source to be preserved, got %v", err)
	}
	assertFileNotContains(t, filepath.Join(workspaceRoot, "backend", "modules", "business", "generated_registry.go"), "business/asset")

	registration, err := service.RegisterManagedModule("business.asset")
	if err != nil {
		t.Fatalf("register managed module: %v", err)
	}
	if registration.Status != ModuleStatusPendingActivation {
		t.Fatalf("expected pending activation, got %d", registration.Status)
	}
	assertFileContains(t, filepath.Join(workspaceRoot, "backend", "modules", "business", "generated_registry.go"), "business/asset")
}

func TestSyncBuiltInModules_PromotesGeneratedSchemaToManagedModule(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustWriteGeneratedRegistryStubs(t, workspaceRoot)
	mustWriteFile(t, filepath.Join(workspaceRoot, "schema", "generated", "business", "cmdb", "host.json"), `{
  "name": "cmdb/host",
  "displayName": "主机管理",
  "scope": "business",
  "model": { "tableName": "biz_cmdb_host" }
}`)
	mustWriteFile(t, filepath.Join(workspaceRoot, "backend", "modules", "business", "cmdb", "host", "module.go"), "package host\n")
	mustWriteFile(t, filepath.Join(workspaceRoot, "frontend", "src", "modules", "business", "cmdb", "host", "index.ts"), "export const CmdbHostModule = {}\n")
	mustWriteFile(t, filepath.Join(workspaceRoot, "frontend", "src", "modules", "business", "cmdb", "host", "CmdbHostList.tsx"), "export default function CmdbHostList() { return null }\n")
	mustCreateSystemMenuTable(t, db)
	mustInsertSystemMenuPath(t, db, "/business/cmdb/host")

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	if err := service.SyncBuiltInModules(); err != nil {
		t.Fatalf("sync modules: %v", err)
	}

	var registration ModuleRegistration
	if err := db.Where("name = ?", "business.cmdb.host").First(&registration).Error; err != nil {
		t.Fatalf("load registration: %v", err)
	}
	if registration.ModelTableName != "biz_cmdb_host" {
		t.Fatalf("unexpected table name: %s", registration.ModelTableName)
	}
	if registration.Status != ModuleStatusActive {
		t.Fatalf("unexpected status: %d", registration.Status)
	}
	assertFileContains(t, filepath.Join(workspaceRoot, "backend", "modules", "business", "generated_registry.go"), "backend/modules/business/cmdb/host")
	assertFileContains(t, filepath.Join(workspaceRoot, "frontend", "src", "core", "router", "generatedComponentRegistry.ts"), "business/cmdb/host/CmdbHostList")
}

func TestSyncBuiltInModules_PreservesUninstalledManagedModuleAndRemovesRegistryRefs(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustWriteGeneratedRegistryStubs(t, workspaceRoot)
	mustWriteFile(t, filepath.Join(workspaceRoot, "schema", "generated", "business", "asset.json"), `{
  "name": "asset",
  "displayName": "资产管理",
  "scope": "business",
  "model": { "tableName": "biz_asset" }
}`)
	mustWriteFile(t, filepath.Join(workspaceRoot, "backend", "modules", "business", "asset", "module.go"), "package asset\n")
	mustWriteFile(t, filepath.Join(workspaceRoot, "frontend", "src", "modules", "business", "asset", "index.ts"), "export const AssetModule = {}\n")
	mustWriteFile(t, filepath.Join(workspaceRoot, "frontend", "src", "modules", "business", "asset", "AssetList.tsx"), "export default function AssetList() { return null }\n")
	mustCreateSystemMenuTable(t, db)
	mustInsertSystemMenuPath(t, db, "/business/asset")

	registration := ModuleRegistration{
		Name:           "business.asset",
		DisplayName:    "资产管理",
		Scope:          "business",
		Source:         "generated",
		ModelTableName: "biz_asset",
		Status:         ModuleStatusUninstalled,
		InstalledAt:    "2026-04-01T00:00:00Z",
		UninstalledAt:  "2026-04-02T00:00:00Z",
	}
	if err := db.Create(&registration).Error; err != nil {
		t.Fatalf("seed registration: %v", err)
	}
	if err := scaffold.WriteGeneratedRegistries(workspaceRoot, []scaffold.GeneratedModuleRef{{Scope: "business", Name: "asset"}}); err != nil {
		t.Fatalf("seed generated registries: %v", err)
	}

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	if err := service.SyncBuiltInModules(); err != nil {
		t.Fatalf("sync modules: %v", err)
	}

	var refreshed ModuleRegistration
	if err := db.Where("name = ?", "business.asset").First(&refreshed).Error; err != nil {
		t.Fatalf("load registration: %v", err)
	}
	if refreshed.Status != ModuleStatusUninstalled {
		t.Fatalf("expected uninstalled status to be preserved, got %d", refreshed.Status)
	}
	assertFileNotContains(t, filepath.Join(workspaceRoot, "backend", "modules", "business", "generated_registry.go"), "business/asset")
	assertFileNotContains(t, filepath.Join(workspaceRoot, "frontend", "src", "core", "router", "generatedComponentRegistry.ts"), "business/asset/AssetList")
}

func TestRebuildGeneratedRegistries_SkipsMissingManagedSource(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustWriteGeneratedRegistryStubs(t, workspaceRoot)

	registration := ModuleRegistration{
		Name:           "business.asset",
		DisplayName:    "资产管理",
		Scope:          "business",
		Source:         "generated",
		ModelTableName: "biz_asset",
		Status:         ModuleStatusActive,
		InstalledAt:    "2026-04-01T00:00:00Z",
	}
	if err := db.Create(&registration).Error; err != nil {
		t.Fatalf("seed registration: %v", err)
	}
	if err := scaffold.WriteGeneratedRegistries(workspaceRoot, []scaffold.GeneratedModuleRef{{Scope: "business", Name: "asset"}}); err != nil {
		t.Fatalf("seed generated registries: %v", err)
	}

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	if err := service.RebuildGeneratedRegistries(); err != nil {
		t.Fatalf("rebuild registries: %v", err)
	}

	assertFileNotContains(t, filepath.Join(workspaceRoot, "backend", "modules", "business", "generated_registry.go"), "business/asset")
	assertFileNotContains(t, filepath.Join(workspaceRoot, "frontend", "src", "modules", "generated", "business.ts"), "AssetModule")
	assertFileNotContains(t, filepath.Join(workspaceRoot, "frontend", "src", "core", "router", "generatedComponentRegistry.ts"), "business/asset/AssetList")
	assertFileNotContains(t, filepath.Join(workspaceRoot, "frontend", "src", "core", "router", "generatedComponentRegistry.ts"), "lazy(() =>")
}

func TestAuditAndRepairGeneratedRegistries_MarksMissingSourceAndRewritesSummary(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustWriteGeneratedRegistryStubs(t, workspaceRoot)
	mustWriteFile(t, filepath.Join(workspaceRoot, "schema", "generated", "business", "ticket.json"), `{
  "name": "ticket",
  "displayName": "工单管理",
  "scope": "business",
  "model": { "tableName": "biz_ticket" }
}`)
	mustWriteFile(t, filepath.Join(workspaceRoot, "backend", "modules", "business", "ticket", "module.go"), "package ticket\n")
	mustWriteFile(t, filepath.Join(workspaceRoot, "frontend", "src", "modules", "business", "ticket", "index.ts"), "export const TicketModule = {}\n")
	mustWriteFile(t, filepath.Join(workspaceRoot, "frontend", "src", "modules", "business", "ticket", "TicketList.tsx"), "export default function TicketList() { return null }\n")

	activeMissing := ModuleRegistration{
		Name:           "business.asset",
		DisplayName:    "资产管理",
		Scope:          "business",
		Source:         "generated",
		ModelTableName: "biz_asset",
		Status:         ModuleStatusActive,
		InstalledAt:    "2026-04-01T00:00:00Z",
	}
	keptUninstalled := ModuleRegistration{
		Name:           "business.vendor",
		DisplayName:    "供应商管理",
		Scope:          "business",
		Source:         "generated",
		ModelTableName: "biz_vendor",
		Status:         ModuleStatusUninstalled,
		InstalledAt:    "2026-04-01T00:00:00Z",
		UninstalledAt:  "2026-04-02T00:00:00Z",
	}
	if err := db.Create(&activeMissing).Error; err != nil {
		t.Fatalf("seed active registration: %v", err)
	}
	if err := db.Create(&keptUninstalled).Error; err != nil {
		t.Fatalf("seed uninstalled registration: %v", err)
	}
	if err := scaffold.WriteGeneratedRegistries(workspaceRoot, []scaffold.GeneratedModuleRef{
		{Scope: "business", Name: "asset"},
		{Scope: "business", Name: "vendor"},
	}); err != nil {
		t.Fatalf("seed generated registries: %v", err)
	}

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	summary, err := service.AuditAndRepairGeneratedRegistries()
	if err != nil {
		t.Fatalf("audit and repair registries: %v", err)
	}
	if summary == nil {
		t.Fatal("expected repair summary")
	}
	if summary.CheckedModules != 3 {
		t.Fatalf("unexpected checked module count: %d", summary.CheckedModules)
	}
	if summary.GeneratedRegistryRefs != 1 {
		t.Fatalf("unexpected generated registry refs: %d", summary.GeneratedRegistryRefs)
	}
	if summary.MarkedUninstalledModules != 1 {
		t.Fatalf("unexpected marked uninstalled count: %d", summary.MarkedUninstalledModules)
	}
	if summary.ArtifactReadyModules != 1 {
		t.Fatalf("unexpected artifact ready count: %d", summary.ArtifactReadyModules)
	}
	if summary.PreservedUninstalledCount != 2 {
		t.Fatalf("unexpected preserved uninstalled count: %d", summary.PreservedUninstalledCount)
	}

	var refreshed ModuleRegistration
	if err := db.Where("name = ?", "business.asset").First(&refreshed).Error; err != nil {
		t.Fatalf("load repaired registration: %v", err)
	}
	if refreshed.Status != ModuleStatusUninstalled {
		t.Fatalf("expected asset to be marked uninstalled, got %d", refreshed.Status)
	}
	if refreshed.LastError != "module.artifacts_missing" {
		t.Fatalf("expected artifact missing diagnostic, got %q", refreshed.LastError)
	}
	if strings.TrimSpace(refreshed.LastVerifiedAt) == "" {
		t.Fatal("expected repair verification time to be persisted")
	}
	if !strings.Contains(refreshed.LastVerificationResult, `"code":"artifact_check"`) {
		t.Fatalf("expected artifact check verification result, got %s", refreshed.LastVerificationResult)
	}

	assertFileContains(t, filepath.Join(workspaceRoot, "backend", "modules", "business", "generated_registry.go"), "business/ticket")
	assertFileNotContains(t, filepath.Join(workspaceRoot, "backend", "modules", "business", "generated_registry.go"), "business/asset")
	assertFileNotContains(t, filepath.Join(workspaceRoot, "backend", "modules", "business", "generated_registry.go"), "business/vendor")
	assertFileContains(t, filepath.Join(workspaceRoot, "frontend", "src", "core", "router", "generatedComponentRegistry.ts"), "business/ticket/TicketList")
}

func TestListRegisteredModulesDoesNotRewriteGeneratedRegistries(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustWriteGeneratedRegistryStubs(t, workspaceRoot)
	if err := scaffold.WriteGeneratedRegistries(workspaceRoot, []scaffold.GeneratedModuleRef{{Scope: "business", Name: "asset"}}); err != nil {
		t.Fatalf("seed generated registries: %v", err)
	}

	registration := ModuleRegistration{
		Name:           "business.asset",
		DisplayName:    "资产管理",
		Scope:          "business",
		Source:         "generated",
		ModelTableName: "biz_asset",
		Status:         ModuleStatusUninstalled,
		InstalledAt:    "2026-04-01T00:00:00Z",
		UninstalledAt:  "2026-04-02T00:00:00Z",
	}
	if err := db.Create(&registration).Error; err != nil {
		t.Fatalf("seed registration: %v", err)
	}

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	if _, err := service.ListRegisteredModules(); err != nil {
		t.Fatalf("list registered modules: %v", err)
	}

	assertFileContains(t, filepath.Join(workspaceRoot, "backend", "modules", "business", "generated_registry.go"), "business/asset")
	assertFileContains(t, filepath.Join(workspaceRoot, "frontend", "src", "core", "router", "generatedComponentRegistry.ts"), "business/asset/AssetList")
}

func TestListRegisteredModulesDoesNotMarkBusinessStaticModuleAsBuiltIn(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustCreateSystemMenuTable(t, db)

	records := []ModuleRegistration{
		{
			Name:        "business.cmdb",
			DisplayName: "CMDB",
			Scope:       "business",
			Source:      "static",
			Status:      ModuleStatusActive,
			InstalledAt: "2026-05-04T12:29:36+08:00",
		},
		{
			Name:        "system",
			DisplayName: "system",
			Scope:       "system",
			Source:      "core",
			Status:      ModuleStatusActive,
			InstalledAt: "2026-04-25T16:31:20+08:00",
		},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("seed registrations: %v", err)
	}

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	modules, err := service.ListRegisteredModules()
	if err != nil {
		t.Fatalf("list registered modules: %v", err)
	}

	byName := make(map[string]ModuleRegistrationResp, len(modules))
	for _, module := range modules {
		byName[module.Name] = module
	}
	if byName["business.cmdb"].BuiltIn {
		t.Fatal("expected business static module to not be marked built-in")
	}
	if !byName["system"].BuiltIn {
		t.Fatal("expected system shell module to be marked built-in")
	}
}

func TestPurgeModuleAllowsBusinessStaticModuleWithoutTable(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustWriteGeneratedRegistryStubs(t, workspaceRoot)
	mustCreateSystemMenuTable(t, db)
	mustCreateSystemI18nTable(t, db)

	registration := ModuleRegistration{
		Name:        "business.cmdb",
		DisplayName: "CMDB",
		Scope:       "business",
		Source:      "static",
		Status:      ModuleStatusActive,
		InstalledAt: "2026-05-04T12:29:36+08:00",
	}
	if err := db.Create(&registration).Error; err != nil {
		t.Fatalf("seed registration: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_menu (path, type, module) VALUES ('/business/cmdb', 'M', 'business.cmdb')`).Error; err != nil {
		t.Fatalf("seed menu: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_i18n (module) VALUES ('business.cmdb')`).Error; err != nil {
		t.Fatalf("seed i18n: %v", err)
	}

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	if err := service.PurgeModule("business.cmdb", false, true); err != nil {
		t.Fatalf("purge business static module: %v", err)
	}

	var count int64
	if err := db.Model(&ModuleRegistration{}).Where("name = ?", "business.cmdb").Count(&count).Error; err != nil {
		t.Fatalf("count registration: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected registration to be deleted, got %d", count)
	}
	if err := db.Table("system_menu").Where("module = ?", "business.cmdb").Count(&count).Error; err != nil {
		t.Fatalf("count menu: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected menu rows to be deleted, got %d", count)
	}
	if err := db.Table("system_i18n").Where("module = ?", "business.cmdb").Count(&count).Error; err != nil {
		t.Fatalf("count i18n: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected i18n rows to be deleted, got %d", count)
	}
}

func TestUnregisterModuleRejectsUnsafeManagedTableName(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustCreateSystemMenuTable(t, db)
	mustCreateSystemRolePermissionTable(t, db)

	registration := ModuleRegistration{
		Name:           "business.asset",
		DisplayName:    "资产管理",
		Scope:          "business",
		Source:         "generated",
		ModelTableName: "biz_asset;drop table system_user",
		Status:         ModuleStatusActive,
		InstalledAt:    "2026-04-01T00:00:00Z",
	}
	if err := db.Create(&registration).Error; err != nil {
		t.Fatalf("seed registration: %v", err)
	}

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	err := service.UnregisterModule("business.asset", true, false)
	if err == nil || err.Error() != "module.generate.invalid_table_name" {
		t.Fatalf("expected invalid table name error, got %v", err)
	}
}

func openDynamicModuleTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := testmysql.Open(t)
	if err := db.AutoMigrate(&ModuleRegistration{}); err != nil {
		t.Fatalf("migrate module registration: %v", err)
	}
	return db
}

func mustCreateSystemMenuTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`CREATE TABLE system_menu (
		id BIGINT PRIMARY KEY AUTO_INCREMENT,
		path VARCHAR(255),
		type VARCHAR(8),
		module VARCHAR(64)
	)`).Error; err != nil {
		t.Fatalf("create system_menu table: %v", err)
	}
}

func mustInsertSystemMenuPath(t *testing.T, db *gorm.DB, path string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO system_menu (path, type, module) VALUES (?, 'C', 'business.cmdb')`, path).Error; err != nil {
		t.Fatalf("insert system_menu path %s: %v", path, err)
	}
}

func mustCreateSystemRolePermissionTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`CREATE TABLE system_role_permission (
		id BIGINT PRIMARY KEY AUTO_INCREMENT,
		role_id BIGINT,
		permission_key VARCHAR(255)
	)`).Error; err != nil {
		t.Fatalf("create system_role_permission table: %v", err)
	}
}

func mustCreateSystemI18nTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`CREATE TABLE system_i18n (
		id BIGINT PRIMARY KEY AUTO_INCREMENT,
		module VARCHAR(64)
	)`).Error; err != nil {
		t.Fatalf("create system_i18n table: %v", err)
	}
}

func prepareDynamicModuleWorkspace(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "go.mod"), "module pantheon-ops\n\ngo 1.25.4\n")
	mustMkdirAll(t, filepath.Join(root, "backend", "modules", "business"))
	mustMkdirAll(t, filepath.Join(root, "backend", "modules", "system", "iam", "menu"))
	mustMkdirAll(t, filepath.Join(root, "frontend", "src", "modules", "generated"))
	mustMkdirAll(t, filepath.Join(root, "frontend", "src", "core", "router"))
	mustMkdirAll(t, filepath.Join(root, "schema", "generated", "business"))
	return root
}

func mustWriteGeneratedRegistryStubs(t *testing.T, root string) {
	t.Helper()
	mustWriteFile(t, filepath.Join(root, "backend", "modules", "business", "generated_registry.go"), "package business\n")
	mustWriteFile(t, filepath.Join(root, "backend", "modules", "system", "generated_registry.go"), "package system\n")
	mustWriteFile(t, filepath.Join(root, "backend", "modules", "system", "iam", "menu", "generated_component_registry.go"), "package iam\n")
	mustWriteFile(t, filepath.Join(root, "frontend", "src", "modules", "generated", "business.ts"), "export const generatedBusinessModules = []\n")
	mustWriteFile(t, filepath.Join(root, "frontend", "src", "modules", "generated", "system.ts"), "export const generatedSystemModules = []\n")
	mustWriteFile(t, filepath.Join(root, "frontend", "src", "core", "router", "generatedComponentRegistry.ts"), "export const generatedComponentRegistry = {}\n")
}

func newGeneratedModuleRequest(scope string, name string, displayName string, tableName string) *scaffold.RegisterGeneratedModuleRequest {
	req := &scaffold.RegisterGeneratedModuleRequest{
		Files: []scaffold.GeneratedFile{
			{
				Path:     filepath.ToSlash(filepath.Join("backend", "modules", scope, name, "module.go")),
				Content:  "package " + name + "\n",
				Language: "go",
			},
			{
				Path:     filepath.ToSlash(filepath.Join("frontend", "src", "modules", scope, name, "index.ts")),
				Content:  "export const TicketModule = {}\n",
				Language: "typescript",
			},
			{
				Path:     filepath.ToSlash(filepath.Join("frontend", "src", "modules", scope, name, "TicketList.tsx")),
				Content:  "export default function TicketList() { return null; }\n",
				Language: "tsx",
			},
		},
	}
	req.Schema.Name = name
	req.Schema.Scope = scope
	req.Schema.DisplayName = displayName
	req.Schema.Model.TableName = tableName
	return req
}

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}

func mustWriteFile(t *testing.T, target string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", target, err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", target, err)
	}
}

func assertFileContains(t *testing.T, target string, fragment string) {
	t.Helper()
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read %s: %v", target, err)
	}
	if !strings.Contains(string(content), fragment) {
		t.Fatalf("expected %s to contain %q, got:\n%s", target, fragment, string(content))
	}
}

func assertFileNotContains(t *testing.T, target string, fragment string) {
	t.Helper()
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read %s: %v", target, err)
	}
	if strings.Contains(string(content), fragment) {
		t.Fatalf("expected %s not to contain %q, got:\n%s", target, fragment, string(content))
	}
}

func assertHasVerification(t *testing.T, items []GeneratedModuleVerification, code string, status string) {
	t.Helper()
	for _, item := range items {
		if item.Code == code && item.Status == status {
			return
		}
	}
	t.Fatalf("expected verification %s with status %s, got %#v", code, status, items)
}
