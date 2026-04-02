package tensile

import (
	"log"
	"sync"
)

type TestingT struct {
	lock    sync.RWMutex
	errored bool
}

func (v *TestingT) Errorf(format string, args ...any) {
	v.lock.Lock()
	defer v.lock.Unlock()
	v.errored = true
	log.Printf(format, args...)
}

func (v *TestingT) Errored() bool {
	v.lock.RLock()
	defer v.lock.RUnlock()
	return v.errored
}
