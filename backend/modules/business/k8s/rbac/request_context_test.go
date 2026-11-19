package rbac

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/namespace"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type rbacRequestMarker struct{}

func newRBACRequestFixture(t *testing.T) (*gorm.DB, *K8sRBACService) {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&namespace.NamespaceBinding{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&namespace.NamespaceBinding{ClusterID: 51, Namespace: "owned", BusinessScopeID: 41, Environment: "test", AllowedActions: `["service:update"]`}).Error; err != nil {
		t.Fatal(err)
	}
	return db, NewK8sRBACService(db, nil)
}

func rbacRequestContext(ctx context.Context, t *testing.T) *gin.Context {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil).WithContext(ctx)
	return c
}

func TestK8sRBACRequestContextAllowedControl(t *testing.T) {
	_, svc := newRBACRequestFixture(t)
	c := rbacRequestContext(context.Background(), t)
	if !svc.CheckHandlerPermission(c, 51, "owned", ResourceTypeService, ActionUpdate) {
		t.Fatal("real allowed binding must authorize the active request")
	}
	if !svc.namespaceAllows(51, "owned", ResourceTypeService, ActionUpdate) {
		t.Fatal("legacy namespace permission wrapper must retain active behavior")
	}
}

func TestK8sRBACRequestContextCanceledBeforeAdmission(t *testing.T) {
	db, svc := newRBACRequestFixture(t)
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), rbacRequestMarker{}, "request"))
	cancel()
	var observed []context.Context
	const callback = "test:c04-rbac-canceled-query-context"
	if err := db.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "biz_k8s_namespace_binding" {
			observed = append(observed, tx.Statement.Context)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove(callback) })
	allowed := svc.CheckHandlerPermission(rbacRequestContext(ctx, t), 51, "owned", ResourceTypeService, ActionUpdate)
	if allowed || len(observed) == 0 {
		t.Fatalf("canceled admission: allowed=%v query-observations=%d", allowed, len(observed))
	}
	for _, queryCtx := range observed {
		if queryCtx.Value(rbacRequestMarker{}) != "request" || !errors.Is(queryCtx.Err(), context.Canceled) {
			t.Fatalf("RBAC query lost the canceled request context: marker=%v err=%v", queryCtx.Value(rbacRequestMarker{}), queryCtx.Err())
		}
	}
}

func TestK8sRBACRequestContextCancellationAtQueryBoundary(t *testing.T) {
	db, svc := newRBACRequestFixture(t)
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), rbacRequestMarker{}, "query-boundary"))
	entered := make(chan struct{})
	var enteredOnce sync.Once
	var observed []context.Context
	const callback = "test:c04-rbac-query-entry-gate"
	if err := db.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table != "biz_k8s_namespace_binding" {
			return
		}
		observed = append(observed, tx.Statement.Context)
		enteredOnce.Do(func() { close(entered) })
		// Hold the GORM query at entry until the real request is canceled.
		// The subsequent Count uses its own Statement.Context; the test does
		// not inject a query error or claim already-blocked driver evidence.
		<-ctx.Done()
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove(callback) })
	result := make(chan bool, 1)
	finished := make(chan struct{})
	go func() {
		result <- svc.CheckHandlerPermission(rbacRequestContext(ctx, t), 51, "owned", ResourceTypeService, ActionUpdate)
		close(finished)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-finished:
		case <-time.After(5 * time.Second):
			t.Error("RBAC admission worker did not finish during cleanup")
		}
	})
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("RBAC query did not reach the deterministic entry gate")
	}
	cancel()
	var allowed bool
	select {
	case allowed = <-result:
	case <-time.After(5 * time.Second):
		t.Fatal("RBAC query did not terminate after request cancellation")
	}
	if allowed || len(observed) == 0 {
		t.Fatalf("query-boundary cancellation: allowed=%v observations=%d", allowed, len(observed))
	}
	for _, queryCtx := range observed {
		if queryCtx.Value(rbacRequestMarker{}) != "query-boundary" || !errors.Is(queryCtx.Err(), context.Canceled) {
			t.Fatalf("query-boundary context was not retained: marker=%v err=%v", queryCtx.Value(rbacRequestMarker{}), queryCtx.Err())
		}
	}
}
