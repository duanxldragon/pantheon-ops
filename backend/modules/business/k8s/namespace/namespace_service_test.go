package namespace

import (
	"context"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestToNamespaceItem(t *testing.T) {
	created := time.Now()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "default",
			Labels:            map[string]string{"env": "prod"},
			CreationTimestamp: metav1.NewTime(created),
		},
		Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
	}

	item := toNamespaceItem(ns)
	if item.Name != "default" || item.Status != "Active" {
		t.Fatalf("unexpected item: %+v", item)
	}
	if item.Labels["env"] != "prod" {
		t.Fatalf("expected labels, got %v", item.Labels)
	}
	if item.CreationTimestamp == "" {
		t.Fatal("expected creation timestamp to be formatted")
	}
}

func TestReadableNamespacesFollowsClusterOwnership(t *testing.T) {
	db := testmysql.Open(t)
	clusterSvc := cluster.NewClusterService(db)
	if err := clusterSvc.Migrate(); err != nil {
		t.Fatal(err)
	}
	svc := NewNamespaceService(clusterSvc, db)
	if err := svc.Migrate(); err != nil {
		t.Fatal(err)
	}

	owned := cluster.Cluster{Code: "readable-owned", Name: "owned", Environment: "prod", BusinessScopeID: 7, Status: "ready"}
	if err := db.Create(&owned).Error; err != nil {
		t.Fatal(err)
	}
	for _, binding := range []NamespaceBinding{
		{ClusterID: owned.ID, Namespace: "ns-prod", BusinessScopeID: 7, Environment: "prod"},
		{ClusterID: owned.ID, Namespace: "ns-other-scope", BusinessScopeID: 8, Environment: "prod"},
		{ClusterID: owned.ID, Namespace: "ns-other-env", BusinessScopeID: 7, Environment: "dev"},
	} {
		if err := db.Create(&binding).Error; err != nil {
			t.Fatal(err)
		}
	}

	readable, err := svc.ReadableNamespaces(context.Background(), owned.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(readable) != 1 {
		t.Fatalf("expected one readable namespace, got %v", readable)
	}
	if _, ok := readable["ns-prod"]; !ok {
		t.Fatalf("owned namespace missing from %v", readable)
	}

	if err := svc.RequireReadForClusterWithContext(context.Background(), owned.ID, "ns-prod", nil); err != nil {
		t.Fatalf("owned namespace must be readable: %v", err)
	}
	if err := svc.RequireReadForClusterWithContext(context.Background(), owned.ID, "ns-other-scope", nil); err == nil {
		t.Fatal("foreign business scope must not be readable")
	}

	unowned := cluster.Cluster{Code: "readable-unowned", Name: "unowned", Environment: "prod", BusinessScopeID: 0, Status: "ready"}
	if err := db.Create(&unowned).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&NamespaceBinding{ClusterID: unowned.ID, Namespace: "ns-unowned", BusinessScopeID: 0, Environment: "prod"}).Error; err != nil {
		t.Fatal(err)
	}
	empty, err := svc.ReadableNamespaces(context.Background(), unowned.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("cluster without an ownership scope must not expose namespaces: %v", empty)
	}
}
