package formatgoerr

import (
	"fmt"
	"runtime"
	"strconv"
)

//FormatError generate format error message
func FormatError(msg string, args ...int) (err error) {
	httpCode := 500
	businessCode := "000000"
	if len(args) >= 2 {
		httpCode = args[0]
		businessCode = strconv.Itoa(args[1])
		err = fmt.Errorf("%d:%s:%s", httpCode, businessCode, msg)
		return
	}
	if len(args) == 1 {
		httpCode = args[0]
	}
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	_, line := f.FileLine(pc[0])
	// use function name and line as default businessCode
	businessCode = fmt.Sprintf("%s.%03d", f.Name(), line)
	err = fmt.Errorf("%d:%s:%s", httpCode, businessCode, msg)
	return
}
