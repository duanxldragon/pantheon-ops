package cluster

import (
	"fmt"
	"strings"
	"testing"
	"time"

	k8spkg "github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/pkg"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"gorm.io/gorm"
)

const clusterTestKubeconfigKey = "PANTHEON_K8S_KUBECONFIG_KEY"

// clusterTestKubeconfig builds a parseable kubeconfig for a public API
// endpoint. Clients are constructed but never dialed, so no server exists.
func clusterTestKubeconfig(token string) string {
	return fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: http://8.8.8.8:6443
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test
users:
- name: test
  user:
    token: %s
`, token)
}

// rotationFixture creates an isolated MySQL database with one cluster whose
// active credential is the fixture kubeconfig. Testmysql skips the test when no
// DSN is configured; in that case service is nil.
func rotationFixture(t *testing.T) (*ClusterService, *gorm.DB, *Cluster) {
	t.Helper()
	t.Setenv(clusterTestKubeconfigKey, strings.Repeat("a1", 32))
	db := testmysql.Open(t)
	if db == nil {
		return nil, nil, nil
	}
	if err := db.AutoMigrate(&Cluster{}, &ClusterCredentialRef{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	encrypted, err := k8spkg.EncryptKubeconfig(clusterTestKubeconfig("fixture-admin-token"))
	if err != nil {
		t.Fatalf("encrypt fixture kubeconfig: %v", err)
	}
	cluster := Cluster{Code: "rot-fixture", Name: "Rotation Fixture", Environment: "test", Status: "unknown"}
	if err := db.Create(&cluster).Error; err != nil {
		t.Fatalf("create cluster: %v", err)
	}
	ref := ClusterCredentialRef{ClusterID: cluster.ID, Encrypted: encrypted, Version: 1, Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := db.Create(&ref).Error; err != nil {
		t.Fatalf("create credential ref: %v", err)
	}
	cluster.KubeconfigCredentialRefID = ref.ID
	if err := db.Model(&cluster).Update("kubeconfig_credential_ref_id", ref.ID).Error; err != nil {
		t.Fatalf("link credential ref: %v", err)
	}
	return NewClusterService(db), db, &cluster
}

func TestRotateCredentialRebuildsPooledClientset(t *testing.T) {
	service, _, cluster := rotationFixture(t)
	if service == nil {
		return
	}

	oldClientset, err := service.GetClientset(cluster.ID, nil)
	if err != nil {
		t.Fatalf("warm clientset: %v", err)
	}
	sameClientset, err := service.GetClientset(cluster.ID, nil)
	if err != nil {
		t.Fatalf("cached clientset: %v", err)
	}
	if sameClientset != oldClientset {
		t.Fatal("expected the same warmed client for an unchanged credential")
	}

	rotated, err := service.RotateCredential(cluster.ID, RotateClusterCredentialRequest{Kubeconfig: clusterTestKubeconfig("rotated-invalid-token")}, "operator", nil)
	if err != nil {
		t.Fatalf("rotate credential: %v", err)
	}
	if rotated.Version != 2 || rotated.Status != "active" {
		t.Fatalf("unexpected rotated credential metadata: %+v", rotated)
	}

	newClientset, err := service.GetClientset(cluster.ID, nil)
	if err != nil {
		t.Fatalf("clientset after rotation: %v", err)
	}
	if newClientset == oldClientset {
		t.Fatal("old warmed clientset reused after credential rotation")
	}

	credential, err := service.GetCredential(cluster.ID, nil)
	if err != nil {
		t.Fatalf("get credential: %v", err)
	}
	if credential.Version != 2 {
		t.Fatalf("expected credential version 2 after rotation, got %d", credential.Version)
	}
}

func TestUpdateKubeconfigRotatesAndRebuildsPooledClientset(t *testing.T) {
	service, _, cluster := rotationFixture(t)
	if service == nil {
		return
	}

	oldClientset, err := service.GetClientset(cluster.ID, nil)
	if err != nil {
		t.Fatalf("warm clientset: %v", err)
	}

	newKubeconfig := clusterTestKubeconfig("updated-token")
	updated, err := service.Update(cluster.ID, UpdateClusterRequest{Kubeconfig: &newKubeconfig}, "updater", nil)
	if err != nil {
		t.Fatalf("update cluster: %v", err)
	}
	if updated.CredentialVersion != 2 || updated.CredentialStatus != "active" {
		t.Fatalf("unexpected update credential metadata: %+v", updated)
	}

	newClientset, err := service.GetClientset(cluster.ID, nil)
	if err != nil {
		t.Fatalf("clientset after update: %v", err)
	}
	if newClientset == oldClientset {
		t.Fatal("old warmed clientset reused after kubeconfig update")
	}
}

func TestFailedUpdatePreservesCredentialAndWarmClientset(t *testing.T) {
	service, _, cluster := rotationFixture(t)
	if service == nil {
		return
	}

	oldClientset, err := service.GetClientset(cluster.ID, nil)
	if err != nil {
		t.Fatalf("warm clientset: %v", err)
	}

	// The transaction fails on the unresolvable business scope before the new
	// credential is stored; it must roll back without touching the credential
	// or the warmed cache.
	badScope := uint64(987654)
	newKubeconfig := clusterTestKubeconfig("must-not-persist-token")
	if _, err := service.Update(cluster.ID, UpdateClusterRequest{BusinessScopeID: &badScope, Kubeconfig: &newKubeconfig}, "updater", nil); err == nil {
		t.Fatal("expected update to fail on unresolvable business scope")
	}

	credential, err := service.GetCredential(cluster.ID, nil)
	if err != nil {
		t.Fatalf("get credential: %v", err)
	}
	if credential.Version != 1 {
		t.Fatalf("expected credential version to stay 1 after failed update, got %d", credential.Version)
	}

	again, err := service.GetClientset(cluster.ID, nil)
	if err != nil {
		t.Fatalf("clientset after failed update: %v", err)
	}
	if again != oldClientset {
		t.Fatal("warm client should survive a failed credential update transaction")
	}
}

func TestCreatePopulatesCredentialMetadata(t *testing.T) {
	t.Setenv(clusterTestKubeconfigKey, strings.Repeat("a1", 32))
	db := testmysql.Open(t)
	if db == nil {
		return
	}
	if err := db.AutoMigrate(&Cluster{}, &ClusterCredentialRef{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	service := NewClusterService(db)

	created, err := service.Create(CreateClusterRequest{
		Code:        "k03b-create",
		Name:        "K03B",
		Environment: "test",
		Kubeconfig:  clusterTestKubeconfig("create-token"),
	}, "creator", 0)
	if err != nil {
		t.Fatalf("create cluster: %v", err)
	}
	if created.CredentialRefID == 0 {
		t.Fatal("expected create response to include credentialRefId")
	}
	if created.CredentialVersion != 1 || created.CredentialStatus != "active" {
		t.Fatalf("expected credential version 1/active in create response, got %d/%q", created.CredentialVersion, created.CredentialStatus)
	}

	reloaded, err := service.GetByID(created.ID, nil)
	if err != nil {
		t.Fatalf("get created cluster: %v", err)
	}
	if reloaded.CredentialVersion != created.CredentialVersion || reloaded.CredentialStatus != created.CredentialStatus || reloaded.CredentialRefID != created.CredentialRefID {
		t.Fatalf("create response credential metadata diverged from get: create=%+v get=%+v", created, reloaded)
	}
}
