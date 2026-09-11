package errors

import (
	stderrors "errors"
	"fmt"
	"testing"
)

func TestBusinessErrorWithHelpersDoNotMutateSentinel(t *testing.T) {
	baseCode := ErrCreateFailed.Code
	baseMessage := ErrCreateFailed.Message

	wrapped := ErrCreateFailed.WithError(fmt.Errorf("db down"))
	if ErrCreateFailed.Err != nil {
		t.Fatalf("sentinel ErrCreateFailed.Err mutated: %v", ErrCreateFailed.Err)
	}
	if wrapped.Err == nil || wrapped.Err.Error() != "db down" {
		t.Fatalf("wrapped error missing cause: %v", wrapped.Err)
	}
	if wrapped.Code != baseCode || ErrCreateFailed.Code != baseCode {
		t.Fatalf("code mismatch: wrapped=%s sentinel=%s", wrapped.Code, ErrCreateFailed.Code)
	}

	originalInvalidMsg := ErrInvalidArgument.Message
	custom := ErrInvalidArgument.WithMessage("custom msg")
	if ErrInvalidArgument.Message != originalInvalidMsg {
		t.Fatal("sentinel ErrInvalidArgument.Message was mutated")
	}
	if custom.Message != "custom msg" {
		t.Fatalf("custom message not applied: %s", custom.Message)
	}

	withParams := ErrInsufficientBalance.WithParams(map[string]any{"balance": 12.5})
	if len(ErrInsufficientBalance.Params) != 0 {
		t.Fatalf("sentinel Params mutated: %#v", ErrInsufficientBalance.Params)
	}
	if withParams.Params["balance"] != 12.5 {
		t.Fatalf("params not applied: %#v", withParams.Params)
	}

	chained := ErrCreateFailed.WithError(fmt.Errorf("a")).WithMessage("m").WithParams(map[string]any{"k": 1})
	if ErrCreateFailed.Err != nil || ErrCreateFailed.Message != baseMessage || len(ErrCreateFailed.Params) != 0 {
		t.Fatal("chained helpers mutated sentinel ErrCreateFailed")
	}
	if chained.Err == nil || chained.Message != "m" || chained.Params["k"] != 1 {
		t.Fatalf("chained copy incomplete: %#v", chained)
	}
}

func TestGetBusinessErrorSupportsUnwrap(t *testing.T) {
	inner := ErrRecordNotFound.WithMessage("missing")
	wrapped := fmt.Errorf("outer: %w", inner)

	got, ok := GetBusinessError(wrapped)
	if !ok {
		t.Fatal("expected GetBusinessError to unwrap")
	}
	if got.Code != ErrRecordNotFound.Code {
		t.Fatalf("unexpected code: %s", got.Code)
	}
	if !IsBusinessError(wrapped) {
		t.Fatal("expected IsBusinessError to unwrap")
	}
	if !stderrors.Is(wrapped, ErrRecordNotFound) {
		t.Fatal("expected errors.Is to match by code via Unwrap chain")
	}
}
