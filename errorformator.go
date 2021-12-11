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
)

const (
	SEPARATOR_DEFAULT = '#'
	WITH_CALL_CHAIN   = false
	DEPTH_DEFAULT     = 2
)

type ErrorFormator struct {
	Filename      string `json:"filename"`
	mutex         sync.Mutex
	Separator     byte `json:"Separator"`
	WithCallChain bool `json:"withCallChain"`
	Depth         int  `json:"depth"`
}
type ErrMap struct {
	BusinessCode string `json:"businessCode"`
	Package      string `json:"package"`
	FunctionName string `json:"functionName"`
	Line         string `json:"line"`
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

	errorFormator = &ErrorFormator{
		Filename:      fileName,
		Separator:     SEPARATOR_DEFAULT,
		WithCallChain: WITH_CALL_CHAIN,
		Depth:         DEPTH_DEFAULT,
	}
	return
}

//FormatError generate format error message
func (errorFormator *ErrorFormator) Format(msg string, args ...int) (err error) {
	httpCode := 500
	businessCode := "000000000"
	formatTpl := "%c%d:%s%c%s"
	if len(args) >= 2 {
		httpCode = args[0]
		businessCode = strconv.Itoa(args[1])
		err = fmt.Errorf(formatTpl, errorFormator.Separator, httpCode, businessCode, errorFormator.Separator, msg)
		return
	}
	if len(args) == 1 {
		httpCode = args[0]
	}
	if !errorFormator.WithCallChain { // Detect whether it is in target format
		if msg[0] == byte(errorFormator.Separator) {
			return fmt.Errorf(msg)
		}
	}

	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(errorFormator.Depth, pc)
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
			Line:         strconv.Itoa(line),
		}
		errorFormator.updateMapFile(errMap)
	}
	err = fmt.Errorf(formatTpl, errorFormator.Separator, httpCode, businessCode, errorFormator.Separator, msg)
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

var defaultErrorFormator = &ErrorFormator{
	Separator:     SEPARATOR_DEFAULT,
	WithCallChain: WITH_CALL_CHAIN,
	Depth:         3,
}

//Format format the error
func Format(msg string, args ...int) (err error) {
	err = defaultErrorFormator.Format(msg, args...)
	return
}
