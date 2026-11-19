package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func serviceInstanceUpdateFixture(t *testing.T) (*Manager, InstanceRef, *common.DataScopeReq) {
	t.Helper()
	m := setupServiceTestDB(t)
	scope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 7}
	app, err := m.CreateApplication(CreateApplicationRequest{Code: "update-app", Name: "Update App", BusinessScopeID: 10}, "1", scope)
	if err != nil {
		t.Fatal(err)
	}
	svc, err := m.CreateService(CreateServiceRequest{ApplicationID: app.ID, Code: "api", Name: "API", RuntimeType: RuntimeTypeHybrid}, "1", scope)
	if err != nil {
		t.Fatal(err)
	}
	instance, err := m.CreateInstance(context.Background(), CreateInstanceRequest{ServiceID: svc.ID, Environment: "prod", TargetType: TargetTypeVM, HostID: 55}, "1", scope)
	if err != nil {
		t.Fatal(err)
	}
	return m, instance, scope
}

func decodeInstanceUpdate(t *testing.T, payload map[string]any) UpdateInstanceRequest {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var req UpdateInstanceRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatal(err)
	}
	return req
}

func instanceUpdateFacts(t *testing.T, m *Manager, id uint64) string {
	t.Helper()
	var row ServiceInstance
	if err := m.db.First(&row, id).Error; err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func vmInstanceTarget(hostID uint64) map[string]any {
	return map[string]any{"targetType": TargetTypeVM, "hostId": hostID}
}

func TestServiceInstanceUpdateRejectsStaleVersion(t *testing.T) {
	m, instance, scope := serviceInstanceUpdateFixture(t)
	before := instanceUpdateFacts(t, m, instance.ID)
	_, err := m.UpdateInstance(instance.ID, decodeInstanceUpdate(t, map[string]any{
		"expectedLifecycleVersion": instance.LifecycleVersion + 1, "desiredVersion": "stale-intent",
	}), "2", scope)
	if err == nil || err.Error() != errStateStale {
		t.Fatalf("stale edit did not conflict: %v", err)
	}
	if instanceUpdateFacts(t, m, instance.ID) != before {
		t.Fatal("stale version edit changed owner facts")
	}
}

func TestServiceInstanceUpdateAcceptsZeroVersionAndRejectsReplay(t *testing.T) {
	m, instance, scope := serviceInstanceUpdateFixture(t)
	if instance.LifecycleVersion != 0 {
		t.Fatalf("fixture no longer covers zero CAS: %d", instance.LifecycleVersion)
	}
	req := decodeInstanceUpdate(t, map[string]any{"expectedLifecycleVersion": 0, "desiredVersion": "2.0"})
	updated, err := m.UpdateInstance(instance.ID, req, "2", scope)
	if err != nil || updated.DesiredVersion != "2.0" || updated.CurrentVersion != instance.CurrentVersion || updated.LifecycleVersion != 1 {
		t.Fatalf("valid zero-version edit: %+v, %v", updated, err)
	}
	before := instanceUpdateFacts(t, m, instance.ID)
	if _, err := m.UpdateInstance(instance.ID, req, "2", scope); err == nil || err.Error() != errStateStale {
		t.Fatalf("replayed catalog edit did not conflict: %v", err)
	}
	if instanceUpdateFacts(t, m, instance.ID) != before {
		t.Fatal("replayed edit changed owner facts")
	}
}

func TestServiceInstanceUpdateReplacesVMTarget(t *testing.T) {
	m, instance, scope := serviceInstanceUpdateFixture(t)
	m.cmdbReader = fakeHostReader{ref: bizcap.HostRef{ID: 56, BusinessScopeID: 10, DeptID: 7, Status: "assigned"}}
	updated, err := m.UpdateInstance(instance.ID, decodeInstanceUpdate(t, map[string]any{
		"expectedLifecycleVersion": instance.LifecycleVersion, "target": vmInstanceTarget(56), "desiredVersion": "2.0",
	}), "2", scope)
	if err != nil || updated.HostID != 56 || updated.TargetType != TargetTypeVM || updated.DesiredVersion != "2.0" || updated.LifecycleVersion != instance.LifecycleVersion+1 {
		t.Fatalf("target was not atomically replaced: %+v, %v", updated, err)
	}
	if updated.K8sClusterID != 0 || updated.Namespace != "" || updated.WorkloadKind != "" || updated.WorkloadName != "" || updated.CurrentVersion != instance.CurrentVersion {
		t.Fatalf("target replacement changed unrelated observed facts: %+v", updated)
	}
}

func TestServiceInstanceUpdateRejectsInvalidTargetsWithoutMutation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target map[string]any
		host   bizcap.HostRef
		key    string
	}{
		{"missing", vmInstanceTarget(0), bizcap.HostRef{ID: 56, BusinessScopeID: 10}, errTargetInvalid},
		{"mixed", map[string]any{"targetType": TargetTypeVM, "hostId": 56, "namespace": "prod"}, bizcap.HostRef{ID: 56, BusinessScopeID: 10}, errTargetInvalid},
		{"cross-scope", vmInstanceTarget(56), bizcap.HostRef{ID: 56, BusinessScopeID: 99}, errTargetNotFound},
		{"wrong-id", vmInstanceTarget(56), bizcap.HostRef{ID: 57, BusinessScopeID: 10}, errTargetNotFound},
		{"retired-host", vmInstanceTarget(56), bizcap.HostRef{ID: 56, BusinessScopeID: 10, Status: "retired"}, errTargetNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, instance, scope := serviceInstanceUpdateFixture(t)
			m.cmdbReader = fakeHostReader{ref: tc.host}
			before := instanceUpdateFacts(t, m, instance.ID)
			_, err := m.UpdateInstance(instance.ID, decodeInstanceUpdate(t, map[string]any{
				"expectedLifecycleVersion": instance.LifecycleVersion, "target": tc.target, "desiredVersion": "must-not-persist",
			}), "2", scope)
			if err == nil || err.Error() != tc.key {
				t.Fatalf("invalid target did not reject with %s: %v", tc.key, err)
			}
			if instanceUpdateFacts(t, m, instance.ID) != before {
				t.Fatal("invalid target committed partial owner facts")
			}
		})
	}
}

