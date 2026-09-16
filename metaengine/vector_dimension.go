package metaengine

import (
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrVectorDimensionMismatch is the sentinel inside the error engines return
// when an embedding violates the collection's dimension lock (ADR-0140
// companions: first insert establishes the collection's dimension; later
// inserts of a different dimension — including zero-dimension embeddings —
// are rejected instead of silently scoring truncated vectors). Match with
// errors.Is; the error is errorfamily Rejection-classified.
var ErrVectorDimensionMismatch error = errorfamily.NewRejection(
	"metaengine.vector_dimension_mismatch",
	"embedding dimension violates the collection's dimension lock",
)

// CheckVectorDimension enforces the first-insert-wins dimension lock shared
// by every engine's VectorInsert: established is the stored dimension of the
// collection's existing vectors (0 = empty collection, the insert
// establishes the dimension), incoming the new embedding's dimension.
// Returns nil when the insert may proceed.
//
// Collections that predate the lock and already hold mixed dimensions
// normalize to whatever the engine's established-dimension read returns
// (its first stored record); further divergence is rejected from there.
func CheckVectorDimension(collection string, established, incoming int) error {
	if incoming <= 0 {
		return errorfamily.WrapRejection(ErrVectorDimensionMismatch,
			"metaengine.vector_dimension_mismatch",
			fmt.Sprintf("collection %q: rejecting zero-dimension embedding", collection))
	}

	if established == 0 || established == incoming {
		return nil
	}

	return errorfamily.WrapRejection(ErrVectorDimensionMismatch,
		"metaengine.vector_dimension_mismatch",
		fmt.Sprintf("collection %q: embedding dim %d differs from established dim %d",
			collection, incoming, established))
}
