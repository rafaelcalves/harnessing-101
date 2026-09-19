// Package clock is the implementation of ports.Clock (outbound port 9).
package clock

import "time"

// System is the real wall/monotonic clock. MonotonicNow is elapsed time
// since this process started; it is never persisted and never meant to
// survive a restart (boundaries.md: "persisted wall times never
// substitute for a continuous monotonic clock across restart").
type System struct {
	start time.Time
}

func NewSystem() System {
	return System{start: time.Now()}
}

func (s System) WallNow() time.Time {
	return time.Now().UTC()
}

func (s System) MonotonicNow() time.Duration {
	return time.Since(s.start)
}
