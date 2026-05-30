//go:build linux

package handlers

import "golang.org/x/sys/unix"

func getDiskInfo() (used, total uint64) {
	var stat unix.Statfs_t
	if err := unix.Statfs("/", &stat); err != nil {
		return
	}
	total = stat.Blocks * uint64(stat.Bsize) / 1024 / 1024 / 1024
	used = (stat.Blocks - stat.Bfree) * uint64(stat.Bsize) / 1024 / 1024 / 1024
	return
}
