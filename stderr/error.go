package stderr

import (
	"errors"
	"fmt"
	"runtime/debug"
)

type StdError struct {
	err    error
	stacks string
}

func (e *StdError) Error() string {
	return fmt.Sprintf("err:%v\nstacks:%s", e.err, e.stacks)
}
func New(s string) error {
	return Wrap(errors.New(s))
}
func Wrap(err error) error {
	if err == nil {
		return nil
	}
	if v, ok := err.(*StdError); ok {
		return v
	}
	return &StdError{
		err:    err,
		stacks: getStack(),
	}
}

func getStack() string {
	return string(debug.Stack())
}
