//go:build !windows

package gamesrv

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
)

// fileOwner 返回文件属主的 UID（字符串形式）
func fileOwner(fi os.FileInfo) (string, bool) {
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		return strconv.FormatUint(uint64(stat.Uid), 10), true
	}
	return "", false
}

// lookupUsernameByUID 根据 UID 查找用户名
func lookupUsernameByUID(uid string) (string, error) {
	u, err := user.LookupId(uid)
	if err != nil {
		return "", err
	}
	return u.Username, nil
}
