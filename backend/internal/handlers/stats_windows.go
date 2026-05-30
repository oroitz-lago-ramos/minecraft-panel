//go:build windows

package handlers

func getDiskInfo() (used, total uint64) {
	return 0, 0
}