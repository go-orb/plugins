// Package wg implements a double waitgroup that can be used to keep track
// of a global waitgroup at the same time. Adding and stubstracting from
// the waitgroup will impact both the global and local waitgroups, but waiting
// will only wait for the local waitgroup to be done.
//
// It allows you to add to a runtime used globally, but only wait for your added
// routines, and not all other routines, while at the same time giving the
// safety that the global process knows about your routines.
package wg

import (
	"sync"
)

// WaitGroup for global management of connections.
type WaitGroup struct {
	local  sync.WaitGroup
	global *sync.WaitGroup
}

// NewWaitGroup returns a new double waitgroup for global management of processes.
func NewWaitGroup(gWg *sync.WaitGroup) *WaitGroup {
	return &WaitGroup{
		global: gWg,
	}
}

// Add will add to both the global and local waitgroups.
func (w *WaitGroup) Add(i int) {
	w.local.Add(i)

	if w.global != nil {
		w.global.Add(i)
	}
}

// Done will subtract one from both the global and local waitgroup.
func (w *WaitGroup) Done() {
	w.local.Done()

	if w.global != nil {
		w.global.Done()
	}
}

// Wait will only wait for the local waitgroup to complete.
func (w *WaitGroup) Wait() {
	w.local.Wait()
}
