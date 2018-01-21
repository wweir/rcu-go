package rcu

import (
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

const rwmutexMaxReaders = 1 << 30

func RCU(rw *sync.RWMutex, wait int) error {
	v := reflect.ValueOf(rw)
	muV := v.Elem().FieldByName("w")
	mu := (*sync.Mutex)(unsafe.Pointer(muV.UnsafeAddr()))
	readerCountV := v.Elem().FieldByName("readerCount")
	readerCount := (*int32)(unsafe.Pointer(readerCountV.UnsafeAddr()))
	readerWaitV := v.Elem().FieldByName("readerWait")
	readerWait := (*int32)(unsafe.Pointer(readerWaitV.UnsafeAddr()))

	mu.Lock()
	if r := atomic.AddInt32(readerCount, -1); r < 0 {
		if r+1 == 0 || r+1 == -rwmutexMaxReaders {
			panic("sync: RUnlock of unlocked RWMutex")
		}
		atomic.AddInt32(readerWait, -1)
	}

	atomic.AddInt32(readerCount, -rwmutexMaxReaders)
	r := *readerCount + rwmutexMaxReaders

	if r == 0 || atomic.AddInt32(readerWait, r) == 0 {
		return nil
	}

	var fibLast, fibNow time.Duration = 1, 1
	for 0 != *readerWait {
		switch {
		case wait == 0:
			runtime.Gosched()
		case wait < 0:
			fibLast, fibNow = fibNow, fibLast+fibNow
			time.Sleep(fibNow * time.Millisecond)
		default:
			time.Sleep(time.Duration(wait))
		}
	}

	return nil
}
