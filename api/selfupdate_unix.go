//go:build !windows

package api

import "syscall"

func sysProcAttrSetsid() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