type instanceUpdateDeploymentReader struct {
	active bool
	err    error
}

func (r instanceUpdateDeploymentReader) HasExecutingServiceReferences(context.Context, uint64, uint64) (bool, error) {
	return r.active, r.err
}

func (r instanceUpdateDeploymentReader) WithDeploymentReferenceTransaction(tx *gorm.DB) bizcap.DeploymentReferenceReader {
	if _, ok := tx.Statement.ConnPool.(gorm.TxCommitter); !ok {
		return instanceUpdateDeploymentReader{err: errors.New("instance update reference check escaped transaction")}
	}
	return r
}

func TestServiceInstanceUpdateRejectsActiveReferenceAndOwnerFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		deps instanceUpdateDeploymentReader
		key  string
	}{
		{"active-reference", instanceUpdateDeploymentReader{active: true}, errTargetConflict},
		{"owner-failure", instanceUpdateDeploymentReader{err: errors.New("injected deployment reference failure")}, "injected deployment reference failure"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, instance, scope := serviceInstanceUpdateFixture(t)
			m.cmdbReader = fakeHostReader{ref: bizcap.HostRef{ID: 56, BusinessScopeID: 10}}
			m.SetDeploymentReferenceReader(tc.deps)
			before := instanceUpdateFacts(t, m, instance.ID)
			_, err := m.UpdateInstance(instance.ID, decodeInstanceUpdate(t, map[string]any{
				"expectedLifecycleVersion": instance.LifecycleVersion, "target": vmInstanceTarget(56),
			}), "2", scope)
			if err == nil || err.Error() != tc.key {
				t.Fatalf("reference gate lost refusal: %v", err)
			}
			if instanceUpdateFacts(t, m, instance.ID) != before {
				t.Fatal("reference refusal changed owner facts")
			}
		})
	}
}

