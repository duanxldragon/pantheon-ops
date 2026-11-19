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
	biztx "github.com/duanxldragon/pantheon-base/backend/modules/business/shared/transaction"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type retireReferenceReader struct {
	active bool
	err    error
	bind   func(*gorm.DB) bizcap.DeploymentReferenceReader
}

func (r *retireReferenceReader) HasExecutingServiceReferences(context.Context, uint64, uint64) (bool, error) {
	return r.active, r.err
}

func (r *retireReferenceReader) WithDeploymentReferenceTransaction(tx *gorm.DB) bizcap.DeploymentReferenceReader {
	if r.bind != nil {
		return r.bind(tx)
	}
	return r
}

type retireUnboundReader struct{}

func (retireUnboundReader) HasExecutingServiceReferences(context.Context, uint64, uint64) (bool, error) {
	return false, nil
}

func retireAdmissionFixture(t *testing.T, state string) (*Manager, InstanceRef, *common.DataScopeReq) {
	t.Helper()
	m, instance, scope := serviceInstanceUpdateFixture(t)
	if err := m.db.Model(&ServiceInstance{}).Where("id = ?", instance.ID).Updates(map[string]any{
		"observed_state": state, "current_version": "1.0", "desired_version": "2.0", "rollback_version": "0.9",
	}).Error; err != nil {
		t.Fatal(err)
	}
	instance, err := m.GetInstance(context.Background(), instance.ID, scope)
	if err != nil {
		t.Fatal(err)
	}
	return m, instance, scope
}

func retireAdmissionRequest(instance InstanceRef) InstanceStateTransitionRequest {
	return InstanceStateTransitionRequest{InstanceID: instance.ID, Action: TransitionActionRetire,
		ObservedState: ObservedStateRetired, CorrelationID: "retire-admission"}
}

func TestServiceRetireRejectsExecutingReference(t *testing.T) {
	for _, entry := range []string{"owner", "typed", "bound-typed"} {
		t.Run(entry, func(t *testing.T) {
			m, instance, scope := retireAdmissionFixture(t, ObservedStateStopped)
			m.SetDeploymentReferenceReader(&retireReferenceReader{active: true})
			before := instanceUpdateFacts(t, m, instance.ID)
			req := retireAdmissionRequest(instance)
			var err error
			switch entry {
			case "owner":
				_, err = m.TransitionState(context.Background(), req, "2", scope)
			case "typed":
				err = m.ApplyServiceInstanceState(context.Background(), bizcap.ServiceInstanceStateTransition{
					InstanceID: instance.ID, Action: req.Action, ObservedState: req.ObservedState, CorrelationID: req.CorrelationID,
				}, "2", scope)
			case "bound-typed":
				err = biztx.ReadCommittedTransaction(context.Background(), m.db, func(tx *gorm.DB) error {
					return m.WithServiceInstanceStateTransaction(tx).ApplyServiceInstanceState(context.Background(), bizcap.ServiceInstanceStateTransition{
						InstanceID: instance.ID, Action: req.Action, ObservedState: req.ObservedState, CorrelationID: req.CorrelationID,
					}, "2", scope)
				}, nil)
			}
			if err == nil || err.Error() != "business.service.instance_in_use" {
				t.Fatalf("%s bypassed Retire admission: %v", entry, err)
			}
			if instanceUpdateFacts(t, m, instance.ID) != before {
				t.Fatal("rejected Retire changed owner facts")
			}
		})
	}
}

func TestServiceRetireFailsClosedOnReferenceOwner(t *testing.T) {
	injected := errors.New("injected Retire reference query failure")
	var nilReader *retireReferenceReader
	for _, tc := range []struct {
		name   string
		reader bizcap.DeploymentReferenceReader
		want   error
	}{
		{"missing", nil, errors.New("business.service.deployment_reader_not_configured")},
		{"typed-nil", nilReader, errors.New("business.service.deployment_reader_not_configured")},
		{"nonbinding", retireUnboundReader{}, errors.New("business.service.deployment_reader_not_configured")},
		{"nil-bound", &retireReferenceReader{bind: func(*gorm.DB) bizcap.DeploymentReferenceReader { return nil }}, errors.New("business.service.deployment_reader_not_configured")},
		{"typed-nil-bound", &retireReferenceReader{bind: func(*gorm.DB) bizcap.DeploymentReferenceReader { return nilReader }}, errors.New("business.service.deployment_reader_not_configured")},
		{"query-error", &retireReferenceReader{err: injected}, injected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, instance, scope := retireAdmissionFixture(t, ObservedStateStopped)
			m.SetDeploymentReferenceReader(tc.reader)
			before := instanceUpdateFacts(t, m, instance.ID)
			_, err := m.TransitionState(context.Background(), retireAdmissionRequest(instance), "2", scope)
			if err == nil || err.Error() != tc.want.Error() {
				t.Fatalf("Retire lost fail-closed reason: %v", err)
			}
			if tc.want == injected && !errors.Is(err, injected) {
				t.Fatal("Retire lost query error identity")
			}
			if instanceUpdateFacts(t, m, instance.ID) != before {
				t.Fatal("failed reference owner changed facts")
			}
		})
	}
}

