//go:build unit

package server_test

import "context"

func (*stubUserRepo) IncrementSessionVersion(context.Context, int64) error {
	panic("unexpected session revocation")
}
