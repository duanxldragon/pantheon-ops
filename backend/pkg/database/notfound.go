package database

import (
	"errors"

	"gorm.io/gorm"
)

// TranslateNotFound reclassifies a gorm not-found miss as the caller's
// replacement error and passes every other error through unchanged. It lives
// in pkg/database rather than pkg/common because pkg/common is gorm-free and
// stays that way; the service layer that already imports gorm takes the
// dependency here. The replacement is supplied by the caller so each site
// keeps its own output class — common.NewNotFound (404), common.NewBadRequest
// (400) or a plain error (500) — instead of the helper picking one.
func TranslateNotFound(err, replacement error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return replacement
	}
	return err
}
