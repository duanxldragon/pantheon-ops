package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/namespace"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// The independent NOWAIT transactions verify actual MySQL row-lock ownership,
// not a pre-query callback or a scheduling delay. Provider is simulated only.
func TestK8sServiceMutationHoldsAdmissionLocksThroughProvider(t *testing.T) {
	for _, operation := range []string{"create", "update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			f := newServiceMutationFixture(t)
			gate := &serviceProviderReadGate{entered: make(chan struct{}), exited: make(chan struct{}), release: make(chan struct{})}
			f.provider.mutationGate = gate
			finished := make(chan struct{})
			var mutationError error
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(func() {
				gate.releaseOnce.Do(func() { close(gate.release) })
				cancel()
				select {
				case <-finished:
				case <-time.After(5 * time.Second):
					t.Error("mutation worker did not finish during cleanup")
				}
			})
			go func() {
				mutationError = runAdmissionMutation(ctx, f, operation)
				close(finished)
			}()
			waitServiceMutationSignal(t, gate.entered, "provider mutation entry")
			for _, owner := range admissionOwnerRows(f) {
				err := tryAdmissionRowLock(f.db, owner.model, owner.id)
				var driverError *mysql.MySQLError
				if !errors.As(err, &driverError) || driverError.Number != 3572 {
					t.Fatalf("%s owner row not protected during provider %s: %v", owner.name, operation, err)
				}
			}
			gate.releaseOnce.Do(func() { close(gate.release) })
			select {
			case <-finished:
				if mutationError != nil {
					t.Fatalf("allowed %s failed: %v", operation, mutationError)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("mutation did not complete after provider release")
			}
			for _, owner := range admissionOwnerRows(f) {
				if err := tryAdmissionRowLock(f.db, owner.model, owner.id); err != nil {
					t.Fatalf("%s lock not released after %s: %v", owner.name, operation, err)
				}
			}
			f.provider.mu.Lock()
			defer f.provider.mu.Unlock()
			if f.provider.mutations != 1 {
				t.Fatalf("allowed %s mutation count=%d", operation, f.provider.mutations)
			}
		})
	}
}

func TestK8sServiceMutationAdmissionLocksReleaseOnCancellation(t *testing.T) {
	for _, operation := range []string{"create", "update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			f := newServiceMutationFixture(t)
			gate := &serviceProviderReadGate{entered: make(chan struct{}), exited: make(chan struct{}), release: make(chan struct{})}
			f.provider.mutationGate = gate
			ctx, cancel := context.WithCancel(context.Background())
			finished := make(chan struct{})
			var mutationError error
			go func() {
				mutationError = runAdmissionMutation(ctx, f, operation)
				close(finished)
			}()
			t.Cleanup(func() {
				cancel()
				gate.releaseOnce.Do(func() { close(gate.release) })
				waitServiceMutationSignal(t, finished, "canceled mutation cleanup")
			})
			waitServiceMutationSignal(t, gate.entered, "provider mutation entry")
			cancel()
			waitServiceMutationSignal(t, finished, "canceled mutation completion")
			waitServiceMutationSignal(t, gate.exited, "canceled provider mutation exit")
			if !errors.Is(mutationError, context.Canceled) {
				t.Fatalf("%s cancellation cause lost: %v", operation, mutationError)
			}
			for _, owner := range admissionOwnerRows(f) {
				waitAdmissionRowUnlock(t, f.db, owner.model, owner.id)
			}
			f.provider.mu.Lock()
			defer f.provider.mu.Unlock()
			if f.provider.mutations != 0 || f.provider.object.ResourceVersion != "13" {
				t.Fatalf("canceled %s committed fixture mutation: mutations=%d revision=%s", operation, f.provider.mutations, f.provider.object.ResourceVersion)
			}
		})
	}
}

func TestK8sServiceMutationAdmissionLocksReleaseOnProviderError(t *testing.T) {
	f := newServiceMutationFixture(t)
	f.provider.readStatus = http.StatusForbidden
	err := runAdmissionMutation(context.Background(), f, "update")
	if !errors.Is(err, common.ErrForbidden) || !apierrors.IsForbidden(err) {
		t.Fatalf("provider rejection cause lost: %v", err)
	}
	for _, owner := range admissionOwnerRows(f) {
		if err := tryAdmissionRowLock(f.db, owner.model, owner.id); err != nil {
			t.Fatalf("%s owner lock retained after provider error: %v", owner.name, err)
		}
	}
	if f.provider.requests != 1 || f.provider.mutations != 0 {
		t.Fatalf("provider rejection control: requests=%d mutations=%d", f.provider.requests, f.provider.mutations)
	}
}

