package database

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTranslateNotFoundTranslatesRecordNotFound(t *testing.T) {
	got := TranslateNotFound(gorm.ErrRecordNotFound, errors.New("user.not_found"))
	require.EqualError(t, got, "user.not_found")
}

func TestTranslateNotFoundTranslatesWrappedRecordNotFound(t *testing.T) {
	wrapped := fmt.Errorf("query users: %w", gorm.ErrRecordNotFound)
	got := TranslateNotFound(wrapped, errors.New("tenant.not_found"))
	require.EqualError(t, got, "tenant.not_found")
}

func TestTranslateNotFoundPassesOtherErrorsThroughUnchanged(t *testing.T) {
	other := errors.New("connection refused")
	require.Same(t, other, TranslateNotFound(other, errors.New("user.not_found")))

	plainNotFound := errors.New("user not found")
	require.Same(t, plainNotFound, TranslateNotFound(plainNotFound, errors.New("user.not_found")))
}

func TestTranslateNotFoundNilStaysNil(t *testing.T) {
	require.NoError(t, TranslateNotFound(nil, errors.New("user.not_found")))
}
