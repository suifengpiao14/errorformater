package errorformatter

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/sigurn/crc8"
	"golang.org/x/mod/modfile"
)

const (
	SEPARATOR       = '#'
	WITH_CALL_CHAIN = false
	SKIP            = 2
	FORMAT_TPL      = "%c%d:%s%c%s"
)

type Formatter struct {
	Include    []string
	Exclude    []string
	HttpStatus func(packageName string, funcName string) (int, bool)
	PCs        func(err error, pc []uintptr) (n int)
	Cause      func(err error) (tagetErr error)
	Chan       chan<- *ErrorCode
}

type CodeInfo struct {
	Code     string    `json:"code"`
	Package  string    `json:"package"`
	Function string    `json:"function"`
	Line     string    `json:"line"`
	Msg      string    `json:"msg"`
	Cause    *CodeInfo `json:"cause"`
}

type ErrorCode struct {
	HttpStatus int    `json:"-"`
	Code       string `json:"code"`
	Msg        string `json:"msg"`
	cause      error  `json:"-"`
	CodeInfo   *CodeInfo
}

func (e *ErrorCode) Error() string {

	msg := fmt.Sprintf(FORMAT_TPL, SEPARATOR, e.HttpStatus, e.Code, SEPARATOR, e.Msg)
	return msg
}
func (e *ErrorCode) Cause() error { return e.cause }

//ParseMsg parse string to *ErrorCode
func (e *ErrorCode) ParseMsg(msg string) (ok bool) {
	ok = false
	if msg[0] != byte(SEPARATOR) {
		return
	}
	arr := strings.SplitN(msg, string(SEPARATOR), 3)
	if len(arr) < 3 {
		return
	}
	codeArr := strings.SplitN(arr[1], ":", 2)
	if len(codeArr) < 2 {
		return
	}
	httpStatus, err := strconv.Atoi(codeArr[0])
	if err != nil {
		return
	}
	e.HttpStatus = httpStatus
	e.Code = codeArr[1]
	e.Msg = arr[2]
	ok = true
	return
}

func New(
	include []string,
	exclude []string,
	httpStatus func(packageName string, funcName string) (int, bool),
	pcs func(err error, pc []uintptr) (n int),
	cause func(err error) (tagetErr error),
	ch chan<- *ErrorCode,
) (formatter *Formatter) {
	formatter = &Formatter{
		Include:    include,
		Exclude:    exclude,
		HttpStatus: httpStatus,
		PCs:        pcs,
		Cause:      cause,
		Chan:       ch,
	}
	return
}

//Msg generate *ErrorCode from msg
func (formatter *Formatter) Msg(msg string, args ...int) (err *ErrorCode) {
	httpStatus := 500
	code := "000000000"
	if len(args) >= 2 {
		httpStatus = args[0]
		code = strconv.Itoa(args[1])
		err = &ErrorCode{
			HttpStatus: httpStatus,
			Code:       code,
			Msg:        msg,
		}
		return
	}
	if len(args) == 1 {
		httpStatus = args[0]
	}
	pcArr := make([]uintptr, 32) // at least 1 entry needed
	n := runtime.Callers(SKIP, pcArr)
	frames := runtime.CallersFrames(pcArr[:n])
	codeInfo := formatter.Frames(frames)
	codeInfo.Msg = msg
	if formatter.HttpStatus != nil {
		tmpHttpStatus, ok := formatter.HttpStatus(codeInfo.Package, codeInfo.Function)
		if ok {
			httpStatus = tmpHttpStatus
		}
	}
	err = &ErrorCode{
		HttpStatus: httpStatus,
		Code:       codeInfo.Code,
		Msg:        msg,
		CodeInfo:   codeInfo,
	}
	return
}

//Error generate *ErrorCode from error
func (formatter *Formatter) SetError(err error) (newErr *ErrorCode) {
	if formatter.Cause != nil {
		err = formatter.Cause(err)
	}
	e, ok := err.(*ErrorCode)
	if ok {
		return e
	}
	httpStatus := 500
	pcArr := make([]uintptr, 32) // at least 1 entry needed
	var frames *runtime.Frames
	n := 0
	if formatter.PCs != nil {
		n = formatter.PCs(err, pcArr)
	} else {
		n = runtime.Callers(SKIP, pcArr)

	}
	frames = runtime.CallersFrames(pcArr[:n])
	codeInfo := formatter.Frames(frames)
	msg := err.Error()
	codeInfo.Msg = msg
	if formatter.HttpStatus != nil {
		tmpHttpStatus, ok := formatter.HttpStatus(codeInfo.Package, codeInfo.Function)
		if ok {
			httpStatus = tmpHttpStatus
		}
	}
	newErr = &ErrorCode{
		HttpStatus: httpStatus,
		Code:       codeInfo.Code,
		Msg:        msg,
		cause:      err,
		CodeInfo:   codeInfo,
	}
	return
}

