package k8s

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	serviceowner "github.com/duanxldragon/pantheon-base/backend/modules/business/service"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type capabilityErrorClusters struct {
	client    *kubernetes.Clientset
	metaReads int
}

func (c *capabilityErrorClusters) GetClientset(uint64, *common.DataScopeReq) (*kubernetes.Clientset, error) {
	return c.client, nil
}

func (c *capabilityErrorClusters) GetMeta(uint64, *common.DataScopeReq) (uint64, uint64, error) {
	c.metaReads++
	return 10, 7, nil
}

type capabilityErrorTransport func(*http.Request) (*http.Response, error)

func (f capabilityErrorTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func capabilityErrorClient(t *testing.T, failure string) (*kubernetes.Clientset, func(error) bool) {
	t.Helper()
	if failure == "network" {
		fault := errors.New("injected provider connection failure")
		client, err := kubernetes.NewForConfig(&rest.Config{Host: "http://127.0.0.1", Transport: capabilityErrorTransport(func(*http.Request) (*http.Response, error) { return nil, fault })})
		if err != nil {
			t.Fatal(err)
		}
		return client, func(err error) bool { return errors.Is(err, fault) }
	}
	statusCode, reason := http.StatusNotFound, metav1.StatusReasonNotFound
	check := func(err error) bool { return err != nil && err.Error() == "business.service.k8s_target_not_found" }
	switch failure {
	case "forbidden":
		statusCode, reason = http.StatusForbidden, metav1.StatusReasonForbidden
		check = apierrors.IsForbidden
	case "server":
		statusCode, reason = http.StatusInternalServerError, metav1.StatusReasonInternalError
		check = apierrors.IsInternalError
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/apis/apps/v1/namespaces/prod/") {
			t.Errorf("unexpected provider request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(metav1.Status{
			TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"}, Status: metav1.StatusFailure,
			Reason: reason, Code: int32(statusCode), Message: "provider fixture failure",
		})
	}))
	t.Cleanup(server.Close)
	client, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return client, check
}

func TestK8sTargetReaderPreservesProviderErrors(t *testing.T) {
	for _, kind := range []string{"Deployment", "StatefulSet", "DaemonSet"} {
		for _, failure := range []string{"not-found", "forbidden", "server", "network"} {
			t.Run(kind+"/"+failure, func(t *testing.T) {
				client, check := capabilityErrorClient(t, failure)
				clusters := &capabilityErrorClusters{client: client}
				reader := &targetReader{clusters: clusters}
				_, err := reader.ResolveTarget(context.Background(), bizcap.K8sTargetQuery{ClusterID: 99, Namespace: "prod", WorkloadKind: kind, WorkloadName: "api"})
				if !check(err) {
					t.Fatalf("%s provider failure lost its cause/classification: %v", failure, err)
				}
				if failure != "not-found" && apierrors.IsNotFound(err) {
					t.Fatalf("%s provider fault became not-found: %v", failure, err)
				}
				if clusters.metaReads != 0 {
					t.Fatal("failed workload lookup continued into target ownership metadata")
				}
			})
		}
	}
}

func TestK8sTargetReaderPreservesInFlightCancellation(t *testing.T) {
	for _, kind := range []string{"Deployment", "StatefulSet", "DaemonSet"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			entered := make(chan struct{})
			client, err := kubernetes.NewForConfig(&rest.Config{Host: "http://127.0.0.1", Transport: capabilityErrorTransport(func(request *http.Request) (*http.Response, error) {
				close(entered)
				<-request.Context().Done()
				return nil, request.Context().Err()
			})})
			if err != nil {
				t.Fatal(err)
			}
			clusters := &capabilityErrorClusters{client: client}
			reader := &targetReader{clusters: clusters}
			done := make(chan error, 1)
			go func() {
				_, err := reader.ResolveTarget(ctx, bizcap.K8sTargetQuery{ClusterID: 99, Namespace: "prod", WorkloadKind: kind, WorkloadName: "api"})
				done <- err
			}()
			select {
			case <-entered:
			case <-ctx.Done():
				t.Fatal("provider request never reached the cancellation gate")
			}
			cancel()
			if err := <-done; !errors.Is(err, context.Canceled) {
				t.Fatalf("in-flight cancellation lost its cause: %v", err)
			}
			if clusters.metaReads != 0 {
				t.Fatal("canceled lookup continued into ownership metadata")
			}
		})
	}
}

func capabilityServiceFacts(t *testing.T, db *gorm.DB, id uint64) string {
	t.Helper()
	var row serviceowner.ServiceInstance
	if err := db.First(&row, id).Error; err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestK8sTargetReaderServiceFaultClassification(t *testing.T) {
	for _, failure := range []string{"not-found", "forbidden", "server", "network"} {
		t.Run(failure, func(t *testing.T) {
			client, check := capabilityErrorClient(t, failure)
			reader := &targetReader{clusters: &capabilityErrorClusters{client: client}}
			db := testmysql.Open(t)
			owner := serviceowner.NewManager(db, serviceowner.Dependencies{K8sReader: reader})
			if err := owner.Migrate(); err != nil {
				t.Fatal(err)
			}
			app := serviceowner.Application{Code: "provider-app", Name: "Provider App", BusinessScopeID: 10, DeptID: 7, Status: serviceowner.StatusActive}
			if err := db.Create(&app).Error; err != nil {
				t.Fatal(err)
			}
			svc := serviceowner.Service{ApplicationID: app.ID, Code: "api", Name: "API", RuntimeType: serviceowner.RuntimeTypeHybrid, Status: serviceowner.StatusActive}
			if err := db.Create(&svc).Error; err != nil {
				t.Fatal(err)
			}
			instance := serviceowner.ServiceInstance{ServiceID: svc.ID, Environment: "prod", TargetType: serviceowner.TargetTypeVM, HostID: 55, DesiredState: serviceowner.DesiredStateStopped, ObservedState: serviceowner.ObservedStateStopped, Status: serviceowner.InstanceStatusActive}
			if err := db.Create(&instance).Error; err != nil {
				t.Fatal(err)
			}
			before := capabilityServiceFacts(t, db, instance.ID)
			version, desired := uint64(0), "must-not-persist"
			scope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 7}
			req := serviceowner.UpdateInstanceRequest{ExpectedLifecycleVersion: &version, DesiredVersion: &desired, Target: &serviceowner.InstanceTargetRequest{TargetType: "k8s", K8sClusterID: 99, Namespace: "prod", WorkloadKind: "Deployment", WorkloadName: "api"}}
			_, err := owner.UpdateInstance(instance.ID, req, "2", scope)
			code := common.CodeError
			if failure == "not-found" {
				code = common.CodeNotFound
				if err == nil || err.Error() != "business.service.target_not_found" {
					t.Fatalf("real provider 404 did not reach canonical Service refusal: %v", err)
				}
			} else if !check(err) {
				t.Fatalf("real provider %s lost its cause at Service: %v", failure, err)
			}
			router := gin.New()
			group := router.Group("/api/v1/business/service")
			group.Use(func(c *gin.Context) { c.Set(common.DataScopeContextKey, scope) })
			serviceowner.NewHandler(owner).RegisterRoutes(group)
			raw, err := json.Marshal(req)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/business/service/instances/%d", instance.ID), strings.NewReader(string(raw)))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			var response common.Response
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Code != code {
				t.Fatalf("real provider %s HTTP was misclassified: %+v", failure, response)
			}
			if capabilityServiceFacts(t, db, instance.ID) != before {
				t.Fatal("provider failure committed partial Service intent or target facts")
			}
		})
	}
}
