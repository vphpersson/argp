package errors

import "errors"

var (
	ErrUnknownOption = errors.New("unknown option")
	ErrUnexpectedKind = errors.New("unexpected kind")
	ErrUnexpectedInput = errors.New("unexpected input")
	ErrShowHelp = errors.New("show help")
	ErrMissingValue = errors.New("missing value")
)
