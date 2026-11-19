package filedist

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"gorm.io/gorm"
)

func TestDistributionActiveCancellationIsParentScoped(t *testing.T) {
	engine := NewTransferEngine(nil, nil, 1)
	first, releaseFirst, ok := engine.beginTransfer(context.Background(), &TransferTask{ID: 1, TaskID: 10})
	if !ok {
		t.Fatal("first operation rejected")
	}
	defer releaseFirst()
	second, releaseSecond, ok := engine.beginTransfer(context.Background(), &TransferTask{ID: 2, TaskID: 20})
	if !ok {
		t.Fatal("second operation rejected")
	}
	defer releaseSecond()
	if _, _, ok := engine.beginTransfer(context.Background(), &TransferTask{ID: 1, TaskID: 10}); ok {
		t.Fatal("duplicate target execution accepted")
	}
	engine.CancelDistribution(10)
	if first.Err() != context.Canceled || second.Err() != nil {
		t.Fatal("cancellation escaped its parent or missed active operation")
	}
	engine.Stop()
	select {
	case <-second.Done():
	case <-time.After(time.Second):
		t.Fatal("shutdown did not cancel remaining active operation")
	}
	if err := engine.SubmitTask(&TransferTask{ID: 3}); err == nil {
		t.Fatal("stopped engine accepted new work")
	}
}

func TestDistributionContextReaderStopsAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader := &contextReader{ctx: ctx, reader: strings.NewReader("payload")}
	buffer := make([]byte, 1)
	if count, err := reader.Read(buffer); err != nil || count != 1 {
		t.Fatalf("initial read: %d %v", count, err)
	}
	cancel()
	if count, err := reader.Read(buffer); count != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled reader consumed more bytes: %d %v", count, err)
	}
}

func TestDistributionLeaseLossCancelsOwnedTransfers(t *testing.T) {
	engine := NewTransferEngine(nil, nil, 1)
	defer engine.Stop()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine.Start(ctx)
	operation, release, ok := engine.beginTransfer(ctx, &TransferTask{ID: 1, TaskID: 10})
	if !ok {
		t.Fatal("operation rejected")
	}
	defer release()
	done := make(chan struct{})
	go monitorDistributionLease(ctx, done, func(context.Context) error { return errors.New("lease-lost") }, cancel)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("lease loss did not stop dispatcher")
	}
	select {
	case <-operation.Done():
	case <-time.After(time.Second):
		t.Fatal("lease loss did not cancel owned transfer")
	}
}

func TestDistributionSFTPHandshakeHonorsCancellation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan struct{})
	stopServer := make(chan struct{})
	defer close(stopServer)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		defer connection.Close()
		close(accepted)
		<-stopServer // Deliberately never sends an SSH banner.
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	address := listener.Addr().(*net.TCPAddr)
	client := NewSFTPClient(SFTPConfig{Host: "127.0.0.1", Port: address.Port, HostFingerprint: "SHA256:fixture"})
	defer client.Close()
	result := make(chan error, 1)
	go func() { result <- client.ConnectContext(ctx) }()
	select {
	case <-accepted:
	case <-time.After(2 * time.Second):
		t.Fatal("local handshake fixture did not accept connection")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("handshake returned %v instead of cancellation", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled SSH handshake remained blocked")
	}
}

func newDistributionLifecycleFixture(t *testing.T) (*gorm.DB, *DistributionService, *TransferEngine, FileRecord) {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&FileStorage{}, &FileRecord{}, &DistributionTask{}, &DistributionTarget{}); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	storage := FileStorage{Name: "lifecycle", Type: "local", BasePath: root, Status: "active"}
	if err := db.Create(&storage).Error; err != nil {
		t.Fatal(err)
	}
	file := FileRecord{StorageID: storage.ID, Name: "payload.txt", Path: "payload.txt", Size: 7, MD5: "fixture", UploadedAt: time.Now()}
	if err := db.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "payload.txt"), []byte("payload"), 0600); err != nil {
		t.Fatal(err)
	}
	storageService := NewStorageService(db)
	engine := NewTransferEngine(db, storageService, 1)
	service := NewDistributionService(db, storageService, engine)
	t.Cleanup(service.Stop)
	return db, service, engine, file
}

