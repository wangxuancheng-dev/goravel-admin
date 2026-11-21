package payment

import "errors"

// Sentinel errors for gateway drivers. Host apps may wrap these into their own error types.
var (
	ErrConfigRequired = errors.New("payment: config required")
	ErrNotImplemented = errors.New("payment: gateway not implemented")
	ErrNotifyInvalid  = errors.New("payment: notify invalid")
	ErrCreateFailed   = errors.New("payment: create failed")
	ErrStatusInvalid  = errors.New("payment: status invalid")
)
