package release

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// TestReleaseConflictPreservesTypedConflict regresses the real-server run10
// defect: a stale Deployment resourceVersion between Get/Update surfaced as
// HTTP 500 k8s.release.apply_failed even though the durable record already
// carried k8s.release.resource_version_conflict. The CAS conflict must stay
// typed (common.ErrConflict), the durable release must be failed with the
// conflict message, the image before must be captured, the provider deployment
// must remain on the old image, and the intent must never be retried.
func TestReleaseConflictPreservesTypedConflict(t *testing.T) {
	clientset := k8sfake.NewSimpleClientset(readyDeployment("api", "old:v1"))
	db, releaseSvc, clusterID := setupReleaseService(t, clientset)

	updates := 0
	clientset.PrependReactor("update", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
		updates++
		return true, nil, apierrors.NewConflict(schema.GroupResource{Resource: "deployments"}, "api", errors.New("forced conflict"))
	})

	_, err := releaseSvc.Create(CreateReleaseRequest{
		Name:         "conflict",
		ClusterID:    clusterID,
		Namespace:    "default",
		WorkloadType: "deployment",
		WorkloadName: "api",
		Image:        "api:v2",
	}, "1", nil)
	if !errors.Is(err, common.ErrConflict) {
		t.Fatalf("expected typed ErrConflict, got %v", err)
	}

	var record Release
	if err := db.Order("id DESC").First(&record).Error; err != nil {
		t.Fatalf("load failed release: %v", err)
	}
	if record.Status != releaseStatusFailed {
		t.Fatalf("expected durable failed status, got %q", record.Status)
	}
	if !strings.Contains(record.ErrorMessage, "k8s.release.resource_version_conflict") {
		t.Fatalf("expected conflict message preserved, got %q", record.ErrorMessage)
	}
	if record.ImageBefore != "old:v1" {
		t.Fatalf("expected image before captured, got %q", record.ImageBefore)
	}
	if record.FinishedAt == nil {
		t.Fatal("expected durable finished timestamp")
	}
	if updates != 1 {
		t.Fatalf("expected exactly one Kubernetes update (no implicit retry), got %d", updates)
	}

	deployment, err := clientset.AppsV1().Deployments("default").Get(context.Background(), "api", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get provider deployment: %v", err)
	}
	image, err := containerImage(deployment.Spec.Template.Spec.Containers, "app")
	if err != nil {
		t.Fatalf("read provider image: %v", err)
	}
	if image != "old:v1" {
		t.Fatalf("expected provider image unchanged, got %q", image)
	}
}

// TestReleaseErrorCodeClassifiesConflict regresses the HTTP mapping: a typed
// or wrapped stale-resource-version conflict must surface as code 409, every
// other provider failure stays 500.
func TestReleaseErrorCodeClassifiesConflict(t *testing.T) {
	if got := releaseErrorCode(common.NewConflict("k8s.release.resource_version_conflict")); got != codeConflict {
		t.Fatalf("conflict: got code %d, want %d", got, codeConflict)
	}
	wrapped := fmt.Errorf("outer: %w", common.NewConflict("k8s.release.resource_version_conflict"))
	if got := releaseErrorCode(wrapped); got != codeConflict {
		t.Fatalf("wrapped conflict: got code %d, want %d", got, codeConflict)
	}
	if got := releaseErrorCode(errors.New("provider down")); got != common.CodeError {
		t.Fatalf("generic: got code %d, want %d", got, common.CodeError)
	}
}
