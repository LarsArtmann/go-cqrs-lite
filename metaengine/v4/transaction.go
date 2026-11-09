package metaengine

import "context"

// Transactional is an optional capability for engines that support explicit
// transactions.
type Transactional interface {
	RunInTx(ctx context.Context, fn func(context.Context) error) error
}
