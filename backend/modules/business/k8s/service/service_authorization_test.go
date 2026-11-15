package service

import (
	"context"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	ns "github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/namespace"
	k8spkg "github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/pkg"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

func TestServiceMutationAuthorizationBeforeProvider(t *testing.T) {
	t.Setenv("PANTHEON_K8S_KUBECONFIG_KEY", strings.Repeat("12", 32))
	db := testmysql.Open(t)
	clusterSvc := cluster.NewClusterService(db)
	if err := clusterSvc.Migrate(); err != nil {
		t.Fatal(err)
	}
	namespaceSvc := ns.NewNamespaceService(clusterSvc, db)
	if err := namespaceSvc.Migrate(); err != nil {
		t.Fatal(err)
	}
	// Client construction is lazy: these negative cases never dial this public fixture endpoint.
	encrypted, err := k8spkg.EncryptKubeconfig("apiVersion: v1\nkind: Config\nclusters:\n- name: test\n  cluster:\n    server: https://8.8.8.8:6443\ncontexts:\n- name: test\n  context:\n    cluster: test\n    user: test\ncurrent-context: test\nusers:\n- name: test\n  user:\n    token: fixture\n")
	if err != nil {
		t.Fatal(err)
	}
	target := cluster.Cluster{Code: "service-cas", Name: "Service CAS", DeptID: 10, KubeconfigEncrypted: encrypted}
	if err := db.Create(&target).Error; err != nil {
		t.Fatal(err)
	}
	binding := ns.NamespaceBinding{ClusterID: target.ID, Namespace: "test", BusinessScopeID: 1, Environment: "test", AllowedActions: `["service:delete"]`}
	if err := db.Create(&binding).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewServiceService(clusterSvc, namespaceSvc)
	ownScope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 10}
	deniedScope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 20}
	t.Run("foreign department cannot inspect provider", func(t *testing.T) {
		_, err := svc.UpdateContext(context.Background(), target.ID, "test", "app", reviewedServiceUpdate(), deniedScope)
		if err == nil || err.Error() != "k8s.cluster.not_found" {
			t.Fatalf("expected scoped denial: %v", err)
		}
	})
	t.Run("unbound namespace cannot mutate", func(t *testing.T) {
		_, err := svc.UpdateContext(context.Background(), target.ID, "unbound", "app", reviewedServiceUpdate(), ownScope)
		if err == nil || err.Error() != "k8s.namespace.binding_required" {
			t.Fatalf("expected namespace denial: %v", err)
		}
	})
	t.Run("delete grant does not confer update", func(t *testing.T) {
		_, err := svc.UpdateContext(context.Background(), target.ID, "test", "app", reviewedServiceUpdate(), ownScope)
		if err == nil || err.Error() != "k8s.namespace.action_forbidden" {
			t.Fatalf("expected action denial: %v", err)
		}
	})
	t.Run("pre-canceled scope read stays canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := svc.UpdateContext(ctx, target.ID, "test", "app", reviewedServiceUpdate(), ownScope)
		if err != context.Canceled {
			t.Fatalf("expected cancellation: %v", err)
		}
	})
}
