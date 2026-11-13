package tenant

import (
	"context"
	"errors"
	"reflect"
)

// ModeReader reads authoritative deployment mode without a fallback or cache.
type ModeReader interface {
	ReadMode(context.Context) (string, error)
}

// ErrModeUnavailable reports that no authoritative explicit mode could be read.
var ErrModeUnavailable = errors.New("tenant.mode.unavailable")

type strictSettingModeReader struct{ reader SettingFlagReader }

// NewStrictSettingModeReader wraps a settings reader with strict validation.
func NewStrictSettingModeReader(reader SettingFlagReader) ModeReader {
	return &strictSettingModeReader{reader: reader}
}

func (r *strictSettingModeReader) ReadMode(ctx context.Context) (string, error) {
	if ctx == nil || r == nil || r.reader == nil {
		return "", ErrModeUnavailable
	}
	v := reflect.ValueOf(r.reader)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return "", ErrModeUnavailable
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	raw, err := r.reader.GetByKey(FeatureFlagSettingKey)
	if err != nil {
		return "", ErrModeUnavailable
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	// Historical bootstrap stores a JSON string; only known explicit values count.
	value := NormalizeFlagValue(raw)
	if value != ModeCompat && value != ModeMulti {
		return "", ErrModeUnavailable
	}
	return value, nil
}