func TestK8sServiceMutationAdmissionLockWaitHonorsDeadline(t *testing.T) {
	for _, ownerName := range []string{"cluster", "binding"} {
		t.Run(ownerName, func(t *testing.T) {
			f := newServiceMutationFixture(t)
			var ownerModel any
			var ownerID uint64
			for _, owner := range admissionOwnerRows(f) {
				if owner.name == ownerName {
					ownerModel, ownerID = owner.model, owner.id
				}
			}
			holder := f.db.Begin()
			if holder.Error != nil {
				t.Fatal(holder.Error)
			}
			t.Cleanup(func() { _ = holder.Rollback().Error })
			if err := holder.Model(ownerModel).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", ownerID).First(ownerModel).Error; err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
			defer cancel()
			err := runAdmissionMutation(ctx, f, "create")
			if !errors.Is(err, context.DeadlineExceeded) || f.provider.requests != 0 || f.provider.mutations != 0 {
				t.Fatalf("blocked %s admission ignored deadline: err=%v requests=%d mutations=%d", ownerName, err, f.provider.requests, f.provider.mutations)
			}
			if err := holder.Rollback().Error; err != nil {
				t.Fatal(err)
			}
			for _, owner := range admissionOwnerRows(f) {
				waitAdmissionRowUnlock(t, f.db, owner.model, owner.id)
			}
			if err := runAdmissionMutation(context.Background(), f, "create"); err != nil {
				t.Fatalf("new request failed after canceled admission: %v", err)
			}
			if f.provider.mutations != 1 {
				t.Fatalf("post-deadline control mutations=%d", f.provider.mutations)
			}
		})
	}
}

func TestK8sServiceMutationAdmissionUsesSingleConnection(t *testing.T) {
	f := newServiceMutationFixture(t)
	if err := f.db.AutoMigrate(&cluster.ClusterCredentialRef{}); err != nil {
		t.Fatal(err)
	}
	ref := cluster.ClusterCredentialRef{ClusterID: f.cluster.ID, Encrypted: f.cluster.KubeconfigEncrypted, Version: 1, Status: "active"}
	if err := f.db.Create(&ref).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Model(&cluster.Cluster{}).Where("id = ?", f.cluster.ID).Updates(map[string]any{"kubeconfig_credential_ref_id": ref.ID, "kubeconfig_encrypted": ""}).Error; err != nil {
		t.Fatal(err)
	}
	connection, err := f.db.DB()
	if err != nil {
		t.Fatal(err)
	}
	connection.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := runAdmissionMutation(ctx, f, "create"); err != nil {
		t.Fatalf("credential/binding reads need an extra connection while holding the owner transaction: %v", err)
	}
	if f.provider.requests != 1 || f.provider.mutations != 1 {
		t.Fatalf("single-connection allowed control: requests=%d mutations=%d", f.provider.requests, f.provider.mutations)
	}
}

func runAdmissionMutation(ctx context.Context, f *serviceMutationFixture, operation string) error {
	ports := []ServicePort{{Port: 80, TargetPort: intstr.FromInt(8080), Protocol: "TCP"}}
	switch operation {
	case "create":
		_, err := f.manager.CreateWithContext(ctx, f.cluster.ID, "owned", CreateServiceRequest{Name: "new-api", Ports: ports}, nil)
		return err
	case "update":
		_, err := f.manager.UpdateWithContext(ctx, f.cluster.ID, "owned", "catalog-api", UpdateServiceRequest{Ports: ports, ResourceVersion: "13", UID: "original-uid"}, nil)
		return err
	case "delete":
		return f.manager.DeleteWithResourceVersionAndContext(ctx, f.cluster.ID, "owned", "catalog-api", "13", "original-uid", nil)
	default:
		return fmt.Errorf("unknown fixture operation %q", operation)
	}
}

func admissionOwnerRows(f *serviceMutationFixture) []struct {
	name  string
	model any
	id    uint64
} {
	return []struct {
		name  string
		model any
		id    uint64
	}{{"cluster", &cluster.Cluster{}, f.cluster.ID}, {"binding", &namespace.NamespaceBinding{}, f.binding.ID}}
}

func tryAdmissionRowLock(db *gorm.DB, model any, id uint64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Model(model).Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).Where("id = ?", id).First(model).Error
	})
}

func waitAdmissionRowUnlock(t *testing.T, db *gorm.DB, model any, id uint64) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := tryAdmissionRowLock(db, model, id)
		if err == nil {
			return
		}
		var driverError *mysql.MySQLError
		if !errors.As(err, &driverError) || driverError.Number != 3572 || time.Now().After(deadline) {
			t.Fatalf("admission row lock not released: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
