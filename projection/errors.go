package projection

import "github.com/cockroachdb/errors"

var (
	ErrNilHandler     = errors.New("projection: nil handler")
	ErrNilBus         = errors.New("projection: nil bus")
	ErrNilCheckpoint  = errors.New("projection: nil checkpoint store")
	ErrNoProjections  = errors.New("projection: no projections registered")
)
