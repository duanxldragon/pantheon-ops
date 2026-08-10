package iam

import (
	"testing"
)

func TestUserService_ListUsersKeywordMatchesMultipleFields(t *testing.T) {
	db := setupUserTestDB(t)
	s := NewUserService(db)

	seed := []*UserCreateReq{
		{Username: "zhangsan", Password: "password123", Nickname: "张三", Email: "zhang@corp.com", Status: 1},
		{Username: "lisi", Password: "password123", Nickname: "李四", Email: "lisi@corp.com", Status: 1},
		{Username: "wangwu", Password: "password123", Nickname: "王五", Email: "wang@other.io", Status: 1},
	}
	for _, req := range seed {
		if _, err := s.CreateUser(req); err != nil {
			t.Fatalf("seed user %s: %v", req.Username, err)
		}
	}

	cases := []struct {
		name    string
		keyword string
		want    int
	}{
		{name: "matches username", keyword: "zhangsan", want: 1},
		{name: "matches nickname", keyword: "李四", want: 1},
		{name: "matches email domain across users", keyword: "corp.com", want: 2},
		{name: "no match", keyword: "nonexistent", want: 0},
		{name: "escapes like wildcards", keyword: "%", want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := s.ListUsers(&UserListQuery{Keyword: tc.keyword, Page: 1, PageSize: 20}, nil)
			if err != nil {
				t.Fatalf("list users: %v", err)
			}
			if len(resp.Items) != tc.want {
				t.Fatalf("keyword %q: expected %d users, got %d", tc.keyword, tc.want, len(resp.Items))
			}
		})
	}

	t.Run("keyword combines with status filter", func(t *testing.T) {
		disabled := 2
		resp, err := s.ListUsers(&UserListQuery{Keyword: "corp.com", Status: &disabled, Page: 1, PageSize: 20}, nil)
		if err != nil {
			t.Fatalf("list users: %v", err)
		}
		if len(resp.Items) != 0 {
			t.Fatalf("expected keyword AND status to combine, got %d users", len(resp.Items))
		}
	})
}
