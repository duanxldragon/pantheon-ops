package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/namespace"
	k8spkg "github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/pkg"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/rbac"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
)

// This fixture uses the real owner SQL and client-go HTTP serialization, but
// the Kubernetes API is a bounded test server. It is not live cluster evidence.
type serviceMutationFixture struct {
	db       *gorm.DB
	cluster  cluster.Cluster
	binding  namespace.NamespaceBinding
	manager  *ServiceService
	rbacSvc  *rbac.K8sRBACService
	provider *serviceContractProvider
}

type serviceProviderReadGate struct {
	entered     chan struct{}
	exited      chan struct{}
	release     chan struct{}
	releaseOnce sync.Once
}

type serviceContractProvider struct {
	mu             sync.Mutex
	object         *corev1.Service
	requests       int
	mutations      int
	conflicts      int
	reviseAfterGet bool
	readStatus     int
	readGate       *serviceProviderReadGate
	mutationGate   *serviceProviderReadGate
	deleteOptions  metav1.DeleteOptions
}

func (p *serviceContractProvider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.requests++
	w.Header().Set("Content-Type", "application/json")
	write := func(code int, value any) { w.WriteHeader(code); _ = json.NewEncoder(w).Encode(value) }
	// This proxy is an in-memory API fixture, never a forwarding proxy.
	if r.Host != "8.8.8.8:6443" {
		write(http.StatusForbidden, metav1.Status{Code: http.StatusForbidden})
		return
	}
	conflict := func() {
		p.conflicts++
		write(http.StatusConflict, metav1.Status{TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"}, Status: metav1.StatusFailure, Reason: metav1.StatusReasonConflict, Message: "service precondition conflict", Code: http.StatusConflict})
	}
	decode := func(object runtime.Object) bool {
		data, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
		if err == nil {
			_, _, err = scheme.Codecs.UniversalDeserializer().Decode(data, nil, object)
		}
		if err != nil {
			write(http.StatusBadRequest, metav1.Status{Status: metav1.StatusFailure, Message: "invalid fixture request", Code: http.StatusBadRequest})
			return false
		}
		return true
	}
	if !strings.HasPrefix(r.URL.Path, "/api/v1/namespaces/owned/services") {
		write(http.StatusNotFound, metav1.Status{Code: http.StatusNotFound})
		return
	}
	// Consume the bounded body before waiting: net/http starts its background
	// disconnect reader only after body EOF, which lets request cancellation
	// reach this fixture while the provider write remains gated.
	waitMutation := func() bool {
		if p.mutationGate == nil {
			return true
		}
		close(p.mutationGate.entered)
		defer close(p.mutationGate.exited)
		select {
		case <-r.Context().Done():
			return false
		case <-p.mutationGate.release:
			return true
		}
	}
	switch r.Method {
	case http.MethodGet:
		if p.readGate != nil {
			close(p.readGate.entered)
			select {
			case <-r.Context().Done():
			case <-p.readGate.release:
			}
			close(p.readGate.exited)
			return
		}
		if p.readStatus != 0 {
			reason := metav1.StatusReasonInternalError
			switch p.readStatus {
			case http.StatusNotFound:
				reason = metav1.StatusReasonNotFound
			case http.StatusForbidden:
				reason = metav1.StatusReasonForbidden
			}
			write(p.readStatus, metav1.Status{TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"}, Status: metav1.StatusFailure, Reason: reason, Message: "fixture provider failure", Code: int32(p.readStatus)})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/services") {
			write(http.StatusOK, &corev1.ServiceList{TypeMeta: metav1.TypeMeta{Kind: "ServiceList", APIVersion: "v1"}, Items: []corev1.Service{*p.object.DeepCopy()}})
		} else {
			write(http.StatusOK, p.object.DeepCopy())
			if p.reviseAfterGet {
				// Deterministically emulate kubectl committing after the read,
				// before this handler releases the provider for the next request.
				p.object.ResourceVersion = "14"
				p.object.Spec.Selector = map[string]string{"app": "external"}
				p.reviseAfterGet = false
			}
		}
	case http.MethodPost:
		var object corev1.Service
		if !decode(&object) || !waitMutation() {
			return
		}
		object.TypeMeta = metav1.TypeMeta{Kind: "Service", APIVersion: "v1"}
		object.UID, object.ResourceVersion = types.UID("created-uid"), "1"
		p.object = object.DeepCopy()
		p.mutations++
		write(http.StatusCreated, p.object)
	case http.MethodPut:
		var object corev1.Service
		if !decode(&object) || !waitMutation() {
			return
		}
		if object.ResourceVersion != p.object.ResourceVersion || object.UID != p.object.UID {
			conflict()
			return
		}
		object.ResourceVersion = "14"
		p.object = object.DeepCopy()
		p.mutations++
		write(http.StatusOK, p.object)
	case http.MethodDelete:
		var options metav1.DeleteOptions
		if !decode(&options) || !waitMutation() {
			return
		}
		p.deleteOptions = options
		if options.Preconditions != nil {
			if options.Preconditions.ResourceVersion != nil && *options.Preconditions.ResourceVersion != p.object.ResourceVersion {
				conflict()
				return
			}
			if options.Preconditions.UID != nil && *options.Preconditions.UID != p.object.UID {
				conflict()
				return
			}
		}
		p.mutations++
		write(http.StatusOK, metav1.Status{TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"}, Status: metav1.StatusSuccess, Code: http.StatusOK})
	default:
		write(http.StatusMethodNotAllowed, metav1.Status{Code: http.StatusMethodNotAllowed})
	}
}

func newServiceMutationFixture(t *testing.T) *serviceMutationFixture {
	t.Helper()
	t.Setenv("PANTHEON_K8S_KUBECONFIG_KEY", strings.Repeat("a1", 32))
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&cluster.Cluster{}, &namespace.NamespaceBinding{}); err != nil {
		t.Fatal(err)
	}
	provider := &serviceContractProvider{object: &corev1.Service{
		TypeMeta:   metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "catalog-api", Namespace: "owned", UID: "original-uid", ResourceVersion: "13", Labels: map[string]string{"retained": "true"}},
		Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP, ClusterIP: "10.96.0.5", ClusterIPs: []string{"10.96.0.5"}, Selector: map[string]string{"app": "old"}, Ports: []corev1.ServicePort{{Name: "http", Port: 80, TargetPort: intstr.FromString("web"), Protocol: corev1.ProtocolTCP}}},
	}}
	server := httptest.NewServer(provider)
	t.Cleanup(server.Close)
	// Production factory validates the configured API endpoint. The explicit
	// fixture proxy routes client-go HTTP locally; no request dials the public
	// endpoint, and production SSRF validation is neither changed nor disabled.
	kubeconfig := fmt.Sprintf("apiVersion: v1\nkind: Config\nclusters:\n- name: test\n  cluster:\n    server: http://8.8.8.8:6443\n    proxy-url: %s\ncontexts:\n- name: test\n  context:\n    cluster: test\n    user: test\ncurrent-context: test\nusers:\n- name: test\n  user: {}\n", server.URL)
	encrypted, err := k8spkg.EncryptKubeconfig(kubeconfig)
	if err != nil {
		t.Fatal(err)
	}
	row := cluster.Cluster{Code: "c04-contract", Name: "C04 contract fixture", Environment: "test", BusinessScopeID: 41, DeptID: 7, Status: "healthy", KubeconfigEncrypted: encrypted}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	binding := namespace.NamespaceBinding{ClusterID: row.ID, Namespace: "owned", BusinessScopeID: 41, Environment: "test", AllowedActions: `["service:create","service:update","service:delete"]`}
	if err := db.Create(&binding).Error; err != nil {
		t.Fatal(err)
	}
	clusterManager := cluster.NewClusterService(db)
	return &serviceMutationFixture{db: db, cluster: row, binding: binding, manager: NewServiceService(clusterManager, namespace.NewNamespaceService(clusterManager, db)), provider: provider}
}

func (f *serviceMutationFixture) request(t *testing.T, method, path, body string, scope *common.DataScopeReq) (*httptest.ResponseRecorder, map[string]any) {
	return f.requestWithContext(context.Background(), t, method, path, body, scope)
}

func (f *serviceMutationFixture) requestWithContext(ctx context.Context, t *testing.T, method, path, body string, scope *common.DataScopeReq) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	router := gin.New()
	group := router.Group("/api/v1/business/k8s")
	group.Use(func(c *gin.Context) { c.Set(common.DataScopeContextKey, scope) })
	NewServiceHandler(f.manager, f.rbacSvc).RegisterRoutes(group)
	request := httptest.NewRequest(method, fmt.Sprintf("/api/v1/business/k8s/clusters/%d/services%s", f.cluster.ID, path), bytes.NewBufferString(body)).WithContext(ctx)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	var payload map[string]any
	_ = json.Unmarshal(recorder.Body.Bytes(), &payload)
	return recorder, payload
}

func TestK8sServiceMutationReadPublishesIdentity(t *testing.T) {
	f := newServiceMutationFixture(t)
	response, payload := f.request(t, http.MethodGet, "/catalog-api?namespace=owned", "", nil)
	if response.Code != http.StatusOK || payload["code"] != float64(common.CodeSuccess) {
		t.Fatalf("read: HTTP %d %s", response.Code, response.Body)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok || data["resourceVersion"] != "13" || data["uid"] != "original-uid" {
		t.Fatalf("read loses concurrency identity: %v", data)
	}
	ports, _ := data["ports"].([]any)
	if len(ports) != 1 || ports[0].(map[string]any)["targetPort"] != "web" {
		t.Fatalf("read loses named target port: %v", ports)
	}
}

func TestK8sServiceMutationReadProviderErrorsRemainDistinct(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		code       int
		key        string
		cause      error
		isProvider func(error) bool
	}{
		{"not-found", http.StatusNotFound, common.CodeNotFound, "k8s.service.not_found", common.ErrNotFound, apierrors.IsNotFound},
		{"forbidden", http.StatusForbidden, common.CodeForbidden, "k8s.rbac.permission_denied", common.ErrForbidden, apierrors.IsForbidden},
		{"internal", http.StatusInternalServerError, common.CodeError, "request.failed", nil, apierrors.IsInternalError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newServiceMutationFixture(t)
			f.provider.readStatus = tc.status
			_, err := f.manager.GetWithContext(context.Background(), f.cluster.ID, "owned", "catalog-api", nil)
			if !tc.isProvider(err) || (tc.cause != nil && !errors.Is(err, tc.cause)) || (tc.cause == nil && (errors.Is(err, common.ErrNotFound) || errors.Is(err, common.ErrForbidden))) {
				t.Fatalf("provider cause changed: %v", err)
			}
			response, payload := f.request(t, http.MethodGet, "/catalog-api?namespace=owned", "", nil)
			if payload["code"] != float64(tc.code) || payload["message"] != tc.key {
				t.Fatalf("provider failure envelope changed: HTTP %d %s", response.Code, response.Body)
			}
			f.provider.mu.Lock()
			defer f.provider.mu.Unlock()
			if f.provider.requests != 2 || f.provider.mutations != 0 {
				t.Fatalf("provider failure fixture not reached or changed object: requests=%d mutations=%d", f.provider.requests, f.provider.mutations)
			}
		})
	}
}

func TestK8sServiceMutationUpdatePreservesProviderFields(t *testing.T) {
	f := newServiceMutationFixture(t)
	body := `{"type":"ClusterIP","selector":{"app":"new"},"ports":[{"name":"http","port":8080,"targetPort":"web","protocol":"TCP"}],"resourceVersion":"13","uid":"original-uid"}`
	response, _ := f.request(t, http.MethodPut, "/catalog-api?namespace=owned", body, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("update route/contract: HTTP %d %s", response.Code, response.Body)
	}
	f.provider.mu.Lock()
	defer f.provider.mu.Unlock()
	object := f.provider.object
	if f.provider.mutations != 1 || object.Spec.Selector["app"] != "new" || object.Spec.Ports[0].Port != 8080 || object.Spec.Ports[0].TargetPort.StrVal != "web" || object.Spec.ClusterIP != "10.96.0.5" || object.Labels["retained"] != "true" {
		t.Fatalf("structured update lost fields: %+v", object)
	}
}

func TestK8sServiceMutationUpdatePreservesOpaquePortFields(t *testing.T) {
	for _, tc := range []struct {
		name, currentName, updateName, protocol string
		port                                    int32
		preserve                                bool
	}{
		{"named-selector-only", "http", "http", "TCP", 80, true},
		{"named-routing-change", "http", "http", "UDP", 8080, true},
		{"unnamed-unchanged-identity", "", "", "TCP", 80, true},
		{"renamed-port-is-new", "http", "replacement", "TCP", 80, false},
		{"unnamed-changed-port-is-new", "", "", "TCP", 8080, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newServiceMutationFixture(t)
			appProtocol := "kubernetes.io/h2c"
			f.provider.object.Spec.Ports[0].Name = tc.currentName
			f.provider.object.Spec.Ports[0].AppProtocol = &appProtocol
			request := UpdateServiceRequest{Type: "ClusterIP", Selector: map[string]string{"app": "new"}, Ports: []ServicePort{{Name: tc.updateName, Port: tc.port, TargetPort: intstr.FromString("web"), Protocol: tc.protocol}}, ResourceVersion: "13", UID: "original-uid"}
			body, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			response, payload := f.request(t, http.MethodPut, "/catalog-api?namespace=owned", string(body), nil)
			if response.Code != http.StatusOK || payload["code"] != float64(common.CodeSuccess) {
				t.Fatalf("opaque port update: HTTP %d %s", response.Code, response.Body)
			}
			f.provider.mu.Lock()
			defer f.provider.mu.Unlock()
			port := f.provider.object.Spec.Ports[0]
			if f.provider.requests != 2 || f.provider.mutations != 1 || port.Name != tc.updateName || port.Port != tc.port || string(port.Protocol) != tc.protocol || f.provider.object.Spec.Selector["app"] != "new" {
				t.Fatalf("provider update control failed: requests=%d mutations=%d port=%+v", f.provider.requests, f.provider.mutations, port)
			}
			if tc.preserve {
				if port.AppProtocol == nil || *port.AppProtocol != appProtocol {
					t.Fatalf("existing logical port lost opaque appProtocol: %+v", port)
				}
			} else if port.AppProtocol != nil {
				t.Fatalf("new port inherited unrelated opaque appProtocol: %+v", port)
			}
		})
	}
}

