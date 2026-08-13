package iam

import (
	"errors"
	"testing"

	"pantheon-base/pkg/common"
)

func TestPermissionServiceErrorCode(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{name: "forbidden", err: common.NewForbidden("permission.escalation.forbidden"), want: common.CodeForbidden},
		{name: "generic", err: errors.New("boom"), want: common.CodeError},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := permissionServiceErrorCode(tt.err); got != tt.want {
				t.Fatalf("permissionServiceErrorCode() = %d, want %d", got, tt.want)
			}
		})
	}
}
