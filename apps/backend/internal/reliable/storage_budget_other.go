//go:build !windows

package reliable

import "golang.org/x/sys/unix"

func storageAvailableBytes(path string) (uint64, error) {
	var stat unix.Statfs_t
	err := unix.Statfs(path, &stat)
	return stat.Bavail * uint64(stat.Bsize), err
}