func TestK8sServiceMutationUpdateRejectsStale(t *testing.T) {
	f := newServiceMutationFixture(t)
	body := `{"type":"ClusterIP","selector":{"app":"new"},"ports":[{"name":"http","port":8080,"targetPort":8080,"protocol":"TCP"}],"resourceVersion":"12","uid":"original-uid"}`
	response, payload := f.request(t, http.MethodPut, "/catalog-api?namespace=owned", body, nil)
	if response.Code != http.StatusConflict || payload["message"] != "k8s.service.resource_version_conflict" {
		t.Fatalf("stale update: HTTP %d %s", response.Code, response.Body)
	}
	if f.provider.mutations != 0 || f.provider.object.Spec.Selector["app"] != "old" {
		t.Fatal("stale update changed object")
	}
}

func TestK8sServiceMutationUpdateRejectsProviderRevisionChange(t *testing.T) {
	f := newServiceMutationFixture(t)
	f.provider.reviseAfterGet = true
	body := `{"type":"ClusterIP","selector":{"app":"new"},"ports":[{"name":"http","port":8080,"targetPort":8080,"protocol":"TCP"}],"resourceVersion":"13","uid":"original-uid"}`
	response, payload := f.request(t, http.MethodPut, "/catalog-api?namespace=owned", body, nil)
	if response.Code != http.StatusConflict || payload["message"] != "k8s.service.resource_version_conflict" {
		t.Fatalf("provider revision conflict: HTTP %d %s", response.Code, response.Body)
	}
	f.provider.mu.Lock()
	defer f.provider.mu.Unlock()
	if f.provider.requests != 2 || f.provider.conflicts != 1 || f.provider.mutations != 0 || f.provider.object.ResourceVersion != "14" || f.provider.object.Spec.Selector["app"] != "external" {
		t.Fatalf("provider revision overwritten or conflict not reached: requests=%d conflicts=%d mutations=%d object=%+v", f.provider.requests, f.provider.conflicts, f.provider.mutations, f.provider.object)
	}
}

