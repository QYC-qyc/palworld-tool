//go:build windows

package gamesrv

import (
	"errors"
	"os"
)

// Windows 上不需要 root 降权逻辑（面板直接运行游戏进程）。
func fileOwner(fi os.FileInfo) (string, bool) { return "", false }

func lookupUsernameByUID(uid string) (string, error) {
	return "", errors.New("Windows 不支持按 UID 查找用户")
}
