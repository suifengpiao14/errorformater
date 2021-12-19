package errorformatter

import (
	"strconv"

	"github.com/pkg/errors"
)

// ErrorChain 链式调用
type ErrorChain interface {
	SetError(err error) ErrorChain
	Error() error
}

type chain struct {
	err error
}

// NewErrorChain nee ErrorChain
func NewErrorChain() ErrorChain {
	return &chain{}
}

// Error get error from chain
func (c *chain) Error() error {
	return c.err
}

// SetError sets the error
func (c *chain) SetError(err error) ErrorChain {
	if c.err != nil {
		return c
	}
	_, ok := err.(GithubComPkgErrorsStackTracer)
	if !ok {
		err = errors.WithStack(err)
	}
	c.err = err
	return c
}

//Str2Int help function for string to int conversion
func Str2Int(s string, out *int) (err error) {
	*out, err = strconv.Atoi(s)
	return err
}