func TestK8sServiceMutationDeleteUsesIdentityPreconditions(t *testing.T) {
	for _, tc := range []struct {
		name, version, uid string
		status             int
	}{
		{"matching", "13", "original-uid", http.StatusOK},
		{"externally-updated", "12", "original-uid", http.StatusConflict},
		{"recreated-name", "13", "replaced-uid", http.StatusConflict},
		{"missing-version", "", "original-uid", http.StatusBadRequest},
		{"missing-uid", "13", "", http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newServiceMutationFixture(t)
			response, _ := f.request(t, http.MethodDelete, "/catalog-api?namespace=owned&resourceVersion="+tc.version+"&uid="+tc.uid, "", nil)
			if response.Code != tc.status {
				t.Fatalf("delete identity: HTTP %d want %d %s", response.Code, tc.status, response.Body)
			}
			f.provider.mu.Lock()
			defer f.provider.mu.Unlock()
			if tc.status == http.StatusOK {
				pre := f.provider.deleteOptions.Preconditions
				if pre == nil || pre.ResourceVersion == nil || *pre.ResourceVersion != tc.version || pre.UID == nil || string(*pre.UID) != tc.uid || f.provider.mutations != 1 {
					t.Fatalf("unguarded delete: %+v", pre)
				}
			} else if f.provider.mutations != 0 {
				t.Fatal("stale/invalid delete changed object")
			}
			if tc.status == http.StatusBadRequest && f.provider.requests != 0 {
				t.Fatal("invalid identity reached provider")
			}
		})
	}
}

