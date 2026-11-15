package backup

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/namespace"
	k8spkg "github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/pkg"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

type finalRestoreFixture struct {
	db       *gorm.DB
	svc      *BackupService
	cs       *cluster.ClusterService
	target   cluster.Cluster
	ref      cluster.ClusterCredentialRef
	source   VeleroBackup
	provider *dynamicfake.FakeDynamicClient
	scope    *common.DataScopeReq
	req      CreateRestoreRequest
}

// Observe return from the worker's final lock path, including a preexisting
// terminal attempt. A sleep or an already-Failed row cannot prove worker exit.
type finalLockObserver struct {
	backupClusterClients
	calls atomic.Int32
	done  chan error
}

func (c *finalLockObserver) WithClusterLockContext(ctx context.Context, id uint64, scope *common.DataScopeReq, action func(*gorm.DB, *cluster.Cluster) error) error {
	call := c.calls.Add(1)
	err := c.backupClusterClients.WithClusterLockContext(ctx, id, scope, action)
	if call == 2 {
		c.done <- err
	}
	return err
}

func newFinalRestoreFixture(t *testing.T) *finalRestoreFixture {
	t.Helper()
	t.Setenv("PANTHEON_K8S_KUBECONFIG_KEY", strings.Repeat("12", 32))
	f := &finalRestoreFixture{db: testmysql.Open(t).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)}), scope: &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 10}}
	f.cs = cluster.NewClusterService(f.db)
	if err := f.cs.Migrate(); err != nil {
		t.Fatal(err)
	}
	ns := namespace.NewNamespaceService(f.cs, f.db)
	if err := ns.Migrate(); err != nil {
		t.Fatal(err)
	}
	f.svc = NewBackupService(f.db, f.cs, ns)
	if err := f.svc.Migrate(); err != nil {
		t.Fatal(err)
	}
	encrypted, err := k8spkg.EncryptKubeconfig("apiVersion: v1\nkind: Config\nclusters:\n- name: test\n  cluster:\n    server: https://115.190.251.143:6443\ncontexts:\n- name: test\n  context:\n    cluster: test\n    user: empty\ncurrent-context: test\nusers:\n- name: empty\n  user: {}\n")
	if err != nil {
		t.Fatal(err)
	}
	f.target = cluster.Cluster{Code: "final-restore", Name: "final-restore", DeptID: 10, BusinessScopeID: 7, Environment: "test", KubeconfigEncrypted: encrypted}
	if err := f.db.Create(&f.target).Error; err != nil {
		t.Fatal(err)
	}
	f.ref = cluster.ClusterCredentialRef{ClusterID: f.target.ID, Encrypted: encrypted, Status: "active", Version: 1}
	if err := f.db.Create(&f.ref).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Model(&f.target).Update("kubeconfig_credential_ref_id", f.ref.ID).Error; err != nil {
		t.Fatal(err)
	}
	f.source = VeleroBackup{Name: "source", ClusterID: f.target.ID, Status: StatusCompleted, Phase: StatusCompleted}
	if err := f.db.Create(&f.source).Error; err != nil {
		t.Fatal(err)
	}
	binding := namespace.NamespaceBinding{ClusterID: f.target.ID, Namespace: "owned-target", BusinessScopeID: 7, Environment: "test", AllowedActions: `["backup:restore"]`}
	if err := f.db.Create(&binding).Error; err != nil {
		t.Fatal(err)
	}
	obj := &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": veleroAPIVersion, "kind": "Backup", "metadata": map[string]interface{}{"name": f.source.Name, "namespace": veleroNamespace}, "status": map[string]interface{}{"phase": StatusCompleted}}}
	f.provider = dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{backupGVR: "BackupList", restoreGVR: "RestoreList"}, obj)
	f.svc.clusterService = &restoreTestClients{ClusterService: f.cs, provider: f.provider}
	f.req = CreateRestoreRequest{Name: "final-restore", ClusterID: f.target.ID, BackupName: f.source.Name, IncludedNamespaces: []string{"owned-source"}, NamespaceMapping: map[string]string{"owned-source": "owned-target"}, IncludedResources: []string{"configmaps"}}
	if _, err := f.cs.GetClientsetWithContext(context.Background(), f.target.ID, f.scope); err != nil {
		t.Fatal(err)
	}
	if _, err := f.cs.GetDynamicClientWithContext(context.Background(), f.target.ID, f.scope); err != nil {
		t.Fatal(err)
	}
	return f
}

