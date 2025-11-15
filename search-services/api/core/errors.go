package core

import "errors"

var (
	ErrBadArguments       = errors.New("arguments are not acceptable")
	ErrServiceUnavailable = errors.New("service unavailable")
	ErrInternal           = errors.New("internal error")
	ErrUpdateInProgress   = errors.New("update in progress")
)
