package dynamicmodule

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/gorm"
)

// AUD-001 的编译闸门测试。
//
// 覆盖三条被漏掉的路径：
//  1. 生产构造函数必须注入真实编译（否则闸门在重构中会静默消失）；
//  2. 生成包编译失败 → 拒绝注册、不推进 registration 状态；
//  3. 工具链缺失 / 配置错误 → 同样归为 source_uncompilable，而不是放行。

func TestNewDynamicModuleServiceWiresGeneratedModuleBuild(t *testing.T) {
	service := NewDynamicModuleService(nil)
	if service.buildGeneratedModule == nil {
		t.Fatal("NewDynamicModuleService must inject the generated-module build check; " +
			"a nil builder disables the AUD-001 gate in production")
	}
}

// seedManagedGeneratedModule 造出一个"源码与 schema 都已存在"的受管模块工作区。
//
// 注册前置会依次校验：模块名合法 → 生成目录存在 → 生成包可编译。
// 前两步必须满足，测试才真正走到编译闸门，否则断言会变成"因为别的错误而失败"。
func seedManagedGeneratedModule(t *testing.T, workspaceRoot, scope, name string) {
	t.Helper()

	mustMkdirAll(t, filepath.Join(workspaceRoot, "backend", "modules", scope, name))
	mustMkdirAll(t, filepath.Join(workspaceRoot, "frontend", "src", "modules", scope, name))
	mustWriteFile(
		t,
		filepath.Join(workspaceRoot, "schema", "generated", scope, name+".json"),
		`{"name":"`+name+`","displayName":"资产管理","scope":"`+scope+`","model":{"tableName":"biz_`+name+`"}}`,
	)
}

func countModuleRegistrations(t *testing.T, db *gorm.DB, moduleName string) int64 {
	t.Helper()

	var count int64
	if err := db.Model(&ModuleRegistration{}).Where("name = ?", moduleName).Count(&count).Error; err != nil {
		t.Fatalf("count registrations: %v", err)
	}
	return count
}

func TestRegisterManagedModuleRejectsUncompilableGeneratedSource(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	seedManagedGeneratedModule(t, workspaceRoot, "business", "asset")

	var builtPackage string
	service := &DynamicModuleService{
		db:            db,
		workspaceRoot: workspaceRoot,
		buildGeneratedModule: func(_, packagePath string) error {
			builtPackage = packagePath
			return errors.New("module.register.source_uncompilable: build failed: undefined: contracts.ProtectedGroup")
		},
	}

	_, err := service.RegisterManagedModule("business.asset")
	if err == nil {
		t.Fatal("expected registration to be rejected when the generated package does not compile")
	}
	if got := common.ErrMessage(err); got != msgSourceUncompilable {
		t.Fatalf("expected error key %q, got %q (%v)", msgSourceUncompilable, got, err)
	}
	if builtPackage != "./modules/business/asset" {
		t.Fatalf("expected build target ./modules/business/asset, got %q", builtPackage)
	}
	if count := countModuleRegistrations(t, db, "business.asset"); count != 0 {
		t.Fatalf("rejected registration must not persist a row, found %d", count)
	}
}

func TestRegisterManagedModuleAdvancesStatusWhenGeneratedSourceCompiles(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustWriteGeneratedRegistryStubs(t, workspaceRoot)
	seedManagedGeneratedModule(t, workspaceRoot, "business", "asset")

	service := &DynamicModuleService{
		db:                   db,
		workspaceRoot:        workspaceRoot,
		buildGeneratedModule: func(string, string) error { return nil },
	}

	registration, err := service.RegisterManagedModule("business.asset")
	if err != nil {
		t.Fatalf("register managed module: %v", err)
	}
	if registration.Status != ModuleStatusPendingActivation {
		t.Fatalf("expected status %d, got %d", ModuleStatusPendingActivation, registration.Status)
	}
	if count := countModuleRegistrations(t, db, "business.asset"); count != 1 {
		t.Fatalf("expected exactly one registration row, found %d", count)
	}
}

// 这个用例把"测试专用的空 builder"钉成显式契约：只有测试构造的 service 才会缺 builder，
// 生产路径由 TestNewDynamicModuleServiceWiresGeneratedModuleBuild 守住。
func TestRegisterManagedModuleSkipsBuildCheckWhenBuilderIsNil(t *testing.T) {
	db := openDynamicModuleTestDB(t)
	workspaceRoot := prepareDynamicModuleWorkspace(t)
	mustWriteGeneratedRegistryStubs(t, workspaceRoot)
	seedManagedGeneratedModule(t, workspaceRoot, "business", "asset")

	service := &DynamicModuleService{db: db, workspaceRoot: workspaceRoot}

	if _, err := service.RegisterManagedModule("business.asset"); err != nil {
		t.Fatalf("nil builder must skip the compile check, got %v", err)
	}
}

func TestResolveGoBinaryRejectsRelativeOverride(t *testing.T) {
	t.Setenv(goBinaryEnvKey, "go")

	_, err := resolveGoBinary()
	if err == nil {
		t.Fatalf("%s must be an absolute path; a relative value must be rejected", goBinaryEnvKey)
	}
	if !strings.Contains(err.Error(), msgSourceUncompilable) {
		t.Fatalf("expected error to carry %q, got %v", msgSourceUncompilable, err)
	}
}

func TestRunGoBuildGeneratedModuleReportsMissingToolchain(t *testing.T) {
	// 指向一个必然不存在的绝对路径：不依赖本机是否装了 go，也不编译真实代码，
	// 只验证 exec 失败会被翻译成 source_uncompilable 而不是原样冒泡。
	missing := filepath.Join(t.TempDir(), "definitely-not-a-go-binary")
	t.Setenv(goBinaryEnvKey, missing)

	err := runGoBuildGeneratedModule(t.TempDir(), "./modules/business/asset")
	if err == nil {
		t.Fatal("expected build failure when the go binary does not exist")
	}
	if !strings.Contains(err.Error(), msgSourceUncompilable) {
		t.Fatalf("expected error to carry %q, got %v", msgSourceUncompilable, err)
	}
}

func TestRunGoBuildGeneratedModuleFailsOutsideGoModule(t *testing.T) {
	if _, err := resolveGoBinary(); err != nil {
		t.Skipf("go toolchain unavailable: %v", err)
	}

	// 空目录没有 go.mod，`go build` 必然失败；这条覆盖"真实 exec 路径"的失败分支，
	// 与上面用假二进制覆盖的分支互补。
	workspaceRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspaceRoot, "backend"), 0o755); err != nil {
		t.Fatalf("mkdir backend: %v", err)
	}

	err := runGoBuildGeneratedModule(workspaceRoot, "./modules/business/asset")
	if err == nil {
		t.Fatal("expected go build to fail outside a module")
	}
	if !strings.Contains(err.Error(), msgSourceUncompilable) {
		t.Fatalf("expected error to carry %q, got %v", msgSourceUncompilable, err)
	}
}
