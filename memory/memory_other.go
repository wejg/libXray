//go:build !ios

package memory

import "runtime/debug"

func InitForceFree() {}

func SetMemoryLimit(memoryMB int) {
	if memoryMB <= 0 {
		memoryMB = 30
	}
	debug.SetMemoryLimit(int64(memoryMB) * 1024 * 1024)
}
