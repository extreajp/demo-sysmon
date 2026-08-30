package procfs

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type FS struct {
	Proc string
	Sys  string
}

func New(proc, sys string) FS {
	if proc == "" {
		proc = "/proc"
	}
	if sys == "" {
		sys = "/sys"
	}
	return FS{Proc: proc, Sys: sys}
}

func (fs FS) ReadFile(rel string) (string, error) {
	b, err := os.ReadFile(rel)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (fs FS) ProcFile(name string) (string, error) {
	return fs.ReadFile(filepath.Join(fs.Proc, name))
}

func (fs FS) SysFile(name string) (string, error) {
	return fs.ReadFile(filepath.Join(fs.Sys, name))
}

func Fields(s string) []string {
	return strings.Fields(s)
}

func ParseUint(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}

func ParseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func FirstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
