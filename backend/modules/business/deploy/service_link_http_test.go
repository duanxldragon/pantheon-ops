package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	serviceowner "github.com/duanxldragon/pantheon-base/backend/modules/business/service"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type failingLinkReader struct {
	serviceowner.Reader
	owner   *serviceowner.Manager
	kind    string
	failure error
}

func (r *failingLinkReader) WithReaderTransaction(tx *gorm.DB) serviceowner.Reader {
	return &failingLinkReader{Reader: r.owner.WithReaderTransaction(tx), owner: r.owner, kind: r.kind, failure: r.failure}
}
func (r *failingLinkReader) GetInstance(ctx context.Context, id uint64, scope *common.DataScopeReq) (serviceowner.InstanceRef, error) {
	if r.kind == "instance" {
		return serviceowner.InstanceRef{}, r.failure
	}
	return r.Reader.GetInstance(ctx, id, scope)
}
func (r *failingLinkReader) GetService(ctx context.Context, id uint64, scope *common.DataScopeReq) (serviceowner.ServiceRef, error) {
	if r.kind == "service" {
		return serviceowner.ServiceRef{}, r.failure
	}
	return r.Reader.GetService(ctx, id, scope)
}
func (r *failingLinkReader) GetApplication(ctx context.Context, id uint64, scope *common.DataScopeReq) (serviceowner.ApplicationRef, error) {
	if r.kind == "application" {
		return serviceowner.ApplicationRef{}, r.failure
	}
	return r.Reader.GetApplication(ctx, id, scope)
}

func TestServiceLinkHTTPOwnerReadFailuresRemainIdentifiable(t *testing.T) {
	for _, kind := range []string{"instance", "service", "application"} {
		t.Run(kind, func(t *testing.T) {
			db, svc, task, owner, instance := serviceLinkDraftFixture(t)
			linkDraft(t, svc, task, instance)
			failure := errors.New("injected owner database connection failure")
			svc.SetServiceReader(&failingLinkReader{Reader: owner, owner: owner, kind: kind, failure: failure})
			before := linkedServiceFacts(t, db)
			_, err := svc.CreateTask(CreateTaskRequest{Name: "fault-create", PackageID: task.PackageID, BusinessScopeID: task.BusinessScopeID, TargetType: TargetTypeHost, TargetIDs: task.TargetIDs, ServiceID: instance.ServiceID, ServiceInstanceID: instance.ID}, "1", nil)
			if !errors.Is(err, failure) {
				t.Fatalf("create swallowed %s owner cause: %v", kind, err)
			}
			_, err = svc.UpdateTask(task.ID, UpdateTaskRequest{ServiceID: &instance.ServiceID, ServiceInstanceID: &instance.ID}, "1", nil)
			if !errors.Is(err, failure) {
				t.Fatalf("update swallowed %s owner cause: %v", kind, err)
			}
			_, err = svc.StartTask(task.ID, StartTaskRequest{}, "1", nil)
			if !errors.Is(err, failure) {
				t.Fatalf("start swallowed %s owner cause: %v", kind, err)
			}
			if linkedServiceFacts(t, db) != before {
				t.Fatal("owner infrastructure failure committed partial facts")
			}
		})
	}
}

func serviceLinkHTTP(t *testing.T, svc *DeployService, method, target string, payload any) common.Response {
	t.Helper()
	return serviceLinkHTTPContext(context.Background(), t, svc, method, target, payload)
}

