package errorformator

import (
	"errors"
	"fmt"
	"testing"
)

func TestFormatError(t *testing.T) {
	err := errors.New("test")
	err = Format(err.Error())
	fmt.Println(err)
}

func TestFormatErrorNew(t *testing.T) {
	filename := "/tmp/errMap.json"
	errorformator, err := New(filename)
	if err != nil {
		panic(err)
	}
	testErr := errors.New("test")
	err = errorformator.Format(testErr.Error())
	fmt.Println(err)
}

func Error2() (err error) {
	msg := "error2"
	err = Format(msg)
	return
}

func TestError2(t *testing.T) {
	filename := "/tmp/errMap.json"
	errorformator, err := New(filename)
	if err != nil {
		panic(err)
	}
	testErr := Error2()
	err = errorformator.Format(testErr.Error())
	fmt.Println(err)
}