func TestDistributionCancelStopsInFlightAndPreservesTerminalState(t *testing.T) {
	db, service, engine, file := newDistributionLifecycleFixture(t)
	task := DistributionTask{Name: "active", FileID: file.ID, Status: statusRunning}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	target := DistributionTarget{TaskID: task.ID, Host: "fixture", TargetPath: "/fixture", Status: statusPending}
	if err := db.Create(&target).Error; err != nil {
		t.Fatal(err)
	}
	started, finished := make(chan struct{}), make(chan struct{})
	engine.upload = func(ctx context.Context, _ SFTPConfig, _, _ string) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}
	go func() {
		defer close(finished)
		engine.processTask(context.Background(), &TransferTask{ID: target.ID, TaskID: task.ID, FileID: file.ID, MaxRetries: 5})
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("transfer did not start")
	}
	if err := service.CancelTask(task.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled upload did not terminate")
	}
	engine.finishTarget(target.ID, statusCompleted, 100, "") // Simulate a late successful callback.
	service.markTaskFailed(task.ID, "late-error")
	if err := db.First(&task, task.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&target, target.ID).Error; err != nil {
		t.Fatal(err)
	}
	if task.Status != statusCancelled || target.Status != statusCancelled || target.RetryCount != 0 || len(engine.taskQueue) != 0 {
		t.Fatal("cancelled task/target resurrected or retry was queued")
	}
	if err := service.CancelTask(task.ID); err != nil {
		t.Fatalf("cancellation replay: %v", err)
	}
}

func TestDistributionDueDispatchDoesNotStartFutureOrDuplicateTasks(t *testing.T) {
	db, service, engine, file := newDistributionLifecycleFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	now := time.Now()
	dueAt, futureAt := now.Add(-time.Minute), now.Add(time.Hour)
	due := DistributionTask{Name: "due", FileID: file.ID, Status: statusPending, ScheduledAt: &dueAt}
	future := DistributionTask{Name: "future", FileID: file.ID, Status: statusPending, ScheduledAt: &futureAt}
	for _, task := range []*DistributionTask{&due, &future} {
		if err := db.Create(task).Error; err != nil {
			t.Fatal(err)
		}
		target := DistributionTarget{TaskID: task.ID, Host: "fixture", TargetPath: "/fixture", Status: statusPending}
		if err := db.Create(&target).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := service.dispatchDueTasks(ctx, now); err != nil {
		t.Fatal(err)
	}
	if err := service.dispatchDueTasks(ctx, now); err != nil {
		t.Fatal(err)
	}
	if len(engine.taskQueue) != 1 {
		t.Fatalf("due dispatch queued %d operations", len(engine.taskQueue))
	}
	service.executeTask(future.ID) // Explicit dispatch must also obey the due fence.
	if err := service.StartTask(future.ID); err == nil {
		t.Fatal("manual start bypassed future schedule")
	}
	if err := db.First(&future, future.ID).Error; err != nil || future.Status != statusPending {
		t.Fatal("future task was changed before its due time")
	}
	queued := <-engine.taskQueue
	if queued.TaskID != due.ID || queued.FileID != file.ID {
		t.Fatal("queue lost persistent parent/file identity")
	}
	service.Stop()
}

func TestDistributionRecoveryFailsInterruptedWorkWithoutReplaying(t *testing.T) {
	db, service, engine, file := newDistributionLifecycleFixture(t)
	task := DistributionTask{Name: "interrupted", FileID: file.ID, Status: statusRunning}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	targets := []DistributionTarget{
		{TaskID: task.ID, Host: "completed", TargetPath: "/fixture", Status: statusCompleted, Progress: 100},
		{TaskID: task.ID, Host: "unfinished", TargetPath: "/fixture", Status: statusTransferring},
	}
	if err := db.Create(&targets).Error; err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := service.recoverInterruptedTasks(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.First(&task, task.ID).Error; err != nil || task.Status != statusFailed || task.Error != "business.filedist.task.executionInterrupted" || task.FinishedAt == nil {
		t.Fatal("interrupted task did not reach deterministic failure")
	}
	if err := db.Order("id ASC").Find(&targets).Error; err != nil {
		t.Fatal(err)
	}
	if targets[0].Status != statusCompleted || targets[0].Progress != 100 || targets[1].Status != statusFailed || len(engine.taskQueue) != 0 || task.SuccessCount != 1 || task.FailedCount != 1 || task.Progress != 100 {
		t.Fatal("recovery lost completed history or replayed ambiguous remote work")
	}
}

func TestDistributionLifecycleLeasePreventsConcurrentRecovery(t *testing.T) {
	db, service, _, _ := newDistributionLifecycleFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := service.Start(ctx); err != nil {
		t.Fatal(err)
	}
	otherEngine := NewTransferEngine(db, NewStorageService(db), 1)
	other := NewDistributionService(db, NewStorageService(db), otherEngine)
	t.Cleanup(other.Stop)
	if err := other.Start(ctx); err == nil {
		t.Fatal("second instance acquired lifecycle/recovery ownership")
	}
	service.Stop()
	if err := other.Start(ctx); err != nil {
		t.Fatalf("released lifecycle ownership remained held: %v", err)
	}
	other.Stop()
}

func TestDistributionLifecycleLeaseHealthDetectsReleasedOwnership(t *testing.T) {
	db, _, _, _ := newDistributionLifecycleFixture(t)
	release, check, err := acquireDistributionLease(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if err := check(context.Background()); err != nil {
		release()
		t.Fatalf("owned lease failed its health check: %v", err)
	}
	release()
	if err := check(context.Background()); err == nil {
		t.Fatal("closed lease connection remained healthy")
	}
}
