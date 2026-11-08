//go:build unit

package service

import "context"

// These repositories serve unrelated tests; unexpected revocation must fail loudly.
func (*userRepoStubForGroupUpdate) IncrementSessionVersion(context.Context, int64) error {
	panic("unexpected session revocation")
}
func (*userRepoStub) IncrementSessionVersion(context.Context, int64) error {
	panic("unexpected session revocation")
}
func (*emailSyncRepoStub) IncrementSessionVersion(context.Context, int64) error {
	panic("unexpected session revocation")
}
func (*mockUserRepo) IncrementSessionVersion(context.Context, int64) error {
	panic("unexpected session revocation")
}
