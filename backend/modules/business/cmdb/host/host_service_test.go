package host

import (
	"strings"
	"testing"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/testmysql"

	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&Host{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestCreateHost(t *testing.T) {
	db := setupTestDB(t)
	svc := NewHostService(db)

	resp, err := svc.Create(CreateHostRequest{
		Hostname: "test-host",
		IP:       "192.168.1.1",
		OS:       "linux",
		SSHPort:  22,
	}, "1")
	if err != nil {
		t.Fatalf("create host: %v", err)
	}
	if resp.Hostname != "test-host" {
		t.Errorf("expected hostname test-host, got %s", resp.Hostname)
	}
	if resp.Status != "pending" {
		t.Errorf("expected status pending, got %s", resp.Status)
	}
}

func TestCreateHostDuplicateIP(t *testing.T) {
	db := setupTestDB(t)
	svc := NewHostService(db)

	svc.Create(CreateHostRequest{Hostname: "h1", IP: "10.0.0.1", OS: "linux"}, "1")
	_, err := svc.Create(CreateHostRequest{Hostname: "h2", IP: "10.0.0.1", OS: "linux"}, "1")
	if err == nil {
		t.Error("expected duplicate IP error")
	}
}

func TestListHosts(t *testing.T) {
	db := setupTestDB(t)
	svc := NewHostService(db)
	svc.Create(CreateHostRequest{Hostname: "h1", IP: "10.0.0.10", OS: "linux"}, "1")
	svc.Create(CreateHostRequest{Hostname: "h2", IP: "10.0.0.20", OS: "windows"}, "1")

	resp, err := svc.List(HostListQuery{Page: 1, PageSize: 10}, nil)
	if err != nil {
		t.Fatalf("list hosts: %v", err)
	}
	if resp.Total < 2 {
		t.Errorf("expected at least 2 hosts, got %d", resp.Total)
	}
}

func TestListHostsAppliesDeptAndChildrenDataScope(t *testing.T) {
	db := setupTestDB(t)
	svc := NewHostService(db)

	dept10 := uint64(10)
	dept11 := uint64(11)
	dept20 := uint64(20)
	if _, err := svc.Create(CreateHostRequest{Hostname: "dept-10", IP: "10.10.0.10", OS: "linux", DeptID: dept10}, "1"); err != nil {
		t.Fatalf("seed dept 10 host: %v", err)
	}
	if _, err := svc.Create(CreateHostRequest{Hostname: "dept-11", IP: "10.10.0.11", OS: "linux", DeptID: dept11}, "1"); err != nil {
		t.Fatalf("seed dept 11 host: %v", err)
	}
	if _, err := svc.Create(CreateHostRequest{Hostname: "dept-20", IP: "10.10.0.20", OS: "linux", DeptID: dept20}, "1"); err != nil {
		t.Fatalf("seed dept 20 host: %v", err)
	}

	resp, err := svc.List(HostListQuery{Page: 1, PageSize: 10}, &common.DataScopeReq{
		Mode:    common.DataScopeModeDeptAndChildren,
		DeptID:  dept10,
		DeptIDs: []uint64{dept10, dept11},
	})
	if err != nil {
		t.Fatalf("list scoped hosts: %v", err)
	}
	if resp.Total != 2 {
		t.Fatalf("expected 2 scoped hosts, got %d", resp.Total)
	}
	for _, item := range resp.Items {
		if item.DeptID != dept10 && item.DeptID != dept11 {
			t.Fatalf("unexpected host outside scope: %+v", item)
		}
	}
}

func TestUpdateHost(t *testing.T) {
	db := setupTestDB(t)
	svc := NewHostService(db)
	created, _ := svc.Create(CreateHostRequest{Hostname: "h1", IP: "10.0.0.30", OS: "linux"}, "1")

	newHostname := "h1-updated"
	_, err := svc.Update(created.ID, UpdateHostRequest{Hostname: &newHostname}, "1", nil)
	if err != nil {
		t.Fatalf("update host: %v", err)
	}
	resp, _ := svc.GetByID(created.ID, nil)
	if resp.Hostname != "h1-updated" {
		t.Errorf("expected h1-updated, got %s", resp.Hostname)
	}
}

func TestDeleteHost(t *testing.T) {
	db := setupTestDB(t)
	svc := NewHostService(db)
	created, _ := svc.Create(CreateHostRequest{Hostname: "h1-del", IP: "10.0.0.40", OS: "linux"}, "1")

	if err := svc.Delete(created.ID, nil); err != nil {
		t.Fatalf("delete host: %v", err)
	}
	_, err := svc.GetByID(created.ID, nil)
	if err == nil {
		t.Error("expected not_found error after delete")
	}
}

func TestUpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	svc := NewHostService(db)
	created, _ := svc.Create(CreateHostRequest{Hostname: "h1-status", IP: "10.0.0.50", OS: "linux"}, "1")

	if err := svc.UpdateStatus(created.ID, "online", nil); err != nil {
		t.Fatalf("update status: %v", err)
	}
	resp, _ := svc.GetByID(created.ID, nil)
	if resp.Status != "online" {
		t.Errorf("expected online, got %s", resp.Status)
	}
}

func TestParseSystemInfoOutput(t *testing.T) {
	output := strings.Join([]string{
		"os=linux",
		"os_version=Ubuntu 24.04.2 LTS",
		"cpu_cores=4",
		"memory_gb=7.8",
		"disk_gb=40",
	}, "\n")

	info, err := parseSystemInfoOutput([]byte(output))
	if err != nil {
		t.Fatalf("parse system info: %v", err)
	}
	if info.OS != "linux" {
		t.Fatalf("expected linux, got %s", info.OS)
	}
	if info.OSVersion != "Ubuntu 24.04.2 LTS" {
		t.Fatalf("expected kernel version, got %s", info.OSVersion)
	}
	if info.CPUCores != 4 {
		t.Fatalf("expected 4 CPU cores, got %d", info.CPUCores)
	}
	if info.MemoryGB != 7.8 {
		t.Fatalf("expected 7.8 GB memory, got %.1f", info.MemoryGB)
	}
	if info.DiskGB != 40 {
		t.Fatalf("expected 40 GB disk, got %.1f", info.DiskGB)
	}
}

func TestParseSystemInfoOutputRejectsIncompleteCollection(t *testing.T) {
	if _, err := parseSystemInfoOutput([]byte("Linux\n6.8.0\n4\n7.8")); err == nil {
		t.Fatal("expected incomplete collection output to fail")
	}
}
