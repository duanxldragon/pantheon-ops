package dynamicmodule

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	systemi18n "pantheon-ops/backend/modules/system/i18n"
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

func TestNewDynamicModuleServiceHonorsConfiguredWorkspaceRoot(t *testing.T) {
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	t.Setenv("PANTHEON_WORKSPACE_ROOT", workspaceRoot)

	service := NewDynamicModuleService(nil)
	if service.workspaceRoot != workspaceRoot {
		t.Fatalf("expected workspace root %s, got %s", workspaceRoot, service.workspaceRoot)
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
	if len(writtenFiles) != 5 {
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
	assertFileContains(t, filepath.Join(workspaceRoot, "frontend", "src", "core", "router", "generatedComponentRegistry.ts"), "'business/ticket/TicketDetail'")
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

func TestRegisterGeneratedModulePersistsAutoRecycleFlag(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	req := newGeneratedModuleRequest("business", "qaticket", "QA 工单", "biz_qa_ticket")
	req.Schema.Metadata.AutoRecycle = true

	if _, _, _, err := service.RegisterGeneratedModule(req); err != nil {
		t.Fatalf("register generated module: %v", err)
	}

	var registration ModuleRegistration
	if err := db.Where("name = ?", "business.qaticket").First(&registration).Error; err != nil {
		t.Fatalf("load registration: %v", err)
	}
	if !registration.AutoRecycle {
		t.Fatal("expected auto recycle flag to be persisted")
	}
}

func TestGenerateAndRegisterModuleHandlerPersistsAutoRecycleMetadata(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}
	handler := NewDynamicModuleHandler(service)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/system/dynamic-modules/generate", handler.GenerateAndRegisterModule)

	body := map[string]any{
		"schema": map[string]any{
			"name":        "autorecycletest",
			"scope":       "business",
			"displayName": "自动回收测试",
			"metadata": map[string]any{
				"owner":       "codex",
				"autoRecycle": true,
			},
			"model": map[string]any{
				"tableName": "biz_autorecycle_test",
			},
		},
		"overwrite": true,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/system/dynamic-modules/generate", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Code int `json:"code"`
		Data struct {
			Module ModuleRegistration `json:"module"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Data.Module.AutoRecycle {
		t.Fatal("expected handler response to preserve auto recycle flag")
	}

	var registration ModuleRegistration
	if err := db.Where("name = ?", "business.autorecycletest").First(&registration).Error; err != nil {
		t.Fatalf("load registration: %v", err)
	}
	if !registration.AutoRecycle {
		t.Fatal("expected persisted registration to preserve auto recycle flag")
	}

	schema, err := service.GetManagedModuleSchema("business.autorecycletest")
	if err != nil {
		t.Fatalf("load generated schema: %v", err)
	}
	if !schema.Metadata.AutoRecycle {
		t.Fatal("expected generated schema file to preserve auto recycle flag")
	}
}

func TestListRegisteredModulesKeepsPendingGeneratedModulePendingBeforeActivationAudit(t *testing.T) {
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

	modules, err := service.ListRegisteredModules()
	if err != nil {
		t.Fatalf("list registered modules: %v", err)
	}

	if len(modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(modules))
	}
	if modules[0].Status != ModuleStatusPendingActivation {
		t.Fatalf("expected module to stay pending before activation audit, got %d", modules[0].Status)
	}
}

func TestAuditPendingGeneratedModuleActivationsPromotesModuleAfterRuntimeAndBundleSignals(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustCreateSystemMenuTable(t, db)

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	req := newGeneratedModuleRequest("business", "ticket", "工单管理", "biz_ticket")
	if _, _, _, err := service.RegisterGeneratedModule(req); err != nil {
		t.Fatalf("register generated module: %v", err)
	}
	mustInsertSystemMenuModule(t, db, "/business/ticket", "business.ticket")
	mustWriteFile(t, filepath.Join(workspaceRoot, "frontend", "dist", "assets", "app.js"), "built")

	summary, err := service.AuditPendingGeneratedModuleActivations()
	if err != nil {
		t.Fatalf("audit pending activations: %v", err)
	}
	if summary.ActivatedModules != 1 || summary.PendingModules != 0 {
		t.Fatalf("unexpected activation summary: %+v", summary)
	}

	var registration ModuleRegistration
	if err := db.Where("name = ?", "business.ticket").First(&registration).Error; err != nil {
		t.Fatalf("load registration: %v", err)
	}
	if registration.Status != ModuleStatusActive {
		t.Fatalf("expected module to be activated, got %d", registration.Status)
	}
	if !strings.Contains(registration.LastVerificationResult, `"code":"activation_ready"`) {
		t.Fatalf("expected activation ready verification, got %s", registration.LastVerificationResult)
	}
}

func TestAuditPendingGeneratedModuleActivationsKeepsModulePendingWhenFrontendBuildIsMissing(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustCreateSystemMenuTable(t, db)

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	req := newGeneratedModuleRequest("business", "ticket", "工单管理", "biz_ticket")
	if _, _, _, err := service.RegisterGeneratedModule(req); err != nil {
		t.Fatalf("register generated module: %v", err)
	}
	mustInsertSystemMenuModule(t, db, "/business/ticket", "business.ticket")

	summary, err := service.AuditPendingGeneratedModuleActivations()
	if err != nil {
		t.Fatalf("audit pending activations: %v", err)
	}
	if summary.ActivatedModules != 0 || summary.PendingModules != 1 {
		t.Fatalf("unexpected activation summary: %+v", summary)
	}

	var registration ModuleRegistration
	if err := db.Where("name = ?", "business.ticket").First(&registration).Error; err != nil {
		t.Fatalf("load registration: %v", err)
	}
	if registration.Status != ModuleStatusPendingActivation {
		t.Fatalf("expected module to stay pending, got %d", registration.Status)
	}
	if !strings.Contains(registration.LastVerificationResult, `"code":"frontend_build_required"`) {
		t.Fatalf("expected frontend build required verification, got %s", registration.LastVerificationResult)
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

func TestGetManagedModuleSchemaLoadsGeneratedSchema(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	req := newGeneratedModuleRequest("business", "cmdb/change", "变更记录", "biz_cmdb_change")
	req.Schema.Relations = []scaffold.ModuleRelation{{
		Name:         "changeAsset",
		Type:         "oneToMany",
		TargetModule: "cmdb/asset",
		LocalField:   "id",
		TargetField:  "assetId",
	}}

	if _, _, _, err := service.RegisterGeneratedModule(req); err != nil {
		t.Fatalf("register generated module: %v", err)
	}

	schema, err := service.GetManagedModuleSchema("business.cmdb.change")
	if err != nil {
		t.Fatalf("get managed module schema: %v", err)
	}
	if schema.Name != "cmdb/change" {
		t.Fatalf("unexpected schema name: %s", schema.Name)
	}
	if schema.Scope != "business" {
		t.Fatalf("unexpected schema scope: %s", schema.Scope)
	}
	if schema.Model.TableName != "biz_cmdb_change" {
		t.Fatalf("unexpected schema table name: %s", schema.Model.TableName)
	}
	if len(schema.Relations) != 1 || schema.Relations[0].TargetField != "assetId" {
		t.Fatalf("unexpected schema relations: %+v", schema.Relations)
	}
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
	if _, err := service.UnregisterModule("business.asset", false, false); err != nil {
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
	assertFileContains(t, filepath.Join(workspaceRoot, "frontend", "src", "core", "router", "generatedComponentRegistry.ts"), "business/cmdb/host/CmdbHostDetail")
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

func TestSyncBuiltInModules_NormalizesLegacyLowcodeAlias(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustWriteGeneratedRegistryStubs(t, workspaceRoot)
	mustCreateSystemMenuTable(t, db)
	mustCreateSystemI18nTable(t, db)
	mustInsertSystemMenuModule(t, db, "/system/lowcode", "platform.lowcode")

	legacy := ModuleRegistration{
		Name:        "platform.lowcode",
		DisplayName: "platform.lowcode",
		Scope:       "platform",
		Source:      "core",
		Status:      ModuleStatusActive,
		InstalledAt: "2026-05-20T00:00:00Z",
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("seed legacy registration: %v", err)
	}
	if err := db.Exec(`INSERT INTO system_i18n (module) VALUES ('platform.lowcode')`).Error; err != nil {
		t.Fatalf("seed legacy i18n: %v", err)
	}

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	if err := service.SyncBuiltInModules(); err != nil {
		t.Fatalf("sync modules: %v", err)
	}

	var menuModule string
	if err := db.Table("system_menu").Select("module").Where("path = ?", "/system/lowcode").Scan(&menuModule).Error; err != nil {
		t.Fatalf("load repaired menu module: %v", err)
	}
	if menuModule != "system.lowcode" {
		t.Fatalf("expected menu module to be normalized, got %s", menuModule)
	}

	var registration ModuleRegistration
	if err := db.Where("name = ?", "system.lowcode").First(&registration).Error; err != nil {
		t.Fatalf("load normalized registration: %v", err)
	}
	if registration.Scope != "system" {
		t.Fatalf("expected normalized scope system, got %s", registration.Scope)
	}

	var legacyCount int64
	if err := db.Model(&ModuleRegistration{}).Where("name = ?", "platform.lowcode").Count(&legacyCount).Error; err != nil {
		t.Fatalf("count legacy registration: %v", err)
	}
	if legacyCount != 0 {
		t.Fatalf("expected legacy registration removed, got %d", legacyCount)
	}

	var i18nCount int64
	if err := db.Table("system_i18n").Where("module = ?", "system.lowcode").Count(&i18nCount).Error; err != nil {
		t.Fatalf("count normalized i18n: %v", err)
	}
	if i18nCount != 1 {
		t.Fatalf("expected normalized i18n row, got %d", i18nCount)
	}
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
	mustCreateSystemRolePermissionTable(t, db)
	mustMigrateSystemI18n(t, db)

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
	i18nService := systemi18n.NewI18nService(db)
	if err := i18nService.BatchInsert([]systemi18n.SystemI18n{
		{Module: "business.cmdb", Group: "messages", Key: "business.cmdb.title", Locale: "zh-CN", Value: "配置中心"},
		{Module: "business.cmdb", Group: "messages", Key: "business.cmdb.title", Locale: "en-US", Value: "CMDB"},
	}); err != nil {
		t.Fatalf("seed i18n: %v", err)
	}

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	summary, err := service.PurgeModule("business.cmdb", false, true)
	if err != nil {
		t.Fatalf("purge business static module: %v", err)
	}
	if !summary.Triggered {
		t.Fatal("expected static module purge to trigger i18n lifecycle governance")
	}
	if summary.ObservedRows != 2 {
		t.Fatalf("expected static module purge to observe 2 rows, got %d", summary.ObservedRows)
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
	var rows []systemi18n.SystemI18n
	if err := db.Where("module = ?", "business.cmdb").Find(&rows).Error; err != nil {
		t.Fatalf("load i18n rows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected i18n rows to remain for lifecycle handling, got %d", len(rows))
	}
	for _, row := range rows {
		if row.LifecycleStatus != systemi18n.I18nLifecycleStatusObserving {
			t.Fatalf("expected observing lifecycle status, got %s", row.LifecycleStatus)
		}
	}
}

func TestPurgeModuleAutoDropsTableForAutoRecycleManagedModule(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustWriteGeneratedRegistryStubs(t, workspaceRoot)
	mustCreateManagedTable(t, db, "biz_tmp_asset")

	registration := ModuleRegistration{
		Name:           "business.tmpasset",
		DisplayName:    "临时资产",
		Scope:          "business",
		Source:         "generated",
		ModelTableName: "biz_tmp_asset",
		Status:         ModuleStatusActive,
		InstalledAt:    "2026-05-20T00:00:00Z",
		AutoRecycle:    true,
	}
	if err := db.Create(&registration).Error; err != nil {
		t.Fatalf("seed registration: %v", err)
	}

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	if _, err := service.PurgeModule("business.tmpasset", false, false); err != nil {
		t.Fatalf("purge managed module: %v", err)
	}
	if db.Migrator().HasTable("biz_tmp_asset") {
		t.Fatal("expected managed table to be dropped automatically")
	}
}

func TestPurgeModuleDoesNotDropTableForRegularManagedModuleWithoutExplicitDrop(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustWriteGeneratedRegistryStubs(t, workspaceRoot)
	mustCreateManagedTable(t, db, "biz_keep_asset")

	registration := ModuleRegistration{
		Name:           "business.keepasset",
		DisplayName:    "普通资产",
		Scope:          "business",
		Source:         "generated",
		ModelTableName: "biz_keep_asset",
		Status:         ModuleStatusActive,
		InstalledAt:    "2026-05-20T00:00:00Z",
	}
	if err := db.Create(&registration).Error; err != nil {
		t.Fatalf("seed registration: %v", err)
	}

	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
	}

	if _, err := service.PurgeModule("business.keepasset", false, false); err != nil {
		t.Fatalf("purge managed module: %v", err)
	}
	if !db.Migrator().HasTable("biz_keep_asset") {
		t.Fatal("expected managed table to be preserved without explicit drop")
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

	_, err := service.UnregisterModule("business.asset", true, false)
	if err == nil || err.Error() != "module.generate.invalid_table_name" {
		t.Fatalf("expected invalid table name error, got %v", err)
	}
}

func mustMigrateSystemI18n(t *testing.T, db *gorm.DB) {
	t.Helper()
	service := systemi18n.NewI18nService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate system_i18n: %v", err)
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

func mustInsertSystemMenuModule(t *testing.T, db *gorm.DB, path string, module string) {
	t.Helper()
	if err := db.Exec(`INSERT INTO system_menu (path, type, module) VALUES (?, 'C', ?)`, path, module).Error; err != nil {
		t.Fatalf("insert system_menu module %s: %v", module, err)
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

func mustCreateManagedTable(t *testing.T, db *gorm.DB, tableName string) {
	t.Helper()
	if err := db.Exec("CREATE TABLE " + tableName + " (id BIGINT PRIMARY KEY AUTO_INCREMENT)").Error; err != nil {
		t.Fatalf("create managed table %s: %v", tableName, err)
	}
}

func prepareDynamicModuleWorkspace(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "go.mod"), "module pantheon-ops\n\ngo 1.25.4\n")
	mustMkdirAll(t, filepath.Join(root, "backend", "modules", "business"))
	mustMkdirAll(t, filepath.Join(root, "backend", "modules", "system", "iam", "menu"))
	mustMkdirAll(t, filepath.Join(root, "frontend", "scripts"))
	mustMkdirAll(t, filepath.Join(root, "frontend", "src", "modules", "generated"))
	mustMkdirAll(t, filepath.Join(root, "frontend", "src", "core", "router"))
	mustMkdirAll(t, filepath.Join(root, "schema", "generated", "business"))
	mustWriteFile(t, filepath.Join(root, "frontend", "scripts", "export-generated-module.mjs"), `import { readFileSync } from 'node:fs';

function pascalCase(value) {
  return String(value || '')
    .split(/[_\-/\s]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1).toLowerCase())
    .join('');
}

const schema = JSON.parse(readFileSync(process.argv[2], 'utf8'));
const scope = schema.scope;
const name = schema.name;
const leafName = name.split('/').filter(Boolean).at(-1) || 'module';
const modelName = pascalCase(name);
const files = [
  {
    path: 'backend/modules/' + scope + '/' + name + '/module.go',
    content: 'package ' + leafName + '\n',
    language: 'go',
  },
  {
    path: 'frontend/src/modules/' + scope + '/' + name + '/index.ts',
    content: 'export const ' + modelName + 'Module = {}\n',
    language: 'typescript',
  },
  {
    path: 'frontend/src/modules/' + scope + '/' + name + '/' + modelName + 'List.tsx',
    content: 'export default function ' + modelName + 'List() { return null; }\n',
    language: 'tsx',
  },
  {
    path: 'frontend/src/modules/' + scope + '/' + name + '/' + modelName + 'Detail.tsx',
    content: 'export default function ' + modelName + 'Detail() { return null; }\n',
    language: 'tsx',
  },
];
process.stdout.write(JSON.stringify(files));
`)
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
	req := &scaffold.RegisterGeneratedModuleRequest{}
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
