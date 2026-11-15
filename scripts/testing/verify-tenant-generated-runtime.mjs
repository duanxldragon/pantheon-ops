import assert from 'node:assert/strict';
import { createHash, randomUUID } from 'node:crypto';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { spawnSync } from 'node:child_process';
import { dirname, isAbsolute, join, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const evidence = join(root, '.harness/evidence/2026-11-07-ops-formal-queue-foundation-gaps');
const runID = randomUUID();
const output = join(evidence, 'resume-o4-generator-exports', runID);
const staging = join(output, 'workspace');
const dsn = process.env.PANTHEON_TENANT_GENERATED_TEST_DSN || process.env.PANTHEON_TEST_DSN;
const redisAddr = process.env.PANTHEON_TENANT_GENERATED_TEST_REDIS_ADDR;
const redisPassword = process.env.PANTHEON_TENANT_GENERATED_TEST_REDIS_PASSWORD;
const receipt = { runID, startedUTC: new Date().toISOString(), staging, output, commands: [], exports: [],
  authorization: 'Bounded fixture read port; real Redis token middleware and emitted handlers. Auth/IAM freshness is verified by its owning lane.',
  requiredMySQL: true, requiredRedis: true, success: false };
mkdirSync(output, { recursive: true });

const sourcePaths = [
  'backend/pkg/contracts/module.go', 'backend/pkg/contracts/module_test.go', 'backend/pkg/contracts/module_required_test.go',
  'backend/internal/scaffold/contract.go', 'backend/internal/scaffold/types.go', 'backend/internal/scaffold/workspace_test.go',
  'frontend/src/modules/lowcode/generator/schema.ts', 'frontend/src/modules/lowcode/generator/backendGenerator.ts',
  'frontend/src/modules/lowcode/generator/exporter.ts', 'frontend/src/modules/lowcode/generator/pages/ModuleWizard.tsx',
  'frontend/src/modules/lowcode/generator/frontendGenerator.ts', 'frontend/src/modules/lowcode/generator/typeMapping.ts',
  'frontend/scripts/export-generated-module.mjs', 'frontend/tests/generator/generator-quality-contract.test.ts',
  'frontend/tests/generator/fixtures/tenantv2-flat.json', 'frontend/tests/generator/fixtures/legacyv1-flat.json',
  'scripts/testing/verify-tenant-generated-runtime.mjs',
  'backend/pkg/contracts/business_tenant_groups.go', 'backend/pkg/contracts/businessauth/businessauth.go',
  'backend/internal/middleware/business_tenant_context.go', 'backend/internal/middleware/business_token_middleware.go',
  'backend/internal/middleware/business_authorization.go', 'backend/pkg/database/business_policy.go',
];
const sourceHashes = () => sourcePaths.map((path) => ({path,sha256:existsSync(join(root,path))?hash(readFileSync(join(root,path))):null}));
receipt.sourceBefore = sourceHashes();

function hash(content) { return createHash('sha256').update(content).digest('hex'); }
function redact(text) {
  let safe = String(text ?? '');
  for (const value of [dsn, redisPassword]) if (value) safe = safe.replaceAll(value, '[REDACTED]');
  return safe;
}
function command(name, executable, args, cwd, extraEnv = {}) {
  const startedUTC = new Date().toISOString();
  const result = spawnSync(executable, args, { cwd, encoding: 'utf8', timeout: 600_000,
    env: (() => {
      const env = { ...process.env, ...extraEnv };
      if (process.platform === 'win32' && extraEnv.PATH) {
        for (const key of Object.keys(env)) if (key.toUpperCase() === 'PATH' && key !== 'PATH') delete env[key];
      }
      return env;
    })() });
  const log = redact(`${result.stdout || ''}${result.stderr || ''}${result.error?.message || ''}`);
  const logPath = join(output, `${name}.txt`);
  writeFileSync(logPath, log);
  receipt.commands.push({ name, executable, args, cwd, startedUTC, finishedUTC: new Date().toISOString(),
    exitCode: result.status, logPath, logSHA256: hash(log) });
  assert.equal(result.status, 0, `${name} failed; see ${logPath}`);
  return result.stdout;
}

const tenantRuntime = String.raw`package item

import (
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "net/http"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "sync"
 "testing"
 "time"

 "github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
 "github.com/duanxldragon/pantheon-base/backend/pkg/common"
 "github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
 "github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
 "github.com/duanxldragon/pantheon-base/backend/pkg/database"
 "github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
 "github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
 "github.com/gin-gonic/gin"
 "github.com/redis/go-redis/v9"
 "gorm.io/gorm"
 "gorm.io/gorm/logger"
)

type fixtureMode struct{}
func (fixtureMode) ReadMode(ctx context.Context) (string,error) { return tenant.ModeMulti,ctx.Err() }
type fixtureAuthorization struct{}
func (fixtureAuthorization) RefreshPrincipal(ctx context.Context, identity businessauth.Identity) (businessauth.Principal,error) {
 if ctx.Err()!=nil || identity.ActorID==0 { return businessauth.Principal{},businessauth.ErrDenied }
 return businessauth.Principal{Identity:identity},nil
}
func (fixtureAuthorization) Authorize(ctx context.Context, input businessauth.Input) (businessauth.Authorization,error) {
 if ctx.Err()!=nil || input.Identity.ActorID==0 || input.ResourceTenantID==0 || input.ResourceTenantID!=input.Identity.ClaimTenantID || input.ResourceTenantID!=input.Resolution.TenantID { return businessauth.Authorization{},businessauth.ErrDenied }
 return businessauth.Authorization{DataScope:&common.DataScopeReq{UserID:input.Identity.ActorID,Mode:common.DataScopeModeAll}},nil
}

var runtimeRedis *redis.Client
func TestMain(m *testing.M) {
 gin.SetMode(gin.TestMode)
 runtimeRedis=redis.NewClient(&redis.Options{Addr:os.Getenv("PANTHEON_TENANT_GENERATED_TEST_REDIS_ADDR"),Password:os.Getenv("PANTHEON_TENANT_GENERATED_TEST_REDIS_PASSWORD")})
 database.RDB=runtimeRedis
 if err:=runtimeRedis.Ping(context.Background()).Err();err!=nil { fmt.Fprintln(os.Stderr,"required Redis unavailable");os.Exit(1) }
 if err:=contracts.RegisterBusinessTenantModeReader(fixtureMode{});err!=nil { panic(err) }
 if err:=contracts.RegisterBusinessAuthorizationReader(fixtureAuthorization{});err!=nil { panic(err) }
 if err:=contracts.SealBusinessTenantProviders();err!=nil { panic(err) }
 code:=m.Run();_ = runtimeRedis.Close();os.Exit(code)
}

func runtimeDB(t *testing.T) *gorm.DB { t.Helper();return testmysql.Open(t).Session(&gorm.Session{Logger:logger.Default.LogMode(logger.Silent)}) }
func schemaReceipt(t *testing.T,db *gorm.DB,mode string) {
 t.Helper();var columns []struct {ColumnName string;ColumnType string;IsNullable string;ColumnDefault *string}
 if err:=db.Raw("SELECT COLUMN_NAME,COLUMN_TYPE,IS_NULLABLE,COLUMN_DEFAULT FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? ORDER BY ORDINAL_POSITION","biz_tenantfixture_item").Scan(&columns).Error;err!=nil {t.Fatal(err)}
 valid:=false;for _,column:=range columns {if column.ColumnName=="tenant_id" {valid=strings.Contains(column.ColumnType,"unsigned") && column.IsNullable=="NO" && column.ColumnDefault!=nil && *column.ColumnDefault=="0"}}
 if !valid {t.Fatal("real MySQL tenant column shape incorrect")}
 var indexes []struct {IndexName string;ColumnName string;SeqInIndex int;NonUnique int}
 if err:=db.Raw("SELECT INDEX_NAME,COLUMN_NAME,SEQ_IN_INDEX,NON_UNIQUE FROM INFORMATION_SCHEMA.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? ORDER BY INDEX_NAME,SEQ_IN_INDEX","biz_tenantfixture_item").Scan(&indexes).Error;err!=nil {t.Fatal(err)}
 shape:=struct{Columns any;Indexes any}{columns,indexes};data,err:=json.MarshalIndent(shape,"","  ");if err!=nil {t.Fatal(err)}
 name:="default";if mode!="" {name=mode}
 if err:=os.WriteFile(filepath.Join(os.Getenv("PANTHEON_TENANT_GENERATED_RUNTIME_EVIDENCE_DIR"),"tenant-"+name+"-shape.json"),data,0600);err!=nil {t.Fatal(err)}
}
func startup(t *testing.T, db *gorm.DB, wantPanic bool) *gin.Engine {
 t.Helper();r:=gin.New();panicked:=false
 func(){defer func(){panicked=recover()!=nil}();InitTenantItemModule(r.Group("/api/v1"),db)}()
 if panicked!=wantPanic { t.Fatalf("startup panic=%v want=%v",panicked,wantPanic) }
 if wantPanic && len(r.Routes())!=0 { t.Fatal("failed startup exposed routes") }
 if !wantPanic && len(r.Routes())!=6 { t.Fatalf("generated routes=%d want 6",len(r.Routes())) }
 return r
}
type envelope struct { Code int;Data json.RawMessage }
func call(t *testing.T,r *gin.Engine,method,path,token,body string) envelope {
 t.Helper();req:=httptest.NewRequest(method,"/api/v1/business/tenantfixture/item"+path,strings.NewReader(body))
 req.Header.Set("Authorization","Bearer "+token);req.Header.Set("Content-Type","application/json")
 response:=httptest.NewRecorder();r.ServeHTTP(response,req)
 var result envelope;if err:=json.Unmarshal(response.Body.Bytes(),&result);err!=nil {t.Fatalf("invalid response: %v",err)}
 return result
}
func token(t *testing.T,tid uint64) string {
 t.Helper();tok:=authtoken.NewAccessToken()
 if err:=authtoken.StoreSession(context.Background(),runtimeRedis,tok,&authtoken.SessionData{UserID:7,SessionID:authtoken.NewAccessToken(),TenantID:tid,LastActivityAt:time.Now().Unix()},time.Minute);err!=nil {t.Fatal("store fixture session failed")}
 t.Cleanup(func(){_ = authtoken.DeleteSession(context.Background(),runtimeRedis,tok)});return tok
}
func success(t *testing.T,result envelope) {t.Helper();if result.Code!=common.CodeSuccess {t.Fatalf("response code=%d",result.Code)} }
func denied(t *testing.T,result envelope) {t.Helper();if result.Code==common.CodeSuccess {t.Fatal("foreign/missing scope operation succeeded")} }

func TestGeneratedDefaultDevHostileCRUD(t *testing.T) {
 for _,mode:=range []string{"","false","true"} {t.Run("auto="+mode,func(t *testing.T){
  t.Setenv("PANTHEON_AUTO_MIGRATE",mode);db:=runtimeDB(t)
  if err:=db.AutoMigrate(&tenant.Tenant{},&tenant.Membership{});err!=nil {t.Fatal(err)}
  for _,tid:=range []uint64{101,202} {
   if err:=db.Create(&tenant.Tenant{ID:tid,Code:fmt.Sprintf("fixture-%d",tid),Name:"fixture",Status:tenant.TenantStatusActive}).Error;err!=nil {t.Fatal(err)}
   if err:=db.Create(&tenant.Membership{TenantID:tid,UserID:7,Status:tenant.MembershipActive}).Error;err!=nil {t.Fatal(err)}
  }
  for _,sql:=range []string{
   "CREATE TABLE system_role (id BIGINT PRIMARY KEY, tenant_id BIGINT NOT NULL, status BIGINT NOT NULL, deleted_at DATETIME NULL)",
   "CREATE TABLE system_user_role (user_id BIGINT NOT NULL, role_id BIGINT NOT NULL, tenant_id BIGINT NOT NULL)",
   "INSERT INTO system_role (id,tenant_id,status) VALUES (1,101,1),(2,202,1)",
   "INSERT INTO system_user_role (user_id,role_id,tenant_id) VALUES (7,1,101),(7,2,202)",
  } {if err:=db.Exec(sql).Error;err!=nil {t.Fatal(err)}}
  r:=startup(t,db,false)
  schemaReceipt(t,db,mode)
  a,b:=token(t,101),token(t,202)
  first:=call(t,r,http.MethodPost,"",a,"{\"code\":\"shared\",\"name\":\"alpha\",\"deptId\":11,\"tenantId\":202,\"tenant_id\":202}");success(t,first)
  var ar TenantItemListResp;if err:=json.Unmarshal(first.Data,&ar);err!=nil {t.Fatal(err)}
  second:=call(t,r,http.MethodPost,"",b,"{\"code\":\"shared\",\"name\":\"beta\",\"deptId\":22}");success(t,second)
  var br TenantItemListResp;if err:=json.Unmarshal(second.Data,&br);err!=nil {t.Fatal(err)}
  denied(t,call(t,r,http.MethodPost,"",a,"{\"code\":\"shared\",\"name\":\"duplicate\"}"))
  for _,tc:=range []struct{tok string;id,foreign,tid uint64}{{a,ar.ID,br.ID,101},{b,br.ID,ar.ID,202}} {
   list:=call(t,r,http.MethodGet,"/list?keyword=shared",tc.tok,"");success(t,list)
   var page TenantItemListPageResp;if err:=json.Unmarshal(list.Data,&page);err!=nil {t.Fatal(err)}
   if page.Total!=1 || len(page.Items)!=1 || page.Items[0].ID!=tc.id {t.Fatal("foreign list/count escaped tenancy")}
   options:=call(t,r,http.MethodGet,"/options",tc.tok,"");success(t,options)
   var opts []TenantItemOptionItem;_ = json.Unmarshal(options.Data,&opts)
   if len(opts)!=1 || opts[0].ID!=tc.id {t.Fatal("foreign options escaped tenancy")}
   success(t,call(t,r,http.MethodGet,fmt.Sprintf("/%d",tc.id),tc.tok,""))
   denied(t,call(t,r,http.MethodGet,fmt.Sprintf("/%d",tc.foreign),tc.tok,""))
   denied(t,call(t,r,http.MethodPut,fmt.Sprintf("/%d",tc.foreign),tc.tok,"{\"name\":\"hostile\"}"))
   denied(t,call(t,r,http.MethodDelete,fmt.Sprintf("/%d",tc.foreign),tc.tok,""))
   var stored TenantItem;if err:=db.First(&stored,tc.id).Error;err!=nil {t.Fatal(err)}
   if stored.TenantID!=tc.tid || stored.Name=="hostile" {t.Fatal("client/foreign mutation changed ownership")}
  }
  success(t,call(t,r,http.MethodPut,fmt.Sprintf("/%d",ar.ID),a,"{\"name\":\"updated\",\"tenantId\":202}"))
  var updated TenantItem;_ = db.First(&updated,ar.ID).Error
  if updated.Name!="updated" || updated.TenantID!=101 {t.Fatal("legitimate update or immutable ownership failed")}
  denied(t,call(t,r,http.MethodGet,"/list",authtoken.NewAccessToken(),""))
  startup(t,db,false) // actual generated replay/startup, preserving A/B rows
  var count int64;_ = db.Model(&TenantItem{}).Count(&count).Error;if count!=2 {t.Fatal("startup replay changed rows")}
  success(t,call(t,r,http.MethodDelete,fmt.Sprintf("/%d",ar.ID),a,""))
  success(t,call(t,r,http.MethodGet,fmt.Sprintf("/%d",br.ID),b,""))
 })}
}

func TestGeneratedScopeClonesDataScopeCancellationAndConcurrency(t *testing.T) {
 db:=runtimeDB(t);t.Setenv("PANTHEON_AUTO_MIGRATE","");startup(t,db,false);base:=NewTenantItemService(db)
 all:=&common.DataScopeReq{Mode:common.DataScopeModeAll}
 a:=base.WithTenantContext(&tenant.Context{TenantID:101,Mode:tenant.ModeMulti}).WithDataScope(all)
 b:=base.WithTenantContext(&tenant.Context{TenantID:202,Mode:tenant.ModeMulti}).WithDataScope(all)
 one,err:=a.CreateTenantItem(&TenantItemCreateReq{Code:"one",Name:"a",DeptId:11});if err!=nil {t.Fatal(err)}
 two,err:=b.CreateTenantItem(&TenantItemCreateReq{Code:"one",Name:"b",DeptId:11});if err!=nil {t.Fatal(err)}
 ctx:=&tenant.Context{TenantID:101,Mode:tenant.ModeMulti};owned:=base.WithTenantContext(ctx);ctx.TenantID=202
 if owned.TenantCtx.TenantID!=101 || base.TenantCtx!=nil {t.Fatal("tenant context/receiver mutated")}
 policy:=&common.DataScopeReq{Mode:common.DataScopeModeCustom,DeptIDs:[]uint64{11}}
 restricted:=a.WithDataScope(policy);policy.DeptIDs[0]=22
 if _,err:=restricted.GetTenantItemDetail(one.ID);err!=nil {t.Fatal("DataScope clone was mutated")}
 if _,err:=restricted.GetTenantItemDetail(two.ID);err==nil {t.Fatal("DataScope escaped tenant predicate")}
 if _,err:=a.WithDataScope(&common.DataScopeReq{IsAdmin:true}).GetTenantItemDetail(two.ID);err==nil {t.Fatal("admin DataScope escaped tenant predicate")}
 hidden:=a.WithDataScope(&common.DataScopeReq{Mode:common.DataScopeModeCustom,DeptIDs:[]uint64{22}})
 if _,err:=hidden.GetTenantItemDetail(one.ID);err==nil {t.Fatal("detail missed DataScope")}
 name:="hidden";if _,err:=hidden.UpdateTenantItem(one.ID,&TenantItemUpdateReq{Name:&name});err==nil {t.Fatal("update missed DataScope")}
 if err:=hidden.DeleteTenantItem(one.ID);err==nil {t.Fatal("delete missed DataScope")}
 opts,err:=hidden.ListTenantItemOptions();if err!=nil || len(opts)!=0 {t.Fatal("options missed DataScope")}
 for _,unbound:=range []*TenantItemService{base,a.WithTenantContext(nil)} {
  if _,err:=unbound.ListTenantItems(nil,nil);!errors.Is(err,tenant.ErrTenantContextMissing) {t.Fatal("unbound list allowed")}
  if _,err:=unbound.ListTenantItemOptions();!errors.Is(err,tenant.ErrTenantContextMissing) {t.Fatal("unbound options allowed")}
  if _,err:=unbound.GetTenantItemDetail(one.ID);!errors.Is(err,tenant.ErrTenantContextMissing) {t.Fatal("unbound detail allowed")}
  if _,err:=unbound.CreateTenantItem(nil);!errors.Is(err,tenant.ErrTenantContextMissing) {t.Fatal("unbound create allowed")}
  if _,err:=unbound.UpdateTenantItem(one.ID,nil);!errors.Is(err,tenant.ErrTenantContextMissing) {t.Fatal("unbound update allowed")}
  if err:=unbound.DeleteTenantItem(one.ID);!errors.Is(err,tenant.ErrTenantContextMissing) {t.Fatal("unbound delete allowed")}
 }
 cancelCtx,cancel:=context.WithCancel(context.Background());cancel()
 if _,err:=a.WithContext(cancelCtx).CreateTenantItem(&TenantItemCreateReq{Code:"cancelled",Name:"cancelled"});!errors.Is(err,context.Canceled) {t.Fatalf("request cancellation not retained: %v",err)}
 var wg sync.WaitGroup;failures:=make(chan error,2)
 for _,service:=range []*TenantItemService{a,b} {wg.Add(1);go func(service *TenantItemService){defer wg.Done()
  for i:=0;i<10;i++ {code:=fmt.Sprintf("concurrent-%d",i);item,err:=service.CreateTenantItem(&TenantItemCreateReq{Code:code,Name:code});if err!=nil {failures<-err;return}
   if _,err:=service.GetTenantItemDetail(item.ID);err!=nil {failures<-err;return}
   if err:=service.DeleteTenantItem(item.ID);err!=nil {failures<-err;return}
  }
 }(service)};wg.Wait();close(failures);for err:=range failures {t.Fatal(err)}
 var count int64;_ = db.Model(&TenantItem{}).Count(&count).Error;if count!=2 {t.Fatal("denied/cancelled/concurrent operations changed retained rows")}
}

func TestGeneratedRequiredReadinessFailure(t *testing.T) {
 for _,mode:=range []string{"","true"} {for _,failure:=range []string{"nil-db","cancelled-bootstrap","legacy-schema","signed-tenant","global-unique","seed","nil-redis"} {t.Run("auto="+mode+"/"+failure,func(t *testing.T){
  t.Setenv("PANTHEON_AUTO_MIGRATE",mode)
  if failure=="nil-db" {startup(t,nil,true);return}
  db:=runtimeDB(t)
  switch failure {
  case "cancelled-bootstrap":ctx,cancel:=context.WithCancel(context.Background());cancel();db=db.WithContext(ctx)
  case "legacy-schema":if err:=db.Exec("CREATE TABLE biz_tenantfixture_item (id BIGINT PRIMARY KEY, code VARCHAR(191))").Error;err!=nil {t.Fatal(err)}
  case "signed-tenant":if err:=NewTenantItemService(db).Migrate();err!=nil {t.Fatal(err)};if err:=db.Exec("ALTER TABLE biz_tenantfixture_item MODIFY tenant_id BIGINT NOT NULL DEFAULT 0").Error;err!=nil {t.Fatal(err)}
  case "global-unique":if err:=NewTenantItemService(db).Migrate();err!=nil {t.Fatal(err)};if err:=db.Exec("CREATE UNIQUE INDEX fixture_global_unique ON biz_tenantfixture_item (code)").Error;err!=nil {t.Fatal(err)}
  case "seed":if err:=db.Exec("CREATE TABLE system_menu (unrelated BIGINT)").Error;err!=nil {t.Fatal(err)}
  case "nil-redis":database.RDB=nil;t.Cleanup(func(){database.RDB=runtimeRedis})
  }
  startup(t,db,true)
  if failure=="legacy-schema" && db.Migrator().HasColumn(&TenantItem{},"tenant_id") {t.Fatal("silently converted legacy table")}
 })}}
}

func TestGeneratedHandlerMissingScopeBeforeBinding(t *testing.T) {
 r:=gin.New();handler:=NewTenantItemHandler(NewTenantItemService(nil))
 r.POST("/",handler.CreateTenantItem);r.GET("/",handler.GetTenantItemOptions)
 for _,method:=range []string{http.MethodPost,http.MethodGet} {
  response:=httptest.NewRecorder();req:=httptest.NewRequest(method,"/",strings.NewReader("invalid json"));req.Header.Set("Content-Type","application/json");r.ServeHTTP(response,req)
  var result envelope;_ = json.Unmarshal(response.Body.Bytes(),&result);if result.Code!=common.CodeForbidden {t.Fatalf("missing scope bound/executed request: code=%d",result.Code)}
 }
}
`;

const legacyRuntime = String.raw`package item
import (
 "testing"
 "github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
 "github.com/gin-gonic/gin"
 "gorm.io/gorm"
 "gorm.io/gorm/logger"
)
func TestActualLegacyDefaultDevAndReplay(t *testing.T) {
 gin.SetMode(gin.TestMode)
 for _,mode:=range []string{"","true"} {t.Run("auto="+mode,func(t *testing.T){
  t.Setenv("PANTHEON_AUTO_MIGRATE",mode);db:=testmysql.Open(t).Session(&gorm.Session{Logger:logger.Default.LogMode(logger.Silent)})
  r:=gin.New();InitLegacyItemModule(r.Group("/api/v1"),db);if len(r.Routes())!=6 {t.Fatal("legacy routes changed")}
  if db.Migrator().HasColumn(&LegacyItem{},"tenant_id") {t.Fatal("legacy default opted into tenancy")}
  service:=NewLegacyItemService(db);item,err:=service.CreateLegacyItem(&LegacyItemCreateReq{Code:"global",Name:"legacy"});if err!=nil {t.Fatal(err)}
  if _,err:=service.CreateLegacyItem(&LegacyItemCreateReq{Code:"global",Name:"duplicate"});err==nil {t.Fatal("legacy global unique changed")}
  if _,err:=service.GetLegacyItemDetail(item.ID);err!=nil {t.Fatal(err)}
  opts,err:=service.ListLegacyItemOptions();if err!=nil || len(opts)!=1 {t.Fatal("legacy options changed")}
  r=gin.New();InitLegacyItemModule(r.Group("/api/v1"),db)
  var count int64;_ = db.Model(&LegacyItem{}).Count(&count).Error;if count!=1 {t.Fatal("legacy replay changed data")}
  name:="updated";if _,err:=service.UpdateLegacyItem(item.ID,&LegacyItemUpdateReq{Name:&name});err!=nil {t.Fatal(err)}
  if err:=service.DeleteLegacyItem(item.ID);err!=nil {t.Fatal(err)}
 })}
}
`;

try {
  assert.ok(dsn, 'Required MySQL DSN missing: set PANTHEON_TENANT_GENERATED_TEST_DSN or PANTHEON_TEST_DSN; runtime cannot skip');
  assert.ok(redisAddr, 'Required isolated Redis address missing: set PANTHEON_TENANT_GENERATED_TEST_REDIS_ADDR');
  for (const fixture of ['tenantv2-flat', 'legacyv1-flat', 'tenantv2-flat-no-datascope']) {
    let schemaPath = join(root, 'frontend/tests/generator/fixtures', `${fixture}.json`);
    if (fixture === 'tenantv2-flat-no-datascope') {
      const schema = JSON.parse(readFileSync(join(root, 'frontend/tests/generator/fixtures/tenantv2-flat.json'), 'utf8'));
      schema.name = 'tenantfixture/nodata';
      schema.model.modelName = 'TenantNoDataItem';
      schema.model.tableName = 'biz_tenantfixture_nodata';
      schema.enableDataScope = false;
      // Empty seeds keep this derived variant focused on tenancy without DataScope.
      schema.menus = [];
      schema.permissions = [];
      schema.i18n.translations = {zh:{},en:{}};
      schemaPath = join(output, 'tenantv2-flat-no-datascope-schema.json');
      writeFileSync(schemaPath, JSON.stringify(schema, null, 2) + '\n');
    }
    const schema = JSON.parse(readFileSync(schemaPath, 'utf8'));
    const exported = command(`export-${fixture}`, process.execPath,
      ['scripts/export-generated-module.mjs', schemaPath], join(root, 'frontend'));
    const files = JSON.parse(exported);
    writeFileSync(join(output, `${fixture}-export.json`), exported);
    const entries = [];
    for (const file of files) {
      assert.ok(!isAbsolute(file.path) && !file.path.split('/').includes('..'), 'unsafe emitted path');
      const target = resolve(staging, file.path);
      const rel = relative(staging, target);
      assert.ok(rel !== '..' && !rel.startsWith(`..${sep}`) && !isAbsolute(rel), 'emitted path escaped staging');
      mkdirSync(dirname(target), { recursive: true });
      writeFileSync(target, file.content);
      entries.push({ path: file.path, emittedPath: target, sha256: hash(file.content) });
    }
    const testPath = join(staging, 'backend/modules/business', schema.name, 'generated_runtime_test.go');
    let runtime = fixture === 'legacyv1-flat' ? legacyRuntime : tenantRuntime;
    if (fixture === 'tenantv2-flat-no-datascope') {
      // The actual emitted no-DataScope package runs default/false/dev hostile
      // HTTP CRUD, options, replay, and MySQL schema assertions without changing
      // generated source or borrowing its DataScope-dependent service tests.
      runtime = tenantRuntime.slice(0, tenantRuntime.indexOf('func TestGeneratedScopeClonesDataScopeCancellationAndConcurrency'))
        .replace('package item', 'package nodata')
        .replaceAll('TenantItem', 'TenantNoDataItem')
        .replaceAll('biz_tenantfixture_item', 'biz_tenantfixture_nodata')
        .replaceAll('/tenantfixture/item', '/tenantfixture/nodata')
        .replace('"sync"', '')
        .replace('"errors"', '')
        .replace('"tenant-"+name', '"tenant-no-datascope-"+name');
    }
    writeFileSync(testPath, runtime);
    command(`format-runtime-${fixture}`, 'gofmt', ['-w', testPath], root);
    const unformatted = command(`format-export-${fixture}`, 'gofmt', ['-l', ...entries.filter((entry) => entry.path.endsWith('.go')).map((entry) => entry.emittedPath)], root);
    assert.equal(unformatted.trim(), '', `emitted ${fixture} source is not gofmt stable: ${unformatted}`);
    receipt.exports.push({ fixture: relative(root, schemaPath).replaceAll(sep, '/'), schemaSHA256: hash(readFileSync(schemaPath)),
      exportPath: join(output, `${fixture}-export.json`), exportSHA256: hash(exported), files: entries,
      runtimeTestPath: testPath, runtimeTestSHA256: hash(readFileSync(testPath)) });
  }
  const baseModule = 'github.com/duanxldragon/pantheon-base/backend';
  const fixtureMod = readFileSync(join(root, 'backend/go.mod'), 'utf8')
    .replace(/^module .+$/m, 'module github.com/duanxldragon/pantheon-base/generated-runtime-fixture')
    .concat(`\nrequire ${baseModule} v0.0.0\nreplace ${baseModule} => "${join(root, 'backend').replaceAll('\\', '/')}"\n`);
  writeFileSync(join(staging, 'backend/go.mod'), fixtureMod);
  writeFileSync(join(staging, 'backend/go.sum'), readFileSync(join(root, 'backend/go.sum')));
  const env = { PANTHEON_TEST_DSN: dsn, PANTHEON_TEST_DSN_REQUIRED: '1', CGO_ENABLED: '1',
    PANTHEON_TENANT_GENERATED_RUNTIME_EVIDENCE_DIR: output };
  if (process.platform === 'win32' && existsSync('D:/msys64/mingw64/bin/gcc.exe')) {
    env.PATH = `D:\\msys64\\mingw64\\bin;${process.env.PATH || process.env.Path || ''}`;
    env.CC = 'D:/msys64/mingw64/bin/gcc.exe';
  }
  receipt.compiler = { platform: process.platform, cc: env.CC, cgoEnabled: env.CGO_ENABLED,
    pathPrefix: env.PATH?.split(';')[0] };
  const tests = command('generated-race', 'go', ['test', '-mod=readonly', '-race', '-count=1', '-v', './...'], join(staging, 'backend'), env);
  assert.ok(!tests.includes('--- SKIP:'), 'Required generated runtime must not skip tests');
  command('generated-vet', 'go', ['vet', '-mod=readonly', './...'], join(staging, 'backend'), env);
  command('generated-build', 'go', ['build', '-mod=readonly', './...'], join(staging, 'backend'), env);
  receipt.sourceAfter = sourceHashes();
  assert.deepEqual(receipt.sourceAfter, receipt.sourceBefore, 'Source changed during generated runtime; rerun against stable source');
  receipt.success = true;
} catch (error) {
  receipt.failure = redact(error.message);
  process.exitCode = 1;
} finally {
  receipt.finishedUTC = new Date().toISOString();
  const receiptPath = join(output, 'receipt.json');
  writeFileSync(receiptPath, JSON.stringify(receipt, null, 2) + '\n');
  process.stdout.write(JSON.stringify({ success: receipt.success, receiptPath, failure: receipt.failure }) + '\n');
}
