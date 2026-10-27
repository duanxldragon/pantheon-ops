package main

import "testing"

func TestMatrixModeDefaultsToCompat(t *testing.T) {
	t.Setenv("PANTHEON_MATRIX_MODE", "")
	mode, err := matrixMode()
	if err != nil {
		t.Fatalf("matrixMode() returned error: %v", err)
	}
	if mode != flagCompat {
		t.Fatalf("matrixMode() = %q, want %q", mode, flagCompat)
	}
}

func TestMatrixModeAcceptsCompatAndMulti(t *testing.T) {
	for _, value := range []string{flagCompat, "MULTI"} {
		t.Setenv("PANTHEON_MATRIX_MODE", value)
		mode, err := matrixMode()
		if err != nil {
			t.Fatalf("matrixMode(%q) returned error: %v", value, err)
		}
		want := value
		if value == "MULTI" {
			want = flagMulti
		}
		if mode != want {
			t.Fatalf("matrixMode(%q) = %q, want %q", value, mode, want)
		}
	}
}

func TestMatrixModeRejectsUnknownValue(t *testing.T) {
	t.Setenv("PANTHEON_MATRIX_MODE", "staging")
	if _, err := matrixMode(); err == nil {
		t.Fatal("matrixMode(staging) returned nil error")
	}
}

func TestSplitSQLStatementsSplitsMultipleStatementsOnSameLine(t *testing.T) {
	got := splitSQLStatements("SET @s := 1; SET @q := 'DROP INDEX'; PREPARE x FROM @q; EXECUTE x;")
	want := []string{"SET @s := 1;", "SET @q := 'DROP INDEX';", "PREPARE x FROM @q;", "EXECUTE x;"}
	if len(got) != len(want) {
		t.Fatalf("splitSQLStatements() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitSQLStatements()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