func TestServiceInstanceUpdateRejectsTargetEditWhileActiveOrRetired(t *testing.T) {
	for _, state := range []string{ObservedStateRunning, ObservedStateInstalling, ObservedStateStarting, ObservedStateStopping, ObservedStateUpgrading, ObservedStateRetired} {
		t.Run(state, func(t *testing.T) {
			m, instance, scope := serviceInstanceUpdateFixture(t)
			if err := m.db.Model(&ServiceInstance{}).Where("id = ?", instance.ID).Update("observed_state", state).Error; err != nil {
				t.Fatal(err)
			}
			m.cmdbReader = fakeHostReader{ref: bizcap.HostRef{ID: 56, BusinessScopeID: 10}}
			before := instanceUpdateFacts(t, m, instance.ID)
			_, err := m.UpdateInstance(instance.ID, decodeInstanceUpdate(t, map[string]any{
				"expectedLifecycleVersion": instance.LifecycleVersion, "target": vmInstanceTarget(56),
			}), "2", scope)
			if err == nil || err.Error() != errTargetConflict {
				t.Fatalf("active/retired target edit was not refused: %v", err)
			}
			if instanceUpdateFacts(t, m, instance.ID) != before {
				t.Fatal("active/retired target edit changed owner facts")
			}
		})
	}
}

func serviceInstanceUpdateHTTP(ctx context.Context, t *testing.T, m *Manager, id uint64, payload map[string]any) common.Response {
	t.Helper()
	router := gin.New()
	NewHandler(m).RegisterRoutes(router)
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/instances/%d", id), strings.NewReader(string(raw))).WithContext(ctx)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	var response common.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func TestServiceInstanceUpdateHTTPRequiresVersionAndClassifiesStale(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload map[string]any
		code    int
	}{
		{"missing-version", map[string]any{"desiredVersion": "2.0"}, common.CodeParamInvalid},
		{"stale-version", map[string]any{"expectedLifecycleVersion": 9, "desiredVersion": "2.0"}, http.StatusConflict},
		{"observed-version-write", map[string]any{"expectedLifecycleVersion": 0, "currentVersion": "forged", "desiredVersion": "must-not-persist"}, common.CodeParamInvalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, instance, _ := serviceInstanceUpdateFixture(t)
			before := instanceUpdateFacts(t, m, instance.ID)
			response := serviceInstanceUpdateHTTP(context.Background(), t, m, instance.ID, tc.payload)
			if response.Code != tc.code {
				t.Fatalf("HTTP edit misclassified: %+v", response)
			}
			if instanceUpdateFacts(t, m, instance.ID) != before {
				t.Fatal("rejected HTTP edit changed owner facts")
			}
		})
	}
}

func TestServiceInstanceUpdateHTTPHonorsCanceledRequest(t *testing.T) {
	m, instance, _ := serviceInstanceUpdateFixture(t)
	before := instanceUpdateFacts(t, m, instance.ID)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	response := serviceInstanceUpdateHTTP(ctx, t, m, instance.ID, map[string]any{"expectedLifecycleVersion": 0, "desiredVersion": "canceled"})
	if response.Code != common.CodeError {
		t.Fatalf("canceled HTTP edit succeeded: %+v", response)
	}
	if instanceUpdateFacts(t, m, instance.ID) != before {
		t.Fatal("canceled HTTP edit changed owner facts")
	}
}

