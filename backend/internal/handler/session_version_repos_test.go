package handler

import "context"

func (*hiddenUIUserRepo) IncrementSessionVersion(context.Context, int64) error {
	panic("unexpected session revocation")
}

func (*oauthPendingFlowUserRepo) IncrementSessionVersion(context.Context, int64) error {
	panic("unexpected session revocation")
}
