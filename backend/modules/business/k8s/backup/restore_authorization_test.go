package backup

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/namespace"
	k8spkg "github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/pkg"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

// Only provider IO is replaced. Every client lookup still executes the actual
// shared cluster authorization and credential checks against required MySQL.
type restoreTestClients struct {
	*cluster.ClusterService
	provider *dynamicfake.FakeDynamicClient
}

func (c *restoreTestClients) GetDynamicClientWithContext(ctx context.Context, id uint64, scope *common.DataScopeReq) (dynamic.Interface, error) {
	if _, err := c.ClusterService.GetDynamicClientWithContext(ctx, id, scope); err != nil {
		return nil, err
	}
	return c.provider, nil
}

func TestCreateRestoreRejectsBeforePersistence(t *testing.T) {
	for _, reason := range []string{"missing_credential", "canceled", "expired", "revoked_source", "revoked_credential"} {
		t.Run(reason, func(t *testing.T) {
			f := newFinalRestoreFixture(t)
			ctx := context.Background()
			want := "k8s.cluster.credential_not_found"
			var cancel context.CancelFunc
			switch reason {
			case "missing_credential":
				if err := f.db.Delete(&f.ref).Error; err != nil {
					t.Fatal(err)
				}
			case "revoked_credential":
				if err := f.db.Model(&f.ref).Update("status", "revoked").Error; err != nil {
					t.Fatal(err)
				}
			case "revoked_source":
				if err := f.db.Delete(&f.source).Error; err != nil {
					t.Fatal(err)
				}
				want = "backup.not_found"
			case "canceled":
				ctx, cancel = context.WithCancel(ctx)
				cancel()
				want = context.Canceled.Error()
			case "expired":
				ctx, cancel = context.WithDeadline(ctx, time.Now().Add(-time.Second))
				defer cancel()
				want = context.DeadlineExceeded.Error()
			}
			resp, err := f.svc.CreateRestore(ctx, f.req, f.scope, "owner")
			if err == nil || resp != nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("want %s, got %v", want, err)
			}
			var count int64
			if err := f.db.Model(&VeleroRestore{}).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 || len(f.provider.Actions()) != 0 {
				t.Fatal("denial caused persistence or provider IO")
			}
		})
	}
}

