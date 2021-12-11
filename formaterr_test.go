package errorformater

import (
	"errors"
	"fmt"
	"testing"
)

func TestFormatError(t *testing.T) {
	err := errors.New("test")
	err = FormatError(err.Error())
	fmt.Println(err)
}

func TestFormatErrorNew(t *testing.T) {
	filename := "/tmp/errMap.json"
	errorFormater, err := New(filename)
	if err != nil {
		panic(err)
	}
	testErr := errors.New("test")
	err = errorFormater.FormatError(testErr.Error())
	fmt.Println(err)
}
