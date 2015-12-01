package conveyor

import (
	"time"

	"code.google.com/p/go-uuid/uuid"
)

const (
	// DefaultTimeout is the default amount of time to wait for a build
	// to complete before cancelling it.
	DefaultTimeout = 20 * time.Minute
)

// newID returns a new unique identifier.
var newID = uuid.New