func serviceLinkHTTPContext(ctx context.Context, t *testing.T, svc *DeployService, method, target string, payload any) common.Response {
	t.Helper()
	router := gin.New()
	handler := NewDeployHandler(svc)
	router.PUT("/tasks/:id", handler.UpdateTask)
	router.POST("/task-hosts/:id/result", handler.MarkHostResult)
	router.POST("/tasks/:id/resume", handler.ResumeTask)
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, target, strings.NewReader(string(raw))).WithContext(ctx)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	var response common.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func TestServiceLinkHTTPResultHonorsCanceledRequest(t *testing.T) {
	db, svc, task, _, _ := startedServiceLink(t)
	before := linkedServiceFacts(t, db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	response := serviceLinkHTTPContext(ctx, t, svc, http.MethodPost, fmt.Sprintf("/task-hosts/%d/result", task.Hosts[0].ID), MarkHostResultRequest{Status: TaskHostStatusSuccess, ExecutionVersion: task.Hosts[0].ExecutionVersion})
	if response.Code != common.CodeError {
		t.Fatalf("canceled result request succeeded: %+v", response)
	}
	if linkedServiceFacts(t, db) != before {
		t.Fatal("canceled HTTP request changed task/owner/CMDB/leases")
	}
}

func TestServiceLinkHTTPClassifiesDomainAndInfrastructureFailures(t *testing.T) {
	t.Run("invalid-association", func(t *testing.T) {
		db, svc, task, _, instance := serviceLinkDraftFixture(t)
		before := linkedServiceFacts(t, db)
		response := serviceLinkHTTP(t, svc, http.MethodPut, fmt.Sprintf("/tasks/%d", task.ID), map[string]any{"serviceId": instance.ServiceID})
		if response.Code != common.CodeParamInvalid || response.Message != errDeployServiceLinkInvalid {
			t.Fatalf("invalid association response: %+v", response)
		}
		if linkedServiceFacts(t, db) != before {
			t.Fatal("invalid HTTP association changed facts")
		}
	})
	t.Run("stale-owner-result-and-resume", func(t *testing.T) {
		db, svc, task, owner, instance := startedServiceLink(t)
		version := "new-operator-version"
		if _, err := owner.UpdateInstance(instance.ID, serviceowner.UpdateInstanceRequest{DesiredVersion: &version}, "2", nil); err != nil {
			t.Fatal(err)
		}
		before := linkedServiceFacts(t, db)
		response := serviceLinkHTTP(t, svc, http.MethodPost, fmt.Sprintf("/task-hosts/%d/result", task.Hosts[0].ID), MarkHostResultRequest{Status: TaskHostStatusSuccess, ExecutionVersion: task.Hosts[0].ExecutionVersion})
		if response.Code != 409 || response.Message != stateStaleKey {
			t.Fatalf("stale owner response: %+v", response)
		}
		if linkedServiceFacts(t, db) != before {
			t.Fatal("stale HTTP result changed facts")
		}
		paused, err := svc.PauseTask(task.ID, "1", nil)
		if err != nil {
			t.Fatal(err)
		}
		before = linkedServiceFacts(t, db)
		response = serviceLinkHTTP(t, svc, http.MethodPost, fmt.Sprintf("/tasks/%d/resume", task.ID), TaskCommandRequest{StateVersion: paused.StateVersion})
		if response.Code != 409 || response.Message != stateStaleKey {
			t.Fatalf("stale owner resume response: %+v", response)
		}
		if linkedServiceFacts(t, db) != before {
			t.Fatal("stale HTTP resume changed facts")
		}
	})
	t.Run("infrastructure", func(t *testing.T) {
		db, svc, task, owner, instance := serviceLinkDraftFixture(t)
		linkDraft(t, svc, task, instance)
		svc.SetServiceReader(&failingLinkReader{Reader: owner, owner: owner, kind: "service", failure: errors.New("injected owner connection failure")})
		before := linkedServiceFacts(t, db)
		response := serviceLinkHTTP(t, svc, http.MethodPut, fmt.Sprintf("/tasks/%d", task.ID), map[string]any{"serviceId": instance.ServiceID, "serviceInstanceId": instance.ID})
		if response.Code != common.CodeError {
			t.Fatalf("infrastructure misclassified: %+v", response)
		}
		if linkedServiceFacts(t, db) != before {
			t.Fatal("failed HTTP infrastructure request changed facts")
		}
	})
}
