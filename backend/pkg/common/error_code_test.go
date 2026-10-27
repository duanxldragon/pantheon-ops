package common

import (
	"errors"
	"fmt"
	"testing"
)

// TestCodeForError pins the sentinel -> envelope-code mapping used by handlers
// that would otherwise collapse every service failure to 500.
func TestCodeForError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil is an internal default", nil, CodeError},
		{"bad request maps to 400", NewBadRequest("dept.parent.required"), CodeParamInvalid},
		{"conflict has no 409 sentinel, so it maps to 400", NewConflict("post.code.exists"), CodeParamInvalid},
		{"not found maps to 404", NewNotFound("menu.parent.not_found"), CodeNotFound},
		{"forbidden maps to 403", NewForbidden("post.dept.root_forbidden"), CodeForbidden},
		{"unauthorized maps to 401", NewUnauthorized("auth.operation.verify.error"), CodeUnauthorized},
		{"internal stays 500", NewInternal("menu.update.error.parent_self"), CodeError},
		{"a raw driver error stays 500", errors.New("Error 1052: ambiguous column"), CodeError},
		{"a wrapped sentinel is still classified", fmt.Errorf("create dept: %w", NewBadRequest("dept.parent.required")), CodeParamInvalid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CodeForError(tc.err); got != tc.want {
				t.Fatalf("CodeForError(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}

// TestErrMessageExtractsKey guards the extraction the menu handler relies on:
// the key must survive the "sentinel: key" wrapping NewBadRequest applies.
func TestErrMessageExtractsKey(t *testing.T) {
	if got := ErrMessage(NewBadRequest("menu.page_perm.required")); got != "menu.page_perm.required" {
		t.Fatalf("ErrMessage = %q, want %q", got, "menu.page_perm.required")
	}
	if got := ErrMessage(errors.New("plain")); got != "plain" {
		t.Fatalf("ErrMessage = %q, want %q", got, "plain")
	}
	if got := ErrMessage(nil); got != "" {
		t.Fatalf("ErrMessage(nil) = %q, want empty", got)
	}
}
