package release

import (
	"errors"
	"strings"
	"testing"
	"time"

	"pantheon-base/modules/business/k8s/cluster"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/testmysql"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestSetContainerImageStrict(t *testing.T) {
	containers := []corev1.Container{
		{Name: "app", Image: "old:v1"},
		{Name: "sidecar", Image: "busybox:1.36"},
	}

	if err := setContainerImageStrict(containers, "", "new:v2"); err != nil {
		t.Fatalf("update first container: %v", err)
	}
	if containers[0].Image != "new:v2" {
		t.Fatalf("expected first container updated, got %q", containers[0].Image)
	}
	if containers[1].Image != "busybox:1.36" {
		t.Fatalf("second container should be untouched, got %q", containers[1].Image)
	}

	if err := setContainerImageStrict(containers, "sidecar", "busybox:1.37"); err != nil {
		t.Fatalf("update named container: %v", err)
	}
	if containers[1].Image != "busybox:1.37" {
		t.Fatalf("expected named container updated, got %q", containers[1].Image)
	}

	if err := setContainerImageStrict(containers, "missing", "fallback:v3"); !errors.Is(err, errReleaseContainerNotFound) {
		t.Fatalf("expected unknown container to be rejected, got %v", err)
	}
}

func TestReleaseDurationFromEnvIsBounded(t *testing.T) {
	t.Setenv("PANTHEON_K8S_RELEASE_TIMEOUT", "45s")
	if got := releaseDurationFromEnv("PANTHEON_K8S_RELEASE_TIMEOUT", time.Second, time.Minute); got != 45*time.Second {
		t.Fatalf("expected configured duration, got %s", got)
	}
	t.Setenv("PANTHEON_K8S_RELEASE_TIMEOUT", "2m")
	if got := releaseDurationFromEnv("PANTHEON_K8S_RELEASE_TIMEOUT", time.Second, time.Minute); got != time.Second {
		t.Fatalf("expected fallback for unbounded duration, got %s", got)
	}
}

func TestToResponse(t *testing.T) {
	now := time.Now()
	r := &Release{
		ID:            1,
		Name:          "release-1",
		ClusterID:     10,
		Namespace:     "default",
		WorkloadType:  "deployment",
		WorkloadName:  "nginx",
		ContainerName: "app",
		ImageBefore:   "nginx:1.24",
		ImageAfter:    "nginx:1.25",
		Strategy:      "RollingUpdate",
		Status:        "success",
		StartedAt:     &now,
		FinishedAt:    &now,
		CreatedBy:     "1",
		CreatedAt:     now,
	}

	resp := toResponse(r)
	if resp.ID != 1 || resp.WorkloadName != "nginx" || resp.ImageAfter != "nginx:1.25" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.StartedAt == "" || resp.FinishedAt == "" {
		t.Fatal("expected started/finished timestamps to be formatted")
	}

	r.StartedAt = nil
	r.FinishedAt = nil
	resp = toResponse(r)
	if resp.StartedAt != "" || resp.FinishedAt != "" {
		t.Fatalf("expected empty timestamps for nil, got %q/%q", resp.StartedAt, resp.FinishedAt)
	}
}

func setupReleaseService(t *testing.T, clientset kubernetes.Interface) (*gorm.DB, *ReleaseService, uint64) {
	t.Helper()

	db := testmysql.Open(t)
	clusterSvc := cluster.NewClusterService(db)
	if err := clusterSvc.Migrate(); err != nil {
		t.Fatalf("migrate clusters: %v", err)
	}
	releaseSvc := NewReleaseService(db, clusterSvc)
	if err := releaseSvc.Migrate(); err != nil {
		t.Fatalf("migrate releases: %v", err)
	}
	clusterSvc.SetReferenceChecker(releaseSvc)

	item := cluster.Cluster{
		Code:        "release-test-" + sanitizeTestName(t.Name()),
		Name:        "Release Test Cluster",
		Environment: "test",
		Status:      "healthy",
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatalf("seed cluster: %v", err)
	}
	releaseSvc.SetClientProvider(func(clusterID uint64, _ *common.DataScopeReq) (kubernetes.Interface, error) {
		if clusterID != item.ID {
			return nil, errors.New("unexpected cluster")
		}
		return clientset, nil
	})
	releaseSvc.SetRolloutPolicy(30*time.Millisecond, time.Millisecond)
	return db, releaseSvc, item.ID
}