func TestK8sServiceMutationAdmissionBeforeProvider(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		scope                *common.DataScopeReq
		bindingScope         uint64
		environment, actions string
		code                 int
		key                  string
	}{
		{"cross-department", &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 8}, 41, "test", `["service:create","service:update","service:delete"]`, common.CodeNotFound, "k8s.cluster.not_found"},
		{"cross-binding-scope", nil, 42, "test", `["service:create","service:update","service:delete"]`, common.CodeForbidden, "k8s.namespace.binding_invalid"},
		{"cross-binding-environment", nil, 41, "prod", `["service:create","service:update","service:delete"]`, common.CodeForbidden, "k8s.namespace.binding_invalid"},
		{"missing-action", nil, 41, "test", `[]`, common.CodeForbidden, "k8s.namespace.action_forbidden"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, operation := range []struct{ name, method, path, body string }{
				{"create", http.MethodPost, "?namespace=owned", `{"name":"new-api","type":"ClusterIP","selector":{"app":"new"},"ports":[{"port":80,"targetPort":8080,"protocol":"TCP"}]}`},
				{"update", http.MethodPut, "/catalog-api?namespace=owned", `{"type":"ClusterIP","selector":{"app":"new"},"ports":[{"port":80,"targetPort":8080,"protocol":"TCP"}],"resourceVersion":"13","uid":"original-uid"}`},
				{"delete", http.MethodDelete, "/catalog-api?namespace=owned&resourceVersion=13&uid=original-uid", ""},
			} {
				t.Run(operation.name, func(t *testing.T) {
					f := newServiceMutationFixture(t)
					if err := f.db.Model(&namespace.NamespaceBinding{}).Where("id = ?", f.binding.ID).Updates(map[string]any{"business_scope_id": tc.bindingScope, "environment": tc.environment, "allowed_actions": tc.actions}).Error; err != nil {
						t.Fatal(err)
					}
					response, payload := f.request(t, operation.method, operation.path, operation.body, tc.scope)
					if payload["code"] != float64(tc.code) || payload["message"] != tc.key || f.provider.requests != 0 || f.provider.mutations != 0 {
						t.Fatalf("admission contract: HTTP %d requests=%d mutations=%d %s", response.Code, f.provider.requests, f.provider.mutations, response.Body)
					}
				})
			}
		})
	}
}

