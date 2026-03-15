package command

import "github.com/cockroachdb/errors"

var (
	ErrHandlerNotFound   = errors.New("handler not found for command")
	ErrCommandValidation = errors.New("command validation failed")
	ErrDispatcherClosed  = errors.New("command dispatcher is closed")
)