func TestReleaseCreatePersistsIntentBeforeApply(t *testing.T) {
	db := testmysql.Open(t)
	clusterSvc := cluster.NewClusterService(db)
	if err := clusterSvc.Migrate(); err != nil {
		t.Fatalf("migrate clusters: %v", err)
	}
	clusterItem := cluster.Cluster{Code: "intent-first-" + sanitizeTestName(t.Name()), Name: "Intent First", Environment: "test"}
	if err := db.Create(&clusterItem).Error; err != nil {
		t.Fatalf("seed cluster: %v", err)
	}

	releaseSvc := NewReleaseService(db, clusterSvc)
	called := false
	releaseSvc.SetClientProvider(func(uint64, *common.DataScopeReq) (kubernetes.Interface, error) {
		called = true
		return k8sfake.NewSimpleClientset(), nil
	})

	_, err := releaseSvc.Create(CreateReleaseRequest{
		Name:           "intent-first",
		ClusterID:      clusterItem.ID,
		Namespace:      "default",
		WorkloadType:   "deployment",
		WorkloadName:   "api",
		Image:          "api:v2",
		IdempotencyKey: "intent-first-key",
	}, "1", nil)
	if err == nil {
		t.Fatal("expected release intent persistence failure without release table")
	}
	if called {
		t.Fatal("kubernetes client must not be called when durable intent persistence fails")
	}
}

func TestReleaseCreateIsIdempotentAfterRollout(t *testing.T) {
	clientset := k8sfake.NewSimpleClientset(readyDeployment("api", "old:v1"))
	db, releaseSvc, clusterID := setupReleaseService(t, clientset)

	updates := 0
	clientset.PrependReactor("update", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
		updates++
		return false, nil, nil
	})
	request := CreateReleaseRequest{
		Name:           "api-v2",
		ClusterID:      clusterID,
		Namespace:      "default",
		WorkloadType:   "deployment",
		WorkloadName:   "api",
		Image:          "api:v2",
		IdempotencyKey: "api-v2-key",
	}

	first, err := releaseSvc.Create(request, "1", nil)
	if err != nil {
		t.Fatalf("create release: %v", err)
	}
	second, err := releaseSvc.Create(request, "1", nil)
	if err != nil {
		t.Fatalf("replay release: %v", err)
	}
	if first.ID != second.ID || first.Status != releaseStatusSucceeded {
		t.Fatalf("expected idempotent succeeded release, got first=%+v second=%+v", first, second)
	}
	if updates != 1 {
		t.Fatalf("expected exactly one Kubernetes update, got %d", updates)
	}

	var persisted Release
	if err := db.First(&persisted, first.ID).Error; err != nil {
		t.Fatalf("load persisted release: %v", err)
	}
	if persisted.ImageBefore != "old:v1" || persisted.Status != releaseStatusSucceeded || persisted.RequestSnapshot == nil {
		t.Fatalf("unexpected durable release state: %+v", persisted)
	}
}

func TestReleaseCreateRejectsConflictingIdempotencySnapshot(t *testing.T) {
	clientset := k8sfake.NewSimpleClientset(readyDeployment("api", "old:v1"))
	_, releaseSvc, clusterID := setupReleaseService(t, clientset)
	request := CreateReleaseRequest{
		Name:           "api-v2",
		ClusterID:      clusterID,
		Namespace:      "default",
		WorkloadType:   "deployment",
		WorkloadName:   "api",
		Image:          "api:v2",
		IdempotencyKey: "api-v2-conflict-key",
	}
	if _, err := releaseSvc.Create(request, "1", nil); err != nil {
		t.Fatalf("create release: %v", err)
	}

	request.Image = "api:v3"
	if _, err := releaseSvc.Create(request, "1", nil); !errors.Is(err, errReleaseIdempotency) {
		t.Fatalf("expected conflicting snapshot to be rejected, got %v", err)
	}
}