func TestK8sServiceMutationCanceledBeforeProvider(t *testing.T) {
	for _, operation := range []string{"list", "get", "create", "update", "delete", "http-update"} {
		t.Run(operation, func(t *testing.T) {
			f := newServiceMutationFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			var err error
			ports := []ServicePort{{Port: 80, TargetPort: intstr.FromInt(8080), Protocol: "TCP"}}
			switch operation {
			case "list":
				_, err = f.manager.ListWithContext(ctx, f.cluster.ID, ServiceListQuery{Namespace: "owned"}, nil)
			case "get":
				_, err = f.manager.GetWithContext(ctx, f.cluster.ID, "owned", "catalog-api", nil)
			case "create":
				_, err = f.manager.CreateWithContext(ctx, f.cluster.ID, "owned", CreateServiceRequest{Name: "new-api", Ports: ports}, nil)
			case "update":
				_, err = f.manager.UpdateWithContext(ctx, f.cluster.ID, "owned", "catalog-api", UpdateServiceRequest{Ports: ports, ResourceVersion: "13", UID: "original-uid"}, nil)
			case "delete":
				err = f.manager.DeleteWithResourceVersionAndContext(ctx, f.cluster.ID, "owned", "catalog-api", "13", "original-uid", nil)
			case "http-update":
				response, payload := f.requestWithContext(ctx, t, http.MethodPut, "/catalog-api?namespace=owned", `{"ports":[{"port":80,"targetPort":8080}],"resourceVersion":"13","uid":"original-uid"}`, nil)
				if payload["code"] != float64(common.CodeError) {
					t.Fatalf("canceled HTTP request: HTTP %d %s", response.Code, response.Body)
				}
			}
			if operation != "http-update" && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation cause lost: %v", err)
			}
			f.provider.mu.Lock()
			defer f.provider.mu.Unlock()
			if f.provider.requests != 0 || f.provider.mutations != 0 {
				t.Fatalf("canceled request reached provider: requests=%d mutations=%d", f.provider.requests, f.provider.mutations)
			}
		})
	}
}