func TestServiceInstanceUpdateConcurrentCASHasOneWinner(t *testing.T) {
	m, instance, scope := serviceInstanceUpdateFixture(t)
	requests := []UpdateInstanceRequest{
		decodeInstanceUpdate(t, map[string]any{"expectedLifecycleVersion": 0, "desiredVersion": "first"}),
		decodeInstanceUpdate(t, map[string]any{"expectedLifecycleVersion": 0, "desiredVersion": "second"}),
	}
	type result struct {
		response *InstanceResponse
		err      error
	}
	start := make(chan struct{})
	results := make(chan result, len(requests))
	for _, request := range requests {
		go func(req UpdateInstanceRequest) {
			<-start
			response, err := m.UpdateInstance(instance.ID, req, "2", scope)
			results <- result{response: response, err: err}
		}(request)
	}
	close(start)
	var winner *InstanceResponse
	for range requests {
		outcome := <-results
		if outcome.err == nil {
			if winner != nil {
				t.Fatal("both editors committed the same lifecycle version")
			}
			winner = outcome.response
		} else if outcome.err.Error() != errStateStale {
			t.Fatalf("concurrent edit lost conflict semantics: %v", outcome.err)
		}
	}
	if winner == nil {
		t.Fatal("neither valid concurrent editor committed")
	}
	stored, err := m.GetInstance(context.Background(), instance.ID, scope)
	if err != nil || stored.LifecycleVersion != 1 || stored.DesiredVersion != winner.DesiredVersion || stored.HostID != instance.HostID || stored.CurrentVersion != instance.CurrentVersion {
		t.Fatalf("concurrent edit facts do not match sole winner: %+v, %v", stored, err)
	}
}

func TestServiceInstanceUpdateRejectsDuplicateTargetAndAllowsSelf(t *testing.T) {
	m, instance, scope := serviceInstanceUpdateFixture(t)
	updated, err := m.UpdateInstance(instance.ID, decodeInstanceUpdate(t, map[string]any{
		"expectedLifecycleVersion": 0, "target": vmInstanceTarget(55), "desiredVersion": "self-edit",
	}), "2", scope)
	if err != nil || updated.DesiredVersion != "self-edit" || updated.LifecycleVersion != 1 {
		t.Fatalf("unchanged target was treated as a duplicate: %+v, %v", updated, err)
	}
	m.cmdbReader = fakeHostReader{ref: bizcap.HostRef{ID: 56, BusinessScopeID: 10}}
	if _, err := m.CreateInstance(context.Background(), CreateInstanceRequest{ServiceID: instance.ServiceID, Environment: "prod", TargetType: TargetTypeVM, HostID: 56}, "1", scope); err != nil {
		t.Fatal(err)
	}
	before := instanceUpdateFacts(t, m, instance.ID)
	_, err = m.UpdateInstance(instance.ID, decodeInstanceUpdate(t, map[string]any{
		"expectedLifecycleVersion": 1, "target": vmInstanceTarget(56), "desiredVersion": "must-not-persist",
	}), "2", scope)
	if err == nil || err.Error() != errTargetConflict {
		t.Fatalf("duplicate target did not conflict: %v", err)
	}
	if instanceUpdateFacts(t, m, instance.ID) != before {
		t.Fatal("duplicate target changed owner facts")
	}
}

func TestServiceInstanceUpdateReplacesTargetTypesAtomically(t *testing.T) {
	m, instance, scope := serviceInstanceUpdateFixture(t)
	target := map[string]any{"targetType": TargetTypeK8s, "k8sClusterId": 99, "namespace": "prod", "workloadKind": "Deployment", "workloadName": "api"}
	updated, err := m.UpdateInstance(instance.ID, decodeInstanceUpdate(t, map[string]any{"expectedLifecycleVersion": 0, "target": target}), "2", scope)
	if err != nil || updated.TargetType != TargetTypeK8s || updated.HostID != 0 || updated.K8sClusterID != 99 || updated.Namespace != "prod" || updated.WorkloadKind != "Deployment" || updated.WorkloadName != "api" || updated.LifecycleVersion != 1 {
		t.Fatalf("VM to K8s replacement was incomplete: %+v, %v", updated, err)
	}
	m.cmdbReader = fakeHostReader{ref: bizcap.HostRef{ID: 56, BusinessScopeID: 10}}
	updated, err = m.UpdateInstance(instance.ID, decodeInstanceUpdate(t, map[string]any{"expectedLifecycleVersion": 1, "target": vmInstanceTarget(56)}), "2", scope)
	if err != nil || updated.TargetType != TargetTypeVM || updated.HostID != 56 || updated.K8sClusterID != 0 || updated.Namespace != "" || updated.WorkloadKind != "" || updated.WorkloadName != "" || updated.LifecycleVersion != 2 || updated.CurrentVersion != instance.CurrentVersion {
		t.Fatalf("K8s to VM replacement was incomplete: %+v, %v", updated, err)
	}
}

