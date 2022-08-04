package errorformatter

import (
	"fmt"
	"math"
	"testing"

	"hash/crc32"

	"github.com/pkg/errors"
	"github.com/sigurn/crc16"
)

func TestFormatError(t *testing.T) {
	include := []string{}
	exclude := []string{}
	formatter := New(include, exclude, nil, nil, nil, nil)
	testErr := errors.New("test")
	err := formatter.Msg(testErr.Error())
	fmt.Println(err)
}

func Error2() (err error) {
	msg := "error2"
	err = errors.New(msg)
	return
}

func TestError2(t *testing.T) {
	g := &GithubComPkgErrors{}
	include := []string{"errorformatter"}
	exclude := []string{}
	formatter := New(include, exclude, nil, g.PCs, g.Cause, nil)
	testErr := Error2()
	err := formatter.WrapError(testErr)
	fmt.Println(err)
}

func TestFmtErrorf(t *testing.T) {
	err := fmt.Errorf("test")
	fmt.Println("%w", err)
}

func TestXOR(t *testing.T) {
	poly := 4
	bpoly := fmt.Sprintf("%b", poly)
	data := 0
	bdata := fmt.Sprintf("%b", data)
	oxr := poly ^ data
	borx := fmt.Sprintf("%b", oxr)
	fmt.Println(bpoly)
	fmt.Println(bdata)
	fmt.Println(borx)
}

func TestCRC32(t *testing.T) {
	data := []byte("abeceonvee")
	crc32 := crc32.ChecksumIEEE(data)
	fmt.Printf("%U\n", crc32)
}
func TestCRC16(t *testing.T) {
	table := crc16.MakeTable(crc16.CRC16_MAXIM)
	//data := []byte("abeceonvee1")
	data := []byte("5a435a435a43")
	crc := crc16.Checksum(data, table)
	fmt.Printf("%x\n", crc)
}

var ValueArea = 10000 //256 * 256

func TestProbabilityN(t *testing.T) {
	n := 118
	p := ProbabilityN(n, 8)
	totalP := (1 - p)
	fmt.Println(totalP)
}

func ProbabilityN(n int, precision int) (out float64) {
	p := 1.0
	for i := 1; i <= n; i++ {
		p *= float64(ValueArea-i+1) / float64(ValueArea)
	}
	unit := math.Pow10(precision)
	out = float64(int64(p*unit+0.5)) / unit
	return out
}
