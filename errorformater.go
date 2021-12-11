package errorformater

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
)

type ErrorFormator struct {
	Filename string `json:"filename"`
	mutex    sync.Mutex
}

func New(fileName string) (errorFormator *ErrorFormator, err error) {
	err = Mkdir(filepath.Dir(fileName))
	if err != nil {
		return
	}
	f, err := os.Create(fileName)
	if err != nil {
		return
	}
	defer f.Close()
	errorFormator = &ErrorFormator{
		Filename: fileName,
	}
	return
}

type ErrMap struct {
	BusinessCode string `json:"businessCode"`
	Package      string `json:"package"`
	FunctionName string `json:"functionName"`
	Line         int    `json:"line"`
}

//FormatError generate format error message
func (errorFormator *ErrorFormator) FormatError(msg string, args ...int) (err error) {
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
	fullname := f.Name()
	lastIndex := strings.LastIndex(fullname, ".")
	packageName := fullname[:lastIndex]
	funcName := fullname[lastIndex+1:]
	table := crc8.MakeTable(crc8.CRC8)
	packeCrc := crc8.Checksum([]byte(packageName), table)
	funcCrc := crc8.Checksum([]byte(funcName), table)
	businessCode = fmt.Sprintf("%03d%03d%03d", packeCrc, funcCrc, line)
	if errorFormator.Filename != "" {
		errMap := &ErrMap{
			BusinessCode: businessCode,
			Package:      packageName,
			FunctionName: funcName,
			Line:         line,
		}
		errorFormator.updateMapFile(errMap)
	}
	err = fmt.Errorf("%d:%s:%s", httpCode, businessCode, msg)
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

var defaultErrorFormator = &ErrorFormator{}

//FormatError 格式化错误
func FormatError(msg string, args ...int) (err error) {
	err = defaultErrorFormator.FormatError(msg, args...)
	return
}