func TestK8sServiceMutationAllowedCreateControl(t *testing.T) {
	f := newServiceMutationFixture(t)
	response, payload := f.request(t, http.MethodPost, "?namespace=owned", `{"name":"new-api","type":"ClusterIP","selector":{"app":"new"},"ports":[{"port":80,"targetPort":8080,"protocol":"TCP"}]}`, nil)
	if response.Code != http.StatusOK || payload["code"] != float64(common.CodeSuccess) || f.provider.mutations != 1 || f.provider.object.Name != "new-api" {
		t.Fatalf("allowed fixture must reach provider: HTTP %d mutations=%d %s", response.Code, f.provider.mutations, response.Body)
	}
}

func TestK8sServiceMutationRealRBACAllowedCreateControl(t *testing.T) {
	f := newServiceMutationFixture(t)
	f.rbacSvc = rbac.NewK8sRBACService(f.db, f.manager.clusterSvc)
	response, payload := f.request(t, http.MethodPost, "?namespace=owned", `{"name":"new-api","type":"ClusterIP","selector":{"app":"new"},"ports":[{"port":80,"targetPort":8080,"protocol":"TCP"}]}`, nil)
	f.provider.mu.Lock()
	defer f.provider.mu.Unlock()
	if response.Code != http.StatusOK || payload["code"] != float64(common.CodeSuccess) || f.provider.requests != 1 || f.provider.mutations != 1 || f.provider.object.Name != "new-api" {
		t.Fatalf("real RBAC control must reach provider: HTTP %d requests=%d mutations=%d %s", response.Code, f.provider.requests, f.provider.mutations, response.Body)
	}
}

