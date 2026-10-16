//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package fibe

import (
	"os"

	"golang.org/x/sys/unix"
)

func lockStoreFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_EX)
}

func unlockStoreFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}
