package response_cache

import "errors"
import "io"
import "sync"

var Uncacheable = errors.New("Uncacheable")

const (
	StatePending = iota
	StateReading
	StateSuccess
	StateFailure
	StateUncacheable
)

type progressTracker struct {
	cond sync.Cond
	state int
	length int64
	reason error
}

func newProgressTracker() *progressTracker {
	return &progressTracker{
		cond: sync.Cond{L: &sync.Mutex{}},
		state: StatePending,
	}
}

func (progress *progressTracker) Reading() {
	progress.cond.L.Lock()
	defer progress.cond.L.Unlock()
	progress.state = StateReading
	progress.cond.Broadcast()
}

func (progress *progressTracker) Wrote(n int64) {
	progress.cond.L.Lock()
	defer progress.cond.L.Unlock()
	progress.length += n
	progress.cond.Broadcast()
}

func (progress *progressTracker) Success() {
	progress.cond.L.Lock()
	defer progress.cond.L.Unlock()
	progress.state = StateSuccess
	progress.cond.Broadcast()
}

func (progress *progressTracker) Failure(reason error) {
	progress.cond.L.Lock()
	defer progress.cond.L.Unlock()
	progress.state = StateFailure
	if reason == Uncacheable {
		// use this object as is
		progress.reason = reason
	} else {
		// it's not clearly defined if error objects in general will be thread-safe, so convert to a dumb errorString struct
		progress.reason = errors.New(reason.Error())
	}
	progress.cond.Broadcast()
}

func (progress *progressTracker) WaitForResponse() error {
	progress.cond.L.Lock()
	defer progress.cond.L.Unlock()

	for {
		switch (progress.state) {
		case StatePending:
			progress.cond.Wait()

		case StateReading, StateSuccess:
			return nil

		case StateFailure:
			return progress.reason
		}
	}
}

func (progress *progressTracker) WaitForMore(position int64) error {
	progress.cond.L.Lock()
	defer progress.cond.L.Unlock()

	for {
		switch (progress.state) {
		case StatePending:
			return errors.New("waitForMore used before the header was complete")

		case StateReading:
			if (progress.length > position) {
				return nil
			}
			progress.cond.Wait()

		case StateSuccess:
			if (progress.length > position) {
				return nil
			}
			return io.EOF

		case StateFailure:
			return progress.reason
		}
	}
}
