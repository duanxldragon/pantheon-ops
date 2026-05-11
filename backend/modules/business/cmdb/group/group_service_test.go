package group

import (
	"encoding/json"
	"errors"
	"testing"

	"pantheon-ops/backend/pkg/testmysql"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&Group{}); err != nil {
		t.Fatalf("migrate group: %v", err)
	}
	if err := db.AutoMigrate(&Host{}); err != nil {
		t.Fatalf("migrate host: %v", err)
	}
	return db
}

func seedTestHost(t *testing.T, db *gorm.DB, ip string, labelsJSON string) {
	t.Helper()
	if err := db.Exec("INSERT INTO biz_cmdb_host (hostname, ip, os, label_values, status, created_at, updated_at) VALUES (?, ?, 'linux', ?, 'online', NOW(), NOW())", "host-"+ip, ip, labelsJSON).Error; err != nil {
		t.Fatalf("seed host %s: %v", ip, err)
	}
}

func jsonCondition(t *testing.T, expr ConditionExpression) datatypes.JSON {
	t.Helper()
	payload, err := json.Marshal(expr)
	if err != nil {
		t.Fatalf("marshal condition: %v", err)
	}
	return datatypes.JSON(payload)
}

func jsonLabels(t *testing.T, labels []LabelEntry) datatypes.JSON {
	t.Helper()
	payload, err := json.Marshal(labels)
	if err != nil {
		t.Fatalf("marshal labels: %v", err)
	}
	return datatypes.JSON(payload)
}

func TestCreateGroup(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)

	cond := ConditionExpression{
		Operator: "AND",
		Rules:    []ConditionRule{{Key: "env", Op: "eq", Val: "production"}},
	}
	resp, err := svc.Create(CreateGroupRequest{Name: "production", Conditions: cond}, nil)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if resp.Name != "production" {
		t.Errorf("expected production, got %s", resp.Name)
	}
}

func TestGroupMembersFiltering(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)

	seedTestHost(t, db, "10.0.0.1", `[{"key":"env","val":"production"},{"key":"biz","val":"order"}]`)
	seedTestHost(t, db, "10.0.0.2", `[{"key":"env","val":"test"},{"key":"biz","val":"order"}]`)
	seedTestHost(t, db, "10.0.0.3", `[{"key":"env","val":"production"},{"key":"biz","val":"user"}]`)

	cond := ConditionExpression{
		Operator: "AND",
		Rules:    []ConditionRule{{Key: "env", Op: "eq", Val: "production"}},
	}
	created, _ := svc.Create(CreateGroupRequest{Name: "prod", Conditions: cond}, nil)

	members, group, err := svc.GetMembers(created.ID, nil)
	if err != nil {
		t.Fatalf("get members: %v", err)
	}
	if group.Name != "prod" {
		t.Errorf("expected prod, got %s", group.Name)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}
}

func TestFilterHostsByConditionChain(t *testing.T) {
	hosts := []Host{
		{
			IP: "10.0.0.1",
			LabelValues: jsonLabels(t, []LabelEntry{
				{Key: "region", Val: "西安开发环境"},
				{Key: "role", Val: "monitor"},
			}),
		},
		{
			IP: "10.0.0.2",
			LabelValues: jsonLabels(t, []LabelEntry{
				{Key: "region", Val: "西安开发环境"},
				{Key: "role", Val: "app"},
			}),
		},
		{
			IP: "10.0.0.3",
			LabelValues: jsonLabels(t, []LabelEntry{
				{Key: "region", Val: "上海开发环境"},
				{Key: "role", Val: "monitor"},
			}),
		},
	}
	parent := jsonCondition(t, ConditionExpression{
		Operator: "AND",
		Rules:    []ConditionRule{{Key: "region", Op: "eq", Val: "西安开发环境"}},
	})
	child := jsonCondition(t, ConditionExpression{
		Operator: "AND",
		Rules:    []ConditionRule{{Key: "role", Op: "eq", Val: "monitor"}},
	})

	members := filterHostsByConditionChain(hosts, []datatypes.JSON{parent, child})
	if len(members) != 1 {
		t.Fatalf("expected inherited chain to return 1 member, got %d", len(members))
	}
	if members[0].IP != "10.0.0.1" {
		t.Fatalf("expected 10.0.0.1, got %s", members[0].IP)
	}
}

func TestUpdateGroup(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)
	cond := ConditionExpression{Operator: "AND", Rules: []ConditionRule{{Key: "env", Op: "eq", Val: "production"}}}
	created, _ := svc.Create(CreateGroupRequest{Name: "old", Conditions: cond}, nil)

	newName := "new-name"
	_, err := svc.Update(created.ID, UpdateGroupRequest{Name: &newName}, nil)
	if err != nil {
		t.Fatalf("update group: %v", err)
	}
	resp, _ := svc.GetByID(created.ID, nil)
	if resp.Name != "new-name" {
		t.Errorf("expected new-name, got %s", resp.Name)
	}
}

