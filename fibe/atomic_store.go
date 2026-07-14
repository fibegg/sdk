package fibe

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var storeLocks sync.Map

func storeLock(path string) *sync.Mutex {
	clean := filepath.Clean(path)
	lock, _ := storeLocks.LoadOrStore(clean, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func withStoreLock[T any](path string, operation func() (T, error)) (result T, err error) {
	processLock := storeLock(path)
	processLock.Lock()
	defer processLock.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return result, err
	}
	file, err := openStoreLockFile(path + ".lock")
	if err != nil {
		return result, err
	}
	locked := false
	defer func() {
		var cleanupErr error
		if locked {
			cleanupErr = unlockStoreFile(file)
		}
		cleanupErr = errors.Join(cleanupErr, file.Close())
		if err == nil && cleanupErr != nil {
			err = cleanupErr
		}
	}()
	if err := lockStoreFile(file); err != nil {
		return result, err
	}
	locked = true
	return operation()
}

func withStoreLockError(path string, operation func() error) error {
	_, err := withStoreLock(path, func() (struct{}, error) {
		return struct{}{}, operation()
	})
	return err
}

func openStoreLockFile(path string) (*os.File, error) {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("refusing to use non-regular store lock file %s", path)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	// #nosec G304 -- path is the private sibling lock for the configured store.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	closeOnError := func(err error) (*os.File, error) {
		return nil, errors.Join(err, file.Close())
	}
	opened, err := file.Stat()
	if err != nil {
		return closeOnError(err)
	}
	current, err := os.Lstat(path)
	if err != nil {
		return closeOnError(err)
	}
	if current.Mode()&os.ModeSymlink != 0 || !opened.Mode().IsRegular() || !current.Mode().IsRegular() || !os.SameFile(opened, current) {
		return closeOnError(fmt.Errorf("refusing to use replaced or non-regular store lock file %s", path))
	}
	if err := file.Chmod(0o600); err != nil {
		return closeOnError(err)
	}
	return file, nil
}

func atomicWriteFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	// #nosec G302 -- 0700 is the required restrictive mode for a secret-bearing directory.
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("refusing to replace non-regular store file %s", path)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	removeTemp := true
	defer func() {
		_ = tmp.Close()
		if removeTemp {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(mode); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	removeTemp = false
	if err := os.Chmod(path, mode); err != nil {
		return err
	}
	// #nosec G304 -- dir is the validated parent of the configured atomic store path.
	dirFile, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer dirFile.Close()
	return dirFile.Sync()
}

func readStoreFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("refusing to read non-regular store file %s", path)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return nil, err
	}
	// #nosec G302 -- 0700 is the required restrictive mode for a secret-bearing directory.
	if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	// #nosec G304 -- the caller-supplied store path passed the non-symlink regular-file checks above.
	return os.ReadFile(path)
}
