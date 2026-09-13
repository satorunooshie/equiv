// Package clock provides the small time source abstraction used by caches.
package clock

import "time"

type (
	Clock interface{ Now() time.Time }
	Real  struct{}
)

func (Real) Now() time.Time { return time.Now() }