func TestGroupListBuildsSubGroupTree(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)
	cond := ConditionExpression{Operator: "AND", Rules: []ConditionRule{{Key: "env", Op: "eq", Val: "production"}}}
	parent, err := svc.Create(CreateGroupRequest{Name: "parent", Conditions: cond}, nil)
	if err != nil {
		t.Fatalf("create parent group: %v", err)
	}
	child, err := svc.Create(CreateGroupRequest{Name: "child", ParentID: parent.ID, Conditions: cond}, nil)
	if err != nil {
		t.Fatalf("create child group: %v", err)
	}

	items, err := svc.List(nil)
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one root group, got %d", len(items))
	}
	if items[0].ID != parent.ID {
		t.Fatalf("expected parent root %d, got %d", parent.ID, items[0].ID)
	}
	if items[0].ChildCount != 1 || len(items[0].Children) != 1 {
		t.Fatalf("expected one child, childCount=%d children=%d", items[0].ChildCount, len(items[0].Children))
	}
	if items[0].Children[0].ID != child.ID || items[0].Children[0].ParentID != parent.ID {
		t.Fatalf("unexpected child response: %+v", items[0].Children[0])
	}
}

func TestSubGroupMembersInheritParentConditions(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)

	seedTestHost(t, db, "10.0.0.1", `[{"key":"region","val":"西安开发环境"},{"key":"role","val":"monitor"}]`)
	seedTestHost(t, db, "10.0.0.2", `[{"key":"region","val":"西安开发环境"},{"key":"role","val":"app"}]`)
	seedTestHost(t, db, "10.0.0.3", `[{"key":"region","val":"上海开发环境"},{"key":"role","val":"monitor"}]`)

	parentCond := ConditionExpression{
		Operator: "AND",
		Rules:    []ConditionRule{{Key: "region", Op: "eq", Val: "西安开发环境"}},
	}
	parent, err := svc.Create(CreateGroupRequest{Name: "西安开发环境", Conditions: parentCond}, nil)
	if err != nil {
		t.Fatalf("create parent group: %v", err)
	}
	childCond := ConditionExpression{
		Operator: "AND",
		Rules:    []ConditionRule{{Key: "role", Op: "eq", Val: "monitor"}},
	}
	child, err := svc.Create(CreateGroupRequest{Name: "监控服务器", ParentID: parent.ID, Conditions: childCond}, nil)
	if err != nil {
		t.Fatalf("create child group: %v", err)
	}

	members, _, err := svc.GetMembers(child.ID, nil)
	if err != nil {
		t.Fatalf("get child members: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected child group to inherit parent condition and return 1 member, got %d", len(members))
	}
	if members[0].IP != "10.0.0.1" {
		t.Fatalf("expected inherited child group to include 10.0.0.1, got %s", members[0].IP)
	}

	items, err := svc.List(nil)
	if err != nil {
		t.Fatalf("list groups: %v", err)
	}
	if len(items) != 1 || len(items[0].Children) != 1 {
		t.Fatalf("expected parent with one child, got %+v", items)
	}
	if items[0].MemberCount != 2 {
		t.Fatalf("expected parent member count 2, got %d", items[0].MemberCount)
	}
	if items[0].AggregateMemberCount != 2 {
		t.Fatalf("expected parent aggregate member count 2, got %d", items[0].AggregateMemberCount)
	}
	if items[0].DescendantGroupCount != 1 {
		t.Fatalf("expected parent descendant group count 1, got %d", items[0].DescendantGroupCount)
	}
	if items[0].Children[0].MemberCount != 1 {
		t.Fatalf("expected child inherited member count 1, got %d", items[0].Children[0].MemberCount)
	}
	if items[0].Children[0].AggregateMemberCount != 1 {
		t.Fatalf("expected child aggregate member count 1, got %d", items[0].Children[0].AggregateMemberCount)
	}
}

func TestDeleteGroupRejectsParentWithChildren(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)
	cond := ConditionExpression{Operator: "AND", Rules: []ConditionRule{{Key: "env", Op: "eq", Val: "production"}}}
	parent, err := svc.Create(CreateGroupRequest{Name: "parent", Conditions: cond}, nil)
	if err != nil {
		t.Fatalf("create parent group: %v", err)
	}
	if _, err := svc.Create(CreateGroupRequest{Name: "child", ParentID: parent.ID, Conditions: cond}, nil); err != nil {
		t.Fatalf("create child group: %v", err)
	}

	err = svc.Delete(parent.ID)
	if !errors.Is(err, ErrGroupHasChildren) {
		t.Fatalf("expected ErrGroupHasChildren, got %v", err)
	}
}

func TestUpdateGroupRejectsCycle(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)
	cond := ConditionExpression{Operator: "AND", Rules: []ConditionRule{{Key: "env", Op: "eq", Val: "production"}}}
	parent, err := svc.Create(CreateGroupRequest{Name: "parent", Conditions: cond}, nil)
	if err != nil {
		t.Fatalf("create parent group: %v", err)
	}
	child, err := svc.Create(CreateGroupRequest{Name: "child", ParentID: parent.ID, Conditions: cond}, nil)
	if err != nil {
		t.Fatalf("create child group: %v", err)
	}

	nextParent := child.ID
	_, err = svc.Update(parent.ID, UpdateGroupRequest{ParentID: &nextParent}, nil)
	if !errors.Is(err, ErrGroupCycle) {
		t.Fatalf("expected ErrGroupCycle, got %v", err)
	}
}

func TestDeleteGroup(t *testing.T) {
	db := setupTestDB(t)
	svc := NewGroupService(db)
	cond := ConditionExpression{Operator: "AND", Rules: []ConditionRule{{Key: "env", Op: "eq", Val: "production"}}}
	created, _ := svc.Create(CreateGroupRequest{Name: "tmp", Conditions: cond}, nil)

	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("delete group: %v", err)
	}
	_, err := svc.GetByID(created.ID, nil)
	if err == nil {
		t.Error("expected not_found error after delete")
	}
}
