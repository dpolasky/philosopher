//go:build windows
// +build windows

package wrk

import (
	"errors"
	"syscall"

	"philosopher/lib/msg"
)

// HideFile makes the .meta folder hidden on Windows
func HideFile(filename string) {
	filenameW, e := syscall.UTF16PtrFromString(filename)
	if e != nil {
		msg.Custom(e, "error")
	}
	e = syscall.SetFileAttributes(filenameW, syscall.FILE_ATTRIBUTE_HIDDEN)
	if e != nil {
		msg.Custom(errors.New("Cannot hide file"), "fatal")
	}
	return
}
