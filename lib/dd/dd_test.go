package dd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDiskDump(t *testing.T) {
	pwd, _ := os.Getwd()
	input := filepath.Join(pwd, "testfile")
	output := filepath.Join(pwd, "testfile1")
	err := DiskDump(input, output)
	fmt.Println("finish exec", err)

}
