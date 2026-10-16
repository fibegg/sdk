//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !windows

package fibe

import "os"

func lockStoreFile(_ *os.File) error {
	return nil
}

func unlockStoreFile(_ *os.File) error {
	return nil
}
