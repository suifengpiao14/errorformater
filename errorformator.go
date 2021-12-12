package errorformator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/sigurn/crc8"
	"golang.org/x/mod/modfile"
)

const (
	SEPARATOR_DEFAULT = '#'
	WITH_CALL_CHAIN   = false
	SKIP_DEFAULT      = 2
	MOD_FILE_DEFAULT  = "go.mod"
)

type ErrorFormator struct {
	Filename          string `json:"filename"`
	mutex             sync.Mutex
	WithCallChain     bool                                                  `json:"withCallChain"`
	Skip              int                                                   `json:"skip"`
	PackageNamePrefix string                                                `json:"packageName"`
	GetFuncHttpStatus func(packageName string, funcName string) (int, bool) `json:"-"`
	GetPCs            func(err error, pc []uintptr) (n int)                 `json:"-"`
}
type ErrMap struct {
	BusinessCode string `json:"businessCode"`
	Package      string `json:"package"`
	FunctionName string `json:"functionName"`
	Line         string `json:"line"`
}

type BusinessCodeError struct {
	HttpStatus   int    `json:"-"`
	BusinessCode string `json:"code"`
	Msg          string `json:"msg"`
	cause        error
}

var formatTpl = "%c%d:%s%c%s"

func (e *BusinessCodeError) Error() string {

	msg := fmt.Sprintf(formatTpl, SEPARATOR_DEFAULT, e.HttpStatus, e.BusinessCode, SEPARATOR_DEFAULT, e.Msg)
	return msg
}
func (e *BusinessCodeError) Cause() error { return e.cause }

// ParseMsg parse string to *BusinessCodeErr
func (e *BusinessCodeError) ParseMsg(msg string) (ok bool) {
	ok = false
	if msg[0] != byte(SEPARATOR_DEFAULT) {
		return
	}
	arr := strings.SplitN(msg, string(SEPARATOR_DEFAULT), 3)
	if len(arr) < 3 {
		return
	}
	codeArr := strings.SplitN(arr[1], ":", 2)
	if len(codeArr) < 2 {
		return
	}
	httpCode, err := strconv.Atoi(codeArr[0])
	if err != nil {
		return
	}
	e.HttpStatus = httpCode
	e.BusinessCode = codeArr[1]
	e.Msg = arr[2]
	ok = true
	return
}

func New(fileName string) (errorFormator *ErrorFormator, err error) {
	err = Mkdir(filepath.Dir(fileName))
	if err != nil {
		return
	}
	if !IsExist(fileName) { // check file permision
		f, err := os.Create(fileName)
		if err != nil {
			return nil, err
		}
		f.Close()
		fd, err := os.Open(fileName)
		if err != nil {
			return nil, err
		}
		fd.Close()
	}
	packageName, _ := GetModuleName(MOD_FILE_DEFAULT)
	errorFormator = &ErrorFormator{
		Filename:          fileName,
		WithCallChain:     WITH_CALL_CHAIN,
		Skip:              SKIP_DEFAULT,
		PackageNamePrefix: packageName,
	}
	return
}

func (errorFormator *ErrorFormator) Cause(err error) error {
	targetErr := err
	type causer interface {
		Cause() error
	}
	for err != nil {
		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
		if businessCode, ok := err.(*BusinessCodeError); ok {
			targetErr = businessCode
		} else if errorFormator.GetPCs != nil {
			pcArr := make([]uintptr, 32)
			n := errorFormator.GetPCs(err, pcArr)
			if n > 0 {
				targetErr = err
			}
		}
	}
	return targetErr
}

//FormatError generate format error message
func (errorFormator *ErrorFormator) FormatMsg(msg string, args ...int) (err *BusinessCodeError) {
	httpCode := 500
	businessCode := "000000000"
	if len(args) >= 2 {
		httpCode = args[0]
		businessCode = strconv.Itoa(args[1])
		err = &BusinessCodeError{
			HttpStatus:   httpCode,
			BusinessCode: businessCode,
			Msg:          msg,
		}
		return
	}
	if len(args) == 1 {
		httpCode = args[0]
	}
	if !errorFormator.WithCallChain { // Detect whether it is in target format
		e := &BusinessCodeError{}
		ok := e.ParseMsg(msg)
		if ok {
			return e
		}
	}

	pcArr := make([]uintptr, 32) // at least 1 entry needed
	n := runtime.Callers(errorFormator.Skip, pcArr)
	frames := runtime.CallersFrames(pcArr[:n])
	businessCode, packageName, funcName, line := errorFormator.ParseFrames(frames)
	if errorFormator.Filename != "" {
		errMap := &ErrMap{
			BusinessCode: businessCode,
			Package:      packageName,
			FunctionName: funcName,
			Line:         strconv.Itoa(line),
		}
		go errorFormator.updateMapFile(errMap)
	}
	if errorFormator.GetFuncHttpStatus != nil {
		code, ok := errorFormator.GetFuncHttpStatus(packageName, funcName)
		if ok {
			httpCode = code
		}
	}
	err = &BusinessCodeError{
		HttpStatus:   httpCode,
		BusinessCode: businessCode,
		Msg:          msg,
	}
	return
}

