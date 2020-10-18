package application

import (
	"errors"
	"testing"
)

func TestErrorAppend(t *testing.T) {
	var err *Error
	err1 := errors.New("test 1")
	err2 := errors.New("test 2")

	err = err.Append(err1)
	err = err.Append(err2)

	if len(err.Errors) != 2 {
		t.Fatalf("errors did not get appended: %v", err)
	}

	if err.Errors[0] != err1 {
		t.Errorf("unexpected error: expected %q; got %q", err1.Error(), err.Errors[0].Error())
	}
	if err.Errors[1] != err2 {
		t.Errorf("unexpected error: expected %q; got %q", err2.Error(), err.Errors[1].Error())
	}
}
