package cluster

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	k8spkg "github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/pkg"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCredentialReferenceFailsClosedWithInline(t *testing.T) {
	for _, cached := range []bool{false, true} {
		for _, state := range []string{"revoked", "missing", "empty", "foreign"} {
			t.Run(state+map[bool]string{false: "_lazy", true: "_cached"}[cached], func(t *testing.T) {
				t.Setenv("PANTHEON_K8S_KUBECONFIG_KEY", strings.Repeat("12", 32))
				db := testmysql.Open(t).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
				svc := NewClusterService(db)
				if err := svc.Migrate(); err != nil {
					t.Fatal(err)
				}
				encrypted, err := k8spkg.EncryptKubeconfig("apiVersion: v1\nkind: Config\nclusters:\n- name: test\n  cluster:\n    server: https://115.190.251.143:6443\ncontexts:\n- name: test\n  context:\n    cluster: test\n    user: empty\ncurrent-context: test\nusers:\n- name: empty\n  user: {}\n")
				if err != nil {
					t.Fatal(err)
				}
				target := Cluster{Code: "dual", Name: "dual", DeptID: 10, KubeconfigEncrypted: encrypted}
				if err := db.Create(&target).Error; err != nil {
					t.Fatal(err)
				}
				ref := ClusterCredentialRef{ClusterID: target.ID, Encrypted: encrypted, Version: 1, Status: "active"}
				if err := db.Create(&ref).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.Model(&target).Update("kubeconfig_credential_ref_id", ref.ID).Error; err != nil {
					t.Fatal(err)
				}
				scope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 10}
				if cached {
					if _, err := svc.GetClientsetWithContext(context.Background(), target.ID, scope); err != nil {
						t.Fatal(err)
					}
					if _, err := svc.GetDynamicClientWithContext(context.Background(), target.ID, scope); err != nil {
						t.Fatal(err)
					}
				}
				switch state {
				case "revoked":
					err = db.Model(&ref).Update("status", "revoked").Error
				case "missing":
					err = db.Delete(&ref).Error
				case "empty":
					err = db.Model(&ref).Update("encrypted", "").Error
				case "foreign":
					err = db.Model(&ref).Update("cluster_id", target.ID+1).Error
				}
				if err != nil {
					t.Fatal(err)
				}
				if _, err := svc.GetClientsetWithContext(context.Background(), target.ID, scope); err == nil {
					t.Error("native client fell back to inline after explicit reference failure")
				}
				if _, err := svc.GetDynamicClientWithContext(context.Background(), target.ID, scope); err == nil {
					t.Error("dynamic client fell back to inline after explicit reference failure")
				}
				// Clearing the reference is the sole legacy path, and requires an explicit write.
				if err := db.Model(&target).Update("kubeconfig_credential_ref_id", 0).Error; err != nil {
					t.Fatal(err)
				}
				if _, err := svc.GetClientsetWithContext(context.Background(), target.ID, scope); err != nil {
					t.Fatal(err)
				}
				if _, err := svc.GetDynamicClientWithContext(context.Background(), target.ID, scope); err != nil {
					t.Fatal(err)
				}
				for _, want := range []error{context.Canceled, context.DeadlineExceeded} {
					var ctx context.Context
					var cancel context.CancelFunc
					if want == context.Canceled {
						ctx, cancel = context.WithCancel(context.Background())
						cancel()
					} else {
						ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
					}
					if _, err := svc.GetClientsetWithContext(ctx, target.ID, scope); !errors.Is(err, want) {
						t.Errorf("native cancellation: %v", err)
					}
					if _, err := svc.GetDynamicClientWithContext(ctx, target.ID, scope); !errors.Is(err, want) {
						t.Errorf("dynamic cancellation: %v", err)
					}
					cancel()
				}
			})
		}
	}
}
