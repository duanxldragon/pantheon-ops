package deploy

import (
 "context"
 "encoding/json"
 "errors"
 "testing"
 "gorm.io/gorm"
)

// Independent review: never alter a foreign physical lease or create a reservation
// when settling an admitted task that has no attempt.
func TestIndependentCurrentO2CancellationPhysicalBoundary(t *testing.T) {
 for _, mode := range []string{"no-lease", "foreign-lease", "delete-error"} {
  t.Run(mode, func(t *testing.T) {
   db, a, b, task, host := ownershipFixture(t)
   if mode != "no-lease" {
    ownerID := task.ID
    if mode == "foreign-lease" { ownerID += 100 }
    if err := a.deploymentTransaction(func(tx *gorm.DB) error {
     return acquireHostLease(tx, host.HostID, ownerID, deployHostLeaseOwner(ownerID), task.CreatedAt)
    }); err != nil { t.Fatal(err) }
   }
   leaseFacts := func() string {
    var leases []DeployHostLease
    if err := db.Order("host_id").Find(&leases).Error; err != nil { t.Fatal(err) }
    raw, err := json.Marshal(leases)
    if err != nil { t.Fatal(err) }
    return string(raw)
   }
   originalLeases := leaseFacts()
   current, err := a.GetTask(task.ID, nil)
   if err != nil { t.Fatal(err) }
   if _, err := a.ControlTask(context.Background(), task.ID, "independent-current-code-review", nil, "cancel", TaskCommandRequest{StateVersion: current.StateVersion}); err != nil { t.Fatal(err) }
   if mode == "delete-error" {
    before := ownershipFacts(t, db)
    injected := errors.New("independent physical release failure")
    if err := db.Callback().Delete().After("gorm:delete").Register("independent_physical_delete_error", func(tx *gorm.DB) {
     if tx.Statement.Table == "biz_deploy_host_lease" { tx.AddError(injected) }
    }); err != nil { t.Fatal(err) }
    _, err := b.ReconcileDeployQueue("independent-current-code-review")
    db.Callback().Delete().Remove("independent_physical_delete_error")
    if !errors.Is(err, injected) { t.Fatalf("unexpected recovery failure: %v", err) }
    if ownershipFacts(t, db) != before { t.Fatal("physical release error committed partial task/host/event/lease changes") }
   }
   if _, err := b.ReconcileDeployQueue("independent-current-code-review"); err != nil { t.Fatal(err) }
   if err := db.First(&task, task.ID).Error; err != nil { t.Fatal(err) }
   if err := db.First(&host, host.ID).Error; err != nil { t.Fatal(err) }
   if task.Status != TaskStatusCanceled || task.FinishedAt == nil || host.Status != TaskHostStatusSkipped || host.FinishedAt == nil { t.Fatal("unclaimed cancellation did not settle") }
   var attempts, events int64
   if err := db.Model(&DeployTaskAttempt{}).Where("task_id = ?", task.ID).Count(&attempts).Error; err != nil { t.Fatal(err) }
   if err := db.Model(&DeployTaskEvent{}).Where("task_id = ? AND event_type = ?", task.ID, "task.canceled").Count(&events).Error; err != nil { t.Fatal(err) }
   if attempts != 0 || events != 1 { t.Fatalf("unexpected new attempt or duplicated cancel: attempts=%d events=%d", attempts, events) }
   if mode == "foreign-lease" {
    if leaseFacts() != originalLeases { t.Fatal("unclaimed cancellation altered another task's physical lease") }
   } else {
    var reservations int64
    if err := db.Model(&DeployHostLease{}).Count(&reservations).Error; err != nil { t.Fatal(err) }
    if reservations != 0 { t.Fatalf("unclaimed cancellation leaked physical lease/reservation: %d", reservations) }
   }
   settled := ownershipFacts(t, db)
   if _, err := a.ReconcileDeployQueue("independent-current-code-review"); err != nil { t.Fatal(err) }
   if ownershipFacts(t, db) != settled { t.Fatal("settled recovery replay changed durable facts") }
  })
 }
}