func TestCreateRestoreEligibleSourceAndEnqueue(t *testing.T) {
	t.Setenv("PANTHEON_K8S_KUBECONFIG_KEY", strings.Repeat("12", 32))
	db := testmysql.Open(t).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	cs := cluster.NewClusterService(db)
	if err := cs.Migrate(); err != nil {
		t.Fatal(err)
	}
	ns := namespace.NewNamespaceService(cs, db)
	if err := ns.Migrate(); err != nil {
		t.Fatal(err)
	}
	svc := NewBackupService(db, cs, ns)
	if err := svc.Migrate(); err != nil {
		t.Fatal(err)
	}
	encrypted, err := k8spkg.EncryptKubeconfig("apiVersion: v1\nkind: Config\nclusters:\n- name: test\n  cluster:\n    server: https://115.190.251.143:6443\n    tls-server-name: kubernetes\ncontexts:\n- name: test\n  context:\n    cluster: test\n    user: empty\ncurrent-context: test\nusers:\n- name: empty\n  user: {}\n")
	if err != nil {
		t.Fatal(err)
	}
	target := cluster.Cluster{Code: "restore-enqueue", Name: "restore-enqueue", DeptID: 10, BusinessScopeID: 7, Environment: "test", KubeconfigEncrypted: encrypted}
	if err := db.Create(&target).Error; err != nil {
		t.Fatal(err)
	}
	source := VeleroBackup{Name: "source", ClusterID: target.ID, Status: StatusCompleted, Phase: StatusCompleted}
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	binding := namespace.NamespaceBinding{ClusterID: target.ID, Namespace: "owned-target", BusinessScopeID: 7, Environment: "test", AllowedActions: `["backup:restore"]`}
	if err := db.Create(&binding).Error; err != nil {
		t.Fatal(err)
	}
	owner := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 10}
	req := CreateRestoreRequest{Name: "owned-restore", ClusterID: target.ID, BackupName: source.Name, NamespaceMapping: map[string]string{"owned-source": "owned-target"}, IncludedNamespaces: []string{"owned-source"}, IncludedResources: []string{"configmaps"}}
	newProvider := func(phase, expiration string) *dynamicfake.FakeDynamicClient {
		obj := &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": veleroAPIVersion, "kind": "Backup", "metadata": map[string]interface{}{"name": source.Name, "namespace": veleroNamespace}, "status": map[string]interface{}{"phase": phase, "expiration": expiration}}}
		p := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), obj)
		svc.clusterService = &restoreTestClients{ClusterService: cs, provider: p}
		return p
	}
	assertNoRow := func(t *testing.T) {
		t.Helper()
		var count int64
		if err := db.Model(&VeleroRestore{}).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("rejected request persisted %d rows", count)
		}
	}
	t.Run("revoked registry source with authorized cached client", func(t *testing.T) {
		p := newProvider(StatusCompleted, "")
		if _, err := cs.GetClientsetWithContext(context.Background(), target.ID, owner); err != nil {
			t.Fatal(err)
		}
		if err := db.Delete(&source).Error; err != nil {
			t.Fatal(err)
		}
		resp, err := svc.CreateRestore(context.Background(), req, owner, "test")
		if err == nil || resp != nil {
			t.Error("revoked registry source accepted")
		}
		assertNoRow(t)
		if len(p.Actions()) != 0 {
			t.Error("revoked source called provider")
		}
		if err := db.Unscoped().Model(&source).Update("deleted_at", nil).Error; err != nil {
			t.Fatal(err)
		}
		source.DeletedAt = gorm.DeletedAt{}
	})
	for _, status := range []string{StatusNew, StatusInProgress, StatusFailed} {
		t.Run("registry_"+status, func(t *testing.T) {
			p := newProvider(StatusCompleted, "")
			if err := db.Model(&source).Update("status", status).Error; err != nil {
				t.Fatal(err)
			}
			resp, err := svc.CreateRestore(context.Background(), req, owner, "test")
			if err == nil || resp != nil {
				t.Error("ineligible source accepted")
			}
			assertNoRow(t)
			if len(p.Actions()) != 0 {
				t.Error("ineligible registry called provider")
			}
			if err := db.Model(&source).Update("status", StatusCompleted).Error; err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Run("expired registry", func(t *testing.T) {
		p := newProvider(StatusCompleted, "")
		if err := db.Model(&source).Update("expiration", time.Now().Add(-time.Hour)).Error; err != nil {
			t.Fatal(err)
		}
		_, err := svc.CreateRestore(context.Background(), req, owner, "test")
		if err == nil {
			t.Error("expired source accepted")
		}
		assertNoRow(t)
		if len(p.Actions()) != 0 {
			t.Error("expired registry called provider")
		}
		if err := db.Model(&source).Update("expiration", nil).Error; err != nil {
			t.Fatal(err)
		}
	})
	for _, phase := range []string{"", StatusNew, StatusInProgress, StatusFailed, "PartiallyFailed", "Deleting"} {
		t.Run("live_"+phase, func(t *testing.T) {
			p := newProvider(phase, "")
			resp, err := svc.CreateRestore(context.Background(), req, owner, "test")
			if err == nil || resp != nil {
				t.Error("live ineligible source accepted")
			}
			assertNoRow(t)
			for _, a := range p.Actions() {
				if a.GetVerb() != "get" {
					t.Error("ineligible source mutated provider")
				}
			}
		})
	}
	t.Run("deleted live source", func(t *testing.T) {
		p := newProvider(StatusCompleted, "")
		if err := p.Resource(backupGVR).Namespace(veleroNamespace).Delete(context.Background(), source.Name, metav1.DeleteOptions{}); err != nil {
			t.Fatal(err)
		}
		_, err := svc.CreateRestore(context.Background(), req, owner, "test")
		if err == nil {
			t.Error("deleted provider source accepted")
		}
		assertNoRow(t)
	})
	for _, expiration := range []string{time.Now().Add(-time.Hour).UTC().Format(time.RFC3339), "malformed"} {
		t.Run("live expiration_"+expiration, func(t *testing.T) {
			newProvider(StatusCompleted, expiration)
			_, err := svc.CreateRestore(context.Background(), req, owner, "test")
			if err == nil {
				t.Error("invalid live expiration accepted")
			}
			assertNoRow(t)
		})
	}
	t.Run("cancel during source inspection", func(t *testing.T) {
		p := newProvider(StatusCompleted, "")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		p.PrependReactor("get", "backups", func(k8stesting.Action) (bool, runtime.Object, error) { cancel(); return false, nil, nil })
		resp, err := svc.CreateRestore(ctx, req, owner, "test")
		if !errors.Is(err, context.Canceled) || resp != nil {
			t.Errorf("cancellation not preserved: %v", err)
		}
		assertNoRow(t)
	})
	for _, field := range []string{"department", "credential", "source"} {
		t.Run("revoked during live inspection_"+field, func(t *testing.T) {
			p := newProvider(StatusCompleted, "")
			p.PrependReactor("get", "backups", func(k8stesting.Action) (bool, runtime.Object, error) {
				var err error
				switch field {
				case "department":
					err = db.Model(&target).Update("dept_id", 20).Error
				case "credential":
					err = db.Model(&target).Update("kubeconfig_encrypted", "").Error
				case "source":
					err = db.Model(&source).Update("status", StatusFailed).Error
				}
				if err != nil {
					t.Fatal(err)
				}
				return false, nil, nil
			})
			resp, err := svc.CreateRestore(context.Background(), req, owner, "test")
			if err == nil || resp != nil {
				t.Error("revoked request accepted after provider inspection")
			}
			assertNoRow(t)
			for _, a := range p.Actions() {
				if a.GetVerb() != "get" {
					t.Error("revoked request mutated provider")
				}
			}
			if err := db.Model(&target).Updates(map[string]interface{}{"dept_id": 10, "kubeconfig_encrypted": encrypted}).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&source).Update("status", StatusCompleted).Error; err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Run("worker rechecks live source", func(t *testing.T) {
		p := newProvider(StatusCompleted, "")
		gets := 0
		p.PrependReactor("get", "backups", func(k8stesting.Action) (bool, runtime.Object, error) {
			gets++
			if gets == 1 {
				return false, nil, nil
			}
			return true, &unstructured.Unstructured{Object: map[string]interface{}{"status": map[string]interface{}{"phase": StatusFailed}}}, nil
		})
		resp, err := svc.CreateRestore(context.Background(), req, owner, "test")
		if err != nil {
			t.Fatal(err)
		}
		until := time.Now().Add(3 * time.Second)
		var record VeleroRestore
		for time.Now().Before(until) {
			if err := db.First(&record, resp.ID).Error; err != nil {
				t.Fatal(err)
			}
			if record.Status == StatusFailed {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if record.Status != StatusFailed {
			t.Fatal("worker failed to reject revoked source")
		}
		for _, a := range p.Actions() {
			if a.GetVerb() != "get" {
				t.Error("worker created restore from failed source")
			}
		}
		if err := db.Unscoped().Delete(&record).Error; err != nil {
			t.Fatal(err)
		}
	})
	t.Run("authorized successful enqueue", func(t *testing.T) {
		p := newProvider(StatusCompleted, time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
		callbackName := "restore_auth_assert_locks"
		if err := db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
			if tx.Statement.Table != (VeleroRestore{}).TableName() {
				return
			}
			testmysql.AssertRowLocked(t, db, (cluster.Cluster{}).TableName(), target.ID)
			testmysql.AssertRowLocked(t, db, (VeleroBackup{}).TableName(), source.ID)
		}); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = db.Callback().Create().Remove(callbackName) }()
		created := make(chan *unstructured.Unstructured, 1)
		p.PrependReactor("create", "restores", func(a k8stesting.Action) (bool, runtime.Object, error) {
			obj := a.(k8stesting.CreateAction).GetObject().(*unstructured.Unstructured)
			created <- obj.DeepCopy()
			return true, obj, nil
		})
		resp, err := svc.CreateRestore(context.Background(), req, owner, "test")
		if err != nil {
			t.Fatal(err)
		}
		if resp.Status != StatusNew {
			t.Fatalf("expected New, got %s", resp.Status)
		}
		select {
		case obj := <-created:
			if obj.GetName() != req.Name || obj.GetNamespace() != veleroNamespace {
				t.Error("wrong restore target")
			}
			spec, _ := nestedMap(obj.Object, "spec")
			if nestedString(spec, "backupName") != source.Name {
				t.Error("wrong source")
			}
			mapping, _ := nestedMap(spec, "namespaceMapping")
			if nestedString(mapping, "owned-source") != "owned-target" {
				t.Error("namespace mapping lost")
			}
		case <-time.After(3 * time.Second):
			t.Fatal("authorized request never submitted")
		}
		var record VeleroRestore
		if err := db.First(&record, resp.ID).Error; err != nil {
			t.Fatal(err)
		}
		if record.ClusterID != target.ID || record.CreatedBy != "test" {
			t.Error("wrong persisted owner")
		}
	})
}
