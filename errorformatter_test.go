package errorformatter

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
)

func TestFormatError(t *testing.T) {
	include := []string{}
	exclude := []string{}
	formatter := New(include, exclude, nil, nil, nil, nil)
	testErr := errors.New("test")
	err := formatter.Msg(testErr.Error())
	fmt.Println(err)
}

func Error2() (err error) {
	msg := "error2"
	err = errors.New(msg)
	return
}

func TestError2(t *testing.T) {
	g := &GithubComPkgErrors{}
	include := []string{"errorformatter"}
	exclude := []string{}
	formatter := New(include, exclude, nil, g.PCs, g.Cause, nil)
	testErr := Error2()
	err := formatter.SetError(testErr)
	fmt.Println(err)
}

func TestFmtErrorf(t *testing.T) {
	err := fmt.Errorf("test")
	fmt.Println("%w", err)
}
