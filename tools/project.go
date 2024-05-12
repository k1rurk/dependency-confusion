package tools

import (
	"path/filepath"
	"runtime"
)

// GetDirectoryProject get the current working directory of main project
func GetDirectoryProject() string {
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	p := filepath.Clean(filepath.Join(basepath, ".."))
	return p
}