func TestK8sServiceMutationRealRBACCanceledAdmission(t *testing.T) {
	for _, operation := range []struct{ name, method, path, body string }{
		{"create", http.MethodPost, "?namespace=owned", `{"name":"new-api","ports":[{"port":80,"targetPort":8080}]}`},
		{"update", http.MethodPut, "/catalog-api?namespace=owned", `{"ports":[{"port":80,"targetPort":8080}],"resourceVersion":"13","uid":"original-uid"}`},
		{"delete", http.MethodDelete, "/catalog-api?namespace=owned&resourceVersion=13&uid=original-uid", ""},
	} {
		t.Run(operation.name, func(t *testing.T) {
			f := newServiceMutationFixture(t)
			f.rbacSvc = rbac.NewK8sRBACService(f.db, f.manager.clusterSvc)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			var observed []context.Context
			const callback = "test:c04-service-real-rbac-context"
			if err := f.db.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
				if tx.Statement.Table == "biz_k8s_namespace_binding" {
					observed = append(observed, tx.Statement.Context)
				}
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = f.db.Callback().Query().Remove(callback) })
			response, payload := f.requestWithContext(ctx, t, operation.method, operation.path, operation.body, nil)
			if payload["code"] != float64(common.CodeForbidden) || payload["message"] != "k8s.rbac.permission_denied" || len(observed) == 0 {
				t.Fatalf("real RBAC ignored request cancellation: HTTP %d query-observations=%d %s", response.Code, len(observed), response.Body)
			}
			for _, queryCtx := range observed {
				if !errors.Is(queryCtx.Err(), context.Canceled) {
					t.Fatalf("real RBAC query did not inherit canceled request: %v", queryCtx.Err())
				}
			}
			f.provider.mu.Lock()
			defer f.provider.mu.Unlock()
			if f.provider.requests != 0 || f.provider.mutations != 0 {
				t.Fatalf("canceled RBAC reached provider: requests=%d mutations=%d", f.provider.requests, f.provider.mutations)
			}
		})
	}
}

