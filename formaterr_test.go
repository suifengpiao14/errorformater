package formatgoerr

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