func TestReleaseCreatePersistsRolloutFailureAndTimeout(t *testing.T) {
	t.Run("failure", func(t *testing.T) {
		deployment := readyDeployment("api", "old:v1")
		deployment.Status.Conditions = []appsv1.DeploymentCondition{{
			Type:   appsv1.DeploymentProgressing,
			Status: corev1.ConditionFalse,
			Reason: "ProgressDeadlineExceeded",
		}}
		clientset := k8sfake.NewSimpleClientset(deployment)
		db, releaseSvc, clusterID := setupReleaseService(t, clientset)

		_, err := releaseSvc.Create(CreateReleaseRequest{
			Name: "failure", ClusterID: clusterID, Namespace: "default", WorkloadType: "deployment", WorkloadName: "api", Image: "api:v2",
		}, "1", nil)
		if !errors.Is(err, errReleaseRolloutFailed) {
			t.Fatalf("expected rollout failure, got %v", err)
		}
		var record Release
		if err := db.Order("id DESC").First(&record).Error; err != nil {
			t.Fatalf("load failed release: %v", err)
		}
		if record.Status != releaseStatusFailed || record.FinishedAt == nil || record.ConditionSummary == "" {
			t.Fatalf("expected persisted failed rollout, got %+v", record)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		deployment := readyDeployment("api", "old:v1")
		deployment.Status.UpdatedReplicas = 0
		deployment.Status.AvailableReplicas = 0
		deployment.Status.ReadyReplicas = 0
		deployment.Status.Conditions = nil
		clientset := k8sfake.NewSimpleClientset(deployment)
		db, releaseSvc, clusterID := setupReleaseService(t, clientset)
		releaseSvc.SetRolloutPolicy(8*time.Millisecond, time.Millisecond)

		_, err := releaseSvc.Create(CreateReleaseRequest{
			Name: "timeout", ClusterID: clusterID, Namespace: "default", WorkloadType: "deployment", WorkloadName: "api", Image: "api:v2",
		}, "1", nil)
		if !errors.Is(err, errReleaseRolloutTimedOut) {
			t.Fatalf("expected rollout timeout, got %v", err)
		}
		var record Release
		if err := db.Order("id DESC").First(&record).Error; err != nil {
			t.Fatalf("load timed-out release: %v", err)
		}
		if record.Status != releaseStatusTimedOut || record.FinishedAt == nil {
			t.Fatalf("expected persisted timed-out rollout, got %+v", record)
		}
	})
}

func TestReleaseRollbackIsIdempotentAndClusterDeleteIsBlocked(t *testing.T) {
	clientset := k8sfake.NewSimpleClientset(readyDeployment("api", "old:v1"))
	_, releaseSvc, clusterID := setupReleaseService(t, clientset)

	forward, err := releaseSvc.Create(CreateReleaseRequest{
		Name: "forward", ClusterID: clusterID, Namespace: "default", WorkloadType: "deployment", WorkloadName: "api", Image: "api:v2",
	}, "1", nil)
	if err != nil {
		t.Fatalf("create forward release: %v", err)
	}
	first, err := releaseSvc.Rollback(forward.ID, RollbackReleaseRequest{IdempotencyKey: "rollback-key"}, "1", nil)
	if err != nil {
		t.Fatalf("create rollback: %v", err)
	}
	second, err := releaseSvc.Rollback(forward.ID, RollbackReleaseRequest{IdempotencyKey: "rollback-key"}, "1", nil)
	if err != nil {
		t.Fatalf("replay rollback: %v", err)
	}
	if first.ID != second.ID || first.PreviousReleaseID != forward.ID || first.Status != releaseStatusSucceeded {
		t.Fatalf("unexpected rollback replay: first=%+v second=%+v", first, second)
	}
	if err := releaseSvc.clusterSvc.Delete(clusterID, nil); err == nil || err.Error() != "k8s.cluster.has_references" {
		t.Fatalf("expected referenced cluster deletion to be rejected, got %v", err)
	}
}

func TestReleaseReconcileCompletesAmbiguousTimedOutRecord(t *testing.T) {
	clientset := k8sfake.NewSimpleClientset(readyDeployment("api", "api:v2"))
	db, releaseSvc, clusterID := setupReleaseService(t, clientset)
	now := time.Now()
	record := Release{
		Name:               "ambiguous",
		ClusterID:          clusterID,
		Namespace:          "default",
		WorkloadType:       "deployment",
		WorkloadName:       "api",
		ImageAfter:         "api:v2",
		Status:             releaseStatusTimedOut,
		IdempotencyKey:     "ambiguous-key",
		RequestFingerprint: "fingerprint",
		RequestSnapshot:    datatypes.JSON([]byte(`{"operation":"create"}`)),
		TargetGeneration:   1,
		StartedAt:          &now,
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("seed timed-out release: %v", err)
	}

	resp, err := releaseSvc.Reconcile(record.ID, nil)
	if err != nil {
		t.Fatalf("reconcile timed-out release: %v", err)
	}
	if resp.Status != releaseStatusSucceeded || resp.LastReconciledAt == "" {
		t.Fatalf("expected reconciliation to close release, got %+v", resp)
	}
}

func TestReleaseReconcileDoesNotFinishBeforeTargetIsDurable(t *testing.T) {
	deployment := readyDeployment("api", "api:v2")
	deployment.Status.Conditions = []appsv1.DeploymentCondition{{
		Type:   appsv1.DeploymentProgressing,
		Status: corev1.ConditionFalse,
		Reason: "ProgressDeadlineExceeded",
	}}
	clientset := k8sfake.NewSimpleClientset(deployment)
	db, releaseSvc, clusterID := setupReleaseService(t, clientset)

	now := time.Now()
	record := Release{
		Name:               "target-not-recorded",
		ClusterID:          clusterID,
		Namespace:          "default",
		WorkloadType:       "deployment",
		WorkloadName:       "api",
		ImageAfter:         "api:v2",
		Status:             releaseStatusApplying,
		IdempotencyKey:     "target-not-recorded-key",
		RequestFingerprint: "fingerprint",
		RequestSnapshot:    datatypes.JSON([]byte(`{"operation":"create"}`)),
		StartedAt:          &now,
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("seed applying release: %v", err)
	}

	resp, err := releaseSvc.Reconcile(record.ID, nil)
	if !errors.Is(err, errReleaseReconcilePending) {
		t.Fatalf("expected reconciliation to remain pending, got response=%+v err=%v", resp, err)
	}
	var persisted Release
	if err := db.First(&persisted, record.ID).Error; err != nil {
		t.Fatalf("load applying release: %v", err)
	}
	if persisted.Status != releaseStatusApplying || persisted.LastReconciledAt == nil {
		t.Fatalf("reconciliation must not terminally close un-targeted release: %+v", persisted)
	}
}

func TestReleaseFinishIsIdempotentForSameTerminalOutcome(t *testing.T) {
	db, releaseSvc, clusterID := setupReleaseService(t, k8sfake.NewSimpleClientset())
	record := Release{
		Name:               "finish-idempotent",
		ClusterID:          clusterID,
		Namespace:          "default",
		WorkloadType:       "deployment",
		WorkloadName:       "api",
		ImageAfter:         "api:v2",
		Status:             releaseStatusApplying,
		IdempotencyKey:     "finish-idempotent-key",
		RequestFingerprint: "fingerprint",
		RequestSnapshot:    datatypes.JSON([]byte(`{"operation":"create"}`)),
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("seed applying release: %v", err)
	}

	if err := releaseSvc.finish(record.ID, releaseStatusSucceeded, workloadObservation{}, ""); err != nil {
		t.Fatalf("first terminal transition: %v", err)
	}
	if err := releaseSvc.finish(record.ID, releaseStatusSucceeded, workloadObservation{}, ""); err != nil {
		t.Fatalf("same terminal transition should be idempotent: %v", err)
	}
}

func readyDeployment(name, image string) *appsv1.Deployment {
	replicas := int32(1)
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", Generation: 1},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "app", Image: image}},
			}},
		},
		Status: appsv1.DeploymentStatus{
			ObservedGeneration: 1,
			UpdatedReplicas:    1,
			AvailableReplicas:  1,
			ReadyReplicas:      1,
			Conditions: []appsv1.DeploymentCondition{{
				Type:   appsv1.DeploymentAvailable,
				Status: corev1.ConditionTrue,
			}},
		},
	}
}

func sanitizeTestName(name string) string {
	return strings.NewReplacer("/", "-", " ", "-", "_", "-").Replace(strings.ToLower(name))
}