type retireTransactionalProbe struct {
	tx         *gorm.DB
	instanceID uint64
	calls      *int
}

func (r retireTransactionalProbe) HasExecutingServiceReferences(ctx context.Context, serviceID, instanceID uint64) (bool, error) {
	(*r.calls)++
	if _, ok := r.tx.Statement.ConnPool.(gorm.TxCommitter); !ok {
		return false, errors.New("Retire reference read escaped owner transaction")
	}
	if instanceID != r.instanceID || serviceID == 0 {
		return false, errors.New("Retire reference scope mismatch")
	}
	var count int64
	err := r.tx.WithContext(ctx).Table("retire_reference_probe").Where("instance_id = ?", instanceID).Count(&count).Error
	return count != 0, err
}

func TestServiceRetireUsesCallerTransactionAndRollback(t *testing.T) {
	for _, active := range []bool{false, true} {
		t.Run(map[bool]string{false: "outer-rollback", true: "uncommitted-reference"}[active], func(t *testing.T) {
			m, instance, scope := retireAdmissionFixture(t, ObservedStateStopped)
			if err := m.db.Exec("CREATE TABLE retire_reference_probe (instance_id BIGINT UNSIGNED PRIMARY KEY)").Error; err != nil {
				t.Fatal(err)
			}
			before := instanceUpdateFacts(t, m, instance.ID)
			calls := 0
			rollback := errors.New("rollback outer Retire transaction")
			err := biztx.ReadCommittedTransaction(context.Background(), m.db, func(tx *gorm.DB) error {
				if active {
					if err := tx.Exec("INSERT INTO retire_reference_probe (instance_id) VALUES (?)", instance.ID).Error; err != nil {
						return err
					}
				}
				m.SetDeploymentReferenceReader(&retireReferenceReader{bind: func(bound *gorm.DB) bizcap.DeploymentReferenceReader {
					if bound.Statement.ConnPool != tx.Statement.ConnPool {
						return &retireReferenceReader{err: errors.New("Retire rebound caller transaction")}
					}
					return retireTransactionalProbe{tx: bound, instanceID: instance.ID, calls: &calls}
				}})
				applyErr := m.WithServiceInstanceStateTransaction(tx).ApplyServiceInstanceState(context.Background(), bizcap.ServiceInstanceStateTransition{
					InstanceID: instance.ID, Action: TransitionActionRetire, ObservedState: ObservedStateRetired, CorrelationID: "outer-retire",
				}, "2", scope)
				if active {
					if applyErr == nil || applyErr.Error() != "business.service.instance_in_use" {
						t.Errorf("uncommitted executing reference was ignored: %v", applyErr)
					}
				} else if applyErr != nil {
					t.Errorf("legal bound Retire failed: %v", applyErr)
				}
				return rollback
			}, nil)
			if !errors.Is(err, rollback) {
				t.Fatalf("outer rollback lost: %v", err)
			}
			if calls != 1 {
				t.Fatalf("expected one same-transaction reference query, got %d", calls)
			}
			if instanceUpdateFacts(t, m, instance.ID) != before {
				t.Fatal("outer rollback leaked Retire owner changes")
			}
		})
	}
}

