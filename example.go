package errorformatter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

var ErrorMapFile = "errorMapFile.json"
var Include []string
var Exclude []string
var PackageHttpstatusMap map[string]int
var FuncHttpStatusMap map[string]int

var errFormatter *Formatter
var errFormatorOnce sync.Once

// GetErrFormatter is a error formatter
func GetErrFormatter() *Formatter {
	if errFormatter == nil {
		InitErrFormatter()
	}
	return errFormatter
}

var errorChain = make(chan *ErrorCode, 10)

func InitErrFormatter() *Formatter {
	errFormatorOnce.Do(func() {
		githubComPkgErrors := &GithubComPkgErrors{}

		errFormatter = New(Include, Exclude, GetFuncHttpStatus, githubComPkgErrors.PCs, githubComPkgErrors.Cause, errorChain)
		if ErrorMapFile != "" {
			err := os.MkdirAll(filepath.Dir(ErrorMapFile), os.ModePerm)
			if err != nil {
				panic(err)
			}
		}
		SaveCodeInfo()
	})
	return errFormatter
}

//GetFuncHttpStatus 配置每个方法返回错误时，http 默认状态码
func GetFuncHttpStatus(packageName string, funcName string) (int, bool) {
	httpStatus, ok := PackageHttpstatusMap[packageName]
	if ok {
		return httpStatus, true
	}
	fullName := fmt.Sprintf("%s.%s", packageName, funcName)
	httpStatus, ok = FuncHttpStatusMap[fullName]
	if ok {
		return httpStatus, true
	}
	return http.StatusInternalServerError, true
}

// GetErrorChain export errorChain
func GetErrorChain() <-chan *ErrorCode {
	return errorChain
}

//SaveCodeInfo 存储错误码信息
func SaveCodeInfo() {

	ch := GetErrorChain()
	codeCodeTable := make(map[string][]*CodeInfo)
	var filename string
	var err error
	if ErrorMapFile != "" {
		filename, err = filepath.Abs(ErrorMapFile)
		if err != nil {
			panic(err)
		}
	}

	if filename != "" {
		b, err := os.ReadFile(filename)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(b, &codeCodeTable)
		if err != nil {
			codeCodeTable = make(map[string][]*CodeInfo)
		}
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println(err) // drop err
			}
		}()

		for err := range ch {
			codeCodeTable[err.Code] = err.TraceInfo()
			if filename != "" {
				b, err := json.Marshal(codeCodeTable)
				if err == nil {
					os.WriteFile(filename, b, os.ModePerm)
				}
			}
		}
	}()

}
