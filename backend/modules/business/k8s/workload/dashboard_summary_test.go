package workload

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	k8spkg "github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/pkg"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWorkloadDashboardBoundedSingleClusterProviderAndCancellation(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&cluster.Cluster{}, &cluster.ClusterCredentialRef{}); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	var invalidRequest atomic.Bool
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/apis/apps/v1/deployments" || r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("continue") != "" {
			invalidRequest.Store(true)
		}
		w.Header().Set("Content-Type", "application/json")
		replicas := int32(1)
		rows := []appsv1.Deployment{}
		for i := 0; i < 4; i++ {
			rows = append(rows, appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("api-%d", i), Namespace: "payments"},
				Spec: appsv1.DeploymentSpec{Replicas: &replicas, Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "api", Image: "secret-image-marker"}}}}}, Status: appsv1.DeploymentStatus{ReadyReplicas: 1}})
		}
		_ = json.NewEncoder(w).Encode(appsv1.DeploymentList{TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DeploymentList"}, ListMeta: metav1.ListMeta{Continue: "must-not-follow"}, Items: rows})
	}))
	defer provider.Close()
	t.Setenv("PANTHEON_K8S_KUBECONFIG_KEY", strings.Repeat("11", 32))
	encrypted, err := k8spkg.EncryptKubeconfig(fmt.Sprintf("apiVersion: v1\nkind: Config\nclusters:\n- name: test\n  cluster:\n    server: %s\ncontexts:\n- name: test\n  context:\n    cluster: test\n    user: test\ncurrent-context: test\nusers:\n- name: test\n  user:\n    token: local-test-marker\n", provider.URL))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		row := cluster.Cluster{Code: fmt.Sprintf("visible-%d", i), Name: "visible", Environment: "test", DeptID: 71, BusinessScopeID: 9, KubeconfigEncrypted: encrypted}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	hidden := cluster.Cluster{Code: "hidden", Name: "hidden", Environment: "test", DeptID: 72, KubeconfigEncrypted: "invalid-hidden-credential"}
	if err := db.Create(&hidden).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewWorkloadService(cluster.NewClusterService(db), nil)
	scope := &common.DataScopeReq{Mode: "dept", DeptID: 71}
	resp, err := svc.DashboardSummary(context.Background(), 2, scope)
	if err != nil || len(resp.Items) != 2 || resp.ClusterID != 2 || !resp.Partial || calls.Load() != 1 || invalidRequest.Load() {
		t.Fatalf("bounded provider: %+v err=%v calls=%d invalid=%v", resp, err, calls.Load(), invalidRequest.Load())
	}
	encoded, _ := json.Marshal(resp)
	if strings.Contains(string(encoded), "secret") || strings.Contains(string(encoded), "token") || strings.Contains(string(encoded), "images") {
		t.Fatal("unsafe workload summary")
	}
	if _, err := svc.DashboardSummary(context.Background(), 2, nil); err == nil {
		t.Fatal("nil scope admitted")
	}
	before := calls.Load()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.DashboardSummary(ctx, 2, scope); err == nil {
		t.Fatal("canceled query admitted")
	}
	if calls.Load() != before {
		t.Fatal("canceled request reached provider")
	}
	empty, err := svc.DashboardSummary(context.Background(), 2, &common.DataScopeReq{Mode: "dept", DeptID: 99})
	if err != nil || len(empty.Items) != 0 || empty.ClusterID != 0 || calls.Load() != before {
		t.Fatalf("empty visible scope: %+v %v", empty, err)
	}
	deadline, stop := context.WithTimeout(context.Background(), time.Nanosecond)
	defer stop()
	if _, err := svc.DashboardSummary(deadline, 2, scope); err == nil {
		t.Fatal("expired deadline admitted")
	}
}
