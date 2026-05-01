package projection

import "github.com/cockroachdb/errors"

var (
	ErrRunnerStopped    = errors.New("projection: runner stopped")
	ErrNilHandler       = errors.New("projection: nil handler")
	ErrDuplicateHandler = errors.New("projection: duplicate handler")
	ErrCheckpointLoad   = errors.New("projection: checkpoint load failed")
	ErrStoreLoad        = errors.New("projection: store load failed")
	ErrNilStore         = errors.New("projection: nil store")
	ErrNilBus           = errors.New("projection: nil bus")
	ErrNilCheckpoint    = errors.New("projection: nil checkpoint store")
	ErrNoProjections    = errors.New("projection: no projections registered")
)
