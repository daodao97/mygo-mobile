//go:build android && cgo

package backend

import "github.com/egoist/mygo/internal/platform"

// Power is the unsupported backend's, except KeepRunning: Android freezes
// and kills a backgrounded app's process, and a foreground service is what
// keeps it running (see MygoKeepRunning.kt).
func (b *Backend) Power() platform.Power { return androidPower{Power: b.Backend.Power(), b: b} }

type androidPower struct {
	platform.Power
	b *Backend
}

// keepRunning holds the outstanding KeepRunning reasons, in call order. It
// belongs to the pump: mygo calls KeepRunning and its release on the main
// thread.
var keepRunning struct {
	next    int
	order   []int
	reasons map[int]string
}

// KeepRunning starts the host's foreground service, whose ongoing
// notification shows the most recent outstanding reason. The service stops
// when every release has been called. A process the system or the person
// ends anyway is not started again.
func (p androidPower) KeepRunning(reason string) func() {
	if keepRunning.reasons == nil {
		keepRunning.reasons = map[int]string{}
	}
	keepRunning.next++
	id := keepRunning.next
	keepRunning.order = append(keepRunning.order, id)
	keepRunning.reasons[id] = reason
	p.b.updateKeepRunning()
	return func() {
		if _, ok := keepRunning.reasons[id]; !ok {
			return
		}
		delete(keepRunning.reasons, id)
		for i, v := range keepRunning.order {
			if v == id {
				keepRunning.order = append(keepRunning.order[:i:i], keepRunning.order[i+1:]...)
				break
			}
		}
		p.b.updateKeepRunning()
	}
}

func (b *Backend) updateKeepRunning() {
	var reason any // null stops the service
	if n := len(keepRunning.order); n > 0 {
		reason = keepRunning.reasons[keepRunning.order[n-1]]
	}
	_ = b.NativeFire("keep-running", reason)
}
