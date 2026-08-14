//go:build windows

package gamesrv

import "os"

func fileOwner(fi os.FileInfo) (string, bool) { return "", false }

func lookupUsernameByUID(uid string) (string, error) { return "", nil }
