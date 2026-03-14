//go:build ios

package memory

/*
#include <os/log.h>
#include <stdlib.h>

// 使用纯 C 接口，完美避开 Objective-C 的编译错误
static inline void iOSLog(const char* msg) {
    // OS_LOG_DEFAULT: 系统默认日志
    // OS_LOG_TYPE_DEFAULT: 默认类型
    // "%{public}s": 关键点！强制显示内容，不显示 <private>
    os_log_with_type(OS_LOG_DEFAULT, OS_LOG_TYPE_DEFAULT, "%{public}s", msg);
}
*/
import "C"
import (
	"log"
	"runtime"
	"runtime/debug"
	"time"
	"unsafe"
)

// 自定义一个 Writer，让所有的 log.Printf 都走 C 的 iOSLog
type iosWriter struct{}

func (w *iosWriter) Write(p []byte) (n int, err error) {
	cStr := C.CString(string(p))
	defer C.free(unsafe.Pointer(cStr))
	C.iOSLog(cStr)
	return len(p), nil
}

const (
	interval = 1
	// 30M
	maxMemory   = 30 * 1024 * 1024
	warnLevel   = 1 * 1024 * 1024 // 接近阈值
	dangerLevel = 2 * 1024 * 1024 // 危险阈值
)

func forceFree(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			debug.FreeOSMemory()
		}
	}()
}

func InitForceFree() {

	// --- 关键修改点：重定向 log 输出 ---
	log.SetOutput(&iosWriter{})
	log.SetFlags(0) // 既然 NSLog 自带时间戳，可以关掉 log 的默认时间戳

	debug.SetGCPercent(10)
	debug.SetMemoryLimit(maxMemory)
	duration := time.Duration(interval) * time.Second
	forceFree(duration)
	startMemMonitor()
}
func startMemMonitor() {
	go func() {
		var lastPauseNS uint64
		for {
			time.Sleep(time.Second * 10)

			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			// 当前进程从 OS 申请的内存
			used := m.Sys
			// 接近阈值
			if used >= warnLevel && used < dangerLevel {
				log.Printf("[Warning] dep: [内存监控] 内存使用量接近警告阈值:%dMB", used/1024/1024)
			}
			// 超过危险阈值，打印详细 GC/堆信息
			if used >= dangerLevel {
				// 本次前后 GC 次数 / 暂停时间
				pauseCount := m.NumGC
				pauseTotal := time.Duration(m.PauseTotalNs)
				recentPauses := []time.Duration{}
				// 只拉最近几次 GC 的暂停时间（简单版）
				for i := uint32(0); i < 4 && i < pauseCount; i++ {
					idx := (pauseCount - 1 - i) % uint32(len(m.PauseNs))
					recentPauses = append(recentPauses, time.Duration(m.PauseNs[idx]))
				}
				log.Printf("[Warning] dep: [内存释放] <<<< 释放前状态 >>>>| 已分配内存: %d MiB | 总分配内存: %d MiB | 系统内存: %d MiB | GC次数: %d | 下次GC阈值: %d MiB | 堆内存: %d MiB | 堆对象: %d | 堆空闲: %d MiB | 堆使用: %d MiB | 栈内存: %d MiB | 最后GC时间: %s",
					m.Alloc/1024/1024,
					m.TotalAlloc/1024/1024,
					m.Sys/1024/1024,
					m.NumGC,
					m.NextGC/1024/1024,
					m.HeapSys/1024/1024,
					m.HeapObjects,
					m.HeapIdle/1024/1024,
					m.HeapInuse/1024/1024,
					m.StackSys/1024/1024,
					time.Unix(0, int64(m.LastGC)).Format("2006-01-02 15:04:05"),
				)
				// 强制一次 GC + FreeOSMemory
				debug.FreeOSMemory()
				// 再读一次
				runtime.ReadMemStats(&m)
				log.Printf("[Warning] dep: [内存释放] >>>> 释放后状态 <<<<| 已分配内存: %d MiB | 总分配内存: %d MiB | 系统内存: %d MiB | GC次数: %d | 下次GC阈值: %d MiB | 堆内存: %d MiB | 堆对象: %d | 堆空闲: %d MiB | 堆使用: %d MiB | 栈内存: %d MiB | 最后GC时间: %s",
					m.Alloc/1024/1024,
					m.TotalAlloc/1024/1024,
					m.Sys/1024/1024,
					m.NumGC,
					m.NextGC/1024/1024,
					m.HeapSys/1024/1024,
					m.HeapObjects,
					m.HeapIdle/1024/1024,
					m.HeapInuse/1024/1024,
					m.StackSys/1024/1024,
					time.Unix(0, int64(m.LastGC)).Format("2006-01-02 15:04:05"),
				)
				// GC 统计
				if m.PauseTotalNs != lastPauseNS {
					log.Printf("[Warning] dep: [GC统计] ----暂停次数: %d | 总暂停时间: %s", m.NumGC, pauseTotal)
					lastPauseNS = m.PauseTotalNs
				}
			}
		}
	}()
}
