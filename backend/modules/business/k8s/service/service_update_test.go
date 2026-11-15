package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

// A loopback API protocol fixture. Real-cluster acceptance is opt-in and
// separately recorded; this fixture does not attest production authorization.
type serviceAPIFixture struct {
	mu            sync.Mutex
	current       *corev1.Service
	writes, reads int
	fail          int
	delay         bool
	lastDelete    metav1.DeleteOptions
}

func serviceProtocolFixture(t *testing.T) (*serviceAPIFixture, typedcorev1.ServiceInterface) {
	t.Helper()
	f := &serviceAPIFixture{current: &corev1.Service{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Service"}, ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "test", UID: types.UID("uid-app"), ResourceVersion: "43", Annotations: map[string]string{"private": "do-not-project"}}, Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP, ClusterIP: "10.0.0.5", Selector: map[string]string{"app": "old"}, Ports: []corev1.ServicePort{{Name: "web", Port: 80, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromString("http")}}}}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Method == http.MethodGet {
			f.reads++
			_ = json.NewEncoder(w).Encode(f.current)
			return
		}
		f.writes++
		if f.delay {
			_, _ = io.Copy(io.Discard, r.Body)
			f.mu.Unlock()
			select {
			case <-r.Context().Done():
			case <-time.After(time.Second):
			}
			f.mu.Lock()
			return
		}
		status := f.fail
		if r.Method == http.MethodPut {
			var desired corev1.Service
			if err := json.NewDecoder(r.Body).Decode(&desired); err != nil {
				t.Errorf("decode update: %v", err)
				status = 400
			}
			if desired.UID != f.current.UID || desired.ResourceVersion != f.current.ResourceVersion {
				status = 409
			}
			if status == 0 {
				desired.ResourceVersion = "44"
				f.current = &desired
				_ = json.NewEncoder(w).Encode(f.current)
				return
			}
		} else if r.Method == http.MethodDelete {
			if err := json.NewDecoder(r.Body).Decode(&f.lastDelete); err != nil {
				t.Errorf("decode deletion: %v", err)
				status = 400
			}
			p := f.lastDelete.Preconditions
			if p == nil || p.UID == nil || p.ResourceVersion == nil || *p.UID != f.current.UID || *p.ResourceVersion != f.current.ResourceVersion {
				status = 409
			}
			if status == 0 {
				_ = json.NewEncoder(w).Encode(metav1.Status{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"}, Status: "Success"})
				return
			}
		} else {
			status = 405
		}
		reason := metav1.StatusReasonUnknown
		if status == 409 {
			reason = metav1.StatusReasonConflict
		}
		if status == 403 {
			reason = metav1.StatusReasonForbidden
		}
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(metav1.Status{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"}, Status: "Failure", Code: int32(status), Reason: reason, Message: "provider token=must-not-escape"})
	}))
	t.Cleanup(server.Close)
	client, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL, Timeout: time.Second, ContentConfig: rest.ContentConfig{ContentType: "application/json", AcceptContentTypes: "application/json"}})
	if err != nil {
		t.Fatal(err)
	}
	return f, client.CoreV1().Services("test")
}
func reviewedServiceUpdate() UpdateServiceRequest {
	selector := map[string]string{"app": "new"}
	return UpdateServiceRequest{UID: "uid-app", ResourceVersion: "43", Selector: &selector}
}
func TestServiceUpdateProviderCAS(t *testing.T) {
	t.Run("preserves unedited provider fields and named ports", func(t *testing.T) {
		f, api := serviceProtocolFixture(t)
		item, err := updateService(context.Background(), api, "app", reviewedServiceUpdate())
		if err != nil {
			t.Fatal(err)
		}
		if item.ResourceVersion != "44" || item.UID != "uid-app" || item.Ports[0].TargetPortName != "http" || item.ClusterIP != "10.0.0.5" {
			t.Fatalf("lost provider fields: %+v", item)
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.current.Annotations["private"] != "do-not-project" || f.current.Spec.Selector["app"] != "new" {
			t.Fatal("provider state was not preserved")
		}
		raw, _ := json.Marshal(item)
		if strings.Contains(string(raw), "do-not-project") {
			t.Fatal("private annotations projected")
		}
	})
	for _, field := range []string{"uid", "version"} {
		t.Run("stale "+field+" never mutates", func(t *testing.T) {
			f, api := serviceProtocolFixture(t)
			req := reviewedServiceUpdate()
			if field == "uid" {
				req.UID = "replaced"
			} else {
				req.ResourceVersion = "42"
			}
			_, err := updateService(context.Background(), api, "app", req)
			if err == nil || err.Error() != msgServiceConflict {
				t.Fatalf("expected conflict: %v", err)
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			if f.writes != 0 {
				t.Fatal("stale review mutated provider")
			}
		})
	}
	t.Run("conflict after read is not replayed", func(t *testing.T) {
		f, api := serviceProtocolFixture(t)
		f.fail = 409
		_, err := updateService(context.Background(), api, "app", reviewedServiceUpdate())
		if err == nil || err.Error() != msgServiceConflict {
			t.Fatalf("expected conflict: %v", err)
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.writes != 1 || f.current.ResourceVersion != "43" {
			t.Fatal("conflict was replayed or changed state")
		}
	})
	t.Run("concurrent updates have one winner", func(t *testing.T) {
		f, api := serviceProtocolFixture(t)
		outcomes := make(chan error, 2)
		var group sync.WaitGroup
		for i := 0; i < 2; i++ {
			group.Add(1)
			go func() {
				defer group.Done()
				_, err := updateService(context.Background(), api, "app", reviewedServiceUpdate())
				outcomes <- err
			}()
		}
		group.Wait()
		close(outcomes)
		success, conflict := 0, 0
		for err := range outcomes {
			if err == nil {
				success++
			} else if err.Error() == msgServiceConflict {
				conflict++
			} else {
				t.Fatal(err)
			}
		}
		if success != 1 || conflict != 1 {
			t.Fatalf("success=%d conflict=%d", success, conflict)
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.current.ResourceVersion != "44" {
			t.Fatal("multiple updates committed")
		}
	})
	t.Run("raw provider errors are suppressed", func(t *testing.T) {
		f, api := serviceProtocolFixture(t)
		f.fail = 500
		_, err := updateService(context.Background(), api, "app", reviewedServiceUpdate())
		if err == nil || err.Error() != "business.k8s.service.updateFailed" {
			t.Fatalf("unsafe failure: %v", err)
		}
	})
	t.Run("provider revocation is classified", func(t *testing.T) {
		f, api := serviceProtocolFixture(t)
		f.fail = 403
		_, err := updateService(context.Background(), api, "app", reviewedServiceUpdate())
		if err == nil || err.Error() != "k8s.rbac.permission_denied" {
			t.Fatalf("expected denial: %v", err)
		}
	})
	t.Run("cancellation reaches write", func(t *testing.T) {
		f, api := serviceProtocolFixture(t)
		f.delay = true
		ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
		defer cancel()
		_, err := updateService(ctx, api, "app", reviewedServiceUpdate())
		if err != context.DeadlineExceeded {
			t.Fatalf("expected deadline: %v", err)
		}
	})
}
func TestServiceDeletionProviderPreconditions(t *testing.T) {
	f, api := serviceProtocolFixture(t)
	if err := deleteService(context.Background(), api, "app", "uid-app", "43"); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	p := f.lastDelete.Preconditions
	f.mu.Unlock()
	if p == nil || p.UID == nil || string(*p.UID) != "uid-app" || p.ResourceVersion == nil || *p.ResourceVersion != "43" {
		t.Fatal("provider deletion preconditions omitted")
	}
	if err := deleteService(context.Background(), api, "app", "replacement", "43"); err == nil || err.Error() != msgServiceConflict {
		t.Fatalf("expected replaced resource conflict: %v", err)
	}
}
func TestServiceUpdateRejectsInvalidBeforeProvider(t *testing.T) {
	svc := NewServiceService(nil, nil)
	ports := []ServicePort{{Port: 0, TargetPort: 80}}
	selector := map[string]string{"invalid key": "value"}
	for _, req := range []UpdateServiceRequest{{}, {UID: "u", ResourceVersion: "v"}, {UID: "u", ResourceVersion: "v", Ports: &ports}, {UID: "u", ResourceVersion: "v", Selector: &selector}} {
		_, err := svc.UpdateContext(context.Background(), 1, "test", "app", req, &common.DataScopeReq{IsAdmin: true})
		if err == nil || err.Error() != msgParamInvalid {
			t.Fatalf("invalid input reached provider: %v", err)
		}
	}
}
