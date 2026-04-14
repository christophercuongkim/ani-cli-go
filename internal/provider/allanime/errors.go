package allanime

import "errors"

// Sentinel errors for allanime package
var (
	ErrNotImplemented   = errors.New("not implemented")
	ErrInvalidHexString = errors.New("invalid hex string")
	ErrShowNotFound     = errors.New("show not found")
)
