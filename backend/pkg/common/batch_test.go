package common

import (
	"errors"
	"testing"
)

func TestNormalizeUint64IDsRemovesDuplicates(t *testing.T) {
	result := NormalizeUint64IDs([]uint64{1, 2, 2, 3, 3, 3})
	if len(result) != 3 {
		t.Fatalf("expected 3 unique ids, got %d: %v", len(result), result)
	}
	if result[0] != 1 || result[1] != 2 || result[2] != 3 {
		t.Fatalf("unexpected order: %v", result)
	}
}

func TestNormalizeUint64IDsRemovesZeros(t *testing.T) {
	result := NormalizeUint64IDs([]uint64{0, 1, 0, 2, 0})
	if len(result) != 2 {
		t.Fatalf("expected 2 ids after removing zeros, got %d: %v", len(result), result)
	}
	if result[0] != 1 || result[1] != 2 {
		t.Fatalf("unexpected result: %v", result)
	}
}

func TestNormalizeUint64IDsEmptyInput(t *testing.T) {
	result := NormalizeUint64IDs(nil)
	if len(result) != 0 {
		t.Fatalf("expected empty result for nil input, got %v", result)
	}

	result = NormalizeUint64IDs([]uint64{})
	if len(result) != 0 {
		t.Fatalf("expected empty result for empty slice, got %v", result)
	}
}

func TestNormalizeUint64IDsAllZeros(t *testing.T) {
	result := NormalizeUint64IDs([]uint64{0, 0, 0})
	if len(result) != 0 {
		t.Fatalf("expected empty result for all zeros, got %v", result)
	}
}

func TestBatchDeleteAllSucceed(t *testing.T) {
	counter := 0
	resp := BatchDelete([]uint64{1, 2, 3}, func(id uint64) error {
		counter++
		return nil
	})
	if resp.DeletedCount != 3 {
		t.Fatalf("expected 3 deleted, got %d", resp.DeletedCount)
	}
	if resp.FailedCount != 0 {
		t.Fatalf("expected 0 failures, got %d", resp.FailedCount)
	}
	if len(resp.Failures) != 0 {
		t.Fatalf("expected 0 failure entries, got %d", len(resp.Failures))
	}
	if counter != 3 {
		t.Fatalf("expected deleteOne called 3 times, got %d", counter)
	}
}

func TestBatchDeleteWithFailures(t *testing.T) {
	resp := BatchDelete([]uint64{1, 2, 3}, func(id uint64) error {
		if id == 2 {
			return errors.New("user.in_use")
		}
		return nil
	})
	if resp.DeletedCount != 2 {
		t.Fatalf("expected 2 deleted, got %d", resp.DeletedCount)
	}
	if resp.FailedCount != 1 {
		t.Fatalf("expected 1 failure, got %d", resp.FailedCount)
	}
	if len(resp.Failures) != 1 || resp.Failures[0].ID != 2 {
		t.Fatalf("expected failure for id 2, got %+v", resp.Failures)
	}
}

func TestBatchDeleteAllFail(t *testing.T) {
	resp := BatchDelete([]uint64{1, 2}, func(id uint64) error {
		return errors.New("permission.denied")
	})
	if resp.DeletedCount != 0 {
		t.Fatalf("expected 0 deleted, got %d", resp.DeletedCount)
	}
	if resp.FailedCount != 2 {
		t.Fatalf("expected 2 failures, got %d", resp.FailedCount)
	}
	if len(resp.Failures) != 2 {
		t.Fatalf("expected 2 failure entries, got %d", len(resp.Failures))
	}
}

func TestBatchDeleteEmptyInput(t *testing.T) {
	called := false
	resp := BatchDelete(nil, func(id uint64) error {
		called = true
		return nil
	})
	if resp.DeletedCount != 0 {
		t.Fatalf("expected 0 deleted for nil input, got %d", resp.DeletedCount)
	}
	if called {
		t.Fatal("deleteOne should not be called for nil input")
	}

	resp = BatchDelete([]uint64{}, func(id uint64) error {
		called = true
		return nil
	})
	if resp.DeletedCount != 0 {
		t.Fatalf("expected 0 deleted for empty input, got %d", resp.DeletedCount)
	}
}

func TestBatchDeleteRemovesDuplicates(t *testing.T) {
	callCount := 0
	resp := BatchDelete([]uint64{1, 1, 2, 2}, func(id uint64) error {
		callCount++
		return nil
	})
	if resp.DeletedCount != 2 {
		t.Fatalf("expected 2 deleted after dedup, got %d", resp.DeletedCount)
	}
	if callCount != 2 {
		t.Fatalf("expected deleteOne called 2 times, got %d", callCount)
	}
}