func (errorFormator *ErrorFormator) FormatError(err error) (newErr *BusinessCodeError) {
	err = errorFormator.Cause(err)
	e, ok := err.(*BusinessCodeError)
	if ok {
		return e
	}
	httpCode := 500
	pcArr := make([]uintptr, 32) // at least 1 entry needed
	var frames *runtime.Frames
	n := 0
	if errorFormator.GetPCs != nil {
		n = errorFormator.GetPCs(err, pcArr)
	} else {
		n = runtime.Callers(errorFormator.Skip, pcArr)

	}
	frames = runtime.CallersFrames(pcArr[:n])
	businessCode, packageName, funcName, line := errorFormator.ParseFrames(frames)
	if errorFormator.Filename != "" {
		errMap := &ErrMap{
			BusinessCode: businessCode,
			Package:      packageName,
			FunctionName: funcName,
			Line:         strconv.Itoa(line),
		}
		errorFormator.updateMapFile(errMap)
	}
	if errorFormator.GetFuncHttpStatus != nil {
		code, ok := errorFormator.GetFuncHttpStatus(packageName, funcName)
		if ok {
			httpCode = code
		}
	}
	newErr = &BusinessCodeError{
		HttpStatus:   httpCode,
		BusinessCode: businessCode,
		Msg:          err.Error(),
		cause:        err,
	}
	return
}

func (errorFormator *ErrorFormator) ParseFrames(frames *runtime.Frames) (businessCode string, packageName string, funcName string, line int) {
	fullname := ""
	for {
		frame, hasNext := frames.Next()
		if !hasNext {
			break
		}
		fullname = frame.Function
		line = frame.Line
		if errorFormator.PackageNamePrefix == "" {
			break
		}
		// Find first information of interest
		if strings.Contains(fullname, errorFormator.PackageNamePrefix) {
			break
		}
	}
	lastSlashIndex := strings.LastIndex(fullname, "/")
	basename := fullname[lastSlashIndex:]
	firstDotIndex := lastSlashIndex + strings.Index(basename, ".")
	packageName = fullname[:firstDotIndex]
	funcName = fullname[firstDotIndex+1:]
	table := crc8.MakeTable(crc8.CRC8)
	packeCrc := crc8.Checksum([]byte(packageName), table)
	funcCrc := crc8.Checksum([]byte(funcName), table)
	businessCode = fmt.Sprintf("%03d%03d%03d", packeCrc, funcCrc, line)
	return
}

func (errorFormator *ErrorFormator) updateMapFile(errMap *ErrMap) (err error) {
	errorFormator.mutex.Lock()
	defer errorFormator.mutex.Unlock()
	b, err := os.ReadFile(errorFormator.Filename)
	if err != nil {
		return
	}
	errMapTable := map[string]*ErrMap{}
	if len(b) > 0 {
		err = json.Unmarshal(b, &errMapTable)
		if err != nil {
			return
		}
	}

	_, ok := errMapTable[errMap.BusinessCode]
	if ok {
		return
	}
	errMapTable[errMap.BusinessCode] = errMap
	jsonByte, err := json.Marshal(errMapTable)
	if err != nil {
		return
	}
	err = os.WriteFile(errorFormator.Filename, jsonByte, os.ModePerm)
	if err != nil {
		return
	}
	return
}

func IsExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return true
}
func Mkdir(filePath string) error {
	if !IsExist(filePath) {
		err := os.MkdirAll(filePath, os.ModePerm)
		return err
	}
	return nil
}

var modPackageName, _ = GetModuleName(MOD_FILE_DEFAULT)
var defaultErrorFormator = &ErrorFormator{
	WithCallChain:     WITH_CALL_CHAIN,
	Skip:              3,
	PackageNamePrefix: modPackageName,
}

//Format format the error
func Format(msg string, args ...int) (err error) {
	err = defaultErrorFormator.FormatMsg(msg, args...)
	return
}

func FormatError(err error) (newErr error) {
	newErr = defaultErrorFormator.FormatError(err)
	return
}

//GetModuleName get mod package name from go.mod
func GetModuleName(goModelfile string) (modName string, err error) {
	goModBytes, err := os.ReadFile(goModelfile)
	if err != nil {
		return
	}
	modName = modfile.ModulePath(goModBytes)
	return
}