func awaitFinalRestore(t *testing.T, f *finalRestoreFixture, id uint64, status string) VeleroRestore {
	t.Helper()
	var row VeleroRestore
	until := time.Now().Add(5 * time.Second)
	for time.Now().Before(until) {
		if err := f.db.First(&row, id).Error; err != nil {
			t.Fatal(err)
		}
		if row.Status == status {
			return row
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("want %s, got %s", status, row.Status)
	return row
}

func assertFinalDenied(t *testing.T, f *finalRestoreFixture, want string) {
	t.Helper()
	resp, err := f.svc.CreateRestore(context.Background(), f.req, f.scope, "owner")
	if err == nil || resp != nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("want denial %q, got response=%v error=%v", want, resp, err)
	}
	var count int64
	if err := f.db.Model(&VeleroRestore{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("denial persisted %d records", count)
	}
	for _, a := range f.provider.Actions() {
		if a.GetVerb() != "get" {
			t.Fatalf("denial mutated provider: %s", a.GetVerb())
		}
	}
}

func TestRestoreFinalIndependentDenials(t *testing.T) {
	for _, reason := range []string{"foreign_department", "department_revoked_cached", "foreign_source"} {
		t.Run(reason, func(t *testing.T) {
			f := newFinalRestoreFixture(t)
			switch reason {
			case "foreign_department":
				f.scope.DeptID = 20
			case "department_revoked_cached":
				if err := f.db.Model(&f.target).Update("dept_id", 20).Error; err != nil {
					t.Fatal(err)
				}
			case "foreign_source":
				other := cluster.Cluster{Code: "valid-second", Name: "valid-second", DeptID: 10, KubeconfigEncrypted: f.target.KubeconfigEncrypted}
				if err := f.db.Create(&other).Error; err != nil {
					t.Fatal(err)
				}
				if _, err := f.cs.GetClientsetWithContext(context.Background(), other.ID, f.scope); err != nil {
					t.Fatal(err)
				}
				if _, err := f.cs.GetDynamicClientWithContext(context.Background(), other.ID, f.scope); err != nil {
					t.Fatal(err)
				}
				f.req.ClusterID = other.ID
			}
			want := "k8s.cluster.not_found"
			if reason == "foreign_source" {
				want = "backup.not_found"
			}
			assertFinalDenied(t, f, want)
			if len(f.provider.Actions()) != 0 {
				t.Fatal("authorization denial called provider")
			}
		})
	}
}

func TestRestoreFinalSecondGETRevocation(t *testing.T) {
	for _, revoke := range []string{"department", "credential", "source", "binding", "attempt"} {
		t.Run(revoke, func(t *testing.T) {
			f := newFinalRestoreFixture(t)
			entered, release, created := make(chan struct{}), make(chan struct{}), make(chan struct{}, 1)
			observer := &finalLockObserver{backupClusterClients: f.svc.clusterService, done: make(chan error, 1)}
			f.svc.clusterService = observer
			var gets atomic.Int32
			f.provider.PrependReactor("get", "backups", func(k8stesting.Action) (bool, runtime.Object, error) {
				if gets.Add(1) == 2 {
					close(entered)
					<-release
				}
				return false, nil, nil
			})
			f.provider.PrependReactor("create", "restores", func(a k8stesting.Action) (bool, runtime.Object, error) {
				created <- struct{}{}
				return true, a.(k8stesting.CreateAction).GetObject(), nil
			})
			resp, err := f.svc.CreateRestore(context.Background(), f.req, f.scope, "owner")
			if err != nil {
				t.Fatal(err)
			}
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				close(release)
				t.Fatal("second GET never blocked")
			}
			switch revoke {
			case "department":
				err = f.db.Model(&f.target).Update("dept_id", 20).Error
			case "credential":
				err = f.db.Model(&f.ref).Update("status", "revoked").Error
			case "source":
				err = f.db.Model(&f.source).Update("status", StatusFailed).Error
			case "binding":
				err = f.db.Model(&namespace.NamespaceBinding{}).Where("cluster_id = ?", f.target.ID).Update("allowed_actions", "[]").Error
			case "attempt":
				err = f.db.Model(&VeleroRestore{}).Where("id = ?", resp.ID).Update("status", StatusFailed).Error
			}
			close(release)
			if err != nil {
				t.Fatal(err)
			}
			if revoke == "attempt" {
				select {
				case err := <-observer.done:
					if err == nil {
						t.Error("worker final boundary did not refuse revoked state")
					}
				case <-created:
					t.Fatal("revoked request submitted Restore after second GET")
				case <-time.After(5 * time.Second):
					t.Fatal("worker did not finish final boundary")
				}
			} else {
				// A newly written Failed state proves the worker reached its final
				// status write, including refusal before entering the lock callback.
				awaitFinalRestore(t, f, resp.ID, StatusFailed)
			}
			select {
			case <-created:
				t.Fatal("revoked request submitted Restore after second GET")
			default:
			}
			awaitFinalRestore(t, f, resp.ID, StatusFailed)
		})
	}
}

func TestRestoreFinalTargetDenials(t *testing.T) {
	for _, reason := range []string{"unbound_remap", "unmapped_source", "no_action", "cross_scope", "wrong_environment", "wildcard", "all_namespaces", "pv_cluster_resource"} {
		t.Run(reason, func(t *testing.T) {
			f := newFinalRestoreFixture(t)
			want := "k8s.namespace.binding_required"
			switch reason {
			case "unbound_remap":
				f.req.NamespaceMapping["owned-source"] = "foreign-target"
			case "unmapped_source":
				f.req.NamespaceMapping = nil
			case "no_action":
				if err := f.db.Model(&namespace.NamespaceBinding{}).Where("cluster_id = ?", f.target.ID).Update("allowed_actions", "[]").Error; err != nil {
					t.Fatal(err)
				}
				want = "k8s.namespace.action_forbidden"
			case "cross_scope":
				if err := f.db.Model(&namespace.NamespaceBinding{}).Where("cluster_id = ?", f.target.ID).Update("business_scope_id", 8).Error; err != nil {
					t.Fatal(err)
				}
			case "wrong_environment":
				if err := f.db.Model(&namespace.NamespaceBinding{}).Where("cluster_id = ?", f.target.ID).Update("environment", "other").Error; err != nil {
					t.Fatal(err)
				}
			case "wildcard":
				f.req.IncludedNamespaces = []string{"*"}
			case "all_namespaces":
				f.req.IncludedNamespaces = nil
			case "pv_cluster_resource":
				f.req.RestorePVs = true
				want = "k8s.backup.restore_create_failed"
			}
			assertFinalDenied(t, f, want)
		})
	}
}

func TestRestoreFinalAcceptedResponseTimeout(t *testing.T) {
	f := newFinalRestoreFixture(t)
	var creates atomic.Int32
	f.provider.PrependReactor("create", "restores", func(a k8stesting.Action) (bool, runtime.Object, error) {
		creates.Add(1)
		obj := a.(k8stesting.CreateAction).GetObject().(*unstructured.Unstructured).DeepCopy()
		obj.Object["status"] = map[string]interface{}{"phase": StatusCompleted}
		if err := f.provider.Tracker().Create(restoreGVR, obj, veleroNamespace); err != nil {
			return true, nil, err
		}
		return true, nil, context.DeadlineExceeded
	})
	resp, err := f.svc.CreateRestore(context.Background(), f.req, f.scope, "owner")
	if err != nil {
		t.Fatal(err)
	}
	awaitFinalRestore(t, f, resp.ID, "Unknown")
	// A new reconciler represents process restart: only DB and provider survive.
	r := NewReconciler(f.db, f.cs, time.Second)
	r.clusterService = f.svc.clusterService
	if err := r.reconcileRestores(context.Background()); err != nil {
		t.Fatal(err)
	}
	awaitFinalRestore(t, f, resp.ID, StatusCompleted)
	obj, err := f.provider.Resource(restoreGVR).Namespace(veleroNamespace).Get(context.Background(), resp.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if obj.GetAnnotations()["pantheon.io/velero-record-id"] == "" {
		t.Error("provider identity missing")
	}
	clusterResources, _, err := unstructured.NestedBool(obj.Object, "spec", "includeClusterResources")
	if err != nil || clusterResources {
		t.Fatal("restore includes unauthorized cluster resources")
	}
	if creates.Load() != 1 {
		t.Fatal("submission retried")
	}
}

func TestFinalRestoreLocksSingleCreate(t *testing.T) {
	f := newFinalRestoreFixture(t)
	var creates atomic.Int32
	f.provider.PrependReactor("create", "restores", func(a k8stesting.Action) (bool, runtime.Object, error) {
		creates.Add(1)
		obj := a.(k8stesting.CreateAction).GetObject().(*unstructured.Unstructured)
		var row VeleroRestore
		if err := f.db.Where("name = ?", f.req.Name).First(&row).Error; err != nil {
			return true, nil, err
		}
		var binding namespace.NamespaceBinding
		if err := f.db.Where("cluster_id = ?", f.target.ID).First(&binding).Error; err != nil {
			return true, nil, err
		}
		testmysql.AssertRowLocked(t, f.db, (cluster.Cluster{}).TableName(), f.target.ID)
		testmysql.AssertRowLocked(t, f.db, (cluster.ClusterCredentialRef{}).TableName(), f.ref.ID)
		testmysql.AssertRowLocked(t, f.db, (VeleroBackup{}).TableName(), f.source.ID)
		testmysql.AssertRowLocked(t, f.db, (VeleroRestore{}).TableName(), row.ID)
		testmysql.AssertRowLocked(t, f.db, (namespace.NamespaceBinding{}).TableName(), binding.ID)
		return true, obj, nil
	})
	resp, err := f.svc.CreateRestore(context.Background(), f.req, f.scope, "owner")
	if err != nil {
		t.Fatal(err)
	}
	awaitFinalRestore(t, f, resp.ID, StatusInProgress)
	if creates.Load() != 1 {
		t.Fatal("restore submitted more than once")
	}
}

func TestFinalRestoreRecordNamespaceOnly(t *testing.T) {
	f := newFinalRestoreFixture(t)
	resp, err := f.svc.CreateRestore(context.Background(), f.req, f.scope, "owner")
	if err != nil {
		t.Fatal(err)
	}
	row := awaitFinalRestore(t, f, resp.ID, StatusInProgress)
	if resp.RestorePVs || row.RestorePVs {
		t.Fatal("database default re-enabled PV restore in a namespace-only intent")
	}
}

func TestFinalRestoreLiveExpirationWhileWaitingForLock(t *testing.T) {
	f := newFinalRestoreFixture(t)
	entered, releaseGET := make(chan struct{}), make(chan struct{})
	var gets atomic.Int32
	f.provider.PrependReactor("get", "backups", func(k8stesting.Action) (bool, runtime.Object, error) {
		if gets.Add(1) != 2 {
			return false, nil, nil
		}
		close(entered)
		<-releaseGET
		return true, &unstructured.Unstructured{Object: map[string]interface{}{"status": map[string]interface{}{"phase": StatusCompleted, "expiration": time.Now().Add(1500 * time.Millisecond).UTC().Format(time.RFC3339Nano)}}}, nil
	})
	resp, err := f.svc.CreateRestore(context.Background(), f.req, f.scope, "owner")
	if err != nil {
		close(releaseGET)
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		close(releaseGET)
		t.Fatal("second GET never blocked")
	}
	locked, releaseLock, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- f.cs.WithClusterLockContext(context.Background(), f.target.ID, f.scope, func(*gorm.DB, *cluster.Cluster) error { close(locked); <-releaseLock; return nil })
	}()
	select {
	case <-locked:
	case <-time.After(5 * time.Second):
		close(releaseLock)
		close(releaseGET)
		t.Fatal("holder did not lock cluster")
	}
	close(releaseGET)
	time.Sleep(1700 * time.Millisecond)
	close(releaseLock)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	awaitFinalRestore(t, f, resp.ID, StatusFailed)
	for _, a := range f.provider.Actions() {
		if a.GetVerb() == "create" {
			t.Fatal("expired live backup submitted after lock wait")
		}
	}
}

func TestFinalBackupAcceptedTimeoutAndRestart(t *testing.T) {
	for _, refused := range []bool{false, true} {
		t.Run(fmt.Sprint("refused=", refused), func(t *testing.T) {
			f := newFinalRestoreFixture(t)
			var creates atomic.Int32
			f.provider.PrependReactor("create", "backups", func(a k8stesting.Action) (bool, runtime.Object, error) {
				creates.Add(1)
				if refused {
					return true, nil, apierrors.NewForbidden(backupGVR.GroupResource(), "timeout-backup", errors.New("denied"))
				}
				obj := a.(k8stesting.CreateAction).GetObject().(*unstructured.Unstructured).DeepCopy()
				obj.Object["status"] = map[string]interface{}{"phase": StatusCompleted}
				if err := f.provider.Tracker().Create(backupGVR, obj, veleroNamespace); err != nil {
					return true, nil, err
				}
				return true, nil, context.DeadlineExceeded
			})
			resp, err := f.svc.CreateBackup(context.Background(), CreateBackupRequest{Name: "timeout-backup", ClusterID: f.target.ID, BackupType: BackupTypeResource, IncludedNamespaces: []string{"owned-source"}, IncludedResources: []string{"configmaps"}}, f.scope, "owner")
			if err != nil {
				t.Fatal(err)
			}
			want := StatusUnknown
			if refused {
				want = StatusFailed
			}
			until := time.Now().Add(5 * time.Second)
			var row VeleroBackup
			for time.Now().Before(until) {
				if err := f.db.First(&row, resp.ID).Error; err != nil {
					t.Fatal(err)
				}
				if row.Status == want {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if row.Status != want {
				t.Fatalf("want %s, got %s", want, row.Status)
			}
			r := NewReconciler(f.db, f.cs, time.Second)
			r.clusterService = f.svc.clusterService
			if err := r.reconcileBackups(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := f.db.First(&row, resp.ID).Error; err != nil {
				t.Fatal(err)
			}
			if !refused && row.Status != StatusCompleted {
				t.Fatal("accepted backup was not corrected after restart")
			}
			if refused && row.Status != StatusFailed {
				t.Fatal("deterministic denial changed state")
			}
			if creates.Load() != 1 {
				t.Fatal("backup was blindly resent")
			}
		})
	}
}

func TestFinalRestartUnknownDoesNotResend(t *testing.T) {
	f := newFinalRestoreFixture(t)
	row := VeleroRestore{Name: "uncertain", ClusterID: f.target.ID, BackupName: f.source.Name, Status: StatusUnknown, Phase: phaseSubmissionUnknown}
	if err := f.db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	r := NewReconciler(f.db, f.cs, time.Second)
	r.clusterService = f.svc.clusterService
	if err := r.reconcileRestores(context.Background()); err != nil {
		t.Fatal(err)
	}
	awaitFinalRestore(t, f, row.ID, StatusUnknown)
	obj := &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": veleroAPIVersion, "kind": "Restore", "metadata": map[string]interface{}{"name": row.Name, "namespace": veleroNamespace}, "status": map[string]interface{}{"phase": StatusCompleted}}}
	obj.SetAnnotations(map[string]string{recordIdentityAnnotation: "wrong-record"})
	if err := f.provider.Tracker().Create(restoreGVR, obj, veleroNamespace); err != nil {
		t.Fatal(err)
	}
	if err := r.reconcileRestore(context.Background(), &row); err == nil {
		t.Fatal("foreign provider identity reconciled")
	}
	awaitFinalRestore(t, f, row.ID, StatusUnknown)
	obj.SetAnnotations(map[string]string{recordIdentityAnnotation: fmt.Sprint(row.ID)})
	if err := f.provider.Tracker().Update(restoreGVR, obj, veleroNamespace); err != nil {
		t.Fatal(err)
	}
	if err := r.reconcileRestores(context.Background()); err != nil {
		t.Fatal(err)
	}
	awaitFinalRestore(t, f, row.ID, StatusCompleted)
	for _, a := range f.provider.Actions() {
		if a.GetVerb() != "get" {
			t.Fatalf("restart resent mutation: %s", a.GetVerb())
		}
	}
}

func TestRestoreFinalDeterministicProviderRefusal(t *testing.T) {
	f := newFinalRestoreFixture(t)
	f.provider.PrependReactor("create", "restores", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(restoreGVR.GroupResource(), f.req.Name, errors.New("denied"))
	})
	resp, err := f.svc.CreateRestore(context.Background(), f.req, f.scope, "owner")
	if err != nil {
		t.Fatal(err)
	}
	awaitFinalRestore(t, f, resp.ID, StatusFailed)
}
