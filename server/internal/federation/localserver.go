// Package federation manages this server's identity and, later, its relations
// with peer servers.
package federation

import (
	"context"
	"errors"
	"fmt"

	"github.com/AbuDubu/amethyst/server/internal/federation/store"
)

// ErrOriginChanged means the configured canonical origin differs from the one
// this server's database recorded on first start.
var ErrOriginChanged = errors.New("canonical origin changed")

// EnsureLocalServer records origin as this server's identity on first start
// and verifies it on every later start. Peers and accounts refer to a server
// by its origin, so a changed origin is refused rather than applied: changing
// a server's identity requires a deliberate migration that does not exist yet.
func EnsureLocalServer(ctx context.Context, q *store.Queries, origin string) (store.Server, error) {
	if err := q.InsertLocalServerIfAbsent(ctx, origin); err != nil {
		return store.Server{}, fmt.Errorf("record local server: %w", err)
	}
	local, err := q.GetLocalServer(ctx)
	if err != nil {
		return store.Server{}, fmt.Errorf("load local server: %w", err)
	}
	if local.CanonicalOrigin != origin {
		return store.Server{}, fmt.Errorf("%w: this database belongs to %s, but AMETHYST_CANONICAL_ORIGIN is %s; changing a server's identity is not supported",
			ErrOriginChanged, local.CanonicalOrigin, origin)
	}
	return local, nil
}
