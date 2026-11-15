package release

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	k8spkg "github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/pkg"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPublicReleaseRequiresReviewedIntent(t *testing.T) {
	client := k8sfake.NewSimpleClientset(readyDeployment("api", "old:v1"))
	db, svc, clusterID := setupReleaseService(t, client)
	router := gin.New()
	NewReleaseHandler(svc).RegisterRoutes(router)
	body, err := json.Marshal(CreateReleaseRequest{Name: "guarded", ClusterID: clusterID, Namespace: "default", WorkloadType: "deployment", WorkloadName: "api", Image: "new:v2"})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/releases", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)
	var envelope struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Code != 409 {
		t.Fatalf("unreviewed mutation must conflict, got %s", response.Body.String())
	}
	var count int64
	if err := db.Model(&Release{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("unreviewed request persisted %d intents", count)
	}
	for _, action := range client.Actions() {
		if action.GetVerb() == "update" {
			t.Fatal("unreviewed request reached Kubernetes mutation")
		}
	}
}

func setupReviewedRelease(t *testing.T) (*gorm.DB, *ReleaseService, uint64, *k8sfake.Clientset, CreateReleaseRequest) {
	t.Helper()
	t.Setenv("PANTHEON_K8S_KUBECONFIG_KEY", strings.Repeat("12", 32))
	target := readyDeployment("api", "old:v1")
	target.UID = types.UID("authoritative-object")
	target.ResourceVersion = "15"
	target.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{{Name: "SECRET_KEY", Value: "must-not-project-this"}}
	client := k8sfake.NewSimpleClientset(target)
	db, svc, id := setupReleaseService(t, client)
	// This successful reviewed path includes real MySQL lock/commit I/O.
	svc.SetRolloutPolicy(2*time.Second, time.Millisecond)
	req := CreateReleaseRequest{Name: "api-v2", ClusterID: id, Namespace: "default", WorkloadType: "deployment", WorkloadName: "api", ContainerName: "app", Image: "api:v2", IdempotencyKey: "reviewed-key"}
	return db, svc, id, client, req
}

func TestReleasePreviewReadOnlyAndSafeProjection(t *testing.T) {
	db, svc, _, client, req := setupReviewedRelease(t)
	preview, err := svc.PreviewCreate(context.Background(), req, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if preview.CurrentVersion != "15" || preview.TargetCount != 1 || preview.IntentID != req.IdempotencyKey || preview.Changes[0].Before != "old:v1" {
		t.Fatalf("unexpected preview %+v", preview)
	}
	bytes, err := json.Marshal(preview)
	if err != nil {
		t.Fatal(err)
	}
	for _, hidden := range []string{"must-not-project-this", "SECRET_KEY", "kubeconfig", "containers", "env"} {
		if strings.Contains(string(bytes), `"`+hidden+`"`) || strings.Contains(string(bytes), "must-not-project-this") {
			t.Fatalf("private workload field projected: %s", hidden)
		}
	}
	var count int64
	db.Model(&Release{}).Count(&count)
	if count != 0 {
		t.Fatal("preview persisted intent")
	}
	for _, action := range client.Actions() {
		if action.GetVerb() != "get" {
			t.Fatalf("preview action %s", action.GetVerb())
		}
	}
}

func TestReleaseReviewedIntentBinding(t *testing.T) {
	for _, change := range []string{"actor", "request", "key", "tamper", "expired", "cluster", "scope", "canceled", "unsafe-image"} {
		t.Run(change, func(t *testing.T) {
			db, svc, id, client, req := setupReviewedRelease(t)
			preview, err := svc.PreviewCreate(context.Background(), req, "1", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.IntentVersion = preview.IntentVersion
			actor := "1"
			ctx := context.Background()
			var scope *common.DataScopeReq
			switch change {
			case "actor":
				actor = "2"
			case "request":
				req.Image = "different:v3"
			case "key":
				req.IdempotencyKey = "different-key"
			case "tamper":
				req.IntentVersion = "x" + req.IntentVersion
			case "expired":
				payload, e := k8spkg.OpenReleaseReview(req.IntentVersion)
				if e != nil {
					t.Fatal(e)
				}
				var ticket releaseReviewTicket
				json.Unmarshal(payload, &ticket)
				ticket.Expires = time.Now().Unix() - 1
				payload, _ = json.Marshal(ticket)
				req.IntentVersion, e = k8spkg.SignReleaseReview(payload)
				if e != nil {
					t.Fatal(e)
				}
			case "cluster":
				if e := db.Model(&cluster.Cluster{}).Where("id = ?", id).Update("environment", "changed").Error; e != nil {
					t.Fatal(e)
				}
			case "scope":
				scope = &common.DataScopeReq{Mode: "dept", DeptID: 99999}
			case "canceled":
				cancelCtx, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelCtx
			case "unsafe-image":
				req.Image = "https://user:secret@example.test/app"
			}
			_, err = svc.CreateReviewed(ctx, req, actor, scope)
			if err == nil {
				t.Fatal("changed review was accepted")
			}
			var count int64
			db.Model(&Release{}).Count(&count)
			if count != 0 {
				t.Fatalf("invalid review persisted %d intents", count)
			}
			for _, action := range client.Actions() {
				if action.GetVerb() == "update" {
					t.Fatal("invalid review mutated Kubernetes")
				}
			}
		})
	}
}

func TestReleaseReviewedTargetDriftReturnsConflict(t *testing.T) {
	for _, drift := range []string{"resource-version", "uid", "api-server-race"} {
		t.Run(drift, func(t *testing.T) {
			db, svc, _, client, req := setupReviewedRelease(t)
			preview, err := svc.PreviewCreate(context.Background(), req, "1", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.IntentVersion = preview.IntentVersion
			if drift == "api-server-race" {
				client.PrependReactor("update", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, apierrors.NewConflict(schema.GroupResource{Group: "apps", Resource: "deployments"}, "api", errors.New("object changed"))
				})
			} else {
				item, e := client.AppsV1().Deployments("default").Get(context.Background(), "api", metav1.GetOptions{})
				if e != nil {
					t.Fatal(e)
				}
				if drift == "uid" {
					item.UID = types.UID("replacement")
				} else {
					item.ResourceVersion = "16"
				}
				if e := client.Tracker().Update(appsv1.SchemeGroupVersion.WithResource("deployments"), item, "default"); e != nil {
					t.Fatal(e)
				}
			}
			result, err := svc.CreateReviewed(context.Background(), req, "1", nil)
			if !errors.Is(err, common.ErrConflict) {
				t.Fatalf("drift must conflict, result=%+v err=%v", result, err)
			}
			item, e := client.AppsV1().Deployments("default").Get(context.Background(), "api", metav1.GetOptions{})
			if e != nil {
				t.Fatal(e)
			}
			if item.Spec.Template.Spec.Containers[0].Image != "old:v1" {
				t.Fatal("changed target accepted mutation")
			}
			var record Release
			if e := db.First(&record).Error; e != nil {
				t.Fatal(e)
			}
			if record.Status != releaseStatusFailed {
				t.Fatalf("conflicting intent state %s", record.Status)
			}
		})
	}
}

func TestReleaseReviewedCreateReplayAndRollback(t *testing.T) {
	db, svc, _, client, req := setupReviewedRelease(t)
	preview, err := svc.PreviewCreate(context.Background(), req, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.IntentVersion = preview.IntentVersion
	first, err := svc.CreateReviewed(context.Background(), req, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	again, err := svc.CreateReviewed(context.Background(), req, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != again.ID || first.Status != releaseStatusSucceeded {
		t.Fatal("reviewed replay diverged")
	}
	rollbackReq := RollbackReleaseRequest{IdempotencyKey: "reviewed-rollback"}
	rollback, err := svc.PreviewRollback(context.Background(), first.ID, rollbackReq, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if rollback.Changes[0].After != "old:v1" {
		t.Fatal("rollback target is not recorded image")
	}
	rollbackReq.IntentVersion = rollback.IntentVersion
	restored, err := svc.RollbackReviewed(context.Background(), first.ID, rollbackReq, "1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if restored.PreviousReleaseID != first.ID || restored.ImageAfter != "old:v1" {
		t.Fatal("invalid restored intent")
	}
	var source Release
	db.First(&source, first.ID)
	if source.ImageAfter != "api:v2" {
		t.Fatal("rollback modified source intent")
	}
	updates := 0
	for _, action := range client.Actions() {
		if action.GetVerb() == "update" {
			updates++
		}
	}
	if updates != 2 {
		t.Fatalf("forward and rollback must each apply once: %d", updates)
	}
}

func TestReviewTargetGuardAcrossWorkloadKinds(t *testing.T) {
	for _, kind := range []string{"deployment", "statefulset", "daemonset"} {
		t.Run(kind, func(t *testing.T) {
			meta := metav1.ObjectMeta{Name: "api", Namespace: "default", UID: types.UID("original"), ResourceVersion: "10"}
			template := corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "old:v1"}}}}
			var obj runtime.Object
			switch kind {
			case "deployment":
				obj = &appsv1.Deployment{ObjectMeta: meta, Spec: appsv1.DeploymentSpec{Template: template}}
			case "statefulset":
				obj = &appsv1.StatefulSet{ObjectMeta: meta, Spec: appsv1.StatefulSetSpec{Template: template}}
			case "daemonset":
				obj = &appsv1.DaemonSet{ObjectMeta: meta, Spec: appsv1.DaemonSetSpec{Template: template}}
			}
			client := k8sfake.NewSimpleClientset(obj)
			_, err := applyImage(context.Background(), client, kind, "default", "api", "app", "new:v2", &releaseReviewGuard{UID: "original", ResourceVersion: "old"})
			if !errors.Is(err, common.ErrConflict) {
				t.Fatalf("%s accepted stale RV: %v", kind, err)
			}
			for _, action := range client.Actions() {
				if action.GetVerb() == "update" {
					t.Fatal("stale review reached update")
				}
			}
		})
	}
}

func TestReviewedReplayAfterExpiryOrRotationCannotMutateAgain(t *testing.T) {
	for _, change := range []string{"expiry", "credential-rotation", "actor"} {
		t.Run(change, func(t *testing.T) {
			db, svc, id, client, req := setupReviewedRelease(t)
			preview, err := svc.PreviewCreate(context.Background(), req, "1", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.IntentVersion = preview.IntentVersion
			first, err := svc.CreateReviewed(context.Background(), req, "1", nil)
			if err != nil {
				t.Fatal(err)
			}
			actor := "1"
			switch change {
			case "expiry":
				payload, e := k8spkg.OpenReleaseReview(req.IntentVersion)
				if e != nil {
					t.Fatal(e)
				}
				var ticket releaseReviewTicket
				json.Unmarshal(payload, &ticket)
				ticket.Expires = time.Now().Unix() - 1
				payload, _ = json.Marshal(ticket)
				req.IntentVersion, e = k8spkg.SignReleaseReview(payload)
				if e != nil {
					t.Fatal(e)
				}
			case "credential-rotation":
				db.Model(&cluster.Cluster{}).Where("id = ?", id).Update("kubeconfig_encrypted", "rotated-test-ciphertext")
			case "actor":
				actor = "2"
			}
			again, err := svc.CreateReviewed(context.Background(), req, actor, nil)
			if change == "actor" {
				if err == nil {
					t.Fatal("another actor recovered replay")
				}
			} else if err != nil || again.ID != first.ID {
				t.Fatalf("could not recover persisted outcome: %v", err)
			}
			updates := 0
			for _, action := range client.Actions() {
				if action.GetVerb() == "update" {
					updates++
				}
			}
			if updates != 1 {
				t.Fatalf("ambiguous retry applied %d times", updates)
			}
		})
	}
}

func TestReviewClusterCancellationReleasesTransactionLock(t *testing.T) {
	db, svc, id, _, _ := setupReviewedRelease(t)
	ctx, cancel := context.WithCancel(context.Background())
	err := svc.clusterSvc.WithClusterLockContext(ctx, id, nil, func(tx *gorm.DB, item *cluster.Cluster) error {
		if e := tx.Model(item).Update("name", "uncommitted-review-name").Error; e != nil {
			return e
		}
		cancel()
		return ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled cluster transaction: %v", err)
	}
	acquired := false
	if err := svc.clusterSvc.WithClusterLock(id, nil, func(_ *gorm.DB, item *cluster.Cluster) error {
		acquired = true
		if item.Name == "uncommitted-review-name" {
			t.Fatal("canceled transaction committed")
		}
		return nil
	}); err != nil {
		t.Fatalf("next writer cannot acquire cluster lock: %v", err)
	}
	if !acquired {
		t.Fatal("cluster lock was not released")
	}
	var item cluster.Cluster
	if err := db.First(&item, id).Error; err != nil {
		t.Fatal(err)
	}
	if item.Name == "uncommitted-review-name" {
		t.Fatal("canceled metadata survived")
	}
}
