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

func TestFormatErrorNew2(t *testing.T) {
	filename := "/tmp/errMap.json"
	errorformator, err := New(filename)
	if err != nil {
		panic(err)
	}
	testErr := errors.New("test")
	err = errorformator.Format(testErr.Error())
	fmt.Println(err)
}
