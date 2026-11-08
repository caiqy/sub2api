//go:build unit

package middleware

import "context"

func (*stubUserRepo) IncrementSessionVersion(context.Context, int64) error {
	panic("unexpected session revocation")
}
