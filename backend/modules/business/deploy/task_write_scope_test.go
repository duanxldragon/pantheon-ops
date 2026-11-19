package deploy

import (
	"context"
	"reflect"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
)

func TestTaskWritesRejectOutOfScopeWithoutMutation(t *testing.T) {
	for _, operation := range []string{"update", "cancel", "result"} {
		t.Run(operation, func(t *testing.T) {
			db, svc, task, hostID := previewFixture(t, false)
			if err := db.Table("biz_cmdb_host").Where("id = ?", hostID).Update("dept_id", 20).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Table("biz_business_scope").Where("id = ?", task.BusinessScopeID).Update("dept_id", 20).Error; err != nil {
				t.Fatal(err)
			}
			allowed := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 20}
			denied := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 999}
			if operation != "update" {
				var err error
				task, err = svc.StartTask(task.ID, StartTaskRequest{}, "1", allowed)
				if err != nil {
					t.Fatal(err)
				}
			}
			var beforeTask DeployTask
			if err := db.First(&beforeTask, task.ID).Error; err != nil {
				t.Fatal(err)
			}
			var beforeHost DeployTaskHost
			var beforeHosts []DeployTaskHost
			if err := db.Where("task_id = ?", task.ID).Order("id").Find(&beforeHosts).Error; err != nil {
				t.Fatal(err)
			}
			if operation != "update" {
				if len(beforeHosts) != 1 {
					t.Fatalf("expected one writable host, got %d", len(beforeHosts))
				}
				beforeHost = beforeHosts[0]
			}
			var beforeEvents int64
			if err := db.Model(&DeployTaskEvent{}).Where("task_id = ?", task.ID).Count(&beforeEvents).Error; err != nil {
				t.Fatal(err)
			}
			name := "scoped rename"
			run := func(scope *common.DataScopeReq) error {
				switch operation {
				case "update":
					_, err := svc.UpdateTaskContext(context.Background(), task.ID, UpdateTaskRequest{Name: &name}, "1", scope)
					return err
				case "cancel":
					_, err := svc.ControlTask(context.Background(), task.ID, "1", scope, "cancel", TaskCommandRequest{StateVersion: task.StateVersion})
					return err
				default:
					_, err := svc.MarkHostResult(beforeHost.ID, MarkHostResultRequest{Status: TaskHostStatusFailed, ErrorMessage: "scoped result"}, "1", scope)
					return err
				}
			}
			if err := run(denied); !isDeployTaskForbidden(err) {
				t.Fatalf("out-of-scope write must return forbidden: %v", err)
			}
			var afterTask DeployTask
			var afterHost DeployTaskHost
			var afterHosts []DeployTaskHost
			if err := db.First(&afterTask, task.ID).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Where("task_id = ?", task.ID).Order("id").Find(&afterHosts).Error; err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(beforeTask, afterTask) || !reflect.DeepEqual(beforeHosts, afterHosts) {
				t.Fatal("rejected write changed task or host execution state")
			}
			var events int64
			if err := db.Model(&DeployTaskEvent{}).Where("task_id = ?", task.ID).Count(&events).Error; err != nil {
				t.Fatal(err)
			}
			if events != beforeEvents {
				t.Fatalf("rejected write changed task events from %d to %d", beforeEvents, events)
			}
			if err := run(allowed); err != nil {
				t.Fatalf("same valid request must succeed in scope: %v", err)
			}
			if err := db.First(&afterTask, task.ID).Error; err != nil {
				t.Fatal(err)
			}
			if operation == "update" && afterTask.Name != name {
				t.Fatalf("authorized update did not rename task: %s", afterTask.Name)
			}
			if operation == "cancel" && afterTask.Status != TaskStatusCanceled {
				t.Fatalf("authorized cancel did not cancel task: %s", afterTask.Status)
			}
			if operation == "result" {
				if err := db.First(&afterHost, beforeHost.ID).Error; err != nil {
					t.Fatal(err)
				}
				if afterHost.Status != TaskHostStatusFailed || afterHost.ErrorMessage != "scoped result" {
					t.Fatal("authorized result did not persist host failure")
				}
			}
		})
	}
}