func TestServiceRetirePreservesLifecycleReplayAndScope(t *testing.T) {
	for _, state := range []string{ObservedStateUnknown, ObservedStateStopped, ObservedStateFailed} {
		t.Run(state, func(t *testing.T) {
			m, instance, scope := retireAdmissionFixture(t, state)
			calls := 0
			m.SetDeploymentReferenceReader(&retireReferenceReader{bind: func(tx *gorm.DB) bizcap.DeploymentReferenceReader {
				calls++
				if _, ok := tx.Statement.ConnPool.(gorm.TxCommitter); !ok {
					return &retireReferenceReader{err: errors.New("ordinary Retire admission has no transaction")}
				}
				return &retireReferenceReader{}
			}})
			req := retireAdmissionRequest(instance)
			updated, err := m.TransitionState(context.Background(), req, "2", scope)
			if err != nil {
				t.Fatalf("legal %s Retire: %v", state, err)
			}
			if calls != 1 {
				t.Fatalf("ordinary Retire skipped reference gate: %d", calls)
			}
			if updated.ObservedState != ObservedStateRetired || updated.DesiredState != DesiredStateRetired || updated.LifecycleVersion != instance.LifecycleVersion+1 {
				t.Fatalf("Retire lifecycle: %+v", updated)
			}
			if updated.CurrentVersion != instance.CurrentVersion || updated.DesiredVersion != instance.DesiredVersion || updated.RollbackVersion != instance.RollbackVersion {
				t.Fatal("Retire changed software versions")
			}
			before := instanceUpdateFacts(t, m, instance.ID)
			m.SetDeploymentReferenceReader(&retireReferenceReader{active: true})
			if _, err := m.TransitionState(context.Background(), req, "2", scope); err != nil {
				t.Fatalf("same Retire replay: %v", err)
			}
			if instanceUpdateFacts(t, m, instance.ID) != before {
				t.Fatal("Retire replay mutated facts")
			}
		})
	}
	for _, tc := range []struct {
		name  string
		scope *common.DataScopeReq
		stale bool
		key   string
	}{
		{"stale", nil, true, errStateStale},
		{"foreign-dept", &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 99}, false, errInstanceNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, instance, _ := retireAdmissionFixture(t, ObservedStateStopped)
			before := instanceUpdateFacts(t, m, instance.ID)
			req := retireAdmissionRequest(instance)
			if tc.stale {
				req.ExpectedLifecycleVersion = instance.LifecycleVersion + 1
			}
			if _, err := m.TransitionState(context.Background(), req, "2", tc.scope); err == nil || err.Error() != tc.key {
				t.Fatalf("Retire %s: %v", tc.name, err)
			}
			if instanceUpdateFacts(t, m, instance.ID) != before {
				t.Fatal("rejected lifecycle/scope changed facts")
			}
		})
	}
}

func TestServiceRetirePATCHAdmission(t *testing.T) {
	for _, tc := range []struct {
		name   string
		reader bizcap.DeploymentReferenceReader
		want   string
	}{
		{"executing", &retireReferenceReader{active: true}, "business.service.instance_in_use"},
		{"unavailable", nil, "business.service.deployment_reader_not_configured"},
		{"legal", &retireReferenceReader{}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, instance, _ := retireAdmissionFixture(t, ObservedStateStopped)
			m.SetDeploymentReferenceReader(tc.reader)
			before := instanceUpdateFacts(t, m, instance.ID)
			raw, err := json.Marshal(retireAdmissionRequest(instance))
			if err != nil {
				t.Fatal(err)
			}
			router := gin.New()
			NewHandler(m).RegisterRoutes(router)
			request := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/instances/%d/state", instance.ID), strings.NewReader(string(raw)))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			var response common.Response
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if tc.want != "" {
				if response.Code == http.StatusOK || response.Message != tc.want {
					t.Fatalf("PATCH lost admission reason: %+v", response)
				}
				if instanceUpdateFacts(t, m, instance.ID) != before {
					t.Fatal("rejected PATCH changed owner facts")
				}
			} else if response.Code != http.StatusOK {
				t.Fatalf("legal Retire PATCH: %+v", response)
			}
		})
	}
}

func TestServiceRetireRejectsIllegalLifecycleAndCanceledContext(t *testing.T) {
	for _, state := range []string{ObservedStateRunning, ObservedStateInstalling, ObservedStateStarting, ObservedStateStopping, ObservedStateUpgrading, ObservedStateRetired} {
		t.Run(state, func(t *testing.T) {
			m, instance, scope := retireAdmissionFixture(t, state)
			before := instanceUpdateFacts(t, m, instance.ID)
			if _, err := m.TransitionState(context.Background(), retireAdmissionRequest(instance), "2", scope); err == nil || err.Error() != errStateInvalid {
				t.Fatalf("illegal %s Retire: %v", state, err)
			}
			if instanceUpdateFacts(t, m, instance.ID) != before {
				t.Fatal("illegal Retire changed owner facts")
			}
		})
	}
	t.Run("canceled-context", func(t *testing.T) {
		m, instance, scope := retireAdmissionFixture(t, ObservedStateStopped)
		before := instanceUpdateFacts(t, m, instance.ID)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := m.TransitionState(ctx, retireAdmissionRequest(instance), "2", scope); !errors.Is(err, context.Canceled) {
			t.Fatalf("Retire lost caller cancellation: %v", err)
		}
		if instanceUpdateFacts(t, m, instance.ID) != before {
			t.Fatal("canceled Retire changed owner facts")
		}
	})
}