func TestK8sServiceMutationRealRBACProviderInFlightCancellation(t *testing.T) {
	for _, operation := range []string{"http-update", "manager-update-cause"} {
		t.Run(operation, func(t *testing.T) {
			f := newServiceMutationFixture(t)
			f.rbacSvc = rbac.NewK8sRBACService(f.db, f.manager.clusterSvc)
			gate := &serviceProviderReadGate{entered: make(chan struct{}), exited: make(chan struct{}), release: make(chan struct{})}
			f.provider.readGate = gate
			ctx, cancel := context.WithCancel(context.Background())
			finished := make(chan struct{})
			var operationError error
			var response *httptest.ResponseRecorder
			var payload map[string]any
			go func() {
				defer close(finished)
				if operation == "http-update" {
					response, payload = f.requestWithContext(ctx, t, http.MethodPut, "/catalog-api?namespace=owned", `{"ports":[{"port":80,"targetPort":8080}],"resourceVersion":"13","uid":"original-uid"}`, nil)
					return
				}
				_, operationError = f.manager.UpdateWithContext(ctx, f.cluster.ID, "owned", "catalog-api", UpdateServiceRequest{Ports: []ServicePort{{Port: 80, TargetPort: intstr.FromInt(8080)}}, ResourceVersion: "13", UID: "original-uid"}, nil)
			}()
			t.Cleanup(func() {
				cancel()
				gate.releaseOnce.Do(func() { close(gate.release) })
				select {
				case <-finished:
				case <-time.After(5 * time.Second):
					t.Error("Service cancellation worker did not finish during cleanup")
				}
			})
			waitServiceMutationSignal(t, gate.entered, "provider GET entry")
			cancel()
			waitServiceMutationSignal(t, finished, "caller cancellation completion")
			waitServiceMutationSignal(t, gate.exited, "provider GET exit")
			if operation == "http-update" {
				if payload["code"] != float64(common.CodeError) {
					t.Fatalf("in-flight HTTP cancellation: HTTP %d %s", response.Code, response.Body)
				}
			} else if !errors.Is(operationError, context.Canceled) {
				t.Fatalf("in-flight Manager cancellation cause lost: %v", operationError)
			}
			f.provider.mu.Lock()
			defer f.provider.mu.Unlock()
			if f.provider.requests != 1 || f.provider.mutations != 0 || f.provider.object.ResourceVersion != "13" {
				t.Fatalf("canceled provider read continued to a write: requests=%d mutations=%d revision=%s", f.provider.requests, f.provider.mutations, f.provider.object.ResourceVersion)
			}
		})
	}
}

func waitServiceMutationSignal(t *testing.T, signal <-chan struct{}, label string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", label)
	}
}

// This test isolates the HTTP fixture from SQL and the service manager. The
// provider must observe cancellation before its mutation gate is released.
func TestK8sServiceMutationProviderFixtureCancellation(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			object := &corev1.Service{
				TypeMeta:   metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "catalog-api", Namespace: "owned", UID: "original-uid", ResourceVersion: "13"},
			}
			gate := &serviceProviderReadGate{entered: make(chan struct{}), exited: make(chan struct{}), release: make(chan struct{})}
			provider := &serviceContractProvider{object: object.DeepCopy(), mutationGate: gate}
			server := httptest.NewServer(provider)
			client := server.Client()
			ctx, cancel := context.WithCancel(context.Background())
			finished := make(chan struct{})
			t.Cleanup(func() {
				cancel()
				gate.releaseOnce.Do(func() { close(gate.release) })
				select {
				case <-finished:
				case <-time.After(5 * time.Second):
					t.Error("fixture cancellation client did not finish during cleanup")
				}
				client.CloseIdleConnections()
				server.Close()
			})
			var payload any = object
			path := "/api/v1/namespaces/owned/services/catalog-api"
			switch method {
			case http.MethodPost:
				path = "/api/v1/namespaces/owned/services"
			case http.MethodDelete:
				payload = &metav1.DeleteOptions{TypeMeta: metav1.TypeMeta{Kind: "DeleteOptions", APIVersion: "v1"}}
			}
			body, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			request, err := http.NewRequestWithContext(ctx, method, server.URL+path, bytes.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			request.Host = "8.8.8.8:6443"
			request.Header.Set("Content-Type", "application/json")
			var requestError error
			go func() {
				defer close(finished)
				response, err := client.Do(request)
				requestError = err
				if response != nil {
					_ = response.Body.Close()
				}
			}()
			waitServiceMutationSignal(t, gate.entered, "fixture mutation entry")
			cancel()
			waitServiceMutationSignal(t, finished, "fixture client cancellation")
			if !errors.Is(requestError, context.Canceled) {
				t.Fatalf("fixture client cancellation cause lost: %v", requestError)
			}
			waitServiceMutationSignal(t, gate.exited, "fixture canceled provider exit")
			provider.mu.Lock()
			defer provider.mu.Unlock()
			if provider.requests != 1 || provider.mutations != 0 || provider.object.ResourceVersion != "13" {
				t.Fatalf("canceled fixture mutated: requests=%d mutations=%d revision=%s", provider.requests, provider.mutations, provider.object.ResourceVersion)
			}
		})
	}
}
