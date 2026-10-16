package storage

import (
	"os"
	"syscall"
)

// LockDir takes an exclusive advisory lock on dir, blocking until it is
// available, and returns a func that releases it. Locking the directory
// (rather than a sidecar lockfile) leaves nothing behind in the project's
// repo, and — unlike locking the data file itself — survives
// WriteFileAtomic replacing that file's inode via rename.
//
// The lock is cross-process: it's what serializes writes between the HTTP
// server and every separately-launched MCP server process sharing a
// project. It only coordinates processes that see the same filesystem
// (e.g. all inside one container), which is how this app is deployed.
func LockDir(dir string) (unlock func(), err error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
}