// Frames generate *CodeInfo from frames
func (formatter *Formatter) Frames(frames *runtime.Frames) (codeInfo *CodeInfo) {
	root := &CodeInfo{}
	point := root
	codeInfo = root
	codeArr := make([]string, 0)
	msgArr := make([]string, 0)
	for {
		frame, hasNext := frames.Next()
		fullname := frame.Function
		line := frame.Line
		if point.Code != "" {
			codeArr = append(codeArr, point.Code)
		}
		if point.Msg != "" {
			msgArr = append(msgArr, point.Msg)
		}

		if len(formatter.Include) > 0 { //Include 中匹配任意规则即可
			any := false
			for _, keyword := range formatter.Include {
				any = strings.Contains(fullname, keyword)
				if any {
					break
				}
			}
			if !any {
				if hasNext {
					continue
				} else {
					break
				}
			}
		}

		if len(formatter.Exclude) > 0 { //Exclude 中匹配任意规则即排除
			any := false
			for _, keyword := range formatter.Exclude {
				any = strings.Contains(fullname, keyword)
				if any {
					break
				}
			}
			if any {
				if hasNext {
					continue
				} else {
					break
				}
			}
		}
		point.Cause = formatter.FuncName2CodeInfo(fullname, line)
		point = point.Cause

	}
	// msgArr、codeArr 第一个为root的，全部为空，没有意义
	root.Msg = strings.Join(msgArr, ":")
	switch len(codeArr) {
	case 0:
		root.Code = "000000000"
	case 1:
		root.Code = codeArr[0]
	default:
		firstCode := codeArr[0]
		restCode := codeArr[1:]
		restCodeStr := strings.Join(restCode, ":")
		table := crc8.MakeTable(crc8.CRC8)
		codePrefix := crc8.Checksum([]byte(restCodeStr), table)
		root.Code = fmt.Sprintf("%3d%s", codePrefix, firstCode[3:])
	}
	codeInfo = root
	return
}

//FuncName2CodeInfo generate *CodeInfo from full function name
func (formatter *Formatter) FuncName2CodeInfo(fullFuncName string, line int) (codeInfo *CodeInfo) {
	if fullFuncName == "" {
		return &CodeInfo{}
	}
	lastSlashIndex := strings.LastIndex(fullFuncName, "/")
	basename := fullFuncName
	if lastSlashIndex > -1 {
		basename = fullFuncName[lastSlashIndex:]
	}
	firstDotIndex := lastSlashIndex + strings.Index(basename, ".")
	packageName := fullFuncName[:firstDotIndex]
	funcName := fullFuncName[firstDotIndex+1:]
	table := crc8.MakeTable(crc8.CRC8)
	packeCrc := crc8.Checksum([]byte(packageName), table)
	funcCrc := crc8.Checksum([]byte(funcName), table)
	code := fmt.Sprintf("%03d%03d%03d", packeCrc, funcCrc, line)
	codeInfo = &CodeInfo{
		Code:     code,
		Package:  packageName,
		Function: funcName,
		Line:     strconv.Itoa(line),
	}
	return
}

//FuncName2CodeInfo generate *CodeInfo from full function name
func (formatter *Formatter) SendToChain(errorCode *ErrorCode) (err error) {
	if formatter.Chan == nil {
		return
	}
	formatter.Chan <- errorCode
	return
}

//ModuleName help function, get mod package name from go.mod
func ModuleName(goModelfile string) (modName string, err error) {
	goModBytes, err := os.ReadFile(goModelfile)
	if err != nil {
		return
	}
	modName = modfile.ModulePath(goModBytes)
	return
}

type Causer interface {
	Cause() error
}

//GithubComPkgErrors github.com/pkg/errors package implementation
type GithubComPkgErrors struct{}
type GithubComPkgErrorsStackTracer interface {
	StackTrace() errors.StackTrace
}

//PCs implementation (*Formatter).PCs function
func (pkgErrors *GithubComPkgErrors) PCs(err error, pc []uintptr) (n int) {

	n = 0
	stackErr, ok := err.(GithubComPkgErrorsStackTracer)
	if ok {
		stack := stackErr.StackTrace()
		n = len(stack)
		for i, frame := range stack {
			pc[i] = uintptr(frame) - 1
		}
	}
	return n
}

//Cause implementation (*Formatter).Cause function
func (pkgErrors *GithubComPkgErrors) Cause(err error) error {
	targetErr := err

	for err != nil {
		cause, ok := err.(Causer)
		if !ok {
			break
		}
		err = cause.Cause()
		if err != nil {
			if code, ok := err.(*ErrorCode); ok {
				targetErr = code
			} else {
				pcArr := make([]uintptr, 32)
				n := pkgErrors.PCs(err, pcArr)
				if n > 0 {
					targetErr = err
				}
			}
		}
	}
	return targetErr
}
