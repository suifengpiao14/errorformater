package errorformatter

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
)

func TestFormatError(t *testing.T) {
	formatter := New("", false, nil, nil, nil, nil)
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
	ch := make(chan *ErrorInfo, 10)
	formatter := New("errorformatter", false, ch, nil, g.PCs, g.Cause)
	testErr := Error2()
	err := formatter.Error(testErr)
	errorInfo := <-ch
	fmt.Println(errorInfo)
	fmt.Println(err)
}

func TestFmtErrorf(t *testing.T) {
	err := fmt.Errorf("test")
	fmt.Println("%w", err)
}
