//go:build unit

package service_test

import "context"

func (*emailBindUserRepoStub) IncrementSessionVersion(context.Context, int64) error {
	panic("unexpected session revocation")
}