type instanceUpdateK8sReader struct {
	ref bizcap.K8sTargetRef
	err error
}

func (r instanceUpdateK8sReader) ResolveTarget(context.Context, bizcap.K8sTargetQuery) (bizcap.K8sTargetRef, error) {
	return r.ref, r.err
}

func TestServiceInstanceUpdateRejectsMismatchedK8sOwnerLocator(t *testing.T) {
	valid := bizcap.K8sTargetRef{ClusterID: 99, Namespace: "prod", WorkloadKind: "Deployment", WorkloadName: "api", BusinessScopeID: 10}
	for _, field := range []string{"cluster", "namespace", "kind", "name", "scope"} {
		t.Run(field, func(t *testing.T) {
			m, instance, scope := serviceInstanceUpdateFixture(t)
			ref := valid
			switch field {
			case "cluster":
				ref.ClusterID = 100
			case "namespace":
				ref.Namespace = "other"
			case "kind":
				ref.WorkloadKind = "StatefulSet"
			case "name":
				ref.WorkloadName = "other"
			case "scope":
				ref.BusinessScopeID = 99
			}
			m.k8sReader = instanceUpdateK8sReader{ref: ref}
			before := instanceUpdateFacts(t, m, instance.ID)
			_, err := m.UpdateInstance(instance.ID, decodeInstanceUpdate(t, map[string]any{
				"expectedLifecycleVersion": 0, "desiredVersion": "must-not-persist",
				"target": map[string]any{"targetType": TargetTypeK8s, "k8sClusterId": 99, "namespace": "prod", "workloadKind": "Deployment", "workloadName": "api"},
			}), "2", scope)
			if err == nil || err.Error() != errTargetNotFound {
				t.Fatalf("mismatched owner locator did not reject: %v", err)
			}
			if instanceUpdateFacts(t, m, instance.ID) != before {
				t.Fatal("mismatched owner locator changed persisted facts")
			}
		})
	}
}

func TestServiceInstanceUpdateRejectsChangedTargetWithRunningOrRetiredIntent(t *testing.T) {
	for _, state := range []string{DesiredStateRunning, DesiredStateRetired} {
		t.Run(state, func(t *testing.T) {
			m, instance, scope := serviceInstanceUpdateFixture(t)
			if err := m.db.Model(&ServiceInstance{}).Where("id = ?", instance.ID).Update("desired_state", state).Error; err != nil {
				t.Fatal(err)
			}
			m.cmdbReader = fakeHostReader{ref: bizcap.HostRef{ID: 56, BusinessScopeID: 10}}
			before := instanceUpdateFacts(t, m, instance.ID)
			_, err := m.UpdateInstance(instance.ID, decodeInstanceUpdate(t, map[string]any{"expectedLifecycleVersion": 0, "target": vmInstanceTarget(56)}), "2", scope)
			if err == nil || err.Error() != errTargetConflict {
				t.Fatalf("changed target ignored running/retired intent: %v", err)
			}
			if instanceUpdateFacts(t, m, instance.ID) != before {
				t.Fatal("refused target edit changed owner facts")
			}
		})
	}
}
